package rbooth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
	_ "golang.org/x/image/webp"
)

type Config struct {
	BaseURL    string
	DataDir    string
	StorageDir string
}

const singleBoardCode = "main-board"
const defaultStorageDir = "/mnt/storage/media/rbooth"

type App struct {
	baseURL   string
	dataDir   string
	storePath string
	storage   Storage
	templates *template.Template

	mu      sync.RWMutex
	events  map[string]*Event
	photos  map[string][]Photo
	clients map[string]map[chan Photo]struct{}
}

type Event struct {
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
}

type Photo struct {
	ID          string    `json:"id"`
	EventCode   string    `json:"event_code"`
	Filename    string    `json:"filename"`
	StorageKey  string    `json:"storage_key,omitempty"`
	Caption     string    `json:"caption"`
	CreatedAt   time.Time `json:"created_at"`
	DisplayURL  string    `json:"display_url"`
	FilterLabel string    `json:"filter_label"`
}

type persistedState struct {
	Events []Event `json:"events"`
	Photos []Photo `json:"photos"`
}

type pageData struct {
	Title       string
	BaseURL     string
	Event       *Event
	Photos      []Photo
	CaptureURL  string
	BoardURL    string
	AdminURL    string
	DefaultCode string
}

type samplePhotoSpec struct {
	Filename  string
	Caption   string
	Filter    string
	Primary   color.RGBA
	Secondary color.RGBA
	Accent    color.RGBA
}

func New(cfg Config) (*App, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("base url is required")
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "data"
	}
	if cfg.StorageDir == "" {
		cfg.StorageDir = defaultStorageDir
	}

	templates, err := template.ParseGlob(filepath.Join("web", "templates", "*.tmpl"))
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	app := &App{
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		dataDir:   cfg.DataDir,
		storePath: filepath.Join(cfg.DataDir, "state.json"),
		templates: templates,
		events:    make(map[string]*Event),
		photos:    make(map[string][]Photo),
		clients:   make(map[string]map[chan Photo]struct{}),
	}

	if err := os.MkdirAll(cfg.StorageDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload root: %w", err)
	}

	app.storage = NewLocalStorage(cfg.StorageDir)

	if err := app.loadState(); err != nil {
		return nil, err
	}

	app.ensureDefaultEvent()
	if err := app.ensureSamplePhotos(); err != nil {
		return nil, err
	}

	return app, nil
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join("web", "static")))))
	mux.HandleFunc("GET /media/", a.handleMedia)
	mux.HandleFunc("GET /{$}", a.handleHome)
	mux.HandleFunc("GET /photos", a.handleBoard)
	mux.HandleFunc("GET /capture", a.handleCapture)
	mux.HandleFunc("GET /qr", a.handleQR)
	mux.HandleFunc("GET /api/photos", a.handlePhotos)
	mux.HandleFunc("POST /api/photos", a.handleUpload)
	mux.HandleFunc("GET /stream", a.handleStream)
	mux.HandleFunc("GET /board/{code}", redirectTo("/photos"))
	mux.HandleFunc("GET /capture/{code}", redirectTo("/capture"))
	mux.HandleFunc("GET /qr/{code}", redirectTo("/qr"))
	mux.HandleFunc("GET /api/events/{code}/photos", a.handlePhotos)
	mux.HandleFunc("POST /api/events/{code}/photos", a.handleUpload)
	mux.HandleFunc("GET /events/{code}/stream", a.handleStream)

	return withLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, pattern := mux.Handler(r)
		if pattern == "" {
			a.handleNotFound(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	}))
}

func (a *App) handleHome(w http.ResponseWriter, r *http.Request) {
	event := a.singleEvent()

	data := pageData{
		Title:      "rbooth",
		BaseURL:    a.baseURL,
		Event:      event,
		CaptureURL: a.captureURL(),
		BoardURL:   a.boardURL(),
		AdminURL:   a.adminURL(),
		Photos:     a.listAllPhotos(),
	}
	a.render(w, "home", data)
}

func (a *App) handleCapture(w http.ResponseWriter, r *http.Request) {
	event := a.singleEvent()

	data := pageData{
		Title:      "Capture a Photo",
		BaseURL:    a.baseURL,
		Event:      event,
		CaptureURL: a.captureURL(),
		BoardURL:   a.boardURL(),
		AdminURL:   a.adminURL(),
	}
	a.render(w, "capture", data)
}

