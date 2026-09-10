package profiles

import (
	"context"
	"encoding/json"
	"errors"

	"connectrpc.com/connect"
	profilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/profiles"
	profilesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/profiles/profilesv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
	profilesdomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/profiles"
	"google.golang.org/protobuf/types/known/structpb"
)

type ModuleService struct{ Profiles profilesdomain.Service }

func Module(service profilesdomain.Service, targetInterceptors ...connect.Interceptor) module.Module {
	path, handler := profilesconnect.NewProfileServiceHandler(&connectHandler{service: service}, connect.WithInterceptors(targetInterceptors...))
	return module.Connect("profiles", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "profiles_list", Path: profilesconnect.ProfileServiceListProfilesProcedure, Method: "POST", Summary: "List supported onboarding profiles", Category: "selection"},
	{ID: "profiles_evaluate", Path: profilesconnect.ProfileServiceEvaluateProfileProcedure, Method: "POST", Summary: "Evaluate a declarative onboarding profile", Category: "selection"},
}

type connectHandler struct{ service profilesdomain.Service }

func (h *connectHandler) ListProfiles(ctx context.Context, _ *connect.Request[profilesv1.ListProfilesRequest]) (*connect.Response[profilesv1.ListProfilesResponse], error) {
	items, err := h.service.List(ctx)
	if err != nil {
		return nil, profilesError(err)
	}
	result := make([]*profilesv1.Profile, 0, len(items))
	for _, item := range items {
		result = append(result, toProtoProfile(item))
	}
	return connect.NewResponse(&profilesv1.ListProfilesResponse{Profiles: result}), nil
}

func (h *connectHandler) EvaluateProfile(ctx context.Context, request *connect.Request[profilesv1.EvaluateProfileRequest]) (*connect.Response[profilesv1.EvaluateProfileResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("profile evaluation request is required"))
	}
	answers := map[string]any{}
	for key, value := range request.Msg.GetAnswers() {
		answers[key] = value.AsInterface()
	}
	var targetContext map[string]any
	if request.Msg.GetTargetContext() != nil {
		targetContext = request.Msg.GetTargetContext().AsMap()
	}
	manualDecisions := make(map[string]bool, len(request.Msg.GetManualDecisions()))
	for key, selected := range request.Msg.GetManualDecisions() {
		manualDecisions[key] = selected
	}
	result, err := h.service.EvaluateWithManualDecisions(ctx, request.Msg.GetProfileId(), answers, targetContext, manualDecisions)
	if err != nil {
		return nil, profilesError(err)
	}
	response := &profilesv1.EvaluateProfileResponse{Profile: toProtoProfile(result.Profile), Scenarios: result.Scenarios, Resources: result.Resources, Valid: result.Valid, Digest: result.Digest}
	for _, question := range result.Questions {
		response.Questions = append(response.Questions, toProtoQuestion(question))
	}
	for _, recommendation := range result.Recommendations {
		response.Recommendations = append(response.Recommendations, &profilesv1.ProfileRecommendation{CapabilityRef: recommendation.CapabilityRef, ScenarioRefs: recommendation.ScenarioRefs, ReasonKey: recommendation.ReasonKey, RuleId: recommendation.RuleID, Key: recommendation.Key, Selected: recommendation.Selected, Required: recommendation.Required})
	}
	for _, explanation := range result.Explanations {
		response.Explanations = append(response.Explanations, &profilesv1.ProfileExplanation{RuleId: explanation.RuleID, CapabilityRef: explanation.CapabilityRef, ScenarioRefs: explanation.ScenarioRefs, ReasonKey: explanation.ReasonKey, Selected: explanation.Selected})
	}
	for _, issue := range result.Issues {
		response.Issues = append(response.Issues, &profilesv1.ProfileValidationIssue{Field: issue.Field, Code: issue.Code, Message: issue.Message})
	}
	return connect.NewResponse(response), nil
}

func toProtoProfile(item profilesdomain.ProfileSummary) *profilesv1.Profile {
	return &profilesv1.Profile{Id: item.ID, Version: item.Version, Default: item.Default, TitleKey: item.TitleKey, DescriptionKey: item.DescriptionKey, Owner: item.Owner, ProvenanceSource: item.ProvenanceSource, ProvenanceRevision: item.ProvenanceRevision, SchemaVersion: item.SchemaVersion, CompatibleCatalogMajor: int32(item.CompatibleCatalogMajor), ManualSelectionAvailable: item.ManualSelectionAvailable}
}
func toProtoQuestion(item profilesdomain.QuestionView) *profilesv1.ProfileQuestion {
	result := &profilesv1.ProfileQuestion{Id: item.ID, Type: item.Type, PromptKey: item.PromptKey, Required: item.Required, MinSelections: int32(item.MinSelections), MaxSelections: int32(item.MaxSelections), Visible: item.Visible}
	if len(item.Default) > 0 {
		var value any
		if err := json.Unmarshal(item.Default, &value); err == nil {
			if converted, err := structpb.NewValue(value); err == nil {
				result.DefaultValue = converted
			}
		}
	}
	for _, option := range item.Options {
		result.Options = append(result.Options, &profilesv1.ProfileOption{Id: option.ID, LabelKey: option.LabelKey})
	}
	return result
}
