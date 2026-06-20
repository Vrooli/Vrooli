package config

import (
	"context"
	"net/http"
	"os"
	"time"

	"tunnel-manager/internal/clock"
	"tunnel-manager/internal/cmdrunner"
	internalroutes "tunnel-manager/internal/routes"
)

// ProductionDB is the database surface required by the config service and
// the routes reader it composes. *database.RoutedDB satisfies this in
// production; *sql.DB satisfies it in integration tests.
type ProductionDB interface {
	SQLExecutor
	internalroutes.SQLExecutor
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
	cfConfig, err := store.Resolve(context.Background())
	if err != nil {
		cfConfig = ResolveCloudflareEnv(opts.EnvLookup)
	}
	credentialStatus, err := store.Status(context.Background())
	if err != nil {
		credentialStatus = statusFromCFConfig(cfConfig)
	}
	routesReader := internalroutes.NewService(internalroutes.NewSQLiteRepository(db, clk))
	return NewService(Deps{
		Repo:             NewSQLiteRepository(db),
		Routes:           routesReader,
		Ingress:          NewCFClient(opts.Doer, cfConfig),
		CF:               cfConfig,
		CredentialStatus: credentialStatus,
		Runner:           opts.Runner,
		Clock:            clk,
		LocalConfigPath:  opts.LocalConfigPath,
	})
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
