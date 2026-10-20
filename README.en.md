# Short-lived SSH login certificates

[中文](README.md) | [English](README.en.md)

Paste an OpenSSH public key in the frontend. The Go/Gin backend asks Vault SSH CA to sign it and returns a short-lived user certificate. The Vue build is embedded into the same binary with `go:embed`.

## Layout

```
backend/     Gin service
frontend/    Vue 3 + Vite source
docker/      default config in the image
kubernetes/  k8s manifests
build/       build artifacts (frontend dist + binary)
Makefile     build, run, image
```

## Configuration

There are two TLS layers. Do not mix the certificates:

| Role | Files needed | Where |
| --- | --- | --- |
| This service as HTTPS server | CA, server cert, server key | `server.tls` |
| This service as Vault client | Vault CA, client cert, client key | `VAULT_CACERT` / `VAULT_CLIENT_CERT` / `VAULT_CLIENT_KEY` |

Copy `backend/config.example.yaml` to `backend/config.yaml`. It only holds HTTP/TLS settings:

```yaml
server:
  addr: ":8080"
  cors_origins:
    - "http://localhost:5173"
  tls:
    cert_file: "<server-cert.pem>"       # this service's cert
    key_file: "<server-key.pem>"         # this service's private key
    client_ca_file: "<client-ca.pem>"    # CA used to verify clients, not client.crt
    require_client_cert: true
```

Server-side mTLS only needs those three files. Do not put `client.crt` / `client.key` in `server.tls`.

Vault settings must come from environment variables. Startup validates them and exits with a missing-item list:

```
load config: missing required environment variables:
  - VAULT_ADDR
  - VAULT_TOKEN
  - VAULT_ROLE
```

| Variable | Required | Description |
| --- | --- | --- |
| `VAULT_ADDR` | yes | Vault address |
| `VAULT_TOKEN` | yes | Vault token |
| `VAULT_ROLE` | yes | SSH secrets engine signing role |
| `VAULT_MOUNT` | no | mount path, default `ssh` |
| `VAULT_NAMESPACE` | no | Enterprise namespace |
| `VAULT_DEFAULT_PRINCIPALS` | no | default login user |
| `VAULT_TLS_SKIP_VERIFY` | no | skip Vault TLS verify, default `false` |
| `VAULT_MTLS` | no | `true` forces this service to present a client cert to Vault |
| `VAULT_CACERT` | when talking to Vault over mTLS | CA used to verify Vault's server cert |
| `VAULT_CLIENT_CERT` | when talking to Vault over mTLS | client cert this service presents to Vault |
| `VAULT_CLIENT_KEY` | when talking to Vault over mTLS | matching private key |

These variables are only for **this service calling Vault**. They do not enable mTLS on this service's own `:8080`.

If you set `VAULT_MTLS=true`, or fill any of `VAULT_CACERT` / `VAULT_CLIENT_CERT` / `VAULT_CLIENT_KEY`, all three files must exist or startup exits with the missing list.

```bash
export VAULT_ADDR="https://<vault-host>:8200"
export VAULT_TOKEN="<vault-token>"
export VAULT_ROLE="<ssh-role>"
export VAULT_DEFAULT_PRINCIPALS="<login-user>"

# only when Vault requires a client cert (this service is Vault's client)
export VAULT_MTLS="true"
export VAULT_CACERT="/certs/ca.crt"
export VAULT_CLIENT_CERT="/certs/client.crt"
export VAULT_CLIENT_KEY="/certs/client.key"
```

### This service TLS / mTLS

Leave `server.tls.cert_file` / `key_file` empty for plain HTTP. Set the server cert and key to enable HTTPS; add `client_ca_file` (a CA) to verify client certs (mTLS). With `require_client_cert: true`, the handshake fails unless a client cert signed by that CA is presented.

The server does not load `client.crt` / `client.key`. Those belong to callers (browser, curl, Vite proxy) or to the Vault client path.

Certificate SANs include `ssh.local`, `localhost`, and `127.0.0.1`. Use one of those hostnames in the browser.

On macOS, import the client certificate and always-trust the CA:

```bash
open <client.p12>
open <ca.crt>
```

