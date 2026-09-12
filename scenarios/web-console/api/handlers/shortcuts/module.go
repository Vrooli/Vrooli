// Package shortcuts is the HTTP-handler home for the shortcuts domain.
// It exposes the generated Connect-RPC ShortcutsService (proto schema:
// packages/proto/schemas/web-console/v1/shortcuts).
//
// RPCs (mounted at /vrooli.web_console.v1.shortcuts.ShortcutsService/...):
//
//	GetEffective  — resolved shortcut list (highest-priority scope wins)
//	ListProfiles  — every stored profile
//	UpsertProfile — create or update a profile by id
//	DeleteProfile — idempotent delete by id
package shortcuts

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	shortcutsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shortcuts/shortcuts_v1connect"

	"web-console/internal/module"
)

// Service is the seam the Connect handler depends on. The concrete
// implementation lives in package main (adapts the existing
// ShortcutStore to satisfy this interface).
type Service interface {
	Effective(ctx context.Context) Effective
	List(ctx context.Context) []Profile
	Upsert(ctx context.Context, req UpsertRequest) (Profile, error)
	Delete(ctx context.Context, id string)
}

// Shortcut is the transport-neutral shortcut shape.
//
// AgentID names the coding agent this entry launches, or is "" for a plain
// operator command. It is resolved by the service, never by a consumer reading
// the command text.
type Shortcut struct {
	Label       string
	Command     string
	Description string
	AgentID     string
}

// Effective is the resolved shortcut list plus the identity of the profile it
// was resolved from, so a client that edits the effective list knows which
// profile to write back to. ProfileID is "" when built-in defaults are being
// served and no profile exists yet.
type Effective struct {
	ProfileID string
	Scope     string
	Name      string
	Shortcuts []Shortcut
}

// Profile is the transport-neutral profile shape.
type Profile struct {
	ID        string
	Scope     string
	Name      string
	Shortcuts []Shortcut
	CreatedAt string
	UpdatedAt string
}

// UpsertRequest is the create-or-update payload.
type UpsertRequest struct {
	ID        string
	Scope     string
	Name      string
	Shortcuts []Shortcut
}

// Module wires the shortcuts domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := shortcutsconnect.NewShortcutsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "shortcuts",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
