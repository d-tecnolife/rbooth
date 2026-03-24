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
	appName := envOr("APP_NAME", "rbooth")
	dataDir := envOr("DATA_DIR", "data")
	storageDir := envOr("MEDIA_DIR", "/app/media")
	adminPassword := envOr("ADMIN_PASSWORD", "")
	authSecret := envOr("AUTH_SECRET", "")

	app, err := rbooth.New(rbooth.Config{
		AppName:       appName,
		BaseURL:       baseURL,
		DataDir:       dataDir,
		StorageDir:    storageDir,
		AdminPassword: adminPassword,
		AuthSecret:    authSecret,
	})
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:    ":" + addr,
		Handler: app.Routes(),
	}

	log.Printf("%s listening on %s", appName, server.Addr)
	log.Fatal(server.ListenAndServe())
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
