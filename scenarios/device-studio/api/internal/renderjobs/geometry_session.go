package renderjobs

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"

	"device-studio/internal/jobs"
	"github.com/google/uuid"
)

const geometrySessionOwner = "device_geometry_creation"

type GeometrySessionPolicy struct {
	Version           int        `json:"version"`
	ProjectID         string     `json:"project_id"`
	CandidateID       string     `json:"candidate_id"`
	Brief             string     `json:"brief"`
	EnvelopeMM        [3]float64 `json:"envelope_mm"`
	ReferenceMediaIDs []string   `json:"reference_media_ids"`
	FundingRoute      string     `json:"funding_route"`
	TokensPerCall     int        `json:"tokens_per_call"`
	ParentSessionID   string     `json:"parent_session_id,omitempty"`
}

type GeometrySessionChild struct {
	JobID       string     `json:"job_id"`
	Ordinal     int        `json:"ordinal"`
	State       jobs.State `json:"state"`
	Version     int64      `json:"version"`
	CandidateID string     `json:"candidate_id"`
}

type GeometrySessionSnapshot struct {
	SessionID             string                 `json:"session_id"`
	State                 jobs.RunScopeState     `json:"state"`
	Version               int64                  `json:"version"`
	Policy                GeometrySessionPolicy  `json:"policy"`
	MaxProviderCalls      int                    `json:"max_provider_calls"`
	ProviderCallsReserved int                    `json:"provider_calls_reserved"`
	MaxChildren           int                    `json:"max_children"`
	ChildrenAdmitted      int                    `json:"children_admitted"`
	Children              []GeometrySessionChild `json:"children"`
	NeedsConfirmation     bool                   `json:"needs_confirmation,omitempty"`
	ChangedScope          []string               `json:"changed_scope,omitempty"`
}

type GeometrySessionRequest struct {
	ProjectID         string     `json:"project_id"`
	CandidateID       string     `json:"candidate_id"`
	IdempotencyKey    string     `json:"idempotency_key"`
	Brief             string     `json:"brief"`
	EnvelopeMM        [3]float64 `json:"envelope_mm"`
	ReferenceMediaIDs []string   `json:"reference_media_ids"`
	FundingRoute      string     `json:"funding_route"`
	ConfirmLocalAI    bool       `json:"confirm_local_ai"`
}

type GeometrySessionRefinementRequest struct {
	GeometrySessionRequest
	CorrectionSourceJobID  string `json:"correction_source_job_id,omitempty"`
	CorrectionMode         string `json:"correction_mode,omitempty"`
	CorrectionRecipeSHA256 string `json:"correction_recipe_sha256,omitempty"`
	CorrectionFeedback     string `json:"correction_feedback,omitempty"`
	ConfirmScopeExpansion  bool   `json:"confirm_scope_expansion"`
}

func (c *Coordinator) CreateGeometrySession(ctx context.Context, req GeometrySessionRequest) (GeometrySessionSnapshot, error) {
	return c.createGeometrySession(ctx, req, "")
}

func (c *Coordinator) createGeometrySession(ctx context.Context, req GeometrySessionRequest, parent string) (GeometrySessionSnapshot, error) {
	if c.scopes == nil {
		return GeometrySessionSnapshot{}, errors.New("bounded creation sessions are unavailable")
	}
	if !req.ConfirmLocalAI {
		return GeometrySessionSnapshot{}, errors.New("confirm the disclosed local-only AI request before starting")
	}
	policy, err := canonicalGeometrySessionPolicy(req, parent)
	if err != nil {
		return GeometrySessionSnapshot{}, err
	}
	scopeID := geometrySessionID(policy.ProjectID, req.IdempotencyKey)
	if err := c.ensureGeometryScope(ctx, scopeID, policy); err != nil {
		return GeometrySessionSnapshot{}, err
	}
	snapshot, err := c.geometrySessionSnapshot(ctx, scopeID)
	if err != nil {
		return GeometrySessionSnapshot{}, err
	}
	if len(snapshot.Children) == 0 {
		if _, _, err := c.submitGeometrySessionChild(ctx, policy, scopeID, req.IdempotencyKey+"/baseline", ""); err != nil {
			return GeometrySessionSnapshot{}, err
		}
		snapshot, err = c.geometrySessionSnapshot(ctx, scopeID)
	}
	return snapshot, err
}

