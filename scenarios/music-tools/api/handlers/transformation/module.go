// Package transformation exposes the owner-run deterministic edit operations.
package transformation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"music-tools/internal/jobs"
	"music-tools/internal/module"
	musicstorage "music-tools/internal/storage"
	domain "music-tools/internal/transformation"
)

type handler struct {
	manager *jobs.Manager
	blobs   *musicstorage.Store
}

func RegisterOperations(manager *jobs.Manager, blobs *musicstorage.Store) error {
	return domain.Register(manager, blobs)
}

func Module(manager *jobs.Manager, blobs *musicstorage.Store) (module.Module, error) {
	if manager == nil || blobs == nil {
		return module.Module{}, errors.New("transformation routes require the shared jobs manager and managed blob store")
	}
	h := &handler{manager: manager, blobs: blobs}
	return module.Module{Name: "transformation", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/transformations/remix", h.submitRemix).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/transformations/sample", h.submitSample).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/jobs/{id}/audio", h.audio).Methods(http.MethodGet, http.MethodHead)
	}, Endpoints: Endpoints}, nil
}

func (h *handler) submitRemix(w http.ResponseWriter, r *http.Request) {
	h.submit(w, r, "remix")
}

func (h *handler) submitSample(w http.ResponseWriter, r *http.Request) {
	h.submit(w, r, "sample")
}

func (h *handler) submit(w http.ResponseWriter, r *http.Request, kind string) {
	var input domain.Request
	if !decode(w, r, &input) {
		return
	}
	if input.Kind != "" && input.Kind != kind {
		http.Error(w, "request kind must match the route", http.StatusBadRequest)
		return
	}
	input.Kind = kind
	job, err := domain.Submit(r.Context(), h.manager, input)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, domain.ErrInvalidTransform) {
			status = http.StatusBadRequest
		} else if errors.Is(err, jobs.ErrQueueFull) || errors.Is(err, jobs.ErrNotStarted) {
			status = http.StatusServiceUnavailable
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": job.ID, "operation": job.Operation, "state": string(job.State)})
}

func (h *handler) audio(w http.ResponseWriter, r *http.Request) {
	job, err := h.manager.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		if errors.Is(err, jobs.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "transformation job is unavailable", http.StatusServiceUnavailable)
		return
	}
	receipt, err := domain.ReadJobReceipt(job)
	if err != nil {
		http.Error(w, "job has no verified derived audio", http.StatusPreconditionFailed)
		return
	}
	reader, mediaType, err := h.blobs.Get(r.Context(), receipt.ResultRef)
	if err != nil {
		http.Error(w, "derived audio is unavailable", http.StatusServiceUnavailable)
		return
	}
	defer reader.Close()
	if mediaType != "audio/wav" {
		http.Error(w, "derived audio has an unexpected media type", http.StatusBadGateway)
		return
	}
	temp, err := os.CreateTemp("", "music-tools-derived-audio-*.wav")
	if err != nil {
		http.Error(w, "could not stage derived audio for verified streaming", http.StatusInternalServerError)
		return
	}
	name := temp.Name()
	defer os.Remove(name)
	hasher := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(temp, hasher), io.LimitReader(reader, domain.MaxOutputBytes+1))
	closeErr := temp.Close()
	if copyErr != nil || closeErr != nil || size <= 0 || size > domain.MaxOutputBytes || hex.EncodeToString(hasher.Sum(nil)) != receipt.DerivedTrackSHA256 {
		http.Error(w, "stored audio does not match the owner derivation receipt", http.StatusBadGateway)
		return
	}
	file, err := os.Open(name)
	if err != nil {
		http.Error(w, "could not reopen verified derived audio", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("ETag", fmt.Sprintf("\"sha256:%s\"", receipt.DerivedTrackSHA256))
	http.ServeContent(w, r, "derived.wav", time.Time{}, file)
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "invalid transformation request: "+err.Error(), http.StatusBadRequest)
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		http.Error(w, "request must contain exactly one JSON value", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
