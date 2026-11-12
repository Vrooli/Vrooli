//go:build !linux

package hostwireless

import (
	"context"
	"errors"
)

func nativePrivateIndexes(context.Context, *CopiedActivation, ActivationPlan) error {
	return errors.New("native private module indexing requires Linux")
}
