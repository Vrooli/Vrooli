package supervision

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EffortActor is populated by the existing authenticated owner/run verifier.
// Request text and workspace declarations never construct this authority.
type EffortActor struct {
	ID           string
	Operator     bool
	OwnerSubject string
	Scopes       []string
}

func (a EffortActor) supervises(e *pb.EffortEnrollment) bool {
	if a.Operator {
		return true
	}
	if a.ID == "" {
		return false
	}
	if e.SupervisorRunId != "" && a.ID == e.SupervisorRunId {
		return true
	}
	if e.AuthorizedBy == "" || e.SupervisorOwnerSubject == "" || e.SupervisorScope == "" || a.OwnerSubject != e.SupervisorOwnerSubject {
		return false
	}
	for _, scope := range a.Scopes {
		if scope == e.SupervisorScope {
			return true
		}
	}
	return false
}

type EffortService struct {
	repo              EffortRepository
	controller        ActionController
	policies          *PolicyStore
	config            EffortDiscoveryConfig
	now               func() time.Time
	mu                sync.Mutex
	nextScan          time.Time
	directiveCursor   string
	runRegistry       EffortRunRegistry
	dispatchSecret    []byte
	dispatchProvision func(string) error
	dispatchProfile   func(context.Context, string) error
}

func NewEffortService(repo EffortRepository, controller ActionController, policies *PolicyStore, config EffortDiscoveryConfig) *EffortService {
	if config.ScanLimit <= 0 || config.ScanLimit > 1000 {
		config.ScanLimit = 100
	}
	if config.FileBytes <= 0 || config.FileBytes > 1024*1024 {
		config.FileBytes = 128 * 1024
	}
	if config.Interval < time.Second {
		config.Interval = time.Minute
	}
	if config.StaleAfter <= 0 {
		config.StaleAfter = 5 * time.Minute
	}
	return &EffortService{repo: repo, controller: controller, policies: policies, config: config, now: time.Now}
}
func validateEffortEnrollment(e *pb.EffortEnrollment, grant bool, now time.Time) error {
	if e == nil || strings.TrimSpace(e.EffortRef) == "" || len(e.EffortRef) > 512 || len(e.Subjects) > 100 {
		return errors.New("bounded effort reference and at most 100 subjects required")
	}
	if e.Workspace != "" && (!filepath.IsLocal(e.Workspace) || filepath.Base(e.Workspace) != e.Workspace || strings.ContainsAny(e.Workspace, "/\\")) {
		return errors.New("workspace must be an immediate relative directory")
	}
	if (e.SupervisorOwnerSubject == "") != (e.SupervisorScope == "") || e.SupervisorScope == "*" || len(e.SupervisorOwnerSubject) > 512 || len(e.SupervisorScope) > 512 {
		return errors.New("stable supervisor delegation requires exact owner subject and non-wildcard scope together")
	}
	seen := map[string]bool{}
	parents := 0
	for _, sub := range e.Subjects {
		if sub == nil || sub.Owner == "" || sub.Kind == "" || sub.Reference == "" {
			return errors.New("subjects require owner, kind and exact reference")
		}
		if sub.RunId != "" {
			if sub.Owner != "agent-manager" || sub.Kind != "run" {
				return errors.New("run identity must belong to Agent Manager")
			}
			if _, err := uuid.Parse(sub.RunId); err != nil {
				return errors.New("invalid subject run UUID")
			}
			if seen[sub.RunId] {
				return errors.New("duplicate run subject")
			}
			seen[sub.RunId] = true
		}
		if sub.Role == "orchestrator" && sub.RunId != "" {
			parents++
		}
	}
	if len(e.PermittedActions) > 0 {
		if !grant || e.AuthorityRef == "" || e.TargetRevision == "" || parents != 1 || e.AuthorizedBy == "" || e.MaximumDirectives < 1 || e.MaximumDirectives > 100 {
			return errors.New("steering requires actual owner grant, exact revision, one orchestrator and bounded directive allowance")
		}
		if e.AuthorityExpiresAt == nil || !e.AuthorityExpiresAt.IsValid() || !e.AuthorityExpiresAt.AsTime().After(now) {
			return errors.New("steering grant requires a future expiry")
		}
		if e.SupervisorRunId != "" {
			if _, err := uuid.Parse(e.SupervisorRunId); err != nil {
				return errors.New("invalid supervisor run UUID")
			}
		}
		for _, kind := range e.PermittedActions {
			if kind != pb.WatchActionKind_WATCH_ACTION_KIND_NUDGE && kind != pb.WatchActionKind_WATCH_ACTION_KIND_CONTINUE && kind != pb.WatchActionKind_WATCH_ACTION_KIND_RECOVER_FRESH {
				return errors.New("effort steering permits nudge, continue or separately granted missing-session recovery through the orchestrator owner")
			}
		}
	}
	return nil
}
func (s *EffortService) Enroll(ctx context.Context, req *pb.EnrollEffortRequest, actor EffortActor) (*pb.EffortEnrollment, error) {
	if !actor.Operator || actor.ID == "" {
		return nil, errors.New("authenticated operator authority required to enroll or amend")
	}
	if req == nil || req.Enrollment == nil || strings.TrimSpace(req.IdempotencyKey) == "" {
		return nil, errors.New("enrollment and idempotency key required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	digest := effortDigest(req) + actor.ID
	key := "enroll:" + req.IdempotencyKey
	replay := &pb.EffortEnrollment{}
	if ok, err := s.repo.ReplayEffortOperation(ctx, key, digest, replay); ok || err != nil {
		return replay, err
	}
	e := proto.Clone(req.Enrollment).(*pb.EffortEnrollment)
	// This server-owned binding is issued only by the dedicated owner operation.
	// Any explicit enrollment amendment invalidates previously issued children.
	e.DispatchAuthorization = nil
	e.AuthorizedBy = actor.ID
	e.Withdrawn = false
	e.WithdrawalReason = ""
	if err := validateEffortEnrollment(e, true, s.now()); err != nil {
		return nil, err
	}
	old, o, err := s.repo.GetEffort(ctx, e.EffortRef)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if old == nil && req.ExpectedRevision != 0 || old != nil && old.Revision != req.ExpectedRevision {
		return nil, ErrConflict
	}
	if o == nil {
		o = &pb.EffortBoardRow{RuntimeState: "unknown", OutcomeStanding: &pb.EffortOutcomeStanding{State: "unknown", Attribution: "owner references"}}
	}
	e.Revision = req.ExpectedRevision + 1
	e.UpdatedAt = timestamppb.New(s.now().UTC())
	if err = s.repo.SaveEffort(ctx, e, o, req.ExpectedRevision, key, digest); err != nil {
		return nil, err
	}
	return e, nil
}
func (s *EffortService) Withdraw(ctx context.Context, req *pb.WithdrawEffortRequest, actor EffortActor) (*pb.EffortEnrollment, error) {
	if req == nil || req.IdempotencyKey == "" || req.Reason == "" {
		return nil, errors.New("withdrawal reason and idempotency key required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := "withdraw:" + req.IdempotencyKey
	digest := effortDigest(req) + actor.ID
	replay := &pb.EffortEnrollment{}
	if ok, err := s.repo.ReplayEffortOperation(ctx, key, digest, replay); ok || err != nil {
		return replay, err
	}
	e, o, err := s.repo.GetEffort(ctx, req.EffortRef)
	if err != nil {
		return nil, err
	}
	if !actor.Operator || actor.ID == "" {
		return nil, errors.New("operator authority required for withdrawal")
	}
	if e.Revision != req.ExpectedRevision {
		return nil, ErrConflict
	}
	e.Withdrawn = true
	e.WithdrawalReason = req.Reason
	e.Revision++
	e.UpdatedAt = timestamppb.New(s.now().UTC())
	if err = s.repo.SaveEffort(ctx, e, o, req.ExpectedRevision, key, digest); err != nil {
		return nil, err
	}
	return e, nil
}
func effortPageSize(n uint32) int {
	if n == 0 {
		return 50
	}
	if n > 100 {
		return 100
	}
	return int(n)
}
func (s *EffortService) List(ctx context.Context, req *pb.ListEffortsRequest) (*pb.ListEffortsResponse, error) {
	if req == nil {
		req = &pb.ListEffortsRequest{}
	}
	n := effortPageSize(req.PageSize)
	rows, err := s.repo.ListEfforts(ctx, req.PageToken, n+1)
	if err != nil {
		return nil, err
	}
	response := &pb.ListEffortsResponse{Efforts: rows}
	if len(rows) > n {
		response.Efforts = rows[:n]
		response.NextPageToken = rows[n-1].EffortRef
	}
	// The retained enrollment bound is explicit in Board when this count is partial.
	all, err := s.repo.ListEfforts(ctx, "", 1001)
	if err != nil {
		return nil, err
	}
	for _, e := range all {
		if !e.Withdrawn && !dispatchOnlyEnrollment(e) {
			response.ActiveCount++
		}
	}
	return response, nil
}
func (s *EffortService) Board(ctx context.Context, req *pb.GetEffortBoardRequest) (*pb.EffortBoard, error) {
	if req == nil {
		req = &pb.GetEffortBoardRequest{}
	}
	list, err := s.List(ctx, &pb.ListEffortsRequest{PageSize: req.PageSize, PageToken: req.PageToken})
	if err != nil {
		return nil, err
	}
	d, err := s.repo.GetEffortDiscovery(ctx)
	if err != nil {
		return nil, err
	}
	b := &pb.EffortBoard{ObservedAt: timestamppb.New(s.now().UTC()), Discovery: d, ActiveCount: list.ActiveCount, NextPageToken: list.NextPageToken, Partial: d.Partial}
	if list.ActiveCount > 1000 {
		b.Partial = true
		b.Limitations = append(b.Limitations, "active count is censored at 1001 enrollments")
	}
	if req.EffortRef != "" {
		e, _, getErr := s.repo.GetEffort(ctx, req.EffortRef)
		if getErr != nil {
			return nil, getErr
		}
		list.Efforts = []*pb.EffortEnrollment{e}
		b.NextPageToken = ""
	}
	identity := d.ChangeIdentity
	for _, e := range list.Efforts {
		_, ob, getErr := s.repo.GetEffort(ctx, e.EffortRef)
		if getErr != nil {
			return nil, getErr
		}
		row := s.projectEffort(ctx, e, ob, d)
		assessment, assessmentErr := s.repo.LatestEffortAssessment(ctx, e.EffortRef)
		if assessmentErr != nil && !errors.Is(assessmentErr, ErrNotFound) {
			return nil, assessmentErr
		}
		row.LastAssessment = assessment
		directives, listErr := s.repo.ListEffortDirectives(ctx, e.EffortRef, "", 101)
		if listErr != nil {
			return nil, listErr
		}
		if len(directives) > 100 {
			row.Limitations = append(row.Limitations, "directives capped at 100; use ListEffortDirectives pagination")
			directives = directives[:100]
		}
		for _, directive := range directives {
			copy := proto.Clone(directive).(*pb.EffortDirective)
			copy.SourceSnapshot = nil
			row.Directives = append(row.Directives, copy)
			if !e.Withdrawn && directive.SupersededBy == "" && directive.RecoveryExpectation != nil && directive.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED {
				verification := directive.GetRecoveryVerification()
				if verification.GetState() == "pending" || verification.GetState() == "" {
					row.PendingOperations = append(row.PendingOperations, "agent-manager:effort-directive:"+directive.DirectiveId+":verify-progress")
				} else if verification.GetState() == "failed" || verification.GetState() == "owner-wait" {
					row.PendingOperations = append(row.PendingOperations, verification.GetNextOwnerCondition())
				}
			}
		}
		cut := proto.Clone(row).(*pb.EffortBoardRow)
		cut.ObservedAt = nil
		cut.ChangeIdentity = ""
		cut.VisibilityChangeIdentity = ""
		for _, a := range cut.Assignments {
			a.ObservedAt = nil
		}
		row.VisibilityChangeIdentity = effortDigest(cut)
		row.ChangeIdentity = effortSubjectIdentity(row, ob)
		identity += row.VisibilityChangeIdentity
		b.Rows = append(b.Rows, row)
		if row.Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_FRESH || len(row.Limitations) > 0 || row.Usage.GetPartial() {
			b.Partial = true
		}
	}
	b.ChangeIdentity = bytesDigest([]byte(identity))
	return b, nil
}
func (s *EffortService) projectEffort(ctx context.Context, e *pb.EffortEnrollment, ob *pb.EffortBoardRow, d *pb.EffortDiscovery) *pb.EffortBoardRow {
	row := proto.Clone(ob).(*pb.EffortBoardRow)
	row.Enrollment = e
	row.Assignments = nil
	row.Directives = nil
	row.Usage = &pb.EffortUsage{Source: "Agent Manager current run summaries", Partial: true, Limitations: []string{"billing and unregistered agent work remain unknown; summary cost estimates are not reported charges"}}
	if row.OutcomeStanding == nil {
		row.OutcomeStanding = &pb.EffortOutcomeStanding{State: "unknown"}
	}
	now := s.now().UTC()
	if e.Workspace == "" {
		row.ObservedAt = timestamppb.New(now)
		row.Freshness = pb.EffortFreshness_EFFORT_FRESHNESS_FRESH
	} else if row.Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE {
		stamp := ob.GetObservedAt().AsTime()
		if now.Sub(stamp) > s.config.StaleAfter {
			row.Freshness = pb.EffortFreshness_EFFORT_FRESHNESS_STALE
		}
	}
	seen := map[string]bool{}
	allTokens := true
	tokens := int64(0)
	seconds := float64(0)
	allTime := true
	unavailable := 0
	subjects := append([]*pb.EffortSubject{}, e.Subjects...)
	if e.SupervisorRunId != "" {
		exists := false
		for _, sub := range subjects {
			if sub.RunId == e.SupervisorRunId {
				exists = true
			}
		}
		if !exists {
			subjects = append(subjects, &pb.EffortSubject{Owner: "agent-manager", Kind: "run", Reference: e.SupervisorRunId, RunId: e.SupervisorRunId, Role: "supervisor"})
		}
	}
	for _, sub := range subjects {
		a := &pb.EffortAssignment{Subject: sub, RuntimeState: "unknown", ObservedAt: timestamppb.New(now), Usage: &pb.EffortUsage{Partial: true}}
		row.Assignments = append(row.Assignments, a)
		if sub.Owner != "agent-manager" || sub.Kind != "run" || sub.RunId == "" || s.controller == nil {
			a.UnavailableReason = "current owner telemetry unavailable"
			unavailable++
			allTokens = false
			allTime = false
			continue
		}
		row.Usage.DeclaredRuns++
		id, err := uuid.Parse(sub.RunId)
		if err != nil {
			a.UnavailableReason = "invalid run identity"
			unavailable++
			allTokens = false
			allTime = false
			continue
		}
		run, err := s.controller.GetRun(ctx, id)
		if err != nil || run == nil || run.ID != id {
			a.UnavailableReason = "run owner unavailable"
			unavailable++
			allTokens = false
			allTime = false
			continue
		}
		a.RuntimeState = string(run.Status)
		a.RequestedModel = run.RequestedModel
		a.EffectiveModel = run.ActualModel
		a.EffectiveRunner = run.HarnessKind
		if run.ResolvedConfig != nil {
			a.EffectiveRunner = string(run.ResolvedConfig.RunnerType)
		}
		a.ExecutionIdentity = run.ID.String()
		if run.ImportSourceSessionID != "" {
			a.ExecutionIdentity = domain.NormalizeImportHarness(run.ImportSourceHarness) + ":" + run.ImportSourceSessionID
		} else if run.SessionID != "" {
			a.ExecutionIdentity = a.EffectiveRunner + ":" + run.SessionID
		}
		if config := run.ResolvedConfig; config != nil && config.Admission != nil {
			ad := config.Admission
			a.RequestedModel = ad.RequestedModel
			a.RequestedRunner = ad.RequestedRunner
			a.RequestedReasoning = ad.RequestedEffort
			a.EffectiveReasoning = ad.EffectiveEffort
			if a.EffectiveModel == "" {
				a.EffectiveModel = ad.EffectiveModel
			}
			a.Usage.Limitations = append(a.Usage.Limitations, "effective admission settings are not provider qualification")
		}
		inconsistent := (run.Status.IsActive() || run.Status == domain.RunStatusParked) && run.EndedAt != nil
		if inconsistent {
			a.RuntimeState = "unknown"
			a.UnavailableReason = "owner lifecycle inconsistency: active status retains an ended timestamp; reconcile original run identity"
			if run.ErrorMsg != "" {
				a.Usage.Limitations = append(a.Usage.Limitations, "owner also retains an error alongside active status")
			}
			if run.LastHeartbeat != nil && run.LastHeartbeat.After(*run.EndedAt) {
				a.Usage.Limitations = append(a.Usage.Limitations, "heartbeat is newer than retained end; activity does not resolve prior terminal evidence")
			}
			row.Blockers = append(row.Blockers, "reconcile owner lifecycle evidence for run "+run.ID.String())
			unavailable++
		}
		if run.Summary != nil && run.Summary.TokensUsed > 0 {
			a.Usage.Limitations = append(a.Usage.Limitations, "summary token total is unqualified for per-attempt attribution; cumulative provider values remain unknown")
		}
		if !inconsistent && run.Status.IsTerminal() && run.StartedAt != nil && run.EndedAt != nil {
			v := run.EndedAt.Sub(*run.StartedAt).Seconds()
			if v >= 0 {
				a.Usage.AgentSeconds = &v
			}
		}
		if seen[a.ExecutionIdentity] {
			a.Usage.Limitations = append(a.Usage.Limitations, "shared execution identity excluded from aggregate to avoid double counting")
			continue
		}
		seen[a.ExecutionIdentity] = true
		row.Usage.ObservedRuns++
		if a.Usage.Tokens == nil {
			allTokens = false
		} else {
			tokens += *a.Usage.Tokens
		}
		if a.Usage.AgentSeconds == nil {
			allTime = false
		} else {
			seconds += *a.Usage.AgentSeconds
		}
	}
	if allTokens && row.Usage.ObservedRuns > 0 {
		row.Usage.Tokens = &tokens
	}
	if allTime && row.Usage.ObservedRuns > 0 {
		row.Usage.AgentSeconds = &seconds
	}
	// Usage above covers every role. Product runtime requires an explicit executor
	// role, and cannot borrow liveness from supervision or diagnostic observations.
	// Resolve roles after all owner reads so aliases are independent of ordering.
	executionRoles := map[string]string{}
	identityOf := func(a *pb.EffortAssignment) string {
		if a.ExecutionIdentity != "" {
			return a.ExecutionIdentity
		}
		if a.Subject.RunId != "" {
			return a.Subject.Owner + ":run:" + a.Subject.RunId
		}
		return ""
	}
	recordRole := func(identity, role string) {
		if identity == "" {
			return
		}
		if previous := executionRoles[identity]; previous != "" && previous != role {
			executionRoles[identity] = "conflict"
		} else {
			executionRoles[identity] = role
		}
	}
	for _, a := range row.Assignments {
		role := a.Subject.Role
		if role != "orchestrator" && role != "worker" {
			role = "observation"
		}
		identity := identityOf(a)
		recordRole(identity, role)
		if e.SupervisorRunId != "" && a.Subject.RunId == e.SupervisorRunId {
			recordRole(identity, "observation")
		}
	}
	type runtimeCoverage struct{ declared, active, terminal, unknown int }
	coverage := map[string]*runtimeCoverage{"orchestrator": {}, "worker": {}}
	conflicts := map[string]bool{}
	for _, a := range row.Assignments {
		counts := coverage[a.Subject.Role]
		if counts == nil {
			continue
		}
		counts.declared++
		identity := identityOf(a)
		if executionRoles[identity] == "conflict" {
			counts.unknown++
			a.Usage.Limitations = append(a.Usage.Limitations, "conflicting execution roles; product runtime attribution unknown")
			if !conflicts[identity] {
				row.Limitations = append(row.Limitations, "conflicting execution roles for "+identity+"; product runtime attribution unknown")
				conflicts[identity] = true
			}
			continue
		}
		status := domain.RunStatus(a.RuntimeState)
		switch {
		case a.UnavailableReason != "" || status == domain.RunStatusUnknown:
			counts.unknown++
		case status.IsActive() || status == domain.RunStatusParked:
			counts.active++
		case status.IsTerminal():
			counts.terminal++
		default:
			counts.unknown++
		}
	}
	for _, role := range []string{"orchestrator", "worker"} {
		counts := coverage[role]
		detail := fmt.Sprintf("%s runtime coverage: declared=%d active=%d terminal=%d unknown=%d", role, counts.declared, counts.active, counts.terminal, counts.unknown)
		if counts.declared == 0 {
			detail += "; no explicit assignment, coverage unknown"
		}
		row.Limitations = append(row.Limitations, detail)
	}
	orchestrators, workers := coverage["orchestrator"], coverage["worker"]
	row.RuntimeState = "unknown"
	if orchestrators.active+workers.active > 0 {
		row.RuntimeState = "active"
	} else if orchestrators.declared > 0 && orchestrators.terminal+workers.terminal > 0 && orchestrators.unknown+workers.unknown == 0 {
		row.RuntimeState = "finished"
	}
	row.Limitations = append(row.Limitations, "runtime covers explicit orchestrator/worker assignments; observation roles remain in usage; terminal runtime does not establish outcome acceptance")
	if unavailable > 0 {
		row.Limitations = append(row.Limitations, fmt.Sprintf("%d subject owner observations unavailable", unavailable))
	}
	if dispatchOnlyEnrollment(e) && !e.Withdrawn {
		row.RuntimeState = "authorization"
		row.NextAction = "authorization-only"
		row.Rationale = "recurring dispatch authorization anchor, not an accepted effort or a supervision target"
		row.OutcomeStanding = &pb.EffortOutcomeStanding{State: "not-applicable", Attribution: "owner authorization"}
	} else if e.Withdrawn {
		row.NextAction = "retired"
		row.Rationale = e.WithdrawalReason
	} else if row.NextAction == "authorization-only" {
		row.NextAction = "observe"
		row.Limitations = append(row.Limitations, "source cannot declare itself an authorization-only record")
	} else if row.NextAction == "" {
		row.NextAction = "observe"
		row.Rationale = "no independently evidenced outcome deviation has been classified"
	}
	if e.AuthorizedBy == "" {
		row.Limitations = append(row.Limitations, "steering unavailable: actual owner grant unknown")
	}
	return row
}

// The trigger is a subject cut, not a checksum of the supervisory UI. Inward
// accounting stays visible without waking inference because its own run finished.
func effortSubjectIdentity(row, source *pb.EffortBoardRow) string {
	cut := proto.Clone(source).(*pb.EffortBoardRow)
	cut.ObservedAt, cut.LastAssessment, cut.Usage = nil, nil, nil
	cut.ChangeIdentity, cut.VisibilityChangeIdentity = "", ""
	cut.Enrollment = proto.Clone(row.Enrollment).(*pb.EffortEnrollment)
	e := cut.Enrollment
	e.Revision, e.UpdatedAt, e.Subjects = 0, nil, nil
	e.SupervisorRunId = ""
	cut.Assignments, cut.Directives = nil, nil
	cut.Freshness = row.Freshness
	for _, assignment := range row.Assignments {
		if assignment.Subject.GetRole() == "supervisor" {
			continue
		}
		a := proto.Clone(assignment).(*pb.EffortAssignment)
		a.ObservedAt = nil
		cut.Assignments = append(cut.Assignments, a)
	}
	for _, directive := range row.Directives {
		if directive.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_PENDING {
			continue
		}
		cut.Directives = append(cut.Directives, &pb.EffortDirective{
			DirectiveId: directive.DirectiveId, Delivery: directive.Delivery,
			Acknowledgment: directive.Acknowledgment, ActionRef: directive.ActionRef,
		})
	}
	return effortDigest(cut)
}

// Tick is cheap owner recovery, not agent inference. The surrounding scheduler
// remains alive with no efforts, and failure here does not stop family work.
func (s *EffortService) Tick(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	var scanErr error
	if !now.Before(s.nextScan) {
		_, scanErr = s.reconcileDiscovery(ctx)
		s.nextScan = now.Add(s.config.Interval)
	}
	directives, err := s.repo.ListEffortDirectives(ctx, "", s.directiveCursor, 100)
	if err != nil {
		return err
	}
	if len(directives) == 0 {
		s.directiveCursor = ""
	}
	for _, d := range directives {
		s.directiveCursor = d.DirectiveId
		if d.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_PENDING || d.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN || (d.Kind == pb.WatchActionKind_WATCH_ACTION_KIND_RECOVER_FRESH && d.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED) {
			if _, err = s.deliverDirective(ctx, d); err != nil {
				scanErr = errors.Join(scanErr, err)
			}
		}
	}
	return scanErr
}
