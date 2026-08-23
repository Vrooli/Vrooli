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

// The profile-build contract: `VROOLI_BUILD_MODE=profile vrooli scenario
// restart <name>` makes the lifecycle builder run the UI's `build:profile`
// script, which builds with `vite build --mode profile`. That aliases react-dom
// → react-dom/profiling and keeps identifier names (esbuild.keepNames) so the
// <React.Profiler> boundary emits ⚛ marks.
const (
	profileBuildEnvVar = "VROOLI_BUILD_MODE"
	profileBuildValue  = "profile"
	uiPortKey          = "UI_PORT"
)

// bundleMarkers are react-dom/profiling internals. Their presence in the served
// bundle proves the page was built against the profiling react-dom, which is
// what actually decides whether <React.Profiler>'s onRender fires.
//
// The app-level `onProfilerRender` name is NOT a usable marker: when a scenario
// reaches its profiler util through a dynamic import, the name becomes a chunk
// export and survives minification in the ordinary production build too. Keying
// on it reports a plain prod bundle as instrumented, and the capture then yields
// a trace with no ⚛ marks that reads as "no component activity" rather than
// "not a perf build". Several markers are listed so a React release that renames
// one internal degrades to a narrower check instead of a false negative.
var bundleMarkers = []string{
	"injectProfilingHooks",
	"markComponentRenderStarted",
}

// bundleMarkerDescription names the markers for operator-facing errors.
var bundleMarkerDescription = strings.Join(bundleMarkers, " or ")

// containsBundleMarker reports whether source carries any profiling-build marker.
func containsBundleMarker(source string) bool {
	for _, marker := range bundleMarkers {
		if strings.Contains(source, marker) {
			return true
		}
	}
	return false
}

// CLIBuildController is the production BuildController: it restarts a scenario in
// profile build mode by shelling `vrooli scenario restart` with VROOLI_BUILD_MODE
// set, resolves the served UI URL via discovery, and verifies the served bundle
// actually carries a react-dom/profiling marker before reporting the URL. A
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
		return "", fmt.Errorf("served bundle for %q is not instrumented (missing %s); profile build did not take", scenario, bundleMarkerDescription)
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
// and reports whether any of them carries a profiling-build marker.
func defaultVerifyBundle(ctx context.Context, client *http.Client, uiURL string) (bool, error) {
	html, err := fetch(ctx, client, uiURL)
	if err != nil {
		return false, err
	}
	// Fast path: some bundlers inline the entry; check the HTML itself.
	if containsBundleMarker(html) {
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
		if containsBundleMarker(js) {
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
