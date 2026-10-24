package orchestration

import (
	"agent-manager/internal/domain"
	"context"
	"errors"
	coreidentity "github.com/vrooli/api-core/identity"
	"time"
)

func authenticatedInternalCreateRunFixture(o *Orchestrator, req CreateRunRequest) CreateRunRequest {
	o.ownerIdentity = createOwnerVerifier(func(_ context.Context, token string) (coreidentity.Principal, error) {
		if token != "disposable-test-operator" {
			return coreidentity.Principal{}, errors.New("invalid fixture credential")
		}
		return coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "fixture-operator", Scopes: []string{"agent-manager:write", "agent-manager:read", "agent-manager:supervise"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
	})
	req.OwnerToken = "disposable-test-operator"
	return req
}
func authenticatedAcceptedRunFixture(run *domain.Run) {
	run.OwnerSubject = "fixture-operator"
	run.OwnerScopes = []string{"agent-manager:write", "agent-manager:read", "agent-manager:supervise"}
	if run.ResolvedConfig == nil {
		run.ResolvedConfig = &domain.RunConfig{}
	}
	if run.ResolvedConfig.Admission == nil {
		run.ResolvedConfig.Admission = &domain.RunAdmission{}
	}
	run.ResolvedConfig.Admission.CreateCaller = &domain.CreateRunCaller{Kind: "human", Subject: run.OwnerSubject}
}
