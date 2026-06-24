package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/systemevents"

	_ "modernc.org/sqlite"
)

// productionSchemaPath returns the path to the canonical SQLite schema file.
// Tests load the same schema the runtime uses so that index-defeating query
// regressions are caught here instead of at deploy time.
func productionSchemaPath(t *testing.T) string {
	t.Helper()
	// store_sqlite_test.go lives at scenarios/vrooli-autoheal/api/internal/persistence/
	// schema lives at  scenarios/vrooli-autoheal/initialization/sqlite/schema.sql
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	return filepath.Join(wd, "..", "..", "..", "initialization", "sqlite", "schema.sql")
}

func TestSQLiteStore_SaveAndReadHealthResults(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)

	result := checks.Result{
		CheckID:   "infra-network",
		Status:    checks.StatusOK,
		Message:   "Network connectivity OK",
		Details:   map[string]interface{}{"latencyMs": 8},
		Duration:  25 * time.Millisecond,
		Timestamp: time.Now().UTC(),
	}

	if err := store.SaveResult(context.Background(), result); err != nil {
		t.Fatalf("SaveResult() error = %v", err)
	}

	latest, err := store.GetLatestResultPerCheck(context.Background())
	if err != nil {
		t.Fatalf("GetLatestResultPerCheck() error = %v", err)
	}
	if len(latest) != 1 {
		t.Fatalf("latest count = %d, want 1", len(latest))
	}
	if latest[0].CheckID != "infra-network" {
		t.Fatalf("check id = %q, want infra-network", latest[0].CheckID)
	}

	stats, err := store.GetUptimeStats(context.Background(), 24)
	if err != nil {
		t.Fatalf("GetUptimeStats() error = %v", err)
	}
	if stats.TotalEvents != 1 {
		t.Fatalf("total events = %d, want 1", stats.TotalEvents)
	}
}

func TestSQLiteStore_JournalCursorRoundTrip(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	ctx := context.Background()
	const key = "journalctl/kernel"

	// Cold read: empty state, no error.
	got, err := store.GetJournalCursor(ctx, key)
	if err != nil {
		t.Fatalf("GetJournalCursor (cold): %v", err)
	}
	if got.Cursor != "" || got.BootID != "" {
		t.Fatalf("cold cursor = %+v, want empty", got)
	}

	// Persist and read back.
	want := systemevents.CursorState{Cursor: "s=abc;i=42", BootID: "boot-1", UpdatedAt: time.Now().UTC()}
	if err := store.SetJournalCursor(ctx, key, want); err != nil {
		t.Fatalf("SetJournalCursor: %v", err)
	}
	got, err = store.GetJournalCursor(ctx, key)
	if err != nil {
		t.Fatalf("GetJournalCursor: %v", err)
	}
	if got.Cursor != want.Cursor || got.BootID != want.BootID {
		t.Fatalf("cursor round-trip = %+v, want %+v", got, want)
	}

	// Upsert (advance) overwrites in place.
	advanced := systemevents.CursorState{Cursor: "s=abc;i=99", BootID: "boot-1"}
	if err := store.SetJournalCursor(ctx, key, advanced); err != nil {
		t.Fatalf("SetJournalCursor (advance): %v", err)
	}
	got, _ = store.GetJournalCursor(ctx, key)
	if got.Cursor != "s=abc;i=99" {
		t.Fatalf("advanced cursor = %q, want s=abc;i=99", got.Cursor)
	}
}

func TestSQLiteStore_ScannedBootMarker(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	ctx := context.Background()
	const key = "journalctl/kernel"

	scanned, err := store.IsBootScanned(ctx, key, "boot-x")
	if err != nil {
		t.Fatalf("IsBootScanned (cold): %v", err)
	}
	if scanned {
		t.Fatal("boot-x should not be scanned initially")
	}

	if err := store.MarkBootScanned(ctx, key, "boot-x"); err != nil {
		t.Fatalf("MarkBootScanned: %v", err)
	}
	// Idempotent: marking twice is fine.
	if err := store.MarkBootScanned(ctx, key, "boot-x"); err != nil {
		t.Fatalf("MarkBootScanned (twice): %v", err)
	}
	scanned, _ = store.IsBootScanned(ctx, key, "boot-x")
	if !scanned {
		t.Fatal("boot-x should be scanned after marking")
	}
	// A different boot id under the same key is independent.
	other, _ := store.IsBootScanned(ctx, key, "boot-y")
	if other {
		t.Fatal("boot-y should be independent of boot-x")
	}
}

