package resources

import (
	"context"
	"errors"
	"fmt"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExclusiveGroupSerializesDifferentResourceLifecycles(t *testing.T) {
	group := fmt.Sprintf("group-test-%d", time.Now().UnixNano())
	first := ResourceManifest{Name: "first"}
	first.Lifecycle.ExclusiveGroup = group
	second := ResourceManifest{Name: "second"}
	second.Lifecycle.ExclusiveGroup = group
	release, err := acquireManagedLifecycleLocks(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()
	if unlock, err := acquireManagedLifecycleLocks(ctx, second); !errors.Is(err, context.DeadlineExceeded) {
		if unlock != nil {
			unlock()
		}
		t.Fatalf("second group member entered while first held lifecycle: %v", err)
	}
	release()
	unlock, err := acquireManagedLifecycleLocks(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
}

func TestExclusiveGroupRejectsOwnedLivePeerAndAllowsStoppedPeer(t *testing.T) {
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", t.TempDir())
	root := t.TempDir()
	controller := NewController(root, t.TempDir())
	peer := managedServiceTestManifest(strings.Repeat("a", 64))
	peer.Name, peer.Lifecycle.ExclusiveGroup = "peer-model", "test-music"
	peer.ManagedService.ProviderPolicy.AllowedModes = []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate}
	peer.ManagedService.ProviderPolicy.ExternalAccessCapabilities = nil
	writeResourceManifest(t, root, peer.Name, peer)
	supervisor, _, err := managedServiceSupervisorFor(peer.Name)
	if err != nil {
		t.Fatal(err)
	}
	state := ownershipState()
	state.InstanceID, state.ArtifactSHA256, state.ArtifactVersion = "fixture-instance", strings.Repeat("a", 64), "superseded"
	state.PID = os.Getpid()
	withOwnershipFacts(t, func(int) (map[string]string, error) {
		return map[string]string{managedServiceOwnershipTokenEnv: "owned"}, nil
	}, executableAt(state.ArtifactPath))
	if err := supervisor.writeState(state); err != nil {
		t.Fatal(err)
	}
	selected := peer
	selected.Name = "selected-model"
	if err := checkExclusiveResourceGroup(controller, selected); err == nil || !strings.Contains(err.Error(), `exclusive group "test-music" is occupied by peer-model`) {
		t.Fatalf("live peer not refused: %v", err)
	}
	state.PID = 0
	if err := supervisor.writeState(state); err != nil {
		t.Fatal(err)
	}
	if err := checkExclusiveResourceGroup(controller, selected); err != nil {
		t.Fatalf("stopped peer blocked admission: %v", err)
	}
	if err := os.WriteFile(supervisor.statePath, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkExclusiveResourceGroup(controller, selected); err == nil {
		t.Fatal("unreadable ownership inventory treated as no peer")
	}
	if !strings.HasPrefix(supervisor.statePath, filepath.Clean(os.Getenv("VROOLI_RESOURCE_STORAGE_ROOT"))) {
		t.Fatal("fixture escaped its isolated state root")
	}
}

func TestExclusiveGroupDoesNotSerializeUnrelatedResources(t *testing.T) {
	first := ResourceManifest{Name: "group-first"}
	first.Lifecycle.ExclusiveGroup = fmt.Sprintf("group-test-%d", time.Now().UnixNano())
	second := ResourceManifest{Name: "group-second"}
	second.Lifecycle.ExclusiveGroup = first.Lifecycle.ExclusiveGroup + "-other"
	release, err := acquireManagedLifecycleLocks(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	unlock, err := acquireManagedLifecycleLocks(ctx, second)
	if err != nil {
		t.Fatalf("unrelated group blocked: %v", err)
	}
	unlock()
}

func TestExclusiveGroupCannotBeBypassedThroughSharedBrokerLifecycle(t *testing.T) {
	manifest := ResourceManifest{Name: "grouped-model"}
	manifest.Lifecycle.ExclusiveGroup = "test-music"
	owner := ManagedServiceOwnerLifecycle{Controller: NewController(t.TempDir(), t.TempDir()), Manifest: manifest}
	for _, action := range []string{"start", "restart"} {
		_, err := owner.Manage(t.Context(), ManagedInstance{Resource: manifest.Name, Provider: resourcedeployment.ProviderManagedShared}, action)
		if err == nil || !strings.Contains(err.Error(), "shared broker activation refused") {
			t.Fatalf("bypass %s: %v", action, err)
		}
	}
}
