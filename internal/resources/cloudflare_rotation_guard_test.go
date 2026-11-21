package resources

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/securestore"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestCloudflareRotationGuardBothLaunchPaths(t *testing.T) {
	for _, state := range []string{"absent", "pending", "unavailable"} {
		t.Run(state, func(t *testing.T) {
			store := &memorySecureStore{values: map[string]string{}}
			var backend securestore.Store = store
			if state == "unavailable" {
				backend = securestore.Unavailable("fixture")
			}
			authority := withCredentialStore(t, backend)
			identity, err := credentialauthority.ParseIdentity("vrooli/tunnel-manager")
			if err != nil {
				t.Fatal(err)
			}
			if state != "unavailable" {
				if err := authority.Put(identity, "cloudflare-connector-token", "private-operation-fixture"); err != nil {
					t.Fatal(err)
				}
				if state == "pending" {
					if err := authority.Put(identity, "cloudflare-connector-rotation-intent", "private-intent"); err != nil {
						t.Fatal(err)
					}
				}
			}
			u := updateTestInitial(t)
			root := t.TempDir()
			_, normalErr := materializeManagedServiceCredentialFiles(u.Target.Manifest, []string{"RESOURCE_STATE_DIR=" + root})
			s, _, _ := updateCredentialTestStore(t, u)
			snapshot, err := s.Prepare(context.Background(), u)
			if err != nil {
				t.Fatal(err)
			}
			u.Credentials = snapshot
			r := updateLaunchTestRuntime(t)
			r.StateDir = filepath.Dir(filepath.Dir(filepath.Dir(s.dir)))
			invocation, err := renderManagedUpdateInvocation(u.Target, r, u.ID, "candidate", "role-private-token")
			if err != nil {
				t.Fatal(err)
			}
			updateErr := materializeManagedUpdateInvocation(u, u.Target, invocation)
			if state == "absent" {
				if normalErr != nil || updateErr != nil {
					t.Fatalf("absent intent should permit materialization: normal=%v update=%v", normalErr, updateErr)
				}
				return
			}
			want := "rotation is pending"
			if state == "unavailable" {
				want = "rotation authority unavailable"
			}
			for _, got := range []error{normalErr, updateErr} {
				if got == nil || !strings.Contains(got.Error(), want) || strings.Contains(got.Error(), "private-intent") {
					t.Fatalf("expected safe %s error, got %v", want, got)
				}
			}
			if _, err := os.Stat(filepath.Join(root, "cloudflare-connector-token")); !os.IsNotExist(err) {
				t.Fatalf("blocked normal launch materialized a credential: %v", err)
			}
		})
	}
}

func TestCloudflareRestartPreservesServingProcessWhenRotationBlocks(t *testing.T) {
	body, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(body))
	for _, owner := range []string{"private", "shared"} {
		for _, state := range []string{"pending", "unavailable", "locked", "clean"} {
			t.Run(owner+"/"+state, func(t *testing.T) {
				if owner == "shared" && state == "clean" {
					t.Skip("shared fixture has no HTTP endpoint")
				}
				root := t.TempDir()
				t.Setenv("HOME", root)
				t.Setenv("SUDO_USER", "")
				t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "runtime"))
				t.Setenv("VROOLI_MANAGED_SERVICE_FIXTURE", "1")
				binary := filepath.Join(root, "resources", "fixture-service", "bin", "service")
				if err := os.MkdirAll(filepath.Dir(binary), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(binary, body, 0700); err != nil {
					t.Fatal(err)
				}
				manifest := managedServiceTestManifest(sum)
				controller := NewController(root, root)
				driver := managedServiceDriver{}
				item := Resource{Name: manifest.Name}
				if err := driver.Run(context.Background(), controller, item, manifest, "start", nil, io.Discard, io.Discard); err != nil {
					t.Fatal(err)
				}
				defer driver.Run(context.Background(), controller, item, manifest, "stop", nil, io.Discard, io.Discard)
				supervisor, _, err := managedServiceSupervisorFor(manifest.Name)
				if err != nil {
					t.Fatal(err)
				}
				before, running, err := supervisor.Status()
				if err != nil || !running {
					t.Fatalf("baseline unavailable: %v", err)
				}
				manifest.ManagedService.CredentialFiles = []resourcedeployment.ServiceCredentialFile{{LogicalID: "vrooli/tunnel-manager", Field: "cloudflare-connector-token", Path: "token"}}
				store := &memorySecureStore{values: map[string]string{}}
				var backend securestore.Store = store
				if state == "unavailable" {
					backend = securestore.Unavailable("fixture")
				}
				authority := withCredentialStore(t, backend)
				id, err := credentialauthority.ParseIdentity("vrooli/tunnel-manager")
				if err != nil {
					t.Fatal(err)
				}
				if state == "pending" {
					if err := authority.Put(id, "cloudflare-connector-rotation-intent", "fixture"); err != nil {
						t.Fatal(err)
					}
				}
				if state == "clean" {
					if err := authority.Put(id, "cloudflare-connector-token", "fixture-token"); err != nil {
						t.Fatal(err)
					}
				}
				if owner == "shared" {
					original := managedSharedStoreForHost
					managedSharedStoreForHost = func() securestore.Store { return store }
					t.Cleanup(func() { managedSharedStoreForHost = original })
					managedSharedBootstrappers[manifest.Name] = func(context.Context, *UserResourceHost, ManagedInstance, string) error { return nil }
					t.Cleanup(func() { delete(managedSharedBootstrappers, manifest.Name) })
				}
				var release func()
				if state == "locked" {
					release, err = credentialauthority.AcquireCloudflareConnectorLock(context.Background())
					if err != nil {
						t.Fatal(err)
					}
					defer release()
				}
				timeout := 300 * time.Millisecond
				if state == "clean" {
					timeout = 10 * time.Second
				}
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()
				if owner == "shared" {
					err = driver.runUserHosted(ctx, controller, item, manifest, "restart", io.Discard)
				} else {
					err = driver.runPrivate(ctx, controller, item, manifest, "restart", io.Discard)
				}
				if state == "clean" {
					if err != nil {
						t.Fatalf("clean restart failed: %v", err)
					}
					after, running, err := supervisor.Status()
					if err != nil || !running || before.PID == after.PID {
						t.Fatalf("clean restart did not replace process: %v", err)
					}
					return
				}
				if err == nil {
					t.Fatal("blocked restart succeeded")
				}
				if state == "pending" && !strings.Contains(err.Error(), "rotation is pending") {
					t.Fatalf("unexpected error: %v", err)
				}
				if state == "unavailable" && !strings.Contains(err.Error(), "rotation authority unavailable") {
					t.Fatalf("unexpected error: %v", err)
				}
				if state == "locked" && !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("restart did not wait for rotation lock: %v", err)
				}
				after, running, err := supervisor.Status()
				if err != nil || !running || before.PID != after.PID || before.InstanceID != after.InstanceID {
					t.Fatalf("restart interrupted serving process: before=%+v after=%+v running=%t err=%v", before, after, running, err)
				}
			})
		}
	}
}
