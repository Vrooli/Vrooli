package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tunnel-manager/internal/clock"
	"tunnel-manager/internal/cmdrunner"
	"tunnel-manager/internal/manifest"
)

// Service is the application-layer surface the config handlers depend on.
// Owns the reconcile policy: compute desired ingress from the routes
// manifest, diff it against live ingress (Cloudflare API in remote mode,
// the local config.yml in local mode), and apply the difference.
type Service interface {
	// GetConfig returns the persisted tunnel configuration (defaults when
	// none has been written yet).
	GetConfig(ctx context.Context) (TunnelConfig, error)

	// GetConfigState returns the persisted tunnel configuration plus
	// process-level readiness derived from non-secret credential presence.
	GetConfigState(ctx context.Context) (ConfigState, error)

	// GetCredentialStatus returns browser-safe Cloudflare credential
	// presence/source metadata. It never returns the API token value.
	GetCredentialStatus(ctx context.Context) (CredentialStatus, error)

	// SetCloudflareCredentials stores write-only Cloudflare credentials in the
	// configured credential store and returns redacted status metadata.
	SetCloudflareCredentials(ctx context.Context, values CredentialUpdate) (CredentialStatus, error)

	// ClearCloudflareCredentials removes one or more file-backed Cloudflare
	// credential values and returns redacted status metadata.
	ClearCloudflareCredentials(ctx context.Context, keys []string) (CredentialStatus, error)

	// Sync reconciles live ingress with the desired manifest ADDITIVELY: it
	// publishes desired hostnames merged onto current live, never dropping
	// unmanaged or ignored entries. Drift (unmanaged) and orphaned entries are
	// surfaced but removed only when prune is true (a batch prune removes
	// orphaned entries; per-hostname removal is PruneIngress). dryRun computes
	// the report without applying. NoChanges is set when nothing would change.
	Sync(ctx context.Context, dryRun, prune bool) (SyncResult, error)

	// SwitchMode migrates between remote and local management and persists
	// the new mode. It is PURE: it never writes ingress. Switching to remote
	// requires the remote read path to be available (a read-only credential
	// resolve) so the operator is never stranded in a mode they cannot
	// reconcile; switching never pushes. Apply is a separate explicit Sync.
	// Returns the previous and current modes.
	SwitchMode(ctx context.Context, target Mode) (prev, cur Mode, err error)

	// GetDrift reconciles the desired manifest, the live ingress, and the
	// ownership ledger into a classified read model (no writes). It is the
	// single source of truth for "what is live vs. what TM wants vs. what is
	// unmanaged" in either mode.
	GetDrift(ctx context.Context) (DriftReport, error)

	// AdoptIngress brings an unmanaged live hostname under management. It
	// creates a scenario route when scenario is supplied (or the live target
	// resolves to a known scenario), otherwise an external route, and records
	// MANAGED/EXTERNAL ownership. Returns the reclassified entry.
	AdoptIngress(ctx context.Context, hostname, scenario, target string) (IngressEntry, error)

	// IgnoreIngress records a live hostname as IGNORED so reconcile never
	// pushes or prunes it. Returns the reclassified entry.
	IgnoreIngress(ctx context.Context, hostname, note string) (IngressEntry, error)

	// PruneIngress removes a single named hostname from live ingress and the
	// ownership ledger — the only path that removes a specific entry. Returns
	// false when the hostname was neither live nor tracked.
	PruneIngress(ctx context.Context, hostname string) (bool, error)
}

// Deps wires the seams the config service depends on. IngressClient is nil
// when Cloudflare credentials are absent — remote operations then return
// ErrRemoteUnavailable instead of touching the network.
type Deps struct {
	Repo             ConfigRepository
	Routes           RoutesReader
	RoutesWriter     RoutesManager
	Scenarios        ScenarioResolver
	Ingress          IngressClient
	Ledger           OwnershipLedger
	CredentialStore  CredentialStore
	CF               CFConfig
	CredentialStatus CredentialStatus
	Runner           cmdrunner.Runner
	Clock            clock.Clock
	// LocalConfigPath is where local mode writes the cloudflared config.yml.
	// Defaults to ~/.cloudflared/config.yml when empty.
	LocalConfigPath string
}

