package sessionlease

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// The lease a launch writes carries the owner the launcher read from its own
// process, so `vrooli agent list` can judge the session without inferring.
func TestCreateRecordsTheLaunchOwner(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	open := func(ctx context.Context) (*scenarioruntime.SQLiteStore, error) {
		return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath})
	}
	recorder := Recorder{Open: open, Owner: func() scenarioruntime.EditorLeaseOwner {
		return scenarioruntime.EditorLeaseOwner{Kind: "desktop-app", Ref: "/home/op/.config/web-console-desktop/runtime", EvidenceJSON: `{"app_pid":80}`}
	}}
	if err := recorder.Create(ctx, cliutil.EditorLeaseRecord{SessionID: "s1", Harness: "opencode", Agent: "opencode", PID: 4242, Scope: "vrooli-agent-opencode-1"}); err != nil {
		t.Fatal(err)
	}
	store, err := open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	leases, err := store.ListEditorLeases(ctx, false)
	if err != nil || len(leases) != 1 {
		t.Fatalf("leases = %+v, %v", leases, err)
	}
	if owner := leases[0].Owner; owner.Kind != "desktop-app" || owner.Ref != "/home/op/.config/web-console-desktop/runtime" || owner.Source != scenarioruntime.EditorLeaseOwnerLaunch {
		t.Fatalf("owner = %+v, want the launch-recorded desktop app", owner)
	}
}
