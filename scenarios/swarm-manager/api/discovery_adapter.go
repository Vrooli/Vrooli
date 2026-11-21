package main

import (
	"context"
	"errors"

	"swarm-manager/handlers/discovery"
	"swarm-manager/integrations/audiotools"
)

// audioToolsResolverAdapter bridges the audiotools.URLResolver (no context
// in its Resolve method) to the discovery.AudioToolsResolver interface
// the Connect handler depends on. Holds a pointer back into the Server so
// resolver swaps (e.g., env-var override late binding) are picked up live.
type audioToolsResolverAdapter struct {
	srv *Server
}

func newAudioToolsResolverAdapter(s *Server) discovery.AudioToolsResolver {
	return &audioToolsResolverAdapter{srv: s}
}

func (a *audioToolsResolverAdapter) Resolve(ctx context.Context) (string, error) {
	if a == nil || a.srv == nil || a.srv.audioToolsResolver == nil {
		return "", errors.New("audio-tools resolver not configured")
	}
	// Future: thread ctx into audiotools.URLResolver. For now Resolve()
	// uses its own short timeout (see ScenarioResolver.Timeout).
	_ = ctx
	return a.srv.audioToolsResolver.Resolve()
}

// resolveAudioToolsResolver constructs the resolver based on environment.
// AUDIO_TOOLS_URL takes precedence (dev/test); otherwise the lifecycle
// scenario resolver is used, wrapped in a 30s TTL cache.
func resolveAudioToolsResolver() audiotools.URLResolver {
	envResolver := audiotools.EnvResolver{EnvVar: "AUDIO_TOOLS_URL"}
	if v, err := envResolver.Resolve(); err == nil && v != "" {
		return envResolver
	}
	return &audiotools.CachedResolver{
		Inner: audiotools.ScenarioResolver{Slug: "audio-tools"},
	}
}