type service struct {
	deps Deps
}

// NewService constructs the production Service.
func NewService(d Deps) Service {
	if d.Runner == nil {
		d.Runner = cmdrunner.Default
	}
	if d.Clock == nil {
		d.Clock = clock.System{}
	}
	if d.LocalConfigPath == "" {
		d.LocalConfigPath = filepath.Join(os.Getenv("HOME"), ".cloudflared", "config.yml")
	}
	return &service{deps: d}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) GetConfig(ctx context.Context) (TunnelConfig, error) {
	return s.deps.Repo.Get(ctx)
}

func (s *service) GetConfigState(ctx context.Context) (ConfigState, error) {
	cfg, err := s.deps.Repo.Get(ctx)
	if err != nil {
		return ConfigState{}, err
	}
	readiness, err := s.readiness(ctx, cfg)
	if err != nil {
		return ConfigState{}, err
	}
	return ConfigState{
		Config:    cfg,
		Readiness: readiness,
	}, nil
}

func (s *service) GetCredentialStatus(ctx context.Context) (CredentialStatus, error) {
	return s.credentialStatus(ctx)
}

func (s *service) SetCloudflareCredentials(ctx context.Context, values CredentialUpdate) (CredentialStatus, error) {
	if s.deps.CredentialStore == nil {
		return CredentialStatus{}, ErrInvalidConfig{Field: "credentials", Reason: "credential store is not configured"}
	}
	return s.deps.CredentialStore.Save(ctx, values)
}

func (s *service) ClearCloudflareCredentials(ctx context.Context, keys []string) (CredentialStatus, error) {
	if s.deps.CredentialStore == nil {
		return CredentialStatus{}, ErrInvalidConfig{Field: "credentials", Reason: "credential store is not configured"}
	}
	return s.deps.CredentialStore.Delete(ctx, keys)
}

func (s *service) Sync(ctx context.Context, dryRun, prune bool) (SyncResult, error) {
	cfg, err := s.deps.Repo.Get(ctx)
	if err != nil {
		return SyncResult{}, fmt.Errorf("read config: %w", err)
	}
	// Operate against the real tunnel: when CF credentials are complete the live
	// ingress lives on Cloudflare regardless of the persisted (default-local)
	// mode. Apply stays additive, so this never drops pre-existing ingress.
	cfg.Mode = s.effectiveMode(ctx, cfg)

	desired, err := s.desiredEntries(ctx)
	if err != nil {
		return SyncResult{}, err
	}

	if dryRun {
		if report, handled, err := s.remoteSetupReport(ctx, cfg); handled || err != nil {
			return report, err
		}
	}

	live, err := s.liveIngress(ctx, cfg.Mode)
	if err != nil {
		return SyncResult{}, err
	}
	ledger, err := s.ledgerEntries(ctx)
	if err != nil {
		return SyncResult{}, err
	}

	rep := reconcile(cfg.Mode, desired, live, ledger)
	wouldAdd, unmanaged, orphaned := partitionDrift(rep)

	// A batch prune removes orphaned entries only — entries TM created whose
	// routes are gone. Genuinely unmanaged drift is NEVER batch-removed; it
	// requires an explicit per-entry decision (adopt / ignore / prune).
	var pruneSet []string
	if prune {
		pruneSet = orphaned
	}

	result := SyncResult{
		Mode:           cfg.Mode,
		Added:          wouldAdd,
		DriftUnmanaged: unmanaged,
		Orphaned:       orphaned,
		Pruned:         pruneSet,
		Removed:        pruneSet,
		NoChanges:      len(wouldAdd) == 0 && len(pruneSet) == 0,
	}
	result.Message = syncMessage(cfg.Mode, dryRun, result)

	if dryRun || result.NoChanges {
		return result, nil
	}

	if err := s.applyAdditive(ctx, cfg, desired, live, ledger, pruneSet); err != nil {
		return SyncResult{}, err
	}
	return result, nil
}

