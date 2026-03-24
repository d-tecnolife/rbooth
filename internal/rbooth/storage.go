package rbooth

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

type Storage interface {
	Save(ctx context.Context, key string, contentType string, payload []byte) error
	Open(ctx context.Context, key string) (io.ReadCloser, string, error)
}

func NewLocalStorage(root string) Storage {
	return &LocalStorage{root: root}
}

type LocalStorage struct {
	root string
}

func (s *LocalStorage) Save(_ context.Context, key string, _ string, payload []byte) error {
	target, err := s.pathFor(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create local storage dir: %w", err)
	}
	if err := os.WriteFile(target, payload, 0o644); err != nil {
		return fmt.Errorf("write local storage object: %w", err)
	}
	return nil
}

func (s *LocalStorage) Open(_ context.Context, key string) (io.ReadCloser, string, error) {
	target, err := s.pathFor(key)
	if err != nil {
		return nil, "", err
	}
	file, err := os.Open(target)
	if err != nil {
		return nil, "", err
	}
	return file, mime.TypeByExtension(filepath.Ext(target)), nil
}

func (s *LocalStorage) pathFor(key string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(key))
	if cleaned == "." || strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("invalid storage key")
	}
	return filepath.Join(s.root, cleaned), nil
}
