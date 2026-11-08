//go:build linux

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
)

const execNativeFixtureEnv = "VROOLI_UPDATE_EXEC_NATIVE_FIXTURE"

type execNativeFixtureInput struct {
	Fence     managedUpdateExecFence
	FencePath string
	Marker    string
}

func TestManagedUpdateNativeExecFixture(t *testing.T) {
	input := os.Getenv(execNativeFixtureEnv)
	if input == "" {
		return
	}
	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	var f execNativeFixtureInput
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ref, err := platform.LookupOwnedService(ctx, f.Fence.Intent.NativeIntent)
	if err != nil || ref.RootPID != os.Getpid() {
		t.Fatalf("fixed wrapper is outside its admitted native unit: %v", err)
	}
	if _, err := execFenceTestStore(f.FencePath, f.Fence).Claim(ctx, f.Fence, ref); errors.Is(err, ErrManagedUpdateExecRevoked) {
		return
	} else if err != nil {
		t.Fatal(err)
	}
	// This marker stands in for resource exec. It must never appear when a
	// native submission arrives after the resource owner revoked permission.
	if err := os.WriteFile(f.Marker, []byte("resource exec authorized"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestManagedUpdateCurrentHostRevokedExecProbe(t *testing.T) {
	if os.Getenv("VROOLI_MANAGED_UPDATE_EXEC_PROBE") != "1" {
		t.Skip("set VROOLI_MANAGED_UPDATE_EXEC_PROBE=1 for the exact-owned delayed native dispatch probe")
	}
	root, err := os.MkdirTemp("", "vrooli-managed-update-exec-probe-")
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(root, "native-input.json")
	marker := filepath.Join(root, "resource-executed")
	instance := "late-native-probe-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	spec := platform.OwnedServiceSpec{Path: os.Args[0], Dir: root, Args: []string{"-test.run=^TestManagedUpdateNativeExecFixture$"}, Env: []string{"PATH=/usr/bin:/bin", execNativeFixtureEnv + "=" + input}, Limits: platform.ServiceLimits{MemoryMaxBytes: 512 << 20, TasksMax: 64, CPUWeight: 100}}
	intent, err := platform.NewServiceIntent("resource-update", instance, spec)
	if err != nil {
		t.Fatal(err)
	}
	f := managedUpdateExecFence{SchemaVersion: 1, Resource: "cloudflared", OperationID: instance, Intent: ManagedUpdateLaunchIntent{InstanceID: instance, OwnershipTokenHash: updateTestDigest(instance), ContractDigest: updateTestDigest("fixed-native-probe"), NativeIntent: intent, Runtime: updateLaunchTestRuntime(t)}, State: "prepared"}
	store := execFenceTestStore(filepath.Join(root, "fence.json"), f)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := store.Prepare(ctx, f); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Revoke(ctx, f); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(execNativeFixtureInput{Fence: f, FencePath: store.path, Marker: marker})
	if err != nil {
		t.Fatal(err)
	}
	if err := config.WriteOwnedFileDurable(input, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cleaned := false
	t.Cleanup(func() {
		if !cleaned {
			cleanup, stop := context.WithTimeout(context.Background(), 16*time.Second)
			defer stop()
			ref, err := platform.LookupOwnedService(cleanup, intent)
			if err == nil {
				_, err = platform.StopOwnedService(cleanup, ref)
				if err == nil {
					err = platform.ReleaseOwnedService(cleanup, ref)
				}
			}
			if err != nil {
				t.Errorf("native probe cleanup unconfirmed; retaining exact intent at %s: %v", input, err)
				return
			}
		}
		if err := os.RemoveAll(root); err != nil {
			t.Errorf("remove quiescent native fixture directory: %v", err)
		}
	})
	// Deliberately submit AFTER durable revocation. Native-manager lateness
	// must not resurrect the resource's separate exec permission.
	ref, err := platform.StartOwnedService(ctx, intent, spec)
	if err != nil {
		t.Fatal(err)
	}
	op := platform.OwnedOperation{Ref: platform.OperationRef(ref)}
	if err := op.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("delayed native unit executed the revoked resource: %v", err)
	}
	if receipt, err := platform.StopOwnedService(ctx, ref); err != nil || !receipt.NativeQuiescent {
		t.Fatalf("native fixture quiescence unconfirmed: %v", err)
	}
	if err := platform.ReleaseOwnedService(ctx, ref); err != nil {
		t.Fatal(err)
	}
	cleaned = true
	t.Log("late native dispatch reached the wrapper, but durable revocation prevented resource exec; exact native unit stopped and released")
}
