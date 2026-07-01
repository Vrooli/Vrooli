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

// RelevantContextKind classifies one setup item a fresh/resumed agent should
// load, inspect, or run before acting on a plan or phase.
type RelevantContextKind = planmodel.RelevantContextKind

const (
	RelevantContextSkill   = planmodel.RelevantContextSkill
	RelevantContextDoc     = planmodel.RelevantContextDoc
	RelevantContextCommand = planmodel.RelevantContextCommand
	RelevantContextSearch  = planmodel.RelevantContextSearch
	RelevantContextCodeRef = planmodel.RelevantContextCodeRef
	RelevantContextReqRef  = planmodel.RelevantContextReqRef
	RelevantContextNote    = planmodel.RelevantContextNote
)

// RelevantContextScope says whether an item applies plan-wide or to one phase.
type RelevantContextScope = planmodel.RelevantContextScope

const (
	RelevantContextScopeGlobal = planmodel.RelevantContextScopeGlobal
	RelevantContextScopePhase  = planmodel.RelevantContextScopePhase
)

// RelevantContextRepeatPolicy says when execution should re-emit the item.
type RelevantContextRepeatPolicy = planmodel.RelevantContextRepeatPolicy

const (
	RelevantContextOncePerExecution = planmodel.RelevantContextOncePerExecution
	RelevantContextOnResume         = planmodel.RelevantContextOnResume
	RelevantContextEveryPhase       = planmodel.RelevantContextEveryPhase
	RelevantContextPhaseEntry       = planmodel.RelevantContextPhaseEntry
	RelevantContextAsNeeded         = planmodel.RelevantContextAsNeeded
)

// RelevantContextSource captures how the context item entered the plan.
type RelevantContextSource = planmodel.RelevantContextSource

const (
	RelevantContextSourceAuthored   = planmodel.RelevantContextSourceAuthored
	RelevantContextSourceDiscovered = planmodel.RelevantContextSourceDiscovered
	RelevantContextSourceMigrated   = planmodel.RelevantContextSourceMigrated
	RelevantContextSourceAutofilled = planmodel.RelevantContextSourceAutofilled
)

// RelevantContextStatus records whether the item is usable or degraded.
type RelevantContextStatus = planmodel.RelevantContextStatus

const (
	RelevantContextStatusReady      = planmodel.RelevantContextStatusReady
	RelevantContextStatusDegraded   = planmodel.RelevantContextStatusDegraded
	RelevantContextStatusUnresolved = planmodel.RelevantContextStatusUnresolved
)

// Reference is one connected-code locator on a plan or phase. kind/target/future
// are AUTHORED; resolution/staleness/change_factor are filled by the validation
// domain.
type Reference = planmodel.Reference

// RelevantContextItem is the execution-facing setup contract.
type RelevantContextItem = planmodel.RelevantContextItem

// RegressionAnchor is the "before" anchor captured prior to changes. AUTO-FILLED
// by the authoring wizard (delegating to git-control-tower).
type RegressionAnchor = planmodel.RegressionAnchor

const (
	AnchorStrategyChangeBoundary   = planmodel.AnchorStrategyChangeBoundary
	AnchorStrategyScenarioBaseline = planmodel.AnchorStrategyScenarioBaseline
	AnchorStrategyHeadShaAllowlist = planmodel.AnchorStrategyHeadShaAllowlist
	AnchorStrategyLegacyProse      = planmodel.AnchorStrategyLegacyProse
)

// ChangeBoundary is the plan's first-class blast-radius contract
// (acceptance_allow / acceptance_deny). Scenario identity is derived from it.
type ChangeBoundary = planmodel.ChangeBoundary

// WorkPosture is the Greenfield/Brownfield stance of a plan. AUTOFILLED from
// scenario maturity (default greenfield); never agent-authored.
type WorkPosture = planmodel.WorkPosture

const (
	WorkPostureUnspecified = planmodel.WorkPostureUnspecified
	WorkPostureGreenfield  = planmodel.WorkPostureGreenfield
	WorkPostureBrownfield  = planmodel.WorkPostureBrownfield
)

// WorkPostureSource records how the posture was decided (audit/explainability).
type WorkPostureSource = planmodel.WorkPostureSource

