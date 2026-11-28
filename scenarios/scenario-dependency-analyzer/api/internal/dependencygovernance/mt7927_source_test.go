package dependencygovernance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	mt7927source "github.com/vrooli/vrooli/internal/tools/mt7927-source"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/installgateway"
)

func TestMT7927SourceInstallRequiresGovernanceAndDiskAdmission(t *testing.T) {
	name, version := mt7927source.Name, mt7927source.Version
	sourcePath := filepath.Join("../../../../..", "internal", "tools", name, mt7927source.Manifest)
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	descriptor := filepath.Join(root, "internal", "tools", name, mt7927source.Manifest)
	if err := os.MkdirAll(filepath.Dir(descriptor), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(descriptor, data, 0600); err != nil {
		t.Fatal(err)
	}
	msg := &governancev1.InstallDependencyRequest{
		Scenario: name, Surface: "host-tool", Ecosystem: "source",
		PackageName: name, Version: version, ExactVersion: true,
	}
	installer := &recordingInstaller{}
	broker := &testDiskJobBroker{}
	h := &connectHandler{
		scenariosDir: func() string { return filepath.Join(root, "scenarios") },
		installer:    installer,
		diskBroker:   broker,
	}
	ctx := context.Background()
	writeRegistry(t, root, `{"records":[]}`)
	response, err := h.InstallDependency(ctx, connect.NewRequest(msg))
	if err != nil || !response.Msg.GetBlocked() || response.Msg.GetVerdict() != "unrecorded" {
		t.Fatalf("unrecorded response = %v, %v", response, err)
	}
	record := fmt.Sprintf(`{"records":[{"ecosystem":"source","package_name":%q,"version_range":%q,"range_policy":"exact","state":"approved","allowed_scenarios":[%q],"rationale":"Isolated reviewed fixture"}]}`, name, version, name)
	writeRegistry(t, root, record)
	response, err = h.InstallDependency(ctx, connect.NewRequest(msg))
	if err != nil || response.Msg.GetBlocked() || !response.Msg.GetDryRun() ||
		!strings.Contains(response.Msg.GetMessage(), "stage verified data") || installer.called || len(broker.requests) != 0 {
		t.Fatalf("stage preview = %v, %v", response, err)
	}
	msg.Apply = true
	response, err = h.InstallDependency(ctx, connect.NewRequest(msg))
	if err != nil || response.Msg.GetBlocked() || !response.Msg.GetInstalled() ||
		!installer.called || installer.plan.Profile.LifecycleMode != installgateway.MT7927SourceLifecycle {
		t.Fatalf("stage apply = %v, %v", response, err)
	}
	if len(broker.requests) != 1 || len(broker.requests[0].Paths) != 1 ||
		broker.requests[0].Paths[0].Role != "wireless-source-staging" {
		t.Fatalf("stage disk admission requests = %+v", broker.requests)
	}
}
