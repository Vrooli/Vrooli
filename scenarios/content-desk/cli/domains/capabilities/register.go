// Package capabilities exposes the marketing-capability catalog for operators
// and agents: what capabilities exist, their separate readiness dimensions,
// qualification state, priority, and next action.
package capabilities

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/capabilities"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/capabilities/capabilities_v1connect"
)

// GroupName is the manifest group this package owns.
const GroupName = "capabilities"

type handlers struct {
	client capabilitiesconnect.CapabilitiesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: capabilitiesconnect.NewCapabilitiesServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*capabilitiesv1.ListCapabilitiesResponse, error) {
	response, err := h.client.ListCapabilities(context.Background(), connect.NewRequest(&capabilitiesv1.ListCapabilitiesRequest{Medium: ctx.Flag("medium")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list capabilities", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no capabilities response")
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, message *capabilitiesv1.ListCapabilitiesResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.Capabilities))
	for _, capability := range message.Capabilities {
		results = append(results, formatCapability(capability))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d capability(ies).", len(message.Capabilities))},
		ResultsHeading: "Capabilities",
		Results:        results,
		RetrievalHints: []string{
			"`capabilities get <id>` — show one capability by id",
			"`capabilities get --alias <alias>` — show one capability by alias",
		},
	}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*capabilitiesv1.GetCapabilityResponse, error) {
	id := ctx.Positional("id")
	alias := ctx.Flag("alias")
	if id == "" && alias == "" {
		return nil, fmt.Errorf("provide a capability id or --alias")
	}
	response, err := h.client.GetCapability(context.Background(), connect.NewRequest(&capabilitiesv1.GetCapabilityRequest{Id: id, Alias: alias}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get capability", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.Capability == nil {
		return nil, fmt.Errorf("server returned no capability")
	}
	return response.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, message *capabilitiesv1.GetCapabilityResponse) cliapp.ListReport {
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Capability %s.", message.Capability.Id)},
		ResultsHeading: "Capability",
		Results:        []string{formatCapability(message.Capability)},
	}
}

func formatCapability(capability *capabilitiesv1.Capability) string {
	if capability == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s [medium=%s priority=%d scope=%s owner=%s] definition=%s implementation=%s readiness=%s readiness_source=%s quality=%s quality_source=%s connectivity=%s connectivity_source=%s %s%s next_action=%s%s",
		capability.Id, capability.Name, capability.Medium, capability.Priority, capability.PriorityScope, capability.Owner,
		capability.DefinitionStatus, capability.ImplementationStatus, capability.OperationalReadiness, capability.OperationalReadinessSource,
		capability.OutputQuality, capability.OutputQualitySource, capability.DistributionConnectivity, capability.DistributionConnectivitySource,
		formatQualification(capability.LatestQualification),
		formatLinkage(capability), capability.NextAction, formatReadinessLimitations(capability.ReadinessLimitations))
}

// formatLinkage renders the capability's linked prerequisites, channels,
// applicability, producing operation and provenance. These are the same
// source-bound relationships the API stores; without them a CLI reader sees
// read-only readiness dimensions but not what a capability requires, serves or
// depends on. Each field is omitted when empty rather than fabricated as zero.
func formatLinkage(capability *capabilitiesv1.Capability) string {
	var builder strings.Builder
	builder.WriteString(formatList("aliases", capability.Aliases))
	builder.WriteString(formatList("prerequisites", capability.Prerequisites))
	builder.WriteString(formatList("channels", capability.Channels))
	builder.WriteString(formatField("audience", capability.AudienceApplicability))
	builder.WriteString(formatField("delivery", capability.DeliveryApplicability))
	builder.WriteString(formatField("producer", capability.ProducingOperation))
	builder.WriteString(formatField("priority_reason", capability.PriorityReason))
	builder.WriteString(formatList("sources", capability.SourceRefs))
	return builder.String()
}

func formatList(label string, values []string) string {
	if len(values) == 0 {
		return ""
	}
	return fmt.Sprintf(" %s=[%s]", label, strings.Join(values, "; "))
}

func formatField(label, value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf(" %s=%s", label, value)
}

func formatReadinessLimitations(limitations []string) string {
	if len(limitations) == 0 {
		return ""
	}
	return fmt.Sprintf(" readiness_limitations=[%s]", strings.Join(limitations, "; "))
}

func formatQualification(qualification *capabilitiesv1.CapabilityQualification) string {
	if qualification == nil || qualification.ObservedAt == "" || qualification.ObservedAt == "unknown" {
		return "qualification=unknown"
	}
	return fmt.Sprintf("qualification=observed(env=%s observed_at=%s basis=%s limitation=%s)",
		qualification.Environment, qualification.ObservedAt, qualification.FreshnessBasis, qualification.Limitation)
}

// Register builds the capabilities subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"CapabilitiesService.ListCapabilities": cliapp.ProtoList(h.listCall, h.listReport),
		"CapabilitiesService.GetCapability":    cliapp.ProtoList(h.getCall, h.getReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("capabilities: load from manifest: %w", err)
	}
	return group, nil
}
