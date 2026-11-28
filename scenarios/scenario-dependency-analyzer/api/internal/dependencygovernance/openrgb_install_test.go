package dependencygovernance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"connectrpc.com/connect"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/installgateway"
)

func TestOpenRGBSourceRequiresRecordedApprovalBeforeStaging(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("pinned OpenRGB acquisition is linux-amd64")
	}
	for _, test := range []struct {
		name, record              string
		apply, installed, blocked bool
	}{
		{name: "unrecorded", record: "[]", apply: true, blocked: true},
		{name: "denied", record: `[{"ecosystem":"source","package_name":"openrgb","rationale":"Exact source fixture.","version_range":"1.1","state":"denied"}]`, apply: true, blocked: true},
		{name: "approved preview", record: `[{"ecosystem":"source","package_name":"openrgb","rationale":"Exact source fixture.","version_range":"1.1","range_policy":"exact","state":"approved","allowed_scenarios":["openrgb"]}]`},
		{name: "approved stage", record: `[{"ecosystem":"source","package_name":"openrgb","rationale":"Exact source fixture.","version_range":"1.1","range_policy":"exact","state":"approved","allowed_scenarios":["openrgb"]}]`, apply: true, installed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			directory := filepath.Join(root, "resources", "openrgb")
			if err := os.MkdirAll(directory, 0700); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "resources", "openrgb", "resource.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "resource.json"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			writeRegistry(t, root, fmt.Sprintf(`{"records":%s}`, test.record))
			installer := &recordingInstaller{}
			handler := withTestDiskAdmission(t, &connectHandler{scenariosDir: func() string { return filepath.Join(root, "scenarios") }, installer: installer})
			response, err := handler.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{Scenario: "openrgb", Surface: "host-tool", Ecosystem: "source", PackageName: "openrgb", Version: "1.1", ExactVersion: true, Apply: test.apply}))
			if err != nil {
				t.Fatal(err)
			}
			if response.Msg.Installed != test.installed || response.Msg.Blocked != test.blocked || installer.called != test.installed {
				t.Fatalf("incorrect admission/effect response %+v installer=%v", response.Msg, installer.called)
			}
			if test.installed && !strings.Contains(response.Msg.Message, "serving process unchanged") {
				t.Fatalf("staging claimed lifecycle activation: %s", response.Msg.Message)
			}
		})
	}
}

func TestOpenRGBChangedSameVersionDigestIsDeniedBeforeAcquisition(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("frozen OpenRGB profile is linux-amd64")
	}
	for _, change := range []string{"target", "artifact", "both"} {
		t.Run(change, func(t *testing.T) {
			root := t.TempDir()
			directory := filepath.Join(root, "resources", "openrgb")
			if err := os.MkdirAll(directory, 0700); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "resources", "openrgb", "resource.json"))
			if err != nil {
				t.Fatal(err)
			}
			var manifest map[string]any
			if err := json.Unmarshal(raw, &manifest); err != nil {
				t.Fatal(err)
			}
			service := manifest["managed_service"].(map[string]any)
			if change == "target" || change == "both" {
				service["acquisition"].(map[string]any)["targets"].([]any)[0].(map[string]any)["sha256"] = strings.Repeat("b", 64)
			}
			if change == "artifact" || change == "both" {
				service["artifact"].(map[string]any)["sha256_by_platform"].(map[string]any)["linux-amd64"] = strings.Repeat("b", 64)
			}
			raw, err = json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "resource.json"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			writeRegistry(t, root, `{"records":[{"ecosystem":"source","package_name":"openrgb","version_range":"1.1","range_policy":"exact","state":"approved","rationale":"Frozen exact release fixture."}]}`)
			installer := &recordingInstaller{}
			handler := &connectHandler{scenariosDir: func() string { return filepath.Join(root, "scenarios") }, installer: installer, diskBroker: &testDiskJobBroker{}, diskPlanner: installgateway.PlanDiskAdmission}
			response, err := handler.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{Scenario: "openrgb", Surface: "host-tool", Ecosystem: "source", PackageName: "openrgb", Version: "1.1", ExactVersion: true, Apply: true}))
			if err != nil {
				t.Fatal(err)
			}
			if !response.Msg.Blocked || response.Msg.Verdict != "disk_admission_denied" || installer.called {
				t.Fatalf("changed same-version digest reached acquisition: %+v called=%v", response.Msg, installer.called)
			}
		})
	}
}
