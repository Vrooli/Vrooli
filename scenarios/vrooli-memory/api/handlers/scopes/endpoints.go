package scopes

import (
	scopesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/scopes/scopesv1connect"
	"vrooli-memory/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "scopes_create", Path: scopesconnect.ScopesServiceCreateScopeProcedure, Method: "POST", Summary: "Create a memory scope", Category: "scopes"},
	{ID: "scopes_list", Path: scopesconnect.ScopesServiceListScopesProcedure, Method: "POST", Summary: "List memory scopes", Category: "scopes"},
}
