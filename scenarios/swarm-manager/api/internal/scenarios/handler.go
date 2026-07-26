// Package scenarios provides HTTP handlers for scenario catalog management.
//
// Scenarios are sourced from the Vrooli CLI (vrooli scenario list), then enriched
// with local metadata (priority, greenfield toggle).
// This handler provides read and update access to the scenario catalog with optional
// filtering, search, and metadata management.
//
// DOC: docs/concepts/ARCHITECTURE.md#key-flows
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/SEAMS.md
//
// Related PRD targets: OT-P0-005, OT-P0-006
package scenarios

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/dispatch"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// ScenarioStatus represents the runtime state of a scenario.
type ScenarioStatus string

const (
	StatusRunning ScenarioStatus = "running"
	StatusStopped ScenarioStatus = "stopped"
	StatusError   ScenarioStatus = "error"
	StatusUnknown ScenarioStatus = "unknown"
)

var errScenarioNameRequired = errors.New("scenario name is required")

// Scenario represents a deployed application in the Vrooli ecosystem.
// [REQ:REQ-P0-006] Scenario data structure for catalog listing
// [REQ:REQ-P0-007] Includes metadata for greenfield toggle
type Scenario struct {
	Name              string                  `json:"name"`
	DisplayName       string                  `json:"displayName"`
	Description       string                  `json:"description"`
	Status            ScenarioStatus          `json:"status"`
	Priority          int                     `json:"priority"`
	CompletenessScore *int                    `json:"completenessScore,omitempty"`
	IsGreenfield      bool                    `json:"isGreenfield"`
	Tags              []string                `json:"tags"`
	Health            *ScenarioHealthSnapshot `json:"health,omitempty"`
}

// ScenarioMetadata stores editable scenario settings in a local JSON file.
// [REQ:REQ-P0-007] Persistent metadata for scenario management
type ScenarioMetadata struct {
	IsGreenfield bool `json:"isGreenfield"`
}

// Handler provides HTTP handlers for scenario operations.
type Handler struct {
	scenariosDir       string
	source             Source
	lifecycle          Lifecycle
	completeness       CompletenessSource
	health             HealthSource
	remediationCreator RemediationCreator
	campaignCreator    CampaignCreator
	campaignReader     CampaignReader
	campaignTracker    CampaignTracker
	executionQueuer    ExecutionQueuer
	eventDispatcher    dispatch.NodeDispatcher
	backlogLister      BacklogLister
	executionLister    ExecutionLister
	goalsLister        GoalsLister
}

// NewHandler creates a new scenarios handler.
// If scenariosDir is empty, it defaults to the Vrooli scenarios directory.
func NewHandler(scenariosDir string) *Handler {
	if scenariosDir == "" {
		scenariosDir = "scenarios"
	}
	return NewHandlerWithDeps(
		scenariosDir,
		NewCLIProvider(defaultCLITimeout),
		NewCLILifecycle(),
		NewSCSCompletenessSource(defaultCompletenessTimeout),
	)
}

// NewHandlerWithSource creates a scenarios handler with a custom source.
func NewHandlerWithSource(scenariosDir string, source Source) *Handler {
	return NewHandlerWithDeps(
		scenariosDir,
		source,
		NewCLILifecycle(),
		NewSCSCompletenessSource(defaultCompletenessTimeout),
	)
}

// NewHandlerWithDeps creates a scenarios handler with injected dependencies.
func NewHandlerWithDeps(scenariosDir string, source Source, lifecycle Lifecycle, completeness CompletenessSource) *Handler {
	if scenariosDir == "" {
		scenariosDir = "scenarios"
	}
	if source == nil {
		source = NewCLIProvider(defaultCLITimeout)
	}
	if lifecycle == nil {
		lifecycle = NewCLILifecycle()
	}
	if completeness == nil {
		completeness = NewSCSCompletenessSource(defaultCompletenessTimeout)
	}
	return &Handler{
		scenariosDir: scenariosDir,
		source:       source,
		lifecycle:    lifecycle,
		completeness: completeness,
	}
}

// SetExecutionQueuer sets the execution queuer for spec-sync-archive support.
func (h *Handler) SetExecutionQueuer(eq ExecutionQueuer) {
	h.executionQueuer = eq
}

// SetHealthSource installs the optional provider-owned health projection.
func (h *Handler) SetHealthSource(source HealthSource) { h.health = source }

