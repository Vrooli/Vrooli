package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/tlsinfo"
)

type fakeTLSService struct {
	result tlsinfo.ProbeResult
	err    error
}

func (f fakeTLSService) Probe(_ context.Context, _ string) (tlsinfo.ProbeResult, error) {
	if f.err != nil {
		return tlsinfo.ProbeResult{}, f.err
	}
	return f.result, nil
}

func buildTestDeployment(domainName string) *domain.Deployment {
	manifest := domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS: &domain.ManifestVPS{
				Host: "203.0.113.10",
				Port: 22,
				User: "root",
			},
		},
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite"},
			Resources: []string{},
		},
		Bundle: domain.ManifestBundle{
			IncludePackages: true,
			IncludeAutoheal: true,
		},
		Ports: domain.ManifestPorts{
			"ui":  3000,
			"api": 3001,
			"ws":  3002,
		},
		Edge: domain.ManifestEdge{
			Domain: domainName,
			Caddy:  domain.ManifestCaddy{Enabled: true, Email: "ops@example.com"},
		},
	}
	manifestJSON, _ := json.Marshal(manifest)
	return &domain.Deployment{
		ID:         "dep-1",
		Name:       "test",
		ScenarioID: "landing-page-business-suite",
		Status:     domain.StatusPending,
		Manifest:   manifestJSON,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func newTLSHandlerServer(repo DeploymentRepository, sshRunner ssh.Runner, tlsSvc tlsinfo.Service, alpnRunner tlsinfo.ALPNRunner) *Server {
	srv := &Server{
		config:         &Config{Port: "0"},
		router:         mux.NewRouter(),
		deploymentRepo: repo,
		sshRunner:      sshRunner,
		tlsService:     tlsSvc,
		tlsALPNRunner:  alpnRunner,
	}
	srv.setupRoutes()
	return srv
}

func TestHandleTLSInfoOK(t *testing.T) {
	deployment := buildTestDeployment("example.com")
	repo := &FakeDeploymentRepo{Deployment: deployment}

	tlsSvc := fakeTLSService{
		result: tlsinfo.ProbeResult{
			Domain:        "example.com",
			Valid:         true,
			Issuer:        "Test CA",
			Subject:       "CN=example.com",
			NotBefore:     "Jan 1 00:00:00 2025 UTC",
			NotAfter:      "Feb 1 00:00:00 2025 UTC",
			DaysRemaining: 30,
			SerialNumber:  "deadbeef",
			SANs:          []string{"example.com"},
		},
	}

	alpnRunner := func(_ context.Context, _ string) tlsinfo.ALPNCheck {
		return tlsinfo.ALPNCheck{Status: tlsinfo.ALPNPass, Message: "ok", Protocol: "acme-tls/1"}
	}

	srv := newTLSHandlerServer(repo, &FakeSSHRunner{}, tlsSvc, alpnRunner)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/deployments/dep-1/edge/tls")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var payload TLSInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.OK || !payload.Valid {
		t.Fatalf("expected ok + valid, got %+v", payload)
	}
	if payload.Validation != "time_only" {
		t.Fatalf("expected validation time_only, got %q", payload.Validation)
	}
	if payload.ALPN == nil || payload.ALPN.Status != tlsinfo.ALPNPass {
		t.Fatalf("expected ALPN pass, got %+v", payload.ALPN)
	}
}

func TestHandleTLSInfoProbeError(t *testing.T) {
	deployment := buildTestDeployment("example.com")
	repo := &FakeDeploymentRepo{Deployment: deployment}

	tlsSvc := fakeTLSService{err: fmt.Errorf("probe failed")}
	alpnRunner := func(_ context.Context, _ string) tlsinfo.ALPNCheck {
		return tlsinfo.ALPNCheck{Status: tlsinfo.ALPNWarn, Message: "warn"}
	}

	srv := newTLSHandlerServer(repo, &FakeSSHRunner{}, tlsSvc, alpnRunner)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/deployments/dep-1/edge/tls")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var payload TLSInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.OK {
		t.Fatalf("expected ok=false on probe error")
	}
	if payload.Error == "" {
		t.Fatalf("expected error message")
	}
	if payload.ALPN == nil || payload.ALPN.Status != tlsinfo.ALPNWarn {
		t.Fatalf("expected ALPN warn, got %+v", payload.ALPN)
	}
}

func TestHandleTLSRenewOK(t *testing.T) {
	deployment := buildTestDeployment("example.com")
	repo := &FakeDeploymentRepo{Deployment: deployment}

	cmd := fmt.Sprintf(
		"caddy trust 2>/dev/null; "+
			"systemctl reload caddy && "+
			"sleep 3 && "+
			"curl -sf https://%s >/dev/null && echo 'Certificate valid'",
		"example.com",
	)

	sshRunner := &FakeSSHRunner{
		Responses: map[string]ssh.Result{
			cmd: {ExitCode: 0, Stdout: "Certificate valid"},
		},
	}

	srv := newTLSHandlerServer(repo, sshRunner, fakeTLSService{}, tlsinfo.DefaultALPNRunner)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/deployments/dep-1/edge/tls/renew", bytes.NewBufferString("{}"))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var payload TLSRenewResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.OK {
		t.Fatalf("expected ok=true, got %+v", payload)
	}
}
