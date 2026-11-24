package safeguardpermissions

import (
	"context"
	"encoding/json"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
)

// NewOwnerWriter uses the OperatorState owner's transaction for masked CAS
// and durable replay receipts; it never keeps a second copy of safeguards.
func NewOwnerWriter(service operatorstateapi.Service) Writer { return ownerWriter{service: service} }

type ownerWriter struct{ service operatorstateapi.Service }

func (w ownerWriter) CompareAndSetSafeguard(ctx context.Context, m Mutation) (Receipt, error) {
	choice := map[string]any{}
	if m.OptedIn != nil {
		choice["opted_in"] = *m.OptedIn
	} else if len(m.Config) > 0 {
		config := make(map[string]any, len(m.Config))
		for key, raw := range m.Config {
			var value any
			if err := json.Unmarshal(raw, &value); err != nil {
				return Receipt{}, err
			}
			config[key] = value
		}
		choice["config"] = config
	} else {
		var value any
		if err := json.Unmarshal(m.Value, &value); err != nil {
			return Receipt{}, err
		}
		choice["config"] = map[string]any{m.ConfigKey: value}
	}
	patch, err := json.Marshal(map[string]any{"host_safeguards": map[string]any{m.Name: choice}})
	if err != nil {
		return Receipt{}, err
	}
	actor, ok := identity.PrincipalFromContext(ctx)
	if !ok || !actor.IsHuman() || actor.Subject != m.ActorID {
		return Receipt{}, operatorstate.ErrMutationReceiptConflict
	}
	stored, err := w.service.PatchWithReceipt(ctx, patch, []string{"host_safeguards"}, m.ExpectedRevision, operatorstate.MutationReceiptBinding{
		Owner: Owner, PermissionID: PermissionID(m.Name), ActorID: m.ActorID, Target: LocalTarget,
		Subject: SubjectKind + ":" + m.Name, RequestID: m.RequestID, ExpectedRevision: m.ExpectedRevision,
		RequestDigest: m.RequestDigest, DecisionRef: m.DecisionRef,
	})
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{RequestID: stored.RequestID, ActorID: stored.ActorID, Target: stored.Target, Subject: stored.Subject, ExpectedRevision: stored.ExpectedRevision, NewRevision: stored.ResultRevision, RequestDigest: stored.RequestDigest, Outcome: stored.Outcome, At: stored.At}, nil
}

var _ Writer = ownerWriter{}