func (h *Handler) SetRemediationCreator(creator RemediationCreator) { h.remediationCreator = creator }

// SetEventDispatcher sets an optional event dispatcher for real-time graph updates.
func (h *Handler) SetEventDispatcher(d dispatch.NodeDispatcher) {
	h.eventDispatcher = d
}

// SetBacklogLister sets the backlog lister for review queue computation.
func (h *Handler) SetBacklogLister(bl BacklogLister) {
	h.backlogLister = bl
}

// SetExecutionLister sets the execution lister for review queue computation.
func (h *Handler) SetExecutionLister(el ExecutionLister) {
	h.executionLister = el
}

// LoadAll exposes scenario listing for non-HTTP consumers.
// This keeps data access centralized in the scenarios package.
func (h *Handler) LoadAll() ([]Scenario, error) {
	return h.loadAllScenarios(context.Background())
}

// Load exposes scenario retrieval for non-HTTP consumers.
func (h *Handler) Load(name string) (Scenario, error) {
	return h.loadScenario(context.Background(), name)
}

// getReviewSummaries loads execution records and computes per-scenario review summaries.
// Returns nil if executionLister is not configured (graceful degradation).
func (h *Handler) getReviewSummaries(ctx context.Context) map[string]ScenarioReviewSummary {
	if h.executionLister == nil {
		return nil
	}
	records, err := h.executionLister.List(ctx, execution.ListFilters{})
	if err != nil {
		slog.Warn("failed to load executions for review summaries", "error", err)
		return nil
	}
	return ComputeReviewSummaries(records)
}

// reviewSummaryFor returns a pointer to the review summary for a scenario, or nil.
func reviewSummaryFor(summaries map[string]ScenarioReviewSummary, name string) *ScenarioReviewSummary {
	if summaries == nil {
		return nil
	}
	s, ok := summaries[name]
	if !ok {
		return nil
	}
	return &s
}

func scenarioToProto(s Scenario, review *ScenarioReviewSummary) *domainpb.Scenario {
	var completeness *int32
	if s.CompletenessScore != nil {
		value := int32(*s.CompletenessScore)
		completeness = &value
	}
	proto := &domainpb.Scenario{
		Name:              s.Name,
		DisplayName:       s.DisplayName,
		Description:       s.Description,
		Status:            string(s.Status),
		Priority:          int32(s.Priority),
		CompletenessScore: completeness,
		IsGreenfield:      s.IsGreenfield,
		Tags:              s.Tags,
	}
	if s.Health != nil {
		proto.Health = healthSnapshotToProto(*s.Health)
	}
	if review != nil && review.LastReviewClassification != "" {
		proto.LastReviewClassification = &review.LastReviewClassification
		ts := review.LastReviewAt.Format(time.RFC3339)
		proto.LastReviewAt = &ts
	}
	return proto
}

func healthSnapshotToProto(snapshot ScenarioHealthSnapshot) *domainpb.ScenarioHealthSnapshot {
	proto := &domainpb.ScenarioHealthSnapshot{
		EvidenceState: string(snapshot.EvidenceState),
		Phases:        make([]*domainpb.ScenarioHealthPhase, 0, len(snapshot.Phases)),
		Remediation:   make([]*domainpb.ScenarioRemediationSummary, 0, len(snapshot.Remediation)),
	}
	if snapshot.Reason != "" {
		proto.Reason = &snapshot.Reason
	}
	if snapshot.SourceRunID != "" {
		proto.SourceRunId = &snapshot.SourceRunID
	}
	if snapshot.ObservedAt != "" {
		proto.ObservedAt = &snapshot.ObservedAt
	}
	if snapshot.Freshness != "" {
		proto.Freshness = &snapshot.Freshness
	}
	if snapshot.Verdict != "" {
		proto.Verdict = &snapshot.Verdict
	}
	for _, phase := range snapshot.Phases {
		phaseProto := &domainpb.ScenarioHealthPhase{
			Phase:             phase.Phase,
			BlockingCodes:     append([]string(nil), phase.BlockingCodes...),
			RemediationTopics: append([]string(nil), phase.RemediationTopics...),
		}
		if phase.Label != "" {
			phaseProto.Label = &phase.Label
		}
		if phase.Verdict != "" {
			phaseProto.Verdict = &phase.Verdict
		}
		if phase.CurrentRung != "" {
			phaseProto.CurrentRung = &phase.CurrentRung
		}
		if phase.NextRung != "" {
			phaseProto.NextRung = &phase.NextRung
		}
		if phase.PriorityCapabilityID != "" {
			phaseProto.PriorityCapabilityId = &phase.PriorityCapabilityID
		}
		if phase.PriorityCapabilityLabel != "" {
			phaseProto.PriorityCapabilityLabel = &phase.PriorityCapabilityLabel
		}
		proto.Phases = append(proto.Phases, phaseProto)
	}
	for _, remediation := range snapshot.Remediation {
		remediationProto := &domainpb.ScenarioRemediationSummary{
			Fingerprint: remediation.Fingerprint,
			State:       remediation.State,
		}
		if remediation.WorkRef != "" {
			remediationProto.WorkRef = &remediation.WorkRef
		}
		if remediation.UpdatedAt != "" {
			remediationProto.UpdatedAt = &remediation.UpdatedAt
		}
		proto.Remediation = append(proto.Remediation, remediationProto)
	}
	return proto
}

