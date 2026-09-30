// Package ai_service hosts the BAS AIService Connect-RPC handler.
//
// AIService groups the ephemeral, single-shot AI/automation helpers:
// preview screenshot, link preview, element analysis, element at coordinate,
// AI-element analysis (Ollama text model), and DOM tree extraction. The
// stateful AI vision-navigation surface lives on a separate service.
//
// This package is a thin Connect adapter onto the transport-agnostic methods
// already exposed by api/handlers/ai (ScreenshotHandler/DOMHandler/
// ElementAnalysisHandler/AIAnalysisHandler) and the FetchLinkPreview helper
// in api/handlers. We deliberately reuse those types rather than reimplement
// the business logic.
package ai_service

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai/aiconnect"
)

// LinkPreviewData is the transport-agnostic shape of an OpenGraph preview.
// Kept local to avoid an import cycle with the parent handlers package; the
// adapter constructed in main.go copies fields out of the source struct.
type LinkPreviewData struct {
	Title       string
	Description string
	Image       string
	Favicon     string
	SiteName    string
}

// LinkPreviewFetcher is the narrow seam over handlers.FetchLinkPreview.
// Returns (preview, found, err) — found=false maps to "no metadata".
type LinkPreviewFetcher func(ctx context.Context, url string) (*LinkPreviewData, bool, error)

// Deps wires the AIService handler.
//
// Each handler reference is required: AIService has no graceful degradation
// because the underlying sub-handlers always exist when the Handler is wired
// (see api/handlers/handler.go).
type Deps struct {
	Screenshot      *aihandlers.ScreenshotHandler
	DOM             *aihandlers.DOMHandler
	ElementAnalysis *aihandlers.ElementAnalysisHandler
	AIAnalysis      *aihandlers.AIAnalysisHandler
	LinkPreview     LinkPreviewFetcher
	Logger          *logrus.Logger
}

// Module builds the AIService Connect handler.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("ai_service.Module requires Deps.Logger")
	}
	if d.Screenshot == nil {
		panic("ai_service.Module requires Deps.Screenshot")
	}
	if d.DOM == nil {
		panic("ai_service.Module requires Deps.DOM")
	}
	if d.ElementAnalysis == nil {
		panic("ai_service.Module requires Deps.ElementAnalysis")
	}
	if d.AIAnalysis == nil {
		panic("ai_service.Module requires Deps.AIAnalysis")
	}
	if d.LinkPreview == nil {
		panic("ai_service.Module requires Deps.LinkPreview")
	}
	path, handler := aiconnect.NewAIServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

var _ aiconnect.AIServiceHandler = (*service)(nil)
