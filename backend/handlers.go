package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type SignRequest struct {
	PublicKey       string `json:"public_key"`
	ValidPrincipals string `json:"valid_principals"`
}

type SignResponse struct {
	SignedKey    string `json:"signed_key"`
	SerialNumber string `json:"serial_number"`
	LeaseID      string `json:"lease_id,omitempty"`
	Principals   string `json:"principals"`
}

type InfoResponse struct {
	DefaultPrincipals string `json:"default_principals"`
}

func registerRoutes(r *gin.Engine, cfg *Config, vault *VaultClient) {
	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		api.GET("/info", func(c *gin.Context) {
			handleInfo(c, cfg)
		})
		api.POST("/sign", func(c *gin.Context) {
			handleSign(c, cfg, vault)
		})
	}
}

func handleInfo(c *gin.Context, cfg *Config) {
	c.JSON(http.StatusOK, InfoResponse{
		DefaultPrincipals: cfg.Vault.DefaultPrincipals,
	})
}

func handleSign(c *gin.Context, cfg *Config, vault *VaultClient) {
	var req SignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}

	publicKey := strings.TrimSpace(req.PublicKey)
	if !looksLikeSSHPublicKey(publicKey) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "public_key must be an OpenSSH public key"})
		return
	}

	principals := strings.TrimSpace(req.ValidPrincipals)
	if principals == "" {
		principals = cfg.Vault.DefaultPrincipals
	}

	result, err := vault.Sign(c.Request.Context(), publicKey, principals)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SignResponse{
		SignedKey:    result.SignedKey,
		SerialNumber: result.SerialNumber,
		LeaseID:      result.LeaseID,
		Principals:   principals,
	})
}

func looksLikeSSHPublicKey(key string) bool {
	prefixes := []string{
		"ssh-ed25519 ",
		"ssh-rsa ",
		"ssh-dss ",
		"ecdsa-sha2-nistp256 ",
		"ecdsa-sha2-nistp384 ",
		"ecdsa-sha2-nistp521 ",
		"sk-ssh-ed25519@",
		"sk-ecdsa-sha2-nistp256@",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}