// RegisterRoutes registers the scenarios API routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// review-queue must be registered before the {name} wildcard to avoid capture.
	r.HandleFunc("/api/v1/scenarios/review-queue", h.ReviewQueue).Methods("GET")
	r.HandleFunc("/api/v1/scenarios", h.List).Methods("GET")
	// context must be registered before {name} catch-all so gorilla/mux
	// does not route /scenarios/foo/context to the Get handler.
	r.HandleFunc("/api/v1/scenarios/{name}/context", h.GetContext).Methods("GET")
	r.HandleFunc("/api/v1/scenarios/{name}/remediation/preview", h.PreviewRemediation).Methods("POST")
	r.HandleFunc("/api/v1/scenarios/{name}/remediation/apply", h.ApplyRemediation).Methods("POST")
	r.HandleFunc("/api/v1/scenarios/{name}/maturity-campaign/preview", h.PreviewMaturityCampaign).Methods("POST")
	r.HandleFunc("/api/v1/scenarios/{name}/maturity-campaign/apply", h.ApplyMaturityCampaign).Methods("POST")
	r.HandleFunc("/api/v1/scenarios/{name}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/scenarios/{name}", h.UpdateMetadata).Methods("PATCH")
	r.HandleFunc("/api/v1/scenarios/{name}", h.Delete).Methods("DELETE")
	r.HandleFunc("/api/v1/scenarios/{name}/files", h.ListFiles).Methods("GET")
	r.HandleFunc("/api/v1/scenarios/{name}/spec-sync-archive", h.SpecSyncArchive).Methods("POST")
	r.HandleFunc("/api/v1/scenarios/{name}/start", h.Start).Methods("POST")
	r.HandleFunc("/api/v1/scenarios/{name}/stop", h.Stop).Methods("POST")
	r.HandleFunc("/api/v1/scenarios/{name}/restart", h.Restart).Methods("POST")
}

type RemediationCreator interface {
	Create(backlog.BacklogItem, backlog.CreationContext) error
}

type CampaignCreator interface {
	Create(goals.CreateRequest) (*goals.GoalWithScope, error)
}

type CampaignReader interface {
	Get(string) (*goals.GoalWithScope, error)
}

// CampaignTracker is an optional Architecture Cartographer seam. A nil seam is
// explicitly unavailable; only a successful response yields a tracker ref.
type CampaignTracker interface {
	ReconcileCampaign(context.Context, MaturityCampaignProposal) (string, error)
}

func (h *Handler) SetCampaignTracker(tracker CampaignTracker) { h.campaignTracker = tracker }

func (h *Handler) SetCampaignCreator(creator CampaignCreator) {
	h.campaignCreator = creator
	if reader, ok := creator.(CampaignReader); ok {
		h.campaignReader = reader
	}
}

func campaignGoalName(fingerprint string) string {
	return "scenario-maturity-" + strings.TrimPrefix(fingerprint, "smc:")[:16]
}

func campaignProposalToProto(target *apipb.ScenarioMaturityCampaignTarget, proposal MaturityCampaignProposal) *apipb.ScenarioMaturityCampaignProposal {
	response := &apipb.ScenarioMaturityCampaignProposal{Target: target, Fingerprint: proposal.Fingerprint, Title: proposal.Title, Description: proposal.Description, AcceptanceCriteria: proposal.AcceptanceCriteria, DeclaredWorkflow: proposal.DeclaredWorkflow, TrackerAvailability: proposal.TrackerAvailability}
	if proposal.TrackerRef != "" {
		response.TrackerRef = &proposal.TrackerRef
	}
	return response
}

