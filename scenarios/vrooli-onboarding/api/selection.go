package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	apicoreset "github.com/vrooli/api-core/coreset"
	"github.com/vrooli/vrooli/internal/app/supervision"
	"github.com/vrooli/vrooli/internal/operatorstate"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	resourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/resources"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	selectiondomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/selection"
)

func validateSelectionContract(selection *setupv1.Selection) error {
	if selection == nil {
		return fmt.Errorf("setup selection is required")
	}
	if version := strings.TrimSpace(selection.GetSchemaVersion()); version != "" && version != "v1" {
		return fmt.Errorf("unsupported setup selection schema version %q; supported version is %q", version, "v1")
	}
	return nil
}

func (s *Server) selectionService() selectiondomain.Service {
	return selectiondomain.Service{
		ListScenarios:        s.listScenarios,
		GetCoreSet:           s.getCoreSet,
		GetRecommendation:    s.getRecommendation,
		AcceptRecommendation: s.acceptRecommendation,
		GetClosure:           s.getClosure,
		GetUnion:             s.getUnion,
		CreateHandoff:        s.createHandoff,
	}
}

func (s *Server) listScenarios(ctx context.Context) (*selectionv1.ListScenariosResponse, error) {
	models, err := loadScenarioReadModels()
	if err != nil {
		return nil, err
	}
	result := make([]*selectionv1.Scenario, 0, len(models))
	for _, model := range models {
		result = append(result, &selectionv1.Scenario{Name: model.Name, Description: model.Description, SystemRequired: model.SystemRequired, Enabled: model.Enabled, AutoRestart: model.AutoRestart, Resources: model.Resources})
	}
	return &selectionv1.ListScenariosResponse{Scenarios: result, Count: int32(len(result))}, nil
}

func (s *Server) getCoreSet(ctx context.Context, seed []string) (*selectionv1.GetCoreSetResponse, error) {
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return nil, err
	}
	if state.Core == nil {
		return nil, fmt.Errorf("operator state has no core authority")
	}
	if len(seed) == 0 {
		seed = state.Core.Seed
	}
	authority := apicoreset.NormalizeOperationalAuthority(apicoreset.Authority{Seed: normalizeCoreSeed(seed), TrustedBase: append([]string(nil), state.Core.TrustedBase...)})
	if err := authority.Validate(); err != nil {
		return nil, fmt.Errorf("core authority validation failed: %w", err)
	}
	response := &selectionv1.GetCoreSetResponse{Seed: authority.Seed, TrustedBase: authority.TrustedBase}
	root := strings.TrimSpace(s.roots.RepoRoot)
	if root == "" {
		response.Error = "closure unavailable outside a repository source tree; the operator seed remains authoritative"
		return response, nil
	}
	report := supervision.Compute(root+"/scenarios", authority)
	response.Available, response.MemberCounts, response.LoadErrors = true, map[string]int32{}, map[string]string{}
	for key, value := range report.MemberCounts {
		response.MemberCounts[key] = int32(value)
	}
	for key, value := range report.LoadErrors {
		response.LoadErrors[key] = value
	}
	for _, member := range report.Members {
		item := &selectionv1.SupervisionMember{Name: member.Name, Kind: member.Kind, SupervisionIntent: member.SupervisionIntent}
		for _, step := range member.AttributionChain {
			item.AttributionChain = append(item.AttributionChain, &selectionv1.SupervisionAttributionStep{Name: step.Name, Kind: step.Kind, DeclaredBy: step.DeclaredBy, SupervisionIntent: step.SupervisionIntent, Source: step.Source})
		}
		response.Members = append(response.Members, item)
	}
	return response, nil
}

