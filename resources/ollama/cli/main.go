package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/capacitysync"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/config"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/gateway"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/models"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policycmd"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity/activityedge"
	"github.com/vrooli/vrooli/packages/capacity/companion"
)

// admissionOptOutEnv lets an explicitly-authorized operator bypass the
// fail-closed model-admission gate (e.g. to seat a model whose tool template
// differs but is known-good). Off by default.
const admissionOptOutEnv = "OLLAMA_SKIP_MODEL_ADMISSION"

// newAdmissionValidator builds the fail-closed gate `ensure` runs after pulls.
// It validates only the TOOL-REQUIRING roles among those resolved this run via
// the models-doctor SSOT; a tool role whose seated model cannot tool-call
// fails ensure with a remediation hint. Infra failures (policy unreadable,
// daemon unreachable) degrade to a non-blocking pass — the gate refuses only on
// a definitive behavioral failure, never on uncertainty.
func newAdmissionValidator() ensure.AdmissionValidator {
	return func(ctx context.Context, resolvedRoles []string) error {
		if v := strings.TrimSpace(os.Getenv(admissionOptOutEnv)); v == "1" || strings.EqualFold(v, "true") {
			return nil
		}
		p, _, err := policy.LoadDefaultFile(os.Getenv)
		if err != nil {
			return nil // cannot validate → do not block
		}
		toolRoles := intersect(resolvedRoles, models.ToolRoles(p))
		if len(toolRoles) == 0 {
			return nil
		}
		res, err := models.Doctor(ctx, ensure.NewClient(), p, models.DoctorOptions{Roles: toolRoles})
		if err != nil {
			return nil // infra error → do not block
		}
		if res.Pass {
			return nil
		}
		var reasons []string
		for _, m := range res.Models {
			if !m.Pass {
				reasons = append(reasons, fmt.Sprintf("%s (%s): %s", m.Role, m.Model, strings.Join(m.Reasons, "; ")))
			}
		}
		return fmt.Errorf("a model failed tool-role validation and was not seated — %s. "+
			"Seat a tool-calling-capable model for the role, or set %s=1 to override",
			strings.Join(reasons, " | "), admissionOptOutEnv)
	}
}

func intersect(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, x := range b {
		set[x] = true
	}
	var out []string
	for _, x := range a {
		if set[x] {
			out = append(out, x)
		}
	}
	return out
}

const (
	appName    = "ollama"
	appVersion = "0.1.0"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() {
	app, err := newApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.CLI.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// healthReady is stricter than Ollama's process-level /api/tags response: a
// daemon with no installed model cannot serve the resource's primary
// capability and must remain unready.
func healthReady(_ []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cfg := config.Default()
	client := ensure.NewClient()
	client.BaseURL = cfg.BaseURL
	models, err := client.ListModels(ctx)
	if err != nil {
		return err
	}
	if cfg.RequireModel && len(models) == 0 {
		return fmt.Errorf("no Ollama model is installed")
	}
	return nil
}

func newApp() (*cliapp.ResourceApp, error) {
	env := cliapp.StandardResourceEnv(appName, cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name:                appName,
		Version:             appVersion,
		Description:         "Ollama resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	edge, err := activityedge.ForResource(appName)
	if err != nil {
		return nil, err
	}
	app.SetCommandsWithSubgroups(
		append(app.StandardLifecycleCommands(), cliapp.CommandGroup{Title: "Health", Commands: []cliapp.Command{{Name: "health-ready", Description: "Succeed only when at least one model is installed", Run: healthReady}}}, ensure.CommandGroup(newAdmissionValidator()),
			cliapp.CommandGroup{Title: "Capacity", Commands: []cliapp.Command{capacitysync.Command(nil), activityedge.Command(edge)}}),
		[]cliapp.SubcommandGroup{gateway.Commands(nil), companion.LifecycleCapacityCommands(companion.LifecycleVerbsConfig{
			Resource: appName,
			Steps: []companion.LifecycleStep{
				{Label: "gemma4:12b", Action: "restart", Env: []string{"VROOLI_GPU=on"}},
				{Label: "qwen3.5:9b", Action: "restart", Env: []string{"VROOLI_GPU=on"}},
				{Label: "qwen3.5:4b", Action: "restart", Env: []string{"VROOLI_GPU=on"}},
				{Label: "qwen3-vl:4b", Action: "restart", Env: []string{"VROOLI_GPU=on"}},
				{Label: "nomic-embed-text:latest", Action: "restart", Env: []string{"VROOLI_GPU=on"}},
				{Label: "cpu", Action: "restart", Env: []string{"VROOLI_GPU=off"}},
			},
			Apply: applyOllamaCapacityRung,
		}), policycmd.Commands(nil), models.Commands(nil)},
	)
	return app, nil
}

// applyOllamaCapacityRung preserves the Ollama-specific model residency step
// while the companion owns the shared capacity verbs and lifecycle restart.
func applyOllamaCapacityRung(ctx context.Context, label string) error {
	return applyOllamaCapacityRungWith(ctx, label, ollamaCapacityTarget, ensure.NewClient())
}

type ollamaCapacityClient interface {
	ListRunning(context.Context) ([]ensure.RunningModel, error)
	Unload(context.Context, string) error
}

func ollamaCapacityTarget(label string) (int64, error) {
	if label == "cpu" {
		return 0, nil
	}
	p, _, err := policy.LoadDefaultFile(os.Getenv)
	if err != nil {
		return 0, err
	}
	model, ok := p.Models[label]
	if !ok || model.VRAMGBEstimate <= 0 {
		return 0, fmt.Errorf("capacity rung %q is not declared by the Ollama model policy", label)
	}
	return int64(model.VRAMGBEstimate * 1024 * 1024 * 1024), nil
}

func applyOllamaCapacityRungWith(ctx context.Context, label string, target func(string) (int64, error), client ollamaCapacityClient) error {
	targetBytes, err := target(label)
	if err != nil {
		return err
	}
	running, err := client.ListRunning(ctx)
	if err != nil {
		return fmt.Errorf("list running Ollama models: %w", err)
	}
	sort.SliceStable(running, func(i, j int) bool { return running[i].SizeVRAM > running[j].SizeVRAM })
	var total int64
	for _, model := range running {
		total += model.SizeVRAM
	}
	for total > targetBytes && len(running) > 0 {
		model := running[0]
		if err := client.Unload(ctx, model.Name); err != nil {
			return fmt.Errorf("unload Ollama model %s: %w", model.Name, err)
		}
		total -= model.SizeVRAM
		running = running[1:]
	}
	return nil
}