func TestSQLiteStore_ActionLogRoundTrip(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)

	err := store.SaveActionLog(
		context.Background(),
		"infra-network",
		"restart",
		true,
		false,
		"Restart executed",
		"ok",
		"",
		42,
	)
	if err != nil {
		t.Fatalf("SaveActionLog() error = %v", err)
	}

	logs, err := store.GetActionLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetActionLogs() error = %v", err)
	}
	if logs.Total != 1 {
		t.Fatalf("logs total = %d, want 1", logs.Total)
	}
	if logs.Logs[0].CheckID != "infra-network" {
		t.Fatalf("check id = %q, want infra-network", logs.Logs[0].CheckID)
	}
}

func TestSQLiteStore_SystemEventsDedupFilterAndCorrelate(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	ctx := context.Background()
	base := time.Date(2026, 5, 6, 6, 10, 0, 0, time.UTC)
	events := []systemevents.Event{
		{Fingerprint: "kernel-1", OccurredAt: base, IngestedAt: base, Source: "dpkg-log", Platform: "linux", Category: "kernel", Severity: systemevents.SeverityInfo, Title: "Kernel installed", Summary: "linux-image installed"},
		{Fingerprint: "crash-1", OccurredAt: base.Add(2 * time.Hour), IngestedAt: base.Add(2 * time.Hour), Source: "journalctl", Platform: "linux", Category: "crash", Severity: systemevents.SeverityCritical, Title: "Hardware/reset signal", Summary: "unclean reset"},
	}

	inserted, deduped, err := store.UpsertSystemEvents(ctx, events)
	if err != nil {
		t.Fatalf("UpsertSystemEvents() error = %v", err)
	}
	if inserted != 2 || deduped != 0 {
		t.Fatalf("first upsert inserted/deduped = %d/%d, want 2/0", inserted, deduped)
	}
	inserted, deduped, err = store.UpsertSystemEvents(ctx, events)
	if err != nil {
		t.Fatalf("second UpsertSystemEvents() error = %v", err)
	}
	if inserted != 0 || deduped != 2 {
		t.Fatalf("second upsert inserted/deduped = %d/%d, want 0/2", inserted, deduped)
	}

	resp, err := store.ListSystemEvents(ctx, systemevents.Filters{Category: []string{"kernel"}, Limit: 10, Correlate: true})
	if err != nil {
		t.Fatalf("ListSystemEvents() error = %v", err)
	}
	if len(resp.Events) != 1 || resp.Events[0].Category != "kernel" {
		t.Fatalf("filtered events = %#v, want one kernel event", resp.Events)
	}

	resp, err = store.ListSystemEvents(ctx, systemevents.Filters{Limit: 10, Correlate: true})
	if err != nil {
		t.Fatalf("ListSystemEvents(correlate) error = %v", err)
	}
	if len(resp.Correlations) == 0 {
		t.Fatal("expected correlation hints")
	}
}

func TestSQLiteStore_HostInventorySnapshotRoundTrip(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	_, _, err := store.SaveHostInventorySnapshot(ctx, hostinventory.HostInventory{
		CollectedAt: time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
		Platform:    "linux",
		OS:          "linux",
		Arch:        "amd64",
		BootID:      "boot-a",
		Kernel: hostinventory.KernelInfo{
			Release:           "1.2.3-test",
			ModuleTreePresent: true,
		},
		ProbeStatus: map[string]hostinventory.ProbeState{"kernel": hostinventory.ProbeOK},
	})
	if err != nil {
		t.Fatalf("SaveHostInventorySnapshot() error = %v", err)
	}
	latest, err := store.GetLatestHostInventorySnapshot(ctx)
	if err != nil {
		t.Fatalf("GetLatestHostInventorySnapshot() error = %v", err)
	}
	if latest == nil {
		t.Fatal("expected latest snapshot")
	}
	if latest.KernelRelease != "1.2.3-test" {
		t.Fatalf("kernel release = %q, want 1.2.3-test", latest.KernelRelease)
	}
}

