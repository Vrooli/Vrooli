package personas

import (
	personasconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/personas/personas_v1connect"
	"persona/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "personas_create", Path: personasconnect.PersonasServiceCreatePersonaProcedure, Method: "POST", Summary: "Create a persona", Category: "personas"},
	{ID: "personas_get", Path: personasconnect.PersonasServiceGetPersonaProcedure, Method: "POST", Summary: "Read a persona", Category: "personas"},
	{ID: "personas_list", Path: personasconnect.PersonasServiceListPersonasProcedure, Method: "POST", Summary: "List personas", Category: "personas"},
	{ID: "personas_archive", Path: personasconnect.PersonasServiceArchivePersonaProcedure, Method: "POST", Summary: "Archive a persona", Category: "personas"},
	{ID: "personas_health", Path: personasconnect.PersonasServiceCheckHealthProcedure, Method: "POST", Summary: "Check persona readiness", Category: "personas"},
}
