package queue

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"test-genie/internal/storage/sqliteutil"
	"test-genie/internal/testsqlite"
)

func TestBuildSuiteRequestDefaults(t *testing.T) {
	t.Run("[REQ:TESTGENIE-SUITE-P0] queue builder fills defaults", func(t *testing.T) {
		req, err := buildSuiteRequest(QueueSuiteRequestInput{
			ScenarioName: "document-manager",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if req.ScenarioName != "document-manager" {
			t.Fatalf("unexpected scenario name: %s", req.ScenarioName)
		}

		if req.CoverageTarget != 95 {
			t.Fatalf("expected coverage default to 95, got %d", req.CoverageTarget)
		}

		expectedTypes := []string{"unit", "integration"}
		if !slices.Equal(req.RequestedTypes, expectedTypes) {
			t.Fatalf("expected default types %v, got %v", expectedTypes, req.RequestedTypes)
		}

		if req.Priority != PriorityNormal {
			t.Fatalf("expected default priority %s, got %s", PriorityNormal, req.Priority)
		}

		if req.EstimatedQueueTime == 0 {
			t.Fatal("expected estimated queue time to be set")
		}
	})
}

func TestBuildSuiteRequestInvalidType(t *testing.T) {
	t.Run("[REQ:TESTGENIE-SUITE-P0] queue builder rejects unknown types", func(t *testing.T) {
		_, err := buildSuiteRequest(QueueSuiteRequestInput{
			ScenarioName:   "document-manager",
			RequestedTypes: stringList{"invalid"},
		})
		if err == nil {
			t.Fatal("expected validation error")
		}
	})
}

func TestSuiteRequestRepositoryUpdateStatus(t *testing.T) {
	t.Run("[REQ:TESTGENIE-SUITE-P0] repository updates status transitions", func(t *testing.T) {
		db := testsqlite.Open(t)
		repo := NewSQLiteSuiteRequestRepository(db)
		id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		_, err := db.Exec(`
INSERT INTO suite_requests (
	id, scenario_name, requested_types, coverage_target, priority, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
			id.String(),
			"demo",
			`["unit"]`,
			95,
			PriorityNormal,
			StatusQueued,
			sqliteutil.FormatTimestamp(time.Now().UTC().Add(-time.Minute)),
			sqliteutil.FormatTimestamp(time.Now().UTC().Add(-time.Minute)),
		)
		if err != nil {
			t.Fatalf("seed request: %v", err)
		}

		if err := repo.UpdateStatus(context.Background(), id, StatusRunning); err != nil {
			t.Fatalf("expected update to succeed: %v", err)
		}

		req, err := repo.GetByID(context.Background(), id)
		if err != nil {
			t.Fatalf("expected request to be readable: %v", err)
		}
		if req.Status != StatusRunning {
			t.Fatalf("expected status %s, got %s", StatusRunning, req.Status)
		}
	})
}

func TestSuiteRequestRepositoryStatusSnapshot(t *testing.T) {
	t.Run("[REQ:TESTGENIE-SUITE-P0] repository summarizes queue snapshot", func(t *testing.T) {
		db := testsqlite.Open(t)
		repo := NewSQLiteSuiteRequestRepository(db)
		now := time.Now().UTC()
		repo.clock = func() time.Time { return now }
		repo.activeWindow = 24 * time.Hour

		rows := []struct {
			id        string
			status    string
			updatedAt time.Time
			createdAt time.Time
		}{
			{"1", StatusQueued, now.Add(-2 * time.Minute), now.Add(-10 * time.Minute)},
			{"2", StatusQueued, now.Add(-3 * time.Minute), now.Add(-11 * time.Minute)},
			{"3", StatusDelegated, now.Add(-4 * time.Minute), now.Add(-12 * time.Minute)},
			{"4", StatusRunning, now.Add(-5 * time.Minute), now.Add(-13 * time.Minute)},
			{"5", StatusCompleted, now.Add(-6 * time.Minute), now.Add(-14 * time.Minute)},
			{"6", StatusQueued, now.Add(-25 * time.Hour), now.Add(-26 * time.Hour)},
		}
		for _, row := range rows {
			if _, err := db.Exec(`
INSERT INTO suite_requests (
	id, scenario_name, requested_types, coverage_target, priority, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
				row.id,
				"demo",
				`["unit"]`,
				95,
				PriorityNormal,
				row.status,
				sqliteutil.FormatTimestamp(row.createdAt),
				sqliteutil.FormatTimestamp(row.updatedAt),
			); err != nil {
				t.Fatalf("seed request %s: %v", row.id, err)
			}
		}

		snapshot, err := repo.StatusSnapshot(context.Background())
		if err != nil {
			t.Fatalf("expected snapshot to succeed: %v", err)
		}
		if snapshot.Total != 6 || snapshot.Queued != 2 || snapshot.Delegated != 1 || snapshot.Running != 1 || snapshot.Completed != 1 || snapshot.Stale != 1 {
			t.Fatalf("unexpected snapshot counts: %#v", snapshot)
		}
		if snapshot.OldestQueuedAt == nil {
			t.Fatal("expected oldest queued timestamp to be populated")
		}
		expectedOldest := now.Add(-12 * time.Minute)
		if !snapshot.OldestQueuedAt.Equal(expectedOldest) {
			t.Fatalf("expected oldest queued at %s, got %s", expectedOldest, snapshot.OldestQueuedAt)
		}
	})
}
