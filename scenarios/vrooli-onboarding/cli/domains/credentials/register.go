package credentials

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "credentials", Description: "Inspect and provision onboarding credentials", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "list", Description: "List credential descriptors", Run: func(args []string) error { return get(core, args, "/v2/credentials") }},
		{Name: "provision", Description: "Provision a credential from standard input", Run: func(args []string) error { return provision(core, args) }},
		{Name: "doctor", Description: "Diagnose the credential authority", Run: func(args []string) error { return get(core, args, "/v2/credentials/doctor") }},
	}}
}

func get(core *cliapp.ScenarioApp, args []string, path string) error {
	fs := support.NewFlagSet("credentials")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(body, '\n'))
		return err
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return fmt.Errorf("decode credentials response: %w", err)
	}
	pretty, _ := json.MarshalIndent(value, "", "  ")
	_, err = fmt.Fprintln(os.Stdout, string(pretty))
	return err
}

func provision(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("credentials provision")
	logicalID := fs.String("logical-id", "", "Credential logical id")
	field := fs.String("field", "value", "Credential field")
	valueFlag := fs.String("value", "", "Rejected: credential values must be read from standard input")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*valueFlag) != "" {
		return fmt.Errorf("--value is not accepted; pipe the credential value on standard input")
	}
	if strings.TrimSpace(*logicalID) == "" {
		return fmt.Errorf("--logical-id is required")
	}
	value, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read credential from standard input: %w", err)
	}
	if strings.TrimSpace(string(value)) == "" {
		return fmt.Errorf("standard input did not contain a credential value")
	}
	body, _ := json.Marshal(map[string]string{"logical_id": strings.TrimSpace(*logicalID), "field": strings.TrimSpace(*field), "value": strings.TrimSpace(string(value))})
	response, err := core.Request("POST", "/v2/credentials/provision", nil, body)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(response, '\n'))
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Credential provisioned"}, NextCommand: []string{support.CLIName + " readiness"}})
}
