package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"workspace-sandbox/internal/types"
)

func TestTurnCheckpoint_RecoversDeletedArchiveWithoutReapplying(t *testing.T) {
	for _, scenario := range []string{"matching", "partial-prior", "changed", "wrong-run", "missing-blob", "symlink"} {
		t.Run(scenario, func(t *testing.T) {
			env := newArchiveTestEnv(t)
			ctx := context.Background()
			sb := env.makeSandbox(types.StatusActive, map[string]string{"recover.txt": "retained turn\n"}, nil)
			if err := os.MkdirAll(sb.ScopePath, 0o755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(sb.ScopePath, "recover.txt")
			if err := os.WriteFile(path, []byte("retained turn\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := env.svc.Delete(ctx, sb.ID); err != nil {
				t.Fatal(err)
			}
			req := &types.TurnCheckpointRequest{SandboxID: sb.ID, AgentManagerRunID: "run-archive-test", Actor: "recovery", RunOutcome: "success", Source: types.SourceAgentManagerAutoApply}
			switch scenario {
			case "partial-prior":
				prior := &types.AppliedChange{SandboxID: sb.ID, FilePath: path, ProjectRoot: sb.ProjectRoot, ChangeType: "added", FileSize: 14, AgentManagerRunID: req.AgentManagerRunID, ContentDigest: "sha256:" + hashContent([]byte("retained turn\n")), EvidenceRevision: "prior-checkpoint", ProvenanceState: string(types.ProvenanceFileStateApplied)}
				if err := env.repo.RecordAppliedChanges(ctx, []*types.AppliedChange{prior}); err != nil {
					t.Fatal(err)
				}
			case "changed":
				if err := os.WriteFile(path, []byte("someone else's later change\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			case "wrong-run":
				req.AgentManagerRunID = "different-run"
			case "missing-blob":
				if err := env.blobs.DeleteSandbox(ctx, sb.ID.String()); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				target := filepath.Join(env.tmp, "outside-scope.txt")
				if err := os.WriteFile(target, []byte("retained turn\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				result, err := env.svc.TurnCheckpoint(ctx, req)
				if scenario == "matching" || scenario == "partial-prior" {
					if err != nil || !result.Success || result.Applied != 1 {
						t.Fatalf("attempt %d: result=%+v err=%v", i, result, err)
					}
				} else if err == nil {
					t.Fatalf("invalid evidence %q accepted: %+v", scenario, result)
				}
			}
			changes, err := env.repo.GetFileProvenance(ctx, path, sb.ProjectRoot, 100)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "matching" || scenario == "partial-prior" {
				if len(changes) != 1 || changes[0].AgentManagerRunID != "run-archive-test" || changes[0].ContentDigest != "sha256:"+hashContent(before) || changes[0].EvidenceRevision == "" {
					t.Fatalf("missing or duplicate original-run provenance: %+v", changes)
				}
			} else if len(changes) != 0 {
				t.Fatalf("invalid evidence wrote provenance: %+v", changes)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != string(before) {
				t.Fatalf("recovery changed source: %q %v", after, err)
			}
			stored, err := env.repo.Get(ctx, sb.ID)
			if err != nil || stored.Status != types.StatusDeleted {
				t.Fatalf("recovery resurrected sandbox: %+v %v", stored, err)
			}
		})
	}
}