func (c *Coordinator) GetGeometrySession(ctx context.Context, id string) (GeometrySessionSnapshot, error) {
	if c.scopes == nil {
		return GeometrySessionSnapshot{}, errors.New("bounded creation sessions are unavailable")
	}
	return c.geometrySessionSnapshot(ctx, strings.TrimSpace(id))
}

func (c *Coordinator) RefineGeometrySession(ctx context.Context, id string, req GeometrySessionRefinementRequest) (GeometrySessionSnapshot, error) {
	if c.scopes == nil {
		return GeometrySessionSnapshot{}, errors.New("bounded creation sessions are unavailable")
	}
	current, err := c.scopes.Get(ctx, strings.TrimSpace(id))
	if err != nil || current.OwnerKind != geometrySessionOwner {
		return GeometrySessionSnapshot{}, jobs.ErrNotFound
	}
	var original GeometrySessionPolicy
	if err := json.Unmarshal(current.Policy, &original); err != nil {
		return GeometrySessionSnapshot{}, errors.New("creation session policy is unavailable")
	}
	// The public endpoint deliberately accepts no project_id on refinements.
	// Rehydrate it from the persisted parent policy before comparing scope.
	req.ProjectID = original.ProjectID
	proposed, err := canonicalGeometrySessionPolicy(req.GeometrySessionRequest, original.ParentSessionID)
	if err != nil {
		return GeometrySessionSnapshot{}, err
	}
	changed := changedGeometrySessionScope(original, proposed)
	if len(changed) > 0 {
		if !req.ConfirmScopeExpansion {
			snapshot, snapErr := c.geometrySessionSnapshot(ctx, id)
			snapshot.NeedsConfirmation, snapshot.ChangedScope = true, changed
			return snapshot, snapErr
		}
		child, createErr := c.createGeometrySession(ctx, GeometrySessionRequest{ProjectID: req.ProjectID, CandidateID: req.CandidateID, IdempotencyKey: req.IdempotencyKey, Brief: req.Brief, EnvelopeMM: req.EnvelopeMM, ReferenceMediaIDs: req.ReferenceMediaIDs, FundingRoute: req.FundingRoute, ConfirmLocalAI: req.ConfirmLocalAI}, id)
		if createErr != nil {
			return GeometrySessionSnapshot{}, createErr
		}
		return child, nil
	}
	if !req.ConfirmLocalAI {
		return GeometrySessionSnapshot{}, errors.New("confirm the disclosed local-only AI request before refining")
	}
	if current.State != jobs.RunScopeActive {
		return GeometrySessionSnapshot{}, jobs.ErrScopeAdmissionClosed
	}
	children, err := c.scopes.Children(ctx, id)
	if err != nil {
		return GeometrySessionSnapshot{}, err
	}
	if len(children) != 1 {
		return GeometrySessionSnapshot{}, jobs.ErrScopeAdmissionClosed
	}
	parentJob, err := c.jobs.Get(ctx, children[0].JobID)
	if err != nil {
		return GeometrySessionSnapshot{}, err
	}
	if parentJob.State != jobs.Succeeded && parentJob.State != jobs.Failed {
		return GeometrySessionSnapshot{}, errors.New("the prior creation step must finish before refinement")
	}
	if strings.TrimSpace(req.CorrectionSourceJobID) != parentJob.ID {
		return GeometrySessionSnapshot{}, errors.New("refinement must name the session's exact baseline job")
	}
	if err := c.verifySessionSourceMedia(ctx, original, parentJob); err != nil {
		return GeometrySessionSnapshot{}, err
	}
	jobReq := GeometryCreationRequest{ProjectID: original.ProjectID, WorkflowScopeID: id, CandidateID: proposed.CandidateID, IdempotencyKey: strings.TrimSpace(req.IdempotencyKey), Brief: original.Brief, EnvelopeMM: original.EnvelopeMM, ReferenceMediaIDs: original.ReferenceMediaIDs, FundingRoute: GeometryFundingLocal, CorrectionSourceJobID: req.CorrectionSourceJobID, CorrectionMode: req.CorrectionMode, CorrectionRecipeSHA256: req.CorrectionRecipeSHA256, CorrectionFeedback: req.CorrectionFeedback}
	if _, _, err := c.SubmitGeometryCreation(ctx, jobReq); err != nil {
		return GeometrySessionSnapshot{}, err
	}
	snapshot, err := c.geometrySessionSnapshot(ctx, id)
	return snapshot, err
}

