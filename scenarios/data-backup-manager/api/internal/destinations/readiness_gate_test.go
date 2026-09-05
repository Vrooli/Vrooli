package destinations_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/destinations"
	"data-backup-manager/internal/destinations/mocks"
	enginemocks "data-backup-manager/internal/testutil/mocks"
)

// stubReadiness returns a canned report so the gate can be exercised without a
// real filesystem.
type stubReadiness struct {
	report destinationreadiness.Report
	err    error
	calls  int
}

func (s *stubReadiness) Analyze(context.Context, destinationreadiness.AnalyzeInput) (destinationreadiness.Report, error) {
	s.calls++
	return s.report, s.err
}

func failingDriverReport() destinationreadiness.Report {
	return destinationreadiness.Report{
		OverallSeverity: destinationreadiness.SeverityFail,
		Checks: []destinationreadiness.CheckResult{{
			Code:       "filesystem_suitability",
			Severity:   destinationreadiness.SeverityFail,
			Message:    "destination is mounted with the ntfs3 driver: it faults in kernel context",
			NextAction: "remount through a userspace driver",
		}},
	}
}

func newGatedService(t *testing.T, gate destinations.ReadinessGate) (destinations.Service, *mocks.FakeBundleWriter) {
	t.Helper()
	repo := mocks.NewFakeRepository()
	eng := &enginemocks.FakeKopiaEngine{}
	bundle := mocks.NewFakeBundleWriter()
	return destinations.NewService(repo, eng, bundle, "/protected", destinations.WithReadinessGate(gate)), bundle
}

func fsInput() destinations.CreateInput {
	return destinations.CreateInput{
		Name:     "elements-local",
		Backend:  destinations.BackendFilesystem,
		Location: "/media/user/Elements/vrooli-backups",
	}
}

// The condition that produced the 2026-08-19 host panic must be refused at
// creation, not merely reported.
func TestCreateDestinationRefusesKernelFaultingDriver(t *testing.T) {
	svc, _ := newGatedService(t, &stubReadiness{report: failingDriverReport()})

	_, err := svc.CreateDestination(context.Background(), fsInput())

	if err == nil {
		t.Fatal("expected creation to be refused")
	}
	var invalid destinations.ErrInvalidDestination
	if !errors.As(err, &invalid) {
		t.Fatalf("error = %T, want ErrInvalidDestination", err)
	}
	if !strings.Contains(err.Error(), "ntfs3") {
		t.Fatalf("refusal must name the offending driver, got %q", err.Error())
	}
}

// Acknowledgement is an operator judgement call about their backup. It must not
// extend to a risk borne by the host.
func TestAcknowledgementDoesNotOverrideDriverRefusal(t *testing.T) {
	svc, _ := newGatedService(t, &stubReadiness{report: failingDriverReport()})
	in := fsInput()
	in.AcknowledgeReadinessFailure = true

	if _, err := svc.CreateDestination(context.Background(), in); err == nil {
		t.Fatal("acknowledgement must not override a kernel-faulting driver refusal")
	}
}

// Other readiness failures are judgement calls and stay overridable, so an
// operator with an unusual but deliberate setup is not locked out.
func TestAcknowledgementOverridesOtherReadinessFailures(t *testing.T) {
	report := destinationreadiness.Report{
		OverallSeverity: destinationreadiness.SeverityFail,
		Checks: []destinationreadiness.CheckResult{{
			Code:     "capacity",
			Severity: destinationreadiness.SeverityFail,
			Message:  "free space is below selected target estimate",
		}},
	}
	svc, _ := newGatedService(t, &stubReadiness{report: report})

	if _, err := svc.CreateDestination(context.Background(), fsInput()); err == nil {
		t.Fatal("an unacknowledged readiness failure must be refused")
	}

	in := fsInput()
	in.AcknowledgeReadinessFailure = true
	if _, err := svc.CreateDestination(context.Background(), in); err != nil {
		t.Fatalf("acknowledged readiness failure should proceed, got %v", err)
	}
}

// A refusal must not leave a half-provisioned bundle on a drive just declared
// unfit for backups.
func TestRefusedDestinationWritesNothingToTheDrive(t *testing.T) {
	svc, bundle := newGatedService(t, &stubReadiness{report: failingDriverReport()})

	if _, err := svc.CreateDestination(context.Background(), fsInput()); err == nil {
		t.Fatal("expected refusal")
	}
	if len(bundle.Prepared) != 0 {
		t.Fatalf("bundle was prepared %d times on a refused destination", len(bundle.Prepared))
	}
}

// A readiness probe that errors is not a pass. An unprovable destination stays
// out of the catalog.
func TestUnevaluableReadinessRefusesCreation(t *testing.T) {
	svc, _ := newGatedService(t, &stubReadiness{err: errors.New("probe failed")})

	if _, err := svc.CreateDestination(context.Background(), fsInput()); err == nil {
		t.Fatal("readiness that could not be evaluated must not admit the destination")
	}
}

// S3 destinations have no mounted volume to evaluate; the gate must not run.
func TestGateSkipsNonFilesystemBackends(t *testing.T) {
	stub := &stubReadiness{report: failingDriverReport()}
	svc, _ := newGatedService(t, stub)

	in := destinations.CreateInput{Name: "offsite", Backend: destinations.BackendS3, Location: "bucket/prefix"}
	if _, err := svc.CreateDestination(context.Background(), in); err != nil {
		t.Fatalf("S3 creation should not consult readiness, got %v", err)
	}
	if stub.calls != 0 {
		t.Fatalf("readiness consulted %d times for an S3 destination", stub.calls)
	}
}
