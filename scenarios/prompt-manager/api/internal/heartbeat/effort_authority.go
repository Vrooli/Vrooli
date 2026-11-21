package heartbeat

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/api-core/owneridentity"
)

// requireQualifiedCaller enforces that an in-memory queue entry retains an
// original qualified authority mode. Service mode is represented only by a
// package-private, credential-free marker; the worker must mint a fresh bearer
// before making any owner request.
func requireQualifiedCaller(ctx context.Context) error {
	proof := createRunCaller(ctx)
	if proof.origin == createRunCallerOriginService && proof.authorization == "" && proof.runIdentity == "" && proof.binding == "" {
		return nil
	}
	return owneridentity.RequireCreateRunCaller(ctx, time.Now(), owneridentity.CreateRunCallerEnforced())
}

// carryQualifiedCaller forwards the live create-run caller proof across a
// context boundary (e.g. from a request context to a background execution).
func carryQualifiedCaller(from, to context.Context) (context.Context, error) {
	proof := createRunCaller(from)
	if proof.origin == createRunCallerOriginService {
		if proof.authorization == "" || proof.runIdentity != "" || proof.binding != "" {
			return nil, fmt.Errorf("service caller proof is incomplete")
		}
		// The queue may retain the authority mode, never the credential.
		return withCreateRunCallerOrigin(to, "", "", createRunCallerOriginService), nil
	}
	return owneridentity.CarryCreateRunCaller(from, to, time.Now(), owneridentity.CreateRunCallerEnforced())
}
