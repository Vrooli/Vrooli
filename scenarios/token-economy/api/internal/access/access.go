// Package access owns authenticated reachability for the two public service
// surfaces. Product authorization and state remain in their domain packages.
package access

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
	earningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/earning/earningv1connect"
)

const (
	ScopeMinter  = "token-economy:minter"
	ScopeHolder  = "token-economy:holder"
	ScopeEarning = "token-economy:earning"
)

var (
	ErrUnauthenticated = errors.New("authentication required")
	ErrUnavailable     = errors.New("scenario-authenticator unavailable")
)

type Identity struct {
	Subject string
	Roles   []string
	Scopes  []string
}

func (i Identity) HasScope(scope string) bool {
	for _, candidate := range i.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

type Validator interface {
	Validate(context.Context, string) (Identity, error)
}

type identityKey struct{}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok
}

func Interceptor(validator Validator) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure
			requiredScope := ""
			switch {
			case strings.HasPrefix(procedure, "/"+accessconnect.MinterServiceName+"/"):
				requiredScope = ScopeMinter
			case strings.HasPrefix(procedure, "/"+accessconnect.HolderServiceName+"/"):
				requiredScope = ScopeHolder
			case strings.HasPrefix(procedure, "/"+earningconnect.EarningServiceName+"/"):
				requiredScope = ScopeEarning
			default:
				return next(ctx, req)
			}
			if validator == nil {
				return nil, connect.NewError(connect.CodeUnavailable, ErrUnavailable)
			}
			token := bearerToken(req.Header().Get("Authorization"))
			identity, err := validator.Validate(ctx, token)
			if err != nil {
				if errors.Is(err, ErrUnavailable) {
					return nil, connect.NewError(connect.CodeUnavailable, ErrUnavailable)
				}
				return nil, connect.NewError(connect.CodeUnauthenticated, ErrUnauthenticated)
			}
			if !identity.HasScope(requiredScope) {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("caller lacks required "+requiredScope+" scope"))
			}
			return next(context.WithValue(ctx, identityKey{}, identity), req)
		}
	})
}

func bearerToken(header string) string {
	prefix, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(prefix, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}
