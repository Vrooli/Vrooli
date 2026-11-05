package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/edge"
	"scenario-to-cloud/edgesvc"
	"scenario-to-cloud/tlsinfo"
	"scenario-to-cloud/vps"

	edgev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/edge"
)

// edgeObservationTimeout bounds one DNS + TLS + journal + readiness pass.
const edgeObservationTimeout = 90 * time.Second

// registerEdgeRoutes mounts the typed edge observation beside the legacy
// dns-check/tls routes and the Connect EdgeService.
func (s *Server) registerEdgeRoutes(api *mux.Router) {
	api.HandleFunc("/deployments/{id}/edge", s.handleGetEdgeObservation).Methods("GET")
	path, handler := edgesvc.NewService(s).Handler()
	s.router.PathPrefix(path).Handler(handler)
}

// edgeListenersForDeployment resolves the closure listeners for a manifest.
// The closure service is owned by the Server so distinct instances never
// share catalogs or failure state.
func (s *Server) edgeListenersForDeployment(ctx context.Context, dep *domain.Deployment, manifest domain.CloudManifest) ([]domain.ClosureListener, map[string]bool, error) {
	if s != nil && s.edgeListenersOverride != nil {
		return s.edgeListenersOverride(ctx, dep, manifest)
	}
	svc, err := s.closureService()
	if err != nil {
		return nil, nil, err
	}
	environment := ""
	if dep != nil {
		environment = dep.Environment
	}
	closure, err := svc.Resolve(ctx, manifest.Scenario.ID, environment)
	if err != nil {
		return nil, nil, err
	}
	databases := map[string]bool{}
	for _, component := range closure.Components {
		if component.Kind != domain.ClosureKindResource {
			continue
		}
		lower := strings.ToLower(component.ID)
		for _, marker := range []string{"postgres", "mysql", "mariadb", "redis", "mongo", "qdrant", "database"} {
			if strings.Contains(lower, marker) {
				databases[component.ID] = true
			}
		}
	}
	return closure.Listeners, databases, nil
}

// edgeSpecFor derives the declared edge contract for a deployment. When the
// closure cannot be resolved the manifest ports alone decide (legacy rule:
// only "ui" is public), which is recorded on the spec's listener ids.
func (s *Server) edgeSpecFor(ctx context.Context, dc *DeploymentContext) (domain.EdgeSpec, error) {
	listeners, databases, err := s.edgeListenersForDeployment(ctx, dc.Deployment, dc.Manifest)
	if err != nil {
		listeners, databases = nil, nil
	}
	environment := ""
	if dc.Deployment != nil {
		environment = dc.Deployment.Environment
	}
	var host string
	if dc.Manifest.Target.VPS != nil {
		host = dc.Manifest.Target.VPS.Host
	}
	inputs := edge.PolicyInputs{
		DeploymentID:      dc.ID,
		ScenarioID:        dc.Manifest.Scenario.ID,
		Environment:       environment,
		Domain:            dc.Manifest.Edge.Domain,
		Listeners:         listeners,
		Ports:             dc.Manifest.Ports,
		TargetHost:        host,
		ACMEEmail:         dc.Manifest.Edge.Caddy.Email,
		ACMEEnvironment:   dc.Manifest.Edge.ACMEEnvironment,
		ACMEAuthority:     edge.ACMEAuthorityFromEnvironment(os.LookupEnv),
		DNSProvider:       dns.ProviderBinding(dc.Manifest),
		DatabaseResources: databases,
	}
	if net.ParseIP(strings.TrimSpace(host)) == nil && strings.TrimSpace(host) != "" && s.dnsService != nil {
		lookup := s.dnsService.ResolveHost(ctx, host)
		for _, ip := range lookup.IPs {
			if strings.Contains(ip, ":") {
				inputs.ObservedIPv6 = append(inputs.ObservedIPv6, ip)
			} else {
				inputs.ObservedIPv4 = append(inputs.ObservedIPv4, ip)
			}
		}
	}
	return edge.Derive(inputs)
}

