package memberflow

import (
	"path/filepath"
	"strings"
)

const operatingGraphPromptSectionKindTopicContract = "topic-contract"

type OperatingGraphPromptSectionSourceKind string

const (
	OperatingGraphPromptSectionSourceLive    OperatingGraphPromptSectionSourceKind = "live"
	OperatingGraphPromptSectionSourceDerived OperatingGraphPromptSectionSourceKind = "derived"
)

type OperatingGraphRuntime struct {
	RepoRoot       string
	StoreDir       string
	Members        []MemberTopics
	Contracts      TeamContractRegistry
	PromptSections map[MemberRef][]OperatingGraphPromptSection
}

type OperatingGraphPromptSection struct {
	Team       string                                `json:"team"`
	Member     string                                `json:"member"`
	Kind       string                                `json:"kind"`
	SourcePath string                                `json:"source_path,omitempty"`
	Content    string                                `json:"content,omitempty"`
	SourceKind OperatingGraphPromptSectionSourceKind `json:"source_kind,omitempty"`
}

func BuildOperatingGraphRuntime(repoRoot, configDir string) (OperatingGraphRuntime, error) {
	members, err := LoadAll(configDir)
	if err != nil {
		return OperatingGraphRuntime{}, err
	}
	contracts, err := LoadAllTeamContracts(configDir)
	if err != nil {
		return OperatingGraphRuntime{}, err
	}
	return OperatingGraphRuntime{
		RepoRoot:       repoRoot,
		StoreDir:       configDir,
		Members:        members,
		Contracts:      contracts,
		PromptSections: derivedTopicContractPromptSections(members, contracts),
	}, nil
}

func derivedTopicContractPromptSections(members []MemberTopics, registries ...TeamContractRegistry) map[MemberRef][]OperatingGraphPromptSection {
	var contracts TeamContractRegistry
	if len(registries) > 0 {
		contracts = registries[0]
	}
	sections := make(map[MemberRef][]OperatingGraphPromptSection, len(members))
	for _, m := range members {
		ref := m.Ref
		sections[ref] = []OperatingGraphPromptSection{{
			Team:       ref.Team,
			Member:     ref.Member,
			Kind:       operatingGraphPromptSectionKindTopicContract,
			SourcePath: expectedTopicContractSourcePath(ref.Team, ref.Member),
			Content:    RenderTopicContract(ref.Team, ref.Member, m, contracts.TopicCatalog(ref.Team)...),
			SourceKind: OperatingGraphPromptSectionSourceDerived,
		}}
	}
	return sections
}

func expectedTopicContractContent(runtime OperatingGraphRuntime, team, member string) (string, bool) {
	for _, m := range runtime.Members {
		if m.Ref.Team == team && m.Ref.Member == member {
			return RenderTopicContract(team, member, m, runtime.Contracts.TopicCatalog(team)...), true
		}
	}
	return "", false
}

func (r OperatingGraphRuntime) topicDeclared(team, topic string) bool {
	for _, rel := range BuildRuntimeOperatingRelationships(r, team) {
		switch rel.Kind {
		case operatingRelTopicIntake, operatingRelTopicRequiredRead, operatingRelTopicEvidenceConsumed, operatingRelTopicOutput:
			if topicsOverlap(rel.Topic, topic) {
				return true
			}
		}
	}
	return false
}

func runtimeMemberTopicsPath(runtime OperatingGraphRuntime, team, member string) string {
	return relativeRuntimePath(runtime, filepath.Join(runtime.StoreDir, "teams", team, "members", member, "topics.json"))
}

func expectedTopicContractSourcePath(team, member string) string {
	return filepath.ToSlash(filepath.Join("teams", team, "members", member, "topics.json"))
}

func runtimeTeamPath(runtime OperatingGraphRuntime, team string) string {
	return relativeRuntimePath(runtime, filepath.Join(runtime.StoreDir, "teams", team, "team.json"))
}

func relativeRuntimePath(runtime OperatingGraphRuntime, path string) string {
	if runtime.RepoRoot != "" {
		if rel, err := filepath.Rel(runtime.RepoRoot, path); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(path)
}
