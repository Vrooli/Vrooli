package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

var (
	startScenarioFunc = startScenarioViaVrooli
	healthCheckFunc   = checkHealth
	canAutostartFunc  = canAutostartScenario
)

func ensureScenarioAPIReady(app *cliapp.ScenarioApp, global cliapp.GlobalOptions, scenarioName string) error {
	if app == nil {
		return fmt.Errorf("cli not initialized")
	}

	// If the caller explicitly overrides API base, respect it and don't try to manage scenarios.
	if strings.TrimSpace(global.APIBaseOverride) != "" {
		return nil
	}

	base := cliutil.DetermineAPIBase(app.APIBaseOptions())
	if !isLocalBase(base) {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := healthCheckFunc(ctx, base); err == nil {
		return nil
	}

	if !canAutostartFunc(scenarioName) {
		return fmt.Errorf("%s API is not reachable; start it with `cd scenarios/%s && make start` (or configure --api-base to a running instance)", scenarioName, scenarioName)
	}

	if err := startScenarioFunc(ctx, scenarioName); err != nil {
		return fmt.Errorf("failed to start %s scenario: %w", scenarioName, err)
	}

	// After starting, prefer re-resolving base (port may be allocated dynamically).
	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		base = cliutil.DetermineAPIBase(app.APIBaseOptions())
		if isLocalBase(base) {
			if err := healthCheckFunc(ctx, base); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("API not reachable")
	}
	return fmt.Errorf("%s API did not become ready: %w", scenarioName, lastErr)
}

func canAutostartScenario(scenarioName string) bool {
	if strings.TrimSpace(scenarioName) == "" {
		return false
	}
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		return false
	}
	scenarioPath := filepath.Join(repoRoot, "scenarios", scenarioName)
	if !statDir(scenarioPath) {
		return false
	}

	// Avoid triggering lifecycle setup that installs dependencies. Only auto-start when
	// the scenario appears already built.
	apiBinary := filepath.Join(scenarioPath, "api", scenarioName+"-api")
	if _, err := os.Stat(apiBinary); err != nil {
		return false
	}

	// prd-control-tower lifecycle currently includes UI steps in setup; require a built bundle
	// so auto-start doesn't install pnpm/node modules.
	uiBundle := filepath.Join(scenarioPath, "ui", "dist", "index.html")
	if _, err := os.Stat(uiBundle); err != nil {
		return false
	}

	return true
}

func isLocalBase(base string) bool {
	base = strings.TrimSpace(base)
	if base == "" {
		return false
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" {
		return false
	}
	host := parsed.Hostname()
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func startScenarioViaVrooli(ctx context.Context, scenarioName string) error {
	if strings.TrimSpace(scenarioName) == "" {
		return fmt.Errorf("scenario name is empty")
	}
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "start", scenarioName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func checkHealth(ctx context.Context, base string) error {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return fmt.Errorf("api base is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/health", nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("health returned %d", resp.StatusCode)
	}
	return nil
}
