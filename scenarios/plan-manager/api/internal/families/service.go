package families

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	familiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) Create(ctx context.Context, req *familiesv1.CreateFamilyRequest) (*familiesv1.PlanFamily, error) {
	if req == nil || strings.TrimSpace(req.GetSlug()) == "" || strings.TrimSpace(req.GetOutcome()) == "" {
		return nil, errors.New("slug and outcome are required")
	}
	now := timestamppb.New(s.now().UTC())
	policy := clonePolicy(req.GetPolicy())
	if policy.GetMaximumParallelPlans() == 0 {
		policy.MaximumParallelPlans = 1
	}
	// These are safety invariants, not caller-selectable preferences. Proto3 bool
	// fields cannot distinguish "unset" from false, so accepting false here would
	// let a partially specified policy bypass review or parallelize unknown work.
	policy.UnknownInteractionsSequential = true
	policy.RequireReviewBeforeLaunch = true
	family := &familiesv1.PlanFamily{SchemaVersion: 1, FamilyId: uuid.NewString(), Slug: strings.TrimSpace(req.GetSlug()), Outcome: strings.TrimSpace(req.GetOutcome()), SharedContext: strings.TrimSpace(req.GetSharedContext()), Policy: policy, Revision: 1, CreatedAt: now, UpdatedAt: now, ExecutionState: familiesv1.FamilyExecutionState_FAMILY_EXECUTION_STATE_PENDING, LastActivityAt: now}
	if err := s.repo.Create(ctx, family); err != nil {
		return nil, err
	}
	return cloneFamily(family), nil
}

func (s *Service) Get(ctx context.Context, id string) (*familiesv1.PlanFamily, error) {
	return s.repo.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) List(ctx context.Context, size uint32, token string) ([]*familiesv1.PlanFamily, string, error) {
	return s.repo.List(ctx, size, token)
}

func (s *Service) Update(ctx context.Context, req *familiesv1.UpdateFamilyRequest) (*familiesv1.PlanFamily, error) {
	return s.mutate(ctx, req.GetFamilyId(), req.GetExpectedRevision(), func(f *familiesv1.PlanFamily) error {
		if strings.TrimSpace(req.GetOutcome()) == "" {
			return errors.New("outcome is required")
		}
		f.Outcome, f.SharedContext = strings.TrimSpace(req.GetOutcome()), strings.TrimSpace(req.GetSharedContext())
		if req.GetPolicy() != nil {
			if !proto.Equal(f.Policy, req.GetPolicy()) {
				invalidateGraph(f)
			}
			f.Policy = clonePolicy(req.GetPolicy())
			if f.Policy.GetMaximumParallelPlans() == 0 {
				f.Policy.MaximumParallelPlans = 1
			}
			f.Policy.UnknownInteractionsSequential = true
			f.Policy.RequireReviewBeforeLaunch = true
		}
		return nil
	}, nil, nil)
}

func (s *Service) PutMember(ctx context.Context, req *familiesv1.PutMemberRequest) (*familiesv1.PlanFamily, error) {
	return s.mutate(ctx, req.GetFamilyId(), req.GetExpectedRevision(), func(f *familiesv1.PlanFamily) error {
		member := req.GetMember()
		if member == nil || strings.TrimSpace(member.GetPlanId()) == "" {
			return errors.New("member plan_id is required")
		}
		if member.GetRole() == familiesv1.MemberRole_MEMBER_ROLE_UNSPECIFIED || member.GetState() == familiesv1.MemberState_MEMBER_STATE_UNSPECIFIED {
			return errors.New("member role and state are required")
		}
		var previous *familiesv1.FamilyMember
		for _, current := range f.GetMembers() {
			if current.GetPlanId() == member.GetPlanId() {
				previous = current
			}
		}
		topologyChanged := previous == nil || previous.GetRole() != member.GetRole() || previous.GetSourceRevision() != member.GetSourceRevision()
		if member.GetState() == familiesv1.MemberState_MEMBER_STATE_RUNNING && (previous == nil || previous.GetState() != familiesv1.MemberState_MEMBER_STATE_RUNNING) {
			if topologyChanged || previous.GetState() != familiesv1.MemberState_MEMBER_STATE_PENDING || member.GetExecutionId() == "" || member.GetAdmissionKey() == "" {
				return errors.New("launch requires an unchanged pending member, execution identity and admission key")
			}
			frontier := currentFrontier(f)
			admitted := false
			if frontier.GetLaunchable() && len(frontier.GetBatches()) > 0 {
				for _, id := range frontier.GetBatches()[0].GetPlanIds() {
					admitted = admitted || id == member.GetPlanId()
				}
			}
			if !admitted {
				return errors.New("member is not in the reviewed runnable frontier")
			}
		}
		if previous != nil && previous.GetState() == familiesv1.MemberState_MEMBER_STATE_RUNNING && (member.GetAdmissionKey() != previous.GetAdmissionKey() || member.GetExecutionId() != previous.GetExecutionId()) {
			return errors.New("running admission identity cannot be replaced")
		}
		replaceMember(f, proto.Clone(member).(*familiesv1.FamilyMember))
		if topologyChanged {
			invalidateGraph(f)
		}
		return nil
	}, nil, nil)
}

