// Package assignments is the domain-scoped home for the brand↔scenario link.
//
// Layering mirrors the canonical Vrooli pattern (see internal/brands for the
// sibling domain):
//
//	Connect handler → Service (validates, resolves brand version via the
//	                  BrandResolver seam) → Repository (persist)
//	                     ↑                          ↑
//	                     FakeService (handler tests) FakeRepository (service tests)
//	                                                 Real sqlite (repository tests)
//
// types.go owns the domain entities and the typed sentinels handlers translate
// at the transport edge. The proto wire types live one floor up
// (packages/proto/...) and never import this package; the handler is the only
// translation point (api-steer §7).
package assignments

import (
	"fmt"
	"time"
)

// Assignment is the internal domain shape linking a brand to a scenario.
// Distinct from the proto wire type — handlers translate at the boundary so the
// domain layer never imports proto.
type Assignment struct {
	ID           string
	BrandID      string
	ScenarioName string
	BrandVersion int
	Elements     []string
	AppliedAt    time.Time
}

// ScenarioStatus reports whether a scenario has a brand assigned and, if so,
// which brand version and elements were applied. Mirrors the old REST
// GET /scenarios/{name}/status response shape.
type ScenarioStatus struct {
	Scenario     string
	HasBrand     bool
	BrandID      string
	BrandVersion int
	Elements     []string
	AppliedAt    time.Time
}

// StatusUnassigned returns the status for a scenario with no brand.
func StatusUnassigned(scenario string) ScenarioStatus {
	return ScenarioStatus{Scenario: scenario, HasBrand: false}
}

// StatusFromAssignment builds a status from an existing assignment.
func StatusFromAssignment(a Assignment) ScenarioStatus {
	return ScenarioStatus{
		Scenario:     a.ScenarioName,
		HasBrand:     true,
		BrandID:      a.BrandID,
		BrandVersion: a.BrandVersion,
		Elements:     a.Elements,
		AppliedAt:    a.AppliedAt,
	}
}

// AssignInput is the explicit input DTO Service.Assign accepts. The service
// resolves the brand version and stamps AppliedAt, so callers cannot pass them.
type AssignInput struct {
	BrandID      string
	ScenarioName string
	Elements     []string
}

// ErrAssignmentNotFound is the typed sentinel returned when no assignment
// matches a scenario. Handlers translate via errors.As into a Connect NotFound
// response; the service swallows it for idempotent Unassign.
type ErrAssignmentNotFound struct {
	Scenario string
}

func (e ErrAssignmentNotFound) Error() string {
	return fmt.Sprintf("no brand assigned to scenario %q", e.Scenario)
}

// ErrInvalidAssignment is the typed sentinel returned when validation fails or
// the referenced brand does not exist. Field names the offending field; Reason
// is a human-safe explanation. Handlers translate via errors.As into a Connect
// InvalidArgument response.
type ErrInvalidAssignment struct {
	Field  string
	Reason string
}

func (e ErrInvalidAssignment) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
