package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/repo-contract-go"
	hostconfig "github.com/vrooli/vrooli/internal/config"
	resourceowner "github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/capacitysync"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/config"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/gateway"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/models"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/optimize"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policycmd"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/runtimeprofile"

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
// fails ensure with a remediation hint. Infrastructure failures such as a
// missing policy or unavailable daemon remain non-blocking; uncertain rights
// and an unverified local endpoint are policy failures and stay closed.
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
		authorize := func(role, model, digest string) error {
			return policy.AuthorizeLocalInferenceArtifact(os.Getenv, role, model, digest)
		}
		daemon, err := ensure.NewLoopbackClient()
		if err != nil {
			return fmt.Errorf("verify local Ollama endpoint for model admission: %w", err)
		}
		res, err := models.Doctor(ctx, daemon, p, models.DoctorOptions{Roles: toolRoles}, authorize)
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
	// This is the host-side `upstream` port declared in resource.json. Capacity
	// actuation is the only owner path that calls it directly, after draining
	// the public activity edge; the client still enforces loopback-only transport.
	ollamaCapacityUpstreamURL = "http://127.0.0.1:21434"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() {
	if activityNetworkRequiredForOllamaCommand(os.Args[1:]) {
		self, err := os.Executable()
		entered := false
		if err == nil {
			entered, err = resourceowner.RunActivityNetworkOwnerCommand(appName, append([]string{self}, os.Args[1:]...))
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: enter private Ollama owner network: %v\n", err)
			os.Exit(1)
		}
		if entered {
			return
		}
	}
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

func activityNetworkRequiredForOllamaCommand(args []string) bool {
	if strings.TrimSpace(os.Getenv("VROOLI_ACTIVITY_NETWORK_JOINED")) == "1" || len(args) == 0 {
		return false
	}
	switch args[0] {
	case "ensure", "capacity":
		return true
	case "models":
		// evict reaches the private daemon; reacquire runs ensure.
		return len(args) > 1 && (args[1] == "evict" || args[1] == "reacquire")
	default:
		return false
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
	edge.Wake = runOllamaWake
	edgeLog := func(format string, args ...any) { fmt.Fprintf(os.Stderr, format, args...) }
	normalizer := runtimeprofile.NewEdgeNormalizer(edgeLog)
	if path, err := policy.DefaultPolicyPath(os.Getenv); err == nil {
		normalizer.KeepAlive = policy.NewLive(path, edgeLog).KeepAliveSeconds
	} else {
		edgeLog("ollama model policy unavailable to the activity edge; keep_alive stays at the daemon default: %v\n", err)
	}
	edge.Rewrite = normalizer.Rewrite
	normalizer.Authorize = func(ctx context.Context, model string) error {
		return authorizeOllamaWithWake(ctx, model,
			func(ctx context.Context, model string) (string, error) {
				daemon, err := ensure.NewLoopbackClient()
				if err != nil {
					return "", fmt.Errorf("verify local Ollama endpoint for inference authorization: %w", err)
				}
				return models.CurrentModelDigest(ctx, daemon, model)
			},
			func(model string) (string, error) {
				modelRoot, err := ensure.EffectiveModelRoot()
				if err != nil {
					return "", fmt.Errorf("resolve managed Ollama model store before wake: %w", err)
				}
				return ensure.InstalledManifestDigest(modelRoot, model)
			},
			func(model, digest string) error {
				return policy.AuthorizeLocalInferenceArtifact(os.Getenv, "", model, digest)
			},
			runOllamaWake)
	}
	edge.StartupReconcile = normalizer.ReconcileLoadedResidency
	capacityCommands := companion.LifecycleCapacityCommands(companion.LifecycleVerbsConfig{
		Resource:      appName,
		DeclaredSteps: companion.ProfileSteps(os.Getenv, appName, ollamaCapacityStep),
		Apply:         applyOllamaCapacityRungManaged,
	})
	capacityCommands.Subcommands = append(capacityCommands.Subcommands, capacitysync.ReservationCommands(nil)...)
	app.SetCommandsWithSubgroups(
		append(app.StandardLifecycleCommands(), cliapp.CommandGroup{Title: "Health", Commands: []cliapp.Command{{Name: "health-ready", Description: "Succeed only when at least one model is installed", Run: healthReady}}}, ensure.CommandGroup(newAdmissionValidator()),
			cliapp.CommandGroup{Title: "Capacity", Commands: []cliapp.Command{capacitysync.Command(nil), activityedge.Command(edge), activityedge.BridgeCommand()}}),
		[]cliapp.SubcommandGroup{gateway.Commands(nil), capacityCommands, policycmd.Commands(nil), models.Commands(nil), optimize.Commands(nil), runtimeprofile.Commands(nil)},
	)
	return app, nil
}

// authorizeOllamaWithWake performs the disk-backed artifact and rights checks
// before starting a stopped resource. It then repeats both checks against the
// live daemon inventory before inference can proceed.
func authorizeOllamaWithWake(
	ctx context.Context,
	model string,
	liveDigest func(context.Context, string) (string, error),
	diskDigest func(string) (string, error),
	authorize func(string, string) error,
	wake func(context.Context) error,
) error {
	digest, err := liveDigest(ctx, model)
	if err == nil {
		return authorize(model, digest)
	}
	probeErr := err
	digest, err = diskDigest(model)
	if err != nil {
		return fmt.Errorf("refuse to wake Ollama for an unverified installed model: %w", err)
	}
	if err := authorize(model, digest); err != nil {
		return fmt.Errorf("refuse to wake Ollama for an unauthorized model: %w", err)
	}
	if err := wake(ctx); err != nil {
		return fmt.Errorf("wake managed Ollama for inference authorization: %w (inventory probe: %v)", err, probeErr)
	}
	digest, err = liveDigest(ctx, model)
	if err != nil {
		return fmt.Errorf("read Ollama model inventory after managed wake: %w", err)
	}
	if err := authorize(model, digest); err != nil {
		return fmt.Errorf("authorize live Ollama model after managed wake: %w", err)
	}
	return nil
}

// applyOllamaCapacityRungManaged closes the managed request boundary and unloads
// models before LifecycleCapacityCommands asks the control plane to restart
// Ollama. The stable activity edge stays closed until that restart verifies
// health and placement, then resumes. A failed restart preserves the broker
// claim and a later request recovers through the edge's managed Wake owner.
func applyOllamaCapacityRungManaged(ctx context.Context, label string) error {
	client, err := newOllamaCapacityClient()
	if err != nil {
		return err
	}
	return applyOllamaCapacityRungWithDrain(ctx, label, ollamaCapacityTarget, client, controlOllamaActivityEdge)
}

// applyOllamaCapacityRungWithDrain closes the managed request boundary before
// unloading any resident model. A failed callback can resume only while its
// lifecycle context is still live; canceled or uncertain effects remain gated.
func applyOllamaCapacityRungWithDrain(ctx context.Context, label string, target func(string) (int64, error), client ollamaCapacityClient, control func(context.Context, string) error) error {
	if control == nil {
		return fmt.Errorf("Ollama capacity change requires the managed activity-edge control owner")
	}
	if err := control(ctx, "drain"); err != nil {
		return fmt.Errorf("drain Ollama activity edge before model unload: %w", err)
	}
	if err := applyOllamaCapacityRungWith(ctx, label, target, client); err != nil {
		if ctx.Err() != nil {
			return errors.Join(err, fmt.Errorf("Ollama activity edge remains drained after canceled unload"))
		}
		resumeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if resumeErr := control(resumeCtx, "resume"); resumeErr != nil {
			return errors.Join(err, fmt.Errorf("resume Ollama activity edge after failed unload: %w", resumeErr))
		}
		return err
	}
	return nil
}

func controlOllamaActivityEdge(parent context.Context, action string) error {
	home, err := hostconfig.HomeDir()
	if err != nil {
		return fmt.Errorf("resolve runtime home for Ollama activity edge: %w", err)
	}
	processes, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyProcesses)
	if err != nil {
		return fmt.Errorf("resolve runtime processes directory for Ollama activity edge: %w", err)
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	socket := filepath.Join(processes, "resources", appName, "activity-edge.control.sock")
	return activityedge.ControlSocketRequest(ctx, socket, action)
}

type ollamaCommandRunner func(context.Context, string, ...string) error

func runOllamaWake(ctx context.Context) error {
	return runOllamaWakeWith(ctx, runOllamaCommand)
}

func runOllamaWakeWith(ctx context.Context, run ollamaCommandRunner) error {
	if run == nil {
		return fmt.Errorf("Ollama wake requires the managed resource command owner")
	}
	return run(ctx, "vrooli", "resource", "start", appName)
}

func runOllamaCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type ollamaCapacityClient interface {
	ListRunning(context.Context) ([]ensure.RunningModel, error)
	Unload(context.Context, string) error
}

func newOllamaCapacityClient() (ollamaCapacityClient, error) {
	client, err := ensure.NewManagedDaemonLoopbackClient(ollamaCapacityUpstreamURL)
	if err != nil {
		return nil, fmt.Errorf("resolve managed Ollama upstream for capacity actuation: %w", err)
	}
	return client, nil
}

// ollamaCapacityStep maps a rung of resource.json's claim profile (the
// footprint-ordered materialization of model-policy.json) to its lifecycle
// action: a model rung keeps the GPU, the zero-byte cpu rung leaves it.
func ollamaCapacityStep(step companion.DeclaredStep) companion.LifecycleStep {
	if step.AmountBytes <= 0 {
		return companion.LifecycleStep{Action: "restart", Env: []string{"VROOLI_GPU=off"}, Description: "Unload every model and run on the CPU, holding no device memory"}
	}
	return companion.LifecycleStep{Action: "restart", Env: []string{"VROOLI_GPU=on"}, Description: fmt.Sprintf("Unload models until at most %d bytes stay resident on the GPU", step.AmountBytes)}
}

// ollamaCapacityTarget is the resident footprint a rung allows: the bytes the
// broker reserved for that rung in the declared claim profile.
func ollamaCapacityTarget(label string) (int64, error) {
	profile, err := companion.DeclaredProfile(os.Getenv, appName)
	if err != nil {
		return 0, err
	}
	return ollamaRungBytes(profile.Steps, label)
}

func ollamaRungBytes(steps []companion.DeclaredStep, label string) (int64, error) {
	for _, step := range steps {
		if step.Label == label {
			return step.AmountBytes, nil
		}
	}
	return 0, fmt.Errorf("capacity rung %q is not declared by the Ollama claim profile", label)
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
