package teams

import (
	"context"
	"fmt"

	"prompt-manager/store"
)

type validationError struct {
	message string
}

func (e *validationError) Error() string {
	return e.message
}

func newValidationError(message string) error {
	return &validationError{message: message}
}

func isValidationError(err error) bool {
	_, ok := err.(*validationError)
	return ok
}

func (h *Handlers) teamMemberSet(ctx context.Context, teamID string) (map[string]struct{}, error) {
	if h.relationStore == nil {
		return nil, fmt.Errorf("relation store not configured")
	}
	members, err := h.relationStore.ListTeamMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	memberSet := make(map[string]struct{}, len(members))
	for _, member := range members {
		memberSet[member.AgentID] = struct{}{}
	}
	return memberSet, nil
}

func validateMemberExists(memberSet map[string]struct{}, agentID, roleLabel string) error {
	if agentID == "" {
		return newValidationError(fmt.Sprintf("%s is required", roleLabel))
	}
	if _, ok := memberSet[agentID]; !ok {
		return newValidationError(fmt.Sprintf("%s %s is not a team member", roleLabel, agentID))
	}
	return nil
}

func storeMessageToDTO(message store.TeamMessage) TeamMessageDTO {
	return TeamMessageDTO{
		ID:          message.ID,
		TeamID:      message.TeamID,
		FromAgentID: message.FromAgentID,
		ToAgentID:   message.ToAgentID,
		Content:     message.Content,
		CreatedAt:   message.CreatedAt,
	}
}
