package plansvc

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/identity"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans"
)

func samplePlan(t *testing.T) *execplan.Plan {
	t.Helper()
	plan, err := execplan.Compile(context.Background(), execplan.CompileInputs{
		Deployment:      identity.DeploymentRef{ID: "dep-1", ScenarioID: "app", Target: identity.TargetRef{Transport: identity.TransportSSH, EnrollmentGeneration: 2}},
		DesiredRevision: 3,
		Release:         identity.ReleaseRef{Digest: "sha256:abc", ConfigurationDigest: "sha256:cfg"},
		Manifest: domain.CloudManifest{
			Target:       domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10", Workdir: "/root/Vrooli"}},
			Scenario:     domain.ManifestScenario{ID: "app"},
			Dependencies: domain.ManifestDependencies{Resources: []string{"store"}},
			Ports:        domain.ManifestPorts{"ui": 3000},
			Edge:         domain.ManifestEdge{Domain: "app.example.test", Caddy: domain.ManifestCaddy{Enabled: true}},
		},
		ArtifactPath: "/tmp/app.tar.gz",
		Observations: execplan.Observations{DeploymentRevision: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

// [REQ:STC-P0-016] The wire plan uses proto field names and round-trips to
// the same semantic digest, so REST JSON and Connect carry one identity.
func TestPlanRoundTripsThroughProtoWithSameDigest(t *testing.T) {
	plan := samplePlan(t)
	msg, err := PlanProto(plan)
	if err != nil {
		t.Fatalf("PlanProto: %v", err)
	}
	if msg.GetSchemaVersion() != "1" || msg.GetTarget().GetEnrollmentGeneration() != 2 || len(msg.GetActions()) != len(plan.Actions) {
		t.Fatalf("proto plan lost fields: %v", protojson.Format(msg))
	}
	back, err := PlanFromProto(msg)
	if err != nil {
		t.Fatalf("PlanFromProto: %v", err)
	}
	if back.MustDigest() != plan.MustDigest() {
		t.Fatalf("digest changed across the proto round trip")
	}
	preview, err := PreviewProto(execplan.Render(plan))
	if err != nil {
		t.Fatalf("PreviewProto: %v", err)
	}
	if len(preview.GetChanges()) != len(plan.Actions) || preview.GetDowntime().GetExpectedSeconds() != 60 {
		t.Fatalf("preview lost fields: %v", protojson.Format(preview))
	}
}

type fakePlanner struct {
	compiled *Compiled
	err      error
}

func (f fakePlanner) CompilePlan(context.Context, string, string, bool) (*Compiled, error) {
	return f.compiled, f.err
}

func (f fakePlanner) ApplyPlan(context.Context, string, string, string, string, bool) (*Applied, error) {
	return nil, f.err
}

func TestServiceCarriesTypedErrorsAndDigest(t *testing.T) {
	plan := samplePlan(t)
	svc := New(fakePlanner{compiled: &Compiled{Plan: plan, PlanDigest: plan.MustDigest(), Preview: execplan.Render(plan), ClosureStatus: "derived"}})
	resp, err := svc.CompilePlan(context.Background(), connect.NewRequest(&plansv1.CompilePlanRequest{DeploymentId: "dep-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetPlanDigest() != plan.MustDigest() || resp.Msg.GetSchemaVersion() != SchemaVersion {
		t.Fatalf("unexpected response %v", resp.Msg)
	}
	failing := New(fakePlanner{err: execplan.DigestMismatchError("sha256:a", "sha256:b")})
	_, err = failing.ApplyPlan(context.Background(), connect.NewRequest(&plansv1.ApplyPlanRequest{DeploymentId: "dep-1", PlanDigest: "sha256:a", RequestKey: "k"}))
	var cerr *connect.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("expected a connect error, got %v", err)
	}
	if len(cerr.Details()) == 0 {
		t.Fatalf("expected the typed error detail")
	}
	if _, err := svc.CompilePlan(context.Background(), connect.NewRequest(&plansv1.CompilePlanRequest{})); err == nil || !apierrors.Is(err, apierrors.CodeInvalidRequest) && !errors.As(err, &cerr) {
		t.Fatalf("expected invalid_request for a missing deployment id, got %v", err)
	}
}
