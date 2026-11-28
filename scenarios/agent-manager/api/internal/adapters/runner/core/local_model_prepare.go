package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"agent-manager/internal/domain"
)

type localPreparationReceipt struct {
	Status        string `json:"status"`
	Runtime       string `json:"runtime"`
	Role          string `json:"role"`
	SelectedModel string `json:"selected_model"`
}

// prepareManagedLocalOpenCode joins Agent Manager to OpenCode's existing
// preparation transaction. It only runs for an explicitly selected managed
// llama.cpp model; model permissions and tool policy remain owned by Agent
// Manager and are not rewritten here.
func prepareManagedLocalOpenCode(ctx context.Context, cfg *domain.RunConfig, run func(context.Context, string, ...string) ([]byte, error)) error {
	if cfg == nil || cfg.RunnerType != domain.RunnerTypeOpenCode || !strings.HasPrefix(strings.TrimSpace(cfg.Model), "llama-cpp/") {
		return nil
	}
	if cfg.PolicySnapshot == nil || cfg.PolicySnapshot.SelectedCandidate.ResourceRole != "code.local" {
		return fmt.Errorf("managed llama.cpp requires the selected code.local resource role")
	}
	if strings.TrimSpace(cfg.PolicySnapshot.SelectedCandidate.Model) != strings.TrimSpace(cfg.Model) {
		return fmt.Errorf("Agent Manager's selected local candidate does not match its effective OpenCode model")
	}
	if run == nil {
		return fmt.Errorf("managed llama.cpp preparation command is unavailable")
	}
	if _, ok := ctx.Deadline(); !ok {
		return fmt.Errorf("managed llama.cpp preparation requires a bounded execution context")
	}
	model := strings.TrimPrefix(strings.TrimSpace(cfg.Model), "llama-cpp/")
	if model == "" {
		return fmt.Errorf("managed llama.cpp model identity is empty")
	}
	raw, err := run(ctx, "resource-opencode", "local-model", "select", "--runtime", "llama-cpp", "--role", "code.local", "--json")
	if err != nil {
		return fmt.Errorf("prepare managed local model for Agent Manager OpenCode run: %w", err)
	}
	var receipt localPreparationReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return fmt.Errorf("decode managed local model preparation receipt: %w", err)
	}
	if receipt.Status != "ready" || receipt.Runtime != "llama-cpp" || receipt.Role != "code.local" || receipt.SelectedModel != "llama-cpp/"+model {
		return fmt.Errorf("managed local model preparation selected an unexpected identity")
	}
	return nil
}

func runManagedLocalPreparation(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