func TestSQLiteStore_IncidentLifecycle(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	incident, err := store.UpsertIncident(ctx, incidents.UpsertInput{
		Fingerprint:   incidents.Fingerprint("host_integrity", "host-runtime-integrity"),
		Type:          incidents.TypeHostIntegrity,
		Severity:      incidents.SeverityCritical,
		Title:         "Host integrity issue detected",
		Summary:       "Runtime failed",
		ObservedAt:    time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
		SourceCheckID: "host-runtime-integrity",
	})
	if err != nil {
		t.Fatalf("UpsertIncident() error = %v", err)
	}
	if incident.Status != incidents.StatusOpen {
		t.Fatalf("status = %s, want open", incident.Status)
	}

	updated, err := store.UpdateIncidentStatus(ctx, incident.ID, incidents.StatusAcknowledged, "reviewing")
	if err != nil {
		t.Fatalf("UpdateIncidentStatus() error = %v", err)
	}
	if updated.Status != incidents.StatusAcknowledged {
		t.Fatalf("status = %s, want acknowledged", updated.Status)
	}
	list, err := store.ListIncidents(ctx, incidents.ListFilters{Status: incidents.StatusAcknowledged, Limit: 10})
	if err != nil {
		t.Fatalf("ListIncidents() error = %v", err)
	}
	if list.Total != 1 {
		t.Fatalf("incident count = %d, want 1", list.Total)
	}
}

func TestSQLiteStore_RecordsRemediationArtifactAndPreservesAcrossUpsert(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	input := incidents.UpsertInput{
		Fingerprint:           incidents.Fingerprint("host_integrity", "host-kernel-module-drift", "boot-a"),
		Type:                  incidents.TypeHostIntegrity,
		Severity:              incidents.SeverityCritical,
		Title:                 "NVIDIA module package missing",
		Summary:               "Missing matching module package",
		ObservedAt:            time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
		SourceCheckID:         "host-kernel-module-drift",
		RemediationCandidates: []incidents.RemediationCandidate{{ID: "ubuntu-nvidia-kernel-module-mismatch", Applicability: "applicable"}},
	}
	incident, err := store.UpsertIncident(ctx, input)
	if err != nil {
		t.Fatalf("UpsertIncident() error = %v", err)
	}
	artifact := incidents.RemediationArtifact{
		ID:            "ubuntu-nvidia-kernel-module-mismatch-artifact",
		RemediationID: "ubuntu-nvidia-kernel-module-mismatch",
		Path:          "/tmp/remediation",
		GeneratedAt:   time.Date(2026, 5, 8, 12, 5, 0, 0, time.UTC),
		Metadata:      map[string]any{"templateId": "ubuntu-nvidia-kernel-module-mismatch"},
	}
	incident, err = store.RecordIncidentRemediationArtifact(ctx, incident.ID, artifact)
	if err != nil {
		t.Fatalf("RecordIncidentRemediationArtifact() error = %v", err)
	}
	if len(incident.RemediationArtifacts) != 1 {
		t.Fatalf("artifacts = %d, want 1", len(incident.RemediationArtifacts))
	}

	input.ObservedAt = input.ObservedAt.Add(time.Minute)
	incident, err = store.UpsertIncident(ctx, input)
	if err != nil {
		t.Fatalf("second UpsertIncident() error = %v", err)
	}
	if len(incident.RemediationArtifacts) != 1 || incident.RemediationArtifacts[0].Path != artifact.Path {
		t.Fatalf("artifact not preserved across upsert: %#v", incident.RemediationArtifacts)
	}
}

