package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/hosttoolpermissions"
)

// hostToolPermissionInventory projects the currently selected host-tool
// declarations. Opt-in state is handled by the owner-backed permission provider;
// installation remains with the host control plane.
func hostToolPermissionInventory(ctx context.Context) (string, []hosttoolpermissions.Tool, error) {
	tools, _, err := listHostRequirements(ctx, "local")
	if err != nil {
		return "", nil, err
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })
	result := make([]hosttoolpermissions.Tool, 0, len(tools))
	for _, tool := range tools {
		declaration, err := json.Marshal(struct {
			Name, Reason, Description, Privilege, Bundling string
			Required                                       bool
			Platforms                                      []string
			ConfigSchema                                   map[string]any
		}{tool.Name, tool.Reason, tool.Description, tool.Privilege, tool.Bundling, tool.Required, tool.Platforms, tool.ConfigSchema})
		if err != nil {
			return "", nil, err
		}
		schema := sha256.Sum256(declaration)
		result = append(result, hosttoolpermissions.Tool{
			Name: tool.Name, Scenario: "selected-closure", Required: tool.Required,
			Reason: tool.Reason, Platforms: append([]string(nil), tool.Platforms...),
			Privilege: tool.Privilege, SchemaDigest: "sha256:" + hex.EncodeToString(schema[:]),
		})
	}
	canonical, err := json.Marshal(result)
	if err != nil {
		return "", nil, err
	}
	revision := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(revision[:]), result, nil
}
