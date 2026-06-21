// Package authz owns the operator-boundary authorization seam for privileged
// Tunnel Manager mutations. The default local deployment remains open to the
// operator on localhost; production/shared deployments can enforce a static
// bearer token without changing UI/CLI/API contracts.
package authz

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
)

const (
	OperationConfigSync        = "config.sync"
	OperationConfigSwitchMode  = "config.switch_mode"
	OperationConfigCredentials = "config.credentials"
	OperationRoutesCreate      = "routes.create"
	OperationRoutesUpdate      = "routes.update"
	OperationRoutesDelete      = "routes.delete"
	OperationExposureExpose    = "exposure.expose"
	OperationExposureExtend    = "exposure.extend_lease"
	OperationExposureRevoke    = "exposure.revoke_lease"
	OperationExposureReconcile = "exposure.reconcile"
	OperationRecoveryRecover   = "recovery.recover"
)

const operatorTokenHeader = "X-Vrooli-Operator-Token" // #nosec G101 -- header name, not a credential value.

var (
	ErrTokenRequired = errors.New("operator authorization token required")
	ErrTokenDenied   = errors.New("operator authorization token denied")
)

type Authorizer interface {
	Authorize(ctx context.Context, operation string, headers http.Header) error
}

type StaticTokenAuthorizer struct {
	Enforced bool
	Token    string
}

func AllowLocalOperator() StaticTokenAuthorizer {
	return StaticTokenAuthorizer{}
}

func FromEnv() StaticTokenAuthorizer {
	enforced := truthyEnv("TUNNEL_MANAGER_AUTHZ_ENFORCED")
	token, ok := lookupTrimmedEnv("TUNNEL_MANAGER_OPERATOR_TOKEN")
	if !ok {
		token, _ = lookupTrimmedEnv("API_TOKEN")
	}
	return StaticTokenAuthorizer{Enforced: enforced, Token: token}
}

func (a StaticTokenAuthorizer) Authorize(_ context.Context, _ string, headers http.Header) error {
	if !a.Enforced {
		return nil
	}
	if a.Token == "" {
		return ErrTokenRequired
	}
	candidate := tokenFromHeaders(headers)
	if candidate == "" {
		return ErrTokenRequired
	}
	if subtle.ConstantTimeCompare([]byte(candidate), []byte(a.Token)) != 1 {
		return ErrTokenDenied
	}
	return nil
}

func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrTokenRequired) {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}
	if errors.Is(err, ErrTokenDenied) {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func tokenFromHeaders(headers http.Header) string {
	if headers == nil {
		return ""
	}
	if raw := strings.TrimSpace(headers.Get(operatorTokenHeader)); raw != "" {
		return raw
	}
	auth := strings.TrimSpace(headers.Get("Authorization"))
	if auth == "" {
		return ""
	}
	parts := strings.Fields(auth)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func truthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func truthyEnv(name string) bool {
	raw, ok := lookupTrimmedEnv(name)
	return ok && truthy(raw)
}

func lookupTrimmedEnv(name string) (string, bool) {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return "", false
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	return value, true
}
