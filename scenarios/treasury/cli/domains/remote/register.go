// Package remote binds Treasury workflow groups directly to generated Connect
// descriptors. It contains no request structs or transport-specific DTOs.
package remote

import (
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/instrument"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/ledger"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/wallet"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"treasury/cli/domains/catalog"
)

const (
	agentService        = "vrooli.treasury.v1.authorization.AgentSpend"
	adminService        = "vrooli.treasury.v1.authorization.TreasuryAdmin"
	budgetService       = "vrooli.treasury.v1.budget.BudgetAdminService"
	bookService         = "vrooli.treasury.v1.book.BookAdminService"
	instrumentService   = "vrooli.treasury.v1.instrument.InstrumentAdminService"
	mandateService      = "vrooli.treasury.v1.mandate.MandateAdminService"
	walletService       = "vrooli.treasury.v1.wallet.WalletAdminService"
	fundingService      = "vrooli.treasury.v1.funding.FundingAdminService"
	agentFundingService = "vrooli.treasury.v1.funding.FundingAgentService"
	ledgerService       = "vrooli.treasury.v1.ledger.LedgerService"
)

type serviceSpec struct {
	name    protoreflect.FullName
	methods []string
}

var groups = map[string][]serviceSpec{
	"ledger": {{name: ledgerService, methods: []string{"LedgerService.ListEmissions", "LedgerService.GetLedgerDestination", "LedgerService.SetLedgerDestination"}}},
	"wallet": {
		{name: walletService, methods: []string{"WalletAdminService.CreateResearchWallet", "WalletAdminService.RotateResearchWallet", "WalletAdminService.GetResearchWallet", "WalletAdminService.VerifyBaseUSDCReceipt", "WalletAdminService.RecheckBaseUSDCReceipt", "WalletAdminService.ListBaseUSDCReceiptChecks", "WalletAdminService.ProveResearchWallet", "WalletAdminService.QualifyResearchWalletRecovery", "WalletAdminService.RecoverResearchWallet", "WalletAdminService.ExportPortableResearchWalletRecovery", "WalletAdminService.RestorePortableResearchWalletRecovery", "WalletAdminService.PreparePortableResearchWalletRotation", "WalletAdminService.CommitPortableResearchWalletRotation"}},
		{name: fundingService, methods: []string{"FundingAdminService.CreateFundingIntent", "FundingAdminService.GetFundingIntent", "FundingAdminService.ConfirmFundingTransfer", "FundingAdminService.ListFundingIntents", "FundingAdminService.SubmitReceiptFunding", "FundingAdminService.GetReceiptFunding", "FundingAdminService.ProvisionBudgetWallet", "FundingAdminService.GetBudgetWallet", "FundingAdminService.SetWithdrawalDestination", "FundingAdminService.PreviewWithdrawal", "FundingAdminService.ExecuteWithdrawal", "FundingAdminService.GetWithdrawal"}},
	},
	"book":          {{name: bookService, methods: []string{"BookAdminService.CreateBook", "BookAdminService.GetBook"}}},
	"authorization": {{name: agentService, methods: []string{"AgentSpend.ProposeCharge", "AgentSpend.GetAuthorization"}}},
	"mandate": {
		{name: agentService, methods: []string{"AgentSpend.ListMandates"}},
		{name: mandateService, methods: []string{"MandateAdminService.CreateMandate", "MandateAdminService.RevokeMandate", "MandateAdminService.CancelStandingMandate", "MandateAdminService.RevokeMandateSession", "MandateAdminService.ListMandates"}},
	},
	"budget": {
		{name: agentService, methods: []string{"AgentSpend.GetBudgetHeadroom"}},
		{name: agentFundingService, methods: []string{"FundingAgentService.GetBudgetWallet"}},
		{name: budgetService, methods: []string{"BudgetAdminService.SetBudgetCaps", "BudgetAdminService.SetGating", "BudgetAdminService.FreezeBudget", "BudgetAdminService.UnfreezeBudget", "BudgetAdminService.FreezeBook", "BudgetAdminService.UnfreezeBook", "BudgetAdminService.GetFreezeStatus", "BudgetAdminService.FreezeAll", "BudgetAdminService.UnfreezeAll"}},
	},
	"approval":   {{name: adminService, methods: []string{"TreasuryAdmin.ListApprovals", "TreasuryAdmin.ResolveApproval"}}},
	"instrument": {{name: instrumentService, methods: []string{"InstrumentAdminService.RegisterInstrument"}}},
	"settlement": {
		{name: agentService, methods: []string{"AgentSpend.ReportOutcome"}},
		{name: adminService, methods: []string{"TreasuryAdmin.ReportManualOutcome"}},
	},
}

