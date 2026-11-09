package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type SignRequest struct {
	PublicKey       string `json:"public_key"`
	ValidPrincipals string `json:"valid_principals"`
	Role            string `json:"role"`
}

type SignResponse struct {
	SignedKey    string `json:"signed_key"`
	SerialNumber string `json:"serial_number"`
	LeaseID      string `json:"lease_id,omitempty"`
	Principals   string `json:"principals"`
	Role         string `json:"role"`
}

type RoleInfo struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	DefaultPrincipals string `json:"default_principals,omitempty"`
}

type InfoResponse struct {
	DefaultPrincipals string     `json:"default_principals"`
	Roles             []RoleInfo `json:"roles"`
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
	roles := make([]RoleInfo, 0, len(cfg.Vault.Roles))
	for _, role := range cfg.Vault.Roles {
		roles = append(roles, RoleInfo{
			Name:              role.Role,
			Description:       role.Description,
			DefaultPrincipals: role.DefaultPrincipals,
		})
	}

	c.JSON(http.StatusOK, InfoResponse{
		DefaultPrincipals: cfg.Vault.DefaultPrincipals,
		Roles:             roles,
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

	roleName := strings.TrimSpace(req.Role)
	if roleName == "" {
		if len(cfg.Vault.Roles) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role is required"})
			return
		}
		roleName = cfg.Vault.Roles[0].Role
	}

	role, ok := findRole(cfg.Vault.Roles, roleName)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown role %q", roleName)})
		return
	}

	principals := strings.TrimSpace(req.ValidPrincipals)
	if principals == "" {
		principals = role.DefaultPrincipals
	}
	if principals == "" {
		principals = cfg.Vault.DefaultPrincipals
	}

	result, err := vault.Sign(c.Request.Context(), publicKey, principals, role.Role)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SignResponse{
		SignedKey:    result.SignedKey,
		SerialNumber: result.SerialNumber,
		LeaseID:      result.LeaseID,
		Principals:   principals,
		Role:         role.Role,
	})
}

func findRole(roles []VaultRole, name string) (VaultRole, bool) {
	for _, role := range roles {
		if role.Role == name {
			return role, true
		}
	}
	return VaultRole{}, false
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
