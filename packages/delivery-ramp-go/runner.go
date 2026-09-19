package deliveryramp

import (
	"context"
	"fmt"
	"strings"
)

// JourneyExecutionRequest is the provider-neutral input to the journey
// runner. The Driver owns installation, launch, interaction, and capture;
// the runner owns fail-closed preflight and result normalization.
type JourneyExecutionRequest struct {
	Cell     Cell
	Target   Target
	Artifact Artifact
	Plan     JourneyPlan
	Evidence JourneyEvidenceSink
	RunID    string
}

type JourneyRunner struct {
	Driver ProbingDriver
}

// ProbingDriver combines the already-declared Driver seam with an optional
// capability probe. A ramp may provide a richer implementation without the
// spine learning platform details.
type ProbingDriver interface {
	Execute(context.Context, DriverRequest) (JourneyResult, error)
}

func (r JourneyRunner) Run(ctx context.Context, request JourneyExecutionRequest) JourneyResult {
	base := JourneyResult{
		SchemaVersion: JourneySchemaVersion, EvidenceVersion: JourneyEvidenceVersion,
		SmokeTestID: request.RunID, PlanID: request.Plan.ID, Profile: request.Plan.Profile,
		Capability: request.Plan.Capability, TargetID: request.Target.ID, CellID: request.Cell.ID,
		Disposition: DispositionNotRun,
	}
	if err := validateJourneyRequest(request); err != nil {
		base.Disposition = DispositionUnavailable
		base.DegradedReason = err.Error()
		return base
	}
	if r.Driver == nil {
		base.Disposition = DispositionUnavailable
		base.DegradedReason = "journey driver is unavailable"
		return base
	}
	if ctx == nil {
		base.Disposition = DispositionUnavailable
		base.DegradedReason = "journey context is nil"
		return base
	}
	if err := ctx.Err(); err != nil {
		base.DegradedReason = err.Error()
		return base
	}
	result, err := r.Driver.Execute(ctx, DriverRequest{Cell: request.Cell, Artifact: request.Artifact, Plan: request.Plan, Evidence: request.Evidence, RunID: request.RunID})
	if err != nil {
		base.Disposition = DispositionFailed
		if ctx.Err() != nil {
			base.Disposition = DispositionNotRun
		}
		base.DegradedReason = err.Error()
		return base
	}
	if result.SchemaVersion == 0 {
		result.SchemaVersion = JourneySchemaVersion
	}
	if result.EvidenceVersion == "" {
		result.EvidenceVersion = JourneyEvidenceVersion
	}
	if result.SmokeTestID == "" {
		result.SmokeTestID = request.RunID
	}
	if result.PlanID == "" {
		result.PlanID = request.Plan.ID
	}
	if result.Profile == "" {
		result.Profile = request.Plan.Profile
	}
	if result.Capability == "" {
		result.Capability = request.Plan.Capability
	}
	if result.TargetID == "" {
		result.TargetID = request.Target.ID
	}
	if result.Disposition == "" {
		result.Disposition = DispositionUnavailable
		result.DegradedReason = "driver returned no disposition"
	}
	return result
}

func validateJourneyRequest(request JourneyExecutionRequest) error {
	if strings.TrimSpace(request.RunID) == "" {
		return fmt.Errorf("journey run id is required")
	}
	if strings.TrimSpace(request.Plan.ID) == "" || strings.TrimSpace(request.Plan.Capability) == "" {
		return fmt.Errorf("journey plan identity and capability are required")
	}
	if request.Plan.SchemaVersion != "" && request.Plan.SchemaVersion != JourneyEvidenceVersion {
		return fmt.Errorf("unsupported journey schema %q", request.Plan.SchemaVersion)
	}
	if err := request.Target.Validate(); err != nil {
		return err
	}
	if !request.Target.Supports(request.Plan.Capability) {
		return fmt.Errorf("target %q does not advertise capability %q", request.Target.ID, request.Plan.Capability)
	}
	return nil
}
