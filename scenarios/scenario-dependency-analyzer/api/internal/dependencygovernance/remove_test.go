package dependencygovernance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/installgateway"
)

func removalFixture(t *testing.T) (*connectHandler, *testDiskJobBroker, *recordingInstaller) {
	t.Helper()
	repoRoot := t.TempDir()
	ui := filepath.Join(repoRoot, "scenarios", "demo", "ui")
	if err := os.MkdirAll(ui, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"package.json":   `{"name":"demo-ui","dependencies":{"three":"^0.170.0","yaml":"^2.0.0"},"devDependencies":{"@types/three":"^0.170.0"}}`,
		"pnpm-lock.yaml": "lockfileVersion: '9.0'\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(ui, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	installer := &recordingInstaller{}
	h := withTestDiskAdmission(t, &connectHandler{scenariosDir: func() string { return filepath.Join(repoRoot, "scenarios") }, installer: installer})
	return h, h.diskBroker.(*testDiskJobBroker), installer
}

func removeRequest(apply bool, names ...string) *connect.Request[governancev1.RemoveDependencyRequest] {
	return connect.NewRequest(&governancev1.RemoveDependencyRequest{Scenario: "demo", Surface: "ui", Ecosystem: "npm", PackageNames: names, Apply: apply})
}

func TestRemoveDependencyDryRunPreviewsWithoutRunningOrReserving(t *testing.T) {
	h, broker, installer := removalFixture(t)
	resp, err := h.RemoveDependency(context.Background(), removeRequest(false, "three", "@types/three"))
	if err != nil {
		t.Fatal(err)
	}
	msg := resp.Msg
	if !msg.GetDryRun() || msg.GetBlocked() || msg.GetVerdict() != "removable" || msg.GetRemoved() {
		t.Fatalf("dry run = %v", msg)
	}
	if !strings.Contains(msg.GetCommand(), "pnpm remove three @types/three") || !strings.Contains(msg.GetMessage(), "--apply") {
		t.Fatalf("dry run must show the exact command and how to apply: %v", msg)
	}
	if installer.called || len(broker.requests) != 0 {
		t.Fatalf("a preview ran the installer=%t or reserved disk=%d", installer.called, len(broker.requests))
	}
}

func TestRemoveDependencyApplyReservesDiskAndRemoves(t *testing.T) {
	h, broker, installer := removalFixture(t)
	resp, err := h.RemoveDependency(context.Background(), removeRequest(true, "yaml"))
	if err != nil {
		t.Fatal(err)
	}
	msg := resp.Msg
	if !msg.GetRemoved() || msg.GetBlocked() || msg.GetDryRun() || len(broker.requests) != 1 {
		t.Fatalf("apply = %v reservations=%d", msg, len(broker.requests))
	}
	if !installer.called || strings.Join(installer.plan.RemovedPackages, ",") != "yaml" {
		t.Fatalf("installer was handed %+v", installer.plan)
	}
}

func TestRemoveDependencyBlocksLikeInstallWhenAdmissionRefuses(t *testing.T) {
	h, broker, installer := removalFixture(t)
	h.diskPlanner = func(context.Context, installgateway.Resolution) (installgateway.DiskAdmissionPlan, error) {
		return installgateway.DiskAdmissionPlan{Unsupported: "pnpm could not resolve the removal of yaml in a staging copy"}, nil
	}
	var verdicts [2]string
	for i, apply := range []bool{false, true} {
		resp, err := h.RemoveDependency(context.Background(), removeRequest(apply, "yaml"))
		if err != nil {
			t.Fatal(err)
		}
		msg := resp.Msg
		if !msg.GetBlocked() || msg.GetRemoved() || !strings.Contains(msg.GetMessage(), "could not resolve the removal") || len(msg.GetNextSteps()) == 0 {
			t.Fatalf("apply=%t response = %v", apply, msg)
		}
		verdicts[i] = msg.GetVerdict()
	}
	if verdicts[0] != verdicts[1] || installer.called || len(broker.requests) != 0 {
		t.Fatalf("dry run and apply must agree and nothing may run: %v", verdicts)
	}
}

func TestRemoveDependencyRejectsUndeclaredPackagesBeforeAnyPlanning(t *testing.T) {
	h, broker, installer := removalFixture(t)
	_, err := h.RemoveDependency(context.Background(), removeRequest(true, "three", "left-pad"))
	if connect.CodeOf(err) != connect.CodeInvalidArgument || !strings.Contains(err.Error(), "left-pad") {
		t.Fatalf("err = %v", err)
	}
	if installer.called || len(broker.requests) != 0 {
		t.Fatal("nothing may run for an invalid removal")
	}
}
