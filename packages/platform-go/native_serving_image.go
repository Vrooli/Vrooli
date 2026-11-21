package platform

import (
	"context"
	"os"
)

// OpenInstalledNativeSupervisorImage opens the immutable host selected by the
// exact installed native boot definition. It observes installed code authority,
// not the calling process's mapped image, and requires no manager-self PID.
func OpenInstalledNativeSupervisorImage(ctx context.Context) (*os.File, error) {
	if ctx == nil {
		return nil, ErrOperationIdentity
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return openInstalledNativeSupervisorImage(ctx)
}

// OpenCurrentNativeSystemSupervisorImage opens the exact mapped immutable image
// of the calling machine supervisor after proving its current native manager
// slot. It accepts neither a caller-selected PID nor an executable pathname.
func OpenCurrentNativeSystemSupervisorImage(ctx context.Context) (*os.File, error) {
	if ctx == nil {
		return nil, ErrOperationIdentity
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return openCurrentNativeSystemSupervisorImage(ctx)
}

// OpenOwnedServiceRootImage retains the exact root's birth-established executable
// identity. A Windows root must still possess its inherited image lock.
func OpenOwnedServiceRootImage(ctx context.Context, ref ServiceRef) (*os.File, error) {
	if ctx == nil || ref.ValidateIdentity() != nil || ref.RootPID <= 0 {
		return nil, ErrOperationIdentity
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return openOwnedServiceRootImage(ctx, ref)
}

// OpenOwnedServiceChildImage retains a claimed child executable using its
// pre-resume identity and inherited image lock. Missing birth evidence fails
// closed; an executable pathname alone is not an image observation.
func OpenOwnedServiceChildImage(ctx context.Context, ref ServiceRef, id OwnedImageChildIdentity) (*os.File, error) {
	if ctx == nil || ref.ValidateIdentity() != nil || id.PID <= 0 || id.PID == ref.RootPID {
		return nil, ErrOperationIdentity
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return openOwnedServiceChildImage(ctx, ref, id)
}
