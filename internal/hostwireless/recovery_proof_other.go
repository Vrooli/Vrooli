//go:build !linux

package hostwireless

import (
	"context"
	"errors"
)

func recordNativeRecoveryProof(context.Context, RecoveryProofRequest) (RecoveryProof, error) {
	return RecoveryProof{}, errors.New("stock recovery boot proof requires Linux")
}
func verifyNativeRecoveryProof(context.Context, RecoveryProofRequest) (RecoveryProof, error) {
	return RecoveryProof{}, errors.New("stock recovery boot proof requires Linux")
}
