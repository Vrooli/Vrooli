package providers

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
	"storage-manager/internal/cleanup"
)


var decisionNow = time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)

func testModelRetentionPolicy() ModelRetentionPolicy {
	return ModelRetentionPolicy{
		CheckInterval: 15 * time.Minute, NotifyBelowFreePercent: 15, EvictBelowFreePercent: 10, EvictUntilFreePercent: 15,
		IdleWindow: 14 * 24 * time.Hour, RegistryTagRefetchPenalty: 2, Approval: ModelApprovalOperator,
		Stores: []ModelStoreConfig{{Resource: "ollama", Command: "resource-ollama"}},
	}
}

func idleModel(name string, unique int64, exact bool) modelretention.InventoryModel {
	return modelretention.InventoryModel{Name: name, Digest: strings.Repeat("d", 64), Bytes: unique, UniqueBytes: unique, ModifiedAt: decisionNow.AddDate(0, -3, 0), ReacquireExact: exact, Reacquire: []string{"reacquire", name}}
}

// usageTracked starts tracking 30 days ago with no requests since.
func usageTracked(models map[string]time.Time) modelretention.UsageDocument {
	doc := modelretention.UsageDocument{SchemaVersion: 1, TrackingSince: decisionNow.AddDate(0, 0, -30), Models: map[string]modelretention.UsageEntry{}}
	for name, at := range models {
		doc.Models[modelretention.CanonicalName(name)] = modelretention.UsageEntry{LastRequestAt: at, Requests: 1}
	}
	return doc
}

func space(freePercent float64) FilesystemSpace {
	total := 1000 * gib
	return FilesystemSpace{ID: "dev:1", Path: "/", TotalBytes: total, FreeBytes: int64(freePercent / 100 * float64(total))}
}

func ollamaStore(free float64, models ...modelretention.InventoryModel) ModelStoreObservation {
	var bytes int64
	for _, m := range models {
		bytes += m.Bytes
	}
	return ModelStoreObservation{
		Resource: "ollama", Space: space(free), Usage: usageTracked(nil),
		Inventory: modelretention.Inventory{Resource: "ollama", ModelRoot: "/models", StoreBytes: bytes, Models: models},
	}
}

func selectedNames(d ModelEvictionDecision) []string {
	var out []string
	for _, c := range d.Selected {
		out = append(out, c.Name+"/"+c.SelectedFor)
	}
	return out
}

// The bands are the "decide how often to offload based on free space" rule.
func TestModelEvictionBandsFollowFreeSpace(t *testing.T) {
	models := []modelretention.InventoryModel{idleModel("llama3.1:8b", 40*gib, false), idleModel("llava:7b", 30*gib, false), idleModel("qwen3:4b", 10*gib, false)}
	cases := []struct {
		name     string
		free     float64
		band     ModelBand
		selected []string
	}{
		{"healthy disk does nothing", 20, ModelBandOK, nil},
		{"exactly at the notify line is healthy", 15, ModelBandOK, nil},
		{"below 15% previews and notifies only", 12, ModelBandNotify, nil},
		{"exactly at the evict line only notifies", 10, ModelBandNotify, nil},
		// 9% free of 1000 GiB needs 60 GiB to reach 15%: the two largest.
		{"below 10% evicts largest first until 15%", 9, ModelBandEvict, []string{"llama3.1:8b/free-space", "llava:7b/free-space"}},
		// 14.5% cannot happen in the evict band; 5% needs 100 GiB, more than exists.
		{"deep pressure selects everything eligible and reports the shortfall", 5, ModelBandEvict, []string{"llama3.1:8b/free-space", "llava:7b/free-space", "qwen3:4b/free-space"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := DecideModelEvictions(testModelRetentionPolicy(), []ModelStoreObservation{ollamaStore(tc.free, models...)}, decisionNow)
			if d.Band != tc.band {
				t.Fatalf("band = %s, want %s", d.Band, tc.band)
			}
			if got := strings.Join(selectedNames(d), ","); got != strings.Join(tc.selected, ",") {
				t.Fatalf("selected = %s, want %s", got, strings.Join(tc.selected, ","))
			}
			if tc.free == 5 && d.Filesystems[0].Shortfall == "" {
				t.Fatal("a selection that cannot reach the target must say so")
			}
			if tc.free == 9 && d.Filesystems[0].ProjectedFreePercent < 15 {
				t.Fatalf("projected %.2f%% must reach the 15%% target", d.Filesystems[0].ProjectedFreePercent)
			}
		})
	}
}