func normalizeCoreSeed(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if part = strings.ToLower(strings.TrimSpace(part)); part != "" {
				seen[part] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func (s *Server) getRecommendation(ctx context.Context) (*selectionv1.GetRecommendationResponse, error) {
	recommendation, err := buildRecommendation(ctx)
	if err != nil {
		return nil, err
	}
	return &selectionv1.GetRecommendationResponse{Profile: recommendation.Profile, Scenarios: recommendation.Scenarios, Resources: recommendation.Resources, Explanation: recommendation.Explanation}, nil
}

func (s *Server) acceptRecommendation(ctx context.Context, request *selectionv1.AcceptRecommendationRequest) (*selectionv1.AcceptRecommendationResponse, error) {
	if request.GetSelection() != nil {
		selection := request.GetSelection()
		if err := validateSelectionContract(selection); err != nil {
			return nil, err
		}
		patch := map[string]any{"scenarios": map[string]any{}}
		for _, name := range selection.GetScenarios() {
			patch["scenarios"].(map[string]any)[name] = map[string]any{"enabled": true}
		}
		if request.GetProfile() != "" {
			patch["active_profile"] = request.GetProfile()
		}
		if len(selection.GetOptionalResources()) > 0 {
			resources := map[string]any{}
			for _, name := range selection.GetOptionalResources() {
				resources[name] = map[string]any{"enabled": true}
			}
			patch["resources"] = resources
		}
		state, err := operatorStateService().Apply(ctx, marshalJSON(patch))
		if err != nil {
			return nil, err
		}
		return &selectionv1.AcceptRecommendationResponse{Selection: selection, FirstUnsatisfiedStep: int32(firstUnsatisfiedStep(state))}, nil
	}
	recommendation, err := buildRecommendation(ctx)
	if err != nil {
		return nil, err
	}
	scenarios := make(map[string]operatorstate.ScenarioChoice, len(recommendation.Scenarios))
	for _, name := range recommendation.Scenarios {
		enabled := true
		scenarios[name] = operatorstate.ScenarioChoice{Enabled: &enabled}
	}
	state, err := operatorStateService().Apply(ctx, marshalJSON(map[string]any{"active_profile": recommendation.Profile, "scenarios": scenarios}))
	if err != nil {
		return nil, err
	}
	selection := &setupv1.Selection{SchemaVersion: "v1", Scenarios: append([]string(nil), recommendation.Scenarios...), OptionalResources: append([]string(nil), recommendation.Resources...), Apply: true}
	return &selectionv1.AcceptRecommendationResponse{Selection: selection, FirstUnsatisfiedStep: int32(firstUnsatisfiedStep(state))}, nil
}

func (s *Server) getClosure(ctx context.Context) (*selectionv1.GetClosureResponse, error) {
	root, models, err := s.selectionInputs()
	if err != nil {
		return nil, err
	}
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return nil, err
	}
	result, err := resolveClosureForState(root, models, state)
	if err != nil {
		return nil, err
	}
	return &selectionv1.GetClosureResponse{Scenarios: closureMembersProto(result.Scenarios), Resources: closureMembersProto(result.Resources)}, nil
}

func (s *Server) getUnion(ctx context.Context) (*selectionv1.GetUnionResponse, error) {
	root, models, err := s.selectionInputs()
	if err != nil {
		return nil, err
	}
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return nil, err
	}
	closure, err := resolveClosureForState(root, models, state)
	if err != nil {
		return nil, err
	}
	selected := make([]ScenarioReadModel, 0, len(closure.Scenarios))
	byName := map[string]ScenarioReadModel{}
	for _, model := range models {
		byName[model.Name] = model
	}
	for _, member := range closure.Scenarios {
		if model, ok := byName[member.Name]; ok {
			selected = append(selected, model)
		}
	}
	requirements, err := deriveV2HostRequirements(root, state, selected)
	if err != nil {
		return nil, err
	}
	tools, safeguards := []string{}, []string{}
	for _, item := range requirements.Tools {
		tools = append(tools, item.Name)
	}
	for _, item := range requirements.Safeguards {
		safeguards = append(safeguards, item.Name)
	}
	sort.Strings(tools)
	sort.Strings(safeguards)
	resources, err := resourceService().Catalog(ctx)
	if err != nil {
		// Minimal source fixtures can provide the authoritative closure without
		// carrying the optional resource catalog. Preserve those union members;
		// return richer resource models when the catalog is present.
		resources = make([]*resourcesv1.Resource, 0, len(closure.Resources))
		for _, member := range closure.Resources {
			resources = append(resources, &resourcesv1.Resource{Name: member.Name, Enabled: member.Required})
		}
	}
	byResource := map[string]*resourcesv1.Resource{}
	for _, model := range resources {
		byResource[model.Name] = model
	}
	allResources := make([]*selectionv1.Resource, 0, len(resources))
	requiredResources := []*selectionv1.Resource{}
	optionalResources := []*selectionv1.Resource{}
	standaloneResources := []*selectionv1.Resource{}
	toProto := func(model *resourcesv1.Resource, enabled bool) *selectionv1.Resource {
		return &selectionv1.Resource{Name: model.Name, DisplayName: model.DisplayName, Description: model.Description, Category: model.Category, Enabled: enabled, Installed: model.Installed}
	}
	for _, model := range resources {
		allResources = append(allResources, toProto(model, model.Enabled))
	}
	for _, member := range closure.Resources {
		if model, ok := byResource[member.Name]; ok {
			if member.Required {
				requiredResources = append(requiredResources, toProto(model, true))
			} else {
				optionalResources = append(optionalResources, toProto(model, model.Enabled))
			}
			delete(byResource, member.Name)
		}
	}
	for _, model := range byResource {
		standaloneResources = append(standaloneResources, toProto(model, model.Enabled))
	}
	sort.Slice(allResources, func(i, j int) bool { return allResources[i].Name < allResources[j].Name })
	sort.Slice(requiredResources, func(i, j int) bool { return requiredResources[i].Name < requiredResources[j].Name })
	sort.Slice(optionalResources, func(i, j int) bool { return optionalResources[i].Name < optionalResources[j].Name })
	sort.Slice(standaloneResources, func(i, j int) bool { return standaloneResources[i].Name < standaloneResources[j].Name })
	return &selectionv1.GetUnionResponse{Scenarios: closureMembersProto(closure.Scenarios), Resources: closureMembersProto(closure.Resources), HostTools: tools, Safeguards: safeguards, CatalogPaths: []string{"catalog/scenarios", "catalog/resources", "catalog/internal/tools", "catalog/internal/safeguards"}, ResourceModels: allResources, RequiredResources: requiredResources, OptionalResources: optionalResources, StandaloneResources: standaloneResources}, nil
}

