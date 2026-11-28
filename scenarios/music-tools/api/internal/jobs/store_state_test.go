package jobs

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestStoreRejectsUnknownJobStateOnWriteAndRead(t *testing.T) {
	ctx := context.Background()
	st := newStore(newJobsDB(t))
	invalid := Job{ID: "invalid", State: State("corrupt"), CreatedAt: time.Now()}
	if err := st.insert(ctx, invalid); err == nil || !strings.Contains(err.Error(), "unknown state") {
		t.Fatalf("insert invalid state error=%v", err)
	}
	if err := st.update(ctx, invalid); err == nil || !strings.Contains(err.Error(), "unknown state") {
		t.Fatalf("update invalid state error=%v", err)
	}

	if _, err := st.db.ExecContext(ctx,
		"INSERT INTO jobs (id, operation, lane, state, created_at) VALUES (?, ?, ?, ?, ?)",
		"historical-invalid", "compose", string(LaneCPU), "corrupt", time.Now().Format(jobTimeFormat),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := st.get(ctx, "historical-invalid"); err == nil || !strings.Contains(err.Error(), "unknown state") {
		t.Fatalf("read invalid state error=%v", err)
	}
}
