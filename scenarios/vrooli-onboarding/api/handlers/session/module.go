package session

import (
	"connectrpc.com/connect"
	sessionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session/sessionv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/session"
)

func Module(service session.Service, targetInterceptors ...connect.Interceptor) module.Module {
	interceptors := []connect.Interceptor{authz.MutationInterceptor()}
	interceptors = append(interceptors, targetInterceptors...)
	path, handler := sessionconnect.NewSessionServiceHandler(NewConnectHandler(service), connect.WithInterceptors(interceptors...))
	return module.Connect("session", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "session_get", Path: sessionconnect.SessionServiceGetSessionProcedure, Method: "POST", Summary: "Read onboarding session", Description: "Returns the durable cross-surface wizard pointer.", Category: "session"},
	{ID: "session_advance", Path: sessionconnect.SessionServiceAdvanceSessionStepProcedure, Method: "POST", Summary: "Advance onboarding session", Description: "Persists the stable onboarding step id.", Category: "session", Request: &module.Schema{Type: "object", Properties: map[string]string{"step_id": "string (required)"}}},
	{ID: "session_model", Path: sessionconnect.SessionServiceGetStepModelProcedure, Method: "POST", Summary: "Read onboarding step model", Description: "Returns the stable cross-surface wizard step model.", Category: "session"},
	{ID: "session_draft_get", Path: sessionconnect.SessionServiceGetDraftProcedure, Method: "POST", Summary: "Read onboarding draft", Description: "Returns the target-scoped non-secret draft owned by the verified session; the legacy actor field is ignored.", Category: "session"},
	{ID: "session_draft_save", Path: sessionconnect.SessionServiceSaveDraftProcedure, Method: "POST", Summary: "Save onboarding draft", Description: "Persists non-secret choices without changing effective host state; ownership comes from verified session context.", Category: "session", Request: &module.Schema{Type: "object", Properties: map[string]string{"target": "string (required)", "actor": "ignored legacy field", "expected_revision": "string (recommended for concurrent clients)", "base_revision": "string (required)", "choices": "object"}}},
	{ID: "session_draft_discard", Path: sessionconnect.SessionServiceDiscardDraftProcedure, Method: "POST", Summary: "Discard onboarding draft", Description: "Removes the verified session's target-scoped draft without changing committed intent.", Category: "session"},
}