func (a *App) handleBoard(w http.ResponseWriter, r *http.Request) {
	event := a.singleEvent()

	data := pageData{
		Title:      "rbooth Board",
		BaseURL:    a.baseURL,
		Event:      event,
		CaptureURL: a.captureURL(),
		BoardURL:   a.boardURL(),
		AdminURL:   a.adminURL(),
		Photos:     a.listAllPhotos(),
	}
	a.render(w, "board", data)
}

func (a *App) handleAdmin(w http.ResponseWriter, r *http.Request) {
	event := a.singleEvent()

	data := pageData{
		Title:      "Board Admin",
		BaseURL:    a.baseURL,
		Event:      event,
		CaptureURL: a.captureURL(),
		BoardURL:   a.boardURL(),
		AdminURL:   a.adminURL(),
		Photos:     a.listAllPhotos(),
	}
	a.render(w, "admin", data)
}

func (a *App) handleQR(w http.ResponseWriter, r *http.Request) {
	png, err := qrcode.Encode(a.publicCaptureURL(r), qrcode.Medium, 320)
	if err != nil {
		http.Error(w, "failed to generate qr", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(png)
}

func (a *App) handlePhotos(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"event":  a.singleEvent(),
		"photos": a.listAllPhotos(),
	})
}

func (a *App) handleUpload(w http.ResponseWriter, r *http.Request) {
	event := a.singleEvent()
	const maxUploadSize = 6 << 20

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		http.Error(w, "failed to parse upload", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "photo is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	payload, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
	if err != nil {
		http.Error(w, "failed to read upload", http.StatusBadRequest)
		return
	}
	if len(payload) == 0 {
		http.Error(w, "upload was empty", http.StatusBadRequest)
		return
	}
	if len(payload) > maxUploadSize {
		http.Error(w, "image exceeded the 6 MB limit", http.StatusBadRequest)
		return
	}

	contentType := sniffImageContentType(payload, header.Filename, header.Header.Get("Content-Type"))
	if !isAllowedImageType(contentType) {
		http.Error(w, "unsupported image type", http.StatusBadRequest)
		return
	}

	processed, err := normalizeJPEG(payload)
	if err != nil {
		http.Error(w, "image decode failed", http.StatusBadRequest)
		return
	}

	id := time.Now().UTC().Format("20060102150405") + "-" + strings.ToLower(randomCode(5))
	filename := id + ".jpg"
	storageKey := event.Code + "/" + filename
	if err := a.storage.Save(r.Context(), storageKey, "image/jpeg", processed); err != nil {
		http.Error(w, "failed to store image", http.StatusInternalServerError)
		return
	}

	photo := Photo{
		ID:          id,
		EventCode:   event.Code,
		Filename:    filename,
		StorageKey:  storageKey,
		Caption:     strings.TrimSpace(r.FormValue("caption")),
		CreatedAt:   time.Now().UTC(),
		DisplayURL:  "/media/" + storageKey,
		FilterLabel: strings.TrimSpace(r.FormValue("filterLabel")),
	}

	a.mu.Lock()
	a.photos[event.Code] = append([]Photo{photo}, a.photos[event.Code]...)
	a.mu.Unlock()

	if err := a.saveState(); err != nil {
		http.Error(w, "failed to persist photo", http.StatusInternalServerError)
		return
	}

	a.broadcast(singleBoardCode, photo)
	writeJSON(w, http.StatusCreated, map[string]any{"photo": photo})
}

func (a *App) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	updates := make(chan Photo, 8)
	a.addClient(singleBoardCode, updates)
	defer a.removeClient(singleBoardCode, updates)

	fmt.Fprint(w, "event: ready\ndata: ok\n\n")
	flusher.Flush()

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		case photo := <-updates:
			payload, err := json.Marshal(map[string]any{"photo": photo})
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: photo\ndata: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

func (a *App) handleNotFound(w http.ResponseWriter, r *http.Request) {
	data := pageData{
		Title:      "Page Not Found",
		BaseURL:    a.baseURL,
		CaptureURL: a.captureURL(),
		BoardURL:   a.boardURL(),
		AdminURL:   a.adminURL(),
	}
	a.renderStatus(w, http.StatusNotFound, "notfound", data)
}

func (a *App) handleMedia(w http.ResponseWriter, r *http.Request) {
	objectKey := strings.TrimPrefix(r.URL.Path, "/media/")
	if objectKey == "" {
		http.NotFound(w, r)
		return
	}

	reader, contentType, err := a.storage.Open(r.Context(), objectKey)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer reader.Close()

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = io.Copy(w, reader)
}

func (a *App) render(w http.ResponseWriter, name string, data any) {
	a.renderStatus(w, http.StatusOK, name, data)
}

func (a *App) renderStatus(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := a.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) getOrCreateEvent(code string) (*Event, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if existing, ok := a.events[code]; ok {
		copyEvent := *existing
		return &copyEvent, false
	}

	event := &Event{
		Code:      code,
		CreatedAt: time.Now().UTC(),
	}
	a.events[code] = event
	copyEvent := *event
	return &copyEvent, true
}

func (a *App) listPhotos(code string) []Photo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	photos := a.photos[code]
	result := make([]Photo, len(photos))
	copy(result, photos)
	return result
}

