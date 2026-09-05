package config

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"tunnel-manager/internal/manifest"
)

// AdoptIngress brings an unmanaged live hostname under management. See the
// Service interface for semantics. Adopt-as-external creates an external route
// (Phase 3) pointing at the supplied target (or the live service); adopt-as-
// scenario creates a scenario route when the live service is a localhost port.
func (s *service) AdoptIngress(ctx context.Context, hostname, scenario, target string) (IngressEntry, error) {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return IngressEntry{}, ErrInvalidConfig{Field: "hostname", Reason: "required"}
	}
	if s.deps.RoutesWriter == nil {
		return IngressEntry{}, ErrInvalidConfig{Field: "adopt", Reason: "route management is not configured"}
	}

	cfg, err := s.deps.Repo.Get(ctx)
	if err != nil {
		return IngressEntry{}, fmt.Errorf("read config: %w", err)
	}
	// Resolve live ingress against the real tunnel: with complete CF credentials
	// the drift entries the operator is acting on live on Cloudflare even when
	// the persisted mode is still the local default (see effectiveMode). Adopt
	// and Ignore only write the ledger/routes; Prune's removal targets this same
	// source, so read and write stay consistent.
	cfg.Mode = s.effectiveMode(ctx, cfg)
	live, err := s.liveIngress(ctx, cfg.Mode)
	if err != nil {
		return IngressEntry{}, err
	}
	liveSvc := liveServiceFor(live, hostname)

	subdomain, domain := splitHostname(hostname)
	if subdomain == "" || domain == "" {
		return IngressEntry{}, ErrInvalidConfig{Field: "hostname", Reason: "must be <subdomain>.<domain>"}
	}

	scenario = strings.TrimSpace(scenario)
	target = strings.TrimSpace(target)

	// Decide provenance:
	//   - explicit --scenario (no --target): scenario route, port from the live
	//     localhost target;
	//   - bare adopt (no --scenario, no --target): the common UI case. Auto-
	//     detect — if the subdomain matches a known scenario (resolvable fixed
	//     UI port), adopt it as a SCENARIO route with that port so it shows
	//     under its scenario with the right port; otherwise fall back to an
	//     external route pointing at whatever is live today;
	//   - explicit --target: external route.
	in := manifest.CreateInput{Subdomain: subdomain, Domain: domain, Tier: manifest.TierLeased}
	var owner Owner
	var entrySource RouteSource
	switch {
	case target == "" && scenario != "":
		port, ok := localhostPort(liveSvc)
		if !ok {
			return IngressEntry{}, ErrInvalidConfig{Field: "scenario", Reason: fmt.Sprintf("cannot adopt %q as a scenario route: live target %q is not http://localhost:<port>; pass --target to adopt as external", hostname, liveSvc)}
		}
		in.Scenario = scenario
		in.LocalPort = port
		in.Source = manifest.SourceScenario
		owner = OwnerManaged
		entrySource = SourceScenario
	case target == "" && scenario == "":
		if port, ok := s.scenarioRoutePort(ctx, subdomain, liveSvc); ok {
			// The subdomain is a known scenario — adopt as a scenario route.
			scenario = subdomain
			in.Scenario = subdomain
			in.LocalPort = port
			in.Source = manifest.SourceScenario
			owner = OwnerManaged
			entrySource = SourceScenario
			break
		}
		if liveSvc == "" {
			return IngressEntry{}, ErrInvalidConfig{Field: "target", Reason: "no live service to adopt and no --target supplied"}
		}
		in.Source = manifest.SourceExternal
		in.ServiceTarget = liveSvc
		owner = OwnerExternal
		entrySource = SourceExternal
	default:
		svc := target
		if svc == "" {
			svc = liveSvc
		}
		if svc == "" {
			return IngressEntry{}, ErrInvalidConfig{Field: "target", Reason: "no live service to adopt and no --target supplied"}
		}
		in.Source = manifest.SourceExternal
		in.ServiceTarget = svc
		in.Scenario = scenario // optional label for external routes
		owner = OwnerExternal
		entrySource = SourceExternal
	}

	if err := s.createOrUpdateRoute(ctx, subdomain, domain, in); err != nil {
		return IngressEntry{}, err
	}
	if s.deps.Ledger != nil {
		if err := s.deps.Ledger.Put(ctx, LedgerEntry{Hostname: hostname, Owner: owner, Scenario: scenario}); err != nil {
			return IngressEntry{}, err
		}
	}

	state := StateManaged
	svcTarget := liveSvc
	if entrySource == SourceExternal {
		state = StateExternalOK
		if in.ServiceTarget != "" {
			svcTarget = in.ServiceTarget
		}
	} else {
		svcTarget = fmt.Sprintf("http://localhost:%d", in.LocalPort)
	}
	return IngressEntry{
		Hostname:      hostname,
		ServiceTarget: svcTarget,
		State:         state,
		Source:        entrySource,
		Scenario:      scenario,
	}, nil
}

// IgnoreIngress records a live hostname as IGNORED. No apply is needed: the
// hostname is already live and reconcile will preserve it untouched.
func (s *service) IgnoreIngress(ctx context.Context, hostname, note string) (IngressEntry, error) {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return IngressEntry{}, ErrInvalidConfig{Field: "hostname", Reason: "required"}
	}
	cfg, err := s.deps.Repo.Get(ctx)
	if err != nil {
		return IngressEntry{}, fmt.Errorf("read config: %w", err)
	}
	// Resolve live ingress against the real tunnel: with complete CF credentials
	// the drift entries the operator is acting on live on Cloudflare even when
	// the persisted mode is still the local default (see effectiveMode). Adopt
	// and Ignore only write the ledger/routes; Prune's removal targets this same
	// source, so read and write stay consistent.
	cfg.Mode = s.effectiveMode(ctx, cfg)
	live, err := s.liveIngress(ctx, cfg.Mode)
	if err != nil {
		return IngressEntry{}, err
	}
	if s.deps.Ledger != nil {
		if err := s.deps.Ledger.Put(ctx, LedgerEntry{Hostname: hostname, Owner: OwnerIgnored, Note: note}); err != nil {
			return IngressEntry{}, err
		}
	}
	return IngressEntry{
		Hostname:      hostname,
		ServiceTarget: liveServiceFor(live, hostname),
		State:         StateIgnored,
		Source:        SourceExternal,
		Note:          note,
	}, nil
}

