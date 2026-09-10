package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/edge"
	"scenario-to-cloud/reach/sshadapter"
	"scenario-to-cloud/tlsinfo"

	edgev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/edge"
)

const edgeCanaryToken = "canary-secret-9f3a7c1e"

type fakeEdgeDNS struct{ answers map[string][]string }

func (f fakeEdgeDNS) ResolveHost(_ context.Context, host string) domain.DNSLookupResult {
	ips, ok := f.answers[strings.ToLower(host)]
	if !ok {
		return domain.DNSLookupResult{Host: host, Error: &domain.DNSLookupError{Kind: domain.DNSLookupNotFound, Message: "no such host"}}
	}
	return domain.DNSLookupResult{Host: host, IPs: ips}
}

func (f fakeEdgeDNS) CheckDomainReachability(context.Context, string) domain.ReachabilityResult {
	return domain.ReachabilityResult{}
}

func (f fakeEdgeDNS) CompareDomainToVPS(context.Context, string, string) domain.DNSComparisonResult {
	return domain.DNSComparisonResult{}
}

func edgeTestDeployment(t *testing.T, domainName, host string) *domain.Deployment {
	t.Helper()
	m := domain.CloudManifest{
		Version:  "1.0.0",
		Target:   domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: host, Port: 22, User: "root", Workdir: "/root/Vrooli"}},
		Scenario: domain.ManifestScenario{ID: "app"},
		Ports:    domain.ManifestPorts{"ui": 3000, "api": 3001, "metrics": 9100},
		Edge:     domain.ManifestEdge{Domain: domainName, Caddy: domain.ManifestCaddy{Enabled: true, Email: "ops@example.test"}},
		Secrets: &domain.ManifestSecrets{BundleSecrets: []domain.BundleSecretPlan{{
			ID: "cloudflare-token", Class: "user_prompt", Required: false,
			Target:     domain.BundleSecretTarget{Type: "env", Name: domain.CloudflareAPITokenKey},
			Descriptor: &domain.DescriptorAddress{LogicalID: "vrooli/app", Field: "cloudflare-api-token"},
		}}},
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return &domain.Deployment{ID: "dep-edge-1", Name: "app", ScenarioID: "app", Environment: "staging", Status: domain.StatusDeployed, Manifest: raw, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func newEdgeTestServer(t *testing.T, dep *domain.Deployment, dnsAnswers map[string][]string, probe tlsinfo.ProbeResult, journal string) *Server {
	t.Helper()
	previousListeners := edgeListenersForDeployment
	edgeListenersForDeployment = func(context.Context, *domain.Deployment, domain.CloudManifest) ([]domain.ClosureListener, map[string]bool, error) {
		return []domain.ClosureListener{
			{ID: "app/ui", Owner: "scenario:app", PortName: "ui", Visibility: domain.EdgeVisibilityPublicViaEdge},
			{ID: "app/api", Owner: "scenario:app", PortName: "api", Visibility: domain.EdgeVisibilityPrivate},
			{ID: "postgres/server", Owner: "resource:postgres", PortName: "server", Visibility: ""},
		}, map[string]bool{"postgres": true}, nil
	}
	previousExternal := externalReadinessProbe
	externalReadinessProbe = func(context.Context, string, int) error { return nil }
	t.Cleanup(func() {
		edgeListenersForDeployment = previousListeners
		externalReadinessProbe = previousExternal
	})
	runner := &FakeSSHRunner{Handler: func(command string) (sshadapter.Result, error, bool) {
		if strings.Contains(command, "journalctl") {
			return sshadapter.Result{ExitCode: 0, Stdout: journal}, nil, true
		}
		if strings.Contains(command, "'ss' '-ltnH'") {
			return sshadapter.Result{ExitCode: 0, Stdout: "LISTEN"}, nil, true
		}
		if strings.Contains(command, "curl") {
			return sshadapter.Result{ExitCode: 0}, nil, true
		}
		return sshadapter.Result{}, errors.New("unexpected command: " + command), true
	}}
	srv := newTLSHandlerServer(&FakeDeploymentRepo{Deployment: dep}, runner, fakeTLSService{result: probe}, func(context.Context, string) tlsinfo.ALPNCheck { return tlsinfo.ALPNCheck{Status: tlsinfo.ALPNPass} })
	srv.repo = nil
	srv.dnsService = fakeEdgeDNS{answers: dnsAnswers}
	srv.registerEdgeRoutes(srv.router.PathPrefix("/api/v1").Subrouter())
	return srv
}

func getEdgeObservation(t *testing.T, srv *Server) (*edgev1.GetEdgeObservationResponse, []byte) {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/deployments/dep-edge-1/edge", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp edgev1.GetEdgeObservationResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	return &resp, rec.Body.Bytes()
}

// [REQ:STC-P0-036] P14 step 16 / EDGE canary: the observation carries
// routes, private listeners, DNS, TLS and ACME facts and never a provider
// token, a private key or a journal token, even when the target's journal
// and the caller's environment hold one.
func TestEdgeObservationCarriesExposureFactsWithoutSecrets(t *testing.T) {
	t.Setenv(domain.CloudflareAPITokenKey, edgeCanaryToken)
	dep := edgeTestDeployment(t, "app.example.test", "203.0.113.10")
	journal := `{"level":"error","logger":"tls.obtain","msg":"could not get certificate from issuer","error":"HTTP 401","token":"` + edgeCanaryToken + `"}` + "\n"
	probe := tlsinfo.ProbeResult{Domain: "app.example.test", Valid: true, Issuer: "R11", NotAfter: time.Now().Add(40 * 24 * time.Hour).UTC().Format("Jan 2 15:04:05 2006 MST"), DaysRemaining: 40, SANs: []string{"app.example.test"}}
	srv := newEdgeTestServer(t, dep, map[string][]string{"app.example.test": {"203.0.113.10"}}, probe, journal)
	resp, raw := getEdgeObservation(t, srv)
	if strings.Contains(string(raw), edgeCanaryToken) || strings.Contains(string(raw), "cloudflare-api-token-value") {
		t.Fatalf("observation leaked a secret: %s", raw)
	}
	obs := resp.GetObservation()
	if resp.GetSchemaVersion() != edge.SchemaVersion || obs.GetDeploymentId() != "dep-edge-1" || obs.GetDomain() != "app.example.test" || !strings.HasPrefix(obs.GetSpecDigest(), "sha256:") {
		t.Fatalf("observation identity = %+v", obs)
	}
	if len(obs.GetRoutes()) != 1 || obs.GetRoutes()[0].GetUpstreamPort() != 3000 || obs.GetRoutes()[0].GetListenerId() != "app/ui" {
		t.Fatalf("routes = %+v", obs.GetRoutes())
	}
	private := map[string]string{}
	for _, l := range obs.GetPrivateListeners() {
		private[l.GetId()] = l.GetReason()
	}
	if private["app/api"] != edge.ReasonDeclaredPrivate || private["postgres/server"] != edge.ReasonDatabase || private["app/metrics"] != edge.ReasonMetrics || private["management/bridge"] != edge.ReasonManagement {
		t.Fatalf("private listeners = %v", private)
	}
	if len(obs.GetDns()) != 1 || !obs.GetDns()[0].GetMatch() || obs.GetDns()[0].GetIpv4().GetState() != domain.EdgeBindingMatch || obs.GetDns()[0].GetIpv6().GetState() != domain.EdgeBindingAbsent {
		t.Fatalf("dns = %+v", obs.GetDns())
	}
	if len(obs.GetTls()) != 1 || obs.GetTls()[0].GetRenewalState() != domain.EdgeRenewalFailed || obs.GetTls()[0].GetIssuer() != "R11" || obs.GetTls()[0].GetDaysLeft() < 38 {
		t.Fatalf("tls = %+v", obs.GetTls())
	}
	if obs.GetAcmeEnvironment() != domain.ACMEEnvironmentStaging {
		t.Fatalf("a staging deployment must issue against staging ACME, got %s", obs.GetAcmeEnvironment())
	}
	if obs.GetReadiness().GetLocal().GetStatus() != "passed" || obs.GetReadiness().GetExternal().GetStatus() != "passed" {
		t.Fatalf("readiness = %+v", obs.GetReadiness())
	}
	var sawRenew bool
	for _, a := range obs.GetNextActions() {
		if strings.Contains(a.GetReference(), "tls-renew") {
			sawRenew = true
		}
		if strings.Contains(a.GetReference(), "sudo") || strings.Contains(a.GetReference(), "kill ") {
			t.Fatalf("next action is an ad-hoc shell repair: %+v", a)
		}
	}
	if !sawRenew {
		t.Fatalf("renewal failure must produce a renew action: %+v", obs.GetNextActions())
	}
	if obs.GetProducerRef() != edge.ProducerRef || obs.GetObservedAt() == nil {
		t.Fatalf("producer/observed_at = %+v", obs)
	}
}

// [REQ:STC-P0-036] P14-A05: wrong DNS blocks the external readiness claim
// even though the public probe and the local probe pass.
func TestEdgeObservationWrongDNSCannotClaimExternalReadiness(t *testing.T) {
	dep := edgeTestDeployment(t, "app.example.test", "203.0.113.10")
	probe := tlsinfo.ProbeResult{Domain: "app.example.test", Valid: true, Issuer: "R11", NotAfter: time.Now().Add(60 * 24 * time.Hour).UTC().Format("Jan 2 15:04:05 2006 MST"), SANs: []string{"app.example.test"}}
	srv := newEdgeTestServer(t, dep, map[string][]string{"app.example.test": {"198.51.100.9"}}, probe, "")
	resp, _ := getEdgeObservation(t, srv)
	obs := resp.GetObservation()
	if obs.GetDns()[0].GetMatch() || obs.GetDns()[0].GetReasonCode() != edge.ReasonDNSMismatch {
		t.Fatalf("dns = %+v", obs.GetDns())
	}
	if obs.GetReadiness().GetLocal().GetStatus() != "passed" || obs.GetReadiness().GetExternal().GetStatus() != "blocked" || obs.GetReadiness().GetExternal().GetReasonCode() != edge.ReasonExternalBlocked {
		t.Fatalf("readiness = %+v", obs.GetReadiness())
	}
}

// [REQ:STC-P0-036] Step 5: an IPv6-only target binds on AAAA alone and an
// A record pointing at another host is reported as the mismatch it is.
func TestEdgeObservationIPv6OnlyTarget(t *testing.T) {
	dep := edgeTestDeployment(t, "app.example.test", "2001:db8::10")
	probe := tlsinfo.ProbeResult{Domain: "app.example.test", Valid: true, Issuer: "R11", NotAfter: time.Now().Add(60 * 24 * time.Hour).UTC().Format("Jan 2 15:04:05 2006 MST"), SANs: []string{"app.example.test"}}
	srv := newEdgeTestServer(t, dep, map[string][]string{"app.example.test": {"2001:db8::10"}}, probe, "")
	resp, _ := getEdgeObservation(t, srv)
	dns := resp.GetObservation().GetDns()[0]
	if !dns.GetMatch() || dns.GetIpv6().GetState() != domain.EdgeBindingMatch || dns.GetIpv4().GetState() != domain.EdgeBindingAbsent {
		t.Fatalf("ipv6-only dns = %+v", dns)
	}
	if resp.GetObservation().GetReadiness().GetExternal().GetStatus() != "passed" {
		t.Fatalf("readiness = %+v", resp.GetObservation().GetReadiness())
	}
	srv = newEdgeTestServer(t, dep, map[string][]string{"app.example.test": {"2001:db8::10", "198.51.100.9"}}, probe, "")
	resp, _ = getEdgeObservation(t, srv)
	dns = resp.GetObservation().GetDns()[0]
	if dns.GetMatch() || dns.GetIpv4().GetState() != domain.EdgeBindingMismatch {
		t.Fatalf("stray A record dns = %+v", dns)
	}
}