func (a *App) listAllPhotos() []Photo {
	return a.listPhotos(singleBoardCode)
}

func (a *App) addClient(code string, ch chan Photo) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.clients[code] == nil {
		a.clients[code] = make(map[chan Photo]struct{})
	}
	a.clients[code][ch] = struct{}{}
}

func (a *App) removeClient(code string, ch chan Photo) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if clients := a.clients[code]; clients != nil {
		delete(clients, ch)
		if len(clients) == 0 {
			delete(a.clients, code)
		}
	}
	close(ch)
}

func (a *App) broadcast(code string, photo Photo) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for ch := range a.clients[code] {
		select {
		case ch <- photo:
		default:
		}
	}
}

func (a *App) loadState() error {
	payload, err := os.ReadFile(a.storePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read state: %w", err)
	}

	var state persistedState
	if err := json.Unmarshal(payload, &state); err != nil {
		return fmt.Errorf("parse state: %w", err)
	}

	for _, event := range state.Events {
		copyEvent := event
		a.events[event.Code] = &copyEvent
	}
	for _, photo := range state.Photos {
		if photo.StorageKey == "" && photo.EventCode != "" && photo.Filename != "" {
			photo.StorageKey = photo.EventCode + "/" + photo.Filename
		}
		if photo.DisplayURL == "" && photo.StorageKey != "" {
			photo.DisplayURL = "/media/" + photo.StorageKey
		}
		a.photos[photo.EventCode] = append(a.photos[photo.EventCode], photo)
	}

	for code, photos := range a.photos {
		slices.SortFunc(photos, func(left, right Photo) int {
			if left.CreatedAt.Before(right.CreatedAt) {
				return 1
			}
			if left.CreatedAt.After(right.CreatedAt) {
				return -1
			}
			return 0
		})
		a.photos[code] = photos
	}

	return nil
}

