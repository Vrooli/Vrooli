package heartbeat

import (
	"fmt"

	"prompt-manager/internal/memberflow"
)

type topicContractInputs struct {
	teamID     string
	agentID    string
	memberFlow memberflow.MemberTopics
	catalog    []memberflow.TopicCatalogEntry
}

func LoadTopicContractInputs(configDir, teamID, agentID string) (*topicContractInputs, error) {
	mt, err := memberflow.LoadMember(configDir, teamID, agentID)
	if err != nil {
		return nil, fmt.Errorf("load topics.json: %w", err)
	}
	in := &topicContractInputs{
		teamID:     teamID,
		agentID:    agentID,
		memberFlow: mt,
	}
	if contracts, err := memberflow.LoadAllTeamContracts(configDir); err == nil {
		in.catalog = contracts.TopicCatalog(teamID)
	}
	return in, nil
}

func RenderTopicContract(in *topicContractInputs) string {
	if in == nil {
		return ""
	}
	return memberflow.RenderTopicContract(in.teamID, in.agentID, in.memberFlow, in.catalog...)
}
