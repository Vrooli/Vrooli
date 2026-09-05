package memberflow

import (
	"sort"
	"strings"
)

type OperatingActorResolver interface {
	Resolve(team string, runtime OperatingGraphRuntime, raw string) []OperatingActorReference
	Expand(team string, runtime OperatingGraphRuntime, ref OperatingActorReference) []OperatingActorReference
}

type DefaultOperatingActorResolver struct {
	Aliases map[string]OperatingActorReference
	Groups  map[string][]OperatingActorReference
}

func NewOperatingActorResolver(meta OperatingGraphMetadata, graphs ...OperatingGraph) DefaultOperatingActorResolver {
	resolver := DefaultOperatingActorResolver{
		Aliases: map[string]OperatingActorReference{},
		Groups:  map[string][]OperatingActorReference{},
	}
	for _, graph := range graphs {
		resolver.addGraphNodeAliases(graph)
	}
	for key, value := range meta.Extra {
		switch {
		case strings.HasPrefix(key, "actor_alias."):
			alias := strings.TrimPrefix(key, "actor_alias.")
			ref := parseTypedOperatingActorReference(value)
			if ref.Kind != "" {
				ref.Raw = alias
				resolver.Aliases[normalizeActorAlias(alias)] = ref
			}
		case strings.HasPrefix(key, "actor_group."):
			group := strings.TrimPrefix(key, "actor_group.")
			resolver.Groups[normalizeActorAlias(group)] = parseActorGroupMembers(value, group)
		}
	}
	return resolver
}

func (r DefaultOperatingActorResolver) addGraphNodeAliases(graph OperatingGraph) {
	for _, node := range graph.Nodes {
		ref := graphNodeActorReference(node)
		if ref.Kind == "" {
			continue
		}
		r.addInferredAlias(node.Value, ref)
		r.addInferredAlias(node.Display, ref)
	}
}

func (r DefaultOperatingActorResolver) addInferredAlias(alias string, ref OperatingActorReference) {
	alias = normalizeActorAlias(alias)
	if alias == "" {
		return
	}
	ref.Raw = alias
	if existing, ok := r.Aliases[alias]; ok && (existing.Kind != ref.Kind || existing.Value != ref.Value) {
		delete(r.Aliases, alias)
		return
	}
	r.Aliases[alias] = ref
}

func graphNodeActorReference(node OperatingGraphNode) OperatingActorReference {
	switch node.Kind {
	case OperatingGraphNodeKindMember:
		return OperatingActorReference{Kind: OperatingActorKindMember, Value: node.Value}
	case OperatingGraphNodeKindTeam:
		return OperatingActorReference{Kind: OperatingActorKindTeam, Value: node.Value}
	case OperatingGraphNodeKindExternal:
		return OperatingActorReference{Kind: OperatingActorKindExternal, Value: node.Value}
	case OperatingGraphNodeKindProcess:
		return OperatingActorReference{Kind: OperatingActorKindProcess, Value: node.Value}
	default:
		return OperatingActorReference{}
	}
}

func (r DefaultOperatingActorResolver) Expand(team string, runtime OperatingGraphRuntime, ref OperatingActorReference) []OperatingActorReference {
	if ref.Kind != OperatingActorKindGroup {
		return []OperatingActorReference{ref}
	}
	members, ok := r.Groups[normalizeActorAlias(ref.Value)]
	if !ok {
		return []OperatingActorReference{ref}
	}
	if len(members) == 1 && members[0].Kind == OperatingActorKindGroup && members[0].Value == "team-members" {
		return r.expandTeamMembers(team, runtime, ref)
	}
	if len(members) == 1 && members[0].Kind == OperatingActorKindGroup && members[0].Value == "none" {
		return nil
	}
	out := make([]OperatingActorReference, 0, len(members))
	for _, member := range members {
		member.Raw = ref.Raw
		out = append(out, member)
	}
	return out
}

func (DefaultOperatingActorResolver) expandTeamMembers(team string, runtime OperatingGraphRuntime, ref OperatingActorReference) []OperatingActorReference {
	if team != "" {
		if contract := runtime.Contracts[team]; contract != nil && contract.Contract != nil && len(contract.Contract.Members) > 0 {
			members := make([]string, 0, len(contract.Contract.Members))
			for member := range contract.Contract.Members {
				members = append(members, member)
			}
			sort.Strings(members)
			out := make([]OperatingActorReference, 0, len(members))
			for _, member := range members {
				out = append(out, OperatingActorReference{Kind: OperatingActorKindMember, Value: member, Raw: ref.Raw})
			}
			return out
		}
	}
	return []OperatingActorReference{ref}
}

func splitActorCell(raw string) []string {
	raw = strings.ReplaceAll(raw, " or ", ",")
	raw = strings.ReplaceAll(raw, " and ", ",")
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseOperatingActorReference(resolver DefaultOperatingActorResolver, raw string) OperatingActorReference {
	cleaned := parseInlineCodeToken(raw)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return OperatingActorReference{}
	}
	if ref := parseTypedOperatingActorReference(cleaned); ref.Kind != "" {
		ref.Raw = raw
		return ref
	}
	if ref, ok := resolver.Aliases[normalizeActorAlias(cleaned)]; ok {
		ref.Raw = raw
		return ref
	}
	normalized := normalizeDocsCell(cleaned)
	normalized = strings.TrimSuffix(normalized, " when relevant")
	switch normalized {
	case "operator":
		return OperatingActorReference{Kind: OperatingActorKindExternal, Value: "operator", Raw: raw}
	case "decision workflow", "live system", "system":
		return OperatingActorReference{Kind: OperatingActorKindExternal, Value: strings.ReplaceAll(normalized, " ", "-"), Raw: raw}
	default:
		return OperatingActorReference{Kind: OperatingActorKindUnknown, Value: normalized, Raw: raw}
	}
}

func (r DefaultOperatingActorResolver) Resolve(team string, runtime OperatingGraphRuntime, raw string) []OperatingActorReference {
	var refs []OperatingActorReference
	for _, part := range splitActorCell(raw) {
		ref := parseOperatingActorReference(r, part)
		if ref.Kind == "" && ref.Value == "" {
			continue
		}
		refs = append(refs, ref)
	}
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].Kind != refs[j].Kind {
			return refs[i].Kind < refs[j].Kind
		}
		if refs[i].Value != refs[j].Value {
			return refs[i].Value < refs[j].Value
		}
		return refs[i].Raw < refs[j].Raw
	})
	return refs
}

func parseTypedOperatingActorReference(raw string) OperatingActorReference {
	if kind, _, value, ok := parseOperatingGraphTypedToken(raw); ok {
		return OperatingActorReference{Kind: OperatingActorKind(kind), Value: value}
	}
	return OperatingActorReference{}
}

func parseActorGroupMembers(raw, group string) []OperatingActorReference {
	normalized := normalizeActorAlias(raw)
	if normalized == "team-members" || normalized == "none" {
		return []OperatingActorReference{{Kind: OperatingActorKindGroup, Value: normalized, Raw: group}}
	}
	var refs []OperatingActorReference
	for _, part := range splitActorCell(raw) {
		ref := parseTypedOperatingActorReference(parseInlineCodeToken(part))
		if ref.Kind == "" {
			continue
		}
		refs = append(refs, ref)
	}
	return refs
}

func normalizeActorAlias(raw string) string {
	raw = parseInlineCodeToken(raw)
	raw = strings.TrimSuffix(normalizeDocsCell(raw), " when relevant")
	return raw
}