func (a *App) saveState() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	events := make([]Event, 0, len(a.events))
	for _, event := range a.events {
		events = append(events, *event)
	}
	slices.SortFunc(events, func(left, right Event) int {
		if left.CreatedAt.Before(right.CreatedAt) {
			return -1
		}
		if left.CreatedAt.After(right.CreatedAt) {
			return 1
		}
		return 0
	})

	var photos []Photo
	for _, eventPhotos := range a.photos {
		photos = append(photos, eventPhotos...)
	}
	slices.SortFunc(photos, func(left, right Photo) int {
		if left.CreatedAt.Before(right.CreatedAt) {
			return -1
		}
		if left.CreatedAt.After(right.CreatedAt) {
			return 1
		}
		return 0
	})

	payload, err := json.MarshalIndent(persistedState{
		Events: events,
		Photos: photos,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := os.WriteFile(a.storePath, payload, 0o644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return nil
}

func (a *App) ensureDefaultEvent() {
	if _, created := a.getOrCreateEvent(singleBoardCode); created {
		if err := a.saveState(); err != nil {
			log.Printf("failed to save default event: %v", err)
		}
	}
}

func (a *App) ensureSamplePhotos() error {
	existing := a.listPhotos(singleBoardCode)
	if len(existing) >= 8 {
		return nil
	}

	existingByFile := make(map[string]struct{}, len(existing))
	for _, photo := range existing {
		existingByFile[photo.Filename] = struct{}{}
	}

	samples := []samplePhotoSpec{
		{Filename: "sample-daisy-lane.jpg", Caption: "Daisy lane", Filter: "warm/sunrise", Primary: rgba("#ffd7eb"), Secondary: rgba("#ffc2d9"), Accent: rgba("#fff2a8")},
		{Filename: "sample-soft-lights.jpg", Caption: "Soft lights", Filter: "party/paper", Primary: rgba("#f8d7ff"), Secondary: rgba("#f5b7dc"), Accent: rgba("#c4f1ff")},
		{Filename: "sample-cotton-candy.jpg", Caption: "Cotton candy", Filter: "warm/mint", Primary: rgba("#ffd4f1"), Secondary: rgba("#ffdfe8"), Accent: rgba("#b8f2e6")},
		{Filename: "sample-sparkle-hour.jpg", Caption: "Sparkle hour", Filter: "neutral/sunrise", Primary: rgba("#ffe3f0"), Secondary: rgba("#ffc8dd"), Accent: rgba("#ffe28a")},
		{Filename: "sample-bloom-room.jpg", Caption: "Bloom room", Filter: "cool/mint", Primary: rgba("#ffd9e8"), Secondary: rgba("#e8d7ff"), Accent: rgba("#bde0fe")},
		{Filename: "sample-ribbon-night.jpg", Caption: "Ribbon night", Filter: "cool/night", Primary: rgba("#e8d9ff"), Secondary: rgba("#ffc8dd"), Accent: rgba("#caffbf")},
		{Filename: "sample-rosy-pop.jpg", Caption: "Rosy pop", Filter: "party/sunrise", Primary: rgba("#ffcad4"), Secondary: rgba("#ffd6ff"), Accent: rgba("#fff3b0")},
		{Filename: "sample-dream-club.jpg", Caption: "Dream club", Filter: "neutral/paper", Primary: rgba("#fce1f0"), Secondary: rgba("#dfe7fd"), Accent: rgba("#cdeac0")},
	}

	created := false
	for index, sample := range samples {
		if _, ok := existingByFile[sample.Filename]; ok {
			continue
		}

		payload, err := generateSamplePhoto(sample)
		if err != nil {
			return err
		}

		storageKey := singleBoardCode + "/" + sample.Filename
		if err := a.storage.Save(context.Background(), storageKey, "image/jpeg", payload); err != nil {
			return fmt.Errorf("write sample photo: %w", err)
		}

		photo := Photo{
			ID:          fmt.Sprintf("sample-%02d", index+1),
			EventCode:   singleBoardCode,
			Filename:    sample.Filename,
			StorageKey:  storageKey,
			Caption:     sample.Caption,
			CreatedAt:   time.Now().UTC().Add(-time.Duration(len(samples)-index) * time.Minute),
			DisplayURL:  "/media/" + storageKey,
			FilterLabel: sample.Filter,
		}

		a.mu.Lock()
		a.photos[singleBoardCode] = append(a.photos[singleBoardCode], photo)
		a.mu.Unlock()
		created = true
	}

	if created {
		a.mu.Lock()
		slices.SortFunc(a.photos[singleBoardCode], func(left, right Photo) int {
			if left.CreatedAt.Before(right.CreatedAt) {
				return 1
			}
			if left.CreatedAt.After(right.CreatedAt) {
				return -1
			}
			return 0
		})
		a.mu.Unlock()

		if err := a.saveState(); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) singleEvent() *Event {
	a.mu.RLock()
	event := a.events[singleBoardCode]
	a.mu.RUnlock()
	if event == nil {
		event, _ = a.getOrCreateEvent(singleBoardCode)
	}
	copyEvent := *event
	return &copyEvent
}

func (a *App) captureURL() string {
	return "/capture"
}

func (a *App) boardURL() string {
	return "/photos"
}

func (a *App) adminURL() string {
	return "/admin"
}

func (a *App) publicCaptureURL(r *http.Request) string {
	return requestBaseURL(r, a.baseURL) + a.captureURL()
}

func requestBaseURL(r *http.Request, fallback string) string {
	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}

	if host != "" {
		return proto + "://" + host
	}

	return strings.TrimRight(fallback, "/")
}

func normalizeJPEG(payload []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func sniffImageContentType(payload []byte, filename string, declared string) string {
	sniffLength := len(payload)
	if sniffLength > 512 {
		sniffLength = 512
	}
	contentType := strings.ToLower(strings.TrimSpace(http.DetectContentType(payload[:sniffLength])))
	if semi := strings.Index(contentType, ";"); semi >= 0 {
		contentType = contentType[:semi]
	}
	if contentType != "application/octet-stream" {
		return contentType
	}

	declared = strings.ToLower(strings.TrimSpace(declared))
	if semi := strings.Index(declared, ";"); semi >= 0 {
		declared = declared[:semi]
	}
	if declared != "" {
		return declared
	}

	return strings.ToLower(mime.TypeByExtension(filepath.Ext(filename)))
}

func isAllowedImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "image/heic", "image/heif":
		return true
	default:
		return false
	}
}

func generateSamplePhoto(sample samplePhotoSpec) ([]byte, error) {
	const width = 900
	const height = 1100

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		t := float64(y) / float64(height-1)
		rowColor := mix(sample.Primary, sample.Secondary, t)
		for x := 0; x < width; x++ {
			col := rowColor
			if (x/36+y/42)%2 == 0 {
				col = mix(col, sample.Accent, 0.08)
			}
			img.SetRGBA(x, y, col)
		}
	}

	addBlob(img, width/4, height/4, 170, mix(sample.Accent, color.RGBA{255, 255, 255, 255}, 0.35))
	addBlob(img, width*3/4, height/3, 210, mix(sample.Secondary, color.RGBA{255, 255, 255, 255}, 0.22))
	addBlob(img, width/2, height*3/4, 250, mix(sample.Primary, sample.Accent, 0.35))

	for y := 80; y < height-80; y++ {
		for x := 80; x < width-80; x++ {
			if x < 96 || y < 96 || x > width-97 || y > height-97 {
				img.SetRGBA(x, y, mix(img.RGBAAt(x, y), color.RGBA{255, 255, 255, 255}, 0.45))
			}
		}
	}

	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, fmt.Errorf("encode sample photo: %w", err)
	}
	return buffer.Bytes(), nil
}

