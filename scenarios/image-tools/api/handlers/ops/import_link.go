package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/fetch"
	"image-tools/internal/httpx"
	"image-tools/internal/storage"

	"github.com/google/uuid"
)

type RemoteImageFetcher interface {
	Fetch(context.Context, string) ([]byte, string, error)
}

type ImportLinkStore interface {
	Put(context.Context, string, io.Reader, string) error
}

type importLinkRequest struct {
	URL string `json:"url"`
}

type importLinkResponse struct {
	InputRef string `json:"input_ref"`
	Format   string `json:"format"`
	MIME     string `json:"mime_type"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// importLinkHandler fetches one public image URL into Image Tools-owned blob
// storage. The URL is fetched server-side only after the fetcher's public-IP
// checks; callers receive an opaque ref usable by existing image operations.
const importLinkPeakBytes int64 = 64 << 20
const importLinkFreeFloorBytes int64 = 120 << 30
const importLinkHeartbeatInterval = 10 * time.Second

func importLinkHandler(store ImportLinkStore, fetcher RemoteImageFetcher, guard storage.Guard, disk sharedcapacity.DiskJobAdmitter, reservationRoot func(context.Context) (string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			httpx.WriteError(w, http.StatusMethodNotAllowed, httpx.CodeInvalidRequest, "method not allowed")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		var req importLinkRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil || strings.TrimSpace(req.URL) == "" {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "provide a valid image URL")
			return
		}
		var extra any
		if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "request must contain one JSON object")
			return
		}
		if disk == nil || reservationRoot == nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "disk admission is unavailable")
			return
		}
		root, err := reservationRoot(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "could not prepare import storage")
			return
		}
		lease, err := disk.ReserveDiskJob(r.Context(), sharedcapacity.DiskJobRequest{
			OwnerID:    "image-tools:link-import",
			JobID:      uuid.NewString(),
			Paths:      []sharedcapacity.DiskJobPath{{Path: root, Role: "imported-input-image", PeakBytes: importLinkPeakBytes}},
			FloorBytes: importLinkFreeFloorBytes,
		})
		if err != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "import storage capacity is unavailable")
			return
		}
		jobCtx, cancelJob, heartbeatDone := maintainImportLease(r.Context(), lease, importLinkHeartbeatInterval)
		complete := func() error {
			cancelJob()
			<-heartbeatDone
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
			defer cancel()
			_, err := lease.Complete(ctx)
			return err
		}
		stopHeartbeat := func() {
			cancelJob()
			<-heartbeatDone
		}
		data, _, err := fetcher.Fetch(jobCtx, req.URL)
		if err != nil {
			if completeErr := complete(); completeErr != nil {
				httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to finalize import capacity receipt")
				return
			}
			status := http.StatusUnprocessableEntity
			if errors.Is(err, fetch.ErrUnsafeRemoteURL) {
				status = http.StatusBadRequest
			}
			if errors.Is(err, fetch.ErrRemoteImageTooLarge) {
				status = http.StatusRequestEntityTooLarge
			}
			httpx.WriteError(w, status, httpx.CodeInvalidRequest, err.Error())
			return
		}
		inspected, err := guard.Inspect(bytes.NewReader(data))
		if err != nil {
			if completeErr := complete(); completeErr != nil {
				httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to finalize import capacity receipt")
				return
			}
			status := http.StatusUnprocessableEntity
			if errors.Is(err, storage.ErrTooLarge) {
				status = http.StatusRequestEntityTooLarge
			}
			httpx.WriteError(w, status, httpx.CodeInvalidRequest, err.Error())
			return
		}
		if inspected.Format == "" {
			if completeErr := complete(); completeErr != nil {
				httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to finalize import capacity receipt")
				return
			}
			httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, "URL did not return a supported raster image")
			return
		}
		mime := "image/" + inspected.Format
		if inspected.Format == "jpeg" {
			mime = "image/jpeg"
		}
		ref := "input/" + uuid.NewString() + "." + inspected.Format
		if err := store.Put(jobCtx, ref, bytes.NewReader(inspected.Bytes), mime); err != nil {
			// Put may fail after creating staging bytes or publishing the blob.
			// The store abstraction does not prove that every such artifact is
			// absent, so keep the disk lease active for owner reconciliation.
			stopHeartbeat()
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to store imported image; disk capacity remains reserved pending cleanup verification")
			return
		}
		if err := complete(); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to finalize import capacity receipt")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(importLinkResponse{InputRef: ref, Format: inspected.Format, MIME: mime, Width: inspected.Width, Height: inspected.Height})
	}
}

func maintainImportLease(parent context.Context, lease sharedcapacity.DiskJobLease, interval time.Duration) (context.Context, context.CancelFunc, <-chan struct{}) {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := lease.Heartbeat(ctx); err != nil {
					cancel()
					return
				}
			}
		}
	}()
	return ctx, cancel, done
}
