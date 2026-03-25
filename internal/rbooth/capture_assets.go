package rbooth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	captureBackdropAssetDir = "web/static/editor-assets/backdrops"
	captureFrameAssetDir    = "web/static/editor-assets/frames"
)

type captureOption struct {
	Value string
	Label string
}

type captureAssetCatalog struct {
	Backdrops      []captureOption
	Frames         []captureOption
	BackdropAssets map[string]string
	FrameAssets    map[string]string
}

func defaultCaptureAssetCatalog() captureAssetCatalog {
	return captureAssetCatalog{
		Backdrops: []captureOption{
			{Value: "none", Label: "no background"},
			{Value: "sunrise", Label: "sunrise"},
			{Value: "mint", Label: "mint"},
			{Value: "night", Label: "night"},
			{Value: "paper", Label: "paper"},
			{Value: "studio", Label: "studio"},
		},
		Frames: []captureOption{
			{Value: "none", Label: "none"},
			{Value: "classic", Label: "classic border"},
			{Value: "polaroid", Label: "polaroid"},
			{Value: "sparkle", Label: "sparkle"},
			{Value: "ticket", Label: "ticket"},
		},
		BackdropAssets: map[string]string{},
		FrameAssets:    map[string]string{},
	}
}

func loadCaptureAssetCatalog() (captureAssetCatalog, error) {
	catalog := defaultCaptureAssetCatalog()

	backdrops, backdropMap, err := scanCaptureAssetDir(captureBackdropAssetDir, "/static/editor-assets/backdrops/")
	if err != nil {
		return captureAssetCatalog{}, err
	}
	frames, frameMap, err := scanCaptureAssetDir(captureFrameAssetDir, "/static/editor-assets/frames/")
	if err != nil {
		return captureAssetCatalog{}, err
	}

	catalog.Backdrops = append(catalog.Backdrops, backdrops...)
	catalog.Frames = append(catalog.Frames, frames...)
	catalog.BackdropAssets = backdropMap
	catalog.FrameAssets = frameMap
	return catalog, nil
}

func scanCaptureAssetDir(dirPath, urlPrefix string) ([]captureOption, map[string]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, map[string]string{}, nil
		}
		return nil, nil, fmt.Errorf("read %s: %w", dirPath, err)
	}

	type assetEntry struct {
		option captureOption
		url    string
	}

	var assets []assetEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !isCaptureAssetFile(name) {
			continue
		}

		base := strings.TrimSuffix(name, filepath.Ext(name))
		slug := slugify(base)
		if slug == "" {
			continue
		}

		value := "asset:" + slug
		assets = append(assets, assetEntry{
			option: captureOption{
				Value: value,
				Label: humanizeCaptureAssetName(base),
			},
			url: urlPrefix + name,
		})
	}

	slices.SortFunc(assets, func(left, right assetEntry) int {
		return strings.Compare(left.option.Label, right.option.Label)
	})

	options := make([]captureOption, 0, len(assets))
	urls := make(map[string]string, len(assets))
	for _, asset := range assets {
		options = append(options, asset.option)
		urls[asset.option.Value] = asset.url
	}
	return options, urls, nil
}

func isCaptureAssetFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return true
	default:
		return false
	}
}

func humanizeCaptureAssetName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}

	var builder strings.Builder
	lastSpace := false
	for _, r := range name {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			builder.WriteRune(r)
			lastSpace = false
		case r == '-' || r == '_' || r == ' ':
			if builder.Len() > 0 && !lastSpace {
				builder.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(builder.String())
}

func (c captureAssetCatalog) JSON() string {
	payload, err := json.Marshal(map[string]any{
		"backdrops": c.BackdropAssets,
		"frames":    c.FrameAssets,
	})
	if err != nil {
		return `{"backdrops":{},"frames":{}}`
	}
	return string(payload)
}
