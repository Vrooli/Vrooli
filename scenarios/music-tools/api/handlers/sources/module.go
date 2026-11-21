// Package sources exposes private managed-source import and streaming reads.
package sources

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"music-tools/internal/module"
	sourcestore "music-tools/internal/sources"
	musicstorage "music-tools/internal/storage"
)

const maxMultipartBytes = sourcestore.MaxFileBytes + (2 << 20)

var shaPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type handler struct{ store *sourcestore.Store }

func Module(db sourcestore.SQLExecutor, blobs *musicstorage.Store) (module.Module, error) {
	store, err := sourcestore.New(db, blobs)
	if err != nil {
		return module.Module{}, err
	}
	h := &handler{store: store}
	return module.Module{Name: "sources", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/sources", h.upload).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/sources/{sha256}", h.get).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/sources/{sha256}/audio", h.audio).Methods(http.MethodGet, http.MethodHead)
	}, Endpoints: Endpoints}, nil
}

func (h *handler) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxMultipartBytes)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "source upload exceeds 128 MiB", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "invalid multipart source upload", http.StatusBadRequest)
		}
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
		if len(r.MultipartForm.File) != 1 || len(r.MultipartForm.File["file"]) != 1 {
			http.Error(w, "exactly one file part named file is required", http.StatusBadRequest)
			return
		}
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "audio file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	if header.Size <= 0 || header.Size > sourcestore.MaxFileBytes {
		http.Error(w, "audio file must be between 1 byte and 128 MiB", http.StatusRequestEntityTooLarge)
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	artist := strings.TrimSpace(r.FormValue("artist"))
	uploader := strings.TrimSpace(r.FormValue("uploader"))
	sourceURL := strings.TrimSpace(r.FormValue("source_url"))
	acquiredAt := strings.TrimSpace(r.FormValue("acquired_at"))
	if title == "" || len(title) > 512 || len(artist) > 256 || len(uploader) > 256 || len(sourceURL) > 2048 {
		http.Error(w, "title is required; provenance fields exceed their limits", http.StatusBadRequest)
		return
	}
	if sourceURL != "" && !isAllowedSourceURL(sourceURL) {
		http.Error(w, "source_url must be an HTTPS YouTube URL; it is stored as provenance and never fetched", http.StatusBadRequest)
		return
	}
	if acquiredAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, acquiredAt); err != nil {
			http.Error(w, "acquired_at must be an RFC3339 timestamp", http.StatusBadRequest)
			return
		}
	}

	contentHash := sha256.New()
	size, err := io.CopyN(contentHash, file, sourcestore.MaxFileBytes+1)
	if err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "could not read source upload", http.StatusBadRequest)
		return
	}
	if size <= 0 || size > sourcestore.MaxFileBytes || size != header.Size {
		http.Error(w, "source upload size is invalid or exceeds 128 MiB", http.StatusRequestEntityTooLarge)
		return
	}
	digest := hex.EncodeToString(contentHash.Sum(nil))
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "source upload cannot be rewound safely", http.StatusBadRequest)
		return
	}
	var prefix [512]byte
	n, _ := io.ReadFull(file, prefix[:])
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "source upload cannot be rewound safely", http.StatusBadRequest)
		return
	}
	extension := strings.ToLower(filepath.Ext(filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))))
	mediaType := detectAudioType(extension, prefix[:n])
	if mediaType == "" {
		http.Error(w, "supported audio file required (AAC/M4A, MP3, WAV, FLAC, OGG or Opus)", http.StatusUnsupportedMediaType)
		return
	}
	if !safeAudioExtension(extension) {
		extension = ".audio"
	}
	filename := "source-" + digest + extension
	metadata := sourcestore.Record{SHA256: digest, Title: title, Artist: artist, Uploader: uploader,
		SourceURL: sourceURL, Acquired: acquiredAt, FileName: filename, MediaType: mediaType, SizeBytes: size}
	record, err := h.store.Import(r.Context(), metadata, file)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sourcestore.ErrQuotaExceeded) {
			status = http.StatusInsufficientStorage
		} else if errors.Is(err, sourcestore.ErrConflict) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, http.StatusCreated, record)
}

func Schema() string { return sourcestore.Schema() }

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	digest := mux.Vars(r)["sha256"]
	if !shaPattern.MatchString(digest) {
		http.Error(w, "source SHA-256 is invalid", http.StatusBadRequest)
		return
	}
	record, err := h.store.Get(r.Context(), digest)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "source metadata unavailable", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (h *handler) audio(w http.ResponseWriter, r *http.Request) {
	digest := mux.Vars(r)["sha256"]
	if !shaPattern.MatchString(digest) {
		http.Error(w, "source SHA-256 is invalid", http.StatusBadRequest)
		return
	}
	record, reader, err := h.store.OpenAudio(r.Context(), digest)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "source audio unavailable", http.StatusInternalServerError)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", record.MediaType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": record.FileName}))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if seeker, ok := reader.(io.ReadSeeker); ok {
		http.ServeContent(w, r, record.FileName, record.CreatedAt, seeker)
		return
	}
	data, err := io.ReadAll(io.LimitReader(reader, sourcestore.MaxFileBytes+1))
	if err != nil || int64(len(data)) != record.SizeBytes {
		http.Error(w, "source content failed integrity-length verification", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, record.FileName, record.CreatedAt, bytes.NewReader(data))
}

func detectAudioType(extension string, prefix []byte) string {
	detected := http.DetectContentType(prefix)
	if strings.HasPrefix(detected, "audio/") {
		return detected
	}
	switch extension {
	case ".m4a":
		return "audio/mp4"
	case ".aac":
		return "audio/aac"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".ogg", ".opus":
		return "audio/ogg"
	}
	if typ := mime.TypeByExtension(extension); strings.HasPrefix(typ, "audio/") {
		return typ
	}
	return ""
}

func safeAudioExtension(extension string) bool {
	switch extension {
	case ".m4a", ".aac", ".mp3", ".wav", ".flac", ".ogg", ".opus":
		return true
	default:
		return false
	}
}

func isAllowedSourceURL(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "youtube.com" || host == "youtu.be" || strings.HasSuffix(host, ".youtube.com")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
