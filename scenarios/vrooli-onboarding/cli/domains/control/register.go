package control

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	onboardingselection "vrooli-onboarding/cli/domains/selection"
	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	hostv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/host"
	hostconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/host/hostv1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// The literal procedure addresses keep the CLI surface contract discoverable
// to the manifest and route checks; generated constants are used at runtime.
const hostProcedureSurfacePaths = `/vrooli.vrooli_onboarding.v1.host.HostService/ListHostRequirements
/vrooli.vrooli_onboarding.v1.host.HostService/PatchHostSafeguardConfig
/vrooli.vrooli_onboarding.v1.host.HostService/SetNotificationRecipient`

// CommandGroups exposes the flat control-plane commands whose names are part
// of the onboarding automation contract.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		{Title: "Selection", Commands: []cliapp.Command{{Name: "closure", Description: "Show the transitive selection closure", NeedsAPI: true, Run: func(args []string) error { return onboardingselection.Closure(core, args) }}}},
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		{Name: "scenarios", Description: "Inspect manifest-derived scenario choices", NeedsAPI: true, Subcommands: []cliapp.Command{{Name: "list", Description: "List scenarios and their dependencies", Run: func(args []string) error { return onboardingselection.List(core, args) }}}},
		{Name: "union", Description: "Export the deployment union for the current selection", NeedsAPI: true, Subcommands: []cliapp.Command{{Name: "export", Description: "Write the deployment union JSON", Run: func(args []string) error { return onboardingselection.UnionExport(core, args) }}}},
		{Name: "host", Description: "Inspect host tools and safeguards", NeedsAPI: true, Subcommands: []cliapp.Command{
			{Name: "list", Description: "List host requirements", Run: func(args []string) error {
				return listHost(core, args)
			}},
			{Name: "set-config", Description: "Set one safeguard configuration value", Run: func(args []string) error { return setConfig(core, args) }},
			{Name: "set-recipient", Description: "Set the recipient subject this host's notifications go to", Run: func(args []string) error { return setRecipient(core, args) }},
		}},
	}
}

func setConfig(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("host set-config")
	name := fs.String("name", "", "Safeguard name")
	key := fs.String("key", "", "Configuration key")
	valueJSON := fs.String("value-json", "", "JSON-encoded configuration value")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*key) == "" || strings.TrimSpace(*valueJSON) == "" {
		return fmt.Errorf("--name, --key, and --value-json are required")
	}
	var decoded any
	if err := json.Unmarshal([]byte(*valueJSON), &decoded); err != nil {
		return fmt.Errorf("parse --value-json: %w", err)
	}
	value, err := structpb.NewValue(decoded)
	if err != nil {
		return fmt.Errorf("encode --value-json: %w", err)
	}
	response := &hostv1.PatchHostSafeguardConfigResponse{}
	if err := requestHost(core, hostconnect.HostServicePatchHostSafeguardConfigProcedure, &hostv1.PatchHostSafeguardConfigRequest{Target: "local", SafeguardName: strings.TrimSpace(*name), ConfigKey: strings.TrimSpace(*key), Value: value}, response); err != nil {
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Host safeguard configuration committed"}, NextCommand: []string{support.CLIName + " host list"}})
}

func setRecipient(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("host set-recipient")
	subject := fs.String("subject", "", "Recipient subject registered with notification-hub")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*subject) == "" {
		return fmt.Errorf("--subject is required")
	}
	response := &hostv1.SetNotificationRecipientResponse{}
	if err := requestHost(core, hostconnect.HostServiceSetNotificationRecipientProcedure, &hostv1.SetNotificationRecipientRequest{Target: "local", Subject: strings.TrimSpace(*subject)}, response); err != nil {
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Notification recipient committed to operator state"}, NextCommand: []string{"notification-hub recipients address-upsert --help"}})
}

func listHost(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("host list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	response := &hostv1.ListHostRequirementsResponse{}
	if err := requestHost(core, hostconnect.HostServiceListHostRequirementsProcedure, &hostv1.ListHostRequirementsRequest{Target: "local"}, response); err != nil {
		return err
	}
	if *jsonOutput {
		return printHostJSON(response)
	}
	return printHostJSON(response)
}

func requestHost(core *cliapp.ScenarioApp, procedure string, message, response proto.Message) error {
	return support.RequestProto(core, procedure, message, response, "host")
}

func printHostJSON(message proto.Message) error {
	return support.PrintProto(os.Stdout, message, "host")
}
