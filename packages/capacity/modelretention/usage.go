package modelretention

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	platform "github.com/vrooli/platform-go"
)

// UsageSchemaVersion is the model-usage.json format version.
const UsageSchemaVersion = 1

// UsageEntry is what the request boundary saw for one model.
type UsageEntry struct {
	LastRequestAt time.Time `json:"last_request_at"`
	Requests      int64     `json:"requests"`
	// Callers splits the requests by the process that sent them, keyed by the
	// edge's caller identity (owning scope and executable). Requests whose
	// sender exited before the edge identified it count under UnknownCaller.
	Callers map[string]CallerUsage `json:"callers,omitempty"`
}

// CallerUsage is one caller's share of a model's requests.
type CallerUsage struct {
	LastRequestAt time.Time `json:"last_request_at"`
	Requests      int64     `json:"requests"`
}

// UnknownCaller is the identity of a request whose sender was not identified.
const UnknownCaller = "unknown"

// maxCallersPerModel bounds a model's caller table. Callers are scopes and
// executables, so a real host stays far below it; past it the least recently
// seen caller is dropped.
const maxCallersPerModel = 64

// UsageDocument is model-usage.json. TrackingSince is when recording began:
// a model with no entry has not been requested since then, and nothing is
// known about its use before then.
type UsageDocument struct {
	SchemaVersion int                   `json:"schema_version"`
	Resource      string                `json:"resource"`
	TrackingSince time.Time             `json:"tracking_since"`
	UpdatedAt     time.Time             `json:"updated_at"`
	Models        map[string]UsageEntry `json:"models"`
}

// LastActivity is the latest moment the model is known or assumed to have
// been in use: its last request, or the start of tracking when it has none.
// A zero result means usage is unknown (tracking never ran).
func (d UsageDocument) LastActivity(model string) time.Time {
	entry, ok := d.Models[CanonicalName(model)]
	if ok && entry.LastRequestAt.After(d.TrackingSince) {
		return entry.LastRequestAt
	}
	return d.TrackingSince
}

// ReadUsage reads a usage document. A missing file is not an error: it
// returns a document with zero TrackingSince, meaning usage is unknown.
func ReadUsage(path string) (UsageDocument, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return UsageDocument{SchemaVersion: UsageSchemaVersion, Models: map[string]UsageEntry{}}, nil
	}
	if err != nil {
		return UsageDocument{}, fmt.Errorf("read model usage: %w", err)
	}
	var doc UsageDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return UsageDocument{}, fmt.Errorf("decode model usage %s: %w", path, err)
	}
	if doc.SchemaVersion != UsageSchemaVersion {
		return UsageDocument{}, fmt.Errorf("model usage %s has unsupported schema version %d", path, doc.SchemaVersion)
	}
	if doc.Models == nil {
		doc.Models = map[string]UsageEntry{}
	}
	return doc, nil
}

// Recorder accumulates request timestamps in memory and merges them into the
// usage file. Merging takes the later timestamp per model, so concurrent
// recorders and restarts never move a last-request time backwards.
type Recorder struct {
	resource string
	path     string
	now      func() time.Time

	mu      sync.Mutex
	pending map[string]UsageEntry
}

// NewRecorder returns a recorder for one resource's usage file.
func NewRecorder(resource, path string, now func() time.Time) *Recorder {
	if now == nil {
		now = time.Now
	}
	return &Recorder{resource: resource, path: path, now: now, pending: map[string]UsageEntry{}}
}

// Now is the recorder's clock, so attribution recorded later can carry the
// time its request arrived.
func (r *Recorder) Now() time.Time { return r.now().UTC() }

// Record notes one request for model at the current time.
func (r *Recorder) Record(model string) { r.observe(model, 1) }

// RecordCaller attributes one request for model, observed at at, to caller.
// It is separate from Record because the edge identifies a connection's
// sender off the request path, after the request was counted.
func (r *Recorder) RecordCaller(model, caller string, at time.Time) {
	key := CanonicalName(model)
	if r == nil || key == "" {
		return
	}
	caller = strings.TrimSpace(caller)
	if caller == "" {
		caller = UnknownCaller
	}
	r.mu.Lock()
	entry := r.pending[key]
	entry.addCaller(caller, CallerUsage{LastRequestAt: at.UTC(), Requests: 1})
	r.pending[key] = entry
	r.mu.Unlock()
}

