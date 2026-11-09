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
	MTLS              bool   `yaml:"mtls"`
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
	if cfg.Vault.Mount == "" {
		cfg.Vault.Mount = "ssh"
	}
	if err := applyVaultFromEnv(cfg); err != nil {
		return nil, err
	}
	if err := cfg.Vault.validate(); err != nil {
		return nil, err
	}
	if err := cfg.Server.TLS.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func applyVaultFromEnv(cfg *Config) error {
	var missing []string

	cfg.Vault.Address = requireEnv("VAULT_ADDR", &missing)
	cfg.Vault.Token = requireEnv("VAULT_TOKEN", &missing)
	cfg.Vault.Role = requireEnv("VAULT_ROLE", &missing)

	if v := strings.TrimSpace(os.Getenv("VAULT_MOUNT")); v != "" {
		cfg.Vault.Mount = v
	}
	if v := strings.TrimSpace(os.Getenv("VAULT_NAMESPACE")); v != "" {
		cfg.Vault.Namespace = v
	}
	if v := strings.TrimSpace(os.Getenv("VAULT_DEFAULT_PRINCIPALS")); v != "" {
		cfg.Vault.DefaultPrincipals = v
	}
	if v := strings.TrimSpace(os.Getenv("VAULT_CACERT")); v != "" {
		cfg.Vault.CACert = v
	}
	if v := strings.TrimSpace(os.Getenv("VAULT_CLIENT_CERT")); v != "" {
		cfg.Vault.ClientCert = v
	}
	if v := strings.TrimSpace(os.Getenv("VAULT_CLIENT_KEY")); v != "" {
		cfg.Vault.ClientKey = v
	}

	var err error
	if cfg.Vault.TLSSkipVerify, err = envBoolOr("VAULT_TLS_SKIP_VERIFY", cfg.Vault.TLSSkipVerify); err != nil {
		return err
	}
	if cfg.Vault.MTLS, err = envBoolOr("VAULT_MTLS", cfg.Vault.MTLS); err != nil {
		return err
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables:\n  - %s", strings.Join(missing, "\n  - "))
	}
	return nil
}

// validate 检查连接 Vault 的客户端证书。配置了 mtls、或填写了任意证书路径时，
// ca_cert、client_cert、client_key 三份文件都必须存在。
func (v VaultConfig) validate() error {
	certs := []struct {
		name string
		path string
	}{
		{"vault.ca_cert / VAULT_CACERT", v.CACert},
		{"vault.client_cert / VAULT_CLIENT_CERT", v.ClientCert},
		{"vault.client_key / VAULT_CLIENT_KEY", v.ClientKey},
	}

	hasCert := false
	for _, cert := range certs {
		if cert.path != "" {
			hasCert = true
			break
		}
	}
	if !v.MTLS && !hasCert {
		return nil
	}

	var missing []string
	for _, cert := range certs {
		if cert.path == "" {
			missing = append(missing, cert.name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("vault mTLS requires all three files, missing:\n  - %s", strings.Join(missing, "\n  - "))
	}

	for _, cert := range certs {
		if _, err := os.Stat(cert.path); err != nil {
			return fmt.Errorf("%s: %w", cert.name, err)
		}
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

func envBoolOr(key string, fallback bool) (bool, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
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
