package main

import (
	"context"
	"errors"

	discoveryH "web-console/handlers/discovery"
)

// audioToolsResolverAdapter bridges the server-held audiotoolsint
// URLResolver to the discovery handler's seam.
type audioToolsResolverAdapter struct {
	srv *Server
}

func newAudioToolsResolverAdapter(s *Server) discoveryH.AudioToolsResolver {
	return &audioToolsResolverAdapter{srv: s}
}

func (a *audioToolsResolverAdapter) Resolve(_ context.Context) (string, error) {
	if a == nil || a.srv == nil || a.srv.audioToolsResolver == nil {
		return "", errors.New("audiotools: resolver not configured")
	}
	return a.srv.audioToolsResolver.Resolve()
}
