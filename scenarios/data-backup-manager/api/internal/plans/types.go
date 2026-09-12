// Package plans is the domain-scoped home for backup plans: bindings that
// associate many targets with many destinations, carrying a schedule and a
// retention policy.
//
// Layering mirrors the canonical Vrooli per-domain pattern:
//
//	Connect handler → Service (validates, decides CRUD) → Repository (persists)
//	                      ↑                                      ↑
//	                      FakeService (handler tests)            FakeRepository (service tests)
//	                                                             real sqlite (repository tests)
//
// The proto wire types live one floor up (packages/proto/...) and never import
// this package; the handler is the only translation point.
package plans

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ProtectionTier string

const (
	TierFullPrimary       ProtectionTier = "full_primary"
	TierCriticalPrimary   ProtectionTier = "critical_primary"
	TierCriticalSecondary ProtectionTier = "critical_secondary"
)

// Plan is the internal domain shape for a backup plan. Distinct from the proto
// wire type; handlers translate at the boundary so the domain layer never
// imports proto.
type Plan struct {
	ID                                string
	Name                              string
	TargetIDs                         []string
	DestinationIDs                    []string
	Schedule                          string
	KeepLatest                        int32
	Enabled                           bool
	ProtectionTier                    ProtectionTier
	RecoveryDrillSchedule             string
	DestinationsPhysicallyIndependent bool
	SharedRiskWarnings                []string
	CreatedAt                         time.Time
	UpdatedAt                         time.Time
}

// CreateInput is the explicit DTO Service.Create accepts. Distinct from Plan so
// callers cannot pass an ID or timestamp the service has no way to honour.
type CreateInput struct {
	Name                  string
	TargetIDs             []string
	DestinationIDs        []string
	Schedule              string
	KeepLatest            int32
	Enabled               bool
	ProtectionTier        ProtectionTier
	RecoveryDrillSchedule string
	// AllowIncompleteCoverage bypasses the coverage guard: when false (default)
	// the service rejects the plan if any non-sensitive discovered durable
	// target is still unregistered.
	AllowIncompleteCoverage bool
}

// UpdateInput is the DTO for full-replace updates.
type UpdateInput struct {
	ID                    string
	Name                  string
	TargetIDs             []string
	DestinationIDs        []string
	Schedule              string
	KeepLatest            int32
	Enabled               bool
	ProtectionTier        ProtectionTier
	RecoveryDrillSchedule string
	// AllowIncompleteCoverage — see CreateInput.
	AllowIncompleteCoverage bool
}

// CoverageGuard is the seam the plans service consults before persisting a
// create/update to enforce default backup coverage. It returns the non-sensitive
// discovered durable targets that are recommended for protection but not yet
// registered. The concrete implementation lives in the composition root and is
// backed by discovery; plans never imports discovery or coverage.
//
// A nil guard disables the check (used by the scheduler/test services that never
// create plans).
type CoverageGuard interface {
	UnregisteredDefaultTargets(ctx context.Context) ([]MissingTarget, error)
}

// CriticalTargetPolicy is the narrow classification seam used to enforce
// critical-only plans. The plans domain owns the invariant; the targets
// domain remains the source of the classification.
type CriticalTargetPolicy interface {
	IsCritical(ctx context.Context, targetID string) (bool, error)
}

// CriticalDestinationPolicy validates that critical plans use destinations
// with enough separation to provide the tier they claim. The composition root
// owns filesystem/backend identity; the plans domain owns fail-closed policy.
type CriticalDestinationPolicy interface {
	Validate(ctx context.Context, tier ProtectionTier, targetIDs, destinationIDs []string) error
}

// CriticalDestinationReporter supplies a read-only explanation of the
// destination topology. Unknown physical identity remains a visible warning
// rather than being presented as proven independence.
type CriticalDestinationReporter interface {
	Assess(ctx context.Context, tier ProtectionTier, targetIDs, destinationIDs []string) (DestinationRiskReport, error)
}

type DestinationRiskReport struct {
	PhysicallyIndependent bool
	Warnings              []string
}

// MissingTarget is a non-sensitive recommended target that is not yet
// registered, surfaced in the incomplete-coverage error so operators see
// exactly what default coverage they are about to skip.
type MissingTarget struct {
	Owner   string
	Name    string
	Locator string
}

// ErrIncompleteCoverage is the typed sentinel returned when a plan would be
// created/updated while non-sensitive recommended targets remain unregistered
// and the caller did not set AllowIncompleteCoverage. Handlers translate it into
// connect.CodeFailedPrecondition.
type ErrIncompleteCoverage struct {
	Missing []MissingTarget
}

func (e ErrIncompleteCoverage) Error() string {
	names := make([]string, 0, len(e.Missing))
	for _, m := range e.Missing {
		names = append(names, m.Owner+"/"+m.Name)
	}
	return fmt.Sprintf(
		"incomplete default coverage: %d recommended target(s) not yet registered (%s); run `data-backup-manager coverage accept-defaults` or pass allow_incomplete_coverage to proceed",
		len(e.Missing), strings.Join(names, ", "),
	)
}

// ErrPlanNotFound is the typed sentinel returned by Repository.GetByID when no
// row matches. Handlers translate it into a 404 / connect.CodeNotFound.
type ErrPlanNotFound struct {
	ID string
}

func (e ErrPlanNotFound) Error() string {
	return fmt.Sprintf("plan %q not found", e.ID)
}

// ErrInvalidPlan is the typed sentinel returned by Service validation. Handlers
// translate it into a 400 / connect.CodeInvalidArgument.
type ErrInvalidPlan struct {
	Field  string
	Reason string
}

func (e ErrInvalidPlan) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
