//go:build linux

package resources

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	platform "github.com/vrooli/platform-go"
)

func TestManagedUpdateCurrentHostOriginalStopGuardProbe(t *testing.T) {
	if os.Getenv("VROOLI_MANAGED_UPDATE_ORIGINAL_STOP_PROBE") != "1" {
		t.Skip("set VROOLI_MANAGED_UPDATE_ORIGINAL_STOP_PROBE=1 for isolated original-stop/current-candidate guard qualification")
	}
	root, err := os.MkdirTemp("", "vrooli-original-stop-guard-probe-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "xdg-state"))
	u := updateTestInitial(t)
	name := "original-stop-probe-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	u.ID = name
	u.RequestID = name
	u.Resource = name
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = u.CreatedAt
	bytes, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	oldImage := filepath.Join(root, "old-image")
	if err := os.WriteFile(oldImage, append(bytes, 0), 0o755); err != nil {
		t.Fatal(err)
	}
	oldDigest, err := binaryfetch.FileDigestContext(context.Background(), oldImage)
	if err != nil {
		t.Fatal(err)
	}
	newDigest, err := binaryfetch.FileDigestContext(context.Background(), os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := reservation.Addr().(*net.TCPAddr).Port
	reservation.Close()
	m := u.Target.Manifest
	m.Name = name
	m.Ports[0].Host = port
	m.ManagedService.CredentialFiles = nil
	m.ManagedService.Artifact.Version = "old-fixture"
	m.ManagedService.Artifact.SHA256ByPlatform = nil
	m.ManagedService.Artifact.SHA256 = oldDigest
	m.ManagedService.Environment = map[string]string{"UPDATE_NATIVE_MARKER": filepath.Join(root, "old-executed")}
	m.ManagedService.Arguments = []string{"-test.run=^TestManagedUpdateNativeResourcePayload$", "-test.timeout=${RESOURCE_PORT_HTTP}s"}
	u.Rollback, err = newManagedUpdateContract(m, ManagedServiceArtifact{Resource: name, Path: oldImage, Version: "old-fixture", SHA256: oldDigest})
	if err != nil {
		t.Fatal(err)
	}
	sup, _, err := managedServiceSupervisorFor(name)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	state, err := sup.StartContext(ctx, oldImage, m.ManagedService.Artifact, []string{"-test.run=^TestManagedUpdateNativeResourcePayload$", "-test.timeout=300s"}, []string{"HOME=" + home, "PATH=/usr/bin:/bin", "RESOURCE_PORT_HTTP=" + strconv.Itoa(port), "UPDATE_NATIVE_MARKER=" + filepath.Join(root, "old-executed")}, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	u.Original = state
	var originalScope *platform.ServiceScopeRef
	var store *managedUpdateStore
	cleaned := false
	t.Cleanup(func() {
		if cleaned {
			os.RemoveAll(root)
			return
		}
		cleanup, stop := context.WithTimeout(context.Background(), 20*time.Second)
		defer stop()
		if store != nil {
			current, err := store.Lookup(u.RequestID)
			if err == nil && current.Candidate.Intent != nil {
				i := *current.Candidate.Intent
				fences, e := newManagedUpdateExecStore(name, u.ID, i.InstanceID)
				if e == nil {
					_, e = fences.Revoke(cleanup, managedUpdateExecFence{SchemaVersion: 1, Resource: name, OperationID: u.ID, Intent: i, State: "prepared"})
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
					t.Errorf("candidate fixture cleanup unconfirmed at %s: %v", store.path, e)
				}
			}
		}
		if originalScope != nil {
			if _, err := platform.StopServiceScope(cleanup, *originalScope); err != nil {
				t.Errorf("original fixture cleanup unconfirmed at %s: %v", root, err)
			}
		} else {
			_ = sup.Stop(cleanup)
		}
	})
	waitReady := func(port int) {
		client := &http.Client{Timeout: time.Second}
		for {
			response, err := client.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/ready")
			if err == nil {
				response.Body.Close()
				if response.StatusCode == http.StatusOK {
					return
				}
			}
			select {
			case <-ctx.Done():
				t.Fatal("original fixture readiness expired")
			case <-time.After(20 * time.Millisecond):
			}
		}
	}
	waitReady(port)
	scopePath, err := platform.ProcessScope(state.PID)
	if err != nil {
		t.Fatal(err)
	}
	_, ticks, _ := strings.Cut(state.Incarnation, ":")
	scope, err := platform.CaptureServiceScope(ctx, platform.ScopeRef{Kind: platform.ScopeKindCgroup, Name: resourceServiceScopeName(name, "service"), Path: scopePath}, state.PID, ticks)
	if err != nil {
		t.Fatal(err)
	}
	originalScope = &scope
	u.OriginalScope = &scope
	m.ManagedService.Artifact.Version = "new-fixture"
	m.ManagedService.Artifact.SHA256 = newDigest
	unhealthy := filepath.Join(root, "candidate-unhealthy")
	m.ManagedService.Environment = map[string]string{"UPDATE_NATIVE_MARKER": filepath.Join(root, "new-executed"), "UPDATE_NATIVE_UNHEALTHY": unhealthy}
	u.Target, err = newManagedUpdateContract(m, ManagedServiceArtifact{Resource: name, Path: os.Args[0], Version: "new-fixture", SHA256: newDigest})
	if err != nil {
		t.Fatal(err)
	}
	store, err = newManagedUpdateStore(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Create(ctx, u, false); err != nil {
		t.Fatal(err)
	}
	launcher, err := exec.LookPath("vrooli")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, launcher, "managed-update-candidate", name, u.RequestID)
	cmd.Dir = projectRoot(t)
	cmd.Env = []string{"HOME=" + home, "PATH=/usr/local/bin:/usr/bin:/bin", "XDG_STATE_HOME=" + os.Getenv("XDG_STATE_HOME")}
	output, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf("candidate fixture owner: %v: %s", err, exit.Stderr)
		}
		t.Fatal(err)
	}
	if err := json.Unmarshal(output, &u); err != nil {
		t.Fatal(err)
	}
	if u.Phase != UpdateCandidateReady {
		t.Fatal("candidate never reached readiness")
	}
	release, err := acquireManagedServiceLifecycleLock(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	u.Candidate, err = observeManagedUpdateProcess(ctx, u, u.Candidate, u.Target)
	if err != nil {
		t.Fatal(err)
	}
	u.Primary.Intent = updateTestIntent("primary-fixture-not-dispatched", u.Target)
	u, err = store.Commit(ctx, u, UpdateStopPrimary)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unhealthy, []byte("candidate now returns503"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := stopManagedUpdateOriginal(ctx, u); err == nil {
		t.Fatal("historical ready proof permitted an original stop while candidate currently failed")
	}
	recovering := u
	recovering.Error = "fixture ambiguous original stop acknowledgement"
	recovering, err = store.Commit(ctx, recovering, UpdateFailure)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stopManagedUpdateOriginalForRestoration(ctx, recovering); err == nil {
		t.Fatal("restoration stopped the original without a live serving shadow")
	}
	if actual, running, err := sup.Status(); err != nil || !running || actual.PID != state.PID {
		t.Fatal("failed candidate guard disturbed original fixture")
	}
	if err := os.Remove(unhealthy); err != nil {
		t.Fatal(err)
	}
	if receipt, err := stopManagedUpdateOriginalForRestoration(ctx, recovering); err != nil || !receipt.RootGone || !receipt.TreeQuiescent {
		t.Fatalf("verified serving shadow did not permit recovery of the admitted exact original stop: %v", err)
	}
	if _, running, err := sup.Status(); err != nil || running {
		t.Fatal("original fixture remains running after native stop")
	}
	if _, err := observeManagedUpdateProcess(ctx, u, u.Candidate, u.Target); err != nil {
		t.Fatalf("original stop disturbed candidate: %v", err)
	}
	i := *u.Candidate.Intent
	fences, err := newManagedUpdateExecStore(name, u.ID, i.InstanceID)
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
	if receipt, err := platform.StopOwnedService(ctx, *u.Candidate.NativeRef); err != nil || !receipt.NativeQuiescent {
		t.Fatalf("candidate cleanup failed: %v", err)
	}
	if err := platform.ReleaseOwnedService(ctx, *u.Candidate.NativeRef); err != nil {
		t.Fatal(err)
	}
	cleaned = true
	report, _ := json.Marshal(map[string]any{"original_scope": scope, "candidate_native_ref": u.Candidate.NativeRef, "unhealthy_candidate_prevented_stop": true, "healthy_candidate_survived_original_stop": true, "cleanup_confirmed": true, "scope": "isolated real processes; no actual connector mutation or primary publication"})
	t.Log(string(report))
}
