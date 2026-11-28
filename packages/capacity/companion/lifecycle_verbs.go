package companion

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/envkit-go"

	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
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
	// DeclaredSteps, used instead of Steps, resolves the rungs when a capacity
	// verb runs. A resource whose ladder is its resource.json claim profile
	// (see ProfileSteps) keeps that file the one source without reading it on
	// every unrelated CLI invocation.
	DeclaredSteps func() ([]LifecycleStep, error)
	Stdout        io.Writer
	// Exec runs the control-plane command. nil shells the on-PATH vrooli.
	Exec func(ctx context.Context, env []string, name string, args ...string) error
	// Apply optionally handles a step after validation. It is used when a
	// resource has a small resource-specific guard or pin operation before the
	// shared lifecycle action.
	Apply StepHandler
}

// LifecycleCapacityCommands builds the `capacity` subcommand group for a
// resource whose profile rungs are lifecycle actions.
func LifecycleCapacityCommands(cfg LifecycleVerbsConfig) cliapp.SubcommandGroup {
	usage := fmt.Sprintf("resource-%s capacity degrade --to <declared rung>", cfg.Resource)
	if cfg.DeclaredSteps == nil {
		usage = fmt.Sprintf("resource-%s capacity degrade --to <%s>", cfg.Resource, strings.Join(stepLabels(cfg.Steps), "|"))
	}
	run := func(subcommand string) func([]string) error {
		return func(args []string) error {
			steps := cfg.Steps
			if cfg.DeclaredSteps != nil {
				declared, err := cfg.DeclaredSteps()
				if err != nil {
					return fmt.Errorf("resolve %s capacity rungs: %w", cfg.Resource, err)
				}
				steps = declared
			}
			return cfg.verbs(steps).Run(append([]string{subcommand}, args...))
		}
	}
	return cliapp.SubcommandGroup{
		Name:        "capacity",
		Description: "Respond to the capacity broker",
		Subcommands: []cliapp.Command{
			{
				Name:        "degrade",
				Description: "Move to a declared capacity rung at the broker's request",
				Usage:       usage,
				Run:         run("degrade"),
			},
			{
				Name:        "upshift",
				Description: "Return to a larger capacity rung when headroom returns",
				Usage:       strings.Replace(usage, "degrade", "upshift", 1),
				Run:         run("upshift"),
			},
		},
	}
}

func stepLabels(steps []LifecycleStep) []string {
	labels := make([]string, 0, len(steps))
	for _, step := range steps {
		labels = append(labels, step.Label)
	}
	return labels
}

// verbs binds the shared degrade/upshift dispatcher to one resolved ladder.
func (cfg LifecycleVerbsConfig) verbs(steps []LifecycleStep) Verbs {
	labels := stepLabels(steps)
	byLabel := make(map[string]LifecycleStep, len(steps))
	for _, step := range steps {
		byLabel[step.Label] = step
	}
	applyStep := func(ctx context.Context, label string) error {
		if cfg.Apply != nil {
			if err := cfg.Apply(ctx, label); err != nil {
				return err
			}
		}
		step := byLabel[label]
		run := cfg.Exec
		if run == nil {
			run = runLifecycleAction
		}
		return run(ctx, step.Env, "vrooli", "resource", step.Action, cfg.Resource)
	}
	preview := func(_ context.Context, label string) error {
		step, ok := byLabel[label]
		if !ok {
			return fmt.Errorf("%w %q (declared steps: %s)", ErrUnknownStep, label, strings.Join(labels, ", "))
		}
		out := cfg.Stdout
		if out == nil {
			out = os.Stdout
		}
		fmt.Fprintf(out, "%s capacity dry-run:\n  target rung: %s\n  control-plane action: vrooli resource %s %s\n  environment: %s\n  effect: %s\n", cfg.Resource, label, step.Action, cfg.Resource, strings.Join(step.Env, ", "), step.Description)
		return nil
	}
	return Verbs{Resource: cfg.Resource, Degrade: StepsFromLabels(labels, applyStep), Preview: preview, Stdout: cfg.Stdout}
}

// DeclaredStep is one rung of a resource's resource.json claim profile.
type DeclaredStep = internalcapacity.DegradeStep

// ProfileSteps returns a DeclaredSteps source that reads the resource's own
// acceleration.claim.profile.steps (the same declaration the broker degrades
// against) and maps each rung to its lifecycle action. The manifest loader
// keeps those steps the checked materialization of any steps_source policy.
func ProfileSteps(getenv func(string) string, resource string, action func(DeclaredStep) LifecycleStep) func() ([]LifecycleStep, error) {
	return func() ([]LifecycleStep, error) {
		profile, err := DeclaredProfile(getenv, resource)
		if err != nil {
			return nil, err
		}
		steps := make([]LifecycleStep, 0, len(profile.Steps))
		for _, declared := range profile.Steps {
			step := action(declared)
			step.Label = declared.Label
			steps = append(steps, step)
		}
		return steps, nil
	}
}

// DeclaredProfile reads the resource's declared claim profile.
func DeclaredProfile(getenv func(string) string, resource string) (internalcapacity.DegradeProfile, error) {
	spec, ok, err := DeclaredClaim(getenv)(resource)
	if err != nil {
		return internalcapacity.DegradeProfile{}, err
	}
	if !ok || spec.Profile == nil || len(spec.Profile.Steps) == 0 {
		return internalcapacity.DegradeProfile{}, fmt.Errorf("resource %s declares no acceleration.claim.profile.steps", resource)
	}
	return *spec.Profile, nil
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
