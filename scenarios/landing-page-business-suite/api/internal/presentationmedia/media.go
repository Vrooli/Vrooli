// Package presentationmedia owns content-addressed local MP4 bytes for
// presentation drafts. It is deliberately separate from the Backdrop PNG
// release/cache verifier.
package presentationmedia

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/packages/capacity"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/presentation"
)

const (
	dirName          = "presentation-media"
	maxMP4Bytes      = int64(256 << 20)
	minimumFreeBytes = int64(120 << 30)
)

var hashPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type Handler struct {
	roots         *filerouting.RoutedRoots
	presentations *experimentation.ConfigStore
}

// VerifyPublication is read-only and verifies every local MP4 reference as
// exact content-addressed bytes before a document can become public.
func (h *Handler) VerifyPublication(ctx context.Context, document presentation.Document, _ *presentation.Document) error {
	if h == nil || h.roots == nil {
		return errors.New("local presentation media owner unavailable")
	}
	views, err := presentation.PublicViews(document)
	if err != nil {
		return err
	}
	hashes := map[string]bool{}
	collectMediaHashes(views, hashes)
	rootPath, err := h.roots.PickRequired(ctx, storage.ClassData)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return err
	}
	defer root.Close()
	for hash := range hashes {
		if !hashPattern.MatchString(hash) {
			return fmt.Errorf("invalid local media hash")
		}
		f, err := root.Open(filepath.Join(dirName, hash+".mp4"))
		if err != nil {
			return fmt.Errorf("local media %s is unavailable", hash)
		}
		digest := sha256.New()
		size, copyErr := io.Copy(digest, io.LimitReader(f, maxMP4Bytes+1))
		closeErr := f.Close()
		if copyErr != nil || closeErr != nil || size <= 0 || size > maxMP4Bytes || hex.EncodeToString(digest.Sum(nil)) != hash {
			return fmt.Errorf("local media %s failed exact-byte verification", hash)
		}
	}
	return nil
}

func collectMediaHashes(views []presentation.ResolveResult, hashes map[string]bool) {
	for _, view := range views {
		for _, block := range view.Page.Blocks {
			if demo, ok := block.Content.(presentation.ProductDemoContent); ok && demo.Playback != nil && demo.Playback.Provider == "local" {
				hashes[demo.MediaRef] = true
			}
		}
	}
}

func New(roots *filerouting.RoutedRoots, presentations *experimentation.ConfigStore) *Handler {
	return &Handler{roots: roots, presentations: presentations}
}

// Upload accepts a raw, bounded video/mp4 body. The broker's floor and exact
// peak reservation are acquired before creating the staging file.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.ContentLength <= 0 || r.ContentLength > maxMP4Bytes || !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "video/mp4") {
		http.Error(w, "expected a bounded video/mp4 body with Content-Length", http.StatusBadRequest)
		return
	}
	rootPath, err := h.roots.PickRequired(r.Context(), storage.ClassData)
	if err != nil {
		http.Error(w, "media storage route unavailable", http.StatusServiceUnavailable)
		return
	}
	jobID, err := randomID()
	if err != nil {
		http.Error(w, "cannot create media job identity", http.StatusInternalServerError)
		return
	}
	broker, err := capacity.NewBroker(r.Context(), "")
	if err != nil {
		http.Error(w, "shared disk capacity owner unavailable", http.StatusServiceUnavailable)
		return
	}
	lease, err := broker.ReserveDiskJob(r.Context(), capacity.DiskJobRequest{
		OwnerID: "landing-page-business-suite", JobID: jobID,
		Paths:      []capacity.DiskJobPath{{Path: rootPath, Role: "presentation-local-mp4", PeakBytes: r.ContentLength}},
		FloorBytes: minimumFreeBytes,
	})
	if err != nil {
		_ = broker.Close()
		http.Error(w, "shared disk admission denied", http.StatusServiceUnavailable)
		return
	}
	receipt := lease.Reservation()
	if receipt.ID == "" || receipt.Status != "active" || receipt.OwnerID != "landing-page-business-suite" || receipt.JobID != jobID || len(receipt.Paths) != 1 || len(receipt.Filesystems) != 1 || receipt.Paths[0].PeakBytes < r.ContentLength || receipt.Filesystems[0].FloorBytes < minimumFreeBytes {
		_ = lease.Release(r.Context())
		_ = broker.Close()
		http.Error(w, "shared disk admission receipt is incomplete", http.StatusServiceUnavailable)
		return
	}
	defer broker.Close()
	finished := false
	defer func() {
		if !finished {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = lease.Release(cleanupCtx)
		}
	}()
	writeCtx, cancelWrite := context.WithCancel(r.Context())
	defer cancelWrite()
	stopHeartbeat := make(chan struct{})
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopHeartbeat:
				return
			case <-writeCtx.Done():
				return
			case <-ticker.C:
				if lease.Heartbeat(writeCtx) != nil {
					cancelWrite()
					_ = r.Body.Close()
					return
				}
			}
		}
	}()
	hash, size, err := h.put(writeCtx, rootPath, http.MaxBytesReader(w, r.Body, maxMP4Bytes+1))
	close(stopHeartbeat)
	<-heartbeatDone
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	completeCtx, cancelComplete := context.WithTimeout(context.Background(), 10*time.Second)
	completed, err := lease.Complete(completeCtx)
	cancelComplete()
	if err != nil || completed.Status != "completed" || !completed.CleanupVerified {
		http.Error(w, "media write receipt did not complete cleanly", http.StatusServiceUnavailable)
		return
	}
	finished = true
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprintf(w, `{"sha256":%q,"bytes":%d,"mime":"video/mp4"}`, hash, size)
}