func campaignTargetFromProto(target *apipb.ScenarioMaturityCampaignTarget) MaturityCampaignTarget {
	return MaturityCampaignTarget{Scenario: target.GetScenarioName(), Target: target.GetMaturityTarget(), ProviderPhases: target.GetProviderPhases()}
}

func (h *Handler) PreviewMaturityCampaign(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var req apipb.PreviewScenarioMaturityCampaignRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil || !httputil.ValidateProtoRequest(w, "[scenarios] maturity campaign preview", "invalid request body", &req) {
		return
	}
	if req.Target.GetScenarioName() != name {
		apierr.MapError(w, "[scenarios] maturity campaign preview", apierr.BadRequest("target scenario must match the route"))
		return
	}
	scenario, err := h.loadScenario(r.Context(), name)
	if err != nil || scenario.Health == nil {
		apierr.MapError(w, "[scenarios] maturity campaign preview", apierr.Conflict("current Test Genie health evidence is unavailable"))
		return
	}
	proposal, err := BuildMaturityCampaignProposalForTarget(*scenario.Health, campaignTargetFromProto(req.Target))
	if err != nil {
		apierr.MapError(w, "[scenarios] maturity campaign preview", apierr.Conflict("%s", err))
		return
	}
	response := &apipb.PreviewScenarioMaturityCampaignResponse{Proposal: campaignProposalToProto(req.Target, proposal)}
	if h.campaignReader != nil {
		if _, lookupErr := h.campaignReader.Get(campaignGoalName(proposal.Fingerprint)); lookupErr == nil {
			ref := "goal/" + campaignGoalName(proposal.Fingerprint)
			response.ExistingGoalRef = &ref
		}
	}
	if err := httputil.ProtoJSON(w, response); err != nil {
		apierr.MapError(w, "[scenarios] maturity campaign preview", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) ApplyMaturityCampaign(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if h.campaignCreator == nil {
		apierr.MapError(w, "[scenarios] maturity campaign apply", apierr.Unavailable("maturity campaign application is not configured"))
		return
	}
	var req apipb.ApplyScenarioMaturityCampaignRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil || !httputil.ValidateProtoRequest(w, "[scenarios] maturity campaign apply", "invalid request body", &req) {
		return
	}
	if req.Target.GetScenarioName() != name {
		apierr.MapError(w, "[scenarios] maturity campaign apply", apierr.BadRequest("target scenario must match the route"))
		return
	}
	scenario, err := h.loadScenario(r.Context(), name)
	if err != nil || scenario.Health == nil {
		apierr.MapError(w, "[scenarios] maturity campaign apply", apierr.Conflict("current Test Genie health evidence is unavailable"))
		return
	}
	proposal, err := BuildMaturityCampaignProposalForTarget(*scenario.Health, campaignTargetFromProto(req.Target))
	if err != nil {
		apierr.MapError(w, "[scenarios] maturity campaign apply", apierr.Conflict("%s", err))
		return
	}
	if req.Fingerprint != proposal.Fingerprint {
		apierr.MapError(w, "[scenarios] maturity campaign apply", apierr.Conflict("preview fingerprint no longer matches current evidence"))
		return
	}
	goalName := campaignGoalName(proposal.Fingerprint)
	_, err = h.campaignCreator.Create(goals.CreateRequest{Name: goalName, Title: proposal.Title, Description: proposal.Description + "\n\nAcceptance:\n- " + strings.Join(proposal.AcceptanceCriteria, "\n- ") + "\n\nDeclared workflow: " + proposal.DeclaredWorkflow + "\nCampaign tracker: " + proposal.TrackerAvailability, Priority: 3})
	created := err == nil
	if err != nil && !(errors.Is(err, goals.ErrValidation) && strings.Contains(err.Error(), "already exists")) {
		apierr.MapError(w, "[scenarios] maturity campaign apply", apierr.Internal("failed to create governed maturity goal"))
		return
	}
	if created && h.campaignTracker != nil {
		if ref, trackerErr := h.campaignTracker.ReconcileCampaign(r.Context(), proposal); trackerErr == nil && strings.TrimSpace(ref) != "" {
			proposal.TrackerAvailability, proposal.TrackerRef = "available", strings.TrimSpace(ref)
		} else if trackerErr != nil {
			proposal.TrackerAvailability = "unavailable: Architecture Cartographer could not reconcile this campaign"
		}
	}
	response := &apipb.ApplyScenarioMaturityCampaignResponse{Proposal: campaignProposalToProto(req.Target, proposal), GoalRef: "goal/" + goalName, Created: created, TrackerAvailability: proposal.TrackerAvailability}
	if proposal.TrackerRef != "" {
		response.TrackerRef = &proposal.TrackerRef
	}
	if err := httputil.ProtoJSON(w, response); err != nil {
		apierr.MapError(w, "[scenarios] maturity campaign apply", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) ApplyRemediation(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if h.remediationCreator == nil {
		apierr.MapError(w, "[scenarios] remediation apply", apierr.Unavailable("remediation application is not configured"))
		return
	}
	var req apipb.ApplyScenarioRemediationRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		apierr.MapError(w, "[scenarios] remediation apply", apierr.BadRequest("invalid request body"))
		return
	}
	if !httputil.ValidateProtoRequest(w, "[scenarios] remediation apply", "invalid request body", &req) {
		return
	}
	if req.Target.GetScenarioName() != name {
		apierr.MapError(w, "[scenarios] remediation apply", apierr.BadRequest("target scenario must match the route"))
		return
	}
	scenario, err := h.loadScenario(r.Context(), name)
	if err != nil || scenario.Health == nil {
		apierr.MapError(w, "[scenarios] remediation apply", apierr.Conflict("current Test Genie health evidence is unavailable"))
		return
	}
	proposal, err := BuildPhaseRemediationProposal(*scenario.Health, RemediationTarget{Scenario: name, ProviderPhase: req.Target.GetProviderPhase(), CapabilityID: req.Target.GetCapabilityId()}, "manual")
	if err != nil {
		apierr.MapError(w, "[scenarios] remediation apply", apierr.Conflict("%s", err))
		return
	}
	if req.Fingerprint != proposal.Fingerprint {
		apierr.MapError(w, "[scenarios] remediation apply", apierr.Conflict("preview fingerprint no longer matches current evidence"))
		return
	}
	item := backlog.BacklogItem{Name: RemediationItemName(proposal.Fingerprint), Title: proposal.Title, Description: proposal.Description + "\n\nAcceptance:\n- " + strings.Join(proposal.AcceptanceCriteria, "\n- "), Status: backlog.StatusBacklog, Kind: backlog.KindFix, Priority: 3, Tags: []string{"scenario-health-remediation"}, FindingRef: proposal.Fingerprint, SuggestedSkills: proposal.RecommendedWorkflows, AcceptanceAllow: proposal.AcceptanceAllow}
	err = h.remediationCreator.Create(item, backlog.CreationContext{Context: r.Context(), Source: backlog.SourceProposal, DecidedBy: "operator", Entrypoint: "scenario.remediation.apply"})
	created := err == nil
	if err != nil && !errors.Is(err, backlog.ErrItemExists) {
		apierr.MapError(w, "[scenarios] remediation apply", apierr.Internal("failed to create governed remediation work"))
		return
	}
	response := &apipb.ApplyScenarioRemediationResponse{Proposal: &apipb.ScenarioRemediationProposal{Target: req.Target, Fingerprint: proposal.Fingerprint, Provenance: proposal.Provenance, Title: proposal.Title, Description: proposal.Description, AcceptanceCriteria: proposal.AcceptanceCriteria, AcceptanceAllow: proposal.AcceptanceAllow, RecommendedWorkflows: proposal.RecommendedWorkflows}, WorkRef: string(backlog.KindFix) + "/" + item.Name, Created: created}
	if err := httputil.ProtoJSON(w, response); err != nil {
		apierr.MapError(w, "[scenarios] remediation apply", apierr.Internal("failed to encode response"))
	}
}

func RemediationItemName(fingerprint string) string {
	return "scenario-remediation-" + strings.TrimPrefix(fingerprint, "srh:")[:16]
}

// PreviewRemediation builds a reviewable phase proposal without mutating
// backlog, goals, sessions, or provider evidence.
func (h *Handler) PreviewRemediation(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var req apipb.PreviewScenarioRemediationRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		apierr.MapError(w, "[scenarios] remediation preview", apierr.BadRequest("invalid request body"))
		return
	}
	if !httputil.ValidateProtoRequest(w, "[scenarios] remediation preview", "invalid request body", &req) {
		return
	}
	if req.Target.GetScenarioName() != name {
		apierr.MapError(w, "[scenarios] remediation preview", apierr.BadRequest("target scenario must match the route"))
		return
	}
	scenario, err := h.loadScenario(r.Context(), name)
	if err != nil {
		apierr.MapError(w, "[scenarios] remediation preview", apierr.NotFound("scenario not found"))
		return
	}
	if scenario.Health == nil {
		apierr.MapError(w, "[scenarios] remediation preview", apierr.Conflict("Test Genie health evidence is unavailable"))
		return
	}
	proposal, err := BuildPhaseRemediationProposal(*scenario.Health, RemediationTarget{Scenario: req.Target.GetScenarioName(), ProviderPhase: req.Target.GetProviderPhase(), CapabilityID: req.Target.GetCapabilityId()}, "manual")
	if err != nil {
		apierr.MapError(w, "[scenarios] remediation preview", apierr.Conflict("%s", err))
		return
	}
	response := &apipb.PreviewScenarioRemediationResponse{Proposal: &apipb.ScenarioRemediationProposal{Target: req.Target, Fingerprint: proposal.Fingerprint, Provenance: proposal.Provenance, Title: proposal.Title, Description: proposal.Description, AcceptanceCriteria: proposal.AcceptanceCriteria, AcceptanceAllow: proposal.AcceptanceAllow, RecommendedWorkflows: proposal.RecommendedWorkflows}}
	for _, existing := range scenario.Health.Remediation {
		if existing.Fingerprint == proposal.Fingerprint {
			response.Existing = &domainpb.ScenarioRemediationSummary{Fingerprint: existing.Fingerprint, State: existing.State}
			break
		}
	}
	if err := httputil.ProtoJSON(w, response); err != nil {
		apierr.MapError(w, "[scenarios] remediation preview", apierr.Internal("failed to encode response"))
	}
}

// List returns all scenarios with optional search and filter parameters.
// [REQ:REQ-P0-006] GET /api/v1/scenarios endpoint
//
// Query parameters:
//   - search: Filter by name or description (case-insensitive)
//   - status: Filter by status (running, stopped, error, unknown)
//   - tags: Filter by tags (comma-separated)
//   - sort: Sort field (priority, name, displayName) - default: priority
//   - order: Sort order (asc, desc) - default: asc for priority, asc for name
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	scenarios, err := h.loadAllScenarios(r.Context())
	if err != nil {
		apierr.MapError(w, "[scenarios] list", apierr.Internal("failed to load scenarios"))
		return
	}

	// Extract query params
	query := r.URL.Query()
	search := strings.ToLower(query.Get("search"))
	status := query.Get("status")
	tagsParam := query.Get("tags")
	sortField := query.Get("sort")
	sortOrder := query.Get("order")

	// Apply filters
	scenarios = h.filterScenarios(scenarios, search, status, tagsParam)

	// Sort scenarios
	h.sortScenarios(scenarios, sortField, sortOrder)

	slog.Info("listing scenarios", "count", len(scenarios), "search", search, "status", status, "tags", tagsParam)
	summaries := h.getReviewSummaries(r.Context())
	protoScenarios := make([]*domainpb.Scenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		protoScenarios = append(protoScenarios, scenarioToProto(scenario, reviewSummaryFor(summaries, scenario.Name)))
	}
	resp := &apipb.ListScenariosResponse{Scenarios: protoScenarios}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[scenarios] list", apierr.Internal("failed to encode response"))
	}
}