func (s *Server) createHandoff(ctx context.Context, request *selectionv1.CreateHandoffRequest) (*selectionv1.CreateHandoffResponse, error) {
	if request.GetDesiredSelection() != nil {
		selection := request.GetDesiredSelection()
		if err := validateSelectionContract(selection); err != nil {
			return nil, err
		}
		selection.Apply = true
		sort.Strings(selection.Scenarios)
		sort.Strings(selection.OptionalResources)
		sort.Strings(selection.HostTools)
		sort.Strings(selection.HostSafeguards)
		return &selectionv1.CreateHandoffResponse{Selection: selection}, nil
	}
	if strings.TrimSpace(request.GetNodeId()) == "" {
		return nil, fmt.Errorf("node_id is required")
	}
	root, models, err := s.selectionInputs()
	if err != nil {
		return nil, err
	}
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return nil, err
	}
	closure, err := resolveClosureForState(root, models, state)
	if err != nil {
		return nil, err
	}
	requirements, err := deriveV2HostRequirements(root, state, models)
	if err != nil {
		return nil, err
	}
	selection := &setupv1.Selection{SchemaVersion: "v1", Apply: true, OperatingMode: map[string]string{}}
	for _, model := range models {
		if model.Enabled {
			selection.Scenarios = append(selection.Scenarios, model.Name)
		}
		if choice, ok := state.Scenarios[model.Name]; ok && choice.AutoRestart != nil {
			if *choice.AutoRestart {
				selection.OperatingMode[model.Name] = "auto-restart"
			} else {
				selection.OperatingMode[model.Name] = "manual"
			}
		}
	}
	for _, member := range closure.Resources {
		if !member.Required {
			selection.OptionalResources = append(selection.OptionalResources, member.Name)
		}
	}
	for _, item := range requirements.Tools {
		if item.Status == "opted_in" {
			selection.HostTools = append(selection.HostTools, item.Name)
		}
	}
	for _, item := range requirements.Safeguards {
		if item.Status == "opted_in" {
			selection.HostSafeguards = append(selection.HostSafeguards, item.Name)
		}
	}
	sort.Strings(selection.Scenarios)
	sort.Strings(selection.OptionalResources)
	sort.Strings(selection.HostTools)
	sort.Strings(selection.HostSafeguards)
	if len(selection.OperatingMode) == 0 {
		selection.OperatingMode = nil
	}
	return &selectionv1.CreateHandoffResponse{Selection: selection}, nil
}

func (s *Server) selectionInputs() (string, []ScenarioReadModel, error) {
	root, err := manifestRoot()
	if err != nil {
		return "", nil, err
	}
	models, err := loadScenarioReadModels()
	if err != nil {
		return "", nil, err
	}
	return root, models, nil
}

func closureMembersProto(values []closureMember) []*selectionv1.ClosureMember {
	result := make([]*selectionv1.ClosureMember, 0, len(values))
	for _, value := range values {
		item := &selectionv1.ClosureMember{Name: value.Name, Required: value.Required, Direct: value.Direct, State: value.State, Reason: value.Reason, Policy: value.Policy}
		for _, provenance := range value.Provenance {
			item.Provenance = append(item.Provenance, &selectionv1.ClosureProvenance{Kind: provenance.Kind, From: provenance.From})
		}
		result = append(result, item)
	}
	return result
}

func marshalJSON(value any) []byte { data, _ := json.Marshal(value); return data }