func addBlob(img *image.RGBA, centerX, centerY, radius int, tone color.RGBA) {
	bounds := img.Bounds()
	for y := centerY - radius; y <= centerY+radius; y++ {
		if y < bounds.Min.Y || y >= bounds.Max.Y {
			continue
		}
		for x := centerX - radius; x <= centerX+radius; x++ {
			if x < bounds.Min.X || x >= bounds.Max.X {
				continue
			}
			dx := float64(x-centerX) / float64(radius)
			dy := float64(y-centerY) / float64(radius)
			distance := dx*dx + dy*dy
			if distance > 1 {
				continue
			}
			alpha := 0.42 * (1 - distance)
			img.SetRGBA(x, y, mix(img.RGBAAt(x, y), tone, alpha))
		}
	}
}

func mix(left, right color.RGBA, ratio float64) color.RGBA {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return color.RGBA{
		R: uint8(float64(left.R)*(1-ratio) + float64(right.R)*ratio),
		G: uint8(float64(left.G)*(1-ratio) + float64(right.G)*ratio),
		B: uint8(float64(left.B)*(1-ratio) + float64(right.B)*ratio),
		A: 255,
	}
}

func rgba(hex string) color.RGBA {
	if len(hex) != 7 || hex[0] != '#' {
		return color.RGBA{255, 220, 235, 255}
	}
	parse := func(value byte) uint8 {
		switch {
		case value >= '0' && value <= '9':
			return value - '0'
		case value >= 'a' && value <= 'f':
			return value - 'a' + 10
		case value >= 'A' && value <= 'F':
			return value - 'A' + 10
		default:
			return 0
		}
	}
	return color.RGBA{
		R: parse(hex[1])<<4 | parse(hex[2]),
		G: parse(hex[3])<<4 | parse(hex[4]),
		B: parse(hex[5])<<4 | parse(hex[6]),
		A: 255,
	}
}

func randomCode(length int) string {
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "alpha"
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	return encoded[:length]
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if builder.Len() > 0 && !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return ""
	}
	if len(result) > 24 {
		return result[:24]
	}
	return result
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func redirectTo(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
