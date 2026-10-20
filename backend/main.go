package main

import (
	"flag"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var ginMode = gin.DebugMode

func main() {
	gin.SetMode(ginMode)

	configPath := flag.String("config", "config.yaml", "path to yaml config")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	vault, err := NewVaultClient(cfg.Vault)
	if err != nil {
		log.Fatalf("init vault: %v", err)
	}

	r := gin.Default()
	if len(cfg.Server.CORSOrigins) == 0 {
		r.Use(cors.Default())
	} else {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     cfg.Server.CORSOrigins,
			AllowMethods:     []string{"GET", "POST", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	registerRoutes(r, cfg, vault)
	serveFrontend(r)

	if cfg.Server.TLS.Enabled() {
		mode := "tls"
		if cfg.Server.TLS.ClientCAFile != "" {
			if cfg.Server.TLS.RequireClientCert {
				mode = "mtls (require client cert)"
			} else {
				mode = "mtls (verify client cert if given)"
			}
		}
		log.Printf("listening on %s (%s)", cfg.Server.Addr, mode)
	} else {
		log.Printf("listening on %s", cfg.Server.Addr)
	}
	if err := listenAndServe(r, cfg.Server); err != nil {
		log.Fatalf("http server: %v", err)
	}
}
