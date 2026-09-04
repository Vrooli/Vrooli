package reconcile

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"compute-manager/internal/provider"
	_ "modernc.org/sqlite"
)

func TestSweepReportsBothDirectionsWithoutDestroying(t *testing.T) { // [REQ:COMPUTEM-P0-003]
	fake := &provider.Fake{}
	created, err := fake.Create(context.Background(), provider.Spec{Region: "fsn1", Size: "small"})
	if err != nil {
		t.Fatal(err)
	}
	findings, err := (Service{Provider: fake}).Sweep(context.Background(), []Local{{ProviderID: "missing", State: "running"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings = %#v, want provider-only and local-only", findings)
	}
	if _, err := fake.Describe(context.Background(), created.ID); err != nil {
		t.Fatalf("sweep destroyed provider instance: %v", err)
	}
	if fake.DestroyCalls != 0 {
		t.Fatalf("destroy calls = %d, want 0", fake.DestroyCalls)
	}
}

func TestSweepSettlesLocalOnlyUsageWithoutDestroying(t *testing.T) { // [REQ:COMPUTEM-P0-003] [REQ:COMPUTEM-P0-006]
	fake := &provider.Fake{}
	settled := false
	var settledID string
	findings, err := (Service{
		Provider: fake,
		Settle: func(_ context.Context, item Local) error {
			settled = true
			settledID = item.ReservationID
			return nil
		},
	}).Sweep(context.Background(), []Local{{InstanceID: "local-1", ProviderID: "missing", State: "running", ReservationID: "res-1", CreatedAt: time.Now().Add(-time.Minute)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Kind != LocalOnly {
		t.Fatalf("findings = %#v, want one local-only finding", findings)
	}
	if !settled || settledID != "res-1" {
		t.Fatalf("settled = %v, id = %q, want true/res-1", settled, settledID)
	}
	if fake.DestroyCalls != 0 {
		t.Fatalf("destroy calls = %d, want 0", fake.DestroyCalls)
	}
}

func TestCompareCostReportsOnlyBeyondThreshold(t *testing.T) { // [REQ:COMPUTEM-P1-003]
	observations := CompareCost(map[string]int64{"i1": 64, "i2": 30}, []provider.BillingStatement{{ProviderInstanceID: "i1", Minutes: 60}, {ProviderInstanceID: "i2", Minutes: 30}}, 2)
	if len(observations) != 2 || !observations[0].Alarm || observations[1].Alarm || observations[0].DeltaMinutes != 4 {
		t.Fatalf("observations = %#v", observations)
	}
}

type dailySource struct {
	statements []provider.BillingStatement
	from, to   time.Time
}

func (s *dailySource) BillingStatements(_ context.Context, from, to time.Time) ([]provider.BillingStatement, error) {
	s.from, s.to = from, to
	return s.statements, nil
}

func TestDailyCostRunnerDerivesTransitionsAndReportsWithoutMutation(t *testing.T) { // [REQ:COMPUTEM-P1-003]
	db, err := sql.Open("sqlite", "file:daily-cost?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE instances (provider TEXT, provider_instance_id TEXT, created_at TEXT, destroyed_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO instances(provider,provider_instance_id,created_at,destroyed_at) VALUES ('digitalocean','12','2026-09-03T00:30:00Z','2026-09-03T02:00:00Z'),('digitalocean','ignored','2026-09-02T00:00:00Z','2026-09-02T01:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	source := &dailySource{statements: []provider.BillingStatement{{Provider: "digitalocean", ProviderInstanceID: "12", Minutes: 90}}}
	var got []CostObservation
	day := time.Date(2026, 9, 3, 12, 0, 0, 0, time.FixedZone("EDT", -4*60*60))
	runner := DailyCostRunner{Source: source, DB: db, Provider: "digitalocean", Threshold: 0, Report: func(items []CostObservation) { got = items }}
	if err := runner.Run(context.Background(), day); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Alarm || got[0].MeteredMinutes != 90 {
		t.Fatalf("observations = %#v, want matching non-alarm statement", got)
	}
	if !source.from.Equal(time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)) || !source.to.Equal(time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("source range = %s to %s", source.from, source.to)
	}
}