func TestModelEvictionHoldsRequiredLoadedRecentAndUnknownUsage(t *testing.T) {
	required := idleModel("gemma4:12b", 50*gib, false)
	required.Required, required.RequiredReason = true, "primary model of role code.local"
	loaded := idleModel("qwen3.5:9b", 40*gib, false)
	loaded.Loaded = true
	recentPull := idleModel("eval-candidate:1b", 30*gib, false)
	recentPull.ModifiedAt = decisionNow.AddDate(0, 0, -1)
	recentUse := idleModel("llama3.2:3b", 20*gib, false)
	blocked := idleModel("drifted:1b", 10*gib, true)
	blocked.Blocked = "installed digest differs from the pinned digest"
	shared := idleModel("alias:latest", 0, false)
	shared.Bytes = 5 * gib
	idle := idleModel("llava:7b", 5*gib, false)

	store := ollamaStore(5, required, loaded, recentPull, recentUse, blocked, shared, idle)
	store.Usage = usageTracked(map[string]time.Time{"llama3.2:3b": decisionNow.Add(-2 * time.Hour)})
	d := DecideModelEvictions(testModelRetentionPolicy(), []ModelStoreObservation{store}, decisionNow)
	held := map[string]string{}
	for _, c := range d.Candidates {
		held[c.Name] = c.Held
	}
	expect := map[string]string{
		"gemma4:12b":        "required: primary model of role code.local",
		"qwen3.5:9b":        "loaded in memory",
		"eval-candidate:1b": "used or pulled 24h0m0s ago",
		"llama3.2:3b":       "used or pulled 2h0m0s ago",
		"drifted:1b":        "owner refuses: installed digest differs",
		"alias:latest":      "frees no bytes",
		"llava:7b":          "",
	}
	for name, want := range expect {
		if !strings.HasPrefix(held[name], want) || (want == "" && held[name] != "") {
			t.Errorf("%s held = %q, want prefix %q", name, held[name], want)
		}
	}
	if got := strings.Join(selectedNames(d), ","); got != "llava:7b/free-space" {
		t.Fatalf("only the idle model may be selected, got %s", got)
	}

	// Tracking that began inside the idle window holds a long-idle model and
	// says why.
	fresh := ollamaStore(5, idleModel("llava:7b", 5*gib, false))
	fresh.Usage.TrackingSince = decisionNow.Add(-30 * time.Minute)
	if d := DecideModelEvictions(testModelRetentionPolicy(), []ModelStoreObservation{fresh}, decisionNow); len(d.Selected) != 0 || d.Candidates[0].Held != "usage tracked for only 30m0s; idle window is 14d" {
		t.Fatalf("fresh tracking: %+v", d.Candidates[0])
	}

	// No usage file yet means use is unknown, so nothing is eligible.
	store.Usage = modelretention.UsageDocument{}
	d = DecideModelEvictions(testModelRetentionPolicy(), []ModelStoreObservation{store}, decisionNow)
	if len(d.Selected) != 0 || !strings.HasPrefix(d.Candidates[len(d.Candidates)-1].Held, "") {
		t.Fatalf("untracked usage must hold every model, selected %v", selectedNames(d))
	}
	for _, c := range d.Candidates {
		if c.Name == "llava:7b" && !strings.HasPrefix(c.Held, "usage unknown") {
			t.Fatalf("llava held = %q", c.Held)
		}
	}
}

func TestModelEvictionEnforcesStoreBudgetsAtHealthyFreeSpace(t *testing.T) {
	ollaya := ModelStoreObservation{
		Resource: "ollaya", Space: space(40), Usage: usageTracked(nil), BudgetBytes: 20 * gib,
		Inventory: modelretention.Inventory{Resource: "ollaya", StoreBytes: 46 * gib, Models: []modelretention.InventoryModel{
			idleModel("kev:4b", 9*gib, true), idleModel("pplx-decider:flash", 9*gib, true), idleModel("credence:e4b-calibrated", 8*gib, true), idleModel("coba:1b", 4*gib, true),
		}},
	}
	ollama := ollamaStore(40, idleModel("llama3.1:8b", 5*gib, false))
	ollama.BudgetBytes = 88 * gib
	d := DecideModelEvictions(testModelRetentionPolicy(), []ModelStoreObservation{ollaya, ollama}, decisionNow)
	if d.Band != ModelBandOK {
		t.Fatalf("band = %s", d.Band)
	}
	// 26 GiB over budget: 9 + 9 + 8 = 26.
	if got := strings.Join(selectedNames(d), ","); got != "kev:4b/budget,pplx-decider:flash/budget,credence:e4b-calibrated/budget" {
		t.Fatalf("selected = %s", got)
	}
	if over := d.OverBudget(); len(over) != 1 || over[0].Resource != "ollaya" || over[0].OverBudgetBytes != 26*gib {
		t.Fatalf("over budget = %+v", over)
	}
}

