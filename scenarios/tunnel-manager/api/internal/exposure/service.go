package exposure

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"tunnel-manager/internal/clock"
	"tunnel-manager/internal/manifest"
)

// Manifest is the subset of the routes service this domain needs. Declared
// here (consumer-side) and satisfied by *routes.service; it lets exposure
// reconcile the manifest without owning it.
type Manifest interface {
	List(ctx context.Context, tier manifest.Tier) ([]manifest.Route, error)
	Create(ctx context.Context, in manifest.CreateInput) (manifest.Route, error)
	Update(ctx context.Context, id string, in manifest.UpdateInput) (manifest.Route, error)
	Delete(ctx context.Context, id string) (bool, error)
}

// Ingress reconciles live Cloudflare ingress with the manifest. Satisfied
// by an adapter over the config service's Sync (wired in main.go) so this
// domain never imports the config package.
type Ingress interface {
	Reconcile(ctx context.Context) error
}

// Runner ensures a scenario's process is running before it is exposed.
// Satisfied by a cmdrunner-backed adapter (vrooli scenario start) in
// production; faked in tests.
type Runner interface {
	EnsureRunning(ctx context.Context, scenario string) error
}

// PortResolver returns a scenario's fixed UI port (from its service.json).
// CORE route creation and Expose need the local port the tunnel forwards
// to. Returns ErrPortUnresolved when no fixed port is declared.
type PortResolver interface {
	UIPort(ctx context.Context, scenario string) (int, error)
}

// CoreSetProvider returns the set of scenario names that must always be
// exposed (the api-core/coreset closure). Injected so tests pin the set.
type CoreSetProvider func() []string

// Service is the exposure broker surface the handlers depend on.
type Service interface {
	Expose(ctx context.Context, in ExposeInput) (Lease, string, error)
	ExtendLease(ctx context.Context, leaseID string, ttl time.Duration) (Lease, error)
	RevokeLease(ctx context.Context, leaseID string) (retracted bool, err error)
	Unexpose(ctx context.Context, scenario string) (retracted bool, leaseID string, err error)
	ListLeases(ctx context.Context, status LeaseStatus) ([]Lease, error)
	ListExposures(ctx context.Context) ([]Exposure, error)
	IsExposed(ctx context.Context, scenario string) (bool, string, error)
	Reconcile(ctx context.Context) (coreEnsured int, leasesReaped int, err error)
}

// PortAssigner makes a ranged scenario exposable by ensuring it has a fixed UI
// port (assigning a free in-band one via structure-health) and releasing
// TM-assigned ports on revoke. Optional: when nil, Expose only works for
// scenarios that already declare a fixed UI port (the pre-Phase-4 behaviour).
type PortAssigner interface {
	// EnsureFixed ensures the scenario has a fixed UI port. assignedByTM is true
	// only when TM switched a previously-ranged scenario this call (so revoke
	// knows it is safe to release; a hand-pinned port returns false).
	EnsureFixed(ctx context.Context, scenario string) (assignedByTM bool, err error)
	// Release reverts a fixed port back to a range. The service calls it only for
	// scenarios its ownership store attributes to TM.
	Release(ctx context.Context, scenario string) error
}

// PortOwnership persists which scenarios TM assigned a fixed port to, so revoke
// releases only those (never a hand-pinned fixed port). Optional: nil disables
// release (ports are assigned but never auto-reverted — the safe direction).
type PortOwnership interface {
	Record(ctx context.Context, scenario string) error
	Owned(ctx context.Context, scenario string) (bool, error)
	Clear(ctx context.Context, scenario string) error
}

type service struct {
	repo      Repository
	manifest  Manifest
	ingress   Ingress
	runner    Runner
	ports     PortResolver
	coreSet   CoreSetProvider
	clock     clock.Clock
	assigner  PortAssigner
	portOwner PortOwnership
}

// Option configures optional service collaborators without breaking the
// positional constructor used across the codebase and tests.
type Option func(*service)

