package sessions

import (
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/sessions/sessions_v1connect"
	"program-runtime/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "sessions_create", Method: "POST", Path: sessionsconnect.SessionServiceCreateSessionProcedure, Summary: "Create a governed program session.", Category: "sessions"},
	{ID: "sessions_get", Method: "POST", Path: sessionsconnect.SessionServiceGetSessionProcedure, Summary: "Read a program session.", Category: "sessions"},
	{ID: "sessions_list", Method: "POST", Path: sessionsconnect.SessionServiceListSessionsProcedure, Summary: "List program sessions.", Category: "sessions"},
	{ID: "sessions_delete", Method: "POST", Path: sessionsconnect.SessionServiceDeleteSessionProcedure, Summary: "Delete a program session.", Category: "sessions"},
	{ID: "sessions_grant", Method: "POST", Path: sessionsconnect.SessionServiceGrantSessionProcedure, Summary: "Grant a session capability.", Category: "sessions"},
}
