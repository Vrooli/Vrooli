package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

type fakeOllayaStore struct {
	mu        sync.Mutex
	installed []upstream.InstalledModel
	resident  []string
	deletes   []string
}

func (f *fakeOllayaStore) server(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": f.installed})
		case "/api/ps":
			models := []map[string]string{}
			for _, name := range f.resident {
				models = append(models, map[string]string{"name": name, "device": "cuda:0"})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"models": models})
		case "/api/delete":
			if r.Method != http.MethodDelete {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var body struct {
				Model string `json:"model"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.deletes = append(f.deletes, body.Model)
			kept := f.installed[:0]
			for _, m := range f.installed {
				if m.Name != body.Model {
					kept = append(kept, m)
				}
			}
			f.installed = kept
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func retentionFixture(t *testing.T) (*Handlers, *fakeOllayaStore, modelretention.Paths, policy.Policy) {
	t.Helper()
	p, err := policy.Load(shippedModelPolicyPath(t))
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeOllayaStore{installed: []upstream.InstalledModel{
		{Name: "kev:4b", Digest: p.Models["kev:4b"].Digest, Size: 9489073594, ModifiedAt: time.Date(2026, 11, 20, 17, 34, 31, 0, time.UTC)},
		{Name: "laya:latest", Digest: p.Models["laya"].Digest, Size: 10730},
		{Name: "laya:en", Digest: p.Models["laya:en"].Digest, Size: 852602879},
		{Name: "coba:1b", Digest: strings.Repeat("e", 64), Size: 4445352537},
		{Name: "mystery:1b", Digest: strings.Repeat("f", 64), Size: 100},
	}}
	server := store.server(t)
	home := t.TempDir()
	paths, err := modelretention.StorePathsAt(home, retentionResource)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	h, _, _ := handlers(t, map[string]string{"OLLAYA_URL": server.URL}, "")
	h.RetentionPaths = func() (modelretention.Paths, error) { return paths, nil }
	h.ModelRoot = func() (string, error) { return root, nil }
	h.PrepareModelPull = func(context.Context) (string, *upstream.Client, func(context.Context) error, func(), error) {
		client := &upstream.Client{HTTP: server.Client(), GetEnv: func(key string) string {
			if key == "OLLAYA_URL" {
				return server.URL
			}
			return ""
		}}
		return root, client, func(context.Context) error { return nil }, func() {}, nil
	}
	return h, store, paths, p
}

func TestOllayaRetentionKeepsDefaultPullsAndBlocksUnreproducibleModels(t *testing.T) {
	h, store, _, _ := retentionFixture(t)
	store.resident = []string{"laya:en"}
	inv, err := h.retentionInventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]modelretention.InventoryModel{}
	for _, m := range inv.Models {
		byName[m.Name] = m
	}
	if m := byName["kev:4b"]; m.Required || m.Blocked != "" || !m.ReacquireExact || strings.Join(m.Reacquire, " ") != "resource-ollaya models pull --model kev:4b" || m.ModifiedAt.IsZero() {
		t.Fatalf("bake-off candidate = %+v", m)
	}
	if m := byName["laya:latest"]; !m.Required || m.RequiredReason != "default pull" {
		t.Fatalf("default router = %+v", m)
	}
	if m := byName["laya:en"]; !m.Required || !m.Loaded {
		t.Fatalf("router target = %+v", m)
	}
	if m := byName["coba:1b"]; !strings.Contains(m.Blocked, "differs from the pinned digest") || m.ReacquireExact {
		t.Fatalf("digest drift = %+v", m)
	}
	if m := byName["mystery:1b"]; !strings.Contains(m.Blocked, "not pinned") {
		t.Fatalf("unpinned = %+v", m)
	}
}

func TestOllayaEvictRefusesLoadedRequiredAndUnreproducibleModels(t *testing.T) {
	h, store, paths, p := retentionFixture(t)
	store.resident = []string{"kev:4b"}
	ctx := context.Background()
	if got, err := h.evict(ctx, "kev:4b", p.Models["kev:4b"].Digest, 9489073594, "", ""); err != nil || got.Refused != "model is loaded in memory" {
		t.Fatalf("loaded: %+v %v", got, err)
	}
	if got, _ := h.evict(ctx, "laya", p.Models["laya"].Digest, 10730, "", ""); got.Refused != "model is required: default pull" {
		t.Fatalf("required: %+v", got)
	}
	if got, _ := h.evict(ctx, "coba:1b", strings.Repeat("e", 64), 4445352537, "", ""); !strings.Contains(got.Refused, "cannot be reacquired exactly") {
		t.Fatalf("drift: %+v", got)
	}
	if len(store.deletes) != 0 {
		t.Fatalf("deletes reached the daemon: %v", store.deletes)
	}
	if _, err := os.Stat(paths.ReceiptsFile); err == nil {
		t.Fatal("refusals must not write receipts")
	}
}

func TestOllayaEvictWritesTheReceiptThenDeletes(t *testing.T) {
	h, store, paths, p := retentionFixture(t)
	got, err := h.evict(context.Background(), "kev:4b", p.Models["kev:4b"].Digest, 9489073594, "over budget", "storage-manager")
	if err != nil || !got.Evicted || !got.ReacquireExact {
		t.Fatalf("evict: %+v %v", got, err)
	}
	if strings.Join(store.deletes, ",") != "kev:4b" {
		t.Fatalf("deletes = %v", store.deletes)
	}
	receipt, ok, err := modelretention.LatestReceipt(paths.ReceiptsFile, "kev:4b")
	if err != nil || !ok || receipt.Status != modelretention.ReceiptEvicted || strings.Join(receipt.Reacquire, " ") != "resource-ollaya models pull --model kev:4b" || receipt.RequestedBy != "storage-manager" {
		t.Fatalf("receipt = %+v ok=%t err=%v", receipt, ok, err)
	}
}
