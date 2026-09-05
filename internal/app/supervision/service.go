// Package supervision computes the database-free supervision set from the
// operator-granted core seed and canonical scenario manifests.
package supervision

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	apicoreset "github.com/vrooli/api-core/coreset"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

// Service is the in-process API used by the control plane and scenarios. It
// deliberately has no database dependency and introduces no stored authority.
type Service struct{}

// Read computes the supervision set rooted at repoRoot.
func (Service) Read(repoRoot string) (apicoreset.Report, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return apicoreset.Report{}, fmt.Errorf("repository root is required")
	}
	authority, err := apicoreset.Load(repoRoot)
	if err != nil {
		return apicoreset.Report{}, fmt.Errorf("load operator core authority: %w", err)
	}
	return Compute(filepath.Join(repoRoot, "scenarios"), authority), nil
}

// Compute walks canonical manifests under scenariosDir. It always retains the
// seed, including when a seed manifest is absent or invalid.
func Compute(scenariosDir string, authority apicoreset.Authority) apicoreset.Report {
	seed := normalized(authority.Seed)
	trusted := normalized(authority.TrustedBase)
	if strings.TrimSpace(scenariosDir) == "" {
		members := seedMembers(seed)
		return apicoreset.Report{
			Source:      "fallback",
			CoreSet:     seed,
			Seed:        seed,
			TrustedBase: trustedBaseSubset(seed, trusted),
			Members:     members,
			MemberCounts: map[string]int{
				apicoreset.MemberKindScenario: len(members),
				apicoreset.MemberKindResource: 0,
			},
		}
	}

	members := make(map[string]apicoreset.Member, len(seed))
	for _, name := range seed {
		members[memberKey(apicoreset.MemberKindScenario, name)] = seedMember(name)
	}
	added := map[string]struct{}{}
	loadErrors := map[string]string{}
	trustedViolations := map[string][]string{}
	trustedSet := stringSet(trusted)

	type queueItem struct {
		name  string
		chain []apicoreset.AttributionStep
	}
	queue := make([]queueItem, 0, len(seed))
	for _, name := range seed {
		queue = append(queue, queueItem{name: name, chain: members[memberKey(apicoreset.MemberKindScenario, name)].AttributionChain})
	}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		manifest, err := scenariomodel.ReadService(filepath.Join(scenariosDir, item.name, ".vrooli", "service.json"))
		if err != nil {
			loadErrors[item.name] = err.Error()
			continue
		}

		if _, isTrusted := trustedSet[item.name]; isTrusted {
			for name, dependency := range manifest.Dependencies.Scenarios {
				name = normalize(name)
				if dependency.SupervisionIntent() == apicoreset.IntentMustStart {
					if _, remainsTrusted := trustedSet[name]; !remainsTrusted {
						trustedViolations[item.name] = append(trustedViolations[item.name], name)
					}
				}
			}
			sort.Strings(trustedViolations[item.name])
		}

		for _, edge := range sortedEdges(manifest) {
			intent := edge.dependency.SupervisionIntent()
			if intent != apicoreset.IntentMustStart && intent != apicoreset.IntentTryStart {
				continue
			}
			name := normalize(edge.name)
			if name == "" {
				continue
			}
			key := memberKey(edge.kind, name)
			if _, seen := members[key]; seen {
				continue
			}
			chain := append([]apicoreset.AttributionStep{{
				Name:              name,
				Kind:              edge.kind,
				DeclaredBy:        item.name,
				SupervisionIntent: intent,
				Source:            "manifest.dependencies." + edge.kind + "s",
			}}, item.chain...)
			members[key] = apicoreset.Member{Name: name, Kind: edge.kind, SupervisionIntent: intent, AttributionChain: chain}
			if edge.kind == apicoreset.MemberKindScenario {
				added[name] = struct{}{}
				queue = append(queue, queueItem{name: name, chain: chain})
			}
		}
	}

	typedMembers := sortedMembers(members)
	coreSet := memberNamesByKind(typedMembers, apicoreset.MemberKindScenario)
	report := apicoreset.Report{
		Source:         "computed",
		CoreSet:        coreSet,
		Seed:           seed,
		AddedByClosure: sortedKeys(added),
		TrustedBase:    trustedBaseSubset(coreSet, trusted),
		Members:        typedMembers,
		MemberCounts: map[string]int{
			apicoreset.MemberKindScenario: len(coreSet),
			apicoreset.MemberKindResource: len(memberNamesByKind(typedMembers, apicoreset.MemberKindResource)),
		},
	}
	if len(loadErrors) > 0 {
		report.LoadErrors = loadErrors
	}
	if len(trustedViolations) > 0 {
		report.TrustedBaseViolations = trustedViolations
	}
	return report
}

type dependencyEdge struct {
	name       string
	kind       string
	dependency scenariomodel.Dependency
}

func sortedEdges(manifest scenariomodel.ServiceManifest) []dependencyEdge {
	edges := make([]dependencyEdge, 0, len(manifest.Dependencies.Resources)+len(manifest.Dependencies.Scenarios))
	for name, dependency := range manifest.Dependencies.Resources {
		edges = append(edges, dependencyEdge{name: name, kind: apicoreset.MemberKindResource, dependency: dependency})
	}
	for name, dependency := range manifest.Dependencies.Scenarios {
		edges = append(edges, dependencyEdge{name: name, kind: apicoreset.MemberKindScenario, dependency: dependency})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].kind == edges[j].kind {
			return edges[i].name < edges[j].name
		}
		return edges[i].kind < edges[j].kind
	})
	return edges
}

func seedMember(name string) apicoreset.Member {
	return apicoreset.Member{
		Name:              name,
		Kind:              apicoreset.MemberKindScenario,
		SupervisionIntent: apicoreset.IntentMustStart,
		AttributionChain: []apicoreset.AttributionStep{{
			Name: name, Kind: apicoreset.MemberKindScenario,
			SupervisionIntent: apicoreset.IntentMustStart, Source: "core.seed",
		}},
	}
}

func seedMembers(seed []string) []apicoreset.Member {
	index := make(map[string]apicoreset.Member, len(seed))
	for _, name := range seed {
		index[memberKey(apicoreset.MemberKindScenario, name)] = seedMember(name)
	}
	return sortedMembers(index)
}

func sortedMembers(index map[string]apicoreset.Member) []apicoreset.Member {
	members := make([]apicoreset.Member, 0, len(index))
	for _, member := range index {
		members = append(members, member)
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i].Kind == members[j].Kind {
			return members[i].Name < members[j].Name
		}
		return members[i].Kind < members[j].Kind
	})
	return members
}

func memberNamesByKind(members []apicoreset.Member, kind string) []string {
	names := make([]string, 0)
	for _, member := range members {
		if member.Kind == kind {
			names = append(names, member.Name)
		}
	}
	return names
}

func trustedBaseSubset(values, trusted []string) []string {
	present := stringSet(values)
	out := make([]string, 0, len(trusted))
	for _, name := range trusted {
		if _, ok := present[name]; ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func normalized(values []string) []string {
	return sortedKeys(stringSet(values))
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = normalize(value); value != "" {
			set[value] = struct{}{}
		}
	}
	return set
}

func normalize(value string) string      { return strings.ToLower(strings.TrimSpace(value)) }
func memberKey(kind, name string) string { return kind + ":" + name }

func sortedKeys(set map[string]struct{}) []string {
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}
