package lifecycle

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/vrooli/internal/scenario"
	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
	configconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config/config_v1connect"
)

// AuthenticationBindingResolver supplies non-secret provider metadata to the
// lifecycle. The default implementation delegates ownership to tunnel-manager
// so scenario manifests do not repeat Cloudflare audience configuration.
type AuthenticationBindingResolver interface {
	ResolveAuthenticationBinding(context.Context, scenario.Scenario) (scenario.AuthenticationBinding, error)
}

type tunnelManagerAuthenticationBindingResolver struct {
	httpClient *http.Client
	resolveURL func(context.Context, string) (string, error)
}

func (r tunnelManagerAuthenticationBindingResolver) ResolveAuthenticationBinding(ctx context.Context, item scenario.Scenario) (scenario.AuthenticationBinding, error) {
	baseURL := strings.TrimSpace(os.Getenv("VROOLI_TUNNEL_MANAGER_API_BASE"))
	if baseURL == "" {
		var err error
		baseURL, err = r.resolveURL(ctx, "tunnel-manager")
		if err != nil {
			return scenario.AuthenticationBinding{}, fmt.Errorf("discover tunnel-manager API: %w", err)
		}
	}
	client := configconnect.NewConfigServiceClient(r.httpClient, baseURL)
	response, err := client.GetAuthenticationBinding(ctx, connect.NewRequest(&configv1.GetAuthenticationBindingRequest{
		Scenario: item.Slug,
		Hostname: strings.TrimSpace(item.Manifest.Authentication.Hostname),
	}))
	if err != nil {
		return scenario.AuthenticationBinding{}, fmt.Errorf("resolve tunnel-manager authentication binding: %w", err)
	}
	if response.Msg == nil || response.Msg.Binding == nil {
		return scenario.AuthenticationBinding{}, fmt.Errorf("tunnel-manager returned an empty authentication binding")
	}
	binding := response.Msg.Binding
	if strings.TrimSpace(binding.TeamDomain) == "" || strings.TrimSpace(binding.Audience) == "" {
		return scenario.AuthenticationBinding{}, fmt.Errorf("tunnel-manager returned incomplete authentication metadata")
	}
	return scenario.AuthenticationBinding{
		TeamDomain:  binding.TeamDomain,
		Audience:    binding.Audience,
		RecoveryURL: binding.RecoveryUrl,
	}, nil
}

func defaultAuthenticationBindingResolver() AuthenticationBindingResolver {
	return tunnelManagerAuthenticationBindingResolver{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolveURL: discovery.ResolveScenarioURLDefault,
	}
}
