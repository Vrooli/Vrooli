package meter

import (
	"context"
	"database/sql"
	"testing"

	"connectrpc.com/connect"
	meterv1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/meter"
	_ "modernc.org/sqlite"
)

func TestCeilingReportsConfiguredTenantLimit(t *testing.T) { // [REQ:COMPUTEM-P0-006]
	db, err := sql.Open("sqlite", "file:meter-handler-ceiling?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE usage_records (instance_id TEXT,tenant TEXT,quantity INTEGER,started_at TEXT,ended_at TEXT);
CREATE TABLE instance_intents (id TEXT PRIMARY KEY,requested_by TEXT);
CREATE TABLE instances (id TEXT PRIMARY KEY,tenant TEXT);
CREATE TABLE reservations (id TEXT PRIMARY KEY,instance_id TEXT,intent_id TEXT,state TEXT,quantity INTEGER);
INSERT INTO instances VALUES ('i1','tenant-a');
INSERT INTO reservations VALUES ('res-1','i1','','held',12)`); err != nil {
		t.Fatal(err)
	}
	s := &service{db: db, limit: 90}
	response, err := s.Ceiling(context.Background(), connect.NewRequest(&meterv1.CeilingRequest{Tenant: "tenant-a"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetUsed() != 12 || response.Msg.GetLimit() != 90 {
		t.Fatalf("ceiling = used %d, limit %d; want 12, 90", response.Msg.GetUsed(), response.Msg.GetLimit())
	}
}
