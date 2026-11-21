package cliutil

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// ScenarioAPIBaseOptions separates peer addressing from local APIBaseOptions,
// so local port hints cannot accidentally route a peer client to itself.
type ScenarioAPIBaseOptions struct {
	Target     string
	Override   string
	EnvVars    []string
	ConfigBase string
	// AllowLiveFallback permits a shadow target to retry live only after a
	// typed scenario_not_running result. Variant dependencies stay isolated.
	AllowLiveFallback bool
	// PortCachePolicy opts into caller-bounded lookup caching. The zero value is fresh.
	PortCachePolicy PortCachePolicy
}

// ResolveScenarioAPIBaseURL requires a peer Target. It resolves only the
// explicit static override, environment, and config fields in
// ScenarioAPIBaseOptions before discovering that peer's API_PORT. Local port
// hints and defaults are not part of this API. Live fallback is opt-in and
// happens only after typed scenario_not_running for a non-live target.
func ResolveScenarioAPIBaseURL(ctx context.Context, opts ScenarioAPIBaseOptions) (string, error) {
	if strings.TrimSpace(opts.Target) == "" {
		return "", fmt.Errorf("scenario API target is required")
	}
	static := APIBaseOptions{Override: opts.Override, EnvVars: opts.EnvVars, ConfigBase: opts.ConfigBase}
	base := ResolveAPIBase(static).Base
	if base != "" {
		return NormalizeScenarioAPIBaseURL(base)
	}
	portVar := "API_PORT"
	target := ResolveShadowTarget(strings.TrimSpace(opts.Target))
	outcome := LookupScenarioPort(ctx, target, portVar, opts.PortCachePolicy)
	if !outcome.Resolved() && opts.AllowLiveFallback && ShouldFallbackToLiveInstance(target, IsScenarioPortNotRunning(outcome.Err)) {
		WarnShadowFallback(BareScenarioName(target))
		outcome = LookupScenarioPort(ctx, BareScenarioName(target), portVar, opts.PortCachePolicy)
	}
	if !outcome.Resolved() {
		if outcome.Err != nil {
			return "", fmt.Errorf("resolve scenario %s %s: %w", target, portVar, outcome.Err)
		}
		return "", fmt.Errorf("resolve scenario %s %s: port was empty", target, portVar)
	}
	return scenarioLoopbackBase(outcome.Port)
}

// NormalizeScenarioAPIBaseURL validates and canonicalizes an absolute HTTP(S)
// API base URL. It removes trailing slashes from the path while preserving
// query and fragment values.
func NormalizeScenarioAPIBaseURL(base string) (string, error) {
	base = strings.TrimSpace(base)
	parsed, err := url.Parse(base)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid scenario API base URL %q: expected an absolute http(s) URL", base)
	}
	// url.Parse accepts empty port delimiters and numeric ports outside TCP's
	// valid range. Reject both here so static service bases fail at config time.
	if strings.HasSuffix(parsed.Host, ":") {
		return "", fmt.Errorf("invalid scenario API base URL %q: port must be an integer from 1 to 65535", base)
	}
	if port := parsed.Port(); port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return "", fmt.Errorf("invalid scenario API base URL %q: port must be an integer from 1 to 65535", base)
		}
	}
	return normalizeAPIBase(base), nil
}

func scenarioLoopbackBase(port string) (string, error) {
	port = strings.TrimSpace(port)
	if port == "" {
		return "", fmt.Errorf("scenario API port is empty")
	}
	for _, digit := range port {
		if digit < '0' || digit > '9' {
			return "", fmt.Errorf("invalid scenario API port %q: expected digits only", port)
		}
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", fmt.Errorf("invalid scenario API port %q: expected an integer from 1 to 65535", port)
	}
	return "http://" + net.JoinHostPort("localhost", port), nil
}
