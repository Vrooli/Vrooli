// Package plans is the structured-plan SSOT domain. It owns the first-class
// Plan + Phase record (see docs/concepts/PLAN-MODEL.md): CRUD, the deterministic
// markdown *view* (rendered from the record, never parsed back), lifecycle
// status (COMPUTED from the phase-status set), the supersession/dependency graph
// (via content_hash + plan_edges), per-surface templates, and the plan-source
// resolver (canonical write under the ~/.vrooli home store + import/migrate from
// the hygiene-blessed fallback locations).
//
// Layering mirrors the canonical domain pattern:
//
//	handler → Service → Repository
//	            ↑           ↑ (faked in tests)
//	        (proto edge)  owned home store (SQLite)
//
// The proto wire types live one floor up (handlers/plans) and never import this
// package; the handler is the only translation point (api-steer §7). Authoring,
// execution and validation operate *on* plans but live in their own domains.
package plans

import "plan-manager/internal/planmodel"

// PlanStatus is the lifecycle state of a plan. COMPUTED from the phase-status
// set plus lifecycle actions; never free-text edited.
type PlanStatus = planmodel.PlanStatus

const (
	PlanStatusDraft    = planmodel.PlanStatusDraft
	PlanStatusActive   = planmodel.PlanStatusActive
	PlanStatusComplete = planmodel.PlanStatusComplete
	PlanStatusArchived = planmodel.PlanStatusArchived
)

// PhaseStatus is the state of a single first-class phase.
type PhaseStatus = planmodel.PhaseStatus

const (
	PhaseStatusTodo    = planmodel.PhaseStatusTodo
	PhaseStatusActive  = planmodel.PhaseStatusActive
	PhaseStatusDone    = planmodel.PhaseStatusDone
	PhaseStatusBlocked = planmodel.PhaseStatusBlocked
)

// StalenessTier is the computed validity of referenced code since authoring.
type StalenessTier = planmodel.StalenessTier

const (
	StalenessFresh           = planmodel.StalenessFresh
	StalenessLightlyStale    = planmodel.StalenessLightlyStale
	StalenessDefinitelyStale = planmodel.StalenessDefinitelyStale
	StalenessUnknown         = planmodel.StalenessUnknown
)

// ReferenceKind is the machine-readable locator family.
type ReferenceKind = planmodel.ReferenceKind

const (
	ReferenceCode = planmodel.ReferenceCode
	ReferenceReq  = planmodel.ReferenceReq
	ReferenceDoc  = planmodel.ReferenceDoc
)

// ReferenceResolution is whether a reference resolves against code-facts.
type ReferenceResolution = planmodel.ReferenceResolution

const (
	ResolutionUnspecified = planmodel.ResolutionUnspecified
	ResolutionResolved    = planmodel.ResolutionResolved
	ResolutionUnresolved  = planmodel.ResolutionUnresolved
	ResolutionFuture      = planmodel.ResolutionFuture
	ResolutionMissing     = planmodel.ResolutionMissing
)

// Reference is one connected-code locator on a plan or phase. kind/target/future
// are AUTHORED; resolution/staleness/change_factor are filled by the validation
// domain.
type Reference = planmodel.Reference

// RegressionAnchor is the "before" anchor captured prior to changes. AUTO-FILLED
// by the authoring wizard (delegating to git-control-tower).
type RegressionAnchor = planmodel.RegressionAnchor

// Phase is a first-class phase. order/title/intent/required_reading/reminders/
// acceptance/references are AUTHORED; baseline_scope/status are COMPUTED.
// decisions/findings/last_validation are owned by other domains and are joined
// in at their read paths (this domain stores only the structural fields).
type Phase = planmodel.Phase

// Plan is the top-level structured record.
type Plan = planmodel.Plan

// PlanEdge is one supersession/dependency edge between two plans (the plan
// graph).
type PlanEdge = planmodel.PlanEdge

const (
	EdgeKindSupersedes = planmodel.EdgeKindSupersedes
	EdgeKindDependsOn  = planmodel.EdgeKindDependsOn
)

// PlanTemplate is a per-surface starter plan that pre-scaffolds the phase
// skeleton so a plan starts with the right shape.
type PlanTemplate = planmodel.PlanTemplate

// ListFilter narrows ListPlans. A zero value matches all non-archived plans.
type ListFilter = planmodel.ListFilter