// remoteSetupReport returns a setup-required SyncResult when a remote-mode
// dry-run cannot reach Cloudflare (credentials absent). handled is true when
// the caller should return the report instead of proceeding.
func (s *service) remoteSetupReport(ctx context.Context, cfg TunnelConfig) (SyncResult, bool, error) {
	if cfg.Mode != ModeRemote {
		return SyncResult{}, false, nil
	}
	credsMissing := s.deps.Ingress == nil && s.deps.CredentialStore == nil
	if !credsMissing {
		if _, err := s.remoteIngress(ctx); err != nil {
			if _, ok := err.(ErrRemoteUnavailable); !ok {
				return SyncResult{}, false, err
			}
			credsMissing = true
		}
	}
	if !credsMissing {
		return SyncResult{}, false, nil
	}
	readiness, err := s.readiness(ctx, cfg)
	if err != nil {
		return SyncResult{}, false, err
	}
	return SyncResult{
		Mode:          cfg.Mode,
		SetupRequired: true,
		MissingFields: readiness.MissingFields,
		Message:       readiness.ModeReason,
	}, true, nil
}

// partitionDrift splits a reconcile report into the add/unmanaged/orphaned
// hostname lists a sync reports, each sorted for deterministic output.
func partitionDrift(rep DriftReport) (wouldAdd, unmanaged, orphaned []string) {
	for _, e := range rep.Entries {
		switch e.State {
		case StateMissing:
			wouldAdd = append(wouldAdd, e.Hostname)
		case StateUnmanaged:
			unmanaged = append(unmanaged, e.Hostname)
		case StateOrphaned:
			orphaned = append(orphaned, e.Hostname)
		}
	}
	sort.Strings(wouldAdd)
	sort.Strings(unmanaged)
	sort.Strings(orphaned)
	return wouldAdd, unmanaged, orphaned
}

// syncMessage builds the operator-facing explanation for a sync result.
func syncMessage(mode Mode, dryRun bool, r SyncResult) string {
	switch {
	case r.NoChanges:
		return fmt.Sprintf("Ingress already matches the desired manifest in %s mode (%d unmanaged, %d orphaned).", mode, len(r.DriftUnmanaged), len(r.Orphaned))
	case dryRun:
		return fmt.Sprintf("Dry-run in %s mode: %d to add, %d to prune, %d unmanaged drift, %d orphaned.", mode, len(r.Added), len(r.Pruned), len(r.DriftUnmanaged), len(r.Orphaned))
	default:
		return fmt.Sprintf("Ingress reconciled additively in %s mode: %d added, %d pruned (%d unmanaged left intact).", mode, len(r.Added), len(r.Pruned), len(r.DriftUnmanaged))
	}
}

func (s *service) SwitchMode(ctx context.Context, target Mode) (Mode, Mode, error) {
	if target != ModeRemote && target != ModeLocal {
		return ModeUnspecified, ModeUnspecified, ErrInvalidConfig{Field: "target_mode", Reason: fmt.Sprintf("unknown mode %q (use remote or local)", target)}
	}

	cfg, err := s.deps.Repo.Get(ctx)
	if err != nil {
		return ModeUnspecified, ModeUnspecified, fmt.Errorf("read config: %w", err)
	}
	prev := cfg.Mode
	if prev == target {
		// Idempotent: no-op when already in the target mode.
		return prev, target, nil
	}

	// SwitchMode is PURE — it never writes ingress (that footgun is gone).
	// Switching to remote requires the remote read path be available so the
	// operator is never stranded in a mode they cannot reconcile. This is a
	// read-only credential resolve that constructs (but never calls) the
	// Cloudflare client; it pushes nothing.
	if target == ModeRemote {
		if _, err := s.remoteIngress(ctx); err != nil {
			return ModeUnspecified, ModeUnspecified, err
		}
	}

	cfg.Mode = target
	if _, err := s.deps.Repo.Upsert(ctx, cfg); err != nil {
		return ModeUnspecified, ModeUnspecified, fmt.Errorf("persist mode: %w", err)
	}
	return prev, target, nil
}

