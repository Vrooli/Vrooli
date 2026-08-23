package agentsessions

import (
	"context"
	"strings"

	"swarm-manager/internal/apierr"
)

type StartupBriefResolver interface {
	ResolveSessionStartupBrief(ctx context.Context, kind Kind, jobID string, attached []ContextItem, limits ContextLimits) (ContextItem, error)
}

func (s *Service) StartupBrief(ctx context.Context, sessionID string) (ContextItem, error) {
	store, err := s.storeFor(ctx)
	if err != nil {
		return ContextItem{}, err
	}
	session, err := store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return ContextItem{}, mapStoreError(err)
	}
	resolver, ok := s.startupBriefResolver()
	if !ok {
		return ContextItem{}, apierr.Unavailable("agent session startup brief is unavailable")
	}
	item, err := resolver.ResolveSessionStartupBrief(ctx, session.Kind, session.StarterJob, nil, contextLimitsForKind(session.Kind))
	if err != nil {
		return ContextItem{}, err
	}
	if item.SelectedAt == "" {
		item.SelectedAt = nowRFC3339()
	}
	item.Summary = truncateRunes(strings.TrimSpace(item.Summary), contextLimitsForKind(session.Kind).MaxSummaryRunes)
	return item, nil
}

func (s *Service) resolveMessageContext(ctx context.Context, session Session, refs []ContextRef) ([]ContextItem, error) {
	normalized, err := normalizeContextRefs(session.Kind, refs)
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return nil, nil
	}
	if s.contextResolver == nil {
		return nil, apierr.Unavailable("agent session context resolution is unavailable")
	}
	limits := contextLimitsForKind(session.Kind)
	regularRefs := make([]ContextRef, 0, len(normalized))
	startupIndex := -1
	for i, ref := range normalized {
		if ref.Type == ContextStartupBrief {
			startupIndex = i
			continue
		}
		regularRefs = append(regularRefs, ref)
	}
	items, err := s.contextResolver.ResolveSessionMessageContext(ctx, regularRefs, limits)
	if err != nil {
		return nil, err
	}
	if startupIndex >= 0 {
		resolver, ok := s.startupBriefResolver()
		if !ok {
			return nil, apierr.Unavailable("agent session startup brief is unavailable")
		}
		brief, briefErr := resolver.ResolveSessionStartupBrief(ctx, session.Kind, session.StarterJob, items, limits)
		if briefErr != nil {
			return nil, briefErr
		}
		withBrief := make([]ContextItem, 0, len(items)+1)
		withBrief = append(withBrief, items[:startupIndex]...)
		withBrief = append(withBrief, brief)
		withBrief = append(withBrief, items[startupIndex:]...)
		items = withBrief
	}
	now := nowRFC3339()
	for i := range items {
		if items[i].SelectedAt == "" {
			items[i].SelectedAt = now
		}
		items[i].Summary = truncateRunes(strings.TrimSpace(items[i].Summary), contextLimitsForKind(session.Kind).MaxSummaryRunes)
	}
	return items, nil
}

func normalizeContextRefs(kind Kind, refs []ContextRef) ([]ContextRef, error) {
	limits := contextLimitsForKind(kind)
	if len(refs) > limits.MaxTotal {
		return nil, apierr.BadRequest("no more than %d context items are allowed for %s sessions", limits.MaxTotal, kind)
	}
	seen := make(map[string]struct{}, len(refs))
	counts := make(map[ContextType]int)
	normalized := make([]ContextRef, 0, len(refs))
	for _, ref := range refs {
		contextType := ContextType(strings.TrimSpace(string(ref.Type)))
		value := strings.TrimSpace(ref.Ref)
		if !IsKnownContextType(contextType) {
			return nil, apierr.BadRequest("context type is invalid")
		}
		if value == "" {
			return nil, apierr.BadRequest("context ref is required")
		}
		if max, ok := limits.MaxPerType[contextType]; !ok || max <= 0 {
			return nil, apierr.BadRequest("context type %q is not allowed for %s sessions", contextType, kind)
		}
		key := string(contextType) + "\x00" + value
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		counts[contextType]++
		if counts[contextType] > limits.MaxPerType[contextType] {
			return nil, apierr.BadRequest("too many %s context items; max is %d", contextType, limits.MaxPerType[contextType])
		}
		normalized = append(normalized, ContextRef{Type: contextType, Ref: value})
	}
	if len(normalized) > limits.MaxTotal {
		return nil, apierr.BadRequest("no more than %d context items are allowed for %s sessions", limits.MaxTotal, kind)
	}
	return normalized, nil
}

