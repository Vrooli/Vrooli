package scopes

import (
	scopesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes/scopesv1connect"
	"source-ledger/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "scopes_create", Path: scopesconnect.ScopesServiceCreateScopeProcedure, Method: "POST", Summary: "Create a memory scope", Category: "scopes"},
	{ID: "scopes_list", Path: scopesconnect.ScopesServiceListScopesProcedure, Method: "POST", Summary: "List memory scopes", Category: "scopes"},
	{ID: "scopes_policy_get", Path: scopesconnect.ScopesServiceGetPolicyProcedure, Method: "POST", Summary: "Read effective scope policy and compaction liveness", Category: "scopes"},
	{ID: "scopes_policy_set", Path: scopesconnect.ScopesServiceSetPolicyProcedure, Method: "POST", Summary: "Set scope policy overrides", Category: "scopes"},
	{ID: "scopes_policy_reset", Path: scopesconnect.ScopesServiceResetPolicyProcedure, Method: "POST", Summary: "Reset scope policy overrides", Category: "scopes"},
}
