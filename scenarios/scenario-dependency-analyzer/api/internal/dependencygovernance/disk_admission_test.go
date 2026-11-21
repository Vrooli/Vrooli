package dependencygovernance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/packages/capacity"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/installgateway"
)

type testDiskJobBroker struct {
	reserveErr error
	requests   []capacity.DiskJobRequest
}

func (b *testDiskJobBroker) ReserveDiskJob(_ context.Context, request capacity.DiskJobRequest) (capacity.DiskJobLease, error) {
	b.requests = append(b.requests, request)
	if b.reserveErr != nil {
		return nil, b.reserveErr
	}
	return &testDiskJobLease{receipt: capacity.DiskJobReceipt{
		ID: "test-disk-job", OwnerID: request.OwnerID, JobID: request.JobID,
		Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}}, nil
}

type testDiskJobLease struct {
	receipt capacity.DiskJobReceipt
}

func (l *testDiskJobLease) Heartbeat(context.Context) error { return nil }
func (l *testDiskJobLease) Observe(context.Context) (capacity.DiskJobHeadroom, error) {
	return capacity.DiskJobHeadroom{LeaseID: l.receipt.ID, Status: l.receipt.Status}, nil
}
func (l *testDiskJobLease) Extend(_ context.Context, _ []capacity.DiskJobPath) (capacity.DiskJobReceipt, error) {
	return l.receipt, nil
}
func (l *testDiskJobLease) SamplePath(_ context.Context, path string, _, _ int64) (capacity.DiskJobReceipt, error) {
	return l.receipt, fmt.Errorf("unexpected fixture disk path sample %q", path)
}
func (l *testDiskJobLease) Complete(context.Context) (capacity.DiskJobReceipt, error) {
	l.receipt.Status = "completed"
	l.receipt.CleanupVerified = true
	return l.receipt, nil
}
func (l *testDiskJobLease) Release(context.Context) error        { return nil }
func (l *testDiskJobLease) Reservation() capacity.DiskJobReceipt { return l.receipt }

func withTestDiskAdmission(t *testing.T, h *connectHandler) *connectHandler {
	t.Helper()
	h.diskBroker = &testDiskJobBroker{}
	h.diskPlanner = func(_ context.Context, _ installgateway.Resolution) (installgateway.DiskAdmissionPlan, error) {
		return installgateway.DiskAdmissionPlan{
			Paths:     []installgateway.DiskPathRequirement{{Path: t.TempDir(), Role: "test-target", PeakBytes: 1}},
			CachePath: t.TempDir(), Manager: "test",
		}, nil
	}
	return h
}

func TestPipYTDLPApplyFailsClosedWithoutSurfaceOwnedEnvironment(t *testing.T) {
	repo := t.TempDir()
	writeRegistry(t, repo, `{"records":[{"ecosystem":"pip","package_name":"yt-dlp","version_range":"2026.*","state":"approved","rationale":"Local research fixture.","allowed_surfaces":["api"]}]}`)
	apiRoot := filepath.Join(repo, "scenarios", "demo", "api")
	if err := os.MkdirAll(apiRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	requirements := filepath.Join(apiRoot, "requirements.txt")
	if err := os.WriteFile(requirements, []byte("# empty fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	installer := &recordingInstaller{}
	h := &connectHandler{scenariosDir: func() string { return filepath.Join(repo, "scenarios") }, installer: installer, diskBroker: &testDiskJobBroker{}}
	response, err := h.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{
		Scenario: "demo", Surface: "api", Ecosystem: "pip", PackageName: "yt-dlp", Apply: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !response.Msg.GetBlocked() || installer.called {
		t.Fatalf("pip install was not refused before the installer: blocked=%t installer=%t response=%v", response.Msg.GetBlocked(), installer.called, response.Msg)
	}
	if response.Msg.GetVerdict() != "disk_admission_denied" || !strings.Contains(response.Msg.GetMessage(), "surface-scoped environment") {
		t.Fatalf("refusal did not explain the uncovered pip environment: %v", response.Msg)
	}
	if got, err := os.ReadFile(requirements); err != nil || string(got) != "# empty fixture\n" {
		t.Fatalf("requirements changed before disk admission: %q, %v", got, err)
	}
}
