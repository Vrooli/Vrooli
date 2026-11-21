package runtimesupervisor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestSupervisorInspectionReportsOwnerBuildInsteadOfInspectorBuild(t *testing.T) {
	for _, mode := range []string{"owner", "historical", "recycled", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "runtime.db")
			store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: path})
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			pid := os.Getpid()
			stamp, err := process.Incarnation(pid)
			if err != nil {
				t.Skipf("native incarnation unavailable: %v", err)
			}
			metadata, _ := json.Marshal(supervisorOwnerIdentity{BuildIdentity: "owner-build", ProcessIncarnation: stamp})
			if mode == "historical" {
				metadata = nil
			}
			if mode == "malformed" {
				metadata = []byte("{")
			}
			if mode == "recycled" {
				metadata, _ = json.Marshal(supervisorOwnerIdentity{BuildIdentity: "owner-build", ProcessIncarnation: "foreign:123"})
			}
			_, err = store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{SupervisorID: "owner", HostBootID: "boot", HostSessionID: "session", PID: &pid, MetadataJSON: string(metadata)}, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			inspector := New(Config{DBPath: path, BuildIdentity: "inspector-build", HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot", SessionID: "session"}}})
			defer inspector.Close()
			report, err := inspector.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			want := "owner-build"
			if mode == "historical" || mode == "malformed" {
				want = scenarioruntime.HealthStatusUnknown
			}
			if report.BuildIdentity != want {
				t.Fatalf("reported inspector as owner: got %q, want %q", report.BuildIdentity, want)
			}
			if mode == "recycled" && (report.Status != StatusStale || !strings.Contains(report.StatusReason, "incarnation")) {
				t.Fatalf("recycled live PID treated as owner: %+v", report)
			}
			if mode == "malformed" && report.Status != StatusStale {
				t.Fatalf("malformed owner treated as current: %+v", report)
			}
		})
	}
}
