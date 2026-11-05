package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/vrooli/packages/capabilityprobe"
	capsH "web-console/handlers/capabilities"
)

// Probes come from the same descriptors as everything else here. The table in
// capabilityprobe stays for the dependency-free agents that cannot read them,
// and its own test keeps it equal to the descriptors.
func init() {
	capabilityprobe.AITools = aiToolDefinitions(harness.Active())
}

func aiToolDefinitions(set harness.Set) []capabilityprobe.Definition {
	descriptors := set.All()
	definitions := make([]capabilityprobe.Definition, 0, len(descriptors))
	for _, descriptor := range descriptors {
		definitions = append(definitions, capabilityprobe.Definition{Capability: "ai-cli", ID: descriptor.ID, Label: descriptor.DisplayName, Command: descriptor.Binary, VersionArg: []string{"--version"}})
	}
	return definitions
}

// listHarnesses answers for local and remote targets alike: the resource set is
// defined by this repository, not by the machine. It says what a harness can do
// where it is installed, and never that it is installed on that machine.
func (s *Server) listHarnesses(_ context.Context, targetID string) ([]capsH.HarnessDescriptor, error) {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return nil, fmt.Errorf("target is required")
	}
	if _, ok := s.targetByID(targetID); !ok {
		return nil, fmt.Errorf("%w: %s", capsH.ErrTargetNotFound, targetID)
	}
	descriptors := harness.Active().All()
	result := make([]capsH.HarnessDescriptor, 0, len(descriptors))
	for _, d := range descriptors {
		item := capsH.HarnessDescriptor{
			AgentID: d.ID, Resource: d.Resource, DisplayName: d.DisplayName, Icon: d.Icon,
			SupportsContextLimit: d.Features.ContextLimit != nil, SupportsUsageMode: d.Features.UsageMode != nil,
			SupportsModelSelection: d.Features.ModelSelection != nil, SupportsProviderSelection: d.Features.ProviderSelection,
		}
		for _, window := range d.UsageWindows {
			item.UsageWindows = append(item.UsageWindows, capsH.HarnessUsageWindow{ID: window.ID, WindowMinutes: window.Minutes})
		}
		result = append(result, item)
	}
	return result, nil
}