var readMethods = map[string]bool{
	"FundingAdminService.GetReceiptFunding":        true,
	"WalletAdminService.GetResearchWallet":         true,
	"WalletAdminService.VerifyBaseUSDCReceipt":     true,
	"WalletAdminService.RecheckBaseUSDCReceipt":    true,
	"WalletAdminService.ListBaseUSDCReceiptChecks": true,
	"AgentSpend.GetAuthorization":                  true,
	"AgentSpend.GetBudgetHeadroom":                 true,
	"AgentSpend.ListMandates":                      true,
	"BookAdminService.GetBook":                     true,
	"TreasuryAdmin.ListApprovals":                  true,
	"MandateAdminService.ListMandates":             true,
	"BudgetAdminService.GetFreezeStatus":           true,
	"FundingAgentService.GetBudgetWallet":          true,
	"FundingAdminService.GetBudgetWallet":          true,
	"FundingAdminService.GetFundingIntent":         true,
	"FundingAdminService.ListFundingIntents":       true,
	"FundingAdminService.GetWithdrawal":            true,
	"FundingAdminService.PreviewWithdrawal":        true,
	"LedgerService.ListEmissions":                  true,
	"LedgerService.GetLedgerDestination":           true,
}

func Register(core *cliapp.ScenarioApp, manifest []byte, groupName string) (cliapp.SubcommandGroup, error) {
	specs, ok := groups[groupName]
	if !ok {
		return cliapp.SubcommandGroup{}, fmt.Errorf("unknown Treasury remote group %q", groupName)
	}
	selected := make(map[string]func(cliapp.RunContext) error)
	if groupName == "ledger" {
		selected["describe"] = catalog.Describe(groupName)
	}
	for _, spec := range specs {
		renderers := make(map[string]cliapp.Renderer, len(spec.methods))
		for _, method := range spec.methods {
			if readMethods[method] {
				renderers[method] = renderRead(method)
			} else {
				renderers[method] = renderMutation(method)
			}
		}
		available, err := cliapp.ProtoBindings(core, spec.name, cliapp.ProtoBindingOptions{
			Render: renderers, Prepare: portableRecoveryRequestPreparer(treasurySessionRequestHeaders(core, ProductionDependencies()), os.Stdin),
		})
		if err != nil {
			return cliapp.SubcommandGroup{}, fmt.Errorf("%s: build proto bindings: %w", groupName, err)
		}
		for _, method := range spec.methods {
			handler, exists := available[method]
			if !exists {
				return cliapp.SubcommandGroup{}, fmt.Errorf("%s: descriptor method %s unavailable", groupName, method)
			}
			selected[method] = handler
		}
	}
	group, err := cliapp.LoadFromManifest(manifest, groupName, selected)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load manifest: %w", groupName, err)
	}
	return group, nil
}

func renderRead(method string) cliapp.Renderer {
	return func(ctx cliapp.RunContext, response proto.Message) error {
		return cliapp.RenderProtoList(ctx, response, cliapp.ListReport{
			Summary: []string{method + " completed."}, ResultsHeading: "Typed response",
			Results: []string{"Use --json for the complete generated protobuf response."},
		})
	}
}

func renderMutation(method string) cliapp.Renderer {
	return func(ctx cliapp.RunContext, response proto.Message) error {
		return cliapp.RenderProtoMutation(ctx, response, cliapp.MutationReport{
			Result:      []string{method + " completed."},
			Changes:     []string{"Treasury accepted the typed mutation and returned its authoritative record."},
			NextCommand: []string{"Use the corresponding read command with --json to inspect authoritative state."},
		})
	}
}
