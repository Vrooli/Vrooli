package credentialgrant

import (
	grantconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant/credentialgrant_v1connect"
	"vrooli-bridge/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "credentialgrant_create", Path: grantconnect.CredentialGrantServiceCreateGrantProcedure, Method: "POST", Summary: "Create a credential grant", Description: "Owner-gated metadata-only grant authoring; credential values are never accepted.", Category: "credential-grants"},
	{ID: "credentialgrant_list", Path: grantconnect.CredentialGrantServiceListGrantsProcedure, Method: "POST", Summary: "List credential grants", Description: "Owner-gated grant inventory with generation receipts.", Category: "credential-grants"},
	{ID: "credentialgrant_revoke", Path: grantconnect.CredentialGrantServiceRevokeGrantProcedure, Method: "POST", Summary: "Revoke a credential grant", Description: "Owner-gated grant revocation.", Category: "credential-grants"},
	{ID: "credentialgrant_rotate", Path: grantconnect.CredentialGrantServiceRotateAddressProcedure, Method: "POST", Summary: "Rotate a granted address", Description: "Advances the address generation and fans out sealed values to active grants.", Category: "credential-grants"},
	{ID: "credentialgrant_sync_node", Path: grantconnect.CredentialGrantServiceSyncNodeGrantsProcedure, Method: "POST", Summary: "Reconcile node credential grants", Description: "Node-facing authenticated reconciliation; metadata response plus sealed current-value delivery.", Category: "credential-grants"},
}

func Schema() string { return "" }
