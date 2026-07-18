package agentsessions

import (
	"context"
	"strings"

	"swarm-manager/internal/apierr"
)

type StartupBriefResolver interface {
	ResolveSessionStartupBrief(ctx context.Context, kind Kind, limits ContextLimits) (ContextItem, error)
}

func (s *Service) StartupBrief(ctx context.Context, sessionID string) (ContextItem, error) {
	session, err := s.store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return ContextItem{}, mapStoreError(err)
	}
	resolver, ok := s.startupBriefResolver()
	if !ok {
		return ContextItem{}, apierr.Unavailable("agent session startup brief is unavailable")
	}
	item, err := resolver.ResolveSessionStartupBrief(ctx, session.Kind, contextLimitsForKind(session.Kind))
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
	items, err := s.contextResolver.ResolveSessionMessageContext(ctx, normalized, contextLimitsForKind(session.Kind))
	if err != nil {
		return nil, err
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

func contextLimitsForKind(kind Kind) ContextLimits {
	common := map[ContextType]int{
		ContextBacklogItem:          8,
		ContextInitiative:           4,
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
