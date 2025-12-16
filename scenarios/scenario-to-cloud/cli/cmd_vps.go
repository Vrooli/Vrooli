package main

import (
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdPreflight(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: scenario-to-cloud preflight <manifest.json>")
	}
	manifest, err := readJSONFile(args[0])
	if err != nil {
		return err
	}
	payload, err := a.api.Request("POST", "/api/v1/preflight", nil, manifest)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}

func (a *App) cmdVPSSetupPlan(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("Usage: scenario-to-cloud vps-setup-plan <manifest.json> <bundle.tar.gz>")
	}
	manifest, err := readJSONFile(args[0])
	if err != nil {
		return err
	}
	req := map[string]any{
		"manifest":    manifest,
		"bundle_path": args[1],
	}
	payload, err := a.api.Request("POST", "/api/v1/vps/setup/plan", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}

func (a *App) cmdVPSSetupApply(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("Usage: scenario-to-cloud vps-setup-apply <manifest.json> <bundle.tar.gz>")
	}
	manifest, err := readJSONFile(args[0])
	if err != nil {
		return err
	}
	req := map[string]any{
		"manifest":    manifest,
		"bundle_path": args[1],
	}
	payload, err := a.api.Request("POST", "/api/v1/vps/setup/apply", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}

func (a *App) cmdVPSDeployPlan(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: scenario-to-cloud vps-deploy-plan <manifest.json>")
	}
	manifest, err := readJSONFile(args[0])
	if err != nil {
		return err
	}
	req := map[string]any{"manifest": manifest}
	payload, err := a.api.Request("POST", "/api/v1/vps/deploy/plan", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}

func (a *App) cmdVPSDeployApply(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: scenario-to-cloud vps-deploy-apply <manifest.json>")
	}
	manifest, err := readJSONFile(args[0])
	if err != nil {
		return err
	}
	req := map[string]any{"manifest": manifest}
	payload, err := a.api.Request("POST", "/api/v1/vps/deploy/apply", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}

func (a *App) cmdVPSInspectPlan(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: scenario-to-cloud vps-inspect-plan <manifest.json>")
	}
	manifest, err := readJSONFile(args[0])
	if err != nil {
		return err
	}
	req := map[string]any{
		"manifest": manifest,
		"options": map[string]any{
			"tail_lines": 200,
		},
	}
	payload, err := a.api.Request("POST", "/api/v1/vps/inspect/plan", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}

func (a *App) cmdVPSInspectApply(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: scenario-to-cloud vps-inspect-apply <manifest.json>")
	}
	manifest, err := readJSONFile(args[0])
	if err != nil {
		return err
	}
	req := map[string]any{
		"manifest": manifest,
		"options": map[string]any{
			"tail_lines": 200,
		},
	}
	payload, err := a.api.Request("POST", "/api/v1/vps/inspect/apply", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}

