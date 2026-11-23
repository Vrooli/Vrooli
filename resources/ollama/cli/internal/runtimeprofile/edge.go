package runtimeprofile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/activityedge"
	"github.com/vrooli/vrooli/packages/capacity/modelretention"
)

// maxNormalizedBody bounds how much of a request the edge buffers to inspect.
// Larger bodies (for example many inline images) pass through untouched.
const maxNormalizedBody = 64 << 20

// edgeLogInterval rate-limits the per-caller normalization log line.
const edgeLogInterval = 10 * time.Minute

// normalizedPaths are the native API endpoints whose body carries load
// options and a keep_alive Ollama honors.
var normalizedPaths = map[string]struct{}{
	"/api/generate":   {},
	"/api/chat":       {},
	"/api/embed":      {},
	"/api/embeddings": {},
}

// compatPrefix is Ollama's OpenAI- and Anthropic-compatible API. Those
// endpoints load models but ignore a body keep_alive: on Ollama 0.30.10 a
// /v1/chat/completions request carrying keep_alive 120 left the model at the
// daemon default (never unload). The edge forwards such a request unchanged
// and afterwards reasserts the model's residency with one native load-only
// request, so Ollama's own expiry timer applies the policy.
const compatPrefix = "/v1/"

// compatLoadOnly names the native endpoint that loads a model without running
// it for each compatible endpoint whose models may not support generate. An
// empty /api/embed loads an embedding model and applies its keep_alive;
// /api/generate would refuse it ("does not support generate").
var compatLoadOnly = map[string]string{
	"/v1/embeddings": "/api/embed",
}

// defaultLoadOnly is the native load-only endpoint for every other compatible
// endpoint (chat, completions, responses, messages).
const defaultLoadOnly = "/api/generate"

// defaultReassertDelay coalesces a burst of compatible requests for one model
// into one reassertion after the burst ends.
const defaultReassertDelay = 2 * time.Second

// EdgeNormalizer replaces caller-supplied load options on public Ollama
// requests with the profile's RequestOptions, so every external client
// (OpenCode, scripts, other agents) reuses the same runner as the gateway.
// When KeepAlive is set it also applies the model policy's residency to every
// request that can load a model: native requests carry it in keep_alive (an
// explicit caller unload, keep_alive 0, is always honored); compatible /v1
// requests get it reasserted after they finish.
type EdgeNormalizer struct {
	Profile Profile
	// KeepAlive returns the keep_alive seconds a request for model carries, or
	// false to leave the caller's value and the daemon default alone.
	KeepAlive func(model string) (int, bool)
	// Logf receives one line per distinct caller shape per interval.
	Logf func(format string, args ...any)
	Now  func() time.Time
	// ReassertDelay overrides defaultReassertDelay (tests).
	ReassertDelay time.Duration

	mu   sync.Mutex
	seen map[string]*edgeSeen

	reassertMu sync.Mutex
	reasserts  map[string]*time.Timer
}

type edgeSeen struct {
	lastLogged time.Time
	pending    int
}

// NewEdgeNormalizer returns a normalizer for the embedded profile.
func NewEdgeNormalizer(logf func(format string, args ...any)) *EdgeNormalizer {
	return &EdgeNormalizer{Profile: Current(), Logf: logf, Now: time.Now}
}

// Rewrite normalizes the load options of a matching request body in place and
// returns the residency follow-up a compatible request needs. It is
// fail-open: an unreadable or non-JSON body is forwarded unchanged so Ollama
// reports the caller's error itself.
func (n *EdgeNormalizer) Rewrite(r *http.Request) (activityedge.Completion, error) {
	if r == nil || r.Body == nil || r.Method != http.MethodPost {
		return nil, nil
	}
	path := strings.ToLower(strings.TrimSuffix(r.URL.Path, "/"))
	_, native := normalizedPaths[path]
	compat := strings.HasPrefix(path, compatPrefix)
	if !native && !compat {
		return nil, nil
	}
	original := r.Body
	raw, err := io.ReadAll(io.LimitReader(original, maxNormalizedBody+1))
	if err != nil {
		_ = original.Close()
		return nil, fmt.Errorf("read Ollama request body: %w", err)
	}
	if len(raw) > maxNormalizedBody {
		// Too large to inspect: forward the bytes already read followed by the
		// unread remainder, unchanged.
		r.Body = struct {
			io.Reader
			io.Closer
		}{io.MultiReader(bytes.NewReader(raw), original), original}
		return nil, nil
	}
	_ = original.Close()
	if compat {
		setBody(r, raw)
		var req struct {
			Model string `json:"model"`
		}
		if json.Unmarshal(raw, &req) != nil {
			return nil, nil
		}
		return n.residencyFollowUp(path, req.Model), nil
	}
	body, model, drift := n.Profile.normalizeBody(raw, n.KeepAlive)
	setBody(r, body)
	if len(drift) > 0 {
		n.record(path, model, drift)
	}
	return nil, nil
}

