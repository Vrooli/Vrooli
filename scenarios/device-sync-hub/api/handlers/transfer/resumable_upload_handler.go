package transfer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"device-sync-hub/internal/deviceauth"
	"device-sync-hub/internal/httpx"
	internaltransfer "device-sync-hub/internal/transfer"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	transferv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer"
)

func (h *uploadHandler) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	dev, err := deviceauth.RequireDevice(r.Context())
	if err != nil {
		httpx.WriteError(w, 401, httpx.CodeUnauthenticated, "a trusted device token is required")
		return
	}
	if h.deps.DB == nil {
		httpx.WriteError(w, 500, httpx.CodeInternal, "resumable upload unavailable")
		return
	}
	var in createSessionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil || strings.TrimSpace(in.Name) == "" || in.SizeBytes <= 0 || in.SizeBytes > maxUploadBytes {
		httpx.WriteError(w, 400, httpx.CodeInvalidRequest, "name and a file size up to 2 GiB are required")
		return
	}
	if err := h.deps.Service.CheckQuota(r.Context(), dev.OwnerID, dev.ID, in.SizeBytes); err != nil {
		h.writeServiceError(w, err)
		return
	}
	count := int((in.SizeBytes + chunkUploadBytes - 1) / chunkUploadBytes)
	s := uploadSession{ID: uuid.NewString(), Name: uploadName(in.Name, in.Name), MIME: strings.TrimSpace(in.MIME), SizeBytes: in.SizeBytes, Retention: strings.ToLower(strings.TrimSpace(in.Retention)), TargetDeviceID: strings.TrimSpace(in.TargetDeviceID), ChunkCount: count, Received: []int{}}
	if s.MIME == "" {
		s.MIME = "application/octet-stream"
	}
	_, err = h.deps.DB.ExecContext(r.Context(), `INSERT INTO upload_sessions (id, owner_id, device_id, name, mime, size_bytes, retention, target_device_id, chunk_count, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`, s.ID, dev.OwnerID, dev.ID, s.Name, s.MIME, s.SizeBytes, s.Retention, s.TargetDeviceID, s.ChunkCount)
	if err != nil {
		httpx.WriteError(w, 500, httpx.CodeInternal, "create upload session failed")
		return
	}
	h.writeJSON(w, http.StatusCreated, s)
}

func (h *uploadHandler) handleUploadStatus(w http.ResponseWriter, r *http.Request) {
	s, _, ok := h.sessionFor(w, r, mux.Vars(r)["id"])
	if ok {
		h.writeJSON(w, 200, s)
	}
}

func (h *uploadHandler) handleChunk(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	s, dev, ok := h.sessionFor(w, r, id)
	if !ok {
		return
	}
	index, err := strconv.Atoi(mux.Vars(r)["index"])
	if err != nil || index < 0 || index >= s.ChunkCount {
		httpx.WriteError(w, 400, httpx.CodeInvalidRequest, "invalid chunk index")
		return
	}
	expected := chunkUploadBytes
	if index == s.ChunkCount-1 {
		expected = s.SizeBytes - int64(index)*chunkUploadBytes
	}
	r.Body = http.MaxBytesReader(w, r.Body, expected+1)
	data, err := io.ReadAll(r.Body)
	if err != nil || int64(len(data)) != expected {
		httpx.WriteError(w, 400, httpx.CodeInvalidRequest, "chunk size does not match its expected range")
		return
	}
	key := fmt.Sprintf("transfer-staging/%s/%s/%06d", safeOwnerSegment(dev.OwnerID), id, index)
	if err := h.deps.Store.Put(r.Context(), key, bytes.NewReader(data), "application/octet-stream"); err != nil {
		httpx.WriteError(w, 500, httpx.CodeInternal, "store upload chunk failed")
		return
	}
	// BlobStore writes atomically. Replacing an already-received chunk makes retry idempotent.
	_, err = h.deps.DB.ExecContext(r.Context(), `INSERT INTO upload_chunks (session_id, chunk_index, size_bytes, blob_key) VALUES (?, ?, ?, ?) ON CONFLICT(session_id, chunk_index) DO UPDATE SET size_bytes=excluded.size_bytes, blob_key=excluded.blob_key`, id, index, expected, key)
	if err != nil {
		_ = h.deps.Store.Delete(r.Context(), key)
		httpx.WriteError(w, 500, httpx.CodeInternal, "record upload chunk failed")
		return
	}
	h.writeJSON(w, 200, map[string]any{"received": index})
}

func (h *uploadHandler) handleCompleteSession(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	s, dev, ok := h.sessionFor(w, r, id)
	if !ok {
		return
	}
	if len(s.Received) != s.ChunkCount {
		httpx.WriteError(w, 409, httpx.CodeInvalidRequest, "upload is incomplete")
		return
	}
	rows, err := h.deps.DB.QueryContext(r.Context(), `SELECT blob_key FROM upload_chunks WHERE session_id = ? ORDER BY chunk_index`, id)
	if err != nil {
		httpx.WriteError(w, 500, httpx.CodeInternal, "load upload chunks failed")
		return
	}
	defer rows.Close()
	var readers []io.ReadCloser
	var parts []io.Reader
	var keys []string
	for rows.Next() {
		var key string
		if rows.Scan(&key) != nil {
			httpx.WriteError(w, 500, httpx.CodeInternal, "read upload chunks failed")
			return
		}
		rd, _, e := h.deps.Store.Get(r.Context(), key)
		if e != nil {
			httpx.WriteError(w, 500, httpx.CodeInternal, "read upload chunk failed")
			return
		}
		readers = append(readers, rd)
		parts = append(parts, rd)
		keys = append(keys, key)
	}
	defer func() {
		for _, rd := range readers {
			_ = rd.Close()
		}
	}()
	blobKey := itemBlobKey(dev.OwnerID, s.Name)
	if err := h.deps.Store.Put(r.Context(), blobKey, io.MultiReader(parts...), s.MIME); err != nil {
		httpx.WriteError(w, 500, httpx.CodeInternal, "assemble upload failed")
		return
	}
	item, err := h.deps.Service.CreateFile(r.Context(), internaltransfer.CreateFile{OwnerID: dev.OwnerID, OriginDeviceID: dev.ID, Name: s.Name, MIME: s.MIME, SizeBytes: s.SizeBytes, BlobKey: blobKey, Retention: parseRetention(s.Retention), TargetDeviceID: s.TargetDeviceID})
	if err != nil {
		_ = h.deps.Store.Delete(r.Context(), blobKey)
		h.writeServiceError(w, err)
		return
	}
	_, _ = h.deps.DB.ExecContext(r.Context(), `UPDATE upload_sessions SET completed=1 WHERE id=?`, id)
	for _, key := range keys {
		_ = h.deps.Store.Delete(r.Context(), key)
	}
	httpx.WriteProto(w, http.StatusCreated, &transferv1.UploadItemResponse{Item: itemToProto(item)})
}
