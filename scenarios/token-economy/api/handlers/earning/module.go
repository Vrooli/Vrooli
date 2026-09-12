// Package earning exposes the inbound earning contract at the transport edge.
package earning

import (
	"token-economy/internal/earning"
	"token-economy/internal/module"

	earningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/earning/earningv1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID: "earning_submit", Path: earningconnect.EarningServiceSubmitEarningProcedure,
		Method: "POST", Summary: "Submit earned tokens",
		Description: "Admits work through the shared adapter contract with adapter-scoped replay protection.",
		Category:    "earning",
	},
	{
		ID: "earning_list", Path: earningconnect.EarningServiceListEarningsProcedure,
		Method: "POST", Summary: "List earning submissions",
		Description: "Lists durable dedup outcomes and privacy-minimized payload summaries.",
		Category:    "earning",
	},
}

// Schema re-exports the domain-owned schema for the central boot registry.
func Schema() string { return earning.Schema() }
