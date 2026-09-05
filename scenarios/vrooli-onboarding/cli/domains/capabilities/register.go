package capabilities

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type stringList []string

func (list *stringList) String() string { return strings.Join(*list, ",") }
func (list *stringList) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("--input cannot be empty")
	}
	*list = append(*list, value)
	return nil
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "capabilities", Description: "Inspect and apply operator capabilities", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "list", Description: "List capability descriptors and status", Run: func(args []string) error { return list(core, args) }},
		{Name: "preview", Description: "Preview a capability action", Run: func(args []string) error { return action(core, args, false) }},
		{Name: "apply", Description: "Preview and apply a capability action", Run: func(args []string) error { return action(core, args, true) }},
	}}
}

func list(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("capabilities list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get("/v2/capabilities", nil)
	if err != nil {
		return err
	}
	return renderJSONOrPretty(body, *jsonOutput, "capabilities")
}

func action(core *cliapp.ScenarioApp, args []string, apply bool) error {
	fs := support.NewFlagSet("capabilities action")
	id := fs.String("id", "", "Capability id")
	var inputFlags stringList
	fs.Var(&inputFlags, "input", "Non-secret input as key=value; repeatable")
	secretFlag := fs.String("secret", "", "Rejected: capability secrets must be read from standard input")
	confirm := fs.Bool("confirm", false, "Confirm the reviewed capability preview")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*secretFlag) != "" {
		return fmt.Errorf("--secret is not accepted; pipe capability secret values on standard input")
	}
	if strings.TrimSpace(*id) == "" {
		return fmt.Errorf("--id is required")
	}
	inputs, secretInputs, err := readCapabilityInputs(core, *id, inputFlags)
	if err != nil {
		return err
	}
	for key, value := range secretInputs {
		inputs[key] = value
	}
	request := map[string]any{"capability_id": strings.TrimSpace(*id), "confirm": false, "inputs": inputs}
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	previewBody, err := core.Request("POST", "/v2/capabilities/preview", nil, body)
	if err != nil {
		return err
	}
	if *jsonOutput || !apply {
		if err := renderJSONOrPretty(previewBody, *jsonOutput, "capability preview"); err != nil {
			return err
		}
	}
	if !apply {
		return nil
	}
	if !*confirm {
		reader := bufio.NewReader(os.Stdin)
		_, _ = fmt.Fprint(os.Stdout, "Apply this reviewed capability? answer yes or no: ")
		answer, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return readErr
		}
		if strings.ToLower(strings.TrimSpace(answer)) != "yes" {
			return fmt.Errorf("capability apply not confirmed")
		}
	}
	request["confirm"] = true
	body, err = json.Marshal(request)
	if err != nil {
		return err
	}
	result, err := core.Request("POST", "/v2/capabilities/apply", nil, body)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(result, '\n'))
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Capability applied"}, NextCommand: []string{support.CLIName + " capabilities list"}})
}

func readCapabilityInputs(core *cliapp.ScenarioApp, capabilityID string, flags []string) (map[string]any, map[string]string, error) {
	inputs := map[string]any{}
	for _, raw := range flags {
		key, value, ok := strings.Cut(raw, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, nil, fmt.Errorf("--input must use key=value")
		}
		inputs[strings.TrimSpace(key)] = value
	}
	body, err := core.Get("/v2/capabilities", nil)
	if err != nil {
		return nil, nil, err
	}
	var response struct {
		Capabilities []struct {
			Descriptor struct {
				ID     string `json:"id"`
				Inputs []struct {
					ID   string `json:"id"`
					Kind string `json:"kind"`
				} `json:"inputs"`
			} `json:"descriptor"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, nil, fmt.Errorf("decode capabilities: %w", err)
	}
	var secrets []string
	for _, capability := range response.Capabilities {
		if capability.Descriptor.ID != capabilityID {
			continue
		}
		for _, input := range capability.Descriptor.Inputs {
			if input.Kind == "secret" {
				secrets = append(secrets, input.ID)
			}
		}
	}
	secretValues := map[string]string{}
	if len(secrets) == 0 {
		return inputs, secretValues, nil
	}
	reader := bufio.NewReader(os.Stdin)
	for _, id := range secrets {
		value, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, nil, fmt.Errorf("read secret input %s: %w", id, readErr)
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, nil, fmt.Errorf("secret input %s was empty", id)
		}
		secretValues[id] = value
	}
	return inputs, secretValues, nil
}

func renderJSONOrPretty(body []byte, jsonOutput bool, label string) error {
	if jsonOutput {
		_, err := os.Stdout.Write(append(body, '\n'))
		return err
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return fmt.Errorf("decode %s response: %w", label, err)
	}
	pretty, _ := json.MarshalIndent(value, "", "  ")
	_, err := fmt.Fprintln(os.Stdout, string(pretty))
	return err
}
