package memberflow

import (
	"path/filepath"
	"strings"
)

type OperatingGraphRuntime struct {
	RepoRoot  string
	StoreDir  string
	Members   []MemberTopics
	Contracts TeamContractRegistry
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
		RepoRoot:  repoRoot,
		StoreDir:  storeDir,
		Members:   members,
		Contracts: contracts,
	}, nil
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
