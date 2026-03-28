package queue

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
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
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		repo := NewPostgresSuiteRequestRepository(db)
		id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mock.ExpectExec("UPDATE suite_requests").
			WithArgs(StatusRunning, id).
			WillReturnResult(sqlmock.NewResult(0, 1))

		if err := repo.UpdateStatus(context.Background(), id, StatusRunning); err != nil {
			t.Fatalf("expected update to succeed: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestSuiteRequestRepositoryStatusSnapshot(t *testing.T) {
	t.Run("[REQ:TESTGENIE-SUITE-P0] repository summarizes queue snapshot", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		repo := NewPostgresSuiteRequestRepository(db)
		now := time.Now().UTC()
		repo.clock = func() time.Time { return now }
		repo.activeWindow = 24 * time.Hour

		mock.ExpectQuery("SELECT\\s+COUNT\\(\\*\\) AS total").
			WithArgs(now.Add(-24*time.Hour), StatusQueued, StatusDelegated, StatusRunning, StatusCompleted, StatusFailed).
			WillReturnRows(sqlmock.NewRows([]string{
				"total",
				"queued",
				"delegated",
				"running",
				"completed",
				"failed",
				"stale",
				"oldest_queued_at",
			}).AddRow(6, 2, 1, 1, 1, 0, 1, now.Add(-2*time.Minute)))

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

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}