For the Vite dev proxy over mTLS, set the cert directory (do not commit local paths):

```bash
export MTLS_CERT_DIR=<certs-dir>
export MTLS_CLIENT=<client-name>
```

```bash
# HTTPS only
curl https://127.0.0.1:8080/api/health --cacert <server-ca.pem>

# mTLS
curl https://127.0.0.1:8080/api/health \
  --cacert <server-ca.pem> \
  --cert <client-cert.pem> \
  --key <client-key.pem>
```

Certificate TTL and allowed login users are decided by the Vault SSH role.

## Vault SSH role example

Enable the SSH secrets engine and generate a CA (if not already configured):

```bash
vault secrets enable -path=ssh ssh
vault write -f ssh/config/ca generate_signing_key=true
```

Create a role that signs short-lived user certificates:

```bash
vault write ssh/roles/<ssh-role> \
  key_type=ca \
  allow_user_certificates=true \
  allow_host_certificates=false \
  allowed_users="<login-user>" \
  default_user="<login-user>" \
  ttl=10m \
  max_ttl=30m \
  allowed_extensions="permit-pty" \
  default_extensions='{"permit-pty":""}'
```

Equivalent API:

```bash
curl -s \
  -H "X-Vault-Token: $VAULT_TOKEN" \
  -H "Content-Type: application/json" \
  -X POST http://<vault-host>:8200/v1/ssh/roles/<ssh-role> \
  -d '{
    "key_type": "ca",
    "allow_user_certificates": true,
    "allow_host_certificates": false,
    "allowed_users": "<login-user>",
    "default_user": "<login-user>",
    "ttl": "10m",
    "max_ttl": "30m",
    "allowed_extensions": "permit-pty",
    "default_extensions": {"permit-pty": ""}
  }'
```

`allowed_users` limits certificate principals; `ttl` / `max_ttl` are decided by Vault. `default_extensions` must include `permit-pty`, or interactive logins usually get no TTY.

## Local run

```bash
make run          # build into build/ and start
make open         # same, then open the browser
make build        # artifacts: build/web, build/ssh-sign
make docker       # cross-compile a linux binary; image tag is the current git hash
make clean        # remove binaries
```

Or run the two sides separately in development:

```bash
cd backend && go run . -config config.yaml
cd frontend && npm run dev    # http://localhost:5173, opens the browser
```

Save the returned `signed_key` as `~/.ssh/id_ed25519-cert.pub` (same prefix as the private key), then:

```bash
ssh -i ~/.ssh/id_ed25519 \
  -o CertificateFile=~/.ssh/id_ed25519-cert.pub \
  <login-user>@<target-host>
```

## Docker

The image is based on local `debian:bullseye-v1.4.1` and copies in the linux binary with the frontend embedded. The tag is the current git short hash. Image `docker/config.yaml` leaves `server.tls` empty, so the container listens on plain HTTP.

```bash
make docker
docker run --rm -p 8080:8080 \
  -e VAULT_ADDR="https://<vault-host>:8200" \
  -e VAULT_TOKEN="<vault-token>" \
  -e VAULT_ROLE="<ssh-role>" \
  ssh-sign:<git-hash>
```