func (e *UsageEntry) addCaller(caller string, usage CallerUsage) {
	if e.Callers == nil {
		e.Callers = map[string]CallerUsage{}
	}
	current := e.Callers[caller]
	if usage.LastRequestAt.After(current.LastRequestAt) {
		current.LastRequestAt = usage.LastRequestAt
	}
	current.Requests += usage.Requests
	e.Callers[caller] = current
	for len(e.Callers) > maxCallersPerModel {
		oldest := ""
		for name, candidate := range e.Callers {
			if oldest == "" || candidate.LastRequestAt.Before(e.Callers[oldest].LastRequestAt) {
				oldest = name
			}
		}
		delete(e.Callers, oldest)
	}
}

// merge folds another observation of the same model into e.
func (e *UsageEntry) merge(other UsageEntry) {
	if other.LastRequestAt.After(e.LastRequestAt) {
		e.LastRequestAt = other.LastRequestAt
	}
	e.Requests += other.Requests
	for caller, usage := range other.Callers {
		e.addCaller(caller, usage)
	}
}

// Touch moves the model's last-request time to now without counting a new
// request (a long request is use until it ends).
func (r *Recorder) Touch(model string) { r.observe(model, 0) }

func (r *Recorder) observe(model string, requests int64) {
	key := CanonicalName(model)
	if r == nil || key == "" {
		return
	}
	at := r.now().UTC()
	r.mu.Lock()
	entry := r.pending[key]
	if at.After(entry.LastRequestAt) {
		entry.LastRequestAt = at
	}
	entry.Requests += requests
	r.pending[key] = entry
	r.mu.Unlock()
}

// Flush merges pending observations into the usage file. It also creates
// the file (starting TrackingSince) when none exists, so "no requests since
// tracking began" is distinguishable from "never tracked".
func (r *Recorder) Flush(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	pending := r.pending
	r.pending = map[string]UsageEntry{}
	r.mu.Unlock()
	err := r.merge(ctx, pending)
	if err != nil {
		// Keep the observations for the next flush.
		r.mu.Lock()
		for key, entry := range pending {
			current := r.pending[key]
			current.merge(entry)
			r.pending[key] = current
		}
		r.mu.Unlock()
	}
	return err
}

// Run flushes every interval until ctx ends, then flushes once more.
func (r *Recorder) Run(ctx context.Context, interval time.Duration, logf func(string, ...any)) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	flush := func(c context.Context) {
		if err := r.Flush(c); err != nil && logf != nil {
			logf("model usage flush failed: %v", err)
		}
	}
	flush(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			final, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			flush(final)
			cancel()
			return
		case <-ticker.C:
			flush(ctx)
		}
	}
}

func (r *Recorder) merge(ctx context.Context, pending map[string]UsageEntry) error {
	if strings.TrimSpace(r.path) == "" {
		return errors.New("model usage path is not configured")
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return fmt.Errorf("create model usage directory: %w", err)
	}
	release, err := platform.AcquireFileLockContext(ctx, r.path+".lock")
	if err != nil {
		return fmt.Errorf("lock model usage: %w", err)
	}
	defer release()
	doc, err := ReadUsage(r.path)
	if err != nil {
		return err
	}
	now := r.now().UTC()
	created := doc.TrackingSince.IsZero()
	if created {
		doc.TrackingSince = now
	}
	if len(pending) == 0 && !created {
		return nil
	}
	doc.SchemaVersion = UsageSchemaVersion
	doc.Resource = r.resource
	doc.UpdatedAt = now
	for key, entry := range pending {
		current := doc.Models[key]
		current.merge(entry)
		doc.Models[key] = current
	}
	return writeJSONAtomic(r.path, doc)
}

func writeJSONAtomic(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// RequestModel reads the model a request body names and whether the request
// is use. An unload (keep_alive 0 with nothing to process) names a model but
// is not use. An empty model means the body did not name one.
func RequestModel(body []byte) (model string, use bool) {
	var req struct {
		Model     string          `json:"model"`
		Name      string          `json:"name"`
		KeepAlive json.RawMessage `json:"keep_alive"`
		Prompt    json.RawMessage `json:"prompt"`
		Messages  json.RawMessage `json:"messages"`
		Input     json.RawMessage `json:"input"`
		State     json.RawMessage `json:"state"`
		Questions json.RawMessage `json:"questions"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", false
	}
	model = strings.TrimSpace(req.Model)
	if model == "" {
		model = strings.TrimSpace(req.Name)
	}
	if model == "" {
		return "", false
	}
	if zeroKeepAlive(req.KeepAlive) && empty(req.Prompt) && empty(req.Messages) && empty(req.Input) && empty(req.State) && empty(req.Questions) {
		return model, false
	}
	return model, true
}

func zeroKeepAlive(raw json.RawMessage) bool {
	value := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	switch value {
	case "0", "0s", "0m", "0h":
		return true
	default:
		return false
	}
}

func empty(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	switch string(trimmed) {
	case "", "null", `""`, "[]", "{}":
		return true
	default:
		return false
	}
}
