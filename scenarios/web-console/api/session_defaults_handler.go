package main

import (
	"fmt"

	settingsH "web-console/handlers/settings"
	"web-console/internal/backend"
	"web-console/internal/config"
	"web-console/internal/policy"
)

// settingsAdapter satisfies handlers/settings.Service by translating
// between the transport-neutral Defaults shape and the live
// session.Manager + backendRegistry state on the Server.
type settingsAdapter struct {
	server *Server
}

func newSettingsAdapter(s *Server) settingsH.Service {
	return &settingsAdapter{server: s}
}

// SessionDefaultsResponse is the legacy JSON wire shape that the prior
// REST endpoint returned. Retained as a Go type only because in-process
// callers (tests, sibling files) referenced it; not used on the wire
// anymore. Remove once those callers migrate.
type SessionDefaultsResponse struct {
	DefaultBackend string         `json:"default_backend"`
	DefaultPolicy  *policy.Policy `json:"default_policy"`
}

func (a *settingsAdapter) GetDefaults() settingsH.Defaults {
	cfg := a.server.sessions.GetConfig()
	return settingsH.Defaults{
		DefaultBackend: cfg.DefaultBackend,
		DefaultPolicy: settingsH.Policy{
			Mode:     cfg.DefaultPolicyMode,
			Duration: cfg.DefaultPolicyDuration,
		},
	}
}

func (a *settingsAdapter) UpdateDefaults(req settingsH.UpdateDefaultsRequest) (settingsH.Defaults, error) {
	if req.DefaultBackend != nil {
		bid := backend.ID(*req.DefaultBackend)
		if _, ok := a.server.backendRegistry.Get(bid); !ok {
			return settingsH.Defaults{}, fmt.Errorf("%w: unknown backend %q", settingsH.ErrInvalidArgument, *req.DefaultBackend)
		}
		a.server.sessions.SetConfigField(func(cfg *config.Config) {
			cfg.DefaultBackend = *req.DefaultBackend
		})
	}
	if req.DefaultPolicy != nil {
		p := policy.Policy{
			Mode:     policy.Mode(req.DefaultPolicy.Mode),
			Duration: req.DefaultPolicy.Duration,
		}
		if err := policy.Validate(p); err != nil {
			return settingsH.Defaults{}, fmt.Errorf("%w: %s", settingsH.ErrInvalidArgument, err.Error())
		}
		a.server.sessions.SetConfigField(func(cfg *config.Config) {
			cfg.DefaultPolicyMode = string(req.DefaultPolicy.Mode)
			cfg.DefaultPolicyDuration = req.DefaultPolicy.Duration
		})
	}
	return a.GetDefaults(), nil
}