// GetDrift reconciles desired ∪ live ∪ ledger into a classified read model.
// It performs reads only — never an apply. In remote mode the live read goes
// through the Cloudflare API (ErrRemoteUnavailable when creds are absent); in
// local mode it parses ~/.cloudflared/config.yml best-effort.
func (s *service) GetDrift(ctx context.Context) (DriftReport, error) {
	cfg, err := s.deps.Repo.Get(ctx)
	if err != nil {
		return DriftReport{}, fmt.Errorf("read config: %w", err)
	}
	// Read live ingress from the real tunnel (Cloudflare) whenever credentials
	// are complete, even if the persisted mode is still the local default — see
	// effectiveMode. This is what makes the drift view show pre-existing remote
	// ingress instead of an empty local config.yml.
	cfg.Mode = s.effectiveMode(ctx, cfg)
	desired, err := s.desiredEntries(ctx)
	if err != nil {
		return DriftReport{}, err
	}
	live, err := s.liveIngress(ctx, cfg.Mode)
	if err != nil {
		return DriftReport{}, err
	}
	ledger, err := s.ledgerEntries(ctx)
	if err != nil {
		return DriftReport{}, err
	}
	return reconcile(cfg.Mode, desired, live, ledger), nil
}

// ledgerEntries reads the ownership ledger, tolerating an unwired ledger
// (returns no records ⇒ everything live-but-undesired is UNMANAGED, the safe
// default).
func (s *service) ledgerEntries(ctx context.Context) ([]LedgerEntry, error) {
	if s.deps.Ledger == nil {
		return nil, nil
	}
	return s.deps.Ledger.List(ctx)
}

// desiredIngress computes the ingress the routes manifest implies: one
// rule per enabled route plus a trailing catch-all 404.
//
// CRITICAL: hostnames are derived from route.Subdomain + "." + route.Domain
// (via route.PublicURL with the scheme stripped), NEVER a hardcoded apex.
// The old scenario hardcoded ".vrooli.com"; the live tunnel is
// ".itsagitime.com" and the manifest now carries the domain per route.
func (s *service) desiredIngress(ctx context.Context) ([]IngressRule, error) {
	entries, err := s.desiredEntries(ctx)
	if err != nil {
		return nil, err
	}
	rules := make([]IngressRule, 0, len(entries)+1)
	for _, e := range entries {
		rules = append(rules, IngressRule{Hostname: e.Hostname, Service: e.Service})
	}
	rules = append(rules, catchAll())
	return rules, nil
}

// desiredEntries computes the desired ingress set with provenance — one entry
// per enabled route, tagged scenario vs external so reconcile can classify
// EXTERNAL_OK distinctly from MANAGED. The catch-all (no hostname) is added
// only by desiredIngress at the apply boundary, never here.
func (s *service) desiredEntries(ctx context.Context) ([]DesiredEntry, error) {
	routes, err := s.deps.Routes.List(ctx, manifest.Tier(""))
	if err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}
	out := make([]DesiredEntry, 0, len(routes))
	for _, r := range routes {
		if !r.Enabled {
			continue
		}
		out = append(out, DesiredEntry{
			Hostname: extractHostname(r.PublicURL()),
			Service:  routeServiceTarget(r),
			Source:   routeSource(r),
			Scenario: r.Scenario,
			LeaseID:  r.LeaseID,
		})
	}
	return out, nil
}

