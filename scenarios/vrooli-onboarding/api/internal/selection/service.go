// Package selection owns the transport-free service seam for onboarding
// selection operations. Wire projection stays in the handler package.
package selection

import (
	"context"

	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
)

type Service struct {
	ListScenarios        func(context.Context) (*selectionv1.ListScenariosResponse, error)
	GetCoreSet           func(context.Context, []string) (*selectionv1.GetCoreSetResponse, error)
	GetRecommendation    func(context.Context) (*selectionv1.GetRecommendationResponse, error)
	AcceptRecommendation func(context.Context, *selectionv1.AcceptRecommendationRequest) (*selectionv1.AcceptRecommendationResponse, error)
	GetClosure           func(context.Context) (*selectionv1.GetClosureResponse, error)
	GetUnion             func(context.Context) (*selectionv1.GetUnionResponse, error)
	CreateHandoff        func(context.Context, *selectionv1.CreateHandoffRequest) (*selectionv1.CreateHandoffResponse, error)
	GetHandoff           func(context.Context, *selectionv1.GetHandoffRequest) (*selectionv1.GetHandoffResponse, error)
}

func (s Service) Scenarios(ctx context.Context) (*selectionv1.ListScenariosResponse, error) {
	if s.ListScenarios == nil {
		return nil, context.Canceled
	}
	return s.ListScenarios(ctx)
}
func (s Service) CoreSet(ctx context.Context, seed []string) (*selectionv1.GetCoreSetResponse, error) {
	if s.GetCoreSet == nil {
		return nil, context.Canceled
	}
	return s.GetCoreSet(ctx, seed)
}
func (s Service) Recommendation(ctx context.Context) (*selectionv1.GetRecommendationResponse, error) {
	if s.GetRecommendation == nil {
		return nil, context.Canceled
	}
	return s.GetRecommendation(ctx)
}
func (s Service) Accept(ctx context.Context, request *selectionv1.AcceptRecommendationRequest) (*selectionv1.AcceptRecommendationResponse, error) {
	if s.AcceptRecommendation == nil {
		return nil, context.Canceled
	}
	return s.AcceptRecommendation(ctx, request)
}
func (s Service) Closure(ctx context.Context) (*selectionv1.GetClosureResponse, error) {
	if s.GetClosure == nil {
		return nil, context.Canceled
	}
	return s.GetClosure(ctx)
}
func (s Service) Union(ctx context.Context) (*selectionv1.GetUnionResponse, error) {
	if s.GetUnion == nil {
		return nil, context.Canceled
	}
	return s.GetUnion(ctx)
}
func (s Service) Handoff(ctx context.Context, req *selectionv1.CreateHandoffRequest) (*selectionv1.CreateHandoffResponse, error) {
	if s.CreateHandoff == nil {
		return nil, context.Canceled
	}
	return s.CreateHandoff(ctx, req)
}

func (s Service) ResolveHandoff(ctx context.Context, req *selectionv1.GetHandoffRequest) (*selectionv1.GetHandoffResponse, error) {
	if s.GetHandoff == nil {
		return nil, context.Canceled
	}
	return s.GetHandoff(ctx, req)
}
