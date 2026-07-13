// Package sessions is the HTTP-handler home for the sessions domain.
// It exposes the generated Connect-RPC SessionsService (proto schema:
// packages/proto/schemas/web-console/v1/sessions).
package sessions

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"

	"web-console/internal/module"
)

// Service is the seam the Connect handler depends on. The concrete
// implementation lives in package main and adapts SessionManager,
// SessionStore, the policy resolver, and the recovery flow.
type Service interface {
	Create(ctx context.Context, in CreateInput) (Session, error)
	List(ctx context.Context) ([]Session, error)
	RecoveryStatus(ctx context.Context) RecoveryStatus
	Get(ctx context.Context, id string) (Session, error)
	Delete(ctx context.Context, id string) error

	ListRecoverable(ctx context.Context) ([]RecoverableSession, error)
	DismissRecoverable(ctx context.Context, id string) error
	Recover(ctx context.Context, in RecoverInput) (RecoverResult, error)

	GetPolicy(ctx context.Context, id string) (PolicyView, error)
	UpdatePolicy(ctx context.Context, id string, policy Policy) (PolicyView, error)
}

// Policy is the transport-neutral expiration policy.
type Policy struct {
	Mode     string
	Duration string
}

// RecoveryStatus is the transport-neutral snapshot of startup session recovery.
// Surfaced on List so a client opening the app mid-recovery sees an honest state.
type RecoveryStatus struct {
	InProgress        bool
	Total             int
	Recovered         int
	AwaitingRecovery  int
	Adopted           int
	StartedAtUnixMs   int64
	CompletedAtUnixMs int64
}

// Session mirrors the legacy SessionResponse JSON shape.
type Session struct {
	ID              string
	Shell           string
	CreatedAt       string
	Cols            int
	Rows            int
	Backend         string
	SurvivesRestart bool
	Policy          Policy
	Busy            bool
	Recovered       bool
	// Provenance. Origin is the closed-set vocabulary
	// "ui" | "programmatic" | "remote".
	Origin       string
	Owner        string
	DisplayLabel string
}

// CreateInput carries the fields a caller may set when creating a session.
// HasPolicy distinguishes "client explicitly set the policy" from
// "client omitted policy → use defaults".
type CreateInput struct {
	Shell         string
	Cols          int
	Rows          int
	Backend       string
	Policy        Policy
	HasPolicy     bool
	LaunchCommand string
	// ExecuteLaunchCommand, when true, pastes LaunchCommand into the new
	// pane's PTY server-side so the command runs exactly once without the
	// client typing it after the WebSocket connects.
	ExecuteLaunchCommand bool
	AgentType            string
	IdempotencyKey       string
	// Provenance. Origin is the closed-set vocabulary
	// "ui" | "programmatic" | "remote"; the empty string normalizes to
	// "programmatic" at Create time.
	Origin       string
	Owner        string
	DisplayLabel string
}

// RecoverInput bundles inputs for the recovery RPC.
type RecoverInput struct {
	ID             string
	IdempotencyKey string
}

// RecoverResult is the result returned by Recover.
type RecoverResult struct {
	OldSessionID    string
	NewSessionID    string
	AgentType       string
	CommandSent     string
	CodexHomeCopied bool
	MessagesCopied  bool
}

// RecoverableSession mirrors the legacy RecoverableSessionResponse.
type RecoverableSession struct {
	ID              string
	Backend         string
	Shell           string
	Cols            int
	Rows            int
	CreatedAt       string
	OrphanedAt      string
	LastActivityAt  string
	AgentType       string
	AgentSessionID  string
	LaunchCommand   string
	CWD             string
	LastRolloutPath string
	Recoverable     bool
	NotRecoverable  string
}

// PolicyView mirrors the legacy PolicyResponse: policy with derived
// expiry. HasExpiry encodes the legacy "expires_at/ttl_seconds present"
// signal so the wire schema can stay flat.
type PolicyView struct {
	SessionID  string
	Policy     Policy
	ExpiresAt  string
	TTLSeconds float64
	HasExpiry  bool
}

// Module wires the sessions domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := sessionsconnect.NewSessionsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "sessions",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