// routeServiceTarget is the local service URL the tunnel forwards a route to.
// For scenario routes this is http://localhost:<local_port>; Phase 3 external
// routes carry a free-form target, honoured here once the manifest model
// grows the field.
func routeServiceTarget(r manifest.Route) string {
	if t := routeExternalTarget(r); t != "" {
		return t
	}
	return fmt.Sprintf("http://localhost:%d", r.LocalPort)
}

// routeSource maps a manifest route to its reconcile provenance. Scenario
// routes (the only kind before Phase 3) classify as MANAGED when live;
// external routes classify as EXTERNAL_OK.
func routeSource(r manifest.Route) RouteSource {
	if r.Source == manifest.SourceExternal {
		return SourceExternal
	}
	return SourceScenario
}

// routeExternalTarget returns the explicit service target for an external
// route, or "" for scenario routes (which derive http://localhost:<port>).
func routeExternalTarget(r manifest.Route) string {
	if r.Source == manifest.SourceExternal {
		return r.ServiceTarget
	}
	return ""
}

// effectiveMode resolves the mode whose ingress is the real source of truth for
// reads, reconcile, and per-entry drift actions. When Cloudflare credentials
// are complete the live tunnel IS Cloudflare, so TM operates against the real
// remote tunnel even while the persisted mode is still the local default.
//
// This is the fix for the drift view showing only manifest-derived state: a
// user who configured CF credentials but never flipped the mode toggle (local
// is the default) was previously shown drift computed against the local
// ~/.cloudflared/config.yml — which is empty for a dashboard/remote-managed
// tunnel — so every desired route looked MISSING and the real pre-existing
// remote ingress was invisible. With complete credentials the tunnel is
// remote-managed by definition, so we read and reconcile against Cloudflare.
//
// This never silently writes: apply is always additive and only runs on an
// explicit Sync/Prune. It falls back to the persisted/default mode when remote
// is unavailable (no or partial credentials), where ingress is the local
// cloudflared config.yml.
func (s *service) effectiveMode(ctx context.Context, cfg TunnelConfig) Mode {
	if defaultedMode(cfg.Mode) == ModeRemote {
		return ModeRemote
	}
	if _, err := s.remoteIngress(ctx); err == nil {
		return ModeRemote
	}
	return ModeLocal
}

// liveIngress reads the currently-applied ingress for the active mode.
func (s *service) liveIngress(ctx context.Context, mode Mode) ([]IngressRule, error) {
	if mode == ModeLocal {
		return s.readLocalIngress()
	}
	ingress, err := s.remoteIngress(ctx)
	if err != nil {
		return nil, err
	}
	return ingress.ReadIngress(ctx)
}

// applyAdditive publishes the desired ingress merged onto current live —
// never a full replace that drops unmanaged or ignored entries. It pushes the
// union (desired services win for matching hostnames), removing only the named
// pruneSet, then records ownership in the ledger so future reconciles can tell
// managed/external from drift.
func (s *service) applyAdditive(ctx context.Context, cfg TunnelConfig, desired []DesiredEntry, live []IngressRule, ledger []LedgerEntry, pruneSet []string) error {
	pruneByHost := make(map[string]bool, len(pruneSet))
	for _, h := range pruneSet {
		pruneByHost[h] = true
	}
	ignored := make(map[string]bool)
	for _, l := range ledger {
		if l.Owner == OwnerIgnored {
			ignored[l.Hostname] = true
		}
	}

	merged := mergeIngress(desired, live, ignored, pruneByHost)

	if cfg.Mode == ModeLocal {
		if err := s.writeLocalIngress(cfg, merged); err != nil {
			return err
		}
		if _, err := s.deps.Runner(ctx, "sudo", "systemctl", "restart", "cloudflared"); err != nil {
			return fmt.Errorf("restart cloudflared: %w", err)
		}
	} else {
		ingress, err := s.remoteIngress(ctx)
		if err != nil {
			return err
		}
		if err := ingress.PushIngress(ctx, merged); err != nil {
			return fmt.Errorf("push ingress: %w", err)
		}
	}

	s.recordOwnership(ctx, desired, ignored, pruneByHost)
	return nil
}