// filterScenarios applies search, status, and tag filters to scenarios.
func (h *Handler) filterScenarios(scenarios []Scenario, search, status, tagsParam string) []Scenario {
	// Apply search filter
	if search != "" {
		var filtered []Scenario
		for _, s := range scenarios {
			if matchesSearch(s, search) {
				filtered = append(filtered, s)
			}
		}
		scenarios = filtered
	}

	// Apply status filter
	if status != "" {
		var filtered []Scenario
		for _, s := range scenarios {
			if string(s.Status) == status {
				filtered = append(filtered, s)
			}
		}
		scenarios = filtered
	}

	// Apply tags filter
	if tagsParam != "" {
		filterTags := strings.Split(tagsParam, ",")
		var filtered []Scenario
		for _, s := range scenarios {
			if hasAnyTag(s.Tags, filterTags) {
				filtered = append(filtered, s)
			}
		}
		scenarios = filtered
	}

	return scenarios
}

// matchesSearch checks if a scenario matches a search term.
func matchesSearch(s Scenario, search string) bool {
	return strings.Contains(strings.ToLower(s.Name), search) ||
		strings.Contains(strings.ToLower(s.DisplayName), search) ||
		strings.Contains(strings.ToLower(s.Description), search)
}

// sortScenarios sorts scenarios by the specified field and order.
func (h *Handler) sortScenarios(scenarios []Scenario, sortField, sortOrder string) {
	if sortField == "" {
		sortField = "priority"
	}
	if sortOrder == "" {
		sortOrder = "asc"
	}

	sort.Slice(scenarios, func(i, j int) bool {
		var less bool
		switch sortField {
		case "name":
			less = scenarios[i].Name < scenarios[j].Name
		case "displayName":
			less = scenarios[i].DisplayName < scenarios[j].DisplayName
		default: // priority
			if scenarios[i].Priority != scenarios[j].Priority {
				less = scenarios[i].Priority < scenarios[j].Priority
			} else {
				less = scenarios[i].Name < scenarios[j].Name
			}
		}
		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

// Get returns a single scenario by name.
// [REQ:REQ-P0-006] GET /api/v1/scenarios/{name} endpoint
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	scenario, err := h.loadScenario(r.Context(), name)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "", apierr.NotFound("scenario not found"))
			return
		}
		apierr.MapError(w, "[scenarios] get", apierr.Internal("failed to load scenario"))
		return
	}

	summaries := h.getReviewSummaries(r.Context())
	resp := &apipb.ScenarioResponse{Scenario: scenarioToProto(scenario, reviewSummaryFor(summaries, scenario.Name))}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[scenarios] get", apierr.Internal("failed to encode response"))
	}
}

