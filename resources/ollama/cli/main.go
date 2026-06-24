package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/capacity"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/capacitysync"
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
		append(app.StandardLifecycleCommands(), ensure.CommandGroup(newAdmissionValidator()),
			cliapp.CommandGroup{Title: "Capacity", Commands: []cliapp.Command{capacitysync.Command(nil)}}),
		[]cliapp.SubcommandGroup{gateway.Commands(nil), capacity.Commands(nil), policycmd.Commands(nil), models.Commands(nil)},
	)
	return app, nil
}
