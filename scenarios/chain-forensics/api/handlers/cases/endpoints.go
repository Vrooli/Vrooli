package cases

import (
	"chain-forensics/internal/module"
	casesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases/cases_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "cases_list", Path: casesconnect.CasesServiceListCasesProcedure, Method: "POST", Summary: "List forensic cases", Category: "cases"},
	{ID: "cases_import", Path: casesconnect.CasesServiceImportCaseProcedure, Method: "POST", Summary: "Import validated evidence case", Category: "cases"},
	{ID: "cases_get", Path: casesconnect.CasesServiceGetCaseProcedure, Method: "POST", Summary: "Get forensic case", Category: "cases"},
	{ID: "cases_private_packet", Path: casesconnect.CasesServiceExportPrivatePacketProcedure, Method: "POST", Summary: "Export a private packet locally", Category: "cases"},
	{ID: "cases_add_label", Path: casesconnect.CasesServiceAddAnalystLabelProcedure, Method: "POST", Summary: "Add analyst label separately from chain evidence", Category: "cases"},
	{ID: "cases_list_labels", Path: casesconnect.CasesServiceListAnalystLabelsProcedure, Method: "POST", Summary: "List analyst labels", Category: "cases"},
	{ID: "cases_capture_evm", Path: casesconnect.CasesServiceCaptureEVMTransactionProcedure, Method: "POST", Summary: "Capture a public Ethereum, Base or Arbitrum transaction and transfer evidence into a case", Category: "cases"},
	{ID: "cases_capture_bitcoin", Path: casesconnect.CasesServiceCaptureBitcoinTransactionProcedure, Method: "POST", Summary: "Capture a confirmed public Bitcoin transaction into a case", Category: "cases"},
	{ID: "cases_capture_bitcoin_address_page", Path: casesconnect.CasesServiceCaptureBitcoinAddressPageProcedure, Method: "POST", Summary: "Capture one bounded confirmed Bitcoin address-history page", Category: "cases"},
	{ID: "cases_capture_evm_wallet_snapshot", Path: casesconnect.CasesServiceCaptureEVMWalletSnapshotProcedure, Method: "POST", Summary: "Capture native and selected ERC-20 balances at a pinned EVM block", Category: "cases"},
	{ID: "cases_capture_solana_address_page", Path: casesconnect.CasesServiceCaptureSolanaAddressPageProcedure, Method: "POST", Summary: "Capture one bounded finalized Solana signature page and SPL balance deltas", Category: "cases"},
	{ID: "cases_revise", Path: casesconnect.CasesServiceReviseCaseProcedure, Method: "POST", Summary: "Append typed case evidence with optimistic concurrency", Category: "cases"},
	{ID: "cases_revision_history", Path: casesconnect.CasesServiceListCaseRevisionsProcedure, Method: "POST", Summary: "List immutable case revision receipts", Category: "cases"},
}
