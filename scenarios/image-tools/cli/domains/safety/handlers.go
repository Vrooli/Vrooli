// Package safety is the CLI's Responsible-Use policy surface. `safety policy`
// mirrors the Connect SafetyService.GetPolicy RPC — it reports the resolved
// deployment policy (local = unrestricted; public = consent-gated + NSFW
// auto-scan + provenance + rate-limit) and the per-operation consent weights.
package safety

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety"
	safetyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety/safety_v1connect"
)

// GroupName is the manifest group name this package owns.
const GroupName = "safety"

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers { return &handlers{core: core} }

// policy mirrors SafetyService.GetPolicy (Connect discovery).
func (h *handlers) policy(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := safetyconnect.NewSafetyServiceClient(httpClient, baseURL)
	resp, err := client.GetPolicy(context.Background(), connect.NewRequest(&safetyv1.GetPolicyRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get policy", err, nil)
	}
	p := resp.Msg
	results := []string{
		fmt.Sprintf("tier:               %s", tierLabel(p.GetTier())),
		fmt.Sprintf("require consent:    %t", p.GetRequireConsent()),
		fmt.Sprintf("force NSFW scan:    %t", p.GetForceNsfwScan()),
		fmt.Sprintf("require provenance: %t", p.GetRequireProvenance()),
		fmt.Sprintf("rate limit/min:     %d", p.GetRateLimitPerMin()),
	}
	for _, ow := range p.GetOpWeights() {
		results = append(results, fmt.Sprintf("op %-16s %s", ow.GetOperation(), weightLabel(ow.GetWeight())))
	}
	return cliapp.RenderProtoList(ctx, p, cliapp.ListReport{
		Summary:        []string{p.GetSummary()},
		ResultsHeading: "Policy",
		Results:        results,
	})
}

func tierLabel(t safetyv1.DeploymentTier) string {
	switch t {
	case safetyv1.DeploymentTier_DEPLOYMENT_TIER_PUBLIC:
		return "public"
	case safetyv1.DeploymentTier_DEPLOYMENT_TIER_LOCAL:
		return "local"
	default:
		return "local"
	}
}

func weightLabel(w safetyv1.ConsentWeight) string {
	switch w {
	case safetyv1.ConsentWeight_CONSENT_WEIGHT_HIGH:
		return "high (consent-gated on public)"
	case safetyv1.ConsentWeight_CONSENT_WEIGHT_LOW:
		return "low"
	default:
		return "none"
	}
}
