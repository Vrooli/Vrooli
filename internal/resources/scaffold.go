package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const (
	scaffoldDirectoryMode = 0o755
	scaffoldFileMode      = 0o644
)

var scaffoldNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Scaffold creates the smallest complete resource shape for a supported
// archetype. It writes declarations and entrypoint files only; implementation
// behavior remains the resource author's responsibility.
func (c *Controller) Scaffold(name, driver string) error {
	if !scaffoldNamePattern.MatchString(name) {
		return fmt.Errorf("resource name must contain only lowercase letters, numbers, and hyphens")
	}
	if !slicesContains([]string{"managed-service", "external-cli", "cloud-api", "native-cli"}, driver) {
		return fmt.Errorf("driver %q is not supported; valid archetypes are managed-service, external-cli, cloud-api, and native-cli", driver)
	}
	root := filepath.Join(c.Root, "resources", name)
	if _, err := os.Stat(root); err == nil {
		return fmt.Errorf("resource %q already exists", name)
	} else if !os.IsNotExist(err) {
		return err
	}
	manifest := map[string]any{
		"name":         name,
		"display_name": name,
		"description":  "Generated resource scaffold",
		"driver":       driver,
		"cli": map[string]any{
			"enabled": true, "command": "resource-" + name,
			"adapter":      map[string]any{"kind": "go_module", "module_dir": "cli"},
			"distribution": map[string]any{"kind": "prebuilt_artifact", "artifact_name": "resource-" + name + "_${os}_${arch}"},
			"source_build": map[string]any{"kind": "go_module"},
			"invoke":       map[string]any{"kind": "installed_command", "command": "resource-" + name},
			"freshness":    map[string]any{"inputs": []string{"cli/**", "README.md", "resource.json"}},
		},
		"requirements": map[string]any{"class": "developer-tooling", "weight": 1, "source": "estimated", "confidence": "low"},
		"privilege":    "user",
		"bundling":     "host-required",
	}
	switch driver {
	case accelBridgeExternalCli, "native-cli":
		manifest["binary"] = "resource-" + name
	case "cloud-api":
		manifest["endpoint"] = "https://example.invalid/" + name
	case accelBridgeManagedService:
		manifest["managed_service"] = map[string]any{
			"provider_policy": map[string]any{
				"target_defaults": map[string]string{"control-plane": "managed-private", "desktop-bundle": "managed-private"},
				"allowed_modes":   []string{"managed-private"}, "external_management": "forbidden",
			},
			"artifact": map[string]any{"path": "server/" + name, "version": "0.1.0", "verification": "host-tool"},
		}
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "cli"), scaffoldDirectoryMode); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), scaffoldDirectoryMode); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "resource.json"), append(data, '\n'), scaffoldFileMode); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# "+name+"\n\nGenerated resource scaffold.\n"), scaffoldFileMode); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "README.md"), []byte("# "+name+" documentation\n"), scaffoldFileMode); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("build:\n\tgo build ./cli\n"), scaffoldFileMode); err != nil {
		return err
	}
	module := "module resource-" + name + "\n\ngo 1.25.0\n"
	if err := os.WriteFile(filepath.Join(root, "cli", "go.mod"), []byte(module), scaffoldFileMode); err != nil {
		return err
	}
	mainSource := `package main

import (
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
)

func main() {
	env := cliapp.StandardResourceEnv("` + name + `", cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name: "` + name + `", Version: "0.1.0", Description: "` + name + ` resource CLI",
		SourceRootEnvVars: env.SourceRootEnvVars, ControlPlaneEnvVars: env.ControlPlaneEnvVars,
	})
	if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	app.SetCommands(app.StandardLifecycleCommands())
	if err := app.CLI.Run(os.Args[1:]); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
`
	if err := os.WriteFile(filepath.Join(root, "cli", "main.go"), []byte(mainSource), scaffoldFileMode); err != nil {
		return err
	}
	contractPath := filepath.Join(c.Root, ".vrooli", "service.json")
	contractData, err := os.ReadFile(contractPath)
	if err != nil {
		return err
	}
	var contract map[string]any
	if err := json.Unmarshal(contractData, &contract); err != nil {
		return err
	}
	dependencies, _ := contract["dependencies"].(map[string]any)
	if dependencies == nil {
		dependencies = map[string]any{}
		contract["dependencies"] = dependencies
	}
	resources, _ := dependencies["resources"].(map[string]any)
	if resources == nil {
		resources = map[string]any{}
		dependencies["resources"] = resources
	}
	resources[name] = map[string]any{"enabled": false}
	contractData, err = json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(contractPath, append(contractData, '\n'), scaffoldFileMode)
}

func slicesContains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
