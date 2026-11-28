package bridgeadmin

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSQLStoreBeginKeepsCallerRequestIdentityAndRejectsConflictingReplay(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := NewSQLStore(db)
	now := time.Date(2026, 11, 20, 1, 0, 0, 0, time.UTC)
	receipt := Receipt{Key: "caller-hash", RequestID: "request-1", Caller: "admin@example.test", Scenario: "audio-tools", Operation: "audio render", NodeID: "node-vps", Tier: "vps", Status: "requested", CreatedAt: now}
	insert := regexp.QuoteMeta("INSERT INTO bridge_admin_dispatch_receipts(request_id,external_request_id,caller,scenario,operation,node_id,target_tier,status,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,'requested',$8,$8) ON CONFLICT(request_id) DO NOTHING")
	selectOne := regexp.QuoteMeta("SELECT request_id,external_request_id,caller,scenario,operation,node_id,target_tier,status,run_id,replayed,detail,created_at FROM bridge_admin_dispatch_receipts WHERE request_id=$1")

	mock.ExpectExec(insert).WithArgs(receipt.Key, receipt.RequestID, receipt.Caller, receipt.Scenario, receipt.Operation, receipt.NodeID, receipt.Tier, receipt.CreatedAt).WillReturnResult(sqlmock.NewResult(1, 1))
	created, replay, err := store.Begin(context.Background(), receipt)
	if err != nil || replay || created.Status != "requested" {
		t.Fatalf("first Begin=(%+v,%v,%v)", created, replay, err)
	}

	mock.ExpectExec(insert).WithArgs(receipt.Key, receipt.RequestID, receipt.Caller, receipt.Scenario, receipt.Operation, receipt.NodeID, receipt.Tier, receipt.CreatedAt).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(selectOne).WithArgs(receipt.Key).WillReturnRows(sqlmock.NewRows([]string{"request_id", "external_request_id", "caller", "scenario", "operation", "node_id", "target_tier", "status", "run_id", "replayed", "detail", "created_at"}).AddRow(receipt.Key, receipt.RequestID, receipt.Caller, receipt.Scenario, receipt.Operation, receipt.NodeID, receipt.Tier, "uncertain", "", false, "timeout", now))
	prior, replay, err := store.Begin(context.Background(), receipt)
	if err != nil || !replay || prior.Status != "uncertain" {
		t.Fatalf("replay Begin=(%+v,%v,%v)", prior, replay, err)
	}

	conflict := receipt
	conflict.Scenario = "scenario-to-cloud"
	mock.ExpectExec(insert).WithArgs(conflict.Key, conflict.RequestID, conflict.Caller, conflict.Scenario, conflict.Operation, conflict.NodeID, conflict.Tier, conflict.CreatedAt).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(selectOne).WithArgs(conflict.Key).WillReturnRows(sqlmock.NewRows([]string{"request_id", "external_request_id", "caller", "scenario", "operation", "node_id", "target_tier", "status", "run_id", "replayed", "detail", "created_at"}).AddRow(receipt.Key, receipt.RequestID, receipt.Caller, receipt.Scenario, receipt.Operation, receipt.NodeID, receipt.Tier, "uncertain", "", false, "timeout", now))
	if _, _, err = store.Begin(context.Background(), conflict); err == nil {
		t.Fatalf("conflicting scenario replay err=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
