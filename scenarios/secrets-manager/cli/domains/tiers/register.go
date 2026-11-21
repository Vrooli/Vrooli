package tiers

import (
	"context"
	"fmt"

	"secrets-manager/cli/internal/credentials"
	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

type tierReport struct {
	Tier           string   `json:"tier"`
	Provider       string   `json:"provider"`
	ProviderState  string   `json:"provider_state"`
	ReceiptExists  bool     `json:"receipt_exists"`
	CoveredEntries int      `json:"covered_entries"`
	Uncovered      []string `json:"uncovered"`
}

func Register(_ *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "tiers",
		Description: "Report credential posture by deployment tier",
		Subcommands: []cliapp.Command{
			{Name: "status", Description: "Show metadata-only host and recovery posture", Run: runStatus},
		},
	}
}

func runStatus(args []string) error {
	fs := support.NewFlagSet("tiers status")
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
	report := tierReport{Tier: "tier-1-local", Provider: diagnosis.Provider.Backend, ProviderState: diagnosis.Provider.Condition, ReceiptExists: diagnosis.Recovery.ReceiptExists, CoveredEntries: diagnosis.Recovery.EntryCount, Uncovered: diagnosis.Recovery.Uncovered}
	return support.PrintOperational(jsonOutput, report, cliapp.OperationalReport{Status: []string{fmt.Sprintf("%s provider=%s condition=%s", report.Tier, report.Provider, report.ProviderState), fmt.Sprintf("recovery_receipt=%t covered=%d uncovered=%d", report.ReceiptExists, report.CoveredEntries, len(report.Uncovered))}})
}
