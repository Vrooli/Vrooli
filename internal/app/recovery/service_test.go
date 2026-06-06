package recovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/baselinefloor"
)

// fixedClock returns a deterministic clock pinned to t.
func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// newTestService builds a Service rooted at a temp repo with a temp cache store
// and a pinned clock. It also seeds a scenario working tree with a real source
// file plus an excluded node_modules dir so capture/restore exercise the ladder.
func newTestService(t *testing.T, now time.Time) (Service, string) {
	t.Helper()
	root := t.TempDir()
	cache := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatalf("mkdir scenario: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "node_modules", "pkg", "index.js"), []byte("// junk\n"), 0o644); err != nil {
		t.Fatalf("write node_modules: %v", err)
	}
	svc := Service{Root: root, Store: baselinefloor.NewStore(cache), Clock: fixedClock(now)}
	return svc, scenarioDir
}

func TestCaptureExcludesBuildArtifactsAndRestores(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	svc, scenarioDir := newTestService(t, now)

	out, err := svc.Capture(CaptureRequest{Scenario: "demo", Slug: "abc", NoReflink: true})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if out.Source != scenarioDir {
		t.Fatalf("source = %q, want %q", out.Source, scenarioDir)
	}
	// The restore point must contain the real source file...
	if _, err := os.Stat(filepath.Join(out.RestorePointPath, "api", "main.go")); err != nil {
		t.Fatalf("restore point missing source: %v", err)
	}
	// ...but never the excluded node_modules tree.
	if _, err := os.Stat(filepath.Join(out.RestorePointPath, "node_modules")); !os.IsNotExist(err) {
		t.Fatalf("node_modules leaked into restore point (err=%v)", err)
	}
	if out.Stats.Excluded == 0 {
		t.Fatalf("expected at least one excluded entry, got %+v", out.Stats)
	}

	// Mutate the working tree, then Restore must roll the captured file back.
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "main.go"), []byte("package broken\n"), 0o644); err != nil {
		t.Fatalf("mutate source: %v", err)
	}
	if _, err := svc.Restore(RestoreRequest{Scenario: "demo", Slug: "abc", NoReflink: true}); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(scenarioDir, "api", "main.go"))
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if string(got) != "package main\n" {
		t.Fatalf("restore did not roll back: got %q", string(got))
	}
}

func TestWriteEngagementStampsAndDefaults(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, now)

	view, err := svc.WriteEngagement(WriteRequest{
		Scenario: "demo",
		Slug:     "abc",
		Mode:     "shadow",
		TTL:      3 * time.Hour,
	})
	if err != nil {
		t.Fatalf("WriteEngagement: %v", err)
	}
	if view.Variant != "shadow" {
		t.Fatalf("default variant = %q, want shadow", view.Variant)
	}
	if view.ShadowInstanceKey != "demo@shadow" {
		t.Fatalf("default shadow instance key = %q, want demo@shadow", view.ShadowInstanceKey)
	}
	if !view.CreatedAt.Equal(now) || !view.LastTouchedAt.Equal(now) {
		t.Fatalf("timestamps not stamped: created=%v touched=%v", view.CreatedAt, view.LastTouchedAt)
	}
	if view.RestorePointPath != svc.Store.RestorePointPath("demo", "abc") {
		t.Fatalf("restore point path not derived: %q", view.RestorePointPath)
	}
	wantExpiry := now.Add(3 * time.Hour)
	if view.ExpiresAt == nil || !view.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("expiry = %v, want %v", view.ExpiresAt, wantExpiry)
	}
	if view.Expired {
		t.Fatalf("fresh engagement should not be expired")
	}
}

func TestWriteEngagementPreservesCreatedAtOnRewrite(t *testing.T) {
	created := time.Date(2026, 6, 5, 9, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, created)
	if _, err := svc.WriteEngagement(WriteRequest{Scenario: "demo", Slug: "abc", Mode: "live"}); err != nil {
		t.Fatalf("first write: %v", err)
	}

	later := created.Add(2 * time.Hour)
	svc.Clock = fixedClock(later)
	view, err := svc.WriteEngagement(WriteRequest{Scenario: "demo", Slug: "abc", Mode: "live"})
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if !view.CreatedAt.Equal(created) {
		t.Fatalf("CreatedAt not preserved: got %v want %v", view.CreatedAt, created)
	}
	if !view.LastTouchedAt.Equal(later) {
		t.Fatalf("LastTouchedAt not renewed: got %v want %v", view.LastTouchedAt, later)
	}
	if view.Variant != "live" || view.ShadowInstanceKey != "" {
		t.Fatalf("live mode leaked shadow fields: variant=%q key=%q", view.Variant, view.ShadowInstanceKey)
	}
}

