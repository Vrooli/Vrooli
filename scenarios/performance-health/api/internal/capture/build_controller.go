package capture

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// The profile-build contract honored by the react-vite template's `build`
// script: `VROOLI_BUILD_MODE=profile vrooli scenario restart <name>` rebuilds
// the UI with `vite build --mode profile`, which aliases react-dom → profiling
// and keeps identifier names (esbuild.keepNames) so the <React.Profiler>
// onProfilerRender boundary survives minification and emits ⚛ marks.
const (
	profileBuildEnvVar = "VROOLI_BUILD_MODE"
	profileBuildValue  = "profile"
	uiPortKey          = "UI_PORT"

	// bundleMarker is the identifier the perf-build keeps in the served bundle;
	// its presence proves the page was built in profile mode (Tier-1 ready).
	bundleMarker = "onProfilerRender"
)

// CLIBuildController is the production BuildController: it restarts a scenario in
// profile build mode by shelling `vrooli scenario restart` with VROOLI_BUILD_MODE
// set, resolves the served UI URL via discovery, and verifies the served bundle
// actually carries the onProfilerRender marker before reporting the URL. A
// restart that yields no UI URL (no UI surface) is a clean skip upstream, not an
// error.
type CLIBuildController struct {
	// Restart restarts a scenario with the given extra env; nil shells
	// `vrooli scenario restart`. profileMode selects the build channel.
	Restart func(ctx context.Context, scenario string, profileMode bool) error

	// ResolveUIURL maps a scenario slug → served UI base URL; nil uses the
	// discovery resolver against UI_PORT. An error/empty means "no UI".
	ResolveUIURL func(ctx context.Context, scenario string) (string, error)

	// HTTPClient fetches the served bundle for verification; nil uses a
	// short-timeout default client.
	HTTPClient *http.Client

	// VerifyBundle confirms the served UI carries the profile-build marker;
	// nil uses the default HTTP fetch + marker scan.
	VerifyBundle func(ctx context.Context, uiURL string) (bool, error)
}

var _ BuildController = (*CLIBuildController)(nil)

// StartProfile restarts the scenario in profile build mode and returns the
// served UI URL, but only once the served bundle is verified instrumented.
// Returns ("", nil) when the scenario serves no UI (a clean skip upstream); an
// error only when the restart itself fails or the bundle is NOT instrumented
// (so the orchestrator does not capture a non-profile build and mislabel it).
func (c *CLIBuildController) StartProfile(ctx context.Context, scenario string) (string, error) {
	if err := c.restart(ctx, scenario, true); err != nil {
		return "", fmt.Errorf("profile-mode restart of %q failed: %w", scenario, err)
	}
	uiURL, err := c.resolveUIURL(ctx, scenario)
	if err != nil || strings.TrimSpace(uiURL) == "" {
		// No served UI: not an error — the orchestrator skips cleanly.
		return "", nil
	}
	instrumented, err := c.verifyBundle(ctx, uiURL)
	if err != nil {
		return "", fmt.Errorf("could not verify served bundle for %q: %w", scenario, err)
	}
	if !instrumented {
		return "", fmt.Errorf("served bundle for %q is not instrumented (missing %s); profile build did not take", scenario, bundleMarker)
	}
	return uiURL, nil
}

// RestoreDefault restarts the scenario back to its default (non-profile) build.
func (c *CLIBuildController) RestoreDefault(ctx context.Context, scenario string) error {
	return c.restart(ctx, scenario, false)
}

// ResolveURL returns the scenario's already-served UI URL without restarting it
// — used for Tier-0 captures, which need no profile build. Empty + nil means
// there is no served UI (a clean skip upstream).
func (c *CLIBuildController) ResolveURL(ctx context.Context, scenario string) (string, error) {
	url, err := c.resolveUIURL(ctx, scenario)
	if err != nil {
		// No served UI is a clean skip, not an error.
		return "", nil
	}
	return url, nil
}

func (c *CLIBuildController) restart(ctx context.Context, scenario string, profileMode bool) error {
	if c.Restart != nil {
		return c.Restart(ctx, scenario, profileMode)
	}
	return defaultRestart(ctx, scenario, profileMode)
}

func (c *CLIBuildController) resolveUIURL(ctx context.Context, scenario string) (string, error) {
	if c.ResolveUIURL != nil {
		return c.ResolveUIURL(ctx, scenario)
	}
	if strings.TrimSpace(scenario) == "" {
		return "", errors.New("scenario is required to resolve a UI URL")
	}
	return discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURL(ctx, scenario, uiPortKey)
}

func (c *CLIBuildController) verifyBundle(ctx context.Context, uiURL string) (bool, error) {
	if c.VerifyBundle != nil {
		return c.VerifyBundle(ctx, uiURL)
	}
	return defaultVerifyBundle(ctx, c.httpClient(), uiURL)
}

func (c *CLIBuildController) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

// defaultRestart shells `vrooli scenario restart <scenario>` with
// VROOLI_BUILD_MODE set to "profile" (profile build) or unset (default build).
func defaultRestart(ctx context.Context, scenario string, profileMode bool) error {
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "restart", scenario)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	env := os.Environ()
	if profileMode {
		env = append(env, profileBuildEnvVar+"="+profileBuildValue)
	}
	cmd.Env = env
	return cmd.Run()
}

// scriptSrcRe extracts <script src="..."> module entries from the served HTML.
var scriptSrcRe = regexp.MustCompile(`<script[^>]+src=["']([^"']+)["']`)

// defaultVerifyBundle fetches the served HTML, locates the JS module bundles,
// and reports whether any of them contains the onProfilerRender marker.
func defaultVerifyBundle(ctx context.Context, client *http.Client, uiURL string) (bool, error) {
	html, err := fetch(ctx, client, uiURL)
	if err != nil {
		return false, err
	}
	// Fast path: some bundlers inline the entry; check the HTML itself.
	if strings.Contains(html, bundleMarker) {
		return true, nil
	}
	base := strings.TrimRight(uiURL, "/")
	for _, m := range scriptSrcRe.FindAllStringSubmatch(html, -1) {
		src := strings.TrimSpace(m[1])
		if src == "" {
			continue
		}
		jsURL := src
		if !strings.HasPrefix(src, "http") {
			jsURL = base + "/" + strings.TrimPrefix(strings.TrimPrefix(src, "./"), "/")
		}
		js, err := fetch(ctx, client, jsURL)
		if err != nil {
			continue
		}
		if strings.Contains(js, bundleMarker) {
			return true, nil
		}
	}
	return false, nil
}

func fetch(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}