func (s *Service) RemoveMember(ctx context.Context, req *familiesv1.RemoveMemberRequest) (*familiesv1.PlanFamily, error) {
	return s.mutate(ctx, req.GetFamilyId(), req.GetExpectedRevision(), func(f *familiesv1.PlanFamily) error {
		id := strings.TrimSpace(req.GetPlanId())
		if id == "" {
			return errors.New("plan_id is required")
		}
		members := f.Members[:0]
		found := false
		for _, member := range f.Members {
			if member.GetPlanId() == id {
				found = true
				continue
			}
			members = append(members, member)
		}
		if !found {
			return fmt.Errorf("member %q not found", id)
		}
		f.Members = members
		claims := f.Claims[:0]
		for _, claim := range f.Claims {
			if claim.GetPlanId() != id {
				claims = append(claims, claim)
			}
		}
		f.Claims = claims
		invalidateGraph(f)
		return nil
	}, nil, nil)
}

func (s *Service) PutClaim(ctx context.Context, req *familiesv1.PutClaimRequest) (*familiesv1.PlanFamily, error) {
	return s.mutate(ctx, req.GetFamilyId(), req.GetExpectedRevision(), func(f *familiesv1.PlanFamily) error {
		claim := req.GetClaim()
		if claim == nil || strings.TrimSpace(claim.GetClaimId()) == "" || strings.TrimSpace(claim.GetPlanId()) == "" || strings.TrimSpace(claim.GetResource()) == "" {
			return errors.New("claim_id, plan_id, and resource are required")
		}
		if !hasMember(f, claim.GetPlanId()) {
			return fmt.Errorf("claim references unknown member %q", claim.GetPlanId())
		}
		if claim.GetKind() == familiesv1.ClaimKind_CLAIM_KIND_UNSPECIFIED || claim.GetAccess() == familiesv1.ClaimAccess_CLAIM_ACCESS_UNSPECIFIED {
			return errors.New("claim kind and access are required")
		}
		replaceClaim(f, proto.Clone(claim).(*familiesv1.ResourceClaim))
		invalidateGraph(f)
		return nil
	}, nil, nil)
}

func (s *Service) RemoveClaim(ctx context.Context, req *familiesv1.RemoveClaimRequest) (*familiesv1.PlanFamily, error) {
	return s.mutate(ctx, req.GetFamilyId(), req.GetExpectedRevision(), func(f *familiesv1.PlanFamily) error {
		id := strings.TrimSpace(req.GetClaimId())
		claims := f.Claims[:0]
		found := false
		for _, claim := range f.Claims {
			if claim.GetClaimId() == id {
				found = true
				continue
			}
			claims = append(claims, claim)
		}
		if !found {
			return fmt.Errorf("claim %q not found", id)
		}
		f.Claims = claims
		invalidateGraph(f)
		return nil
	}, nil, nil)
}

func (s *Service) ProposeGraph(ctx context.Context, req *familiesv1.ProposeGraphRequest) (*familiesv1.PlanFamily, error) {
	var graph *familiesv1.GraphRevision
	return s.mutate(ctx, req.GetFamilyId(), req.GetExpectedRevision(), func(f *familiesv1.PlanFamily) error {
		edges, err := compileEdges(f, req.GetExplicitEdges())
		if err != nil {
			return err
		}
		batches, cyclic, diagnostics := compileFrontier(f, edges)
		graph = &familiesv1.GraphRevision{Revision: nextGraphRevision(f), Edges: edges, ProposedFrontier: batches, Cyclic: cyclic, Diagnostics: diagnostics, CreatedAt: timestamppb.New(s.now().UTC())}
		f.Graph, f.Review = graph, nil
		return nil
	}, func() *familiesv1.GraphRevision { return graph }, nil)
}

