package main

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"time"

	"data-backup-manager/cli/domains"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "data-backup-manager"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	confirmLocalDBMOperatorEligible   = cliutil.ConfirmLocalOwnerEligible
	buildFingerprint                  = "unknown"
	buildTimestamp                    = "unknown"
	buildSourceRoot                   = ""
	exchangeLocalDBMOperatorPrincipal = func(ctx context.Context, resource string) (string, error) {
		login, err := authn.ExchangeLocalMachinePrincipalForResource(ctx, resource)
		if err != nil {
			return "", err
		}
		if login == nil || login.Tokens == nil {
			return "", errors.New("operator session exchange returned no token")
		}
		return strings.TrimSpace(login.Tokens.AccessToken), nil
	}
)

type App struct {
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	app := &App{}
	subcommandGroups := func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
		groups, err := domains.SubcommandGroups(core, manifestBytes)
		if err != nil {
			// Manifest parse / binding wiring is a programmer error caught
			// at NewApp time; surface it as a panic so misconfigured builds
			// fail loudly during the first CLI invocation rather than after
			// a user actually runs a command.
			panic(err)
		}
		return groups
	}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "Data Backup Manager CLI",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		CommandGroups:    domains.CommandGroups,
		SubcommandGroups: subcommandGroups,
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	if err := a.ensureWalletRecoverySession(args); err != nil {
		return err
	}
	return a.core.CLI.Run(args)
}

func (a *App) ensureWalletRecoverySession(args []string) error {
	if !isWalletRecoveryReadLease(args) {
		return nil
	}
	if a == nil || a.core == nil {
		return errors.New("DBM wallet recovery CLI is not configured")
	}
	if strings.HasPrefix(cliutil.ResolveCallerActor(""), "agent ") {
		return errors.New("DBM read-lease operations refuse agent-context callers; run explicitly from an eligible operator shell")
	}
	if hasConfiguredDBMToken(a.core) {
		return nil
	}
	base, err := url.Parse(strings.TrimSpace(a.core.APIRootBase()))
	if err != nil || base == nil || !isLoopbackHost(base.Hostname()) {
		return nil // remote DBM callers must supply the explicitly configured bearer token
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := confirmLocalDBMOperatorEligible(ctx); err != nil {
		return errors.New("DBM read-lease operations require confirmed local operator eligibility")
	}
	token, err := exchangeLocalDBMOperatorPrincipal(ctx, "data-backup-manager")
	if err != nil || strings.TrimSpace(token) == "" {
		return errors.New("DBM read-lease operations require the existing local operator session for data-backup-manager")
	}
	a.core.Config.Token = strings.TrimSpace(token)
	return nil
}

func isWalletRecoveryReadLease(args []string) bool {
	remaining, err := cliapp.ParseGlobalFlags(args, &cliapp.GlobalOptions{}, nil)
	return err == nil && len(remaining) >= 2 && remaining[0] == "wallet-recovery" && strings.HasPrefix(remaining[1], "read-lease-")
}

func hasConfiguredDBMToken(core *cliapp.ScenarioApp) bool {
	if core == nil || strings.TrimSpace(core.Config.Token) != "" {
		return core != nil && strings.TrimSpace(core.Config.Token) != ""
	}
	return strings.TrimSpace(os.Getenv("DATA_BACKUP_MANAGER_API_TOKEN")) != "" || strings.TrimSpace(os.Getenv("VROOLI_API_TOKEN")) != ""
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
