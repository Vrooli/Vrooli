package reconcile

import (
	"context"
	"database/sql"
	"testing"

	"compute-manager/internal/provider"
	internalreconcile "compute-manager/internal/reconcile"
	"connectrpc.com/connect"
	reconcilev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/reconcile"
	_ "modernc.org/sqlite"
)

func TestSweepRunnerPersistsBothDirectionsWithoutDestroyingProvider(t *testing.T) { // [REQ:COMPUTEM-P0-003]
	db, err := sql.Open("sqlite", "file:reconcile-runner?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE instances (id TEXT PRIMARY KEY,provider_instance_id TEXT,state TEXT,reservation_id TEXT,created_at TEXT,destroyed_at TEXT); CREATE TABLE reconcile_findings (id TEXT PRIMARY KEY,observed_at TEXT,kind TEXT,provider TEXT,provider_instance_id TEXT,instance_id TEXT,status TEXT,detail_json TEXT); INSERT INTO instances VALUES ('local-1','missing-1','running','reservation-1','2026-09-04T02:00:00Z','')`)
	if err != nil {
		t.Fatal(err)
	}
	fake := &provider.Fake{}
	providerInstance, err := fake.Create(context.Background(), provider.Spec{Region: "fsn1", Size: "small"})
	if err != nil {
		t.Fatal(err)
	}
	settled := ""
	runner := SweepRunner(db, internalreconcile.Service{Provider: fake}, func(_ context.Context, reservationID string, _ float64) error {
		settled = reservationID
		return nil
	})
	findings, err := runner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 || settled != "reservation-1" {
		t.Fatalf("findings=%#v settled=%q; want both directions and settlement", findings, settled)
	}
	if fake.DestroyCalls != 0 {
		t.Fatalf("destroy calls = %d, want 0", fake.DestroyCalls)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reconcile_findings`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("stored findings = %d, want 2", count)
	}
	if _, err := fake.Describe(context.Background(), providerInstance.ID); err != nil {
		t.Fatalf("provider-only instance was destroyed: %v", err)
	}
}

func TestQuarantineFindingDestroysOnlyProviderOrphan(t *testing.T) { // [REQ:COMPUTEM-P0-003]
	db, err := sql.Open("sqlite", "file:reconcile-quarantine?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE reconcile_findings (id TEXT PRIMARY KEY,observed_at TEXT,kind TEXT,provider TEXT,provider_instance_id TEXT,instance_id TEXT,status TEXT,detail_json TEXT)`); err != nil {
		t.Fatal(err)
	}
	fake := &provider.Fake{}
	created, err := fake.Create(context.Background(), provider.Spec{Region: "fsn1", Size: "small"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO reconcile_findings VALUES ('f1','2026-09-04T03:00:00Z','unaccounted_at_provider','fake',?,'','open','{}')`, created.ID); err != nil {
		t.Fatal(err)
	}
	service := &service{db: db, destroy: func(ctx context.Context, providerName, instanceID string) error {
		if providerName != "fake" {
			t.Fatalf("provider = %q, want fake", providerName)
		}
		return fake.Destroy(ctx, instanceID)
	}}
	response, err := service.QuarantineFinding(context.Background(), connect.NewRequest(&reconcilev1.QuarantineFindingRequest{Id: "f1"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetFinding().GetStatus() != "quarantined" || fake.DestroyCalls != 1 {
		t.Fatalf("response=%#v destroy calls=%d", response.Msg.GetFinding(), fake.DestroyCalls)
	}
	if _, err := fake.Describe(context.Background(), created.ID); err == nil {
		t.Fatal("quarantined provider orphan still exists")
	}
}