// mergeIngress computes the additive union to publish: start from current live
// (preserving every unmanaged/ignored/foreign entry), overlay desired
// scenario+external entries (their service wins), drop the named pruneSet, and
// append the catch-all. Ignored hostnames are never pushed from desired even
// if a route would otherwise publish them — the operator's "never touch this"
// decision is authoritative.
func mergeIngress(desired []DesiredEntry, live []IngressRule, ignored, prune map[string]bool) []IngressRule {
	byHost := make(map[string]IngressRule)
	var order []string
	put := func(r IngressRule) {
		if r.Hostname == "" {
			return // catch-all is appended once at the end
		}
		if _, seen := byHost[r.Hostname]; !seen {
			order = append(order, r.Hostname)
		}
		byHost[r.Hostname] = r
	}
	for _, r := range live {
		put(r)
	}
	for _, d := range desired {
		if d.Hostname == "" || ignored[d.Hostname] {
			continue
		}
		put(IngressRule{Hostname: d.Hostname, Service: d.Service})
	}

	out := make([]IngressRule, 0, len(order)+1)
	for _, h := range order {
		if prune[h] {
			continue
		}
		out = append(out, byHost[h])
	}
	out = append(out, catchAll())
	return out
}

// recordOwnership upserts MANAGED/EXTERNAL ledger entries for what we just
// published and clears ledger records for pruned hostnames, so a route later
// removed from the manifest classifies as ORPHANED (a prune candidate) rather
// than reappearing as fresh UNMANAGED drift.
func (s *service) recordOwnership(ctx context.Context, desired []DesiredEntry, ignored, prune map[string]bool) {
	if s.deps.Ledger == nil {
		return
	}
	for _, d := range desired {
		if d.Hostname == "" || ignored[d.Hostname] || prune[d.Hostname] {
			continue
		}
		owner := OwnerManaged
		if d.Source == SourceExternal {
			owner = OwnerExternal
		}
		// Best-effort: ownership tracking must not fail an otherwise-applied
		// reconcile.
		_ = s.deps.Ledger.Put(ctx, LedgerEntry{Hostname: d.Hostname, Owner: owner, Scenario: d.Scenario})
	}
	for h := range prune {
		_, _ = s.deps.Ledger.Delete(ctx, h)
	}
}

func (s *service) remoteIngress(ctx context.Context) (IngressClient, error) {
	if s.deps.CredentialStore == nil {
		if s.deps.Ingress == nil {
			return nil, ErrRemoteUnavailable{}
		}
		return s.deps.Ingress, nil
	}
	cfg, err := s.deps.CredentialStore.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve Cloudflare credentials: %w", err)
	}
	if !cfConfigComplete(cfg) {
		return nil, ErrRemoteUnavailable{}
	}
	if s.deps.Ingress != nil {
		return s.deps.Ingress, nil
	}
	return NewCFClient(nil, cfg), nil
}

