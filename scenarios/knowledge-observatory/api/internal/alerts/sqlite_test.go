package alerts_test

import (
	"context"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/alerts"
	"knowledge-observatory/internal/dbtest"
)

func newRepo(t *testing.T) *alerts.SQLite {
	t.Helper()
	return alerts.NewSQLite(dbtest.New(t, apidb.SchemaProviderFunc(alerts.Schema)))
}

func ptr(v float64) *float64 { return &v }

func TestAlertRoundTripCoversEveryColumn(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	in := alerts.Alert{
		Level:          "warning",
		CollectionName: "vrooli_knowledge",
		MetricName:     "freshness",
		ThresholdValue: ptr(0.6),
		ActualValue:    ptr(0.41),
		Message:        "freshness fell below threshold",
	}
	id, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, ok, err := repo.Get(ctx, id)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.Level != in.Level || got.CollectionName != in.CollectionName || got.MetricName != in.MetricName {
		t.Errorf("identity = %+v", got)
	}
	if got.ThresholdValue == nil || *got.ThresholdValue != 0.6 {
		t.Errorf("threshold_value = %v, want 0.6", got.ThresholdValue)
	}
	if got.ActualValue == nil || *got.ActualValue != 0.41 {
		t.Errorf("actual_value = %v, want 0.41", got.ActualValue)
	}
	if got.Message != in.Message {
		t.Errorf("message = %q", got.Message)
	}
	if got.Acknowledged {
		t.Error("acknowledged should default to false")
	}
	if got.AcknowledgedAt != nil {
		t.Errorf("acknowledged_at = %v, want nil", got.AcknowledgedAt)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at was not defaulted")
	}
}

func TestInvalidLevelIsRejected(t *testing.T) {
	repo := newRepo(t)
	if _, err := repo.Insert(context.Background(), alerts.Alert{Level: "fatal"}); err == nil {
		t.Fatal("expected the CHECK constraint to reject an unknown level")
	}
}

// TestAcknowledgeRemovesFromTheUnacknowledgedIndex exercises the partial index,
// whose Postgres predicate `WHERE NOT acknowledged` had to become
// `WHERE acknowledged = 0` because SQLite stores booleans as integers.
func TestAcknowledgeRemovesFromTheUnacknowledgedIndex(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	id, err := repo.Insert(ctx, alerts.Alert{Level: "critical", Message: "down"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Insert(ctx, alerts.Alert{Level: "info", Message: "fyi"}); err != nil {
		t.Fatal(err)
	}

	open, err := repo.ListUnacknowledged(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(open) != 2 {
		t.Fatalf("got %d unacknowledged, want 2", len(open))
	}

	at := time.Date(2026, 6, 7, 8, 9, 10, 0, time.UTC)
	if err := repo.Acknowledge(ctx, id, at); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}

	open, err = repo.ListUnacknowledged(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 {
		t.Errorf("got %d unacknowledged after ack, want 1", len(open))
	}

	got, _, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Acknowledged {
		t.Error("acknowledged was not set")
	}
	if got.AcknowledgedAt == nil || !got.AcknowledgedAt.Equal(at) {
		t.Errorf("acknowledged_at = %v, want %v", got.AcknowledgedAt, at)
	}
}
