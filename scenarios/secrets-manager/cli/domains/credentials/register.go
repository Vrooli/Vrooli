package credentials

import (
	"context"
	"fmt"
	"strings"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	"secrets-manager/cli/internal/credentials"
	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

type inventoryEntry struct {
	credentialclient.CredentialRef
	Configured    bool   `json:"configured"`
	Provider      string `json:"provider"`
	ProviderState string `json:"provider_state"`
}

func Register(_ *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "credentials",
		Description: "Metadata-safe credential inventory and diagnosis",
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List declared credentials and status without values", Run: runList},
			{Name: "status", Description: "Show metadata-safe status for one credential", Run: runStatus},
			{Name: "doctor", Description: "Diagnose the credential provider and recovery coverage", Run: runDoctor},
			{Name: "provision", Description: "Provision one credential from stdin", Run: runProvision},
			{Name: "delete", Description: "Delete one credential after explicit confirmation", Run: runDelete},
		},
	}
}

func runList(args []string) error {
	fs := support.NewFlagSet("credentials list")
	jsonFlag, format := support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	jsonOutput, err := support.ResolveJSONOutput(*jsonFlag, *format)
	if err != nil {
		return err
	}
	client, err := credentials.New()
	if err != nil {
		return err
	}
	refs, err := client.List(context.Background())
	if err != nil {
		return err
	}
	entries := make([]inventoryEntry, 0, len(refs))
	for _, ref := range refs {
		status, statusErr := client.Status(context.Background(), ref.LogicalID, ref.Field)
		entry := inventoryEntry{CredentialRef: ref}
		if statusErr != nil {
			entry.ProviderState = "unavailable"
		} else {
			entry.Configured = status.Configured
			entry.Provider = status.Provider
			entry.ProviderState = status.ProviderState
		}
		entries = append(entries, entry)
	}
	results := make([]string, 0, len(entries))
	for _, entry := range entries {
		results = append(results, fmt.Sprintf("%s:%s | configured=%t | provider=%s | state=%s", entry.LogicalID, entry.Field, entry.Configured, entry.Provider, entry.ProviderState))
	}
	return support.PrintList(jsonOutput, entries, cliapp.ListReport{Summary: []string{fmt.Sprintf("Declared credentials: %d", len(entries))}, ResultsHeading: "Credentials", Results: results})
}

func runStatus(args []string) error {
	fs := support.NewFlagSet("credentials status")
	identity := fs.String("identity", "", "credential logical identity")
	field := fs.String("field", "", "credential field")
	jsonFlag, format := support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	jsonOutput, err := support.ResolveJSONOutput(*jsonFlag, *format)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*identity) == "" || strings.TrimSpace(*field) == "" {
		return fmt.Errorf("credentials status requires --identity and --field")
	}
	client, err := credentials.New()
	if err != nil {
		return err
	}
	status, err := client.Status(context.Background(), *identity, *field)
	if err != nil {
		return err
	}
	return support.PrintOperational(jsonOutput, status, cliapp.OperationalReport{Status: []string{fmt.Sprintf("%s:%s configured=%t", status.Identity, status.Field, status.Configured), "provider=" + status.Provider, "provider_state=" + status.ProviderState}})
}

func runDoctor(args []string) error {
	fs := support.NewFlagSet("credentials doctor")
	jsonFlag, format := support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	jsonOutput, err := support.ResolveJSONOutput(*jsonFlag, *format)
	if err != nil {
		return err
	}
	client, err := credentials.New()
	if err != nil {
		return err
	}
	diagnosis, err := client.Doctor(context.Background())
	if err != nil {
		return err
	}
	return support.PrintOperational(jsonOutput, diagnosis, cliapp.OperationalReport{Status: []string{"provider=" + diagnosis.Provider.Backend, "condition=" + diagnosis.Provider.Condition, fmt.Sprintf("recovery_receipt=%t", diagnosis.Recovery.ReceiptExists), fmt.Sprintf("uncovered=%d", len(diagnosis.Recovery.Uncovered))}, NextSteps: []string{"secrets-manager backup export --output <bundle> < passphrase"}})
}

func runProvision(args []string) error {
	fs := support.NewFlagSet("credentials provision")
	identity := fs.String("identity", "", "credential logical identity")
	field := fs.String("field", "", "credential field")
	jsonFlag, format := support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	jsonOutput, err := support.ResolveJSONOutput(*jsonFlag, *format)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*identity) == "" || strings.TrimSpace(*field) == "" {
		return fmt.Errorf("credentials provision requires --identity and --field")
	}
	value, err := credentials.ReadPassphrase()
	if err != nil {
		return err
	}
	client, err := credentials.New()
	if err != nil {
		return err
	}
	response, err := client.Provision(context.Background(), credentialclient.ProvisionRequest{Identity: *identity, Field: *field, Value: value})
	if err != nil {
		return err
	}
	return support.PrintMutation(jsonOutput, response, cliapp.MutationReport{Result: []string{"credential provisioned; value was accepted only from stdin"}})
}

func runDelete(args []string) error {
	fs := support.NewFlagSet("credentials delete")
	identity := fs.String("identity", "", "credential logical identity")
	field := fs.String("field", "", "credential field")
	yes := fs.Bool("yes", false, "confirm deletion")
	jsonFlag, format := support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	jsonOutput, err := support.ResolveJSONOutput(*jsonFlag, *format)
	if err != nil {
		return err
	}
	if !*yes {
		return fmt.Errorf("credentials delete requires --yes")
	}
	if strings.TrimSpace(*identity) == "" || strings.TrimSpace(*field) == "" {
		return fmt.Errorf("credentials delete requires --identity and --field")
	}
	client, err := credentials.New()
	if err != nil {
		return err
	}
	if err := client.Delete(context.Background(), *identity, *field); err != nil {
		return err
	}
	return support.PrintMutation(jsonOutput, map[string]any{"identity": *identity, "field": *field, "status": "deleted"}, cliapp.MutationReport{Result: []string{"credential deleted"}})
}
