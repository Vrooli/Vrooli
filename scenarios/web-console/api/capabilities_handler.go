package main

import (
	"context"
	"log"
	"time"

	capabilitiesH "web-console/handlers/capabilities"
)

// capabilitiesAdapter bridges the in-process CapabilityRegistry +
// BackendRegistry to handlers/capabilities.Service so the Connect
// handler can be mounted from main without crossing a package
// boundary the wrong way.
type capabilitiesAdapter struct {
	srv *Server
}

func newCapabilitiesAdapter(s *Server) *capabilitiesAdapter {
	return &capabilitiesAdapter{srv: s}
}

func (a *capabilitiesAdapter) Resolve(ctx context.Context) capabilitiesH.Snapshot {
	start := time.Now()
	caps := a.srv.capabilities.Resolve(ctx)
	log.Printf("capabilities: full resolve took %dms", time.Since(start).Milliseconds())

	snap := capabilitiesH.Snapshot{
		Capabilities: capsToTransport(caps),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
	if a.srv.backendRegistry != nil {
		snap.BackendOptions = backendsToTransport(a.srv.backendRegistry.Available())
		snap.DefaultBackend = string(a.srv.sessions.GetConfig().DefaultBackend)
	}
	return snap
}

func (a *capabilitiesAdapter) Liveness(ctx context.Context) capabilitiesH.Snapshot {
	start := time.Now()
	caps := a.srv.capabilities.ResolveLiveness(ctx)
	log.Printf("capabilities: liveness resolve took %dms", time.Since(start).Milliseconds())
	return capabilitiesH.Snapshot{
		Capabilities: capsToTransport(caps),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
}

func capsToTransport(in []CapabilityState) []capabilitiesH.CapabilityState {
	out := make([]capabilitiesH.CapabilityState, len(in))
	for i, c := range in {
		out[i] = capabilitiesH.CapabilityState{
			ID:             c.ID,
			Name:           c.Name,
			Description:    c.Description,
			DependencyKind: string(c.DependencyKind),
			DependencySlug: c.DependencySlug,
			Features:       c.Features,
			Status:         string(c.Status),
			Message:        c.Message,
			CheckedAt:      c.CheckedAt,
		}
	}
	return out
}

func backendsToTransport(in []BackendDescriptor) []capabilitiesH.BackendOption {
	out := make([]capabilitiesH.BackendOption, len(in))
	for i, b := range in {
		out[i] = capabilitiesH.BackendOption{
			ID:              string(b.ID),
			DisplayName:     b.DisplayName,
			Description:     b.Description,
			SurvivesRestart: b.SurvivesRestart,
			Available:       b.Available,
			Reason:          b.Reason,
		}
	}
	return out
}
