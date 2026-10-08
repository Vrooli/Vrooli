package portability

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	portabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability"
	portabilityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability/portability_v1connect"
)

type handlers struct {
	client portabilityconnect.PortabilityServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: portabilityconnect.NewPortabilityServiceClient(httpClient, baseURL)}
}

func (h *handlers) gridCall(ctx cliapp.OperationContext) (*portabilityv1.GetGridResponse, error) {
	resp, err := h.client.GetGrid(context.Background(), connect.NewRequest(&portabilityv1.GetGridRequest{
		Capabilities: splitList(ctx.Flag("capability")),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("portability grid", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetGrid() == nil {
		return nil, fmt.Errorf("server returned no portability grid")
	}
	return resp.Msg, nil
}

func (h *handlers) gridReport(_ cliapp.OperationContext, msg *portabilityv1.GetGridResponse) cliapp.ListReport {
	grid := msg.GetGrid()
	results := make([]string, 0, len(grid.GetCapabilities())*4)
	for _, entry := range grid.GetCapabilities() {
		results = append(results, formatEntry(entry)...)
	}
	return cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("%d capability row(s) from %d manifest(s).", len(grid.GetCapabilities()), grid.GetManifestsRead()),
			"Manifest root: " + grid.GetManifestRoot(),
		},
		ResultsHeading: "Capabilities",
		Results:        results,
		RetrievalHints: []string{"`portability capability <name>`", "`portability situations --situation real-peer-nobody-wired`", "`portability fleet`"},
	}
}

func (h *handlers) capabilityCall(ctx cliapp.OperationContext) (*portabilityv1.GetCapabilityResponse, error) {
	name := strings.TrimSpace(ctx.Positional("capability"))
	if name == "" {
		return nil, fmt.Errorf("capability name is required")
	}
	resp, err := h.client.GetCapability(context.Background(), connect.NewRequest(&portabilityv1.GetCapabilityRequest{Capability: name}))
	if err != nil {
		return nil, cliapp.WrapAPIError("portability capability", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetCapability() == nil {
		return nil, fmt.Errorf("server returned no capability row")
	}
	return resp.Msg, nil
}

func (h *handlers) capabilityReport(_ cliapp.OperationContext, msg *portabilityv1.GetCapabilityResponse) cliapp.ListReport {
	return cliapp.ListReport{
		Summary:        []string{"Manifest root: " + msg.GetManifestRoot()},
		ResultsHeading: "Capability",
		Results:        formatEntry(msg.GetCapability()),
	}
}

func (h *handlers) situationsCall(ctx cliapp.OperationContext) (*portabilityv1.ListSituationsResponse, error) {
	situation, err := situationFlag(ctx.Flag("situation"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ListSituations(context.Background(), connect.NewRequest(&portabilityv1.ListSituationsRequest{Situation: situation}))
	if err != nil {
		return nil, cliapp.WrapAPIError("portability situations", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no situations response")
	}
	return resp.Msg, nil
}

func (h *handlers) situationsReport(_ cliapp.OperationContext, msg *portabilityv1.ListSituationsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetCapabilities()))
	for _, entry := range msg.GetCapabilities() {
		results = append(results, fmt.Sprintf("%-28s %-24s %s", entry.GetCapability(), situationToken(entry.GetSituation()), entry.GetSituationReason()))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d capability row(s).", len(results)), "Manifest root: " + msg.GetManifestRoot()},
		ResultsHeading: "Situations",
		Results:        results,
	}
}

func (h *handlers) fleetCall(_ cliapp.OperationContext) (*portabilityv1.GetFleetResponse, error) {
	resp, err := h.client.GetFleet(context.Background(), connect.NewRequest(&portabilityv1.GetFleetRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("portability fleet", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetFleet() == nil {
		return nil, fmt.Errorf("server returned no fleet readout")
	}
	return resp.Msg, nil
}

func (h *handlers) fleetReport(ctx cliapp.OperationContext, msg *portabilityv1.GetFleetResponse) cliapp.ListReport {
	fleet := msg.GetFleet()
	if !fleet.GetAvailable() {
		return cliapp.ListReport{
			Summary:        []string{"Fleet verdict unavailable: " + fleet.GetReason()},
			ResultsHeading: "Fleet",
			Results:        []string{"SDA is unavailable; no local dependency resolver was used."},
		}
	}
	view := strings.ToLower(strings.TrimSpace(ctx.Flag("view")))
	results := make([]string, 0)
	switch view {
	case "blocked":
		results = append(results, formatBlocks(fleet.GetBlockedByOs())...)
	case "docker":
		results = append(results, formatBlocks(fleet.GetDockerBlocked())...)
	case "peerless":
		for _, item := range fleet.GetPeerless() {
			results = append(results, fmt.Sprintf("%-28s %-8s %s", item.GetScenario(), osToken(item.GetHostOs()), strings.Join(item.GetCapabilities(), ", ")))
		}
	case "upgrades":
		for _, item := range fleet.GetTierUpgrades() {
			results = append(results, fmt.Sprintf("%-28s %-8s %s -> %s: %s", item.GetScenario(), osToken(item.GetHostOs()), tierToken(item.GetCurrentTier()), tierToken(item.GetNextTier()), item.GetSingleChange()))
		}
	case "desktop":
		results = append(results, fleet.GetDesktopBundling().GetReason())
	default:
		results = append(results,
			fmt.Sprintf("blocked_by_os=%d", len(fleet.GetBlockedByOs())),
			fmt.Sprintf("docker_blocked=%d", len(fleet.GetDockerBlocked())),
			fmt.Sprintf("peerless=%d", len(fleet.GetPeerless())),
			fmt.Sprintf("tier_upgrades=%d", len(fleet.GetTierUpgrades())),
			"desktop_bundling: "+fleet.GetDesktopBundling().GetReason(),
		)
	}
	return cliapp.ListReport{
		Summary:        []string{"Manifest root: " + fleet.GetManifestRoot()},
		ResultsHeading: "Fleet",
		Results:        results,
		RetrievalHints: []string{"`portability fleet --view blocked`", "`portability fleet --view upgrades`"},
	}
}

func formatEntry(entry *portabilityv1.CapabilityEntry) []string {
	out := make([]string, 0, len(entry.GetPlatforms())+1)
	out = append(out, fmt.Sprintf("%s  [%s] %s", entry.GetCapability(), situationToken(entry.GetSituation()), entry.GetSituationReason()))
	for _, platform := range entry.GetPlatforms() {
		out = append(out, fmt.Sprintf("  %-8s %-14s %-15s %s",
			osToken(platform.GetHostOs()),
			statusToken(platform.GetStatus()),
			qualificationToken(platform.GetQualification()),
			firstNonEmpty(platform.GetImplementer(), platform.GetMechanism(), platform.GetReason())))
	}
	return out
}

func formatBlocks(blocks []*portabilityv1.ScenarioBlock) []string {
	out := make([]string, 0, len(blocks))
	for _, block := range blocks {
		names := make([]string, 0, len(block.GetDependencies()))
		for _, dependency := range block.GetDependencies() {
			names = append(names, fmt.Sprintf("%s(%s)", dependency.GetName(), verdictToken(dependency.GetVerdict())))
		}
		out = append(out, fmt.Sprintf("%-28s %-8s %s", block.GetScenario(), osToken(block.GetHostOs()), strings.Join(names, ", ")))
	}
	return out
}

// splitList accepts a comma-separated flag value so one flag can carry a
// repeated proto field without inventing a second syntax.
func splitList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func situationFlag(raw string) (portabilityv1.CapabilitySituation, error) {
	name := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(raw), "-", "_"))
	if name == "" {
		return portabilityv1.CapabilitySituation_CAPABILITY_SITUATION_UNSPECIFIED, nil
	}
	value, ok := portabilityv1.CapabilitySituation_value["CAPABILITY_SITUATION_"+name]
	if !ok || value == 0 {
		return 0, fmt.Errorf("unknown situation %q", raw)
	}
	return portabilityv1.CapabilitySituation(value), nil
}

func enumToken(full, prefix string) string {
	return strings.ToLower(strings.TrimPrefix(full, prefix))
}

func osToken(value portabilityv1.HostOS) string {
	return enumToken(value.String(), "HOST_OS_")
}

func statusToken(value portabilityv1.ResolutionStatus) string {
	return enumToken(value.String(), "RESOLUTION_STATUS_")
}

func qualificationToken(value portabilityv1.Qualification) string {
	return enumToken(value.String(), "QUALIFICATION_")
}

func situationToken(value portabilityv1.CapabilitySituation) string {
	return enumToken(value.String(), "CAPABILITY_SITUATION_")
}

func verdictToken(value portabilityv1.Verdict) string {
	return enumToken(value.String(), "VERDICT_")
}

func tierToken(value portabilityv1.DeliveryTier) string {
	return enumToken(value.String(), "DELIVERY_TIER_")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
