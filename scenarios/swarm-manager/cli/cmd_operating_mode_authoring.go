package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

// Self-serve operating-mode authoring: scaffold a mode folder from the built-in
// template, validate it from disk, and simulate its flow — all with zero Go
// edits and no rebuild (execute needs only a restart so the registry reloads the
// data). These commands speak the typed OperatingModeService Connect contract
// (the new authoring RPCs are Connect-only, not part of the legacy REST surface)
// via the generated client.

func (a *App) operatingModeClient() apiconnect.OperatingModeServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(a.core)
	return apiconnect.NewOperatingModeServiceClient(httpClient, baseURL)
}

func (a *App) cmdOperatingModeScaffold(args []string) error {
	fs := flag.NewFlagSet("operating-mode scaffold", flag.ContinueOnError)
	idFlag := fs.String("id", "", "New mode id (lowercase kebab-case, e.g. my-mode)")
	labelFlag := fs.String("label", "", "Display label (defaults from the id)")
	descFlag := fs.String("description", "", "One-line description of the methodology")
	startFrom := fs.String("start-from", "", "Clone an existing registered mode as the head start (reuse-first) instead of the blank template")
	force := fs.Bool("force", false, "Overwrite an existing mode folder")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	id := strings.TrimSpace(*idFlag)
	if id == "" {
		return fmt.Errorf("usage: operating-mode scaffold --id MODE [--start-from EXISTING] [--label LABEL] [--description TEXT] [--force] [--json]\n\n--id is required")
	}
	resp, err := a.operatingModeClient().ScaffoldMode(context.Background(), connect.NewRequest(&apipb.OperatingModeScaffoldRequest{
		Id:          id,
		Label:       strings.TrimSpace(*labelFlag),
		Description: strings.TrimSpace(*descFlag),
		StartFrom:   strings.TrimSpace(*startFrom),
		Force:       *force,
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	msg := resp.Msg
	printSection("Scaffolded Operating Mode")
	fmt.Printf("  Mode: %s\n", msg.GetMode())
	fmt.Printf("  Dir:  %s\n", msg.GetDir())
	for _, file := range msg.GetCreatedFiles() {
		fmt.Printf("    + %s\n", file)
	}
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Printf("    1. Shape %s/mode.json to the methodology and add an example-run per branch.\n", msg.GetDir())
	fmt.Printf("    2. operating-mode validate --mode %s   (reports any uncovered guarded/classified branch)\n", msg.GetMode())
	fmt.Printf("    3. operating-mode simulate --mode %s --preset <branch>   (walk each branch with the operator)\n", msg.GetMode())
	fmt.Println("    4. Restart swarm-manager (no rebuild) so the registry loads the new mode, then run it.")
	return nil
}

func (a *App) cmdOperatingModeValidate(args []string) error {
	fs := flag.NewFlagSet("operating-mode validate", flag.ContinueOnError)
	modeFlag := fs.String("mode", "", "Mode id to validate from disk")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	mode := strings.TrimSpace(*modeFlag)
	if mode == "" {
		return fmt.Errorf("usage: operating-mode validate --mode MODE [--json]\n\n--mode is required")
	}
	resp, err := a.operatingModeClient().ValidateMode(context.Background(), connect.NewRequest(&apipb.OperatingModeValidateRequest{Mode: mode}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	msg := resp.Msg
	printSection("Operating Mode Validation")
	if msg.GetOk() {
		fmt.Printf("  %s — VALID (%d phase(s), %d example-run(s))\n", msg.GetMode(), msg.GetPhaseCount(), msg.GetExampleRuns())
		if uncovered := msg.GetUncoveredBranches(); len(uncovered) > 0 {
			fmt.Printf("  Branch coverage: %d guarded/classified branch(es) not walked by any example-run — author one per branch before the simulation walkthrough:\n", len(uncovered))
			for _, branch := range uncovered {
				fmt.Printf("    - %s\n", branch)
			}
		} else {
			fmt.Println("  Branch coverage: every guarded/classified branch is walked by an example-run.")
		}
		return nil
	}
	fmt.Printf("  %s — INVALID (%s)\n", msg.GetMode(), msg.GetSummary())
	for _, e := range msg.GetErrors() {
		fmt.Printf("    - %s\n", e)
	}
	// A validation failure is a non-zero-exit signal for scripting.
	return fmt.Errorf("operating mode %q is not valid", msg.GetMode())
}

func (a *App) cmdOperatingModeSimulate(args []string) error {
	fs := flag.NewFlagSet("operating-mode simulate", flag.ContinueOnError)
	modeFlag := fs.String("mode", "", "Mode id to simulate")
	presetFlag := fs.String("preset", "", "Example-run preset id (defaults to happy-path)")
	registered := fs.Bool("registered", false, "Simulate the registered mode instead of the on-disk draft")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	mode := strings.TrimSpace(*modeFlag)
	if mode == "" {
		return fmt.Errorf("usage: operating-mode simulate --mode MODE [--preset ID] [--registered] [--json]\n\n--mode is required")
	}
	// Default to draft (on-disk) simulation: it reflects unsaved authoring edits
	// and is identical to the registered walk for a committed mode. --registered
	// forces the process registry's copy.
	resp, err := a.operatingModeClient().SimulateMode(context.Background(), connect.NewRequest(&apipb.OperatingModeSimulateRequest{
		Mode:   mode,
		Preset: strings.TrimSpace(*presetFlag),
		Draft:  !*registered,
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printOperatingModeSimulation(resp.Msg)
	return nil
}

func printOperatingModeSimulation(sim *apipb.OperatingModeSimulationResponse) {
	printSection("Operating Mode Simulation")
	fmt.Printf("  Mode:   %s\n", sim.GetMode())
	fmt.Printf("  Preset: %s\n", sim.GetActivePreset())
	if presets := sim.GetPresets(); len(presets) > 0 {
		ids := make([]string, 0, len(presets))
		for _, p := range presets {
			ids = append(ids, p.GetId())
		}
		fmt.Printf("  Available presets: %s\n", strings.Join(ids, ", "))
	}
	fmt.Println("  Trace:")
	for _, step := range sim.GetTrace() {
		transition := step.GetTransition()
		if step.GetTerminal() || transition == nil || transition.GetTo() == "" {
			fmt.Printf("    %d. %-14s [terminal]\n", step.GetIndex()+1, step.GetPhase())
			continue
		}
		label := transition.GetLabel()
		if label == "" {
			label = transition.GetConditionKind()
		}
		fmt.Printf("    %d. %-14s → %-14s (%s)\n", step.GetIndex()+1, step.GetPhase(), transition.GetTo(), label)
	}
}
