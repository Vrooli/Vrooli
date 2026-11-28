package coresetpermissions

import (
	"context"
	"encoding/json"

	"github.com/vrooli/api-core/coreset"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
)

// NewOwnerProvider binds reads and writes to the actual onboarding-owned
// OperatorState service. Its PatchWithReceipt path performs masked core-only
// CAS and persists the identity-bound request receipt atomically.
func NewOwnerProvider(service operatorstateapi.Service) *Provider {
	return NewProvider(service, ownerWriter{service: service})
}

type ownerWriter struct{ service operatorstateapi.Service }

func (w ownerWriter) CompareAndSetCore(ctx context.Context, mutation StateMutation) (StateReceipt, error) {
	// This permission is a complete replacement of the CoreSet authority.
	// Emit empty arrays explicitly so an operator can remove the last member of
	// either tier through the same durable owner patch.
	state, err := json.Marshal(map[string]any{"core": map[string]any{
		"seed": mutation.Authority.Seed, "trusted_base": mutation.Authority.TrustedBase,
		"important": append([]coreset.ComponentID{}, mutation.Authority.Important...),
		"members":   append([]coreset.ComponentID{}, mutation.Authority.CoreMembers...),
	}})
	if err != nil {
		return StateReceipt{}, err
	}
	receipt, err := w.service.PatchWithReceipt(ctx, state, []string{"core"}, mutation.ExpectedRevision, operatorstate.MutationReceiptBinding{
		Owner: Owner, PermissionID: PermissionID, ActorID: mutation.ActorID, Target: LocalTarget,
		Subject: WriteSubject, RequestID: mutation.RequestID, ExpectedRevision: mutation.ExpectedRevision,
		RequestDigest: mutation.RequestDigest, AuthorityDigest: mutation.AuthorityDigest,
		DecisionRef: "operator-state-core:" + mutation.ActorID + ":local:mutation",
	})
	if err != nil {
		return StateReceipt{}, err
	}
	return StateReceipt{
		RequestID: receipt.RequestID, ActorID: receipt.ActorID, Target: receipt.Target,
		Subject: receipt.Subject, ExpectedRevision: receipt.ExpectedRevision,
		NewRevision: receipt.ResultRevision, RequestDigest: receipt.RequestDigest,
		AuthorityDigest: receipt.AuthorityDigest, Outcome: receipt.Outcome, CreatedAt: receipt.At,
	}, nil
}

var _ StateReader = operatorstateapi.Service{}
var _ StateWriter = ownerWriter{}
