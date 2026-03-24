package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/russell/rbooth/internal/rbooth"
)

type fileConfig struct {
	Port     string `json:"port"`
	BaseURL  string `json:"base_url"`
	DataDir  string `json:"data_dir"`
	MediaDir string `json:"media_dir"`
}

func main() {
	configPath := envOr("CONFIG_FILE", "config.json")
	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	addr := envOrDefault("PORT", cfg.Port, "8080")
	baseURL := envOrDefault("APP_BASE_URL", cfg.BaseURL, "http://localhost:"+addr)
	dataDir := envOrDefault("DATA_DIR", cfg.DataDir, "data")
	storageDir := envOrDefault("MEDIA_DIR", cfg.MediaDir, "/mnt/storage/media/rbooth")

	app, err := rbooth.New(rbooth.Config{
		BaseURL:    baseURL,
		DataDir:    dataDir,
		StorageDir: storageDir,
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

func envOrDefault(key, configValue, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	if configValue != "" {
		return configValue
	}
	return fallback
}

func loadConfig(path string) (fileConfig, error) {
	payload, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fileConfig{}, nil
	}
	if err != nil {
		return fileConfig{}, fmt.Errorf("read config %s: %w", path, err)
	}
	if len(bytes.TrimSpace(payload)) == 0 {
		return fileConfig{}, nil
	}

	var cfg fileConfig
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return fileConfig{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}
