package glossary

import (
	glossaryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/glossary/glossaryv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
)

func Module() module.Module {
	path, handler := glossaryconnect.NewGlossaryServiceHandler(NewConnectHandler())
	return module.Connect("glossary", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "glossary_search", Path: glossaryconnect.GlossaryServiceSearchGlossaryProcedure, Method: "POST", Summary: "Search onboarding glossary", Category: "documentation"},
	{ID: "configuration_search", Path: glossaryconnect.GlossaryServiceSearchConfigurationProcedure, Method: "POST", Summary: "Search safe onboarding configuration metadata", Category: "configuration"},
}
