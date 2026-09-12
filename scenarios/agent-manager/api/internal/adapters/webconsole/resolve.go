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

// ResolveUIBaseURL resolves the web-console UI base URL (the browser-facing
// origin, not the API base) without hardcoding a port. Resolution mirrors
// ResolveBaseURL but targets the UI port:
//  1. WEB_CONSOLE_UI_URL / WEB_CONSOLE_UI_BASE env override (full base URL),
//  2. WEB_CONSOLE_UI_PORT env → http://localhost:<port>,
//  3. `vrooli scenario port web-console UI_PORT` live detection.
//
// It returns "" when the UI base cannot be located; callers surface that as an
// absent run-detail deep link (the client falls back to the session id) rather
// than building a link to a wrong hardcoded port.
func ResolveUIBaseURL() string {
	return cliutil.DetermineAPIBase(cliutil.APIBaseOptions{
		EnvVars:      []string{"WEB_CONSOLE_UI_URL", "WEB_CONSOLE_UI_BASE"},
		PortEnvVars:  []string{"WEB_CONSOLE_UI_PORT"},
		PortDetector: cliutil.DetectPortFromVrooli("web-console", "UI_PORT"),
	})
}