func (s *Service) ChangeKind(ctx context.Context, req ChangeKindRequest) (ChangeKindResult, error) {
	if !s.sessionKindAvailable(req.Kind) {
		return ChangeKindResult{}, apierr.BadRequest("kind is not declared in the transition registry")
	}
	store, err := s.storeFor(ctx)
	if err != nil {
		return ChangeKindResult{}, err
	}
	session, err := store.LoadSession(strings.TrimSpace(req.SessionID))
	if err != nil {
		return ChangeKindResult{}, mapStoreError(err)
	}
	if session.Status != StatusDraft || strings.TrimSpace(session.RunID) != "" {
		return ChangeKindResult{}, apierr.Conflict("agent session kind cannot change from status %q", session.Status)
	}

	_, dropped, err := filterContextRefsForKind(req.Kind, req.ContextRefs)
	if err != nil {
		return ChangeKindResult{}, err
	}
	cleared := !starterJobAllowedForKind(session.StarterJob, req.Kind)
	session.Kind = req.Kind
	session.SkillID = s.skillIDForKind(req.Kind)
	if cleared {
		session.StarterJob = ""
	}
	session.UpdatedAt = nowRFC3339()
	if err := store.SaveSession(session); err != nil {
		return ChangeKindResult{}, err
	}
	updated, err := store.LoadSession(session.ID)
	if err != nil {
		return ChangeKindResult{}, err
	}
	return ChangeKindResult{Session: updated, DroppedContext: dropped, StarterJobCleared: cleared}, nil
}

func filterContextRefsForKind(kind Kind, refs []ContextRef) ([]ContextRef, []ContextRef, error) {
	if len(refs) > contextLimitsForKind(kind).MaxTotal {
		return nil, nil, apierr.BadRequest("no more than %d context items are allowed for %s sessions", contextLimitsForKind(kind).MaxTotal, kind)
	}
	kept := make([]ContextRef, 0, len(refs))
	dropped := make([]ContextRef, 0)
	seen := map[string]struct{}{}
	for _, raw := range refs {
		ref := ContextRef{Type: ContextType(strings.TrimSpace(string(raw.Type))), Ref: strings.TrimSpace(raw.Ref)}
		if !IsKnownContextType(ref.Type) || ref.Ref == "" {
			return nil, nil, apierr.BadRequest("context type and ref must be valid")
		}
		key := string(ref.Type) + "\x00" + ref.Ref
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		allowed := contextLimitsForKind(kind).MaxPerType[ref.Type] > 0
		if ref.Type == ContextStartupBrief && ref.Ref != StartupBriefRefForKind(kind) {
			allowed = false
		}
		if allowed {
			kept = append(kept, ref)
		} else {
			dropped = append(dropped, ref)
		}
	}
	return kept, dropped, nil
}

func contextLimitsForKind(kind Kind) ContextLimits {
	common := map[ContextType]int{
		ContextBacklogItem:          8,
		ContextGoal:                 1,
		ContextCapture:              4,
		ContextExecution:            6,
		ContextAgentActivity:        6,
		ContextScenario:             3,
		ContextSession:              2,
		ContextOperationsBriefing:   1,
		ContextStartupBrief:         1,
		ContextPlanDependencyCycles: 1,
		ContextPlanEta:              1,
	}
	allowed := map[Kind]map[ContextType]bool{
		KindSwarmOperations: {
			ContextStartupBrief: true, ContextOperationsBriefing: true, ContextGoal: true, ContextBacklogItem: true,
			ContextExecution: true, ContextAgentActivity: true, ContextCapture: true, ContextSession: true,
			ContextPlanDependencyCycles: true, ContextPlanEta: true,
		},
		KindWorkflowAuthoring: {
			ContextStartupBrief: true, ContextGoal: true, ContextBacklogItem: true, ContextScenario: true, ContextSession: true,
		},
		KindMetaOrchestration: {
			ContextStartupBrief: true, ContextGoal: true, ContextBacklogItem: true, ContextCapture: true,
			ContextScenario: true, ContextSession: true, ContextPlanDependencyCycles: true, ContextPlanEta: true,
		},
	}[kind]
	for contextType := range common {
		if !allowed[contextType] {
			delete(common, contextType)
		}
	}
	return ContextLimits{Kind: kind, MaxTotal: 12, MaxPerType: common, MaxSummaryRunes: 1200}
}

func refsWithAutoContext(kind Kind, refs []ContextRef, policy AutoContextPolicy, resolverAvailable bool) []ContextRef {
	if policy == AutoContextNone || !resolverAvailable {
		return refs
	}
	startupRef := StartupBriefRefForKind(kind)
	if startupRef == "" {
		return refs
	}
	for _, ref := range refs {
		if ref.Type == ContextStartupBrief || (kind == KindSwarmOperations && ref.Type == ContextOperationsBriefing) {
			return refs
		}
	}
	return append([]ContextRef{{Type: ContextStartupBrief, Ref: startupRef}}, refs...)
}

func (s *Service) startupBriefResolverAvailable() bool {
	_, ok := s.startupBriefResolver()
	return ok
}

func (s *Service) startupBriefResolver() (StartupBriefResolver, bool) {
	if s.contextResolver == nil {
		return nil, false
	}
	resolver, ok := s.contextResolver.(StartupBriefResolver)
	return resolver, ok
}
