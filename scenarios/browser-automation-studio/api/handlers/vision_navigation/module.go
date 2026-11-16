// Package vision_navigation hosts the BAS VisionNavigationService Connect-RPC
// handler. It wraps the existing services/vision navigator registry and
// Playwright session tracker; AI metering and privacy enforcement are owned by
// AI Gateway.
//
// The complementary /api/v1/internal/ai-navigate/callback webhook (POST from
// the playwright driver) stays on chi as a documented REST exception (webhook
// receiver). See docs/internal/REST_EXCEPTIONS.md.
package vision_navigation

import (
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai/aiconnect"

	"github.com/vrooli/browser-automation-studio/services/vision"
)

// SessionTracker is the narrow seam over the navigators that the handler
// depends on for session-state lookups (status/wait/abort/resume). Tests
// inject a fake; production passes the Playwright navigator tracker that owns navigation state. GetSession must return a snapshot whose Changed()
// channel closes on the next status transition (see vision.NavigationSession).
type SessionTracker = vision.SessionTracker

// Deps wires the vision-navigation handler. Logger is required. Registry is
// required (no registry → no navigators → handler can't function). Tracker
// is optional; without it status / abort / resume return NotFound.
//
// CallbackBase, when non-empty, is the scheme://host base URL used to build
// the callback URL sent to playwright-driver (e.g. "http://127.0.0.1:8110").
// When empty the handler falls back to the request Host header at call time.
type Deps struct {
	Logger              *logrus.Logger
	Registry            *vision.NavigatorRegistry
	Tracker             SessionTracker
	CallbackBase        string
	CredentialAuthority *credentialauthority.Authority
}

// Module builds the VisionNavigationService Connect handler and returns it
// wrapped in a connectx.ServiceMount ready for connectx.RegisterChi.
//
// Missing required deps panic at boot so a forgotten wire-up surfaces at
// startup, not at first request.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("vision_navigation.Module requires Deps.Logger")
	}
	if d.Registry == nil {
		panic("vision_navigation.Module requires Deps.Registry")
	}
	path, handler := aiconnect.NewVisionNavigationServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}
