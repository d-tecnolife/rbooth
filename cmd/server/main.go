package main

import (
	"log"
	"net/http"
	"os"

	"github.com/russell/rbooth/internal/rbooth"
)

func main() {
	addr := envOr("PORT", "8080")
	baseURL := envOr("APP_BASE_URL", "http://localhost:"+addr)
	dataDir := envOr("DATA_DIR", "data")

	app, err := rbooth.New(rbooth.Config{
		BaseURL: baseURL,
		DataDir: dataDir,
	})
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:    ":" + addr,
		Handler: app.Routes(),
	}

	log.Printf("rbooth listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