func (s *service) readiness(ctx context.Context, cfg TunnelConfig) (ConfigReadiness, error) {
	status, err := s.credentialStatus(ctx)
	if err != nil {
		return ConfigReadiness{}, err
	}
	missing := status.MissingFields
	remoteAvailable := status.Ready
	if len(missing) == 0 && !status.Ready && !cfConfigComplete(s.deps.CF) {
		missing = append([]string(nil), cloudflareCredentialFields...)
		status.MissingFields = append([]string(nil), missing...)
		status.Ready = false
	}
	if s.deps.CredentialStore == nil && len(missing) == 0 && cfConfigComplete(s.deps.CF) {
		remoteAvailable = true
		status.Ready = true
	}
	if s.deps.CredentialStore == nil && s.deps.Ingress != nil {
		remoteAvailable = true
		status.Ready = true
	}
	// Report the effective mode so the UI badge/copy match what drift actually
	// reconciles against: complete credentials ⇒ remote, even when the persisted
	// mode is still the local default.
	desiredMode := s.effectiveMode(ctx, cfg)
	syncReady := desiredMode == ModeLocal || remoteAvailable
	return ConfigReadiness{
		DesiredMode:      desiredMode,
		RemoteAvailable:  remoteAvailable,
		MissingFields:    append([]string(nil), missing...),
		CredentialSource: credentialStatusSource(status),
		CredentialRef:    status.Ref,
		CredentialStatus: status,
		LocalConfigPath:  s.deps.LocalConfigPath,
		SyncReady:        syncReady,
		ModeReason:       readinessReason(desiredMode, remoteAvailable),
	}, nil
}

func (s *service) credentialStatus(ctx context.Context) (CredentialStatus, error) {
	if s.deps.CredentialStore != nil {
		return s.deps.CredentialStore.Status(ctx)
	}
	status := s.deps.CredentialStatus
	if hasCredentialStatus(status) {
		if len(status.Fields) == 0 {
			status.Ready = len(status.MissingFields) == 0
		}
		return status, nil
	}
	return statusFromCFConfig(s.deps.CF), nil
}

func hasCredentialStatus(status CredentialStatus) bool {
	return len(status.Fields) > 0 || len(status.MissingFields) > 0 || status.Source != "" || status.Ref != "" || status.Ready
}

func cfConfigComplete(cfg CFConfig) bool {
	return cfg.APIToken != "" && cfg.AccountID != "" && cfg.TunnelID != ""
}

func defaultedMode(mode Mode) Mode {
	if mode == ModeUnspecified {
		return DefaultMode
	}
	return mode
}

func credentialStatusSource(status CredentialStatus) string {
	if status.Source == "" {
		return credentialSourceMissing
	}
	return status.Source
}

func readinessReason(mode Mode, remoteAvailable bool) string {
	if mode != ModeRemote {
		return "Local mode is ready; sync writes the cloudflared config file and restarts cloudflared."
	}
	if remoteAvailable {
		return "Remote mode is ready; Cloudflare API credentials are present."
	}
	return "Remote mode is unavailable until CLOUDFLARE_ACCOUNT_ID, CLOUDFLARE_TUNNEL_ID, and CLOUDFLARE_API_TOKEN are configured."
}

func statusFromCFConfig(cfg CFConfig) CredentialStatus {
	fields := []CredentialFieldStatus{
		{Name: cloudflareAccountIDField, Present: cfg.AccountID != ""},
		{Name: cloudflareTunnelIDField, Present: cfg.TunnelID != ""},
		{Name: cloudflareAPITokenField, Present: cfg.APIToken != "", Ref: cfg.TokenRef},
	}
	for i := range fields {
		if fields[i].Present {
			fields[i].Source = cfg.Source
			fields[i].Writable = false
		} else {
			fields[i].Source = credentialSourceMissing
			fields[i].Writable = true
		}
	}
	status := buildCredentialStatus(fields)
	if cfg.Source != "" && cfg.Source != "none" {
		status.Source = cfg.Source
	}
	if cfg.TokenRef != "" {
		status.Ref = cfg.TokenRef
	}
	if len(cfg.Missing) > 0 {
		status.MissingFields = append([]string(nil), cfg.Missing...)
		status.Ready = false
	}
	return status
}

// readLocalIngress parses the ingress hostnames/services out of the local
// cloudflared config.yml. A missing file is treated as empty ingress (the
// first sync writes it).
func (s *service) readLocalIngress() ([]IngressRule, error) {
	data, err := os.ReadFile(s.deps.LocalConfigPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read local config: %w", err)
	}
	return parseIngressYAML(data), nil
}

