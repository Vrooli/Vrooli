package main

import (
	"errors"

	"github.com/gorilla/mux"

	"scenario-to-cloud/deployment"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/secrets"
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
	}
	srv.setupRoutes()
	return srv
}