func (s *Service) ReviewGraph(ctx context.Context, req *familiesv1.ReviewGraphRequest) (*familiesv1.PlanFamily, error) {
	var review *familiesv1.GraphReview
	return s.mutate(ctx, req.GetFamilyId(), req.GetExpectedRevision(), func(f *familiesv1.PlanFamily) error {
		if f.GetGraph() == nil || f.GetGraph().GetRevision() != req.GetGraphRevision() {
			return errors.New("review must target the current graph revision")
		}
		if f.GetReview() != nil {
			return errors.New("graph revision already reviewed")
		}
		if req.GetDecision() == familiesv1.ReviewDecision_REVIEW_DECISION_UNSPECIFIED || strings.TrimSpace(req.GetReviewer()) == "" || strings.TrimSpace(req.GetRationale()) == "" {
			return errors.New("decision, reviewer, and rationale are required")
		}
		if req.GetDecision() == familiesv1.ReviewDecision_REVIEW_DECISION_APPROVED && f.GetGraph().GetCyclic() {
			return errors.New("cyclic graph cannot be approved")
		}
		if req.GetDecision() == familiesv1.ReviewDecision_REVIEW_DECISION_CORRECTED {
			edges, err := compileEdges(f, req.GetCorrectedEdges())
			if err != nil {
				return err
			}
			_, cyclic, _ := compileFrontier(f, edges)
			if cyclic {
				return errors.New("corrected graph remains cyclic")
			}
		}
		review = &familiesv1.GraphReview{GraphRevision: req.GetGraphRevision(), Decision: req.GetDecision(), Reviewer: strings.TrimSpace(req.GetReviewer()), Rationale: strings.TrimSpace(req.GetRationale()), CorrectedEdges: cloneEdges(req.GetCorrectedEdges()), ReviewedAt: timestamppb.New(s.now().UTC())}
		f.Review = review
		if req.GetDecision() == familiesv1.ReviewDecision_REVIEW_DECISION_APPROVED || req.GetDecision() == familiesv1.ReviewDecision_REVIEW_DECISION_CORRECTED {
			f.ActiveGraphRevision = req.GetGraphRevision()
		}
		return nil
	}, nil, func() *familiesv1.GraphReview { return review })
}

func (s *Service) Frontier(ctx context.Context, id string) (*familiesv1.GetFrontierResponse, error) {
	f, err := s.repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	return currentFrontier(f), nil
}

// Current state is evaluated on the same revision used by PutMember's atomic
// repository CAS. A projected schedule is never a launch authorization.
func currentFrontier(f *familiesv1.PlanFamily) *familiesv1.GetFrontierResponse {
	response := &familiesv1.GetFrontierResponse{Review: f.GetReview()}
	if f.GetGraph() == nil {
		response.Diagnostics = []string{"graph has not been proposed"}
		return response
	}
	response.GraphRevision = f.GetGraph().GetRevision()
	edges := f.GetGraph().GetEdges()
	reviewed := f.GetReview().GetGraphRevision() == f.GetGraph().GetRevision() && (f.GetReview().GetDecision() == familiesv1.ReviewDecision_REVIEW_DECISION_APPROVED || f.GetReview().GetDecision() == familiesv1.ReviewDecision_REVIEW_DECISION_CORRECTED)
	if f.GetReview().GetDecision() == familiesv1.ReviewDecision_REVIEW_DECISION_CORRECTED {
		edges = f.GetReview().GetCorrectedEdges()
	}
	var cyclic bool
	response.Batches, cyclic, response.Diagnostics = compileFrontier(f, edges)
	response.ProjectedBatches = cloneBatches(response.Batches)
	if len(response.Batches) > 1 {
		response.Batches = response.Batches[:1]
	}
	if !reviewed {
		response.Diagnostics = append(response.Diagnostics, "current graph revision is not approved")
	}
	response.Launchable = !cyclic && reviewed && len(response.Batches) > 0
	return response
}

