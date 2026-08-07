package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/capacity"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/capacitysync"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/config"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/gateway"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/models"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policycmd"

	"github.com/vrooli/cli-core/cliapp"
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

func healthGPU(args []string) error {
	fs := flag.NewFlagSet("health-gpu", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "emit JSON")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	models, err := ensure.NewClient().ListRunning(ctx)
	if err != nil {
		return err
	}
	report := ensure.SummarizeProcessors(models)
	host, hostErr := hostinventory.Collect(ctx)
	hostGPU := hostErr == nil && host.HasNvidiaGPU()
	payload := map[string]any{
		"processor":            report.Processor,
		"models":               report.Models,
		"has_cpu_model":        report.HasCPUModel,
		"has_gpu_model":        report.HasGPUModel,
		"host_nvidia_gpu":      hostGPU,
		"host_inventory_error": "",
	}
	if hostErr != nil {
		payload["host_inventory_error"] = hostErr.Error()
	}
	if *jsonOutput {
		if err := json.NewEncoder(os.Stdout).Encode(payload); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(os.Stdout, "processor: %s (models: %d, host_nvidia_gpu: %t)\n", report.Processor, len(models), hostGPU)
	}
	if hostGPU && report.HasCPUModel {
		return fmt.Errorf("loaded Ollama model is executing on CPU while the host exposes an NVIDIA GPU")
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
	app.SetCommandsWithSubgroups(
		append(app.StandardLifecycleCommands(), cliapp.CommandGroup{Title: "Health", Commands: []cliapp.Command{{Name: "health-ready", Description: "Succeed only when at least one model is installed", Run: healthReady}, {Name: "health-gpu", Description: "Report the processor used by every loaded model", Run: healthGPU}}}, ensure.CommandGroup(newAdmissionValidator()),
			cliapp.CommandGroup{Title: "Capacity", Commands: []cliapp.Command{capacitysync.Command(nil)}}),
		[]cliapp.SubcommandGroup{gateway.Commands(nil), capacity.Commands(nil), policycmd.Commands(nil), models.Commands(nil)},
	)
	return app, nil
}
