package backup

import (
	"context"
	"fmt"

	"secrets-manager/cli/internal/credentials"
	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

func Register(_ *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "backup",
		Description: "Export, verify, and migrate credential recovery bundles",
		Subcommands: []cliapp.Command{
			{Name: "status", Description: "Report recovery receipt and uncovered credentials", Run: runStatus},
			{Name: "export", Description: "Export and verify the declared credential inventory", Run: runExport},
			{Name: "migrate", Description: "Move a legacy plaintext JSON file behind a verified recovery bundle", Run: runMigrate},
		},
	}
}

func runStatus(args []string) error {
	fs := support.NewFlagSet("backup status")
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
	response, err := client.Doctor(context.Background())
	if err != nil {
		return err
	}
	return support.PrintOperational(jsonOutput, response, cliapp.OperationalReport{
		Status:    []string{fmt.Sprintf("Recovery receipt: %t", response.Recovery.ReceiptExists), fmt.Sprintf("Covered entries: %d", response.Recovery.EntryCount), fmt.Sprintf("Uncovered: %d", len(response.Recovery.Uncovered))},
		NextSteps: []string{"secrets-manager backup export --output <bundle> < passphrase"},
	})
}

func runExport(args []string) error {
	fs := support.NewFlagSet("backup export")
	output := fs.String("output", "", "recovery bundle output path")
	jsonFlag, format := support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	jsonOutput, err := support.ResolveJSONOutput(*jsonFlag, *format)
	if err != nil {
		return err
	}
	if *output == "" {
		return fmt.Errorf("backup export requires --output")
	}
	passphrase, err := credentials.ReadPassphrase()
	if err != nil {
		return err
	}
	client, err := credentials.New()
	if err != nil {
		return err
	}
	entries, err := client.List(context.Background())
	if err != nil {
		return err
	}
	configured, err := configuredEntries(context.Background(), client, entries)
	if err != nil {
		return err
	}
	if len(configured) == 0 {
		return fmt.Errorf("backup export found no configured credentials")
	}
	response, err := client.RecoveryExport(context.Background(), credentialclient.RecoveryExportRequest{Entries: configured, Passphrase: passphrase, OutputPath: *output})
	if err != nil {
		return err
	}
	if _, err := client.RecoveryVerify(context.Background(), credentialclient.RecoveryVerifyRequest{InputPath: *output, Passphrase: passphrase}); err != nil {
		return fmt.Errorf("verify recovery bundle: %w", err)
	}
	return support.PrintMutation(jsonOutput, response, cliapp.MutationReport{Result: []string{"Recovery bundle exported and verified"}, NextCommand: []string{"keep the bundle and passphrase apart and off this host"}})
}

// configuredEntries mirrors the bootstrap recovery --all rule: declarations
// are inventory, while only values actually present in the authority belong
// in a recovery bundle. The status call is metadata-only and never retrieves a
// credential value.
func configuredEntries(ctx context.Context, client credentialclient.Client, entries []credentialclient.CredentialRef) ([]credentialclient.CredentialRef, error) {
	configured := make([]credentialclient.CredentialRef, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		key := entry.LogicalID + "\x00" + entry.Field
		if seen[key] {
			continue
		}
		seen[key] = true
		status, err := client.Status(ctx, entry.LogicalID, entry.Field)
		if err != nil {
			return nil, fmt.Errorf("check credential %s:%s: %w", entry.LogicalID, entry.Field, err)
		}
		if status.Configured {
			configured = append(configured, entry)
		}
	}
	return configured, nil
}

func runMigrate(args []string) error {
	fs := support.NewFlagSet("backup migrate")
	input := fs.String("input", "", "legacy plaintext JSON path")
	output := fs.String("output", "", "new recovery bundle path")
	jsonFlag, format := support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	jsonOutput, err := support.ResolveJSONOutput(*jsonFlag, *format)
	if err != nil {
		return err
	}
	if *input == "" || *output == "" {
		return fmt.Errorf("backup migrate requires --input and --output")
	}
	passphrase, err := credentials.ReadPassphrase()
	if err != nil {
		return err
	}
	client, err := credentials.New()
	if err != nil {
		return err
	}
	report, err := credentialclient.MigrateLegacyJSON(context.Background(), client, *input, *output, passphrase)
	if err != nil {
		return err
	}
	if len(report.Unmapped) > 0 {
		return support.PrintMutation(jsonOutput, report, cliapp.MutationReport{Result: []string{"Migration stopped; unmapped keys remain in the source"}})
	}
	return support.PrintMutation(jsonOutput, report, cliapp.MutationReport{Result: []string{"Legacy credentials migrated, exported, verified, and source deleted"}})
}
