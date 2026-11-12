//go:build linux

package resources

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
)

func TestManagedUpdateCurrentHostCandidateOwnerProbe(t *testing.T) {
	if os.Getenv("VROOLI_MANAGED_UPDATE_PRODUCTION_PROBE") != "1" {
		t.Skip("set VROOLI_MANAGED_UPDATE_PRODUCTION_PROBE=1 for the isolated candidate owner drill")
	}
	root, err := os.MkdirTemp("", "vrooli-candidate-owner-probe-")
	if err != nil {
		t.Fatal(err)
	}
	isolateNativeUpdateFixtureStorage(t, root)
	u := updateTestInitial(t)
	u.ID = "candidate-owner-probe-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	u.RequestID = u.ID
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = u.CreatedAt
	m := u.Target.Manifest
	m.ManagedService.CredentialFiles = nil
	m.ManagedService.Environment = map[string]string{"UPDATE_NATIVE_MARKER": filepath.Join(root, "executed")}
	m.ManagedService.Arguments = []string{"-test.run=^TestManagedUpdateNativeResourcePayload$", "-test.timeout=${RESOURCE_PORT_HTTP}s"}
	image, err := os.Open(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	digest, err := managedUpdateImageDigest(context.Background(), image)
	image.Close()
	if err != nil {
		t.Fatal(err)
	}
	m.ManagedService.Artifact.SHA256 = digest
	u.Target, err = newManagedUpdateContract(m, ManagedServiceArtifact{Resource: u.Resource, Version: m.ManagedService.Artifact.Version, Path: os.Args[0], SHA256: digest})
	if err != nil {
		t.Fatal(err)
	}
	s, err := newManagedUpdateStore(u.Resource)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	if _, _, err := s.Create(ctx, u, false); err != nil {
		t.Fatal(err)
	}
	cleaned := false
	t.Cleanup(func() {
		if cleaned {
			os.RemoveAll(root)
			return
		}
		current, err := s.Lookup(u.RequestID)
		if err == nil && current.Candidate.Intent != nil {
			cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
			defer stop()
			i := *current.Candidate.Intent
			fences, e := newManagedUpdateExecStore(u.Resource, u.ID, i.InstanceID)
			if e == nil {
				_, e = fences.Revoke(cleanup, managedUpdateExecFence{SchemaVersion: 1, Resource: u.Resource, OperationID: u.ID, Intent: i, State: "prepared"})
			}
			if e == nil {
				ref, lookup := platform.LookupOwnedService(cleanup, i.NativeIntent)
				e = lookup
				if e == nil {
					_, e = platform.StopOwnedService(cleanup, ref)
					if e == nil {
						e = platform.ReleaseOwnedService(cleanup, ref)
					}
				}
			}
			if e != nil {
				t.Errorf("candidate cleanup unconfirmed; operation retained at %s: %v", s.path, e)
			}
		}
	})
	launcher, err := exec.LookPath("vrooli")
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	run := func() ManagedServiceUpdate {
		cmd := exec.CommandContext(ctx, launcher, "managed-update-candidate", u.Resource, u.RequestID)
		cmd.Dir = projectRoot(t)
		cmd.Env = []string{"HOME=" + home, "PATH=/usr/local/bin:/usr/bin:/bin", "XDG_STATE_HOME=" + os.Getenv("XDG_STATE_HOME")}
		output, err := cmd.Output()
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				t.Fatalf("candidate owner: %v: %s", err, exit.Stderr)
			}
			t.Fatal(err)
		}
		var result ManagedServiceUpdate
		if err := json.Unmarshal(output, &result); err != nil {
			t.Fatalf("candidate receipt: %v", err)
		}
		return result
	}
	first := run()
	if first.Phase != UpdateCandidateReady || !freshUpdateProof(first.Candidate, first.Target, time.Now()) || first.Candidate.NativeRef == nil {
		t.Fatal("candidate owner did not durably establish exact native readiness")
	}
	reloaded, err := s.Lookup(u.RequestID)
	if err != nil || reloaded.Phase != UpdateCandidateReady || reloaded.Candidate.Intent == nil {
		t.Fatal("candidate readiness was not durable")
	}
	second := run()
	if *first.Candidate.Intent != *second.Candidate.Intent || *first.Candidate.NativeRef != *second.Candidate.NativeRef || second.Revision != first.Revision {
		t.Fatal("candidate resumption dispatched another identity or invented a durable transition")
	}
	ref := *second.Candidate.NativeRef
	fences, err := newManagedUpdateExecStore(u.Resource, u.ID, second.Candidate.Intent.InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	f, err := fences.read()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fences.Revoke(ctx, f); err != nil {
		t.Fatal(err)
	}
	if receipt, err := platform.StopOwnedService(ctx, ref); err != nil || !receipt.NativeQuiescent {
		t.Fatalf("candidate native quiescence failed: %v", err)
	}
	if err := platform.ReleaseOwnedService(ctx, ref); err != nil {
		t.Fatal(err)
	}
	cleaned = true
	checkCtx, stopCheck := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer stopCheck()
	if _, err := NewController(projectRoot(t), home).AdvanceManagedUpdateCandidate(checkCtx, u.Resource, u.RequestID); err == nil {
		t.Fatal("stopped candidate was reported currently ready")
	}
	retained, err := fences.read()
	if err != nil || retained.State != "revoked" {
		t.Fatal("candidate resumption reset its permanent exec fence")
	}
	report, _ := json.Marshal(map[string]any{"phase": first.Phase, "revision": first.Revision, "native_ref": ref, "same_identity_on_resume": true, "stopped_candidate_not_redispatched": true, "cleanup_confirmed": true, "scope": "isolated fixture; synthetic original serving record; no primary stop path exercised"})
	t.Log(string(report))
}
