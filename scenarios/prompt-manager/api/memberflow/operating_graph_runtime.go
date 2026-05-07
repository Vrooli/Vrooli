package memberflow

import (
	"path/filepath"
	"strings"
)

const operatingGraphPromptSectionKindTopicContract = "topic-contract"

type OperatingGraphRuntime struct {
	RepoRoot       string
	StoreDir       string
	Members        []MemberTopics
	Contracts      TeamContractRegistry
	PromptSections map[MemberRef][]OperatingGraphPromptSection
}

type OperatingGraphPromptSection struct {
	Team       string
	Member     string
	Kind       string
	SourcePath string
	Content    string
}

func BuildOperatingGraphRuntime(repoRoot, storeDir string) (OperatingGraphRuntime, error) {
	members, err := LoadAll(storeDir)
	if err != nil {
		return OperatingGraphRuntime{}, err
	}
	contracts, err := LoadAllTeamContracts(storeDir)
	if err != nil {
		return OperatingGraphRuntime{}, err
	}
	return OperatingGraphRuntime{
		RepoRoot:       repoRoot,
		StoreDir:       storeDir,
		Members:        members,
		Contracts:      contracts,
		PromptSections: derivedTopicContractPromptSections(members),
	}, nil
}

func derivedTopicContractPromptSections(members []MemberTopics) map[MemberRef][]OperatingGraphPromptSection {
	sections := make(map[MemberRef][]OperatingGraphPromptSection, len(members))
	for _, m := range members {
		ref := m.Ref
		sections[ref] = []OperatingGraphPromptSection{{
			Team:       ref.Team,
			Member:     ref.Member,
			Kind:       operatingGraphPromptSectionKindTopicContract,
			SourcePath: expectedTopicContractSourcePath(ref.Team, ref.Member),
		}}
	}
	return sections
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
