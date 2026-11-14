package lifecycle

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestRuntimeRegistryFailureCleanupSurvivesExpiredParentContext(t *testing.T) {
	home := t.TempDir()
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()

	ownerPID := 12345
	instance, err := store.CreateLease(context.Background(), scenarioruntime.Instance{
		Scenario:      "alpha",
		BuildIdentity: "test-build",
		Status:        scenarioruntime.StatusStarting,
		OwnerPID:      &ownerPID,
		HostBootID:    "test-boot",
	}, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	claim, err := store.AcquirePortClaim(context.Background(), scenarioruntime.PortClaim{
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "api",
		EnvVar:     "API_PORT",
		Port:       41001,
		Status:     scenarioruntime.ClaimStatusReserved,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}

	parent, cancel := context.WithCancel(context.Background())
	cancel()
	session := &runtimeRegistrySession{enabled: true, store: store, instance: instance}
	if err := session.fail(parent, errors.New("dependency start deadline exceeded")); err != nil {
		t.Fatalf("fail cleanup with expired parent: %v", err)
	}

	gotInstance, err := store.GetInstance(context.Background(), instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if gotInstance.Status != scenarioruntime.StatusFailed {
		t.Fatalf("instance status = %q, want failed", gotInstance.Status)
	}
	claims, err := store.ListPortClaims(context.Background(), scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(claims) != 1 || claims[0].ClaimID != claim.ClaimID || claims[0].Status != scenarioruntime.ClaimStatusReleased {
		t.Fatalf("claims = %#v, want the owned reservation released", claims)
	}

	dbPath, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		t.Fatalf("DefaultDBPath: %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open runtime registry for event assertion: %v", err)
	}
	defer db.Close()
	var failureEvents int
	if err := db.QueryRow(`SELECT count(*) FROM runtime_events WHERE instance_id = ? AND event_type = 'start_failed'`, instance.InstanceID).Scan(&failureEvents); err != nil {
		t.Fatalf("count start_failed events: %v", err)
	}
	if failureEvents != 1 {
		t.Fatalf("start_failed events = %d, want one", failureEvents)
	}
}
