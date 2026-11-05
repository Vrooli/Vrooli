package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git-control-tower/internal/policygate"
	"git-control-tower/internal/pushsafety"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/cli-core/cliutil"
)

// [REQ:GCT-OT-P0-006] No transfer occurs for incomplete or blocked scans.
func TestPushSafetyStopsEveryPushEntryPoint(t *testing.T) {
	for _, state := range []string{"blocked", "unknown"} {
		for _, entry := range []string{"push", "upstream", "publish"} {
			t.Run(state+"/"+entry, func(t *testing.T) {
				f := NewFakeGitRunner()
				f.Branch.Head = "agi"
				f.Branch.OID = "local123"
				f.Branch.Upstream = "origin/agi"
				f.SafetyReport = &pushsafety.Report{State: state, Complete: state == "blocked", Reason: "oversized or incomplete"}
				switch entry {
				case "push":
					r, e := PushToRemote(authorizedHumanContext(), PushPullDeps{Git: f, RepoDir: f.RepoRoot}, PushRequest{})
					if e != nil || r.Success {
						t.Fatalf("%+v %v", r, e)
					}
				case "upstream":
					r, e := RunUpstreamAction(authorizedHumanContext(), PushPullDeps{Git: f, RepoDir: f.RepoRoot}, UpstreamActionRequest{Action: "push_set_upstream", Branch: "agi"})
					if e != nil || r.Success {
						t.Fatalf("%+v %v", r, e)
					}
				case "publish":
					r, e := PublishBranch(authorizedHumanContext(), BranchDeps{Git: f, RepoDir: f.RepoRoot}, PublishBranchRequest{Branch: "agi"})
					if e != nil || r.Success {
						t.Fatalf("%+v %v", r, e)
					}
				}
				if f.PushCount != 0 {
					t.Fatal("unsafe transfer invoked")
				}
			})
		}
	}
}

func TestRecoveryStorageRequiresLeasedTestRoot(t *testing.T) {
	ctx := database.WithTestMode(context.Background())
	server := &Server{}
	if _, e := server.pushRecoveryRoot(ctx); e == nil {
		t.Fatal("test-mode recovery without roots accepted")
	}
	server.fileRoots = filerouting.New(storage.Paths{DataDir: "/production/data"})
	if _, e := server.pushRecoveryRoot(ctx); e == nil {
		t.Fatal("unleased test-mode recovery fell back to production")
	}
	root := t.TempDir()
	if e := server.fileRoots.InstallTestRoots(storage.Paths{DataDir: root}, "lease", time.Minute); e != nil {
		t.Fatal(e)
	}
	got, e := server.pushRecoveryRoot(ctx)
	if e != nil || got != filepath.Join(root, "push-recovery") {
		t.Fatalf("wrong artifact root: %q %v", got, e)
	}
}
func TestRecoveryRejectsUnapprovedAndStaleBeforeWriter(t *testing.T) {
	f := NewFakeGitRunner()
	f.SafetyReport = &pushsafety.Report{Complete: true, State: "blocked", CanPrepare: true, Fingerprint: "new"}
	d := PushPullDeps{Git: f, RepoDir: f.RepoRoot}
	agent := policygate.WithPrincipal(context.Background(), policygate.Principal{Kind: cliutil.CallerKindVrooliAgent, Subject: "agent", Verified: true})
	if _, e := PreparePushRecovery(agent, d, "origin", "agi", "new", nil); e == nil {
		t.Fatal("agent recovery accepted")
	}
	if _, e := PreparePushRecovery(context.Background(), d, "origin", "agi", "new", nil); e == nil {
		t.Fatal("anonymous recovery accepted")
	}
	if _, e := PreparePushRecovery(authorizedHumanContext(), d, "origin", "agi", "old", nil); e == nil {
		t.Fatal("stale recovery accepted")
	}
	if f.RecoveryCalls != 0 {
		t.Fatal("recovery writer called before preconditions")
	}
	f.RecoveryArtifact = pushsafety.Artifact{State: "prepared"}
	a, e := PreparePushRecovery(authorizedHumanContext(), d, "origin", "agi", "new", nil)
	if e != nil || a.State != "prepared" || f.RecoveryCalls != 1 {
		t.Fatalf("approved candidate not returned: %+v %v", a, e)
	}
}