// residencyFollowUp returns the completion that reasserts model's policy
// keep_alive after a compatible request to path, or nil when the policy
// leaves the model at the daemon default.
func (n *EdgeNormalizer) residencyFollowUp(path, model string) activityedge.Completion {
	model = strings.TrimSpace(model)
	if n.KeepAlive == nil || model == "" {
		return nil
	}
	if _, ok := n.KeepAlive(model); !ok {
		return nil
	}
	loadOnly, ok := compatLoadOnly[path]
	if !ok {
		loadOnly = defaultLoadOnly
	}
	return func(upstream activityedge.Upstream) { n.scheduleReassert(model, loadOnly, upstream) }
}

// scheduleReassert (re)starts model's trailing timer, so the reassertion runs
// once, after the last compatible request of a burst has finished.
func (n *EdgeNormalizer) scheduleReassert(model, loadOnly string, upstream activityedge.Upstream) {
	delay := n.ReassertDelay
	if delay <= 0 {
		delay = defaultReassertDelay
	}
	key := modelretention.CanonicalName(model)
	n.reassertMu.Lock()
	defer n.reassertMu.Unlock()
	if n.reasserts == nil {
		n.reasserts = map[string]*time.Timer{}
	}
	if timer := n.reasserts[key]; timer != nil {
		timer.Stop()
	}
	n.reasserts[key] = time.AfterFunc(delay, func() {
		n.reassertMu.Lock()
		delete(n.reasserts, key)
		n.reassertMu.Unlock()
		if err := n.reassertResidency(model, loadOnly, upstream); err != nil && n.Logf != nil {
			n.Logf("ollama residency: reassert keep_alive for %s failed: %v\n", model, err)
		}
	})
}

// reassertResidency sets a loaded model's keep_alive to the policy value with
// a native load-only request to loadOnly. A model that is no longer loaded is
// left alone: reasserting must never load a model nobody is using.
func (n *EdgeNormalizer) reassertResidency(model, loadOnly string, upstream activityedge.Upstream) error {
	keepAlive, ok := n.KeepAlive(model)
	if !ok || upstream.Client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return reassertLoadedResidency(ctx, model, loadOnly, keepAlive, upstream)
}

// ReconcileLoadedResidency applies the current declaration to models that are
// already loaded when the activity edge starts. It never loads an absent
// model, and policy durations are set from now so old daemon-level keep-alive
// values cannot pin an idle runner indefinitely.
func (n *EdgeNormalizer) ReconcileLoadedResidency(ctx context.Context, upstream activityedge.Upstream) error {
	if n == nil || n.KeepAlive == nil || upstream.Client == nil {
		return nil
	}
	loaded, err := loadedModels(ctx, upstream)
	if err != nil {
		return err
	}
	models := make([]string, 0, len(loaded))
	for model := range loaded {
		models = append(models, model)
	}
	sort.Strings(models)
	var reconcileErrors []error
	for _, model := range models {
		keepAlive, ok := n.KeepAlive(model)
		if !ok {
			continue
		}
		loadOnly, err := loadOnlyEndpoint(ctx, model, upstream)
		if err != nil {
			reconcileErrors = append(reconcileErrors, fmt.Errorf("inspect %s capabilities: %w", model, err))
			continue
		}
		if err := reassertLoadedResidency(ctx, model, loadOnly, keepAlive, upstream); err != nil {
			reconcileErrors = append(reconcileErrors, fmt.Errorf("reconcile %s: %w", model, err))
		}
	}
	return errors.Join(reconcileErrors...)
}