func TestWriteEngagementInvalidMode(t *testing.T) {
	svc, _ := newTestService(t, time.Now())
	if _, err := svc.WriteEngagement(WriteRequest{Scenario: "demo", Slug: "abc", Mode: "bogus"}); err == nil {
		t.Fatal("expected invalid-mode error")
	}
}

func TestTouchRenewsLeaseAndExpiry(t *testing.T) {
	created := time.Date(2026, 6, 5, 9, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, created)
	if _, err := svc.WriteEngagement(WriteRequest{Scenario: "demo", Slug: "abc", Mode: "shadow", TTL: time.Hour}); err != nil {
		t.Fatalf("write: %v", err)
	}

	// 90 min later the engagement is idle-expired against the old touch...
	checkAt := created.Add(90 * time.Minute)
	svc.Clock = fixedClock(checkAt)
	stale, err := svc.ShowEngagement(Ref{Scenario: "demo", Slug: "abc"})
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if !stale.Expired {
		t.Fatal("engagement should be expired before touch")
	}
	// ...but a touch-on-access renews the lease and clears the expiry.
	touched, err := svc.Touch(Ref{Scenario: "demo", Slug: "abc"})
	if err != nil {
		t.Fatalf("touch: %v", err)
	}
	if touched.Expired {
		t.Fatal("touch should renew the lease")
	}
	if !touched.LastTouchedAt.Equal(checkAt) {
		t.Fatalf("touch timestamp = %v, want %v", touched.LastTouchedAt, checkAt)
	}
}

func TestSetTTLClearsWithZero(t *testing.T) {
	now := time.Date(2026, 6, 5, 9, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, now)
	if _, err := svc.WriteEngagement(WriteRequest{Scenario: "demo", Slug: "abc", Mode: "shadow", TTL: time.Hour}); err != nil {
		t.Fatalf("write: %v", err)
	}
	view, err := svc.SetTTL(SetTTLRequest{Scenario: "demo", Slug: "abc", TTL: 0})
	if err != nil {
		t.Fatalf("set-ttl: %v", err)
	}
	if view.TTL != 0 || view.ExpiresAt != nil {
		t.Fatalf("zero TTL should clear expiry: ttl=%v expires=%v", view.TTL, view.ExpiresAt)
	}
}

func TestListGlobsAllEngagementsSorted(t *testing.T) {
	first := time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, first)
	if _, err := svc.WriteEngagement(WriteRequest{Scenario: "demo", Slug: "older", Mode: "shadow"}); err != nil {
		t.Fatalf("write older: %v", err)
	}
	svc.Clock = fixedClock(first.Add(time.Hour))
	if _, err := svc.WriteEngagement(WriteRequest{Scenario: "other", Slug: "newer", Mode: "live"}); err != nil {
		t.Fatalf("write newer: %v", err)
	}

	list, err := svc.ListEngagements()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Engagements) != 2 {
		t.Fatalf("expected 2 engagements, got %d", len(list.Engagements))
	}
	if list.Engagements[0].Slug != "older" || list.Engagements[1].Slug != "newer" {
		t.Fatalf("not sorted oldest-first: %s, %s", list.Engagements[0].Slug, list.Engagements[1].Slug)
	}
}

func TestCleanIsIdempotent(t *testing.T) {
	now := time.Now()
	svc, _ := newTestService(t, now)
	if _, err := svc.WriteEngagement(WriteRequest{Scenario: "demo", Slug: "abc", Mode: "shadow"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := svc.Clean(Ref{Scenario: "demo", Slug: "abc"})
	if err != nil {
		t.Fatalf("clean: %v", err)
	}
	if _, err := os.Stat(out.EngagementDir); !os.IsNotExist(err) {
		t.Fatalf("engagement dir not removed (err=%v)", err)
	}
	// A second clean of a now-missing engagement is not an error.
	if _, err := svc.Clean(Ref{Scenario: "demo", Slug: "abc"}); err != nil {
		t.Fatalf("idempotent clean failed: %v", err)
	}
}

func TestRefValidation(t *testing.T) {
	svc, _ := newTestService(t, time.Now())
	if _, err := svc.Capture(CaptureRequest{Scenario: "", Slug: "abc"}); err == nil {
		t.Fatal("expected scenario-required error")
	}
	if _, err := svc.ShowEngagement(Ref{Scenario: "demo", Slug: ""}); err == nil {
		t.Fatal("expected slug-required error")
	}
}
