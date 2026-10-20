package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
)

type VaultClient struct {
	logical           *api.Logical
	mount             string
	role              string
	defaultPrincipals string
}

type SignResult struct {
	SignedKey    string `json:"signed_key"`
	SerialNumber string `json:"serial_number"`
	LeaseID      string `json:"lease_id,omitempty"`
}

func NewVaultClient(cfg VaultConfig) (*VaultClient, error) {
	vaultCfg := api.DefaultConfig()
	vaultCfg.Address = cfg.Address
	if err := vaultCfg.ConfigureTLS(&api.TLSConfig{
		CACert:     cfg.CACert,
		ClientCert: cfg.ClientCert,
		ClientKey:  cfg.ClientKey,
		Insecure:   cfg.TLSSkipVerify,
	}); err != nil {
		return nil, fmt.Errorf("vault tls: %w", err)
	}

	client, err := api.NewClient(vaultCfg)
	if err != nil {
		return nil, fmt.Errorf("vault client: %w", err)
	}
	client.SetToken(cfg.Token)
	if cfg.Namespace != "" {
		client.SetNamespace(cfg.Namespace)
	}

	return &VaultClient{
		logical:           client.Logical(),
		mount:             strings.Trim(cfg.Mount, "/"),
		role:              cfg.Role,
		defaultPrincipals: cfg.DefaultPrincipals,
	}, nil
}

func (c *VaultClient) Sign(ctx context.Context, publicKey, principals string) (*SignResult, error) {
	publicKey = strings.TrimSpace(publicKey)
	principals = strings.TrimSpace(principals)
	if principals == "" {
		principals = c.defaultPrincipals
	}

	payload := map[string]interface{}{
		"public_key": publicKey,
		"cert_type":  "user",
	}
	if principals != "" {
		payload["valid_principals"] = principals
	}

	path := fmt.Sprintf("%s/sign/%s", c.mount, c.role)
	secret, err := c.logical.WriteWithContext(ctx, path, payload)
	if err != nil {
		return nil, fmt.Errorf("vault sign: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("vault returned empty sign response")
	}

	signed, _ := secret.Data["signed_key"].(string)
	serial, _ := secret.Data["serial_number"].(string)
	if strings.TrimSpace(signed) == "" {
		return nil, fmt.Errorf("vault response missing signed_key")
	}

	return &SignResult{
		SignedKey:    signed,
		SerialNumber: serial,
		LeaseID:      secret.LeaseID,
	}, nil
}
