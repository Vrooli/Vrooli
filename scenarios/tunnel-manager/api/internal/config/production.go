package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tunnel-manager/internal/clock"
	"tunnel-manager/internal/cmdrunner"
	"tunnel-manager/internal/manifest"
)

// ProductionDB is the database surface required by the config service and
// the routes reader it composes. *database.RoutedDB satisfies this in
// production; *sql.DB satisfies it in integration tests. It embeds both the
// singleton config surface (SQLExecutor) and the multi-row ledger surface
// (LedgerSQLExecutor, which adds QueryContext).
type ProductionDB interface {
	SQLExecutor
	LedgerSQLExecutor
}

// ProductionOptions contains the side-effecting seams used only by the
// production builder. Tests can override each seam without duplicating handler
// wiring.
type ProductionOptions struct {
	Doer            httpDoer
	EnvLookup       func(string) string
	UserHomeDir     func() (string, error)
	HomeDir         string
	CredentialStore CredentialStore
	Runner          cmdrunner.Runner
	Routes          RoutesReader
	RoutesWriter    RoutesManager
	Scenarios       ScenarioResolver
	Ledger          OwnershipLedger
	LocalConfigPath string
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// NewProductionService builds the canonical config service used by both the
// config API and exposure's ingress adapter. Keep Cloudflare/env/local-runner
// wiring here so the two modules cannot drift.
func NewProductionService(db ProductionDB, clk clock.Clock, opts ProductionOptions) Service {
	if opts.EnvLookup == nil {
		opts.EnvLookup = os.Getenv
	}
	if opts.Doer == nil {
		opts.Doer = &http.Client{Timeout: 15 * time.Second}
	}
	store := opts.CredentialStore
	if store == nil {
		var err error
		store, err = NewCloudflareCredentialStore(CredentialStoreOptions{
			EnvLookup:   opts.EnvLookup,
			HomeDir:     opts.HomeDir,
			UserHomeDir: opts.UserHomeDir,
		})
		if err != nil {
			store = staticCredentialStore{cfg: ResolveCloudflareEnv(opts.EnvLookup), err: err}
		}
	}
	routesReader := opts.Routes
	if routesReader == nil {
		routesReader = unavailableRoutesReader{}
	}
	ledger := opts.Ledger
	if ledger == nil {
		ledger = NewSQLiteLedger(db, clk)
	}
	return NewService(Deps{
		Repo:            NewSQLiteRepository(db),
		Routes:          routesReader,
		RoutesWriter:    opts.RoutesWriter,
		Scenarios:       opts.Scenarios,
		Ingress:         resolvingIngressClient{store: store, doer: opts.Doer},
		Ledger:          ledger,
		CredentialStore: store,
		Verifier:        NewCFVerifier(opts.Doer),
		DNS:             resolvingDNSClient{store: store, doer: opts.Doer},
		DNSLedger:       NewSQLiteDNSLedger(db, clk),
		Runner:          opts.Runner,
		Clock:           clk,
		LocalConfigPath: opts.LocalConfigPath,
	})
}

// FileScenarioResolver resolves a scenario's fixed UI port from its
// service.json (<root>/<scenario>/.vrooli/service.json, ports.ui.port). It is
// the production ScenarioResolver used by AdoptIngress to tell scenario-backed
// hostnames from external ones. A nil/zero result (scenario unknown or no fixed
// UI port) makes adopt fall back to an external route.
type FileScenarioResolver struct {
	Root string
}

// NewFileScenarioResolver constructs a resolver rooted at the scenarios dir.
func NewFileScenarioResolver(root string) *FileScenarioResolver {
	return &FileScenarioResolver{Root: root}
}

var _ ScenarioResolver = (*FileScenarioResolver)(nil)

func (r *FileScenarioResolver) UIPort(_ context.Context, scenario string) (int, error) {
	if r == nil || strings.TrimSpace(r.Root) == "" || strings.TrimSpace(scenario) == "" {
		return 0, fmt.Errorf("scenario resolver not configured")
	}
	path := filepath.Join(r.Root, scenario, ".vrooli", "service.json")
	data, err := os.ReadFile(path) // #nosec G304 -- path built from a fixed root + validated DNS-label subdomain.
	if err != nil {
		return 0, fmt.Errorf("read service.json for %q: %w", scenario, err)
	}
	var svc struct {
		Ports map[string]struct {
			Port int `json:"port"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(data, &svc); err != nil {
		return 0, fmt.Errorf("parse service.json for %q: %w", scenario, err)
	}
	ui, ok := svc.Ports["ui"]
	if !ok || ui.Port == 0 {
		return 0, fmt.Errorf("scenario %q has no fixed UI port", scenario)
	}
	return ui.Port, nil
}

// IsScenario reports whether the slug names a real scenario (its service.json
// exists), even when that scenario's UI port is ranged rather than fixed.
func (r *FileScenarioResolver) IsScenario(_ context.Context, scenario string) bool {
	if r == nil || strings.TrimSpace(r.Root) == "" || strings.TrimSpace(scenario) == "" {
		return false
	}
	path := filepath.Join(r.Root, scenario, ".vrooli", "service.json")
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

type unavailableRoutesReader struct{}

func (unavailableRoutesReader) List(context.Context, manifest.Tier) ([]manifest.Route, error) {
	return nil, fmt.Errorf("routes reader is not configured")
}

type resolvingIngressClient struct {
	store CredentialStore
	doer  httpDoer
}

func (c resolvingIngressClient) ReadIngress(ctx context.Context) ([]IngressRule, error) {
	ingress, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	return ingress.ReadIngress(ctx)
}

func (c resolvingIngressClient) PushIngress(ctx context.Context, rules []IngressRule) error {
	ingress, err := c.client(ctx)
	if err != nil {
		return err
	}
	return ingress.PushIngress(ctx, rules)
}

func (c resolvingIngressClient) client(ctx context.Context) (IngressClient, error) {
	cfg, err := c.store.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve Cloudflare credentials: %w", err)
	}
	client := NewCFClient(c.doer, cfg)
	if client == nil {
		return nil, ErrRemoteUnavailable{}
	}
	return client, nil
}

// resolvingDNSClient resolves Cloudflare credentials per call and builds a
// cfDNSClient over them, mirroring resolvingIngressClient so the config API and
// exposure's reconcile share one credential-resolution path. It returns nil-ish
// behaviour (no-op via ErrRemoteUnavailable) when credentials are absent.
type resolvingDNSClient struct {
	store CredentialStore
	doer  httpDoer
}

func (c resolvingDNSClient) EnsureRecord(ctx context.Context, hostname string) (DNSResult, error) {
	client, err := c.client(ctx)
	if err != nil {
		return DNSResult{}, err
	}
	return client.EnsureRecord(ctx, hostname)
}

func (c resolvingDNSClient) RemoveRecord(ctx context.Context, hostname string) (bool, error) {
	client, err := c.client(ctx)
	if err != nil {
		return false, err
	}
	return client.RemoveRecord(ctx, hostname)
}

func (c resolvingDNSClient) client(ctx context.Context) (DNSClient, error) {
	cfg, err := c.store.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve Cloudflare credentials: %w", err)
	}
	client := NewCFDNSClient(c.doer, cfg)
	if client == nil {
		return nil, ErrRemoteUnavailable{}
	}
	return client, nil
}

type staticCredentialStore struct {
	cfg CFConfig
	err error
}

func (s staticCredentialStore) Status(context.Context) (CredentialStatus, error) {
	return statusFromCFConfig(s.cfg), nil
}

func (s staticCredentialStore) Resolve(context.Context) (CFConfig, error) {
	return s.cfg, nil
}

func (s staticCredentialStore) Save(context.Context, CredentialUpdate) (CredentialStatus, error) {
	return CredentialStatus{}, s.err
}

func (s staticCredentialStore) Delete(context.Context, []string) (CredentialStatus, error) {
	return CredentialStatus{}, s.err
}