func loadOnlyEndpoint(ctx context.Context, model string, upstream activityedge.Upstream) (string, error) {
	body, err := json.Marshal(map[string]string{"model": model})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstream.BaseURL+"/api/show", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := upstream.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("show returned %s", resp.Status)
	}
	var show struct {
		Capabilities []string `json:"capabilities"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&show); err != nil {
		return "", err
	}
	for _, capability := range show.Capabilities {
		if strings.EqualFold(capability, "embedding") {
			return "/api/embed", nil
		}
	}
	return defaultLoadOnly, nil
}

func reassertLoadedResidency(ctx context.Context, model, loadOnly string, keepAlive int, upstream activityedge.Upstream) error {
	loaded, err := loadedModels(ctx, upstream)
	if err != nil {
		return err
	}
	if !loaded[modelretention.CanonicalName(model)] {
		return nil
	}
	request := map[string]any{"model": model, "keep_alive": keepAlive}
	if loadOnly == "/api/embed" {
		request["input"] = []string{}
	}
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstream.BaseURL+loadOnly, bytes.NewReader(body))
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
		return fmt.Errorf("load-only request returned %s", resp.Status)
	}
	return nil
}

func loadedModels(ctx context.Context, upstream activityedge.Upstream) (map[string]bool, error) {
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
		return nil, fmt.Errorf("list loaded models returned %s", resp.Status)
	}
	var ps struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&ps); err != nil {
		return nil, fmt.Errorf("decode loaded models: %w", err)
	}
	loaded := map[string]bool{}
	for _, m := range ps.Models {
		loaded[modelretention.CanonicalName(m.Name)] = true
		loaded[modelretention.CanonicalName(m.Model)] = true
	}
	return loaded, nil
}

// normalizeBody makes a request's load options exactly RequestOptions for its
// model and, when keepAlive is set, its keep_alive the policy residency. It
// returns the body, the model, and the caller-supplied load options that
// differed from the profile (the drift the edge prevented). A body that
// already matches is returned as the original bytes.
func (p Profile) normalizeBody(raw []byte, keepAlive func(string) (int, bool)) ([]byte, string, []string) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return raw, "", nil
	}
	var model string
	_ = json.Unmarshal(top["model"], &model)
	options := map[string]any{}
	if rawOptions, ok := top["options"]; ok && len(bytes.TrimSpace(rawOptions)) > 0 && !bytes.Equal(bytes.TrimSpace(rawOptions), []byte("null")) {
		decoder := json.NewDecoder(bytes.NewReader(rawOptions))
		decoder.UseNumber()
		if err := decoder.Decode(&options); err != nil {
			return raw, model, nil
		}
	}
	callerValues := map[string]any{}
	for key, value := range options {
		if p.IsLoadOption(key) {
			callerValues[key] = value
		}
	}
	applied := p.RequestOptions(model)
	var drift []string
	changed := false
	for key, value := range callerValues {
		want, ok := applied[key]
		if !ok || fmt.Sprint(value) != fmt.Sprint(want) {
			drift = append(drift, key)
			changed = true
		}
	}
	for key := range applied {
		if _, ok := callerValues[key]; !ok {
			changed = true
		}
	}
	keepAliveValue, keepAliveChanged := residencyKeepAlive(top["keep_alive"], model, keepAlive)
	if !changed && !keepAliveChanged {
		return raw, model, nil
	}
	if changed {
		p.StripLoadOptions(options)
		for key, value := range applied {
			options[key] = value
		}
		if len(options) == 0 {
			delete(top, "options")
		} else {
			encoded, err := json.Marshal(options)
			if err != nil {
				return raw, model, nil
			}
			top["options"] = encoded
		}
	}
	if keepAliveChanged {
		top["keep_alive"] = keepAliveValue
	}
	body, err := json.Marshal(top)
	if err != nil {
		return raw, model, nil
	}
	sort.Strings(drift)
	return body, model, drift
}

// residencyKeepAlive returns the keep_alive the policy wants for model and
// whether the body must change. A caller's explicit unload (0) is kept: it
// releases memory sooner than any residency would.
func residencyKeepAlive(caller json.RawMessage, model string, keepAlive func(string) (int, bool)) (json.RawMessage, bool) {
	if keepAlive == nil || strings.TrimSpace(model) == "" {
		return nil, false
	}
	want, ok := keepAlive(model)
	if !ok {
		return nil, false
	}
	caller = bytes.TrimSpace(caller)
	if isExplicitUnload(caller) {
		return nil, false
	}
	wantRaw := json.RawMessage(strconv.Itoa(want))
	if bytes.Equal(caller, wantRaw) {
		return nil, false
	}
	return wantRaw, true
}

// isExplicitUnload reports whether a caller's keep_alive is zero, written as
// a number or as an Ollama duration string such as "0" or "0s".
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

func (n *EdgeNormalizer) record(path, model string, removed []string) {
	if n.Logf == nil {
		return
	}
	now := time.Now()
	if n.Now != nil {
		now = n.Now()
	}
	keys := append([]string(nil), removed...)
	sort.Strings(keys)
	key := path + "|" + model + "|" + strings.Join(keys, ",")
	n.mu.Lock()
	if n.seen == nil {
		n.seen = map[string]*edgeSeen{}
	}
	entry := n.seen[key]
	if entry == nil {
		entry = &edgeSeen{}
		n.seen[key] = entry
	}
	entry.pending++
	shouldLog := entry.lastLogged.IsZero() || now.Sub(entry.lastLogged) >= edgeLogInterval
	count := entry.pending
	if shouldLog {
		entry.lastLogged = now
		entry.pending = 0
	}
	n.mu.Unlock()
	if shouldLog {
		n.Logf("ollama runtime profile: replaced caller load options at=%s path=%s model=%s keys=%s requests=%d\n", now.Format(time.RFC3339), path, model, strings.Join(keys, ","), count)
	}
}