func TestModelEvictionRanksDigestPinnedAheadOfRegistryTagsOfSimilarSize(t *testing.T) {
	ollama := ollamaStore(9, idleModel("tag-model:8b", 12*gib, false))
	ollaya := ModelStoreObservation{Resource: "ollaya", Space: space(9), Usage: usageTracked(nil), Inventory: modelretention.Inventory{Models: []modelretention.InventoryModel{idleModel("pinned:4b", 8*gib, true)}}}
	d := DecideModelEvictions(testModelRetentionPolicy(), []ModelStoreObservation{ollama, ollaya}, decisionNow)
	if d.Candidates[0].Name != "pinned:4b" {
		t.Fatalf("8 GiB exact refetch (score 8) must outrank a 12 GiB registry tag (score 6): %+v", d.Candidates)
	}
}

func TestModelEvictionReportsUnobservedStoresWithoutGuessing(t *testing.T) {
	broken := ModelStoreObservation{Resource: "ollaya", Err: errors.New("resource-ollaya: connection refused")}
	d := DecideModelEvictions(testModelRetentionPolicy(), []ModelStoreObservation{broken, ollamaStore(20)}, decisionNow)
	if errs := d.Errors(); len(errs) != 1 || errs[0].Resource != "ollaya" {
		t.Fatalf("errors = %+v", errs)
	}
	if lines := strings.Join(DecisionLines(d), "\n"); !strings.Contains(lines, "store ollaya: unobserved: resource-ollaya: connection refused") {
		t.Fatalf("lines must surface the unobserved store:\n%s", lines)
	}
}

func TestShippedModelRetentionPolicyLoads(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "config", "model-retention-policy.json")
	p, err := LoadModelRetentionPolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.NotifyBelowFreePercent != 15 || p.EvictBelowFreePercent != 10 || p.EvictUntilFreePercent != 15 || p.Approval != ModelApprovalOperator || len(p.Stores) != 2 {
		t.Fatalf("shipped policy = %+v", p)
	}
	bad := p
	bad.EvictBelowFreePercent = 20
	if bad.Validate() == nil {
		t.Fatal("an evict band above the notify band must be rejected")
	}
}

// fakeModelStore is the owner boundary with a recorded eviction call.
type fakeModelStore struct {
	name      string
	inventory modelretention.Inventory
	evicted   []ModelEvictionRequest
	refuse    string
}

