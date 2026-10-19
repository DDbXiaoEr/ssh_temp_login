package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func listenAndServe(r *gin.Engine, cfg ServerConfig) error {
	if !cfg.TLS.Enabled() {
		return r.Run(cfg.Addr)
	}

	tlsCfg, err := buildTLSConfig(cfg.TLS)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		TLSConfig:    tlsCfg,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return srv.ListenAndServeTLS(cfg.TLS.CertFile, cfg.TLS.KeyFile)
}

func buildTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if cfg.ClientCAFile == "" {
		return tlsCfg, nil
	}

	pem, err := os.ReadFile(cfg.ClientCAFile)
	if err != nil {
		return nil, fmt.Errorf("read client ca: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("parse client ca: no certificates found in %s", cfg.ClientCAFile)
	}
	tlsCfg.ClientCAs = pool
	if cfg.RequireClientCert {
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	} else {
		tlsCfg.ClientAuth = tls.VerifyClientCertIfGiven
	}
	return tlsCfg, nil
}