// inspectEdge runs the lifecycle checks for a deployment's routes.
func (s *Server) inspectEdge(parent context.Context, dc *DeploymentContext) (*edgev1.EdgeObservation, error) {
	ctx, cancel := context.WithTimeout(parent, edgeObservationTimeout)
	defer cancel()
	spec, err := s.edgeSpecFor(ctx, dc)
	if err != nil {
		return nil, err
	}
	in := edge.ObserveInputs{Spec: spec, Now: time.Now().UTC()}
	if s.dnsService != nil {
		in.Lookup = func(ctx context.Context, host string) ([]string, error) {
			lookup := s.dnsService.ResolveHost(ctx, host)
			if lookup.Error != nil {
				return nil, lookup.Error
			}
			return lookup.IPs, nil
		}
	}
	if s.tlsService != nil {
		in.Certificate = func(ctx context.Context, host string) (edge.CertificateFacts, error) {
			probe, err := s.tlsService.Probe(ctx, host)
			if err != nil {
				return edge.CertificateFacts{}, err
			}
			return certificateFacts(probe), nil
		}
	}
	prober := s.proberFor(dc)
	in.Journal = s.readCaddyJournal(ctx, prober)
	in.Local = func(ctx context.Context, _ string, port int) error {
		// The listener is proven by the socket the lifecycle owner holds on
		// the loopback address; ss is a bounded observation program.
		res, err := prober.Observe(ctx, "ss", "-ltnH", "sport", "=", ":"+strconv.Itoa(port))
		if err != nil {
			return err
		}
		if res.ExitCode != 0 || strings.TrimSpace(res.Stdout) == "" {
			return fmt.Errorf("no listener bound on loopback port %d", port)
		}
		return nil
	}
	in.External = externalReadinessProbe
	observation := edge.Observe(ctx, in)
	return edge.ToProto(observation), nil
}

// readCaddyJournal reads the bounded proxy journal through the prober. The
// evidence keeps messages only.
func (s *Server) readCaddyJournal(ctx context.Context, prober vps.Prober) edge.RenewalEvidence {
	argv := edge.JournalArgv(edge.DefaultJournalTail)
	res, err := prober.Observe(ctx, argv[0], argv[1:]...)
	if err != nil || res.ExitCode != 0 {
		return edge.RenewalEvidence{}
	}
	return edge.ParseCaddyJournal(res.Stdout)
}

func certificateFacts(probe tlsinfo.ProbeResult) edge.CertificateFacts {
	facts := edge.CertificateFacts{Present: true, Valid: probe.Valid, ValidationError: probe.ValidationError, Issuer: probe.Issuer, SANs: probe.SANs}
	if probe.NotAfter != "" {
		if parsed, err := time.Parse("Jan 2 15:04:05 2006 MST", probe.NotAfter); err == nil {
			facts.NotAfter = parsed
		} else if parsed, err := time.Parse(time.RFC3339, probe.NotAfter); err == nil {
			facts.NotAfter = parsed
		}
	}
	return facts
}

// externalReadinessProbe is the producer-side HTTPS probe of a public host.
// Tests replace it; it never disables certificate verification.
var externalReadinessProbe = func(ctx context.Context, host string, port int) error {
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+net.JoinHostPort(host, strconv.Itoa(port))+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("public health returned %s", resp.Status)
	}
	return nil
}

// ObserveEdge implements edgesvc.Producer.
func (s *Server) ObserveEdge(ctx context.Context, deploymentID string) (*edgev1.EdgeObservation, error) {
	dc, err := s.loadDeploymentContext(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	return s.inspectEdge(ctx, dc)
}

// handleGetEdgeObservation serves the secret-free public exposure record.
// GET /api/v1/deployments/{id}/edge
func (s *Server) handleGetEdgeObservation(w http.ResponseWriter, r *http.Request) {
	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return
	}
	observation, err := s.inspectEdge(r.Context(), dc)
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	raw, err := edge.MarshalResponseJSON(observation)
	if err != nil {
		apierrors.Write(w, apierrors.Internal("Failed to encode edge observation", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}
