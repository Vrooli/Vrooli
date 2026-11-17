package wallet

import (
	"treasury/internal/module"
	domain "treasury/internal/wallet"

	walletconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/wallet/wallet_v1connect"
)

func Schema() string { return domain.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "treasury_wallet_create", Path: walletconnect.WalletAdminServiceCreateResearchWalletProcedure, Method: "POST", Summary: "Create an operator-bound receive-only research wallet", Category: "wallet", Request: &module.Schema{Type: "CreateResearchWalletRequest"}, Response: &module.Schema{Type: "ResearchWalletResponse"}, Errors: walletErrors},
	{ID: "treasury_wallet_rotate", Path: walletconnect.WalletAdminServiceRotateResearchWalletProcedure, Method: "POST", Summary: "Deprecated one-shot wallet rotation; fails closed and directs callers to receipt-validated portable rotation", Category: "wallet", Request: &module.Schema{Type: "RotateResearchWalletRequest"}, Response: &module.Schema{Type: "ResearchWalletResponse"}, Errors: walletErrors},
	{ID: "treasury_wallet_get", Path: walletconnect.WalletAdminServiceGetResearchWalletProcedure, Method: "POST", Summary: "Read operator-bound wallet custody metadata", Category: "wallet", Request: &module.Schema{Type: "ResearchWalletRequest"}, Response: &module.Schema{Type: "ResearchWalletResponse"}, Errors: walletErrors},
	{ID: "treasury_wallet_prove", Path: walletconnect.WalletAdminServiceProveResearchWalletProcedure, Method: "POST", Summary: "Prove wallet address control without granting spending authority", Category: "wallet", Request: &module.Schema{Type: "ProveResearchWalletRequest"}, Response: &module.Schema{Type: "ProveResearchWalletResponse"}, Errors: walletErrors},
	{ID: "treasury_wallet_recovery_export", Path: walletconnect.WalletAdminServiceExportPortableResearchWalletRecoveryProcedure, Method: "POST", Summary: "Export an encrypted portable wallet recovery bundle", Category: "wallet", Request: &module.Schema{Type: "ExportPortableResearchWalletRecoveryRequest"}, Response: &module.Schema{Type: "ExportPortableResearchWalletRecoveryResponse"}, Errors: walletErrors},
	{ID: "treasury_wallet_recovery_restore", Path: walletconnect.WalletAdminServiceRestorePortableResearchWalletRecoveryProcedure, Method: "POST", Summary: "Restore a wallet from an encrypted portable recovery bundle", Category: "wallet", Request: &module.Schema{Type: "RestorePortableResearchWalletRecoveryRequest"}, Response: &module.Schema{Type: "ResearchWalletResponse"}, Errors: walletErrors},
	{ID: "treasury_wallet_rotation_prepare", Path: walletconnect.WalletAdminServicePreparePortableResearchWalletRotationProcedure, Method: "POST", Summary: "Prepare a signed rotation tombstone and retain the old scalar pending off-host verification", Category: "wallet", Request: &module.Schema{Type: "PreparePortableResearchWalletRotationRequest"}, Response: &module.Schema{Type: "PreparePortableResearchWalletRotationResponse"}, Errors: walletErrors},
	{ID: "treasury_wallet_rotation_commit", Path: walletconnect.WalletAdminServiceCommitPortableResearchWalletRotationProcedure, Method: "POST", Summary: "Commit wallet rotation after validating the newest off-host tombstone receipt", Category: "wallet", Request: &module.Schema{Type: "CommitPortableResearchWalletRotationRequest"}, Response: &module.Schema{Type: "CommitPortableResearchWalletRotationResponse"}, Errors: walletErrors},
	{ID: "treasury_wallet_recovery_qualify", Path: walletconnect.WalletAdminServiceQualifyResearchWalletRecoveryProcedure, Method: "POST", Summary: "Qualify same-host encrypted wallet recovery", Category: "wallet", Request: &module.Schema{Type: "ResearchWalletRequest"}, Response: &module.Schema{Type: "ResearchWalletResponse"}, Errors: walletErrors},
	{ID: "treasury_wallet_recovery_run", Path: walletconnect.WalletAdminServiceRecoverResearchWalletProcedure, Method: "POST", Summary: "Recover a missing wallet key from qualified custody", Category: "wallet", Request: &module.Schema{Type: "ResearchWalletRequest"}, Response: &module.Schema{Type: "ResearchWalletResponse"}, Errors: walletErrors},
	{ID: "treasury_wallet_receipt_verify", Path: walletconnect.WalletAdminServiceVerifyBaseUSDCReceiptProcedure, Method: "POST", Summary: "Verify an exact Base receipt without posting or allocating funds", Category: "wallet", Request: &module.Schema{Type: "VerifyBaseUSDCReceiptRequest"}, Response: &module.Schema{Type: "VerifyBaseUSDCReceiptResponse"}, Errors: walletErrors},
	{ID: "treasury_wallet_receipt_recheck", Path: walletconnect.WalletAdminServiceRecheckBaseUSDCReceiptProcedure, Method: "POST", Summary: "Recheck a previously retained Base receipt observation", Category: "wallet", Request: &module.Schema{Type: "RecheckBaseUSDCReceiptRequest"}, Response: &module.Schema{Type: "RecheckBaseUSDCReceiptResponse"}, Errors: walletErrors},
	{ID: "treasury_wallet_receipt_history", Path: walletconnect.WalletAdminServiceListBaseUSDCReceiptChecksProcedure, Method: "POST", Summary: "Read durable receipt check history bound to a wallet and transaction", Category: "wallet", Request: &module.Schema{Type: "ListBaseUSDCReceiptChecksRequest"}, Response: &module.Schema{Type: "ListBaseUSDCReceiptChecksResponse"}, Errors: walletErrors},
}

var walletErrors = []module.ErrorDesc{
	{Status: 401, Code: "unauthenticated", Description: "Operator identity is required"},
	{Status: 403, Code: "permission_denied", Description: "Agent-realm or another operator's wallet"},
	{Status: 412, Code: "failed_precondition", Description: "Operator realm or wallet service is unavailable"},
}
