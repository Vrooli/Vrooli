// Package residency applies the Ollaya model policy's residency block at the
// activity edge, the one boundary every local decision crosses. Each POST
// /api/decide leaves the edge carrying the keep_alive the policy resolves for
// its model, so the daemon unloads a model on the policy's schedule instead of
// the daemon-wide OLLAYA_KEEP_ALIVE. Models already loaded when the edge starts
// are brought onto the same schedule.
package residency

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/activityedge"
	"github.com/vrooli/vrooli/packages/capacity/modelretention"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
)

// decidePath is the only daemon endpoint that loads a model.
const decidePath = "/api/decide"

// maxInspectedBody mirrors the daemon's 8 MiB request ceiling; a larger body is
// forwarded untouched and the daemon refuses it itself.
const maxInspectedBody = 8 << 20

// liveCheckInterval bounds how often the edge stats the policy file, so an
// edit takes effect without a restart.
const liveCheckInterval = 5 * time.Second

// Live serves the policy file to the long-running edge. It reloads when the
// file's modification time or size changes and keeps the last valid policy
// when a reload fails, so a half-written edit never drops residency.
type Live struct {
	Path string
	Logf func(format string, args ...any)
	Now  func() time.Time

	mu      sync.Mutex
	checked time.Time
	modTime time.Time
	size    int64
	policy  policy.Policy
	loaded  bool
}

// KeepAliveSeconds applies the current policy's residency to model.
func (l *Live) KeepAliveSeconds(model string) (int, bool) {
	p, ok := l.current()
	if !ok {
		return 0, false
	}
	return p.KeepAliveSeconds(model)
}

func (l *Live) current() (policy.Policy, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if l.Now != nil {
		now = l.Now()
	}
	if l.loaded && now.Sub(l.checked) < liveCheckInterval {
		return l.policy, true
	}
	l.checked = now
	info, err := os.Stat(l.Path)
	if err != nil {
		l.logf("ollaya model policy: stat %s: %v (keeping last loaded policy)\n", l.Path, err)
		return l.policy, l.loaded
	}
	if l.loaded && info.ModTime().Equal(l.modTime) && info.Size() == l.size {
		return l.policy, true
	}
	p, err := policy.Load(l.Path)
	if err != nil {
		l.logf("ollaya model policy: reload %s: %v (keeping last loaded policy)\n", l.Path, err)
		return l.policy, l.loaded
	}
	l.policy, l.loaded, l.modTime, l.size = p, true, info.ModTime(), info.Size()
	return p, true
}

func (l *Live) logf(format string, args ...any) {
	if l.Logf != nil {
		l.Logf(format, args...)
	}
}

// Edge rewrites decision requests to carry the policy's keep_alive.
type Edge struct {
	// KeepAlive returns the keep_alive seconds a decision for model carries,
	// or false to leave the caller's value and the daemon default alone.
	KeepAlive func(model string) (int, bool)
}

// Rewrite sets keep_alive on a POST /api/decide body. A caller's explicit
// unload (keep_alive 0) is kept: it releases memory sooner than any residency.
// It is fail-open: an unreadable, oversized or non-JSON body is forwarded
// unchanged so the daemon reports the caller's error itself.
func (e *Edge) Rewrite(r *http.Request) (activityedge.Completion, error) {
	if e == nil || e.KeepAlive == nil || r == nil || r.Body == nil || r.Method != http.MethodPost ||
		strings.TrimSuffix(r.URL.Path, "/") != decidePath {
		return nil, nil
	}
	original := r.Body
	raw, err := io.ReadAll(io.LimitReader(original, maxInspectedBody+1))
	if err != nil {
		_ = original.Close()
		return nil, fmt.Errorf("read Ollaya request body: %w", err)
	}
	if len(raw) > maxInspectedBody {
		r.Body = struct {
			io.Reader
			io.Closer
		}{io.MultiReader(bytes.NewReader(raw), original), original}
		return nil, nil
	}
	_ = original.Close()
	setBody(r, e.apply(raw))
	return nil, nil
}

// apply returns body with the policy keep_alive for its model, or body
// unchanged when nothing applies. Every other value stays raw JSON, so the
// caller's question order reaches the daemon unchanged.
func (e *Edge) apply(body []byte) []byte {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil || top == nil {
		return body
	}
	var model string
	if err := json.Unmarshal(top["model"], &model); err != nil || strings.TrimSpace(model) == "" {
		return body
	}
	caller := bytes.TrimSpace(top["keep_alive"])
	if isExplicitUnload(caller) {
		return body
	}
	want, ok := e.KeepAlive(model)
	if !ok {
		return body
	}
	wantRaw := json.RawMessage(strconv.Itoa(want))
	if bytes.Equal(caller, wantRaw) {
		return body
	}
	top["keep_alive"] = wantRaw
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(top); err != nil {
		return body
	}
	return bytes.TrimSpace(out.Bytes())
}

// ReconcileLoaded brings every model already loaded when the edge starts onto
// the policy schedule. It never loads a model: only models the daemon lists as
// resident are touched, with a stateless request that decides nothing.
func (e *Edge) ReconcileLoaded(ctx context.Context, upstream activityedge.Upstream) error {
	if e == nil || e.KeepAlive == nil || upstream.Client == nil {
		return nil
	}
	loaded, err := loadedModels(ctx, upstream)
	if err != nil {
		return err
	}
	sort.Strings(loaded)
	var errs []error
	for _, model := range loaded {
		keepAlive, ok := e.KeepAlive(model)
		if !ok {
			continue
		}
		if err := setKeepAlive(ctx, upstream, model, keepAlive); err != nil {
			errs = append(errs, fmt.Errorf("reconcile %s: %w", model, err))
		}
	}
	return errors.Join(errs...)
}

func loadedModels(ctx context.Context, upstream activityedge.Upstream) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream.BaseURL+"/api/ps", nil)
	if err != nil {
		return nil, err
	}
	resp, err := upstream.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list loaded models: %s", resp.Status)
	}
	var running struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&running); err != nil {
		return nil, fmt.Errorf("decode loaded models: %w", err)
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(running.Models))
	for _, entry := range running.Models {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			name = strings.TrimSpace(entry.Model)
		}
		if name == "" || seen[modelretention.CanonicalName(name)] {
			continue
		}
		seen[modelretention.CanonicalName(name)] = true
		out = append(out, name)
	}
	return out, nil
}

func setKeepAlive(ctx context.Context, upstream activityedge.Upstream, model string, keepAlive int) error {
	body, err := json.Marshal(map[string]any{"model": model, "keep_alive": keepAlive})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstream.BaseURL+decidePath, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := upstream.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("keep_alive request returned %s", resp.Status)
	}
	return nil
}

// isExplicitUnload reports whether a caller's keep_alive is zero, written as a
// number or as a duration string such as "0" or "0s".
func isExplicitUnload(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err == nil {
		f, err := number.Float64()
		return err == nil && f == 0
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if f, err := strconv.ParseFloat(text, 64); err == nil {
		return f == 0
	}
	d, err := time.ParseDuration(text)
	return err == nil && d == 0
}

func setBody(r *http.Request, body []byte) {
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))
	r.Header.Set("Content-Length", strconv.Itoa(len(body)))
	r.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
}