// writeLocalIngress renders the cloudflared config.yml from the desired
// ingress, backing up any existing file first.
func (s *service) writeLocalIngress(cfg TunnelConfig, desired []IngressRule) error {
	dir := filepath.Dir(s.deps.LocalConfigPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if existing, err := os.ReadFile(s.deps.LocalConfigPath); err == nil {
		backup := fmt.Sprintf("%s.backup.%s", s.deps.LocalConfigPath, s.deps.Clock.Now().UTC().Format("20060102-150405"))
		if err := os.WriteFile(backup, existing, 0o644); err != nil {
			return fmt.Errorf("write backup: %w", err)
		}
	}
	if err := os.WriteFile(s.deps.LocalConfigPath, renderConfigYAML(cfg, desired), 0o644); err != nil {
		return fmt.Errorf("write local config: %w", err)
	}
	return nil
}

// catchAll is the trailing ingress rule cloudflared requires.
func catchAll() IngressRule { return IngressRule{Service: "http_status:404"} }

// extractHostname strips the URL scheme and path, returning only the
// hostname. e.g. "https://api.itsagitime.com/x" → "api.itsagitime.com".
// Ported from the old adapter/systemd.go so config and tunnel derive
// hostnames identically.
func extractHostname(publicURL string) string {
	host := strings.TrimPrefix(publicURL, "https://")
	host = strings.TrimPrefix(host, "http://")
	if i := strings.IndexByte(host, '/'); i >= 0 {
		host = host[:i]
	}
	return host
}

// renderConfigYAML writes a minimal cloudflared config.yml by hand. We do
// NOT pull in a YAML dependency (dependency changes require SDA, out of
// scope); the shape cloudflared needs is small and stable enough to emit
// directly.
func renderConfigYAML(cfg TunnelConfig, rules []IngressRule) []byte {
	var b bytes.Buffer
	if cfg.TunnelID != "" {
		fmt.Fprintf(&b, "tunnel: %s\n", cfg.TunnelID)
	}
	if cfg.CredRef != "" {
		fmt.Fprintf(&b, "credentials-file: %s\n", cfg.CredRef)
	}
	b.WriteString("ingress:\n")
	for _, r := range rules {
		if r.Hostname != "" {
			fmt.Fprintf(&b, "  - hostname: %s\n    service: %s\n", r.Hostname, r.Service)
		} else {
			fmt.Fprintf(&b, "  - service: %s\n", r.Service)
		}
	}
	return b.Bytes()
}

// parseIngressYAML extracts ingress hostname/service pairs from a minimal
// cloudflared config.yml. It only understands the shape renderConfigYAML
// emits (and the equivalent the operator writes) — enough to diff against
// the manifest. Lines are matched on the `hostname:`/`service:` keys.
func parseIngressYAML(data []byte) []IngressRule {
	var (
		rules   []IngressRule
		inItem  bool
		current IngressRule
	)
	flush := func() {
		if inItem {
			rules = append(rules, current)
		}
		current = IngressRule{}
		inItem = false
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "- hostname:"):
			flush()
			inItem = true
			current.Hostname = strings.TrimSpace(strings.TrimPrefix(line, "- hostname:"))
		case strings.HasPrefix(line, "- service:"):
			flush()
			inItem = true
			current.Service = strings.TrimSpace(strings.TrimPrefix(line, "- service:"))
		case strings.HasPrefix(line, "hostname:") && inItem:
			current.Hostname = strings.TrimSpace(strings.TrimPrefix(line, "hostname:"))
		case strings.HasPrefix(line, "service:") && inItem:
			current.Service = strings.TrimSpace(strings.TrimPrefix(line, "service:"))
		}
	}
	flush()
	// Drop any empty placeholder items.
	out := rules[:0]
	for _, r := range rules {
		if r.Hostname == "" && r.Service == "" {
			continue
		}
		out = append(out, r)
	}
	return out
}
