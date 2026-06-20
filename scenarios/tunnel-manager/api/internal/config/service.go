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
	internalroutes "tunnel-manager/internal/routes"
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

	// Sync reconciles live ingress with the routes manifest. It computes
	// the desired ingress (enabled routes → subdomain.domain hostnames →
	// http://localhost:<port>, plus a catch-all 404), diffs it against the
	// live ingress, and — when !dryRun and there is drift — applies it
	// (PushIngress in remote mode; write config.yml + restart cloudflared
	// in local mode). NoChanges is set when the diff is empty.
	Sync(ctx context.Context, dryRun bool) (SyncResult, error)

	// SwitchMode migrates between remote and local management, persists the
	// new mode, and applies the corresponding ingress for the current
	// manifest. Returns the previous and current modes.
	SwitchMode(ctx context.Context, target Mode) (prev, cur Mode, err error)
}

// Deps wires the seams the config service depends on. IngressClient is nil
// when Cloudflare credentials are absent — remote operations then return
// ErrRemoteUnavailable instead of touching the network.
type Deps struct {
	Repo             ConfigRepository
	Routes           RoutesReader
	Ingress          IngressClient
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
	return ConfigState{
		Config:    cfg,
		Readiness: s.readiness(cfg),
	}, nil
}

func (s *service) Sync(ctx context.Context, dryRun bool) (SyncResult, error) {
	cfg, err := s.deps.Repo.Get(ctx)
	if err != nil {
		return SyncResult{}, fmt.Errorf("read config: %w", err)
	}

	desired, err := s.desiredIngress(ctx)
	if err != nil {
		return SyncResult{}, err
	}

	if cfg.Mode == ModeRemote && s.deps.Ingress == nil && dryRun {
		readiness := s.readiness(cfg)
		return SyncResult{
			Mode:          cfg.Mode,
			SetupRequired: true,
			MissingFields: readiness.MissingFields,
			Message:       readiness.ModeReason,
		}, nil
	}

	live, err := s.liveIngress(ctx, cfg.Mode)
	if err != nil {
		return SyncResult{}, err
	}

	added, removed := diffIngress(live, desired)
	result := SyncResult{
		Mode:      cfg.Mode,
		Added:     added,
		Removed:   removed,
		NoChanges: len(added) == 0 && len(removed) == 0,
	}
	if result.NoChanges {
		result.Message = fmt.Sprintf("Ingress already matches the routes manifest in %s mode.", cfg.Mode)
	} else if dryRun {
		result.Message = fmt.Sprintf("Dry-run found %d hostnames to add and %d to remove in %s mode.", len(added), len(removed), cfg.Mode)
	} else {
		result.Message = fmt.Sprintf("Ingress reconciled in %s mode.", cfg.Mode)
	}

	if dryRun || result.NoChanges {
		return result, nil
	}

	if err := s.applyIngress(ctx, cfg, desired); err != nil {
		return SyncResult{}, err
	}
	return result, nil
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

	desired, err := s.desiredIngress(ctx)
	if err != nil {
		return ModeUnspecified, ModeUnspecified, err
	}

	// Apply the manifest's ingress through the target mode's channel before
	// persisting, so a failed switch leaves the persisted mode unchanged.
	applyCfg := cfg
	applyCfg.Mode = target
	if err := s.applyIngress(ctx, applyCfg, desired); err != nil {
		return ModeUnspecified, ModeUnspecified, err
	}

	cfg.Mode = target
	if _, err := s.deps.Repo.Upsert(ctx, cfg); err != nil {
		return ModeUnspecified, ModeUnspecified, fmt.Errorf("persist mode: %w", err)
	}
	return prev, target, nil
}

// desiredIngress computes the ingress the routes manifest implies: one
// rule per enabled route plus a trailing catch-all 404.
//
// CRITICAL: hostnames are derived from route.Subdomain + "." + route.Domain
// (via route.PublicURL with the scheme stripped), NEVER a hardcoded apex.
// The old scenario hardcoded ".vrooli.com"; the live tunnel is
// ".itsagitime.com" and the manifest now carries the domain per route.
func (s *service) desiredIngress(ctx context.Context) ([]IngressRule, error) {
	routes, err := s.deps.Routes.List(ctx, internalroutes.Tier(""))
	if err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}
	var rules []IngressRule
	for _, r := range routes {
		if !r.Enabled {
			continue
		}
		rules = append(rules, IngressRule{
			Hostname: extractHostname(r.PublicURL()),
			Service:  fmt.Sprintf("http://localhost:%d", r.LocalPort),
		})
	}
	rules = append(rules, catchAll())
	return rules, nil
}

// liveIngress reads the currently-applied ingress for the active mode.
func (s *service) liveIngress(ctx context.Context, mode Mode) ([]IngressRule, error) {
	if mode == ModeLocal {
		return s.readLocalIngress()
	}
	// Remote (and unspecified, which defaults to remote).
	if s.deps.Ingress == nil {
		return nil, ErrRemoteUnavailable{}
	}
	return s.deps.Ingress.ReadIngress(ctx)
}

// applyIngress pushes the desired ingress through the active mode's channel.
func (s *service) applyIngress(ctx context.Context, cfg TunnelConfig, desired []IngressRule) error {
	if cfg.Mode == ModeLocal {
		if err := s.writeLocalIngress(cfg, desired); err != nil {
			return err
		}
		if _, err := s.deps.Runner(ctx, "sudo", "systemctl", "restart", "cloudflared"); err != nil {
			return fmt.Errorf("restart cloudflared: %w", err)
		}
		return nil
	}
	if s.deps.Ingress == nil {
		return ErrRemoteUnavailable{}
	}
	if err := s.deps.Ingress.PushIngress(ctx, desired); err != nil {
		return fmt.Errorf("push ingress: %w", err)
	}
	return nil
}

func (s *service) readiness(cfg TunnelConfig) ConfigReadiness {
	status := s.effectiveCredentialStatus()
	missing := status.MissingFields
	if len(missing) == 0 && !cfConfigComplete(s.deps.CF) {
		missing = append([]string(nil), cloudflareCredentialFields...)
		status.MissingFields = append([]string(nil), missing...)
		status.Ready = false
	}
	remoteAvailable := len(missing) == 0 && cfConfigComplete(s.deps.CF)
	if s.deps.Ingress != nil {
		remoteAvailable = true
		status.Ready = true
	}
	desiredMode := defaultedMode(cfg.Mode)
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
	}
}

func (s *service) effectiveCredentialStatus() CredentialStatus {
	status := s.deps.CredentialStatus
	if hasCredentialStatus(status) {
		if len(status.Fields) == 0 {
			status.Ready = len(status.MissingFields) == 0
		}
		return status
	}
	return statusFromCFConfig(s.deps.CF)
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

// diffIngress compares live and desired ingress by hostname (the catch-all
// 404, which has no hostname, is ignored — it is always present). Returns
// the hostnames added (in desired, not live) and removed (in live, not
// desired), each sorted for deterministic output.
func diffIngress(live, desired []IngressRule) (added, removed []string) {
	liveSet := hostnameSet(live)
	desiredSet := hostnameSet(desired)
	for h := range desiredSet {
		if !liveSet[h] {
			added = append(added, h)
		}
	}
	for h := range liveSet {
		if !desiredSet[h] {
			removed = append(removed, h)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

func hostnameSet(rules []IngressRule) map[string]bool {
	set := make(map[string]bool, len(rules))
	for _, r := range rules {
		if r.Hostname == "" {
			continue // catch-all has no hostname
		}
		set[r.Hostname] = true
	}
	return set
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