func (s *Service) mutate(ctx context.Context, id string, expected uint64, change func(*familiesv1.PlanFamily) error, graph func() *familiesv1.GraphRevision, review func() *familiesv1.GraphReview) (*familiesv1.PlanFamily, error) {
	family, err := s.repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if expected == 0 || family.GetRevision() != expected {
		return nil, ErrConflict
	}
	if err := change(family); err != nil {
		return nil, err
	}
	family.Revision++
	family.UpdatedAt = timestamppb.New(s.now().UTC())
	family.LastActivityAt = family.UpdatedAt
	family.ExecutionState = executionState(family.GetMembers())
	var g *familiesv1.GraphRevision
	if graph != nil {
		g = graph()
	}
	var r *familiesv1.GraphReview
	if review != nil {
		r = review()
	}
	if err := s.repo.Commit(ctx, family, expected, g, r); err != nil {
		return nil, err
	}
	return cloneFamily(family), nil
}

func compileEdges(f *familiesv1.PlanFamily, explicit []*familiesv1.FamilyEdge) ([]*familiesv1.FamilyEdge, error) {
	edges := cloneEdges(explicit)
	for _, edge := range edges {
		if !hasMember(f, edge.GetFromPlanId()) || !hasMember(f, edge.GetToPlanId()) {
			return nil, errors.New("graph edge references unknown member")
		}
		if edge.GetFromPlanId() == edge.GetToPlanId() || edge.GetKind() == familiesv1.EdgeKind_EDGE_KIND_UNSPECIFIED {
			return nil, errors.New("graph edge must connect distinct members with a kind")
		}
		if edge.Provenance == familiesv1.EdgeProvenance_EDGE_PROVENANCE_UNSPECIFIED {
			edge.Provenance = familiesv1.EdgeProvenance_EDGE_PROVENANCE_EXPLICIT
		}
	}
	for i, left := range f.GetClaims() {
		for _, right := range f.GetClaims()[i+1:] {
			if left.GetPlanId() == right.GetPlanId() {
				continue
			}
			unknown := left.GetKind() == familiesv1.ClaimKind_CLAIM_KIND_UNKNOWN_INTERACTION || right.GetKind() == familiesv1.ClaimKind_CLAIM_KIND_UNKNOWN_INTERACTION || !left.GetResolved() || !right.GetResolved()
			// A semantic category does not partition the physical worktree. A
			// directory writer also owns nested schemas, tests and generated files.
			collision := resourcesOverlap(left.GetResource(), right.GetResource()) && (left.GetAccess() == familiesv1.ClaimAccess_CLAIM_ACCESS_EXCLUSIVE_WRITE || right.GetAccess() == familiesv1.ClaimAccess_CLAIM_ACCESS_EXCLUSIVE_WRITE)
			if unknown && !f.GetPolicy().GetUnknownInteractionsSequential() {
				continue
			}
			if unknown || collision {
				edges = append(edges, &familiesv1.FamilyEdge{FromPlanId: left.GetPlanId(), ToPlanId: right.GetPlanId(), Kind: familiesv1.EdgeKind_EDGE_KIND_CONFLICT, Provenance: familiesv1.EdgeProvenance_EDGE_PROVENANCE_INFERRED, Reason: "typed resource claims cannot run concurrently", ClaimIds: []string{left.GetClaimId(), right.GetClaimId()}})
			}
		}
	}
	edges = dedupeEdges(edges)
	sort.Slice(edges, func(i, j int) bool { return edgeKey(edges[i]) < edgeKey(edges[j]) })
	return edges, nil
}

