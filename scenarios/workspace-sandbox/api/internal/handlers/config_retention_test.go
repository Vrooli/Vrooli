package handlers_test

// Phase 4 endpoint tests for /api/v1/config/retention. The handler
// delegates persistence to RetentionStore; the tests use the in-memory
// MemoryRetentionStore to assert the wire shape, validation behavior,
// and the partial-update merge semantics.

import (
	"encoding/json"
	"net/http"
	"testing"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/testutil/mocks/sandboxiface"
)

// withRetention installs a RetentionStore into the Handlers wired by
// newLive. Tests pass either a pre-seeded MemoryRetentionStore or a
// FileRetentionStore at a temp path.
func withRetention(store config.RetentionStore) liveOpt {
	return func(h *handlers.Handlers) { h.RetentionStore = store }
}

// TestGetRetention_Defaults — when the store is freshly seeded, GET
// returns those values verbatim.
func TestGetRetention_Defaults(t *testing.T) {
	seed := config.RetentionConfig{
		MaxArchiveAgeDays:     90,
		MaxArchiveSizeBytes:   10 << 30,
		MaxArchivesPerProject: 0,
	}
	store := config.NewMemoryRetentionStore(seed)
	live := newLive(t, &sandboxiface.FakeService{}, withRetention(store))

	resp, body := live.Do(t, "GET", "/api/v1/config/retention", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	var got struct {
		MaxArchiveAgeDays     int   `json:"maxArchiveAgeDays"`
		MaxArchiveSizeBytes   int64 `json:"maxArchiveSizeBytes"`
		MaxArchivesPerProject int   `json:"maxArchivesPerProject"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, body)
	}
	if got.MaxArchiveAgeDays != 90 || got.MaxArchiveSizeBytes != 10<<30 || got.MaxArchivesPerProject != 0 {
		t.Errorf("got %+v, want seed %+v", got, seed)
	}
}

// TestGetRetention_NoStore — when the handler isn't configured with a
// store, GET returns 503.
func TestGetRetention_NoStore(t *testing.T) {
	live := newLive(t, &sandboxiface.FakeService{})
	resp, _ := live.Do(t, "GET", "/api/v1/config/retention", nil)
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

// TestUpdateRetention_FullUpdate — PUT with all three fields persists
// and the next GET returns the new values.
func TestUpdateRetention_FullUpdate(t *testing.T) {
	store := config.NewMemoryRetentionStore(config.RetentionConfig{})
	live := newLive(t, &sandboxiface.FakeService{}, withRetention(store))

	resp, body := live.DoJSON(t, "PUT", "/api/v1/config/retention",
		`{"maxArchiveAgeDays": 30, "maxArchiveSizeBytes": 1073741824, "maxArchivesPerProject": 50}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	got := store.Get()
	if got.MaxArchiveAgeDays != 30 || got.MaxArchiveSizeBytes != 1073741824 || got.MaxArchivesPerProject != 50 {
		t.Errorf("store after PUT = %+v, want updated", got)
	}
}

// TestUpdateRetention_PartialUpdate — PUT with only one field leaves
// the other two unchanged. Critical for the UI's "tweak one knob" UX.
func TestUpdateRetention_PartialUpdate(t *testing.T) {
	store := config.NewMemoryRetentionStore(config.RetentionConfig{
		MaxArchiveAgeDays:     90,
		MaxArchiveSizeBytes:   10 << 30,
		MaxArchivesPerProject: 100,
	})
	live := newLive(t, &sandboxiface.FakeService{}, withRetention(store))

	resp, _ := live.DoJSON(t, "PUT", "/api/v1/config/retention", `{"maxArchiveAgeDays": 7}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	got := store.Get()
	if got.MaxArchiveAgeDays != 7 {
		t.Errorf("MaxArchiveAgeDays = %d, want 7", got.MaxArchiveAgeDays)
	}
	if got.MaxArchiveSizeBytes != 10<<30 {
		t.Errorf("MaxArchiveSizeBytes = %d, want unchanged %d", got.MaxArchiveSizeBytes, int64(10<<30))
	}
	if got.MaxArchivesPerProject != 100 {
		t.Errorf("MaxArchivesPerProject = %d, want unchanged 100", got.MaxArchivesPerProject)
	}
}

// TestUpdateRetention_ZeroIsExplicit — explicitly passing 0 disables a
// lever; not the same as omitting the field.
func TestUpdateRetention_ZeroIsExplicit(t *testing.T) {
	store := config.NewMemoryRetentionStore(config.RetentionConfig{
		MaxArchiveAgeDays:     90,
		MaxArchiveSizeBytes:   10 << 30,
		MaxArchivesPerProject: 100,
	})
	live := newLive(t, &sandboxiface.FakeService{}, withRetention(store))

	resp, _ := live.DoJSON(t, "PUT", "/api/v1/config/retention", `{"maxArchivesPerProject": 0}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	got := store.Get()
	if got.MaxArchivesPerProject != 0 {
		t.Errorf("MaxArchivesPerProject should be 0 (explicit), got %d", got.MaxArchivesPerProject)
	}
	if got.MaxArchiveAgeDays != 90 {
		t.Errorf("MaxArchiveAgeDays = %d, want unchanged 90", got.MaxArchiveAgeDays)
	}
}

// TestUpdateRetention_RejectsNegative — validation refuses negatives
// without mutating the store.
func TestUpdateRetention_RejectsNegative(t *testing.T) {
	seed := config.RetentionConfig{MaxArchiveAgeDays: 90}
	store := config.NewMemoryRetentionStore(seed)
	live := newLive(t, &sandboxiface.FakeService{}, withRetention(store))

	cases := []string{
		`{"maxArchiveAgeDays": -1}`,
		`{"maxArchiveSizeBytes": -1}`,
		`{"maxArchivesPerProject": -1}`,
	}
	for _, body := range cases {
		resp, _ := live.DoJSON(t, "PUT", "/api/v1/config/retention", body)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("PUT %s -> %d, want 400", body, resp.StatusCode)
		}
	}
	if got := store.Get(); got != seed {
		t.Errorf("store mutated after rejected PUTs: got %+v, want %+v", got, seed)
	}
}

// TestUpdateRetention_MalformedJSON — bad JSON returns 400.
func TestUpdateRetention_MalformedJSON(t *testing.T) {
	store := config.NewMemoryRetentionStore(config.RetentionConfig{})
	live := newLive(t, &sandboxiface.FakeService{}, withRetention(store))

	resp, _ := live.DoJSON(t, "PUT", "/api/v1/config/retention", `{ broken json`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// TestUpdateRetention_NoStore — PUT without a configured store returns
// 503, matching GET.
func TestUpdateRetention_NoStore(t *testing.T) {
	live := newLive(t, &sandboxiface.FakeService{})
	resp, _ := live.DoJSON(t, "PUT", "/api/v1/config/retention", `{"maxArchiveAgeDays": 1}`)
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

// TestUpdateRetention_RoundTrip — the response of PUT is the new
// effective state, equivalent to a follow-up GET. The UI uses this to
// confirm without a second request.
func TestUpdateRetention_RoundTrip(t *testing.T) {
	store := config.NewMemoryRetentionStore(config.RetentionConfig{})
	live := newLive(t, &sandboxiface.FakeService{}, withRetention(store))

	_, body := live.DoJSON(t, "PUT", "/api/v1/config/retention",
		`{"maxArchiveAgeDays": 14, "maxArchiveSizeBytes": 5000, "maxArchivesPerProject": 25}`)

	var put struct {
		MaxArchiveAgeDays     int   `json:"maxArchiveAgeDays"`
		MaxArchiveSizeBytes   int64 `json:"maxArchiveSizeBytes"`
		MaxArchivesPerProject int   `json:"maxArchivesPerProject"`
	}
	if err := json.Unmarshal(body, &put); err != nil {
		t.Fatalf("decode PUT: %v", err)
	}

	_, body2 := live.Do(t, "GET", "/api/v1/config/retention", nil)
	var get struct {
		MaxArchiveAgeDays     int   `json:"maxArchiveAgeDays"`
		MaxArchiveSizeBytes   int64 `json:"maxArchiveSizeBytes"`
		MaxArchivesPerProject int   `json:"maxArchivesPerProject"`
	}
	if err := json.Unmarshal(body2, &get); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if put != get {
		t.Errorf("PUT response (%+v) != follow-up GET (%+v)", put, get)
	}
}
