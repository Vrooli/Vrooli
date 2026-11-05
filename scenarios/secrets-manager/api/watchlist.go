package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
)

const watchlistEncryptionKeyEnv = "SECRETS_MANAGER_WATCHLIST_KEY"

var (
	errWatchlistKeyNotConfigured = errors.New("watchlist_encryption_key_not_configured")
	errWatchlistKeyInvalid       = errors.New("watchlist_encryption_key_invalid")
)

var (
	watchlistKeyOnce     sync.Once
	watchlistKeyBytes    []byte
	watchlistKeyErr      error
	watchlistKeyWarnOnce sync.Once
)

// watchlistKey returns the 32-byte AES-256 key from the env var, or an error
// if the variable is unset or malformed. Callers should treat both errors as
// "service unavailable" at the HTTP layer.
func watchlistKey() ([]byte, error) {
	watchlistKeyOnce.Do(func() {
		raw := strings.TrimSpace(os.Getenv(watchlistEncryptionKeyEnv))
		if raw == "" {
			watchlistKeyErr = errWatchlistKeyNotConfigured
			return
		}
		decoded, err := hex.DecodeString(raw)
		if err != nil {
			watchlistKeyErr = fmt.Errorf("%w: %v", errWatchlistKeyInvalid, err)
			return
		}
		if len(decoded) != 32 {
			watchlistKeyErr = fmt.Errorf("%w: expected 32 bytes, got %d", errWatchlistKeyInvalid, len(decoded))
			return
		}
		watchlistKeyBytes = decoded
	})
	if watchlistKeyErr != nil {
		return nil, watchlistKeyErr
	}
	return watchlistKeyBytes, nil
}

// watchlistKeyAvailable is the lightweight check used by scanFileList and the
// response shape — it emits a single startup warning if the key is missing but
// does not force key initialization on every call.
func watchlistKeyAvailable() bool {
	_, err := watchlistKey()
	if err != nil && errors.Is(err, errWatchlistKeyNotConfigured) {
		watchlistKeyWarnOnce.Do(func() {
			if logger != nil {
				logger.Warning("watchlist encryption key %s is not configured; watchlist features disabled", watchlistEncryptionKeyEnv)
			}
		})
	}
	return err == nil
}

// encryptWatchlistValue seals plaintext with AES-256-GCM and prepends the
// 12-byte nonce so decrypt can recover it without side metadata.
func encryptWatchlistValue(plaintext []byte) ([]byte, error) {
	key, err := watchlistKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nil, nonce, plaintext, nil)
	out := make([]byte, 0, len(nonce)+len(sealed))
	out = append(out, nonce...)
	out = append(out, sealed...)
	return out, nil
}

// decryptWatchlistValue reverses encryptWatchlistValue. A key mismatch (or any
// other tamper) surfaces as an error rather than returning garbage bytes.
func decryptWatchlistValue(ciphertext []byte) ([]byte, error) {
	key, err := watchlistKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := ciphertext[:gcm.NonceSize()]
	payload := ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, payload, nil)
}

var allowedWatchlistTypes = map[string]struct{}{
	"email":  {},
	"phone":  {},
	"path":   {},
	"ssn":    {},
	"custom": {},
}

// WatchlistEntry is the public representation of a watchlist row; the
// decrypted value is never serialized.
type WatchlistEntry struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	ValueType string    `json:"value_type"`
	CreatedAt time.Time `json:"created_at"`
}

// watchlistValue pairs a decrypted entry with metadata used at scan time to
// emit a useful finding when a match is detected.
type watchlistValue struct {
	ID        string
	Label     string
	ValueType string
	Value     []byte
}

type watchlistCreateRequest struct {
	Label     string `json:"label"`
	Value     string `json:"value"`
	ValueType string `json:"value_type"`
}

// WatchlistHandlers exposes CRUD for the encrypted watchlist.
type WatchlistHandlers struct {
	db     *database.RoutedDB
	logger *Logger
}