func (f *fakeModelStore) Resource() string { return f.name }
func (f *fakeModelStore) Inventory(context.Context) (modelretention.Inventory, error) {
	return f.inventory, nil
}
func (f *fakeModelStore) Evict(_ context.Context, req ModelEvictionRequest) (modelretention.EvictResult, error) {
	if f.refuse != "" {
		return modelretention.EvictResult{Model: req.Name, Refused: f.refuse}, nil
	}
	f.evicted = append(f.evicted, req)
	kept := f.inventory.Models[:0]
	for _, m := range f.inventory.Models {
		if m.Name != req.Name {
			kept = append(kept, m)
		}
	}
	f.inventory.Models = kept
	return modelretention.EvictResult{Model: req.Name, Evicted: true, FreedBytes: req.Size, Reacquire: []string{"reacquire", req.Name}}, nil
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func newTestProvider(t *testing.T, free float64, store *fakeModelStore, usage modelretention.UsageDocument) *ModelStoreRetentionProvider {
	t.Helper()
	usagePath := filepath.Join(t.TempDir(), "usage.json")
	data, _ := json.Marshal(usage)
	if err := os.WriteFile(usagePath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return NewModelStoreRetentionProvider(testModelRetentionPolicy(), []ModelStore{store}, ModelStoreEnvironment{
		UsagePath: func(string) (string, error) { return usagePath, nil },
		Space:     func(string) (FilesystemSpace, error) { return space(free), nil },
		Budget:    func(string) (int64, error) { return 0, nil },
	}, fixedClock{decisionNow})
}

func TestProviderAppliesOnlyApprovedModelsThatAreStillEligible(t *testing.T) {
	store := &fakeModelStore{name: "ollama", inventory: modelretention.Inventory{Schema: modelretention.InventorySchema, ModelRoot: "/models", Models: []modelretention.InventoryModel{
		idleModel("llama3.1:8b", 40*gib, false), idleModel("llava:7b", 30*gib, false),
	}}}
	provider := newTestProvider(t, 9, store, usageTracked(nil))
	policy := cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: cleanup.ObservationScope{Now: decisionNow}, Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 2 || !strings.HasPrefix(preview.Items[0].Path, "model-store://ollama/") {
		t.Fatalf("preview = %+v", preview.Items)
	}
	// Without operator approval nothing reaches the owner.
	unapproved, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "k", ApprovalMode: cleanup.ApprovalModeOwner, Preview: preview})
	if err != nil || len(unapproved.SkippedItems) != 2 || len(store.evicted) != 0 {
		t.Fatalf("unapproved apply = %+v err=%v evicted=%v", unapproved, err, store.evicted)
	}
	// llava is reloaded between approval and apply: it must be skipped.
	store.inventory.Models[1].Loaded = true
	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "k2", ApprovalMode: cleanup.ApprovalModeOperator, Preview: preview})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.evicted) != 1 || store.evicted[0].Name != "llama3.1:8b" || store.evicted[0].Digest != strings.Repeat("d", 64) || store.evicted[0].Size != 40*gib {
		t.Fatalf("evicted = %+v", store.evicted)
	}
	if len(result.AppliedItems) != 1 || len(result.SkippedItems) != 1 || !strings.Contains(result.Warnings[0], "loaded in memory") || result.ReclaimedBytes != 40*gib {
		t.Fatalf("result = %+v", result)
	}
	verify, err := provider.Verify(context.Background(), cleanup.VerifyRequest{ApplyResult: result})
	if err != nil || verify.Verified {
		t.Fatalf("a partial apply must not verify clean: %+v", verify)
	}
}

func TestProviderReportsAnOwnerRefusal(t *testing.T) {
	store := &fakeModelStore{name: "ollama", refuse: "model has a request in flight", inventory: modelretention.Inventory{Schema: modelretention.InventorySchema, Models: []modelretention.InventoryModel{idleModel("llava:7b", 30*gib, false)}}}
	provider := newTestProvider(t, 9, store, usageTracked(nil))
	outcomes := provider.ApplyApproved(context.Background(), provider.Decide(context.Background()).Selected, "test", "test")
	if len(outcomes) != 1 || outcomes[0].Evicted || outcomes[0].Refused != "model has a request in flight" {
		t.Fatalf("outcomes = %+v", outcomes)
	}
}

func TestCommandModelStoreParsesOwnerDocuments(t *testing.T) {
	var calls []string
	store := CommandModelStore{Name: "ollama", Command: "resource-ollama", Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if args[1] == "retention" {
			return []byte("rebuilt resource-ollama from current sources\n{\"schema\":\"" + modelretention.InventorySchema + "\",\"resource\":\"ollama\",\"models\":[{\"name\":\"llava:7b\",\"bytes\":5}]}\n"), nil
		}
		return []byte(`{"model":"llava:7b","refused":"model is loaded in memory"}`), errors.New("exit status 1")
	}}
	inv, err := store.Inventory(context.Background())
	if err != nil || len(inv.Models) != 1 {
		t.Fatalf("inventory = %+v err=%v", inv, err)
	}
	result, err := store.Evict(context.Background(), ModelEvictionRequest{Name: "llava:7b", Digest: "abc", Size: 5, RequestedBy: "storage-manager"})
	if err != nil || result.Refused != "model is loaded in memory" {
		t.Fatalf("a refusal is a result, not an error: %+v err=%v", result, err)
	}
	if calls[1] != "resource-ollama models evict llava:7b --digest abc --size 5 --json --requested-by storage-manager" {
		t.Fatalf("evict argv = %q", calls[1])
	}
}