func (c *Coordinator) CancelGeometrySession(ctx context.Context, id string, expected int64) (GeometrySessionSnapshot, error) {
	if c.scopes == nil {
		return GeometrySessionSnapshot{}, errors.New("bounded creation sessions are unavailable")
	}
	scope, children, err := c.scopes.Close(ctx, strings.TrimSpace(id), expected)
	if err != nil {
		return GeometrySessionSnapshot{}, err
	}
	for _, child := range children {
		job, getErr := c.jobs.Get(ctx, child.JobID)
		if getErr != nil || job.State == jobs.Succeeded || job.State == jobs.Failed || job.State == jobs.Cancelled {
			continue
		}
		if _, cancelErr := c.Cancel(ctx, job.ID, job.Version); cancelErr != nil {
			// Keep the parent in closing; GET exposes the unresolved child and
			// operator can resume cancellation from the durable state.
			continue
		}
	}
	children, err = c.scopes.Children(ctx, id)
	if err != nil {
		return GeometrySessionSnapshot{}, err
	}
	allTerminal := true
	for _, child := range children {
		j, e := c.jobs.Get(ctx, child.JobID)
		if e != nil || (j.State != jobs.Succeeded && j.State != jobs.Failed && j.State != jobs.Cancelled) {
			allTerminal = false
		}
	}
	if allTerminal && scope.State == jobs.RunScopeClosing {
		_, _ = c.scopes.Finalize(ctx, id, jobs.RunScopeCancelled, scope.Version)
	}
	return c.geometrySessionSnapshot(ctx, id)
}

func (c *Coordinator) ensureGeometryScope(ctx context.Context, id string, policy GeometrySessionPolicy) error {
	policyBytes, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	scope := jobs.RunScope{ID: id, OwnerKind: geometrySessionOwner, Policy: policyBytes, MaxChildren: 2, MaxProviderCalls: 2}
	if err := c.scopes.Create(ctx, scope); err == nil {
		return nil
	}
	existing, getErr := c.scopes.Get(ctx, id)
	if getErr != nil || existing.OwnerKind != geometrySessionOwner || string(existing.Policy) != string(policyBytes) || existing.MaxChildren != 2 || existing.MaxProviderCalls != 2 {
		return jobs.ErrIdempotencyConflict
	}
	return nil
}

func (c *Coordinator) geometrySessionSnapshot(ctx context.Context, id string) (GeometrySessionSnapshot, error) {
	scope, err := c.scopes.Get(ctx, id)
	if err != nil || scope.OwnerKind != geometrySessionOwner {
		return GeometrySessionSnapshot{}, jobs.ErrNotFound
	}
	var policy GeometrySessionPolicy
	if err := json.Unmarshal(scope.Policy, &policy); err != nil {
		return GeometrySessionSnapshot{}, err
	}
	children, err := c.scopes.Children(ctx, id)
	if err != nil {
		return GeometrySessionSnapshot{}, err
	}
	snapshot := GeometrySessionSnapshot{SessionID: id, State: scope.State, Version: scope.Version, Policy: policy, MaxProviderCalls: scope.MaxProviderCalls, ProviderCallsReserved: scope.ProviderCallsReserved, MaxChildren: scope.MaxChildren, ChildrenAdmitted: scope.ChildrenAdmitted, Children: make([]GeometrySessionChild, 0, len(children))}
	for _, child := range children {
		job, e := c.jobs.Get(ctx, child.JobID)
		if e != nil {
			return GeometrySessionSnapshot{}, e
		}
		snapshot.Children = append(snapshot.Children, GeometrySessionChild{JobID: job.ID, Ordinal: child.Ordinal, State: job.State, Version: job.Version, CandidateID: job.Input.SubjectID})
	}
	return snapshot, nil
}

