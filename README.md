# 临时 SSH 登录证书签发

前端提交 OpenSSH 公钥，Go/Gin 后端读取配置后向 Vault SSH CA 申请签名，把短期用户证书返回给前端。Vue 构建产物通过 `go:embed` 打进同一个二进制。

## 目录

```
backend/   Gin 服务
frontend/  Vue 3 + Vite 源码
build/     构建产物（前端 dist + 二进制）
Makefile   构建、运行
```

## 配置

复制 `backend/config.example.yaml` 为 `backend/config.yaml`：

```yaml
server:
  addr: ":8080"
  cors_origins:
    - "http://localhost:5173"
  tls:
    cert_file: "<server-cert.pem>"
    key_file: "<server-key.pem>"
    client_ca_file: "<client-ca.pem>"
    require_client_cert: true

vault:
  address: "http://<vault-host>:8200"
  token: "<vault-token>"
  mount: "ssh"
  role: "<ssh-role>"
  default_principals: "<login-user>"
```

`VAULT_ADDR`、`VAULT_TOKEN`、`VAULT_NAMESPACE` 环境变量会覆盖对应字段。

### TLS / mTLS

不填 `server.tls.cert_file` / `key_file` 时走明文 HTTP。配上服务端证书后启用 HTTPS；再配 `client_ca_file` 即校验客户端证书（mTLS）。`require_client_cert: true` 时必须出示由该 CA 签发的客户端证书，否则握手失败。

```bash
# 仅 HTTPS
curl https://<listen-host>:8080/api/health --cacert <server-ca.pem>

# mTLS
curl https://<listen-host>:8080/api/health \
  --cacert <server-ca.pem> \
  --cert <client-cert.pem> \
  --key <client-key.pem>
```

证书有效期、允许的登录用户由 Vault SSH role 决定。

## Vault SSH role 示例

启用 SSH secrets engine 并生成 CA（若尚未配置）：

```bash
vault secrets enable -path=ssh ssh
vault write -f ssh/config/ca generate_signing_key=true
```

创建用于签发短期用户证书的 role：

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

等价 API：

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

`allowed_users` 限制证书 principal；`ttl` / `max_ttl` 由 Vault 决定证书有效期。`default_extensions` 必须包含 `permit-pty`，否则交互式登录通常分不到终端。

## 运行

```bash
make run          # 构建到 build/ 并启动 http://127.0.0.1:8080
make open         # 同上，并打开浏览器
make build        # 产物：build/web、build/ssh-sign
make clean        # 删除 build/ssh-sign
```

开发时也可分开跑：

```bash
cd backend && go run . -config config.yaml
cd frontend && npm run dev    # http://localhost:5173，自动开浏览器
```

把返回的 `signed_key` 存为 `~/.ssh/id_ed25519-cert.pub`（与私钥同前缀），即可：

```bash
ssh -i ~/.ssh/id_ed25519 \
  -o CertificateFile=~/.ssh/id_ed25519-cert.pub \
  <login-user>@<target-host>
```

## 目标机信任 Vault CA

CA 公钥（无需 token，未认证接口）：

```bash
curl -s http://<vault-host>:8200/v1/ssh/public_key
```

或带 token：

```bash
curl -s -H "X-Vault-Token: $VAULT_TOKEN" http://<vault-host>:8200/v1/ssh/config/ca
```

在**每一台允许用临时证书登录的机器**上：

```bash
# 写入 CA 公钥，权限 0644，属主 root
install -m 0644 /dev/null /etc/ssh/trusted-user-ca-keys.pem
curl -s http://<vault-host>:8200/v1/ssh/public_key > /etc/ssh/trusted-user-ca-keys.pem

# sshd 信任该 CA 签发的用户证书
grep -q '^TrustedUserCAKeys' /etc/ssh/sshd_config \
  || echo 'TrustedUserCAKeys /etc/ssh/trusted-user-ca-keys.pem' >> /etc/ssh/sshd_config

sshd -t && systemctl reload sshd   # Debian/Ubuntu 也可能是: systemctl reload ssh
```

`sshd` 校验的是证书由该 CA 签出、未过期、principal 匹配登录用户。客户端私钥不必事先放进 `authorized_keys`。

客户端使用：

```bash
# 签发结果存为与私钥同前缀的 -cert.pub
ssh -i ~/.ssh/id_ed25519 \
  -o CertificateFile=~/.ssh/id_ed25519-cert.pub \
  <login-user>@<target-host>
```

## API

`GET /api/health`  
`GET /api/info` — 默认登录用户  
`POST /api/sign`

```json
{
  "public_key": "ssh-ed25519 AAAA...",
  "valid_principals": "<login-user>"
}
```

证书有效期由 Vault role 决定，请求里不传 TTL。
