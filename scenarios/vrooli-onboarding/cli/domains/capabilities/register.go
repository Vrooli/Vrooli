package capabilities

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/capabilities"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/capabilities/capabilitiesv1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
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
	response := &capabilitiesv1.ListCapabilitiesResponse{}
	if err := request(core, capabilitiesconnect.CapabilitiesServiceListCapabilitiesProcedure, &capabilitiesv1.ListCapabilitiesRequest{Target: "local"}, response); err != nil {
		return err
	}
	return renderJSONOrPretty(response, *jsonOutput, "capabilities")
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
	inputStruct, err := structpb.NewStruct(inputs)
	if err != nil {
		return fmt.Errorf("encode capability inputs: %w", err)
	}
	request := &capabilitiesv1.ActionRequest{CapabilityId: strings.TrimSpace(*id), Confirm: false, Inputs: inputStruct}
	preview := &capabilitiesv1.PreviewCapabilityResponse{}
	if err := requestRPC(core, capabilitiesconnect.CapabilitiesServicePreviewCapabilityProcedure, &capabilitiesv1.PreviewCapabilityRequest{Target: "local", Action: request}, preview); err != nil {
		return err
	}
	if *jsonOutput || !apply {
		if err := renderJSONOrPretty(preview, *jsonOutput, "capability preview"); err != nil {
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
	request.Confirm = true
	result := &capabilitiesv1.ApplyCapabilityResponse{}
	if err := requestRPC(core, capabilitiesconnect.CapabilitiesServiceApplyCapabilityProcedure, &capabilitiesv1.ApplyCapabilityRequest{Target: "local", Action: request}, result); err != nil {
		return err
	}
	if *jsonOutput {
		return printJSON(result)
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
	response := &capabilitiesv1.ListCapabilitiesResponse{}
	if err := requestRPC(core, capabilitiesconnect.CapabilitiesServiceListCapabilitiesProcedure, &capabilitiesv1.ListCapabilitiesRequest{Target: "local"}, response); err != nil {
		return nil, nil, err
	}
	var secrets []string
	for _, capability := range response.GetCapabilities() {
		if capability.GetDescriptor_().GetId() != capabilityID {
			continue
		}
		for _, input := range capability.GetDescriptor_().GetInputs() {
			if input.GetKind() == "secret" {
				secrets = append(secrets, input.GetId())
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

func request(core *cliapp.ScenarioApp, procedure string, message, response proto.Message) error {
	return requestRPC(core, procedure, message, response)
}

func requestRPC(core *cliapp.ScenarioApp, procedure string, message, response proto.Message) error {
	return support.RequestProto(core, procedure, message, response, "capability")
}

func printJSON(message proto.Message) error {
	return support.PrintProto(os.Stdout, message, "capability")
}

func renderJSONOrPretty(message proto.Message, jsonOutput bool, label string) error {
	if jsonOutput {
		return printJSON(message)
	}
	return support.PrintProto(os.Stdout, message, label)
}
