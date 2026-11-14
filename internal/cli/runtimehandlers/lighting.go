package runtimehandlers

import (
	"context"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup"
)

func runLightingManifest(ctx context.Context, app *runtimeapp.Service, command *rootcli.CommandContext, args []string) error {
	if len(args) > 0 && args[0] == "permissions" {
		return runLightingPermissionManifest(ctx, app, command, args[1:])
	}
	bindings := map[string]func(cliapp.RunContext) error{}
	for _, action := range []string{"status", "prepare", "setup", "enable", "disable", "devices"} {
		bindings[action] = func(run cliapp.RunContext) error {
			root, err := command.ResolveRoot()
			if err != nil {
				return err
			}
			home, err := command.HomeDir()
			if err != nil {
				return err
			}
			if os.Geteuid() == 0 && os.Getenv("SUDO_UID") != "" {
				home, err = hostreqkit.InvokingUserHomeDir()
				if err != nil {
					return err
				}
			}
			actions, err := app.NewNativeLightingActions(root, home, runtimeapp.EnsureNativeLightingScenario)
			if err != nil {
				return err
			}
			var result *setupv1.LightingRuntimeStatus
			switch action {
			case "prepare":
				if err = actions.PrepareNative(ctx); err != nil {
					return err
				}
				result, err = actions.Status(ctx)
				if err == nil {
					result.Message = "Published the pinned native lighting artifacts and sealed operator supervisor. Device enrollment and enablement remain explicit."
				}
			case "devices":
				result, err := actions.Devices(ctx)
				if err != nil {
					return err
				}
				if run.JSON() || command.Globals.JSON {
					return cliout.WriteProtoJSON(command.Stdout, result)
				}
				if _, err := fmt.Fprintln(command.Stdout, result.Message); err != nil {
					return err
				}
				for _, device := range result.Candidates {
					if _, err := fmt.Fprintf(command.Stdout, "%s: %s (%s)\n", device.Id, device.Label, device.Kind); err != nil {
						return err
					}
				}
				return nil
			case "status":
				result, err = actions.Status(ctx)
			case "disable":
				result, err = actions.Disable(ctx)
			default:
				source := setupv1.InstallationSource_INSTALLATION_SOURCE_UNSPECIFIED
				switch run.Flag("source") {
				case "managed":
					source = setupv1.InstallationSource_INSTALLATION_SOURCE_MANAGED
				case "existing":
					source = setupv1.InstallationSource_INSTALLATION_SOURCE_EXISTING
				}
				selection := &setupv1.LightingSelection{Source: source, Version: run.Flag("version"), Sha256: run.Flag("sha256"), DevicePolicyId: run.Flag("device-policy-id")}
				if action == "setup" {
					result, err = actions.Setup(ctx, selection)
				} else {
					result, err = actions.Enable(ctx, selection)
				}
			}
			if err != nil {
				return err
			}
			if run.JSON() || command.Globals.JSON {
				return cliout.WriteProtoJSON(command.Stdout, result)
			}
			_, err = fmt.Fprintf(command.Stdout, "%s: %s\n", result.State, result.Message)
			return err
		}
	}
	group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "runtime/lighting", bindings)
	if err != nil {
		return err
	}
	core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli runtime", SubcommandGroups: []cliapp.SubcommandGroup{group}})
	return core.RunWithWriters(append([]string{"lighting"}, args...), command.Stdout, command.Stderr)
}

func runLightingPermissionManifest(ctx context.Context, app *runtimeapp.Service, command *rootcli.CommandContext, args []string) error {
	bindings := map[string]func(cliapp.RunContext) error{}
	for _, action := range []string{"plan", "apply", "rollback"} {
		bindings[action] = func(run cliapp.RunContext) error {
			root, err := command.ResolveRoot()
			if err != nil {
				return err
			}
			home, err := command.HomeDir()
			if err != nil {
				return err
			}
			if os.Geteuid() == 0 && os.Getenv("SUDO_UID") != "" {
				home, err = hostreqkit.InvokingUserHomeDir()
				if err != nil {
					return err
				}
			}
			actions, err := app.NewNativeLightingActions(root, home, runtimeapp.EnsureNativeLightingScenario)
			if err != nil {
				return err
			}
			if action == "plan" {
				result, err := actions.PlanDevices(ctx, run.FlagValues("device-id"))
				if err != nil {
					return err
				}
				if run.JSON() || command.Globals.JSON {
					return cliout.WriteProtoJSON(command.Stdout, result)
				}
				if _, err := fmt.Fprintf(command.Stdout, "Plan %s: %s\n", result.Id, result.Message); err != nil {
					return err
				}
				for _, change := range result.Changes {
					if _, err := fmt.Fprintln(command.Stdout, change); err != nil {
						return err
					}
				}
				return nil
			}
			var result *setupv1.LightingDeviceAccessResult
			if action == "apply" {
				result, err = actions.ApplyDevices(ctx, run.Flag("plan-id"), run.Flag("sudo-mode"))
			} else {
				result, err = actions.RollbackDevices(ctx, run.Flag("plan-id"), run.Flag("sudo-mode"))
			}
			if err != nil {
				if result != nil {
					if run.JSON() || command.Globals.JSON {
						if writeErr := cliout.WriteProtoJSON(command.Stdout, result); writeErr != nil {
							return writeErr
						}
					} else {
						fmt.Fprintf(command.Stdout, "%s: %s\n", result.State, result.Message)
					}
				}
				return err
			}
			if run.JSON() || command.Globals.JSON {
				return cliout.WriteProtoJSON(command.Stdout, result)
			}
			_, err = fmt.Fprintf(command.Stdout, "%s: %s\n", result.State, result.Message)
			return err
		}
	}
	group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "runtime/lighting/permissions", bindings)
	if err != nil {
		return err
	}
	core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli runtime lighting", SubcommandGroups: []cliapp.SubcommandGroup{group}})
	return core.RunWithWriters(append([]string{"permissions"}, args...), command.Stdout, command.Stderr)
}
