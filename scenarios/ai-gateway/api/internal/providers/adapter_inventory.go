package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func (a Adapter) ListRoles(ctx context.Context) (Inventory, error) {
	if a.Door != nil {
		return Inventory{Roles: []Role{{Provider: ProviderMetered, Role: "metered", Locality: "remote", Status: "available"}}}, nil
	}
	if a.HTTP != nil {
		return a.HTTP.ListRoles(), nil
	}
	if a.Provider == ProviderMetered {
		return a.listMeteredRoles()
	}
	return a.listPolicyRoles(ctx)
}

func (a Adapter) listMeteredRoles() (Inventory, error) {
	if a.Metered == nil {
		return Inventory{}, &CommandError{Code: "unavailable", Command: a.Provider, ExitCode: -1, Err: errors.New("metered inference client is not configured")}
	}
	return Inventory{Roles: []Role{{Provider: a.Provider, Role: "metered", Locality: "remote", Status: "available"}}}, nil
}

func (a Adapter) listPolicyRoles(ctx context.Context) (Inventory, error) {
	result, err := a.runPolicyRoles(ctx)
	if err != nil {
		return Inventory{}, mapCommandError(a.Provider, err)
	}
	var report rolePolicyReport
	if err := json.Unmarshal([]byte(result.Stdout), &report); err != nil {
		return Inventory{}, &CommandError{Code: "malformed_json", Command: a.command().String(), ExitCode: result.ExitCode, Stderr: result.Stderr, Err: err}
	}
	inventory := Inventory{Roles: make([]Role, 0, len(report.Roles))}
	var currentDigests map[string]string
	if a.Provider == ProviderOllama {
		currentDigests, err = a.ollamaModelDigests(ctx)
		if err != nil {
			currentDigests = map[string]string{}
			inventory.Warnings = append(inventory.Warnings, "Ollama runtime artifact inventory unavailable; local rights remain unknown")
		}
	}
	for _, entry := range report.Roles {
		role, warning := a.inventoryRole(entry, currentDigests)
		if warning != "" {
			inventory.Warnings = append(inventory.Warnings, warning)
		}
		if role != nil {
			inventory.Roles = append(inventory.Roles, *role)
		}
	}
	sort.Slice(inventory.Roles, func(i, j int) bool { return inventory.Roles[i].Role < inventory.Roles[j].Role })
	return inventory, nil
}

func (a Adapter) inventoryRole(entry rolePolicyEntry, currentDigests map[string]string) (*Role, string) {
	roleName := strings.TrimSpace(entry.Role)
	if roleName == "" {
		return nil, fmt.Sprintf("%s policy returned a role without a name", a.Provider)
	}
	capabilities := entry.RequiredCapabilities
	if len(capabilities) == 0 {
		capabilities = entry.Capabilities
	}
	warning := ""
	if len(capabilities) == 0 {
		warning = fmt.Sprintf("%s role %q has no declared capabilities", a.Provider, roleName)
	}
	model := strings.TrimSpace(entry.Model)
	return &Role{
		Provider: a.Provider, Role: roleName, Model: model,
		Limits: entry.Limits, Placement: entry.Placement,
		Rights: entry.Rights, RuntimeIdentity: entry.RuntimeIdentity,
		ObservedRuntimeDigest: currentDigests[model],
		Capabilities:          sortedUnique(capabilities), InputModalities: parseModalities(entry.Modalities.Input),
		Locality: firstNonEmpty(entry.Locality, a.Locality), Status: firstNonEmpty(entry.Status, "available"),
		PolicySchemaVersion: strings.TrimSpace(entry.SchemaVersion), PolicyFileSHA256: strings.TrimSpace(entry.PolicySHA256),
	}, warning
}

func (a Adapter) Smoke(ctx context.Context) SmokeResult {
	inventory, err := a.ListRoles(ctx)
	if err != nil {
		var cmdErr *CommandError
		if errors.As(err, &cmdErr) {
			return SmokeResult{Provider: a.Provider, Status: "unavailable", Code: cmdErr.Code, Message: cmdErr.Error(), ExitCode: cmdErr.ExitCode}
		}
		return SmokeResult{Provider: a.Provider, Status: "unavailable", Code: "unknown", Message: err.Error(), ExitCode: -1}
	}
	warnings := append([]string{}, inventory.Warnings...)
	if len(inventory.Roles) == 0 {
		warnings = append(warnings, "provider policy returned zero roles")
		return SmokeResult{Provider: a.Provider, Status: "degraded", Code: "empty_inventory", Message: "provider command responded but returned no roles", Warnings: warnings}
	}
	return SmokeResult{Provider: a.Provider, Status: "available", Code: "ok", Message: fmt.Sprintf("provider policy returned %d role(s)", len(inventory.Roles)), Warnings: warnings}
}

func (a Adapter) runPolicyRoles(ctx context.Context) (Result, error) {
	return a.runner().Run(ctx, a.command())
}

func (a Adapter) ollamaModelDigests(ctx context.Context) (map[string]string, error) {
	result, err := a.runner().Run(ctx, Command{Name: a.CommandName, Args: []string{"models", "inventory", "--json"}, Timeout: DefaultCommandTimeout, Fact: true})
	if err != nil {
		return nil, mapCommandError(a.Provider, err)
	}
	var report struct {
		Models []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"models"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &report); err != nil {
		return nil, &CommandError{Code: "malformed_inventory", Command: a.command().String(), ExitCode: result.ExitCode, Stderr: result.Stderr, Err: err}
	}
	digests := make(map[string]string, len(report.Models))
	for _, model := range report.Models {
		name := strings.TrimSpace(model.Name)
		if name == "" {
			continue
		}
		if _, duplicate := digests[name]; duplicate {
			return nil, &CommandError{Code: "malformed_inventory", Command: a.command().String(), ExitCode: result.ExitCode, Err: fmt.Errorf("duplicate model identity %q", name)}
		}
		digests[name] = strings.TrimSpace(model.Digest)
	}
	return digests, nil
}

func parseModalities(values []string) []sharedv1.Modality {
	out := make([]sharedv1.Modality, 0, len(values))
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "text":
			out = append(out, sharedv1.Modality_MODALITY_TEXT)
		case "image":
			out = append(out, sharedv1.Modality_MODALITY_IMAGE)
		case "vector":
			out = append(out, sharedv1.Modality_MODALITY_VECTOR)
		case "video":
			out = append(out, sharedv1.Modality_MODALITY_VIDEO)
		case "audio":
			out = append(out, sharedv1.Modality_MODALITY_AUDIO)
		}
	}
	return out
}
