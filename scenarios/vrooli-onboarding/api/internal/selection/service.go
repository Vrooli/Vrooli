// Package selection owns the transport-free service seam for onboarding
// selection operations. Wire projection stays in the handler package.
package selection

import (
	"context"

	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/servicecall"
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
	return servicecall.Invoke(s.ListScenarios != nil, func() (*selectionv1.ListScenariosResponse, error) { return s.ListScenarios(ctx) }, nil)
}
func (s Service) CoreSet(ctx context.Context, seed []string) (*selectionv1.GetCoreSetResponse, error) {
	return servicecall.Invoke(s.GetCoreSet != nil, func() (*selectionv1.GetCoreSetResponse, error) { return s.GetCoreSet(ctx, seed) }, nil)
}
func (s Service) Recommendation(ctx context.Context) (*selectionv1.GetRecommendationResponse, error) {
	return servicecall.Invoke(s.GetRecommendation != nil, func() (*selectionv1.GetRecommendationResponse, error) { return s.GetRecommendation(ctx) }, nil)
}
func (s Service) Accept(ctx context.Context, request *selectionv1.AcceptRecommendationRequest) (*selectionv1.AcceptRecommendationResponse, error) {
	return servicecall.Invoke(s.AcceptRecommendation != nil, func() (*selectionv1.AcceptRecommendationResponse, error) { return s.AcceptRecommendation(ctx, request) }, nil)
}
func (s Service) Closure(ctx context.Context) (*selectionv1.GetClosureResponse, error) {
	return servicecall.Invoke(s.GetClosure != nil, func() (*selectionv1.GetClosureResponse, error) { return s.GetClosure(ctx) }, nil)
}
func (s Service) Union(ctx context.Context) (*selectionv1.GetUnionResponse, error) {
	return servicecall.Invoke(s.GetUnion != nil, func() (*selectionv1.GetUnionResponse, error) { return s.GetUnion(ctx) }, nil)
}
func (s Service) Handoff(ctx context.Context, req *selectionv1.CreateHandoffRequest) (*selectionv1.CreateHandoffResponse, error) {
	return servicecall.Invoke(s.CreateHandoff != nil, func() (*selectionv1.CreateHandoffResponse, error) { return s.CreateHandoff(ctx, req) }, nil)
}

func (s Service) ResolveHandoff(ctx context.Context, req *selectionv1.GetHandoffRequest) (*selectionv1.GetHandoffResponse, error) {
	return servicecall.Invoke(s.GetHandoff != nil, func() (*selectionv1.GetHandoffResponse, error) { return s.GetHandoff(ctx, req) }, nil)
}
