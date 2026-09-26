// Package ramp adapts scenario-to-cloud to the shared delivery-ramp contract
// (packages/delivery-ramp-go): Prober (targets from the cloud deployments and
// their typed health observation), Builder (immutable release artifact set),
// Driver (one capability-profile cell executed against a deployment, with
// target-owned assertions), and Distributor (activation of an approved
// release whose effect receipt is the target's own active-release pointer).
//
// The adapters own no remote transport. They depend on narrow seams the API
// server already provides (deployment repository, health observer, plan
// executor, target pointer reader, evidence recorder) so the same adapters
// run against fakes in the conformance test and against the live server in
// production.
package ramp

import (
	"context"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/evidence"
	"scenario-to-cloud/release"
	"scenario-to-cloud/releasesvc"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

// Deployments is the deployment read seam.
type Deployments interface {
	ListDeployments(ctx context.Context, filter domain.ListFilter) ([]*domain.Deployment, error)
	GetDeployment(ctx context.Context, id string) (*domain.Deployment, error)
}

// Observer produces the typed health observation for one deployment.
type Observer interface {
	Observe(ctx context.Context, deploymentID string) (*healthv1.HealthObservation, error)
}

// ReleaseBuilder builds (or returns the identical existing) release.
type ReleaseBuilder interface {
	Build(ctx context.Context, req releasesvc.BuildRequest) (release.Release, error)
}

// ExecutionRequest asks the owner pipeline to run one profile cell (or the
// activation) for one deployment and release.
type ExecutionRequest struct {
	DeploymentID  string
	ReleaseDigest string
	// CaseID is the certification case the run proves; "PUBLISH" for the
	// activation effect a Distributor requests.
	CaseID     string
	Lane       string
	RequestKey string
	PlanDigest string
}

// ExecutionResult is what the owner pipeline reports. Assertions and receipt
// references are target-owned (operation step receipts, target receipts);
// AcceptedAt alone is dispatch acceptance and never proves the assertion.
type ExecutionResult struct {
	OperationID string
	AcceptedAt  time.Time
	CompletedAt time.Time
	// Outcome is the owner's disposition for the cell.
	Outcome     evidence.Disposition
	Reason      string
	ReceiptRefs []string
	Assertions  []string
}

// Executor runs a cell through the existing pipeline/operations API.
type Executor interface {
	Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error)
}

// TargetReader reads the target's own active-release pointer.
type TargetReader interface {
	ReadActiveRelease(ctx context.Context, deploymentID string) (evidence.TargetReceipt, error)
}

// Recorder appends evidence records.
type Recorder interface {
	AppendEvidenceRecord(ctx context.Context, rec evidence.Record) error
}