If Vault requires a client certificate, also mount the **Vault client** certs (not this service's server certs):

```bash
docker run --rm -p 8080:8080 \
  -e VAULT_ADDR="https://<vault-host>:8200" \
  -e VAULT_TOKEN="<vault-token>" \
  -e VAULT_ROLE="<ssh-role>" \
  -e VAULT_MTLS="true" \
  -e VAULT_CACERT="/certs/ca.crt" \
  -e VAULT_CLIENT_CERT="/certs/client.crt" \
  -e VAULT_CLIENT_KEY="/certs/client.key" \
  -v /path/to/vault-client-certs:/certs:ro \
  ssh-sign:<git-hash>
```

To enable mTLS on this service itself, mount the CA, server cert, and server key, and point `server.tls` in the container `config.yaml` at those paths. Do not put `client.crt` / `client.key` in `server.tls`.

The container exits with a missing-item list if required environment variables are unset.

## Kubernetes

Manifests are in `kubernetes/`, deployed with kustomize. Edit Vault address, token, and role in `kubernetes/secret.yaml`, then set the image tag to the git hash you just built:

```bash
make docker
HASH=$(git rev-parse --short HEAD)

# change the image name to match your registry; for local tests docker save/load onto the node
cd kubernetes
kustomize edit set image ssh-sign=ssh-sign:${HASH}

# fill in real Vault values in secret.yaml, then apply
kubectl apply -k .
```

This creates:

- Namespace `ssh-sign`
- ConfigMap: `/etc/ssh-sign/config.yaml` in the container
- Secret `ssh-sign-vault`: `VAULT_ADDR` / `VAULT_TOKEN` / `VAULT_ROLE` and related vars
- Deployment + Service (ClusterIP `:8080`)

`kubernetes/mtls-patch.yaml` and Secret `ssh-sign-vault-mtls` are only for **this service calling Vault** (client certs). They do not enable HTTPS/mTLS on this service's `:8080`. `server.tls` in the ConfigMap is empty by default.

If Vault requires a client certificate, create a Secret from the Vault CA and client cert, then enable the kustomize patch:

```bash
kubectl -n ssh-sign create secret generic ssh-sign-vault-mtls \
  --from-file=ca.crt=/path/to/ca.crt \
  --from-file=client.crt=/path/to/client.crt \
  --from-file=client.key=/path/to/client.key
```

Uncomment in `kubernetes/kustomization.yaml`:

```yaml
patches:
  - path: mtls-patch.yaml
```

Certs are mounted at `/certs`, with env vars:

```
VAULT_CACERT=/certs/ca.crt
VAULT_CLIENT_CERT=/certs/client.crt
VAULT_CLIENT_KEY=/certs/client.key
```

To enable mTLS on this service itself, change `server.tls` in the ConfigMap (CA + server cert + server key). Do not fill in the `client.crt` / `client.key` above.

Access:

```bash
kubectl -n ssh-sign port-forward svc/ssh-sign 8080:8080
curl http://127.0.0.1:8080/api/health
```

## UI

Open the page, paste a public key. A successful sign returns the certificate and expiry.

Landing page:

![Landing page](docs/01-page.png)

Public key filled in:

![Public key filled in](docs/02-filled.png)

Issued certificate:

![Issued certificate](docs/03-issued.png)

## Trust the Vault CA on target hosts

CA public key (no token, unauthenticated endpoint):

```bash
curl -s http://<vault-host>:8200/v1/ssh/public_key
```

Or with a token:

```bash
curl -s -H "X-Vault-Token: $VAULT_TOKEN" http://<vault-host>:8200/v1/ssh/config/ca
```

On **every host that should accept these short-lived certificates**:

```bash
# write the CA public key, mode 0644, owner root
install -m 0644 /dev/null /etc/ssh/trusted-user-ca-keys.pem
curl -s http://<vault-host>:8200/v1/ssh/public_key > /etc/ssh/trusted-user-ca-keys.pem

# sshd trusts user certs signed by this CA
grep -q '^TrustedUserCAKeys' /etc/ssh/sshd_config \
  || echo 'TrustedUserCAKeys /etc/ssh/trusted-user-ca-keys.pem' >> /etc/ssh/sshd_config

sshd -t && systemctl reload sshd   # on Debian/Ubuntu this may be: systemctl reload ssh
```

`sshd` checks that the cert was signed by that CA, is not expired, and the principal matches the login user. The client private key does not need to be in `authorized_keys` first.

Client usage:

```bash
# save the issued cert next to the private key as -cert.pub
ssh -i ~/.ssh/id_ed25519 \
  -o CertificateFile=~/.ssh/id_ed25519-cert.pub \
  <login-user>@<target-host>
```

## API

`GET /api/health`  
`GET /api/info` — default login user  
`POST /api/sign`

```json
{
  "public_key": "ssh-ed25519 AAAA...",
  "valid_principals": "<login-user>"
}
```

Certificate TTL is decided by the Vault role. Do not send TTL in the request.

## License

[WTFPL](http://www.wtfpl.net/)
