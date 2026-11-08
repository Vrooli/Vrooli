//go:build linux

package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/capacity"

	"github.com/vrooli/binaryfetch"
	platform "github.com/vrooli/platform-go"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestManagedUpdateCurrentHostFullRunnerProbe(t *testing.T) {
	if os.Getenv("VROOLI_MANAGED_UPDATE_FULL_RUNNER_PROBE") != "1" {
		t.Skip("set VROOLI_MANAGED_UPDATE_FULL_RUNNER_PROBE=1 for isolated native full update/rollback fixtures")
	}
	for _, mode := range []string{"success", "primary-failure", "candidate-failure", "caller-loss-primary", "caller-loss-all-services", "original-loss-before-owner"} {
		t.Run(mode, func(t *testing.T) {
			root, err := os.MkdirTemp("", "vrooli-full-update-probe-")
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("XDG_STATE_HOME", filepath.Join(root, "xdg-state"))
			t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", filepath.Join(root, "artifacts"))
			u := updateTestInitial(t)
			u.ID = "full-update-probe-" + strconv.FormatInt(time.Now().UnixNano(), 10)
			u.Resource = u.ID
			u.RequestID = u.ID
			u.CreatedAt = time.Now().UTC()
			u.UpdatedAt = u.CreatedAt
			u.RollbackQualificationRequired = true
			budget := managedUpdateCapacityBudget(u)
			budget.LedgerPath = capacity.LedgerReference(filepath.Join(root, "capacity.db"))
			u.Capacity = &budget
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
			defer cancel()
			imageBytes, err := os.ReadFile(os.Args[0])
			if err != nil {
				t.Fatal(err)
			}
			oldPath := filepath.Join(root, "old-image")
			if err := os.WriteFile(oldPath, append(imageBytes, 0), 0o755); err != nil {
				t.Fatal(err)
			}
			oldDigest, err := binaryfetch.FileDigestContext(ctx, oldPath)
			if err != nil {
				t.Fatal(err)
			}
			newDigest, err := binaryfetch.FileDigestContext(ctx, os.Args[0])
			if err != nil {
				t.Fatal(err)
			}
			reservation, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			port := reservation.Addr().(*net.TCPAddr).Port
			reservation.Close()
			home, err := os.UserHomeDir()
			if err != nil {
				t.Fatal(err)
			}
			sup, paths, err := managedServiceSupervisorFor(u.Resource)
			if err != nil {
				t.Fatal(err)
			}
			credentialPath := filepath.Join(paths.StateDir, "runtime-secret")
			if err := os.WriteFile(credentialPath, []byte("operation-pinned-native-fixture\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			m := u.Target.Manifest
			m.Name = u.Resource
			m.Ports[0].Host = port
			m.ManagedService.Artifact.Version = "old-fixture"
			m.ManagedService.Artifact.SHA256ByPlatform = nil
			m.ManagedService.Artifact.SHA256 = oldDigest
			m.ManagedService.CredentialFiles = []resourcedeployment.ServiceCredentialFile{{LogicalID: "fixture/update", Field: "value", Path: "runtime-secret"}}
			m.ManagedService.Environment = map[string]string{"UPDATE_NATIVE_MARKER": filepath.Join(root, "old-executed"), "UPDATE_NATIVE_CREDENTIAL_FILE": "${RESOURCE_STATE_DIR}/runtime-secret"}
			m.ManagedService.Arguments = []string{"-test.run=^TestManagedUpdateNativeResourcePayload$", "-test.timeout=${RESOURCE_PORT_HTTP}s"}
			u.Rollback, err = newManagedUpdateContract(m, ManagedServiceArtifact{Resource: u.Resource, Path: oldPath, Version: "old-fixture", SHA256: oldDigest})
			if err != nil {
				t.Fatal(err)
			}
			state, err := sup.StartContext(ctx, oldPath, m.ManagedService.Artifact, []string{"-test.run=^TestManagedUpdateNativeResourcePayload$", "-test.timeout=300s"}, []string{"HOME=" + home, "PATH=/usr/bin:/bin", "RESOURCE_PORT_HTTP=" + strconv.Itoa(port), "RESOURCE_STATE_DIR=" + paths.StateDir, "UPDATE_NATIVE_MARKER=" + filepath.Join(root, "old-executed"), "UPDATE_NATIVE_CREDENTIAL_FILE=" + credentialPath}, root, nil)
			if err != nil {
				t.Fatal(err)
			}
			u.Original = state
			var originalScope *platform.ServiceScopeRef
			var store *managedUpdateStore
			cleaned := false
			t.Cleanup(func() {
				cleanup, stop := context.WithTimeout(context.Background(), 45*time.Second)
				defer stop()
				if store != nil {
					current, err := store.Lookup(u.RequestID)
					if err == nil {
						roles := []ManagedUpdateProcess{current.RollbackProbe, current.Candidate, current.Primary, current.Restored}
						for _, retired := range current.RetiredRestores {
							roles = append(roles, retired.Process)
						}
						for _, p := range roles {
							if p.Intent != nil {
								f, e := newManagedUpdateExecStore(current.Resource, current.ID, p.Intent.InstanceID)
								if e == nil {
									_, e = f.Revoke(cleanup, managedUpdateExecFence{SchemaVersion: 1, Resource: current.Resource, OperationID: current.ID, Intent: *p.Intent, State: "prepared"})
								}
								if e != nil {
									t.Errorf("fixture fence revocation failed: %v", e)
								}
							}
						}
						if canonical, e := sup.readState(); e == nil && canonical.UpdateLaunch != nil {
							if e := sup.Stop(cleanup); e != nil {
								t.Errorf("canonical fixture stop failed: %v", e)
							}
						}
						for _, p := range roles {
							if e := cleanupManagedUpdateRole(cleanup, current, p); e != nil {
								t.Errorf("fixture cleanup unconfirmed at %s: %v", store.path, e)
							}
						}
					}
				}
				if originalScope != nil {
					if _, e := platform.StopServiceScope(cleanup, *originalScope); e != nil {
						t.Errorf("original fixture cleanup failed: %v", e)
					}
				} else {
					_ = sup.Stop(cleanup)
				}
				if cleaned && !t.Failed() {
					os.RemoveAll(root)
				} else {
					t.Logf("fixture evidence retained at %s", root)
				}
			})
			client := &http.Client{Timeout: time.Second}
			for {
				r, e := client.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/ready")
				if e == nil {
					r.Body.Close()
					if r.StatusCode == http.StatusOK {
						break
					}
				}
				select {
				case <-ctx.Done():
					t.Fatal("original did not become ready")
				case <-time.After(20 * time.Millisecond):
				}
			}
			scopePath, err := platform.ProcessScope(state.PID)
			if err != nil {
				t.Fatal(err)
			}
			_, stamp, _ := strings.Cut(state.Incarnation, ":")
			scope, err := platform.CaptureServiceScope(ctx, platform.ScopeRef{Kind: platform.ScopeKindCgroup, Name: resourceServiceScopeName(u.Resource, "service"), Path: scopePath}, state.PID, stamp)
			if err != nil {
				t.Fatal(err)
			}
			originalScope = &scope
			u.OriginalScope = &scope
			m.ManagedService.Artifact.Version = "new-fixture"
			m.ManagedService.Artifact.SHA256 = newDigest
			m.ManagedService.Environment = map[string]string{"UPDATE_NATIVE_MARKER": filepath.Join(root, "new-executed"), "UPDATE_NATIVE_CREDENTIAL_FILE": "${RESOURCE_STATE_DIR}/runtime-secret"}
			if mode == "primary-failure" || mode == "caller-loss-primary" || mode == "caller-loss-all-services" {
				m.ManagedService.Environment["UPDATE_NATIVE_FAIL_PORT"] = strconv.Itoa(port)
			}
			if mode == "candidate-failure" {
				bad := filepath.Join(root, "candidate-unhealthy")
				if err := os.WriteFile(bad, []byte("503"), 0o600); err != nil {
					t.Fatal(err)
				}
				m.ManagedService.Environment["UPDATE_NATIVE_UNHEALTHY"] = bad
			}
			u.Target, err = newManagedUpdateContract(m, ManagedServiceArtifact{Resource: u.Resource, Path: os.Args[0], Version: "new-fixture", SHA256: newDigest})
			if err != nil {
				t.Fatal(err)
			}
			credentials, err := newManagedUpdateCredentialStore(paths.StateDir, u.ID)
			if err != nil {
				t.Fatal(err)
			}
			resolved := 0
			credentials.resolve = func(context.Context, ManagedUpdateCredentialItem) (string, error) {
				resolved++
				return "operation-pinned-native-fixture", nil
			}
			u.Credentials, err = credentials.Prepare(ctx, u)
			if err != nil {
				t.Fatal(err)
			}
			if resolved != 1 {
				t.Fatal("shared logical credential was captured more than once")
			}
			store, err = newManagedUpdateStore(u.Resource)
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
			if mode == "original-loss-before-owner" {
				if _, e := platform.StopServiceScope(ctx, *u.OriginalScope); e != nil {
					t.Fatal(e)
				}
				t.Log("original fixture lost after durable admission and before any update-owner execution")
			}
			cmd := exec.CommandContext(ctx, launcher, "managed-update-run", u.Resource, u.RequestID)
			cmd.Dir = projectRoot(t)
			cmd.Env = []string{"HOME=" + home, "PATH=/usr/local/bin:/usr/bin:/bin", "XDG_STATE_HOME=" + os.Getenv("XDG_STATE_HOME")}
			if mode == "caller-loss-primary" || mode == "caller-loss-all-services" {
				var discarded bytes.Buffer
				var interrupted ManagedServiceUpdate
				cmd.Stdout = &discarded
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				deadline := time.Now().Add(12 * time.Second)
				for {
					current, e := store.Lookup(u.RequestID)
					if e == nil && current.Phase == UpdatePrimaryStarting && current.Primary.Intent != nil {
						ref, e := platform.LookupOwnedService(ctx, current.Primary.Intent.NativeIntent)
						if e == nil && ref.RootPID > 0 {
							interrupted = current
							break
						}
					}
					if time.Now().After(deadline) {
						cmd.Process.Kill()
						cmd.Wait()
						t.Fatal("caller-loss fixture did not reach a dispatched primary")
					}
					time.Sleep(20 * time.Millisecond)
				}
				if err := cmd.Process.Kill(); err != nil {
					t.Fatal(err)
				}
				if err := cmd.Wait(); err == nil {
					t.Fatal("caller-loss fixture did not lose its original caller")
				}
				if mode == "caller-loss-all-services" {
					// Remove every fixture role after the exact original stop and
					// caller loss. This qualifies total process loss, not a reboot
					// or a change of the kernel's native identity namespace.
					for _, role := range []ManagedUpdateProcess{interrupted.Primary, interrupted.Candidate, interrupted.RollbackProbe} {
						if role.Intent == nil {
							t.Fatal("total-loss fixture lacks admitted role")
						}
						ref, e := platform.LookupOwnedService(ctx, role.Intent.NativeIntent)
						if e != nil {
							t.Fatal(e)
						}
						if _, e := platform.StopOwnedService(ctx, ref); e != nil {
							t.Fatal(e)
						}
						if e := platform.ReleaseOwnedService(ctx, ref); e != nil {
							t.Fatal(e)
						}
					}
					t.Log("all native fixture roles retired after forced caller loss; immutable ledger and exec fences retained")
				}
				cmd = exec.CommandContext(ctx, launcher, "managed-update-run", u.Resource, u.RequestID)
				cmd.Dir = projectRoot(t)
				cmd.Env = []string{"HOME=" + home, "PATH=/usr/local/bin:/usr/bin:/bin", "XDG_STATE_HOME=" + os.Getenv("XDG_STATE_HOME")}
			}
			output, err := cmd.Output()
			if err != nil {
				if exit, ok := err.(*exec.ExitError); ok {
					t.Fatalf("full update owner: %v: %s", err, exit.Stderr)
				}
				t.Fatal(err)
			}
			var result ManagedServiceUpdate
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatal(err)
			}
			wanted := map[string]ManagedUpdatePhase{"success": UpdateSucceeded, "primary-failure": UpdateFailedRestored, "candidate-failure": UpdateFailedPreserved, "caller-loss-primary": UpdateFailedRestored, "caller-loss-all-services": UpdateFailedRestored, "original-loss-before-owner": UpdateSucceeded}[mode]
			if result.Phase != wanted || !result.CandidateQuiescent || !result.RollbackProbeQuiescent || result.RollbackQualifiedAt == nil {
				t.Fatalf("full update result=%s, expected=%s", result.Phase, wanted)
			}
			canonical, running, err := sup.Status()
			if err != nil || !running {
				t.Fatalf("terminal operation lacks live canonical service: %v", err)
			}
			if mode == "candidate-failure" {
				if canonical.InstanceID != state.InstanceID || canonical.UpdateLaunch != nil {
					t.Fatal("preserved failure replaced original record")
				}
			} else {
				if canonical.UpdateLaunch == nil || canonical.UpdateLaunch.OperationID != u.ID {
					t.Fatal("serving publication lost its native owner")
				}
				if mode == "success" {
					controller := NewController(projectRoot(t), home)
					if e := controller.PruneManagedServiceUpdateInputs(ctx, u.Resource); e != nil {
						t.Fatalf("terminal private input retention: %v", e)
					}
					if _, e := os.Stat(filepath.Join(paths.StateDir, "managed-update-exec", u.ID, "credentials")); !os.IsNotExist(e) {
						t.Fatalf("terminal credential copies were retained: %v", e)
					}
					if _, e := os.Stat(filepath.Join(paths.StateDir, "managed-update-exec", u.ID, "launcher")); !os.IsNotExist(e) {
						t.Fatalf("terminal launcher copies were retained: %v", e)
					}
					liveRuntime, e := canonical.UpdateLaunch.Runtime.roleRoot(u.ID, canonical.InstanceID)
					if e != nil {
						t.Fatal(e)
					}
					if _, e := os.Stat(liveRuntime); e != nil {
						t.Fatalf("cleanup removed live serving inputs: %v", e)
					}
					source, e := controller.observeManagedUpdateAdmissionSource(ctx, m)
					if e != nil {
						t.Fatalf("next update cannot observe published native owner: %v", e)
					}
					second := updateTestInitial(t)
					second.ID, second.RequestID, second.Resource = u.ID+"-next", u.RequestID+"-next", u.Resource
					second.CreatedAt, second.UpdatedAt = time.Now().UTC(), time.Now().UTC()
					second.SourceRevision, second.Target, second.Rollback = source.SourceRevision, u.Rollback, source.Contract
					second.Target, e = controller.managedUpdateAdmissionTarget(ctx, m, store, ManagedServiceUpdateAdmission{RollbackOf: u.RequestID, ExpectedVersion: u.Rollback.Artifact.Version, ExpectedSHA256: u.Rollback.Artifact.SHA256})
					if e != nil {
						t.Fatalf("qualified receipt rollback selection: %v", e)
					}
					second.RollbackOf = u.RequestID
					second.Original, second.OriginalScope = canonical, nil
					second.RollbackQualificationRequired = true
					secondBudget := managedUpdateCapacityBudget(second)
					secondBudget.LedgerPath = budget.LedgerPath
					second.Capacity = &secondBudget
					secrets, e := newManagedUpdateCredentialStore(paths.StateDir, second.ID)
					if e != nil {
						t.Fatal(e)
					}
					secrets.resolve = func(context.Context, ManagedUpdateCredentialItem) (string, error) {
						return "operation-pinned-native-fixture", nil
					}
					second.Credentials, e = secrets.Prepare(ctx, second)
					if e != nil {
						t.Fatal(e)
					}
					if _, _, e := store.Create(ctx, second, false); e != nil {
						t.Fatal(e)
					}
					t.Cleanup(func() {
						cleanup, cancel := context.WithTimeout(context.Background(), 45*time.Second)
						defer cancel()
						if e := sup.Stop(cleanup); e != nil {
							t.Errorf("second fixture canonical cleanup: %v", e)
						}
						if current, e := store.Lookup(second.RequestID); e == nil {
							for _, p := range []ManagedUpdateProcess{current.RollbackProbe, current.Candidate, current.Primary, current.Restored} {
								if e := cleanupManagedUpdateRole(cleanup, current, p); e != nil {
									t.Errorf("second fixture role cleanup: %v", e)
								}
							}
						}
					})
					follow := exec.CommandContext(ctx, launcher, "managed-update-run", u.Resource, second.RequestID)
					follow.Dir, follow.Env = cmd.Dir, cmd.Env
					encoded, e := follow.Output()
					if e != nil {
						if exit, ok := e.(*exec.ExitError); ok {
							t.Fatalf("second native update: %v: %s", e, exit.Stderr)
						}
						t.Fatal(e)
					}
					var next ManagedServiceUpdate
					if e := json.Unmarshal(encoded, &next); e != nil {
						t.Fatal(e)
					}
					if next.Phase != UpdateSucceeded {
						t.Fatalf("second update from native owner reached %s", next.Phase)
					}
					t.Log("second sequential update succeeded from the previously published native owner")
					// Leave the last native unit recorded but stopped, like a dead
					// canonical service. Ordinary recovery must retire that exact
					// unit before replacing its serving record with a scoped start.
					prior, live, e := sup.Status()
					if e != nil || !live || prior.UpdateLaunch == nil {
						t.Fatalf("native recovery fixture has no serving owner: %v", e)
					}
					ref := prior.UpdateLaunch.NativeRef
					if _, e := platform.StopOwnedService(ctx, ref); e != nil {
						t.Fatal(e)
					}
					originalScope = nil // Its former legacy name can now be reused.
					sup.verifyPlacement = func(pid int) error {
						scope, e := platform.ProcessScope(pid)
						if e != nil {
							return e
						}
						if filepath.Base(scope) != resourceServiceScopeName(u.Resource, "service")+".scope" {
							return fmt.Errorf("recovery remained in caller scope")
						}
						return nil
					}
					recovered, e := sup.StartContext(ctx, os.Args[0], u.Target.Manifest.ManagedService.Artifact, []string{"-test.run=^TestManagedUpdateNativeResourcePayload$", "-test.timeout=300s"}, []string{"HOME=" + home, "PATH=/usr/bin:/bin", "RESOURCE_PORT_HTTP=" + strconv.Itoa(port), "UPDATE_NATIVE_MARKER=" + filepath.Join(root, "recovered-executed"), "UPDATE_NATIVE_CREDENTIAL_FILE=" + credentialPath}, root, nil)
					if e != nil {
						t.Fatalf("ordinary recovery from dead native owner: %v", e)
					}
					if recovered.UpdateLaunch != nil {
						t.Fatal("ordinary recovery retained obsolete native ownership")
					}
					if released, e := platform.OwnedServiceReleased(ctx, ref); e != nil || !released {
						t.Fatalf("ordinary recovery lost prior native unit: %v", e)
					}
					t.Log("ordinary scoped recovery retired the prior dead native owner")
					if e := controller.PruneManagedServiceUpdateInputs(ctx, u.Resource); e != nil {
						t.Fatalf("retired serving input cleanup: %v", e)
					}
					if _, e := os.Stat(liveRuntime); !os.IsNotExist(e) {
						t.Fatalf("retired serving inputs survived cleanup: %v", e)
					}
				}
				if err := sup.Stop(ctx); err != nil {
					t.Fatal(err)
				}
				if err := sup.Stop(ctx); err != nil {
					t.Fatalf("repeated canonical stop did not reconcile release: %v", err)
				}
			}
			cleaned = true
			report, _ := json.Marshal(map[string]any{"mode": mode, "phase": result.Phase, "operation": u.ID, "resource": u.Resource, "revision": result.Revision, "credential_snapshot_digest": result.Credentials.Digest, "rollback_qualified": true, "canonical_native_owner": canonical.UpdateLaunch != nil, "scope": "isolated native fixture; no serving connector mutation"})
			t.Log(string(report))
		})
	}
}
