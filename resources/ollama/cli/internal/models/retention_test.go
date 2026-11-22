package models

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

const (
	fallbackDigest = "46e0c10c039e019119339687c3c1757cc81b9da49709a3b3924863ba87ca666e"
	fallbackSize   = 4920753328
)

// fakeStore is an Ollama daemon with tags, ps and DELETE /api/delete.
type fakeStore struct {
	mu      sync.Mutex
	tags    []storeTag
	loaded  []string
	deletes []string
	methods []string
}

func (f *fakeStore) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": f.tags})
		case "/api/ps":
			models := []map[string]string{}
			for _, name := range f.loaded {
				models = append(models, map[string]string{"name": name, "model": name})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"models": models})
		case "/api/delete":
			f.methods = append(f.methods, r.Method)
			if r.Method != http.MethodDelete {
				http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var body struct {
				Model string `json:"model"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.deletes = append(f.deletes, body.Model)
			kept := f.tags[:0]
			for _, tag := range f.tags {
				if tag.Name != body.Model {
					kept = append(kept, tag)
				}
			}
			f.tags = kept
		default:
			http.NotFound(w, r)
		}
	})
}

func retentionPolicy() policy.Policy {
	return policy.Policy{Roles: map[string]policy.Role{
		"code.local": {Model: "gemma4:12b", Fallbacks: []string{"llama3.1:8b"}},
	}}
}

type retentionHarness struct {
	store  *fakeStore
	deps   RetentionDeps
	paths  modelretention.Paths
	server *httptest.Server
}

func newRetentionHarness(t *testing.T) *retentionHarness {
	t.Helper()
	store := &fakeStore{tags: []storeTag{
		{Name: "llama3.1:8b", Size: fallbackSize, Digest: fallbackDigest, ModifiedAt: time.Date(2025, 8, 18, 0, 0, 0, 0, time.UTC)},
		{Name: "gemma4:12b", Size: 7556508396, Digest: strings.Repeat("a", 64)},
	}}
	server := httptest.NewServer(store.handler())
	t.Cleanup(server.Close)
	home := t.TempDir()
	paths, err := modelretention.StorePathsAt(home, "ollama")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(home, "models")
	manifest := filepath.Join(root, "manifests", "registry.ollama.ai", "library", "llama3.1", "8b")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"config":{"digest":"sha256:c","size":500},"layers":[{"digest":"sha256:w","size":4920752828}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	h := &retentionHarness{store: store, paths: paths, server: server}
	h.deps = RetentionDeps{
		PublicStore: func() (*StoreClient, error) { return &StoreClient{BaseURL: server.URL, HTTP: server.Client()}, nil },
		ModelRoot:   func() (string, error) { return root, nil },
		PrepareMutation: func(context.Context) (ensure.ManagedModelMutation, error) {
			return ensure.ManagedModelMutation{
				Client:    &ensure.Client{BaseURL: server.URL, HTTP: server.Client()},
				ModelRoot: root,
				Verify:    func(context.Context) error { return nil },
				Release:   func() {},
			}, nil
		},
		Paths:      func() (modelretention.Paths, error) { return paths, nil },
		LoadPolicy: func() (policy.Policy, error) { return retentionPolicy(), nil },
		Now:        func() time.Time { return time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC) },
	}
	return h
}

func (h *retentionHarness) evict(t *testing.T, name string) modelretention.EvictResult {
	t.Helper()
	result, err := evictModel(context.Background(), h.deps, evictRequest{Name: name, Digest: fallbackDigest, Size: fallbackSize, RequestedBy: "test"})
	if err != nil {
		t.Fatalf("evict %s: %v", name, err)
	}
	return result
}

func (h *retentionHarness) assertNothingDeleted(t *testing.T) {
	t.Helper()
	if len(h.store.deletes) != 0 || len(h.store.methods) != 0 {
		t.Fatalf("delete reached the daemon: %v", h.store.deletes)
	}
	if _, err := os.Stat(h.paths.ReceiptsFile); err == nil {
		t.Fatal("a refused eviction must not write a receipt")
	}
}

func TestEvictRefusesALoadedModel(t *testing.T) {
	h := newRetentionHarness(t)
	h.store.loaded = []string{"llama3.1:8b"}
	if got := h.evict(t, "llama3.1:8b"); got.Refused != "model is loaded in memory" || got.Evicted {
		t.Fatalf("result = %+v", got)
	}
	h.assertNothingDeleted(t)
}

// Residency is checked first, so a live proof can target a loaded model with
// a wrong digest: whichever check runs, nothing can be deleted.
func TestEvictRefusesALoadedModelBeforeLookingAtItsIdentity(t *testing.T) {
	h := newRetentionHarness(t)
	h.store.loaded = []string{"gemma4:12b"}
	h.deps.PrepareMutation = func(context.Context) (ensure.ManagedModelMutation, error) {
		t.Fatal("a refusal on the public pass must not take the lifecycle lock")
		return ensure.ManagedModelMutation{}, nil
	}
	result, err := evictModel(context.Background(), h.deps, evictRequest{Name: "gemma4:12b", Digest: "wrong", Size: 1})
	if err != nil || result.Refused != "model is loaded in memory" {
		t.Fatalf("result = %+v err=%v", result, err)
	}
	h.assertNothingDeleted(t)
}

func TestEvictRefusesWhileARequestIsInFlight(t *testing.T) {
	h := newRetentionHarness(t)
	release, err := modelretention.Fence{Dir: h.paths.FenceDir}.EnterRequest("llama3.1:8b")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if got := h.evict(t, "llama3.1:8b"); !strings.Contains(got.Refused, "in flight") {
		t.Fatalf("result = %+v", got)
	}
	h.assertNothingDeleted(t)
}

func TestEvictRefusesAChangedDigestOrARequiredModel(t *testing.T) {
	h := newRetentionHarness(t)
	h.store.tags[0].Digest = strings.Repeat("b", 64)
	if got := h.evict(t, "llama3.1:8b"); !strings.Contains(got.Refused, "identity changed") {
		t.Fatalf("changed digest: %+v", got)
	}
	if got := h.evict(t, "gemma4:12b"); got.Refused != "model is required: primary model of role code.local" {
		t.Fatalf("required model: %+v", got)
	}
	h.assertNothingDeleted(t)
}

func TestEvictRefusesWhenPolicyMakesTheModelRequiredBeforeDeletion(t *testing.T) {
	h := newRetentionHarness(t)
	// The role is reassigned after the decision, while eviction waits for the
	// lifecycle lock: the policy read under the fence must see it.
	edited := false
	prepare := h.deps.PrepareMutation
	h.deps.PrepareMutation = func(ctx context.Context) (ensure.ManagedModelMutation, error) {
		edited = true
		return prepare(ctx)
	}
	h.deps.LoadPolicy = func() (policy.Policy, error) {
		p := retentionPolicy()
		if edited {
			p.Roles["chat.default"] = policy.Role{Model: "llama3.1:8b"}
		}
		return p, nil
	}
	if got := h.evict(t, "llama3.1:8b"); !strings.Contains(got.Refused, "model is required") {
		t.Fatalf("result = %+v", got)
	}
	h.assertNothingDeleted(t)
}

func TestEvictRecordsTheReacquireReceiptThenDeletes(t *testing.T) {
	h := newRetentionHarness(t)
	got := h.evict(t, "llama3.1:8b")
	if !got.Evicted || got.Refused != "" || got.FreedBytes != fallbackSize {
		t.Fatalf("result = %+v", got)
	}
	if strings.Join(h.store.methods, ",") != http.MethodDelete || strings.Join(h.store.deletes, ",") != "llama3.1:8b" {
		t.Fatalf("delete calls: methods %v models %v", h.store.methods, h.store.deletes)
	}
	data, err := os.ReadFile(h.paths.ReceiptsFile)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("receipts = %q, want evicting then evicted", data)
	}
	var first, last modelretention.Receipt
	_ = json.Unmarshal([]byte(lines[0]), &first)
	_ = json.Unmarshal([]byte(lines[1]), &last)
	if first.Status != modelretention.ReceiptEvicting || last.Status != modelretention.ReceiptEvicted {
		t.Fatalf("receipt order = %s, %s", first.Status, last.Status)
	}
	if strings.Join(first.Reacquire, " ") != "resource-ollama models reacquire llama3.1:8b" || first.Bytes != fallbackSize || first.Digest != fallbackDigest {
		t.Fatalf("receipt = %+v", first)
	}
}

func TestRetentionInventoryMarksPrimariesRequiredAndCountsUniqueBytes(t *testing.T) {
	h := newRetentionHarness(t)
	h.store.loaded = []string{"gemma4:12b"}
	inv, err := buildRetentionInventory(context.Background(), h.deps)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]modelretention.InventoryModel{}
	for _, m := range inv.Models {
		byName[m.Name] = m
	}
	fallback, primary := byName["llama3.1:8b"], byName["gemma4:12b"]
	if fallback.Required || fallback.UniqueBytes != fallbackSize || fallback.ModifiedAt.IsZero() || fallback.ReacquireExact {
		t.Fatalf("fallback = %+v", fallback)
	}
	if !primary.Required || !primary.Loaded {
		t.Fatalf("primary = %+v", primary)
	}
	if inv.Schema != modelretention.InventorySchema || inv.StoreBytes != fallbackSize {
		t.Fatalf("inventory = %+v", inv)
	}
}

func TestReacquireRunsEnsureWithADocumentedDirectModelException(t *testing.T) {
	var got []string
	h := &Handlers{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Retention: RetentionDeps{
		Ensure: func(args []string) error { got = args; return nil },
		Paths:  func() (modelretention.Paths, error) { return modelretention.StorePathsAt(t.TempDir(), "ollama") },
		Now:    func() time.Time { return time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC) },
	}}
	if err := h.Reacquire([]string{"llava:7b"}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "--config-base64" {
		t.Fatalf("ensure args = %v", got)
	}
	raw, _ := base64.StdEncoding.DecodeString(got[1])
	cfg, err := ensure.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Models) != 1 || cfg.Models[0].Ref() != "llava:7b" || cfg.Models[0].Owner != "storage-manager" || cfg.Models[0].ReviewAfter != "2027-02-21" {
		t.Fatalf("ensure config = %+v", cfg)
	}
}
