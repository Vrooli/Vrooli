package main

// Request identity for synchronous owner run creation, before job storage effects.
import (
	"context"
	"net/http"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/owneridentity"
)

func (s *Server) authorizeCreateRunCaller(r *http.Request) (context.Context, error) {
	verifier := s.runCallerValidator
	if verifier == nil {
		verifier = owneridentity.NewClient(owneridentity.Config{Resolver: discovery.NewResolver(discovery.ResolverConfig{})})
	}
	return owneridentity.AuthorizeCreateRunCaller(r.Context(), r.Header, verifier, time.Now(), true)
}
