package companion

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/envkit-go"
)

// Some resources degrade by changing how they run rather than by loading a
// smaller model: kyutai-stt releases its VRAM by stopping, and kokoro and
// speaker-verification release theirs by running on the CPU instead of the
// device. Both are real operating modes, and both were previously expressed by
// giving the broker a different verb per resource — `stop` with no arguments,
// or nothing at all.
//
// LifecycleVerbs gives those resources the same `capacity degrade --to <label>`
// contract as everything else, implemented through the control plane's own
// lifecycle commands rather than a private one.

// LifecycleStep maps one declared profile rung onto a control-plane action.
type LifecycleStep struct {
	// Label is the rung's declared name, matching the manifest profile.
	Label string
	// Action is the `vrooli resource ...` action this rung performs.
	Action string
	// Env is applied to the action, which is how a resource switches device:
	// VROOLI_GPU=off restarts it without its accelerator overlay.
	Env []string
	// Description explains the rung to an operator.
	Description string
}

// LifecycleVerbsConfig declares a resource whose rungs are lifecycle actions.
type LifecycleVerbsConfig struct {
	Resource string
	Steps    []LifecycleStep
	// Exec runs the control-plane command. nil shells the on-PATH vrooli.
	Exec func(ctx context.Context, env []string, name string, args ...string) error
}

// LifecycleCapacityCommands builds the `capacity` subcommand group for a
// resource whose profile rungs are lifecycle actions.
func LifecycleCapacityCommands(cfg LifecycleVerbsConfig) cliapp.SubcommandGroup {
	labels := make([]string, 0, len(cfg.Steps))
	byLabel := make(map[string]LifecycleStep, len(cfg.Steps))
	for _, step := range cfg.Steps {
		labels = append(labels, step.Label)
		byLabel[step.Label] = step
	}

	apply := StepsFromLabels(labels, func(ctx context.Context, label string) error {
		step := byLabel[label]
		run := cfg.Exec
		if run == nil {
			run = runLifecycleAction
		}
		return run(ctx, step.Env, "vrooli", "resource", step.Action, cfg.Resource)
	})

	verbs := Verbs{Resource: cfg.Resource, Degrade: apply}
	usage := fmt.Sprintf("resource-%s capacity degrade --to <%s>", cfg.Resource, strings.Join(labels, "|"))
	return cliapp.SubcommandGroup{
		Name:        "capacity",
		Description: "Respond to the capacity broker",
		Subcommands: []cliapp.Command{
			{
				Name:        "degrade",
				Description: "Move to a declared capacity rung at the broker's request",
				Usage:       usage,
				Run:         func(args []string) error { return verbs.Run(append([]string{"degrade"}, args...)) },
			},
			{
				Name:        "upshift",
				Description: "Return to a larger capacity rung when headroom returns",
				Usage:       strings.Replace(usage, "degrade", "upshift", 1),
				Run:         func(args []string) error { return verbs.Run(append([]string{"upshift"}, args...)) },
			},
		},
	}
}

// runLifecycleAction shells the control plane. Lifecycle is the control plane's
// job, so the resource asks for it rather than implementing it.
func runLifecycleAction(ctx context.Context, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env(env))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// DeviceSteps is the two-rung ladder shared by every resource whose only
// degradation is "run on the CPU instead": hold the device, or do not.
func DeviceSteps(acceleratedLabel string) []LifecycleStep {
	return []LifecycleStep{
		{
			Label:       acceleratedLabel,
			Action:      "restart",
			Env:         []string{"VROOLI_GPU=on"},
			Description: "Run on the accelerator, holding its declared VRAM",
		},
		{
			Label:       "cpu",
			Action:      "restart",
			Env:         []string{"VROOLI_GPU=off"},
			Description: "Run on the CPU, holding no device memory at all",
		},
	}
}
