package main

import (
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// postManifestOnly reads a manifest file and POSTs it directly to the endpoint.
// Used for endpoints that only require a manifest (e.g., preflight).
func (a *App) postManifestOnly(args []string, cmdName, endpoint string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: scenario-to-cloud %s <manifest.json>", cmdName)
	}
	manifest, err := readJSONFile(args[0])
	if err != nil {
		return err
	}
	payload, err := a.api.Request("POST", endpoint, nil, manifest)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}

// postManifestWrapped reads a manifest file and POSTs it wrapped in {"manifest": ...}.
// Used for VPS endpoints that expect the manifest inside a wrapper object.
func (a *App) postManifestWrapped(args []string, cmdName, endpoint string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: scenario-to-cloud %s <manifest.json>", cmdName)
	}
	manifest, err := readJSONFile(args[0])
	if err != nil {
		return err
	}
	req := map[string]any{"manifest": manifest}
	payload, err := a.api.Request("POST", endpoint, nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}

// postManifestWithBundle reads a manifest file and POSTs it with a bundle_path.
// Used for VPS setup endpoints that require both manifest and bundle path.
func (a *App) postManifestWithBundle(args []string, cmdName, endpoint string) error {
	if len(args) != 2 {
		return fmt.Errorf("Usage: scenario-to-cloud %s <manifest.json> <bundle.tar.gz>", cmdName)
	}
	manifest, err := readJSONFile(args[0])
	if err != nil {
		return err
	}
	req := map[string]any{
		"manifest":    manifest,
		"bundle_path": args[1],
	}
	payload, err := a.api.Request("POST", endpoint, nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}

// postManifestWithOptions reads a manifest file and POSTs it with default options.
// Used for VPS inspect endpoints that include tail_lines configuration.
func (a *App) postManifestWithOptions(args []string, cmdName, endpoint string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: scenario-to-cloud %s <manifest.json>", cmdName)
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
	payload, err := a.api.Request("POST", endpoint, nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}

func (a *App) cmdPreflight(args []string) error {
	return a.postManifestOnly(args, "preflight", "/api/v1/preflight")
}

func (a *App) cmdVPSSetupPlan(args []string) error {
	return a.postManifestWithBundle(args, "vps-setup-plan", "/api/v1/vps/setup/plan")
}

func (a *App) cmdVPSSetupApply(args []string) error {
	return a.postManifestWithBundle(args, "vps-setup-apply", "/api/v1/vps/setup/apply")
}

func (a *App) cmdVPSDeployPlan(args []string) error {
	return a.postManifestWrapped(args, "vps-deploy-plan", "/api/v1/vps/deploy/plan")
}

func (a *App) cmdVPSDeployApply(args []string) error {
	return a.postManifestWrapped(args, "vps-deploy-apply", "/api/v1/vps/deploy/apply")
}

func (a *App) cmdVPSInspectPlan(args []string) error {
	return a.postManifestWithOptions(args, "vps-inspect-plan", "/api/v1/vps/inspect/plan")
}

func (a *App) cmdVPSInspectApply(args []string) error {
	return a.postManifestWithOptions(args, "vps-inspect-apply", "/api/v1/vps/inspect/apply")
}

