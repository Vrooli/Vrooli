package authn

import (
	"context"
	"github.com/vrooli/api-core/identity"
)

// RequestOperationAuthority is verified by a domain-owned ingress, never a
// caller-supplied Principal scope. Implementations must pin the principal,
// complete request, operation, target and validity, and recheck current policy
// at the domain's final effect boundary. This does not authorize other APIs.
type RequestOperationAuthority interface {
	AuthorizeOperation(context.Context, identity.Principal, string, string, string) error
}
type requestOperationKey struct{}

// WithRequestOperationAuthority is a trusted ingress API, like WithPrincipal.
// It must only receive an already verified, immutable authority object.
func WithRequestOperationAuthority(ctx context.Context, a RequestOperationAuthority) context.Context {
	return context.WithValue(ctx, requestOperationKey{}, a)
}
func RequestOperationAuthorityFromContext(ctx context.Context) (RequestOperationAuthority, bool) {
	a, ok := ctx.Value(requestOperationKey{}).(RequestOperationAuthority)
	return a, ok && a != nil
}
