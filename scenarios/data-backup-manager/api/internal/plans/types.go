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
	"fmt"
	"time"
)

// Plan is the internal domain shape for a backup plan. Distinct from the proto
// wire type; handlers translate at the boundary so the domain layer never
// imports proto.
type Plan struct {
	ID             string
	Name           string
	TargetIDs      []string
	DestinationIDs []string
	Schedule       string
	KeepLatest     int32
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateInput is the explicit DTO Service.Create accepts. Distinct from Plan so
// callers cannot pass an ID or timestamp the service has no way to honour.
type CreateInput struct {
	Name           string
	TargetIDs      []string
	DestinationIDs []string
	Schedule       string
	KeepLatest     int32
	Enabled        bool
}

// UpdateInput is the DTO for full-replace updates.
type UpdateInput struct {
	ID             string
	Name           string
	TargetIDs      []string
	DestinationIDs []string
	Schedule       string
	KeepLatest     int32
	Enabled        bool
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