func compileFrontier(f *familiesv1.PlanFamily, edges []*familiesv1.FamilyEdge) ([]*familiesv1.FrontierBatch, bool, []string) {
	pending := map[string]bool{}
	diagnostics := []string{}
	for _, m := range f.GetMembers() {
		if m.GetState() == familiesv1.MemberState_MEMBER_STATE_PENDING {
			pending[m.GetPlanId()] = true
		}
	}
	deps, conflicts := map[string]map[string]bool{}, map[string]map[string]bool{}
	for _, edge := range edges {
		if edge.GetKind() == familiesv1.EdgeKind_EDGE_KIND_DEPENDENCY {
			if deps[edge.GetFromPlanId()] == nil {
				deps[edge.GetFromPlanId()] = map[string]bool{}
			}
			deps[edge.GetFromPlanId()][edge.GetToPlanId()] = true
		}
		if edge.GetKind() == familiesv1.EdgeKind_EDGE_KIND_CONFLICT {
			if conflicts[edge.GetFromPlanId()] == nil {
				conflicts[edge.GetFromPlanId()] = map[string]bool{}
			}
			if conflicts[edge.GetToPlanId()] == nil {
				conflicts[edge.GetToPlanId()] = map[string]bool{}
			}
			conflicts[edge.GetFromPlanId()][edge.GetToPlanId()], conflicts[edge.GetToPlanId()][edge.GetFromPlanId()] = true, true
		}
	}
	// Cycles are a graph property, independent of running or failed members.
	visiting, visited := map[string]bool{}, map[string]bool{}
	var cycle func(string) bool
	cycle = func(id string) bool {
		if visiting[id] {
			return true
		}
		if visited[id] {
			return false
		}
		visiting[id] = true
		for dep := range deps[id] {
			if cycle(dep) {
				return true
			}
		}
		visiting[id] = false
		visited[id] = true
		return false
	}
	for _, member := range f.GetMembers() {
		if cycle(member.GetPlanId()) {
			return nil, true, []string{"dependency graph is cyclic"}
		}
	}
	active := map[string]bool{}
	for _, member := range f.GetMembers() {
		if member.GetState() == familiesv1.MemberState_MEMBER_STATE_RUNNING {
			active[member.GetPlanId()] = true
		}
	}
	done := map[string]bool{}
	failed := map[string]bool{}
	for _, m := range f.GetMembers() {
		if m.GetState() == familiesv1.MemberState_MEMBER_STATE_SUCCEEDED {
			done[m.GetPlanId()] = true
		}
		if m.GetState() == familiesv1.MemberState_MEMBER_STATE_FAILED || m.GetState() == familiesv1.MemberState_MEMBER_STATE_BLOCKED || m.GetState() == familiesv1.MemberState_MEMBER_STATE_SKIPPED {
			failed[m.GetPlanId()] = true
		}
	}
	// A failed prerequisite blocks only its reachable dependents. Independent
	// members remain eligible, which is the family-level partial-failure rule.
	for changed := true; changed; {
		changed = false
		for id := range pending {
			for dependency := range deps[id] {
				if failed[dependency] {
					delete(pending, id)
					failed[id] = true
					diagnostics = append(diagnostics, fmt.Sprintf("%s is blocked by failed dependency %s", id, dependency))
					changed = true
					break
				}
			}
		}
	}
	max := int(f.GetPolicy().GetMaximumParallelPlans())
	if max < 1 {
		max = 1
	}
	batches := []*familiesv1.FrontierBatch{}
	for len(pending) > 0 {
		ready := []string{}
		for id := range pending {
			satisfied := true
			for dep := range deps[id] {
				if !done[dep] {
					satisfied = false
				}
			}
			if satisfied {
				ready = append(ready, id)
			}
		}
		sort.Strings(ready)
		if len(ready) == 0 {
			diagnostics = append(diagnostics, "pending members are waiting for prerequisite outcomes")
			return batches, false, diagnostics
		}
		batch := []string{}
		for _, id := range ready {
			compatible := true
			for running := range active {
				if conflicts[id][running] {
					compatible = false
				}
			}
			for _, chosen := range batch {
				if conflicts[id][chosen] {
					compatible = false
				}
			}
			if compatible && len(batch)+len(active) < max {
				batch = append(batch, id)
			}
		}
		if len(batch) == 0 {
			return batches, false, append(diagnostics, "running members occupy required claims or capacity")
		}
		batches = append(batches, &familiesv1.FrontierBatch{Ordinal: uint32(len(batches) + 1), PlanIds: batch})
		for _, id := range batch {
			delete(pending, id)
			done[id] = true
		}
	}
	return batches, false, diagnostics
}