// UpdateMetadata updates editable metadata for a scenario.
// [REQ:REQ-P0-007] PATCH /api/v1/scenarios/{name} endpoint for metadata management
//
// This endpoint allows toggling:
//   - isGreenfield: Whether the scenario is treated as a new project
//
// Metadata is stored in .vrooli/metadata.json within the scenario directory.
func (h *Handler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	source, found, err := h.findScenarioSource(r.Context(), name)
	if err != nil {
		apierr.MapError(w, "[scenarios] update", apierr.Internal("failed to load scenarios from CLI"))
		return
	}
	if !found {
		apierr.MapError(w, "", apierr.NotFound("scenario not found"))
		return
	}

	// Parse request body
	var req apipb.UpdateScenarioMetadataRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		apierr.MapError(w, "[scenarios] update", apierr.BadRequest("invalid request body"))
		return
	}
	if !httputil.ValidateProtoRequest(w, "[scenarios] update", "invalid request body", &req) {
		return
	}

	// Load existing metadata
	metadata, _, err := h.loadMetadata(source.Path)
	if err != nil {
		apierr.MapError(w, "[scenarios] update", apierr.Internal("failed to load metadata"))
		return
	}

	// Apply updates (partial update pattern)
	if req.IsGreenfield != nil {
		metadata.IsGreenfield = *req.IsGreenfield
	}

	// Save updated metadata
	if err := h.saveMetadata(source.Path, metadata); err != nil {
		apierr.MapError(w, "[scenarios] update", apierr.Internal("failed to save metadata"))
		return
	}

	// Return updated scenario
	scenario, err := h.loadScenarioFromSource(source)
	if err != nil {
		apierr.MapError(w, "[scenarios] update", apierr.Internal("failed to load scenario"))
		return
	}
	applyCompletenessScore(&scenario, h.getCompletenessScores(r.Context()))

	slog.Info("scenario metadata updated", "scenario", name, "isGreenfield", scenario.IsGreenfield)
	summaries := h.getReviewSummaries(r.Context())
	resp := &apipb.ScenarioResponse{Scenario: scenarioToProto(scenario, reviewSummaryFor(summaries, scenario.Name))}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[scenarios] update", apierr.Internal("failed to encode response"))
	}
}

// hasAnyTag checks if the scenario has any of the filter tags.
func hasAnyTag(scenarioTags, filterTags []string) bool {
	for _, ft := range filterTags {
		ft = strings.TrimSpace(strings.ToLower(ft))
		for _, st := range scenarioTags {
			if strings.ToLower(st) == ft {
				return true
			}
		}
	}
	return false
}
