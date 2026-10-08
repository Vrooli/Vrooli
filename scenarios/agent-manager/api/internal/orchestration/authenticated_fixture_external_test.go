package orchestration_test

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"context"
	"errors"
	coreidentity "github.com/vrooli/api-core/identity"
	"time"
)

type fixtureOperatorVerifier struct{}

func (fixtureOperatorVerifier) Verify(_ context.Context, token string) (coreidentity.Principal, error) {
	if token != "disposable-test-operator" {
		return coreidentity.Principal{}, errors.New("invalid fixture credential")
	}
	return coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "fixture-operator", Scopes: []string{"agent-manager:write", "agent-manager:read", "agent-manager:supervise"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func fixtureOwnerIdentityOption() orchestration.Option {
	return orchestration.WithOwnerIdentity(fixtureOperatorVerifier{})
}

// Explicit proof at each public request; never automatic HTTP middleware.
func authenticatedCreateRunFixture(req orchestration.CreateRunRequest) orchestration.CreateRunRequest {
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
