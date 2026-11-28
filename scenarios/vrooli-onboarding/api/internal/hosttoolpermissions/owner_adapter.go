package hosttoolpermissions

import (
	"context"
	"encoding/json"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
)

// NewOwnerWriter routes the one allowed host tool mutation through the
// OperatorState owner's masked CAS and durable receipt transaction.
func NewOwnerWriter(service operatorstateapi.Service) StateWriter {
	return ownerWriter{service: service}
}

type ownerWriter struct{ service operatorstateapi.Service }

func (w ownerWriter) CompareAndSetHostTool(ctx context.Context, m StateMutation) (StateReceipt, error) {
	actor, ok := identity.PrincipalFromContext(ctx)
	if !ok || !actor.IsHuman() || !actor.IsVerified() || actor.Subject != m.ActorID {
		return StateReceipt{}, operatorstate.ErrMutationReceiptConflict
	}
	// OperatorState keeps the legacy manifest name key for surface parity; the
	// receipt subject still binds the selected-closure scenario and tool name.
	key := m.Name
	patch, err := json.Marshal(map[string]any{"host_tools": map[string]any{key: map[string]any{"opted_in": m.OptedIn}}})
	if err != nil {
		return StateReceipt{}, err
	}
	permissionID := PermissionPrefix + m.Scenario + "/" + m.Name
	subject := SubjectKind + ":" + m.Scenario + "/" + m.Name
	stored, err := w.service.PatchWithReceipt(ctx, patch, []string{"host_tools"}, m.ExpectedRevision, operatorstate.MutationReceiptBinding{Owner: Owner, PermissionID: permissionID, ActorID: m.ActorID, Target: LocalTarget, Subject: subject, RequestID: m.RequestID, ExpectedRevision: m.ExpectedRevision, RequestDigest: m.RequestDigest, DecisionRef: m.DecisionRef})
	if err != nil {
		return StateReceipt{}, err
	}
	return StateReceipt{RequestID: stored.RequestID, ActorID: stored.ActorID, Target: stored.Target, Subject: stored.Subject, ExpectedRevision: stored.ExpectedRevision, NewRevision: stored.ResultRevision, RequestDigest: stored.RequestDigest, Outcome: stored.Outcome, At: stored.At}, nil
}

var _ StateWriter = ownerWriter{}
