//go:build linux

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
)

func TestManagedUpdateNativeResourcePayload(t *testing.T) {
	marker := os.Getenv("UPDATE_NATIVE_MARKER")
	if marker == "" {
		return
	}
	listener, err := net.Listen("tcp", "127.0.0.1:"+os.Getenv("RESOURCE_PORT_HTTP"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if path := os.Getenv("UPDATE_NATIVE_CREDENTIAL_FILE"); path != "" {
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "operation-pinned-native-fixture\n" {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
		if failPort := os.Getenv("UPDATE_NATIVE_FAIL_PORT"); failPort != "" && failPort == os.Getenv("RESOURCE_PORT_HTTP") {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if unhealthy := os.Getenv("UPDATE_NATIVE_UNHEALTHY"); unhealthy != "" {
			if _, err := os.Stat(unhealthy); err == nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})); err != nil {
		t.Fatal(err)
	}
}

func TestManagedUpdateCurrentHostProductionWrapperProbe(t *testing.T) {
	if os.Getenv("VROOLI_MANAGED_UPDATE_PRODUCTION_PROBE") != "1" {
		t.Skip("set VROOLI_MANAGED_UPDATE_PRODUCTION_PROBE=1 for isolated production CLI/native execution")
	}
	for _, revoked := range []bool{false, true} {
		t.Run(fmt.Sprintf("revoked=%t", revoked), func(t *testing.T) {
			root, err := os.MkdirTemp("", "vrooli-update-production-probe-")
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("XDG_STATE_HOME", filepath.Join(root, "xdg-state"))
			u := updateTestInitial(t)
			u.ID = "production-probe-" + strconv.FormatInt(time.Now().UnixNano(), 10)
			u.RequestID = u.ID
			u.CreatedAt = time.Now().UTC()
			u.UpdatedAt = u.CreatedAt
			marker := filepath.Join(root, "resource-executed")
			m := u.Target.Manifest
			m.ManagedService.CredentialFiles = nil
			m.ManagedService.Environment = map[string]string{"UPDATE_NATIVE_MARKER": marker}
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
			launcher, err := exec.LookPath("vrooli")
			if err != nil {
				t.Fatal(err)
			}
			launcher, err = filepath.EvalSymlinks(launcher)
			if err != nil {
				t.Fatal(err)
			}
			image, err = os.Open(launcher)
			if err != nil {
				t.Fatal(err)
			}
			launcherDigest, err := managedUpdateImageDigest(context.Background(), image)
			image.Close()
			if err != nil {
				t.Fatal(err)
			}
			paths, err := resourceStoragePaths(u.Resource)
			if err != nil {
				t.Fatal(err)
			}
			home, err := os.UserHomeDir()
			if err != nil {
				t.Fatal(err)
			}
			reservation, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			port := reservation.Addr().(*net.TCPAddr).Port
			reservation.Close()
			r := ManagedUpdateRuntime{Root: projectRoot(t), Home: home, StateHome: os.Getenv("XDG_STATE_HOME"), StateDir: paths.StateDir, Launcher: launcher, LauncherSHA256: launcherDigest, Port: port, Limits: platform.ServiceLimits{MemoryMaxBytes: 512 << 20, TasksMax: 128, CPUWeight: 100}}
			instance := u.ID + "-candidate"
			i := ManagedUpdateLaunchIntent{InstanceID: instance, ContractDigest: u.Target.Digest, OwnershipTokenHash: managedServiceTokenHash("isolated-native-token"), Runtime: r}
			i.NativeIntent, err = platform.NewServiceIntent("resource-update", instance, managedUpdateWrapperSpec(u, i))
			if err != nil {
				t.Fatal(err)
			}
			roleRoot, err := r.roleRoot(u.ID, instance)
			if err != nil {
				t.Fatal(err)
			}
			if err := config.WriteOwnedFileDurable(filepath.Join(roleRoot, "ownership-token"), []byte("isolated-native-token"), 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			store, err := newManagedUpdateStore(u.Resource)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := store.Create(ctx, u, false); err != nil {
				t.Fatal(err)
			}
			fences, err := newManagedUpdateExecStore(u.Resource, u.ID, instance)
			if err != nil {
				t.Fatal(err)
			}
			f := managedUpdateExecFence{SchemaVersion: 1, Resource: u.Resource, OperationID: u.ID, Intent: i, State: "prepared"}
			if _, err := fences.Prepare(ctx, f); err != nil {
				t.Fatal(err)
			}
			u.Candidate.Intent = &i
			u, err = store.Commit(ctx, u, UpdateLaunchCandidate)
			if err != nil {
				t.Fatal(err)
			}
			if revoked {
				if _, err := fences.Revoke(ctx, f); err != nil {
					t.Fatal(err)
				}
			}
			cleaned := false
			t.Cleanup(func() {
				if cleaned {
					os.RemoveAll(root)
					return
				}
				cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
				defer stop()
				ref, err := platform.LookupOwnedService(cleanup, i.NativeIntent)
				if err == nil {
					_, err = platform.StopOwnedService(cleanup, ref)
					if err == nil {
						err = platform.ReleaseOwnedService(cleanup, ref)
					}
				}
				if err != nil {
					t.Errorf("native cleanup unconfirmed; exact intent retained at %s: %v", fences.path, err)
				}
			})
			ref, err := platform.StartOwnedService(ctx, i.NativeIntent, managedUpdateWrapperSpec(u, i))
			if err != nil {
				t.Fatal(err)
			}
			refJSON, _ := json.Marshal(ref)
			t.Logf("submitted exact native fixture: %s", refJSON)
			if revoked {
				op := platform.OwnedOperation{Ref: platform.OperationRef(ref)}
				if err := op.Wait(ctx); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(marker); !os.IsNotExist(err) {
					t.Fatal("revoked production wrapper executed resource")
				}
				rejected, readErr := fences.read()
				if readErr != nil || rejected.RejectedClaim == nil || rejected.RejectedAt == nil || *rejected.RejectedClaim != ref {
					observation, inspectErr := platform.InspectOwnedService(ctx, ref)
					journal, journalErr := exec.CommandContext(ctx, "journalctl", "--user", "--unit="+ref.Unit, "--no-pager", "-n", "30").CombinedOutput()
					t.Fatalf("exact revoked native delivery was not durably rejected: read=%v observation=%+v inspect=%v journal=%v: %s", readErr, observation, inspectErr, journalErr, journal)
				}
			} else {
				client := &http.Client{Timeout: time.Second}
				for {
					response, err := client.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/ready")
					if err == nil {
						response.Body.Close()
						if response.StatusCode == http.StatusOK {
							break
						}
					}
					select {
					case <-ctx.Done():
						t.Fatal("production resource never became ready")
					case <-time.After(100 * time.Millisecond):
					}
				}
				pid, err := os.ReadFile(marker)
				if err != nil || string(pid) != strconv.Itoa(ref.RootPID) {
					t.Fatalf("native wrapper did not retain exact root PID across exec: %v", err)
				}
				actual, err := platform.OpenProcessExecutable(ref.RootPID)
				if err != nil {
					t.Fatal(err)
				}
				actualDigest, err := managedUpdateImageDigest(ctx, actual)
				actual.Close()
				if err != nil || actualDigest != digest {
					t.Fatal("running image differs from admitted artifact")
				}
				claimed, err := fences.read()
				if err != nil || claimed.State != "claimed" || claimed.Claim == nil || claimed.Claim.RootPID != ref.RootPID {
					t.Fatal("production exec lacks exact once-only claim")
				}
			}
			if _, err := fences.Revoke(ctx, f); err != nil {
				t.Fatal(err)
			}
			if receipt, err := platform.StopOwnedService(ctx, ref); err != nil || !receipt.NativeQuiescent {
				t.Fatalf("production native scope cleanup failed: %v", err)
			}
			if err := platform.ReleaseOwnedService(ctx, ref); err != nil {
				t.Fatal(err)
			}
			cleaned = true
			report, _ := json.Marshal(map[string]any{"revoked": revoked, "native_intent": i.NativeIntent, "native_ref": ref, "resource_image_sha256": digest, "launcher_sha256": launcherDigest, "cleanup_confirmed": true, "scope": "isolated fixture; synthetic original serving record"})
			t.Log(string(report))
		})
	}
}
