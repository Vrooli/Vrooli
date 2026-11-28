package scopes

import (
	"vrooli-memory/internal/module"

	scopesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/scopes/scopesv1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "scopes_create", Path: scopesconnect.ScopesServiceCreateScopeProcedure, Method: "POST", Summary: "Create a memory scope", Category: "scopes"},
	{ID: "scopes_list", Path: scopesconnect.ScopesServiceListScopesProcedure, Method: "POST", Summary: "List memory scopes", Category: "scopes"},
}
