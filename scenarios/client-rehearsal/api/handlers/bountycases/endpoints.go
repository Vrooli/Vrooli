package bountycases

import (
	"client-rehearsal/internal/module"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/bountycases/bountycases_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "bounty_cases_resolve", Path: connect.BountyCaseServiceResolveBountyReplayCaseProcedure, Method: "POST", Summary: "Resolve a registered bounty replay case", Description: "Resolves one immutable registered issue or case ID for the pinned bounty replay owner flow; returns hidden evaluator and reference roots, and is not for general CLI access.", Category: "bounty-cases"},
	{ID: "bounty_cases_evaluator_bundle", Path: connect.BountyCaseServiceGetFrozenEvaluatorBundleProcedure, Method: "POST", Summary: "Resolve frozen evaluator bytes by opaque bundle ID", Description: "Returns only the content-addressed evaluator file manifest for the internal Test Genie owner path, never a host root.", Category: "bounty-cases"},
}
