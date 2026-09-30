package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity/activityedge"
	"github.com/vrooli/vrooli/packages/capacity/companion"
)

const appName = "ace-step"

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
		Name: appName, Version: "0.1.0", Description: "ACE-Step 1.5 music generation resource",
		SourceRootEnvVars: env.SourceRootEnvVars, ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint: buildFingerprint, BuildTimestamp: buildTimestamp, BuildSourceRoot: buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	edge, err := activityedge.ForResource(appName)
	if err != nil {
		return nil, err
	}
	app.SetCommandsWithSubgroups(
		append(app.StandardLifecycleCommands(), cliapp.CommandGroup{Title: "Capacity", Commands: []cliapp.Command{activityedge.Command(edge)}}),
		[]cliapp.SubcommandGroup{companion.LifecycleCapacityCommands(companion.LifecycleVerbsConfig{Resource: appName, Steps: capacitySteps(), Apply: applyCapacityRung})},
	)
	return app, nil
}

func applyCapacityRung(ctx context.Context, label string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://127.0.0.1:8895/v1/capacity/degrade?to="+label, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("persist capacity rung %q: %w", label, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("persist capacity rung %q: server returned %s", label, resp.Status)
	}
	return nil
}

func capacitySteps() []companion.LifecycleStep {
	return []companion.LifecycleStep{
		{Label: "full", Action: "restart", Env: []string{"ACE_STEP_OFFLOAD_DIT=0"}, Description: "Keep the DiT resident on the GPU."},
		{Label: "offload-dit", Action: "restart", Env: []string{"ACE_STEP_OFFLOAD_DIT=1"}, Description: "Offload the DiT between diffusion steps."},
	}
}