// NewWatchlistHandlers returns a configured handler set.
func NewWatchlistHandlers(db *database.RoutedDB, logger *Logger) *WatchlistHandlers {
	return &WatchlistHandlers{db: db, logger: logger}
}

// RegisterRoutes mounts watchlist endpoints under /security/watchlist.
func (h *WatchlistHandlers) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/watchlist", h.List).Methods(http.MethodGet)
	r.HandleFunc("/watchlist", h.Create).Methods(http.MethodPost)
	r.HandleFunc("/watchlist/{id}", h.Delete).Methods(http.MethodDelete)
}

func (h *WatchlistHandlers) requireKey(w http.ResponseWriter) bool {
	if watchlistKeyAvailable() {
		return true
	}
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error": "watchlist_encryption_key_not_configured",
	})
	return false
}

// List returns watchlist entries without decrypted values.
func (h *WatchlistHandlers) List(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	if !h.requireKey(w) {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT id, label, value_type, created_at
		FROM pii_watchlist
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list watchlist: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	entries := []WatchlistEntry{}
	for rows.Next() {
		var entry WatchlistEntry
		if err := rows.Scan(&entry.ID, &entry.Label, &entry.ValueType, &entry.CreatedAt); err != nil {
			http.Error(w, fmt.Sprintf("failed to read watchlist row: %v", err), http.StatusInternalServerError)
			return
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("failed to iterate watchlist: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	})
}

// Create inserts a new watchlist entry with the value encrypted at rest.
func (h *WatchlistHandlers) Create(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	if !h.requireKey(w) {
		return
	}
	var req watchlistCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	label := strings.TrimSpace(req.Label)
	value := req.Value
	valueType := strings.TrimSpace(req.ValueType)
	if label == "" {
		http.Error(w, "label is required", http.StatusBadRequest)
		return
	}
	if value == "" {
		http.Error(w, "value is required", http.StatusBadRequest)
		return
	}
	if _, ok := allowedWatchlistTypes[valueType]; !ok {
		http.Error(w, "invalid value_type", http.StatusBadRequest)
		return
	}
	encrypted, err := encryptWatchlistValue([]byte(value))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encrypt value: %v", err), http.StatusInternalServerError)
		return
	}
	id := uuid.New().String()
	now := time.Now()
	if _, err := h.db.ExecContext(r.Context(), `
		INSERT INTO pii_watchlist (id, label, encrypted_value, value_type, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, id, label, encrypted, valueType, now); err != nil {
		http.Error(w, fmt.Sprintf("failed to insert: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, WatchlistEntry{
		ID:        id,
		Label:     label,
		ValueType: valueType,
		CreatedAt: now,
	})
}

// Delete removes a watchlist entry by id.
func (h *WatchlistHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	if !h.requireKey(w) {
		return
	}
	id := mux.Vars(r)["id"]
	if strings.TrimSpace(id) == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM pii_watchlist WHERE id = $1`, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete: %v", err), http.StatusInternalServerError)
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// loadWatchlistValues returns all entries with their decrypted values, ready
// for scan-time matching. Returns an empty slice (and nil error) when the key
// is unset so callers can degrade gracefully.
func loadWatchlistValues(ctx context.Context, database *database.RoutedDB) ([]watchlistValue, error) {
	if database == nil {
		return nil, nil
	}
	if !watchlistKeyAvailable() {
		return nil, nil
	}
	rows, err := database.QueryContext(ctx, `
		SELECT id, label, value_type, encrypted_value
		FROM pii_watchlist
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []watchlistValue
	for rows.Next() {
		var v watchlistValue
		var encrypted []byte
		if err := rows.Scan(&v.ID, &v.Label, &v.ValueType, &encrypted); err != nil {
			return nil, err
		}
		plain, err := decryptWatchlistValue(encrypted)
		if err != nil {
			if logger != nil {
				logger.Warning("failed to decrypt watchlist entry %s: %v", v.ID, err)
			}
			continue
		}
		v.Value = plain
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
