package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Vault  VaultConfig  `yaml:"vault"`
}

type ServerConfig struct {
	Addr        string    `yaml:"addr"`
	CORSOrigins []string  `yaml:"cors_origins"`
	TLS         TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	CertFile          string `yaml:"cert_file"`
	KeyFile           string `yaml:"key_file"`
	ClientCAFile      string `yaml:"client_ca_file"`
	RequireClientCert bool   `yaml:"require_client_cert"`
}

type VaultConfig struct {
	Address           string `yaml:"address"`
	Token             string `yaml:"token"`
	Namespace         string `yaml:"namespace"`
	Mount             string `yaml:"mount"`
	Role              string `yaml:"role"`
	TLSSkipVerify     bool   `yaml:"tls_skip_verify"`
	CACert            string `yaml:"ca_cert"`
	ClientCert        string `yaml:"client_cert"`
	ClientKey         string `yaml:"client_key"`
	DefaultPrincipals string `yaml:"default_principals"`
}

func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if err := applyVaultFromEnv(cfg); err != nil {
		return nil, err
	}
	if err := cfg.Server.TLS.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func applyVaultFromEnv(cfg *Config) error {
	var missing []string

	addr := requireEnv("VAULT_ADDR", &missing)
	token := requireEnv("VAULT_TOKEN", &missing)
	role := requireEnv("VAULT_ROLE", &missing)

	mount := envOr("VAULT_MOUNT", "ssh")
	skipVerify, err := envBool("VAULT_TLS_SKIP_VERIFY")
	if err != nil {
		return err
	}
	mtls, err := envBool("VAULT_MTLS")
	if err != nil {
		return err
	}

	caCert := strings.TrimSpace(os.Getenv("VAULT_CACERT"))
	clientCert := strings.TrimSpace(os.Getenv("VAULT_CLIENT_CERT"))
	clientKey := strings.TrimSpace(os.Getenv("VAULT_CLIENT_KEY"))
	if mtls || caCert != "" || clientCert != "" || clientKey != "" {
		if caCert == "" {
			missing = append(missing, "VAULT_CACERT")
		}
		if clientCert == "" {
			missing = append(missing, "VAULT_CLIENT_CERT")
		}
		if clientKey == "" {
			missing = append(missing, "VAULT_CLIENT_KEY")
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables:\n  - %s", strings.Join(missing, "\n  - "))
	}

	if caCert != "" || clientCert != "" || clientKey != "" {
		for _, p := range []struct {
			key  string
			path string
		}{
			{"VAULT_CACERT", caCert},
			{"VAULT_CLIENT_CERT", clientCert},
			{"VAULT_CLIENT_KEY", clientKey},
		} {
			if _, err := os.Stat(p.path); err != nil {
				return fmt.Errorf("%s: %w", p.key, err)
			}
		}
	}

	cfg.Vault = VaultConfig{
		Address:           addr,
		Token:             token,
		Namespace:         strings.TrimSpace(os.Getenv("VAULT_NAMESPACE")),
		Mount:             mount,
		Role:              role,
		TLSSkipVerify:     skipVerify,
		CACert:            caCert,
		ClientCert:        clientCert,
		ClientKey:         clientKey,
		DefaultPrincipals: strings.TrimSpace(os.Getenv("VAULT_DEFAULT_PRINCIPALS")),
	}
	return nil
}

func requireEnv(key string, missing *[]string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		*missing = append(*missing, key)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string) (bool, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return false, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean", key)
	}
	return b, nil
}

func (t TLSConfig) Enabled() bool {
	return t.CertFile != "" || t.KeyFile != "" || t.ClientCAFile != ""
}

func (t TLSConfig) validate() error {
	if !t.Enabled() {
		return nil
	}
	if t.CertFile == "" || t.KeyFile == "" {
		return fmt.Errorf("server.tls.cert_file and server.tls.key_file are required when tls is enabled")
	}
	if t.RequireClientCert && t.ClientCAFile == "" {
		return fmt.Errorf("server.tls.client_ca_file is required when require_client_cert is true")
	}
	return nil
}
