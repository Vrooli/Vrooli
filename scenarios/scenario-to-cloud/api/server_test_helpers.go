package main

import (
	"errors"

	"github.com/gorilla/mux"

	"scenario-to-cloud/dns"
)

func newTestServer() *Server {
	srv := &Server{
		config:           &Config{Port: "0"},
		router:           mux.NewRouter(),
		progressHub:      NewProgressHub(),
		sshRunner:        &FakeSSHRunner{DefaultErr: errors.New("ssh not configured")},
		scpRunner:        &FakeSCPRunner{DefaultErr: errors.New("scp not configured")},
		secretsFetcher:   &FakeSecretsFetcher{},
		secretsGenerator: NewSecretsGenerator(),
		dnsService:       dns.NewService(dns.NetResolver{}),
	}
	srv.setupRoutes()
	return srv
}
