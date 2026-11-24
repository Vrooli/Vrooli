package credentials

import (
	"fmt"
	"io"
	"os"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/credentials"
	credentialsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/credentials/credentialsv1connect"
	"golang.org/x/term"
	"google.golang.org/protobuf/proto"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "credentials", Description: "Inspect and provision onboarding credentials", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "list", Description: "List credential descriptors", Run: func(args []string) error { return list(core, args) }},
		{Name: "provision", Description: "Provision a credential from standard input", Run: func(args []string) error { return provision(core, args) }},
		{Name: "doctor", Description: "Diagnose the credential authority", Run: func(args []string) error { return diagnose(core, args) }},
	}}
}

// ManifestHandlers supplies credential commands whose operations need local
// terminal handling. Provision still crosses the generated Connect contract;
// reveal is deliberately local-only so a secret cannot enter an API response,
// browser, or remote transport.
func ManifestHandlers(core *cliapp.ScenarioApp) map[string]cliapp.PrimitiveHandler {
	return map[string]cliapp.PrimitiveHandler{
		"CredentialsService.ProvisionCredential": cliapp.ExternalDelegation(func(ctx cliapp.RunContext) error {
			response, err := provisionCredential(ctx.Core(), ctx.Flag("logical-id"), ctx.Flag("field"), os.Stdin)
			if err != nil {
				return err
			}
			return renderProvision(ctx.Stdout(), ctx.JSON(), response)
		}),
		"credentials.reveal": cliapp.ExternalDelegation(reveal),
	}
}

type credentialResolver interface {
	Resolve(credentialauthority.Identity, string) (string, error)
}

var revealAuthority = func() (credentialResolver, error) {
	return credentialauthority.Default()
}

func reveal(ctx cliapp.RunContext) error {
	if ctx.JSON() {
		return fmt.Errorf("credentials reveal refuses --json; revealed values may only be written to an interactive terminal")
	}
	logicalID := strings.TrimSpace(ctx.Flag("logical-id"))
	if logicalID == "" {
		return fmt.Errorf("--logical-id is required")
	}
	field := strings.TrimSpace(ctx.Flag("field"))
	if field == "" {
		field = "value"
	}
	if !ctx.BoolFlag("confirm-reveal") {
		return fmt.Errorf("refusing to reveal %s:%s without --confirm-reveal", logicalID, field)
	}
	if !interactiveTerminal(ctx.Stdout()) {
		return fmt.Errorf("refusing to reveal %s:%s because stdout is not an interactive terminal; remove redirection and do not pipe the value", logicalID, field)
	}
	identity, err := credentialauthority.ParseIdentity(logicalID)
	if err != nil {
		return err
	}
	authority, err := revealAuthority()
	if err != nil {
		return fmt.Errorf("credential authority unavailable: %w", err)
	}
	value, err := authority.Resolve(identity, field)
	if err != nil {
		return fmt.Errorf("reveal credential %s:%s: %w", identity, field, err)
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("credential %s:%s is empty", identity, field)
	}
	_, err = fmt.Fprintln(ctx.Stdout(), value)
	return err
}

func interactiveTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		// Test and embedding writers are trusted by the caller; the production
		// default is os.Stdout and is checked below.
		return true
	}
	fd := int(file.Fd())
	return term.IsTerminal(fd)
}

func provisionCredential(core *cliapp.ScenarioApp, logicalID, field string, input io.Reader) (*credentialsv1.ProvisionCredentialResponse, error) {
	logicalID = strings.TrimSpace(logicalID)
	if logicalID == "" {
		return nil, fmt.Errorf("--logical-id is required")
	}
	field = strings.TrimSpace(field)
	if field == "" {
		field = "value"
	}
	value, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("read credential from standard input: %w", err)
	}
	valueText := strings.TrimSpace(string(value))
	if valueText == "" {
		return nil, fmt.Errorf("standard input did not contain a credential value")
	}
	response := &credentialsv1.ProvisionCredentialResponse{}
	request := &credentialsv1.ProvisionCredentialRequest{Target: "local", LogicalId: logicalID, Field: field, Value: valueText}
	if err := requestRPC(core, provisionProcedure, request, response); err != nil {
		return nil, err
	}
	return response, nil
}

func renderProvision(output io.Writer, jsonOutput bool, response *credentialsv1.ProvisionCredentialResponse) error {
	if jsonOutput {
		return support.PrintProto(output, response, "credential")
	}
	return cliapp.RenderMutationReport(output, cliapp.MutationReport{Result: []string{"Credential provisioned"}, NextCommand: []string{support.CLIName + " readiness"}})
}

const (
	listProcedure      = credentialsconnect.CredentialsServiceListCredentialsProcedure
	provisionProcedure = credentialsconnect.CredentialsServiceProvisionCredentialProcedure
	diagnoseProcedure  = credentialsconnect.CredentialsServiceDiagnoseCredentialsProcedure
)

func list(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("credentials list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	response := &credentialsv1.ListCredentialsResponse{}
	if err := request(core, listProcedure, &credentialsv1.ListCredentialsRequest{Target: "local"}, response); err != nil {
		return err
	}
	return renderJSONOrPretty(response, *jsonOutput)
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
	response, err := provisionCredential(core, *logicalID, *field, os.Stdin)
	if err != nil {
		return err
	}
	return renderProvision(os.Stdout, *jsonOutput, response)
}

func diagnose(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("credentials doctor")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	response := &credentialsv1.DiagnoseCredentialsResponse{}
	if err := request(core, diagnoseProcedure, &credentialsv1.DiagnoseCredentialsRequest{Target: "local"}, response); err != nil {
		return err
	}
	return renderJSONOrPretty(response, *jsonOutput)
}

func request(core *cliapp.ScenarioApp, procedure string, message, response proto.Message) error {
	return support.RequestProto(core, procedure, message, response, "credential")
}

func requestRPC(core *cliapp.ScenarioApp, procedure string, message, response proto.Message) error {
	return request(core, procedure, message, response)
}

func printJSON(message proto.Message) error {
	return support.PrintProto(os.Stdout, message, "credential")
}

func renderJSONOrPretty(message proto.Message, jsonOutput bool) error {
	return printJSON(message)
}