// WithPortAssigner wires the structure-health-backed fixed-port assigner +
// ownership store so Expose can auto-fix ranged scenarios and revoke can
// release TM-assigned ports.
func WithPortAssigner(assigner PortAssigner, owner PortOwnership) Option {
	return func(s *service) {
		s.assigner = assigner
		s.portOwner = owner
	}
}

// NewService constructs the production exposure broker.
func NewService(repo Repository, manifest Manifest, ingress Ingress, runner Runner, ports PortResolver, coreSet CoreSetProvider, clk clock.Clock, opts ...Option) Service {
	s := &service{
		repo:     repo,
		manifest: manifest,
		ingress:  ingress,
		runner:   runner,
		ports:    ports,
		coreSet:  coreSet,
		clock:    clk,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

var _ Service = (*service)(nil)

func (s *service) Expose(ctx context.Context, in ExposeInput) (Lease, string, error) {
	scenario := strings.TrimSpace(in.Scenario)
	if scenario == "" {
		return Lease{}, "", ErrInvalidExposure{Field: "scenario", Reason: "required"}
	}
	ttl := in.TTL
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	// Make a ranged scenario exposable: ensure it has a fixed UI port (the
	// tunnel forwards to a concrete localhost:<port>). Best-effort — if the
	// assigner is down but the scenario already has a fixed port, ensureRoute
	// still resolves it; only a genuinely-ranged-and-unassignable scenario fails
	// (with the existing ErrPortUnresolved from the resolver).
	s.ensureFixedPort(ctx, scenario)

	// Ensure a route exists for the scenario (idempotent — reuse any
	// existing route, only create a LEASED one when absent). Creating the
	// route before the lease means a mid-Expose failure leaves at most an
	// orphan route, which Reconcile reaps (failure-topography).
	route, err := s.ensureRoute(ctx, scenario)
	if err != nil {
		return Lease{}, "", err
	}

	// Ensure the scenario process is up, then publish ingress.
	if err := s.runner.EnsureRunning(ctx, scenario); err != nil {
		return Lease{}, "", err
	}
	if err := s.ingress.Reconcile(ctx); err != nil {
		return Lease{}, "", err
	}

	now := s.clock.Now().UTC()

	// Idempotency: an already-active lease is extended, not duplicated.
	existing, err := s.repo.ActiveForScenario(ctx, scenario)
	switch {
	case err == nil:
		existing.ExpiresAt = now.Add(ttl)
		existing.ExtendedCount++
		if in.RequestedBy != "" {
			existing.RequestedBy = in.RequestedBy
		}
		updated, uerr := s.repo.Update(ctx, existing)
		if uerr != nil {
			return Lease{}, "", uerr
		}
		return updated, route.PublicURL(), nil
	case errors.As(err, &ErrLeaseNotFound{}):
		// fall through to create
	default:
		return Lease{}, "", err
	}

	lease, err := s.repo.Create(ctx, Lease{
		Scenario:    scenario,
		RequestedBy: in.RequestedBy,
		ExpiresAt:   now.Add(ttl),
		Status:      LeaseActive,
	})
	if err != nil {
		return Lease{}, "", err
	}
	return lease, route.PublicURL(), nil
}

func (s *service) ExtendLease(ctx context.Context, leaseID string, ttl time.Duration) (Lease, error) {
	if strings.TrimSpace(leaseID) == "" {
		return Lease{}, ErrInvalidExposure{Field: "lease_id", Reason: "required"}
	}
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	lease, err := s.repo.Get(ctx, leaseID)
	if err != nil {
		return Lease{}, err
	}
	lease.ExpiresAt = s.clock.Now().UTC().Add(ttl)
	lease.ExtendedCount++
	lease.Status = LeaseActive
	return s.repo.Update(ctx, lease)
}

func (s *service) RevokeLease(ctx context.Context, leaseID string) (bool, error) {
	if strings.TrimSpace(leaseID) == "" {
		return false, ErrInvalidExposure{Field: "lease_id", Reason: "required"}
	}
	lease, err := s.repo.Get(ctx, leaseID)
	if err != nil {
		return false, err
	}
	lease.Status = LeaseRevoked
	if _, err := s.repo.Update(ctx, lease); err != nil {
		return false, err
	}
	// Retract ingress unless the scenario is also CORE (stays exposed).
	if s.isCore(lease.Scenario) {
		return false, nil
	}
	if err := s.retractRoute(ctx, lease.Scenario); err != nil {
		return false, err
	}
	if err := s.ingress.Reconcile(ctx); err != nil {
		return false, err
	}
	// Release a TM-assigned fixed port back to its range now the scenario is no
	// longer exposed (no-op for hand-pinned ports or when no assigner is wired).
	s.releaseAssignedPort(ctx, lease.Scenario)
	return true, nil
}

// Unexpose revokes a scenario's active lease by name — the thin primitive
// behind the `unexpose` CLI alias. It reuses RevokeLease so ingress + DNS
// retraction and the CORE guard stay in one place. ErrLeaseNotFound surfaces
// when the scenario has no active lease.
func (s *service) Unexpose(ctx context.Context, scenario string) (bool, string, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return false, "", ErrInvalidExposure{Field: "scenario", Reason: "required"}
	}
	lease, err := s.repo.ActiveForScenario(ctx, scenario)
	if err != nil {
		return false, "", err
	}
	retracted, err := s.RevokeLease(ctx, lease.ID)
	if err != nil {
		return false, "", err
	}
	return retracted, lease.ID, nil
}

func (s *service) ListLeases(ctx context.Context, status LeaseStatus) ([]Lease, error) {
	return s.repo.List(ctx, status)
}

func (s *service) ListExposures(ctx context.Context) ([]Exposure, error) {
	routes, err := s.manifest.List(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]Exposure, 0, len(routes))
	for _, r := range routes {
		e := Exposure{
			Scenario:  r.Scenario,
			Subdomain: r.Subdomain,
			PublicURL: r.PublicURL(),
			LocalPort: r.LocalPort,
			Tier:      string(r.Tier),
			Enabled:   r.Enabled,
		}
		if r.Tier == manifest.TierLeased {
			if lease, lerr := s.repo.ActiveForScenario(ctx, r.Scenario); lerr == nil {
				l := lease
				e.Lease = &l
			}
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *service) IsExposed(ctx context.Context, scenario string) (bool, string, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return false, "", ErrInvalidExposure{Field: "scenario", Reason: "required"}
	}
	routes, err := s.manifest.List(ctx, "")
	if err != nil {
		return false, "", err
	}
	for _, r := range routes {
		if r.Scenario != scenario || !r.Enabled {
			continue
		}
		// CORE routes are always reachable; LEASED routes need an active lease.
		if r.Tier == manifest.TierCore {
			return true, r.PublicURL(), nil
		}
		if _, lerr := s.repo.ActiveForScenario(ctx, scenario); lerr == nil {
			return true, r.PublicURL(), nil
		}
	}
	return false, "", nil
}

func (s *service) Reconcile(ctx context.Context) (int, int, error) {
	routes, err := s.manifest.List(ctx, "")
	if err != nil {
		return 0, 0, err
	}
	bySubdomain := make(map[string]manifest.Route, len(routes))
	for _, r := range routes {
		bySubdomain[r.Subdomain] = r
	}

	// 1. Ensure a CORE route for every coreset scenario.
	coreEnsured := 0
	for _, scenario := range sortedUnique(s.coreSet()) {
		existing, ok := bySubdomain[scenario]
		if ok {
			// Promote to CORE + enable if it drifted.
			if existing.Tier != manifest.TierCore || !existing.Enabled {
				core := manifest.TierCore
				enabled := true
				if _, uerr := s.manifest.Update(ctx, existing.ID, manifest.UpdateInput{Tier: core, Enabled: &enabled}); uerr != nil {
					return coreEnsured, 0, uerr
				}
				coreEnsured++
			}
			continue
		}
		port, perr := s.ports.UIPort(ctx, scenario)
		if perr != nil {
			// A core scenario with no resolvable port is skipped rather
			// than failing the whole reconcile — surface it but keep going.
			continue
		}
		if _, cerr := s.manifest.Create(ctx, manifest.CreateInput{
			Subdomain: scenario,
			Scenario:  scenario,
			LocalPort: port,
			Tier:      manifest.TierCore,
		}); cerr != nil {
			return coreEnsured, 0, cerr
		}
		coreEnsured++
	}

	// 2. Reap expired leases and retract their ingress unless also CORE.
	now := s.clock.Now().UTC()
	active, err := s.repo.List(ctx, LeaseActive)
	if err != nil {
		return coreEnsured, 0, err
	}
	reaped := 0
	changed := false
	for _, lease := range active {
		if lease.ExpiresAt.After(now) {
			continue
		}
		lease.Status = LeaseExpired
		if _, uerr := s.repo.Update(ctx, lease); uerr != nil {
			return coreEnsured, reaped, uerr
		}
		reaped++
		if !s.isCore(lease.Scenario) {
			if err := s.retractRoute(ctx, lease.Scenario); err != nil {
				return coreEnsured, reaped, err
			}
			s.releaseAssignedPort(ctx, lease.Scenario)
			changed = true
		}
	}

	if coreEnsured > 0 || changed {
		if err := s.ingress.Reconcile(ctx); err != nil {
			return coreEnsured, reaped, err
		}
	}
	return coreEnsured, reaped, nil
}

// ensureFixedPort asks the assigner to give a ranged scenario a free in-band
// fixed port and records TM ownership when it does. It is best-effort: a failure
// (assigner down, structure-health unreachable) is swallowed so an
// already-fixed scenario still exposes; a genuinely-ranged scenario then fails
// downstream at the port resolver with a clear ErrPortUnresolved.
func (s *service) ensureFixedPort(ctx context.Context, scenario string) {
	if s.assigner == nil {
		return
	}
	assignedByTM, err := s.assigner.EnsureFixed(ctx, scenario)
	if err != nil || !assignedByTM {
		return
	}
	if s.portOwner != nil {
		_ = s.portOwner.Record(ctx, scenario)
	}
}

// releaseAssignedPort reverts a fixed port back to a range, but only for
// scenarios TM assigned (the ownership store gates this so a hand-pinned fixed
// port is never reverted). Best-effort: a release failure leaves the ownership
// row so a later revoke/reap retries.
func (s *service) releaseAssignedPort(ctx context.Context, scenario string) {
	if s.assigner == nil || s.portOwner == nil {
		return
	}
	owned, err := s.portOwner.Owned(ctx, scenario)
	if err != nil || !owned {
		return
	}
	if err := s.assigner.Release(ctx, scenario); err != nil {
		return
	}
	_ = s.portOwner.Clear(ctx, scenario)
}

// ensureRoute returns the existing route for a scenario or creates a
// LEASED one. The local port comes from the PortResolver.
func (s *service) ensureRoute(ctx context.Context, scenario string) (manifest.Route, error) {
	routes, err := s.manifest.List(ctx, "")
	if err != nil {
		return manifest.Route{}, err
	}
	for _, r := range routes {
		if r.Scenario == scenario {
			if !r.Enabled {
				enabled := true
				return s.manifest.Update(ctx, r.ID, manifest.UpdateInput{Enabled: &enabled})
			}
			return r, nil
		}
	}
	port, err := s.ports.UIPort(ctx, scenario)
	if err != nil {
		return manifest.Route{}, err
	}
	return s.manifest.Create(ctx, manifest.CreateInput{
		Subdomain: scenario,
		Scenario:  scenario,
		LocalPort: port,
		Tier:      manifest.TierLeased,
	})
}

// retractRoute disables the LEASED route(s) for a scenario. CORE routes are
// never retracted here (callers gate on isCore first).
func (s *service) retractRoute(ctx context.Context, scenario string) error {
	routes, err := s.manifest.List(ctx, "")
	if err != nil {
		return err
	}
	for _, r := range routes {
		if r.Scenario != scenario || r.Tier == manifest.TierCore {
			continue
		}
		if _, derr := s.manifest.Delete(ctx, r.ID); derr != nil {
			return derr
		}
	}
	return nil
}

func (s *service) isCore(scenario string) bool {
	for _, c := range s.coreSet() {
		if c == scenario {
			return true
		}
	}
	return false
}

func sortedUnique(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