// PruneIngress removes a single named hostname from live ingress and the
// ledger. It is the only path that removes a specific entry.
func (s *service) PruneIngress(ctx context.Context, hostname string) (bool, error) {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return false, ErrInvalidConfig{Field: "hostname", Reason: "required"}
	}
	cfg, err := s.deps.Repo.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("read config: %w", err)
	}
	// Resolve live ingress against the real tunnel: with complete CF credentials
	// the drift entries the operator is acting on live on Cloudflare even when
	// the persisted mode is still the local default (see effectiveMode). Adopt
	// and Ignore only write the ledger/routes; Prune's removal targets this same
	// source, so read and write stay consistent.
	cfg.Mode = s.effectiveMode(ctx, cfg)
	live, err := s.liveIngress(ctx, cfg.Mode)
	if err != nil {
		return false, err
	}
	ledger, err := s.ledgerEntries(ctx)
	if err != nil {
		return false, err
	}

	inLive := false
	for _, r := range live {
		if r.Hostname == hostname {
			inLive = true
			break
		}
	}
	inLedger := false
	for _, l := range ledger {
		if l.Hostname == hostname {
			inLedger = true
			break
		}
	}
	if !inLive && !inLedger {
		return false, nil
	}

	if !inLive {
		// Nothing live to remove — just clear the stale ledger record.
		if s.deps.Ledger != nil {
			if _, err := s.deps.Ledger.Delete(ctx, hostname); err != nil {
				return false, err
			}
		}
		return true, nil
	}

	desired, err := s.desiredEntries(ctx)
	if err != nil {
		return false, err
	}
	if err := s.applyAdditive(ctx, cfg, desired, live, ledger, []string{hostname}); err != nil {
		return false, err
	}
	return true, nil
}

// scenarioRoutePort decides the local port to use when adopting a hostname as a
// scenario route, and whether it should be one at all. A hostname is scenario-
// backed when its subdomain names a known scenario; the port is the scenario's
// fixed UI port when declared, else the live localhost port (covers scenarios
// whose UI port is ranged/dynamically allocated). Returns false — so adopt
// falls back to an external route — when no resolver is wired (Deps.Scenarios
// nil), the slug is not a known scenario, or no port can be determined.
func (s *service) scenarioRoutePort(ctx context.Context, scenario, liveSvc string) (int, bool) {
	if s.deps.Scenarios == nil {
		return 0, false
	}
	if port, err := s.deps.Scenarios.UIPort(ctx, scenario); err == nil && port > 0 {
		return port, true
	}
	if s.deps.Scenarios.IsScenario(ctx, scenario) {
		if port, ok := localhostPort(liveSvc); ok {
			return port, true
		}
	}
	return 0, false
}

// createOrUpdateRoute persists the adopted route idempotently: if a route
// already exists for the subdomain it is updated in place (re-adopt / repair a
// previously mis-classified route), otherwise a new one is created. Treating a
// non-nil GetBySubdomain error as "absent" keeps adopt resilient — a stale read
// at worst surfaces the create's conflict error.
func (s *service) createOrUpdateRoute(ctx context.Context, subdomain, domain string, in manifest.CreateInput) error {
	if existing, err := s.deps.RoutesWriter.GetBySubdomain(ctx, subdomain); err == nil {
		_, uerr := s.deps.RoutesWriter.Update(ctx, existing.ID, manifest.UpdateInput{
			Subdomain:     subdomain,
			Domain:        domain,
			Scenario:      in.Scenario,
			LocalPort:     in.LocalPort,
			Tier:          in.Tier,
			Source:        in.Source,
			ServiceTarget: in.ServiceTarget,
		})
		return uerr
	}
	_, err := s.deps.RoutesWriter.Create(ctx, in)
	return err
}

// liveServiceFor returns the live service target for a hostname, or "".
func liveServiceFor(live []IngressRule, hostname string) string {
	for _, r := range live {
		if r.Hostname == hostname {
			return r.Service
		}
	}
	return ""
}

// splitHostname splits a full hostname into its leftmost label (subdomain) and
// the remaining apex domain. "api.itsagitime.com" → ("api", "itsagitime.com").
func splitHostname(hostname string) (subdomain, domain string) {
	i := strings.IndexByte(hostname, '.')
	if i <= 0 || i == len(hostname)-1 {
		return "", ""
	}
	return hostname[:i], hostname[i+1:]
}

// localhostPort parses the port from an http://localhost:<port> service URL.
func localhostPort(service string) (int, bool) {
	for _, prefix := range []string{"http://localhost:", "https://localhost:", "http://127.0.0.1:", "https://127.0.0.1:"} {
		if strings.HasPrefix(service, prefix) {
			rest := strings.TrimPrefix(service, prefix)
			if i := strings.IndexByte(rest, '/'); i >= 0 {
				rest = rest[:i]
			}
			port, err := strconv.Atoi(rest)
			if err != nil || port < 1 || port > 65535 {
				return 0, false
			}
			return port, true
		}
	}
	return 0, false
}
