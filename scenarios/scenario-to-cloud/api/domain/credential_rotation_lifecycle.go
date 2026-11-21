package domain

import "fmt"

// credentialRotationTransitions is the legal state graph shared by rotate,
// revoke, recover, and break-glass operations in the credential ledger.
var credentialRotationTransitions = map[CredentialRotationState]map[CredentialRotationState]struct{}{
	RotationPlanned: {
		RotationPlanned: {}, RotationNewVersionCreated: {}, RotationProviderPrepared: {},
		RotationOldVersionRevoked: {}, RotationRevocationIncomplete: {}, RotationFailed: {},
	},
	RotationNewVersionCreated: {
		RotationProviderPrepared: {}, RotationRecovering: {}, RotationFailed: {},
	},
	RotationProviderPrepared: {
		RotationConsumersUpdated: {}, RotationRecovering: {}, RotationFailed: {},
	},
	RotationConsumersUpdated: {
		RotationConsumersUpdated: {}, RotationVerified: {}, RotationRecovering: {}, RotationFailed: {},
	},
	RotationVerified: {
		RotationVerified: {}, RotationPendingOperatorInput: {}, RotationOldVersionRevoked: {}, RotationComplete: {}, RotationFailed: {},
	},
	RotationPendingOperatorInput: {
		RotationPendingOperatorInput: {}, RotationVerified: {}, RotationOldVersionRevoked: {}, RotationFailed: {},
	},
	RotationOldVersionRevoked:    {RotationComplete: {}},
	RotationRecovering:           {RotationFailed: {}},
	RotationRevocationIncomplete: {RotationOldVersionRevoked: {}},
}

// LegalCredentialRotationTransition reports whether a state change is part of
// the credential-operation graph. Same-state edges are limited to progress
// receipts for resumable operations.
func LegalCredentialRotationTransition(from, to CredentialRotationState) bool {
	_, ok := credentialRotationTransitions[from][to]
	return ok
}

// TransitionTo is the sole in-memory owner for credential-rotation state.
func (r *CredentialRotation) TransitionTo(next CredentialRotationState) error {
	if r == nil {
		return fmt.Errorf("cannot transition a nil credential rotation")
	}
	if !ValidCredentialRotationState(r.State) {
		return fmt.Errorf("unknown credential-rotation state %q", r.State)
	}
	if !ValidCredentialRotationState(next) {
		return fmt.Errorf("unknown credential-rotation state %q", next)
	}
	if !LegalCredentialRotationTransition(r.State, next) {
		return fmt.Errorf("invalid credential-rotation transition %q -> %q", r.State, next)
	}
	r.State = next
	return nil
}

func ValidCredentialRotationState(state CredentialRotationState) bool {
	switch state {
	case RotationPlanned, RotationNewVersionCreated, RotationProviderPrepared,
		RotationConsumersUpdated, RotationVerified, RotationPendingOperatorInput,
		RotationOldVersionRevoked, RotationComplete, RotationRecovering,
		RotationFailed, RotationRevocationIncomplete:
		return true
	default:
		return false
	}
}
