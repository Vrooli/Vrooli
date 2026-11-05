package discovery

import (
	"context"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	exposure "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure"
	"github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure/exposure_v1connect"
)

// isExposedViaTunnel asks a Tunnel Manager at tunnelBase whether scenarioSlug is
// exposed and, if so, returns its public URL. It is a package var so tests can
// stand in for the network call. A non-exposed scenario or any RPC error
// reports (\"\", false).
var isExposedViaTunnel = func(ctx context.Context, tunnelBase, scenarioSlug string) (string, bool) {
	client := exposure_v1connect.NewExposureServiceClient(http.DefaultClient, tunnelBase)
	resp, err := client.IsExposed(ctx, connect.NewRequest(&exposure.IsExposedRequest{Scenario: scenarioSlug}))
	if err != nil || resp == nil || resp.Msg == nil || !resp.Msg.GetExposed() {
		return "", false
	}
	return strings.TrimRight(strings.TrimSpace(resp.Msg.GetPublicUrl()), "/"), true
}

// ResolveExternalURLForNotice resolves a browser-openable URL for a background
// caller that has NO request origin to work from — a push notification or an
// email, built for a person who is not on this machine, where a localhost port
// would be useless. It prefers the scenario's public tunnel URL (so the link
// works from anywhere) and falls back to a localhost port only when the
// scenario is not exposed.
//
// Unlike ResolveExternalURL, it may make a network call to Tunnel Manager, so it
// is for infrequent background use — never a hot request path.
func (r *Resolver) ResolveExternalURLForNotice(ctx context.Context, scenarioSlug string) (string, error) {
	if pub := r.publicURLFromTunnel(ctx, scenarioSlug); pub != "" {
		return pub, nil
	}
	port, err := r.ResolveScenarioPort(ctx, scenarioSlug, UIPortKey)
	if err != nil {
		return "", err
	}
	// Empty host → a localhost port, the only honest answer without a tunnel.
	return BrowserURLForHost("", scenarioSlug, port), nil
}

// ResolveExternalURLForNotice is a convenience wrapper using the shared resolver.
func ResolveExternalURLForNotice(ctx context.Context, scenarioSlug string) (string, error) {
	return DefaultResolver().ResolveExternalURLForNotice(ctx, scenarioSlug)
}

// publicURLFromTunnel asks Tunnel Manager for scenarioSlug's public URL.
//
// ResolveExternalURLForNotice consults it for a background job (a push
// notification, an email) building a link for a person who is not on this
// machine. Every failure path (Tunnel Manager unresolved or down, the scenario
// not exposed) returns "" so the caller falls back to a localhost port.
func (r *Resolver) publicURLFromTunnel(ctx context.Context, scenarioSlug string) string {
	scenarioSlug = strings.TrimSpace(scenarioSlug)
	if scenarioSlug == "" {
		return ""
	}
	base, err := r.ResolveScenarioURLDefault(ctx, "tunnel-manager")
	if err != nil || strings.TrimSpace(base) == "" {
		return ""
	}
	if url, ok := isExposedViaTunnel(ctx, base, scenarioSlug); ok {
		return url
	}
	return ""
}
