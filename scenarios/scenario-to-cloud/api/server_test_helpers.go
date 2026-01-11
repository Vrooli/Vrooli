package main

import (
	"errors"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/deployment"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/secrets"
	"scenario-to-cloud/tlsinfo"
)

func newTestServer() *Server {
	srv := &Server{
		config:           &Config{Port: "0"},
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
	srv.setupRoutes()
	return srv
}
