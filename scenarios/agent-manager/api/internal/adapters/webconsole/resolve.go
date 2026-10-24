package webconsole

import (
	"github.com/vrooli/cli-core/cliutil"
)

// ResolveBaseURL resolves the web-console API base URL without hardcoding a
// port. Resolution order (via cliutil.DetermineAPIBase):
//  1. WEB_CONSOLE_URL / WEB_CONSOLE_API_BASE env override (full base URL),
//  2. WEB_CONSOLE_API_PORT env → http://localhost:<port>,
//  3. `vrooli scenario port web-console API_PORT` live detection.
//
// It returns "" when web-console cannot be located; callers surface that as a
// clear "web-console unavailable" error at session-create time rather than
// dialing a wrong hardcoded port.
func ResolveBaseURL() string {
	return cliutil.DetermineAPIBase(cliutil.APIBaseOptions{
		EnvVars:      []string{"WEB_CONSOLE_URL", "WEB_CONSOLE_API_BASE"},
		PortEnvVars:  []string{"WEB_CONSOLE_API_PORT"},
		PortDetector: cliutil.DetectPortFromVrooli("web-console", "API_PORT"),
	})
}

// ResolveUIBaseOverride returns an operator-pinned web-console UI base URL from
// the environment (WEB_CONSOLE_UI_URL / WEB_CONSOLE_UI_BASE), or "" when none is
// set.
//
// It deliberately does NOT fall back to a localhost port. An absent override
// means "resolve the UI base per request via the canonical resolver" — which is
// tunnel-aware and therefore produces links that work when opened from another
// machine (a phone), unlike a localhost port captured once at startup.
func ResolveUIBaseOverride() string {
	return cliutil.DetermineAPIBase(cliutil.APIBaseOptions{
		EnvVars: []string{"WEB_CONSOLE_UI_URL", "WEB_CONSOLE_UI_BASE"},
	})
}
