package handlers

import (
	"context"

	"agent-manager/internal/domain"
	"github.com/vrooli/api-core/identity"
)

// workflowInitiator derives the start classification from verified request
// evidence. Caller-controlled attribution headers never create human authority.
func workflowInitiator(ctx context.Context, agentIdentityToken string) domain.WorkflowInitiator {
	if agentIdentityToken != "" {
		return domain.WorkflowInitiatorAgent
	}
	principal, ok := identity.PrincipalFromContext(ctx)
	if ok && principal.IsVerified() && principal.IsHuman() {
		return domain.WorkflowInitiatorHuman
	}
	return domain.WorkflowInitiatorProgrammatic
}
