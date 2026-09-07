package main

import (
	"net/http"
	"strings"

	"git-control-tower/internal/collaboration"
)

type collaborationStatusResponse struct {
	Status       string                           `json:"status"`
	Host         *collaboration.HostIdentity      `json:"host,omitempty"`
	Capabilities []collaboration.CapabilityStatus `json:"capabilities"`
	Reason       string                           `json:"reason,omitempty"`
}

func (s *Server) handleCollaborationStatus(w http.ResponseWriter, r *http.Request) {
	host, err := collaborationHostFromRequest(r)
	if err != nil {
		NewResponse(w).OK(collaborationStatusResponse{Status: "unconfigured", Capabilities: unavailableCapabilities(), Reason: err.Error()})
		return
	}
	NewResponse(w).OK(collaborationStatusResponse{Status: "unavailable", Host: &host, Capabilities: unavailableCapabilities(), Reason: "Integration Hub host adapter is not active"})
}

func collaborationHostFromRequest(r *http.Request) (collaboration.HostIdentity, error) {
	q := r.URL.Query()
	host := collaboration.HostIdentity{
		Kind:           collaboration.HostKind(strings.TrimSpace(q.Get("kind"))),
		InstanceURL:    strings.TrimSpace(q.Get("instance")),
		InstallationID: strings.TrimSpace(q.Get("installation")),
		AccountID:      strings.TrimSpace(q.Get("account")),
		RepositoryID:   strings.TrimSpace(q.Get("repository")),
	}
	if err := host.Validate(); err != nil {
		return collaboration.HostIdentity{}, err
	}
	return host, nil
}

func unavailableCapabilities() []collaboration.CapabilityStatus {
	return []collaboration.CapabilityStatus{
		{Capability: collaboration.CapabilityReadChange, Standing: collaboration.StandingUnconfigured},
		{Capability: collaboration.CapabilityComments, Standing: collaboration.StandingUnconfigured},
		{Capability: collaboration.CapabilityReviews, Standing: collaboration.StandingUnconfigured},
		{Capability: collaboration.CapabilityChecks, Standing: collaboration.StandingUnconfigured},
		{Capability: collaboration.CapabilityReleases, Standing: collaboration.StandingUnconfigured},
		{Capability: collaboration.CapabilityPublish, Standing: collaboration.StandingUnconfigured},
	}
}
