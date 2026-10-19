package main

import (
	"fmt"
	"os"

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

	if v := os.Getenv("VAULT_ADDR"); v != "" {
		cfg.Vault.Address = v
	}
	if v := os.Getenv("VAULT_TOKEN"); v != "" {
		cfg.Vault.Token = v
	}
	if v := os.Getenv("VAULT_NAMESPACE"); v != "" {
		cfg.Vault.Namespace = v
	}

	if cfg.Vault.Address == "" {
		return nil, fmt.Errorf("vault.address is required")
	}
	if cfg.Vault.Token == "" {
		return nil, fmt.Errorf("vault.token is required")
	}
	if cfg.Vault.Role == "" {
		return nil, fmt.Errorf("vault.role is required")
	}
	if err := cfg.Server.TLS.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
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
