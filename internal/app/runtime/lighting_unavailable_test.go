package runtimeapp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/resources"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup"
	deployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func unavailableLightingFixture(t *testing.T) (LightingActions, string) {
	t.Helper()
	storagetest.IsolateStorage(t)
	home := t.TempDir()
	root := t.TempDir()
	source, err := os.ReadFile("../../../resources/openrgb/resource.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err = json.Unmarshal(source, &manifest); err != nil {
		t.Fatal(err)
	}
	artifact := manifest["managed_service"].(map[string]any)["artifact"].(map[string]any)
	target, err := deployment.CanonicalPlatform(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	delete(artifact["sha256_by_platform"].(map[string]any), target.String())
	delete(artifact, "sha256")
	path := filepath.Join(root, "resources", "openrgb", "resource.json")
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	control, err := NewLightingControl(home, lightingruntime.Owners{Resource: &lightingResourceOwnerFake{}, Scenario: &lightingScenarioOwnerFake{}}, func(context.Context) (resources.Status, error) {
		t.Fatal("unsupported target probed backend")
		return resources.Status{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return LightingActions{Control: control, Resources: resources.NewController(root, home)}, target.String()
}

func TestLightingUnavailableTargetStatusKeepsOptOutTruthful(t *testing.T) {
	actions, target := unavailableLightingFixture(t)
	result, err := actions.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "disabled" || !result.Degraded || result.Ready || result.ResourceObserved || result.PinnedSelection != nil || result.SetupVerified || !strings.Contains(result.LaunchBlocker, target) {
		t.Fatalf("unsupported release became fabricated qualification: %+v", result)
	}
}

func TestLightingUnavailableTargetCannotEnableOrProvision(t *testing.T) {
	actions, _ := unavailableLightingFixture(t)
	request := &setupv1.LightingSelection{Source: setupv1.InstallationSource_INSTALLATION_SOURCE_MANAGED, Version: "1.1", Sha256: strings.Repeat("a", 64), DevicePolicyId: "policy"}
	if _, err := actions.Enable(context.Background(), request); !errors.Is(err, errLightingArtifactUnavailable) {
		t.Fatalf("unsupported enable not refused: %v", err)
	}
	if err := prepareNativeLightingMachine(context.Background(), actions.Resources, actions.Resources.Home, actions.Resources.Root); err == nil {
		t.Fatal("unsupported target provisioned native machine")
	}
	intent, err := actions.Control.controller.Current()
	if err != nil || intent.Phase != lightingruntime.Disabled {
		t.Fatalf("unsupported target changed opt-out: %+v %v", intent, err)
	}
}