func (c *Coordinator) submitGeometrySessionChild(ctx context.Context, policy GeometrySessionPolicy, scopeID, key, correction string) (jobs.Job, bool, error) {
	return c.SubmitGeometryCreation(ctx, GeometryCreationRequest{ProjectID: policy.ProjectID, WorkflowScopeID: scopeID, CandidateID: policy.CandidateID, IdempotencyKey: key, Brief: policy.Brief, EnvelopeMM: policy.EnvelopeMM, ReferenceMediaIDs: policy.ReferenceMediaIDs, FundingRoute: GeometryFundingLocal, CorrectionSourceJobID: correction})
}

func canonicalGeometrySessionPolicy(req GeometrySessionRequest, parent string) (GeometrySessionPolicy, error) {
	req.ProjectID, req.CandidateID, req.IdempotencyKey, req.Brief = strings.TrimSpace(req.ProjectID), strings.TrimSpace(req.CandidateID), strings.TrimSpace(req.IdempotencyKey), strings.TrimSpace(req.Brief)
	req.FundingRoute = NormalizeGeometryFundingRoute(req.FundingRoute)
	if req.FundingRoute == "" {
		req.FundingRoute = GeometryFundingLocal
	}
	if req.FundingRoute != GeometryFundingLocal || req.ProjectID == "" || !creationCandidateID.MatchString(req.CandidateID) || req.IdempotencyKey == "" || len(req.IdempotencyKey) > 180 || req.Brief == "" || len(req.Brief) > 4000 || len(req.ReferenceMediaIDs) < 1 || len(req.ReferenceMediaIDs) > 4 {
		return GeometrySessionPolicy{}, errors.New("bounded creation sessions require a project, candidate, brief, references, and the local-only funding route")
	}
	ids := append([]string(nil), req.ReferenceMediaIDs...)
	seen := map[string]bool{}
	for i := range ids {
		ids[i] = strings.TrimSpace(ids[i])
		if ids[i] == "" || seen[ids[i]] {
			return GeometrySessionPolicy{}, errors.New("selected reference media IDs must be distinct")
		}
		seen[ids[i]] = true
	}
	for _, dimension := range req.EnvelopeMM {
		if math.IsNaN(dimension) || math.IsInf(dimension, 0) || dimension <= 0 || dimension > 1000 {
			return GeometrySessionPolicy{}, errors.New("locked dimensions are outside the supported range")
		}
	}
	return GeometrySessionPolicy{Version: 1, ProjectID: req.ProjectID, CandidateID: req.CandidateID, Brief: req.Brief, EnvelopeMM: req.EnvelopeMM, ReferenceMediaIDs: ids, FundingRoute: GeometryFundingLocal, TokensPerCall: 4096, ParentSessionID: parent}, nil
}

func geometrySessionID(project, key string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("device-studio/geometry-session/"+project+"/"+strings.TrimSpace(key))).String()
}

func changedGeometrySessionScope(a, b GeometrySessionPolicy) []string {
	changed := []string{}
	if a.ProjectID != b.ProjectID {
		changed = append(changed, "project_id")
	}
	// Candidate IDs identify each immutable child output. A refinement must
	// mint a new candidate while retaining the session's locked source scope.
	if a.Brief != b.Brief {
		changed = append(changed, "brief")
	}
	if a.EnvelopeMM != b.EnvelopeMM {
		changed = append(changed, "envelope_mm")
	}
	if strings.Join(a.ReferenceMediaIDs, "\x00") != strings.Join(b.ReferenceMediaIDs, "\x00") {
		changed = append(changed, "reference_media_ids")
	}
	if a.FundingRoute != b.FundingRoute {
		changed = append(changed, "funding_route")
	}
	return changed
}

func (c *Coordinator) verifySessionSourceMedia(ctx context.Context, policy GeometrySessionPolicy, parent jobs.Job) error {
	var source geometrySubjectData
	if err := json.Unmarshal(parent.Input.SubjectData, &source); err != nil || source.Brief != policy.Brief || source.EnvelopeMM != policy.EnvelopeMM || len(source.References) != len(policy.ReferenceMediaIDs) {
		return errors.New("creation session's pinned source snapshot is unavailable")
	}
	for i, id := range policy.ReferenceMediaIDs {
		media, err := c.media.Get(ctx, policy.ProjectID, id)
		if err != nil || media.ID != id || media.SHA256 != source.References[i].SHA256 || source.References[i].MediaID != id {
			return errors.New("selected source media changed since the session started; confirm a new session to expand scope")
		}
	}
	return nil
}
