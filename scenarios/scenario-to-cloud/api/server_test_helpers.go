package main

import (
	"context"
	"errors"
	"time"

	"scenario-to-cloud/authz"
	"scenario-to-cloud/deployment"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/secrets"
	"scenario-to-cloud/tlsinfo"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

func newTestServer() *Server {
	srv := &Server{
		config:           &ServerConfig{Port: "0"},
		router:           mux.NewRouter(),
		progressHub:      deployment.NewHub(),
		sshRunner:        &FakeSSHRunner{DefaultErr: errors.New("ssh not configured")},
		scpRunner:        &FakeSCPRunner{DefaultErr: errors.New("scp not configured")},
		secretsFetcher:   &FakeSecretsFetcher{},
		secretsGenerator: secrets.NewGenerator(),
		dnsService:       dns.NewService(dns.NetResolver{}),
		tlsService:       tlsinfo.NewService(tlsinfo.WithTimeout(2 * time.Second)),
		tlsALPNRunner:    tlsinfo.DefaultALPNRunner,
		deploymentRepo:   &FakeDeploymentRepo{},
	}
	srv.authz = newTestEnforcer(srv, testOperator())
	srv.setupRoutes()
	return srv
}

// testOperator is the default fully-scoped human principal used by handler
// tests that are not about authorization.
func testOperator() identity.Principal {
	return identity.Principal{
		Kind: identity.ActorHuman, Subject: "test-operator", Realm: "test", Verified: true,
		Source: identity.SourcePersonalLocal, Sources: []identity.AuthSource{identity.SourcePersonalLocal},
		Scopes: authz.Scopes(),
	}
}

// newTestEnforcer builds the boundary with a fake provider. The provider is a
// pointer so authorization tests can swap the principal or revoke it between
// admission and effect. Host "example.com" is the httptest default.
func newTestEnforcer(srv *Server, principal identity.Principal) *authz.Enforcer {
	return newTestEnforcerWith(srv, principal, testEnforcerOptions{})
}

type testEnforcerOptions struct {
	Policy authz.PolicyDocument
	Logger func(string, map[string]any)
}

func newTestEnforcerWith(srv *Server, principal identity.Principal, opts testEnforcerOptions) *authz.Enforcer {
	provider := &FakePrincipalProvider{Principal: principal}
	logger := opts.Logger
	if logger == nil {
		logger = srv.log
	}
	enforcer, err := authz.New(authz.Config{
		Authn:        authn.Config{Providers: []authn.Provider{provider}},
		Mode:         authz.ModePersonalLocal,
		Policy:       authz.NewStaticPolicy(opts.Policy),
		AllowedHosts: []string{"example.com", "localhost", "127.0.0.1"},
		BindLoopback: true,
		Logger:       logger,
	}, authz.TargetResolverFunc(func(ctx context.Context, id string) (authz.Target, bool, error) {
		return targetResolver{repo: srv.deploymentRepo}.ResolveTarget(ctx, id)
	}))
	if err != nil {
		panic(err)
	}
	srv.testProvider = provider
	return enforcer
}
