package review

import (
	"context"

	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	"react-component-library/internal/designcapture"
	"react-component-library/internal/designcritique"
	sketchdomain "react-component-library/internal/sketch"
)

// AcceptanceEvaluator resolves candidate and render evidence from the Sketch
// domain. It is deliberately narrower than the candidate Connect handler.
type AcceptanceEvaluator interface {
	ValidateAcceptanceCandidate(context.Context, *reviewv1.CheckCandidateAcceptanceRequest) error
	EvaluateAcceptance(context.Context, *reviewv1.CheckCandidateAcceptanceRequest) (*reviewv1.CandidateAcceptanceCheck, error)
}

// AcceptanceStore preserves the sketch domain's prepared-intent idempotency
// protocol while keeping file persistence outside this HTTP handler.
type AcceptanceStore interface {
	RecordAcceptance(context.Context, string, string, sketchdomain.AcceptanceIntent, func(context.Context) (sketchdomain.AcceptanceFacts, error)) (sketchdomain.AcceptanceDecision, error)
	ReadAcceptance(string, string, string) (sketchdomain.AcceptanceDecision, error)
}

// Ports are narrow dependencies for the ReviewService's acceptance and visual
// critique operations. Findings still use the existing review Service.
type Ports struct {
	AcceptanceEvaluator AcceptanceEvaluator
	AcceptanceStoreFor  func(context.Context) (AcceptanceStore, error)
	CritiquesFor        func(context.Context) (designcritique.Repository, error)
	CapturesFor         func(context.Context) (designcapture.Repository, error)
	Screenshots         designcapture.ScreenshotResolver
	AcceptanceWrite     func(context.Context)
}