func hasMember(f *familiesv1.PlanFamily, id string) bool {
	for _, m := range f.GetMembers() {
		if m.GetPlanId() == id {
			return true
		}
	}
	return false
}
func replaceMember(f *familiesv1.PlanFamily, value *familiesv1.FamilyMember) {
	for i, m := range f.Members {
		if m.GetPlanId() == value.GetPlanId() {
			f.Members[i] = value
			return
		}
	}
	f.Members = append(f.Members, value)
	sort.Slice(f.Members, func(i, j int) bool { return f.Members[i].GetPlanId() < f.Members[j].GetPlanId() })
}
func replaceClaim(f *familiesv1.PlanFamily, value *familiesv1.ResourceClaim) {
	for i, c := range f.Claims {
		if c.GetClaimId() == value.GetClaimId() {
			f.Claims[i] = value
			return
		}
	}
	f.Claims = append(f.Claims, value)
	sort.Slice(f.Claims, func(i, j int) bool { return f.Claims[i].GetClaimId() < f.Claims[j].GetClaimId() })
}
func invalidateGraph(f *familiesv1.PlanFamily) {
	f.Graph = nil
	f.Review = nil
	f.ActiveGraphRevision = 0
}
func nextGraphRevision(f *familiesv1.PlanFamily) uint64 { return f.GetRevision() + 1 }
func cloneFamily(f *familiesv1.PlanFamily) *familiesv1.PlanFamily {
	return proto.Clone(f).(*familiesv1.PlanFamily)
}
func clonePolicy(p *familiesv1.FamilyPolicy) *familiesv1.FamilyPolicy {
	if p == nil {
		return &familiesv1.FamilyPolicy{}
	}
	return proto.Clone(p).(*familiesv1.FamilyPolicy)
}
func cloneEdges(v []*familiesv1.FamilyEdge) []*familiesv1.FamilyEdge {
	out := make([]*familiesv1.FamilyEdge, 0, len(v))
	for _, x := range v {
		if x != nil {
			out = append(out, proto.Clone(x).(*familiesv1.FamilyEdge))
		}
	}
	return out
}
func cloneBatches(v []*familiesv1.FrontierBatch) []*familiesv1.FrontierBatch {
	out := make([]*familiesv1.FrontierBatch, 0, len(v))
	for _, x := range v {
		out = append(out, proto.Clone(x).(*familiesv1.FrontierBatch))
	}
	return out
}
func edgeKey(e *familiesv1.FamilyEdge) string {
	a, b := e.GetFromPlanId(), e.GetToPlanId()
	if e.GetKind() == familiesv1.EdgeKind_EDGE_KIND_CONFLICT && a > b {
		a, b = b, a
	}
	return fmt.Sprintf("%d:%s:%s", e.GetKind(), a, b)
}
func resourcesOverlap(left, right string) bool {
	normalize := func(value string) string {
		value = strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(value), "\\", "/"), "path:")
		value = strings.TrimPrefix(value, "./")
		// Compare the guaranteed literal directory prefixes. This can serialize
		// disjoint complex globs, but never proves independence from a wildcard.
		if at := strings.IndexAny(value, "*?[{"); at >= 0 {
			value = value[:at]
			if slash := strings.LastIndex(value, "/"); slash >= 0 {
				value = value[:slash]
			} else {
				value = ""
			}
		}
		value = path.Clean(value)
		if value == "." || value == ".." || strings.HasPrefix(value, "../") {
			return "" // unresolved root claims cannot establish independence.
		}
		return strings.TrimRight(value, "/")
	}
	a, b := normalize(left), normalize(right)
	return a == "" || b == "" || a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}

func executionState(members []*familiesv1.FamilyMember) familiesv1.FamilyExecutionState {
	if len(members) == 0 {
		return familiesv1.FamilyExecutionState_FAMILY_EXECUTION_STATE_PENDING
	}
	running, pending, failed, succeeded := false, false, false, 0
	for _, member := range members {
		switch member.GetState() {
		case familiesv1.MemberState_MEMBER_STATE_RUNNING:
			running = true
		case familiesv1.MemberState_MEMBER_STATE_PENDING:
			pending = true
		case familiesv1.MemberState_MEMBER_STATE_SUCCEEDED:
			succeeded++
		case familiesv1.MemberState_MEMBER_STATE_FAILED, familiesv1.MemberState_MEMBER_STATE_BLOCKED, familiesv1.MemberState_MEMBER_STATE_SKIPPED:
			failed = true
		}
	}
	if running {
		return familiesv1.FamilyExecutionState_FAMILY_EXECUTION_STATE_RUNNING
	}
	if succeeded == len(members) {
		return familiesv1.FamilyExecutionState_FAMILY_EXECUTION_STATE_SUCCEEDED
	}
	if failed && pending {
		return familiesv1.FamilyExecutionState_FAMILY_EXECUTION_STATE_BLOCKED
	}
	if failed {
		return familiesv1.FamilyExecutionState_FAMILY_EXECUTION_STATE_FAILED
	}
	return familiesv1.FamilyExecutionState_FAMILY_EXECUTION_STATE_PENDING
}
func dedupeEdges(v []*familiesv1.FamilyEdge) []*familiesv1.FamilyEdge {
	seen := map[string]bool{}
	out := v[:0]
	for _, e := range v {
		k := edgeKey(e)
		if !seen[k] {
			seen[k] = true
			out = append(out, e)
		}
	}
	return out
}