func (h *Handler) put(ctx context.Context, rootPath string, src io.Reader) (string, int64, error) {
	if h == nil || h.roots == nil {
		return "", 0, errors.New("presentation media owner unavailable")
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return "", 0, err
	}
	defer root.Close()
	if err := root.MkdirAll(dirName, 0700); err != nil {
		return "", 0, err
	}
	stage, err := randomID()
	if err != nil {
		return "", 0, err
	}
	stagePath := filepath.Join(dirName, ".upload-"+stage)
	f, err := root.OpenFile(stagePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", 0, err
	}
	committed := false
	defer func() {
		_ = f.Close()
		if !committed {
			_ = root.Remove(stagePath)
		}
	}()
	hash := sha256.New()
	tee := io.TeeReader(io.LimitReader(src, maxMP4Bytes+1), hash)
	buf := make([]byte, 32*1024)
	var size int64
	prefix := make([]byte, 0, 12)
	for {
		if err := ctx.Err(); err != nil {
			return "", size, err
		}
		n, readErr := tee.Read(buf)
		if n > 0 {
			if size+int64(n) > maxMP4Bytes {
				return "", size, errors.New("MP4 exceeds 256 MiB limit")
			}
			if len(prefix) < 12 {
				take := n
				if take > 12-len(prefix) {
					take = 12 - len(prefix)
				}
				prefix = append(prefix, buf[:take]...)
			}
			if _, err := f.Write(buf[:n]); err != nil {
				return "", size, err
			}
			size += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", size, readErr
		}
	}
	if size == 0 || len(prefix) < 8 || string(prefix[4:8]) != "ftyp" {
		return "", size, errors.New("bytes do not have an MP4 ftyp signature")
	}
	if err := f.Sync(); err != nil {
		return "", size, err
	}
	if err := f.Close(); err != nil {
		return "", size, err
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	target := filepath.Join(dirName, digest+".mp4")
	if existing, err := root.Open(target); err == nil {
		defer existing.Close()
		existingHash := sha256.New()
		if _, err := io.Copy(existingHash, io.LimitReader(existing, maxMP4Bytes+1)); err != nil || hex.EncodeToString(existingHash.Sum(nil)) != digest {
			return "", size, errors.New("immutable local MP4 hash collision or corruption")
		}
		_ = root.Remove(stagePath)
		committed = true
		h.roots.RecordWrite(ctx)
		return digest, size, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", size, err
	}
	if err := root.Rename(stagePath, target); err != nil {
		return "", size, err
	}
	committed = true
	h.roots.RecordWrite(ctx)
	return digest, size, nil
}

func (h *Handler) DraftPlayback(w http.ResponseWriter, r *http.Request)     { h.serve(w, r, true) }
func (h *Handler) PublishedPlayback(w http.ResponseWriter, r *http.Request) { h.serve(w, r, false) }

// DeleteUnreferenced releases an uploaded object only when no draft or retained
// published revision references it. It cannot delete in-use media.
func (h *Handler) DeleteUnreferenced(w http.ResponseWriter, r *http.Request) {
	hash := mux.Vars(r)["hash"]
	if !hashPattern.MatchString(hash) {
		http.NotFound(w, r)
		return
	}
	for _, variant := range h.presentations.ListVariants() {
		if variant == nil {
			continue
		}
		state, err := h.presentations.GetPresentationState(r.Context(), variant.Variant.Slug)
		if err != nil {
			http.Error(w, "cannot verify presentation references", http.StatusServiceUnavailable)
			return
		}
		revisions := append([]string{}, state.PublishedRevisions...)
		if state.DraftRevision != "" {
			revisions = append(revisions, state.DraftRevision)
		}
		for _, revision := range revisions {
			loaded, err := h.presentations.GetPresentationRevision(r.Context(), variant.Variant.Slug, revision)
			if err != nil {
				http.Error(w, "cannot verify presentation references", http.StatusServiceUnavailable)
				return
			}
			if documentUsesHash(loaded.Document, hash) {
				http.Error(w, "media is still referenced by a draft or retained published revision", http.StatusConflict)
				return
			}
		}
	}
	rootPath, err := h.roots.PickRequired(r.Context(), storage.ClassData)
	if err != nil {
		http.Error(w, "media storage route unavailable", http.StatusServiceUnavailable)
		return
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer root.Close()
	if err := root.Remove(filepath.Join(dirName, hash+".mp4")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "media could not be removed", http.StatusInternalServerError)
		}
		return
	}
	h.roots.RecordWrite(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request, draft bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	variant, revision, hash := vars["variant"], vars["revision"], vars["hash"]
	if !hashPattern.MatchString(hash) {
		http.NotFound(w, r)
		return
	}
	state, err := h.presentations.GetPresentationState(r.Context(), variant)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if draft {
		if state.DraftRevision != revision {
			http.NotFound(w, r)
			return
		}
	} else if state.ActiveRevision != revision {
		http.NotFound(w, r)
		return
	}
	loaded, err := h.presentations.GetPresentationRevision(r.Context(), variant, revision)
	if err != nil || !documentUsesHash(loaded.Document, hash) {
		http.NotFound(w, r)
		return
	}
	if !draft {
		views, err := presentation.PublicViews(loaded.Document)
		if err != nil || !viewsUseHash(views, hash) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "private, no-store")
	}
	rootPath, err := h.roots.PickRequired(r.Context(), storage.ClassData)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer root.Close()
	f, err := root.Open(filepath.Join(dirName, hash+".mp4"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() <= 0 || info.Size() > maxMP4Bytes {
		http.NotFound(w, r)
		return
	}
	verified := sha256.New()
	if _, err := io.Copy(verified, io.LimitReader(f, maxMP4Bytes+1)); err != nil || hex.EncodeToString(verified.Sum(nil)) != hash {
		http.NotFound(w, r)
		return
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("ETag", `"`+hash+`"`)
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, hash+".mp4", info.ModTime(), f)
}

func documentUsesHash(doc presentation.Document, hash string) bool {
	for _, page := range doc.Pages {
		for _, block := range page.Blocks {
			if demo, ok := block.Content.(presentation.ProductDemoContent); ok && demo.Playback != nil && demo.Playback.Provider == "local" && demo.MediaRef == hash {
				return true
			}
		}
	}
	return false
}
func viewsUseHash(views []presentation.ResolveResult, hash string) bool {
	for _, view := range views {
		for _, block := range view.Page.Blocks {
			if demo, ok := block.Content.(presentation.ProductDemoContent); ok && demo.Playback != nil && demo.Playback.Provider == "local" && demo.MediaRef == hash {
				return true
			}
		}
	}
	return false
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
