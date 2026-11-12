package platform

import (
	"context"
	"errors"
	"strings"
)

// OwnedServiceAccount names an existing native user by its canonical uid:/sid:
// principal. Resolution is read-only; it neither provisions an account nor
// grants process or device access. Workload launch checks actual caller identity
// independently of this metadata lookup.
type OwnedServiceAccount struct {
	Name      string `json:"name"`
	Principal string `json:"principal"`
}

func ResolveOwnedServiceAccount(ctx context.Context, name string) (OwnedServiceAccount, error) {
	if ctx == nil {
		return OwnedServiceAccount{}, errors.New("platform: native service account context is required")
	}
	if err := ctx.Err(); err != nil {
		return OwnedServiceAccount{}, err
	}
	if name == "" || len(name) > 256 || strings.ContainsAny(name, "\x00\r\n") {
		return OwnedServiceAccount{}, ErrOperationIdentity
	}
	return resolveOwnedServiceAccount(ctx, name)
}
