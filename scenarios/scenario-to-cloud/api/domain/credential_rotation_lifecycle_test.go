package domain

import "testing"

func TestCredentialRotationTransitionMatrixAndInvalidStateImmutability(t *testing.T) {
	states := []CredentialRotationState{
		RotationPlanned,
		RotationNewVersionCreated,
		RotationProviderPrepared,
		RotationConsumersUpdated,
		RotationVerified,
		RotationPendingOperatorInput,
		RotationOldVersionRevoked,
		RotationComplete,
		RotationRecovering,
		RotationFailed,
		RotationRevocationIncomplete,
		"future",
	}
	legal := map[[2]CredentialRotationState]struct{}{}
	for _, pair := range [][2]CredentialRotationState{
		{RotationPlanned, RotationPlanned},
		{RotationPlanned, RotationNewVersionCreated},
		{RotationPlanned, RotationProviderPrepared},
		{RotationPlanned, RotationOldVersionRevoked},
		{RotationPlanned, RotationRevocationIncomplete},
		{RotationPlanned, RotationFailed},
		{RotationNewVersionCreated, RotationProviderPrepared},
		{RotationNewVersionCreated, RotationRecovering},
		{RotationNewVersionCreated, RotationFailed},
		{RotationProviderPrepared, RotationConsumersUpdated},
		{RotationProviderPrepared, RotationRecovering},
		{RotationProviderPrepared, RotationFailed},
		{RotationConsumersUpdated, RotationConsumersUpdated},
		{RotationConsumersUpdated, RotationVerified},
		{RotationConsumersUpdated, RotationRecovering},
		{RotationConsumersUpdated, RotationFailed},
		{RotationVerified, RotationVerified},
		{RotationVerified, RotationPendingOperatorInput},
		{RotationVerified, RotationOldVersionRevoked},
		{RotationVerified, RotationComplete},
		{RotationVerified, RotationFailed},
		{RotationPendingOperatorInput, RotationPendingOperatorInput},
		{RotationPendingOperatorInput, RotationVerified},
		{RotationPendingOperatorInput, RotationOldVersionRevoked},
		{RotationPendingOperatorInput, RotationFailed},
		{RotationOldVersionRevoked, RotationComplete},
		{RotationRecovering, RotationFailed},
		{RotationRevocationIncomplete, RotationOldVersionRevoked},
	} {
		legal[pair] = struct{}{}
	}
	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"/"+string(to), func(t *testing.T) {
				rotation := CredentialRotation{State: from}
				_, valid := legal[[2]CredentialRotationState{from, to}]
				if LegalCredentialRotationTransition(from, to) != valid {
					t.Fatalf("transition table edge %q -> %q did not match expected matrix", from, to)
				}
				err := rotation.TransitionTo(to)
				if valid && err != nil {
					t.Fatalf("legal transition rejected: %v", err)
				}
				if !valid && (err == nil || rotation.State != from) {
					t.Fatalf("illegal transition err=%v mutated state to %q", err, rotation.State)
				}
				if valid && rotation.State != to {
					t.Fatalf("transition ended in %q, want %q", rotation.State, to)
				}
			})
		}
	}
	rotation := CredentialRotation{State: RotationPlanned}
	if err := rotation.TransitionTo("future"); err == nil || rotation.State != RotationPlanned {
		t.Fatalf("unknown next state mutated rotation: state=%q err=%v", rotation.State, err)
	}
}
