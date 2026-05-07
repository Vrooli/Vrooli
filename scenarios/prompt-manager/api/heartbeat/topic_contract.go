package heartbeat

import (
	"fmt"

	"prompt-manager/memberflow"
)

type topicContractInputs struct {
	teamID     string
	agentID    string
	memberFlow memberflow.MemberTopics
}

func LoadTopicContractInputs(storeDir, teamID, agentID string) (*topicContractInputs, error) {
	mt, err := memberflow.LoadMember(storeDir, teamID, agentID)
	if err != nil {
		return nil, fmt.Errorf("load topics.json: %w", err)
	}
	return &topicContractInputs{
		teamID:     teamID,
		agentID:    agentID,
		memberFlow: mt,
	}, nil
}

func RenderTopicContract(in *topicContractInputs) string {
	if in == nil {
		return ""
	}
	return memberflow.RenderTopicContract(in.teamID, in.agentID, in.memberFlow)
}
