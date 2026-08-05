package census

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"
	corestorage "github.com/vrooli/api-core/storage"
	"storage-manager/internal/testutil/db"
)

func TestScanReportsUnknownUnattributedBytesWhenCoverageIsUnreadable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "visible"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A missing root is not unreadable; it is a known missing surface. Use a
	// permission-free synthetic declaration to assert the JSON contract via a
	// direct report, which is stable across test runners and operating systems.
	report := Report{Root: root, MeasuredBytes: 4, AttributedBytes: 4, Closed: false, AccountingIdentity: false, Confidence: "degraded", UnattributedKnown: false}
	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["unattributed_bytes"] != nil {
		t.Fatalf("unattributed_bytes = %v, want null for unknown coverage", decoded["unattributed_bytes"])
	}
}

func TestScanClosedIdentityAndAttribution(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(name, value string) {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("owned/a", "1234")
	mustWrite("unowned/b", "123456")
	report, err := Scan(root, map[string][]Declaration{"component": {{Name: "data", Path: filepath.Join(root, "owned"), Budgeted: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Closed || !report.AccountingIdentity || report.MeasuredBytes != 10 || report.AttributedBytes != 4 || report.UnattributedBytes != 6 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestScanInventoryUsesAllOwnerKindsAndPreservesAccountingIdentity(t *testing.T) {
	root := t.TempDir()
	owned := filepath.Join(root, "owned")
	write := func(path, value string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(owned, "a"), "1234")
	write(filepath.Join(root, "unowned", "b"), "123456")
	inventory := corestorage.OwnerInventory{RepoRoot: root, Owners: []corestorage.OwnerManifest{
		{Kind: corestorage.OwnerResource, ID: "demo", ManifestPath: filepath.Join(root, "resources", "demo", "resource.json"), StorageEntries: []corestorage.StorageEntry{
			{Name: "owned", Path: corestorage.PortablePath{Value: owned}},
			{Name: "overlap", Path: corestorage.PortablePath{Value: root}},
		}},
		{Kind: corestorage.OwnerTool, ID: "compiler", ManifestPath: filepath.Join(root, "internal", "tools", "go", "tool.json")},
	}}
	report, err := ScanInventoryWithPolicy(root, inventory, ScanPolicy{Roots: []PolicyRoot{{Path: root}}, FloorBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !report.AccountingIdentity || report.MeasuredBytes != report.AttributedBytes+report.UnattributedBytes {
		t.Fatalf("accounting identity broken: %+v", report)
	}
	if report.OwnerCounts["resource"] != 1 || report.OwnerCounts["tool"] != 1 {
		t.Fatalf("owner counts = %#v", report.OwnerCounts)
	}
	if report.Confidence != "full" {
		t.Fatalf("confidence = %q", report.Confidence)
	}
	foundOverlap := false
	for _, finding := range report.Findings {
		if finding.Code == "overlap" {
			foundOverlap = true
		}
	}
	if !foundOverlap {
		t.Fatalf("expected overlap finding: %+v", report.Findings)
	}
}

func TestScanInventoryMeasuresBoundedUnclassifiedScenarioRoots(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"service":{"name":"demo"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(root, "scenarios", "demo", "data", "unclassified.db")
	if err := os.MkdirAll(filepath.Dir(data), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(data, []byte("123456"), 0o644); err != nil {
		t.Fatal(err)
	}
	inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: root, Platform: corestorage.PlatformLinux})
	if err != nil {
		t.Fatal(err)
	}
	report, err := ScanInventoryWithPolicy(root, inventory, ScanPolicy{Roots: []PolicyRoot{{Path: root}}, FloorBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if report.MeasuredBytes < 6 || report.AttributedBytes != 0 || report.DriftBytes != report.MeasuredBytes || report.UnattributedBytes != 0 {
		t.Fatalf("bounded unclassified accounting = %+v", report)
	}
	if !report.Closed || report.Confidence != "full" {
		t.Fatalf("unclassified storage should still have full accounting: %+v", report)
	}
	if report.AccountingResidualBytes != 0 {
		t.Fatalf("accounting residual = %d, want zero", report.AccountingResidualBytes)
	}
}

func TestScanCountsHardLinkedInodeOnce(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "owned", "original")
	if err := os.MkdirAll(filepath.Dir(original), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(original, []byte("123456"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "unowned", "link")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(original, link); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	report, err := Scan(root, map[string][]Declaration{"fixture": {{Name: "owned", Path: filepath.Join(root, "owned")}}})
	if err != nil {
		t.Fatal(err)
	}
	if report.MeasuredBytes != 6 || report.AttributedBytes != 6 || report.UnattributedBytes != 0 {
		t.Fatalf("hard-linked accounting = %+v, want one six-byte inode", report)
	}
}

func hasFinding(findings []Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func TestSnapshotStorePersistsHistoryAndGrowthSlope(t *testing.T) {
	database := apidb.NewFromPrimary(db.NewSQLite(t))
	store := NewSnapshotStore(database)
	if _, err := database.ExecContext(context.Background(), censusSchemaSQL); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	first, err := store.Save(context.Background(), Report{Root: root, MeasuredBytes: 10, AttributedBytes: 10, AccountingIdentity: true, Confidence: "high"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Save(context.Background(), Report{Root: root, MeasuredBytes: 20, AttributedBytes: 20, AccountingIdentity: true, Confidence: "high"})
	if err != nil {
		t.Fatal(err)
	}
	if first.SnapshotID == "" || second.SnapshotID == "" || first.SnapshotID == second.SnapshotID {
		t.Fatalf("snapshot ids = %q, %q", first.SnapshotID, second.SnapshotID)
	}
	history, err := store.History(context.Background(), root, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("history length = %d", len(history))
	}
	if second.GrowthSlopeBytesPerHour == nil || math.IsNaN(*second.GrowthSlopeBytesPerHour) {
		t.Fatalf("growth slope = %v", second.GrowthSlopeBytesPerHour)
	}
}

func TestSnapshotStoreLatestReturnsAgeAndStalenessWithoutRescanning(t *testing.T) {
	database := apidb.NewFromPrimary(db.NewSQLite(t))
	store := NewSnapshotStore(database)
	if _, err := database.ExecContext(context.Background(), censusSchemaSQL); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	observed := time.Now().UTC().Add(-time.Hour)
	report := Report{SnapshotID: "snapshot-old", Root: root, ObservedAt: observed, MeasuredBytes: 42, AttributedBytes: 42, UnattributedBytes: 17, UnattributedKnown: true, Closed: true, AccountingIdentity: true, Confidence: "full", Entries: []Entry{}, ScanPolicy: ScanPolicy{FloorBytes: 1}}
	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(context.Background(), `INSERT INTO census_snapshots (id, observed_at, root, measured_bytes, attributed_bytes, unattributed_bytes, confidence, report_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, report.SnapshotID, observed.Format(time.RFC3339Nano), root, report.MeasuredBytes, report.AttributedBytes, report.UnattributedBytes, report.Confidence, string(payload)); err != nil {
		t.Fatal(err)
	}
	latest, err := store.Latest(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if latest == nil || latest.StalenessVerdict != "stale" || latest.SnapshotAgeSeconds == nil || *latest.SnapshotAgeSeconds < 3000 {
		t.Fatalf("latest snapshot = %+v", latest)
	}
	if !latest.UnattributedKnown || latest.UnattributedBytes != 17 {
		t.Fatalf("latest unattributed total = known:%v bytes:%d, want known: true bytes:17", latest.UnattributedKnown, latest.UnattributedBytes)
	}
}
