package main

import (
	"fmt"
	"os"

	"github.com/vrooli/vrooli/resources/pii-detector/cli/internal/capacitysync"
	"github.com/vrooli/vrooli/resources/pii-detector/cli/internal/gateway"
	"github.com/vrooli/vrooli/resources/pii-detector/cli/internal/models"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity/activityedge"
	"github.com/vrooli/vrooli/packages/capacity/companion"
)

const (
	appName    = "pii-detector"
	appVersion = "0.1.0"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

// capacitySteps is the pii-detector's degradation ladder. Unlike a GPU resource
// (gpu/cpu), this CPU model-server degrades between holding the NER model in RAM
// and serving the deterministic regex/checksum pass only. Both rungs restart via
// the control plane; the env toggle tells the server whether to load the model.
func capacitySteps() []companion.LifecycleStep {
	return []companion.LifecycleStep{
		{
			Label:       "model",
			Action:      "restart",
			Env:         []string{"PII_DETECTOR_LOAD_MODEL=on"},
			Description: "Hold the NER model in RAM (full detection)",
		},
		{
			Label:       "regex-only",
			Action:      "restart",
			Env:         []string{"PII_DETECTOR_LOAD_MODEL=off"},
			Description: "Release the NER model; serve the deterministic regex/checksum pass only",
		},
	}
}

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
		Description:         "PII detector resource CLI",
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
		append(app.StandardLifecycleCommands(), models.StatusCommand(nil), cliapp.CommandGroup{
			Title: "Capacity",
			Commands: []cliapp.Command{{
				Name:        "capacity-sync",
				Description: "Keep the pii-detector capacity claim alive",
				Usage:       "resource-pii-detector capacity-sync [--interval 15s] [--once]",
				Run:         capacitysync.Command(nil),
			}, activityedge.Command(edge)},
		}),
		[]cliapp.SubcommandGroup{gateway.Commands(nil), models.Commands(nil), models.PolicyCommands(nil), companion.LifecycleCapacityCommands(companion.LifecycleVerbsConfig{Resource: appName, Steps: capacitySteps()})},
	)
	return app, nil
}