func TestSQLiteStore_RecordsRemediationOutcome(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	incident, err := store.UpsertIncident(ctx, incidents.UpsertInput{
		Fingerprint:   incidents.Fingerprint("host_integrity", "host-kernel-module-drift", "boot-a"),
		Type:          incidents.TypeHostIntegrity,
		Severity:      incidents.SeverityCritical,
		Title:         "NVIDIA module package missing",
		Summary:       "Missing matching module package",
		ObservedAt:    time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
		SourceCheckID: "host-kernel-module-drift",
	})
	if err != nil {
		t.Fatalf("UpsertIncident() error = %v", err)
	}
	updated, err := store.RecordIncidentRemediationOutcome(ctx, incident.ID, incidents.Outcome{
		RemediationID: "ubuntu-nvidia-kernel-module-mismatch",
		Status:        "verified",
		Note:          "post-checks are healthy",
		ReportedAt:    time.Date(2026, 5, 8, 12, 10, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("RecordIncidentRemediationOutcome() error = %v", err)
	}
	if updated.Outcome == nil || updated.Outcome.Status != "verified" {
		t.Fatalf("outcome = %#v, want verified", updated.Outcome)
	}
	if !strings.Contains(updated.OperatorNotes, "post-checks are healthy") {
		t.Fatalf("operator notes = %q, want outcome note", updated.OperatorNotes)
	}
}

func TestSQLiteStore_IncidentUpsertCoalescesRepeatedTickObservations(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	ctx := context.Background()
	fp := incidents.Fingerprint("host_integrity", "host-runtime-integrity", "boot-a")
	firstSeen := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	input := incidents.UpsertInput{
		Fingerprint:   fp,
		Type:          incidents.TypeHostIntegrity,
		Severity:      incidents.SeverityCritical,
		Title:         "Host integrity issue detected",
		Summary:       "Runtime failed",
		ObservedAt:    firstSeen,
		SourceCheckID: "host-runtime-integrity",
		Evidence:      map[string]any{"runtime": "nvidia-smi"},
	}

	incident, err := store.UpsertIncident(ctx, input)
	if err != nil {
		t.Fatalf("first UpsertIncident() error = %v", err)
	}
	if incident.EventCount != 1 || incident.ObservationCount != 1 {
		t.Fatalf("counts after first upsert = events %d observations %d, want 1/1", incident.EventCount, incident.ObservationCount)
	}

	input.ObservedAt = firstSeen.Add(5 * time.Minute)
	incident, err = store.UpsertIncident(ctx, input)
	if err != nil {
		t.Fatalf("second UpsertIncident() error = %v", err)
	}
	if incident.EventCount != 1 || incident.ObservationCount != 1 {
		t.Fatalf("counts after coalesced upsert = events %d observations %d, want 1/1", incident.EventCount, incident.ObservationCount)
	}

	input.ObservedAt = firstSeen.Add(31 * time.Minute)
	incident, err = store.UpsertIncident(ctx, input)
	if err != nil {
		t.Fatalf("quiet-window UpsertIncident() error = %v", err)
	}
	if incident.EventCount != 1 || incident.ObservationCount != 2 {
		t.Fatalf("counts after quiet-window upsert = events %d observations %d, want 1/2", incident.EventCount, incident.ObservationCount)
	}

	observations, err := store.ListIncidentObservations(ctx, incident.ID, 10)
	if err != nil {
		t.Fatalf("ListIncidentObservations() error = %v", err)
	}
	if len(observations) != 2 {
		t.Fatalf("observation rows = %d, want 2", len(observations))
	}
}

func TestSQLiteStore_IncidentReopenRules(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	ctx := context.Background()
	input := incidents.UpsertInput{
		Fingerprint:   incidents.Fingerprint("host_integrity", "host-runtime-integrity", "boot-a"),
		Type:          incidents.TypeHostIntegrity,
		Severity:      incidents.SeverityCritical,
		Title:         "Host integrity issue detected",
		Summary:       "Runtime failed",
		ObservedAt:    time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
		SourceCheckID: "host-runtime-integrity",
	}

	incident, err := store.UpsertIncident(ctx, input)
	if err != nil {
		t.Fatalf("UpsertIncident() error = %v", err)
	}
	_, err = store.UpdateIncidentStatus(ctx, incident.ID, incidents.StatusResolved, "fixed")
	if err != nil {
		t.Fatalf("UpdateIncidentStatus(resolved) error = %v", err)
	}
	input.ObservedAt = input.ObservedAt.Add(time.Hour)
	incident, err = store.UpsertIncident(ctx, input)
	if err != nil {
		t.Fatalf("UpsertIncident(after resolved) error = %v", err)
	}
	if incident.Status != incidents.StatusOpen || incident.EventCount != 2 {
		t.Fatalf("after resolved recurrence status/events = %s/%d, want open/2", incident.Status, incident.EventCount)
	}

	_, err = store.UpdateIncidentStatus(ctx, incident.ID, incidents.StatusIgnored, "intentional")
	if err != nil {
		t.Fatalf("UpdateIncidentStatus(ignored) error = %v", err)
	}
	input.ObservedAt = input.ObservedAt.Add(time.Hour)
	incident, err = store.UpsertIncident(ctx, input)
	if err != nil {
		t.Fatalf("UpsertIncident(after ignored) error = %v", err)
	}
	if incident.Status != incidents.StatusIgnored || incident.EventCount != 2 {
		t.Fatalf("after ignored recurrence status/events = %s/%d, want ignored/2", incident.Status, incident.EventCount)
	}
}

func TestSQLiteStore_HealTrackerRoundTrip_WithTextTimestamps(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)

	lastAttempt := time.Now().UTC().Add(-2 * time.Minute).Round(time.Microsecond)
	lastSuccess := time.Now().UTC().Add(-1 * time.Minute).Round(time.Microsecond)
	cooldownUntil := time.Now().UTC().Add(4 * time.Minute).Round(time.Microsecond)

	tracker := &checks.HealTracker{
		LastAttempt:         lastAttempt,
		LastSuccess:         lastSuccess,
		ConsecutiveFailures: 2,
		TotalAttempts:       7,
		TotalSuccesses:      5,
		CooldownUntil:       cooldownUntil,
	}

	if err := store.SaveHealTracker(context.Background(), "resource-postgres", tracker); err != nil {
		t.Fatalf("SaveHealTracker() error = %v", err)
	}

	loaded, err := store.GetAllHealTrackers(context.Background())
	if err != nil {
		t.Fatalf("GetAllHealTrackers() error = %v", err)
	}

	got, ok := loaded["resource-postgres"]
	if !ok {
		t.Fatal("expected tracker for resource-postgres")
	}

	if !got.LastAttempt.Equal(lastAttempt) {
		t.Fatalf("LastAttempt = %s, want %s", got.LastAttempt, lastAttempt)
	}
	if !got.LastSuccess.Equal(lastSuccess) {
		t.Fatalf("LastSuccess = %s, want %s", got.LastSuccess, lastSuccess)
	}
	if !got.CooldownUntil.Equal(cooldownUntil) {
		t.Fatalf("CooldownUntil = %s, want %s", got.CooldownUntil, cooldownUntil)
	}
	if got.ConsecutiveFailures != 2 {
		t.Fatalf("ConsecutiveFailures = %d, want 2", got.ConsecutiveFailures)
	}
	if got.TotalAttempts != 7 {
		t.Fatalf("TotalAttempts = %d, want 7", got.TotalAttempts)
	}
	if got.TotalSuccesses != 5 {
		t.Fatalf("TotalSuccesses = %d, want 5", got.TotalSuccesses)
	}
}

func TestSQLiteStore_GetAllHealTrackers_ToleratesLegacyOrInvalidRows(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)

	_, err := db.Exec(
		`
		INSERT INTO heal_trackers (
			check_id, last_attempt, last_success, consecutive_failures,
			total_attempts, total_successes, cooldown_until, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"legacy-row",
		"not-a-time",
		"",
		3,
		11,
		2,
		"also-not-a-time",
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		t.Fatalf("seed legacy row failed: %v", err)
	}

	trackers, err := store.GetAllHealTrackers(context.Background())
	if err != nil {
		t.Fatalf("GetAllHealTrackers() should tolerate invalid timestamp rows, got error: %v", err)
	}

	got, ok := trackers["legacy-row"]
	if !ok {
		t.Fatal("expected tracker for legacy-row")
	}
	if !got.LastAttempt.IsZero() {
		t.Fatalf("LastAttempt should be zero for invalid timestamp, got %s", got.LastAttempt)
	}
	if !got.LastSuccess.IsZero() {
		t.Fatalf("LastSuccess should be zero for empty timestamp, got %s", got.LastSuccess)
	}
	if !got.CooldownUntil.IsZero() {
		t.Fatalf("CooldownUntil should be zero for invalid timestamp, got %s", got.CooldownUntil)
	}
	if got.ConsecutiveFailures != 3 {
		t.Fatalf("ConsecutiveFailures = %d, want 3", got.ConsecutiveFailures)
	}
	if got.TotalAttempts != 11 {
		t.Fatalf("TotalAttempts = %d, want 11", got.TotalAttempts)
	}
	if got.TotalSuccesses != 2 {
		t.Fatalf("TotalSuccesses = %d, want 2", got.TotalSuccesses)
	}
}

func TestSQLiteStore_GetCheckTrends_WithSingleConnection(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)

	seedHealthResults(t, db, time.Now().UTC())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	trends, err := store.GetCheckTrends(ctx, 24)
	if err != nil {
		t.Fatalf("GetCheckTrends() error = %v", err)
	}
	if trends.TotalChecks != 2 {
		t.Fatalf("total checks = %d, want 2", trends.TotalChecks)
	}
	for _, trend := range trends.Trends {
		if trend.CurrentStatus == "" {
			t.Fatalf("current status for %s is empty", trend.CheckID)
		}
		if len(trend.RecentStatuses) == 0 {
			t.Fatalf("recent statuses for %s should not be empty", trend.CheckID)
		}
	}
}

func TestSQLiteStore_GetUptimeHistory_WithSingleConnection(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)

	seedHealthResults(t, db, time.Now().UTC())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	history, err := store.GetUptimeHistory(ctx, 24, 24)
	if err != nil {
		t.Fatalf("GetUptimeHistory() error = %v", err)
	}
	if len(history.Buckets) != 24 {
		t.Fatalf("bucket count = %d, want 24", len(history.Buckets))
	}
	if history.WindowHours != 24 {
		t.Fatalf("window hours = %d, want 24", history.WindowHours)
	}
	if history.BucketCount != 24 {
		t.Fatalf("history bucket count = %d, want 24", history.BucketCount)
	}
}

func openSQLiteTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	schemaBytes, err := os.ReadFile(productionSchemaPath(t))
	if err != nil {
		t.Fatalf("read production schema error = %v", err)
	}
	if _, err := db.Exec(string(schemaBytes)); err != nil {
		t.Fatalf("apply production schema error = %v", err)
	}

	return db
}

func seedHealthResults(t *testing.T, db *sql.DB, now time.Time) {
	t.Helper()

	rows := []struct {
		checkID string
		status  string
		offset  time.Duration
	}{
		{checkID: "infra-network", status: "ok", offset: -25 * time.Minute},
		{checkID: "infra-network", status: "warning", offset: -15 * time.Minute},
		{checkID: "infra-network", status: "ok", offset: -5 * time.Minute},
		{checkID: "system-memory", status: "critical", offset: -20 * time.Minute},
		{checkID: "system-memory", status: "warning", offset: -10 * time.Minute},
		{checkID: "system-memory", status: "ok", offset: -2 * time.Minute},
	}

	query := `
		INSERT INTO health_results (check_id, status, message, details, duration_ms, created_at)
		VALUES (?, ?, ?, '{}', 10, ?)
	`
	for _, row := range rows {
		if _, err := db.Exec(
			query,
			row.checkID,
			row.status,
			"seeded test data",
			now.Add(row.offset).Format(time.RFC3339Nano),
		); err != nil {
			t.Fatalf("seed insert failed: %v", err)
		}
	}
}