const (
	WorkPostureSourceUnspecified      = planmodel.WorkPostureSourceUnspecified
	WorkPostureSourceDefault          = planmodel.WorkPostureSourceDefault
	WorkPostureSourceServiceMaturity  = planmodel.WorkPostureSourceServiceMaturity
	WorkPostureSourceExplicitOverride = planmodel.WorkPostureSourceExplicitOverride
	WorkPostureSourceImportLegacy     = planmodel.WorkPostureSourceImportLegacy
)

// RenderedMirrorStatus describes the freshness/repair state of a plan's durable
// rendered markdown projection.
type RenderedMirrorStatus = planmodel.RenderedMirrorStatus

const (
	RenderedMirrorStatusUnspecified = planmodel.RenderedMirrorStatusUnspecified
	RenderedMirrorStatusFresh       = planmodel.RenderedMirrorStatusFresh
	RenderedMirrorStatusMissing     = planmodel.RenderedMirrorStatusMissing
	RenderedMirrorStatusStale       = planmodel.RenderedMirrorStatusStale
	RenderedMirrorStatusWriteFailed = planmodel.RenderedMirrorStatusWriteFailed
	RenderedMirrorStatusUnknown     = planmodel.RenderedMirrorStatusUnknown
)

// RenderedPlanMirror is the file-addressable markdown projection metadata for a
// canonical structured plan.
type RenderedPlanMirror = planmodel.RenderedPlanMirror

// LegacySection is one imported markdown section preserved as provenance.
type LegacySection = planmodel.LegacySection

// ImportProvenance records that a plan was adopted from a legacy markdown source.
type ImportProvenance = planmodel.ImportProvenance

const (
	PreservationReasonUnmapped   = planmodel.PreservationReasonUnmapped
	OriginalFormatLegacyMarkdown = planmodel.OriginalFormatLegacyMarkdown
)

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

// WorkspaceScope anchors filesystem-scoped plan operations to a Vrooli
// workspace root instead of the Plan Manager API process cwd.
type WorkspaceScope struct {
	ID   string
	Root string
}

// ReconcileConflictPolicy controls how bulk legacy adoption handles plans that
// appear to already exist.
type ReconcileConflictPolicy string

const (
	ReconcileConflictReportOnly   ReconcileConflictPolicy = "report_only"
	ReconcileConflictSkipExisting ReconcileConflictPolicy = "skip_existing"
)

// ReconcileRequest describes one bulk mirror/adoption pass. DryRun reports the
// same item decisions without mutating SQLite or mirror files.
type ReconcileRequest struct {
	DryRun                 bool
	RepairMirrors          bool
	AdoptLegacy            bool
	IncludeArchived        bool
	IncludeArchivedLegacy  bool
	ConflictPolicy         ReconcileConflictPolicy
	Workspace              WorkspaceScope
	SourceRuntimeHomePlans bool
	SourceDocsPlans        bool
	SourceRepoPlans        bool
}

// ReconcileAction is the per-item outcome of a reconcile pass.
type ReconcileAction string

const (
	ReconcileActionUnspecified        ReconcileAction = ""
	ReconcileActionAlreadyCanonical   ReconcileAction = "already_canonical"
	ReconcileActionMirrorFresh        ReconcileAction = "mirror_fresh"
	ReconcileActionMirrorRepairNeeded ReconcileAction = "mirror_repair_needed"
	ReconcileActionMirrorRepaired     ReconcileAction = "mirror_repaired"
	ReconcileActionImportPlanned      ReconcileAction = "import_planned"
	ReconcileActionImported           ReconcileAction = "imported"
	ReconcileActionSkippedDuplicate   ReconcileAction = "skipped_duplicate"
	ReconcileActionParseFailed        ReconcileAction = "parse_failed"
	ReconcileActionConflict           ReconcileAction = "conflict"
)

// ReconcileItem reports one inspected canonical plan or legacy source.
type ReconcileItem struct {
	Action          ReconcileAction
	PlanID          string
	Slug            string
	Title           string
	SourcePath      string
	Mirror          RenderedPlanMirror
	SourceUntouched bool
	Error           string
}

// ReconcileResult is the full bulk reconcile report.
type ReconcileResult struct {
	DryRun bool
	Items  []ReconcileItem
}
