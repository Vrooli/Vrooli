package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	apicoreset "github.com/vrooli/api-core/coreset"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/app/supervision"
	"github.com/vrooli/vrooli/internal/operatorstate"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	resourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/resources"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	selectiondomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/selection"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
		GetHandoff:           s.getHandoff,
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
	allResources := make([]*selectionv1.Resource, 0, len(closure.Resources))
	requiredResources := []*selectionv1.Resource{}
	optionalResources := []*selectionv1.Resource{}
	standaloneResources := []*selectionv1.Resource{}
	toProto := func(model *resourcesv1.Resource, enabled bool) *selectionv1.Resource {
		return &selectionv1.Resource{Name: model.Name, DisplayName: model.DisplayName, Description: model.Description, Category: model.Category, Enabled: enabled, Installed: model.Installed}
	}
	for _, member := range closure.Resources {
		if model, ok := byResource[member.Name]; ok {
			allResources = append(allResources, toProto(model, member.Required || model.Enabled))
			if member.Required {
				requiredResources = append(requiredResources, toProto(model, true))
			} else if member.Direct && hasClosureProvenance(member, "operator_enabled") {
				standaloneResources = append(standaloneResources, toProto(model, model.Enabled))
			} else {
				optionalResources = append(optionalResources, toProto(model, model.Enabled))
			}
			delete(byResource, member.Name)
		}
	}
	sort.Slice(allResources, func(i, j int) bool { return allResources[i].Name < allResources[j].Name })
	sort.Slice(requiredResources, func(i, j int) bool { return requiredResources[i].Name < requiredResources[j].Name })
	sort.Slice(optionalResources, func(i, j int) bool { return optionalResources[i].Name < optionalResources[j].Name })
	sort.Slice(standaloneResources, func(i, j int) bool { return standaloneResources[i].Name < standaloneResources[j].Name })
	return &selectionv1.GetUnionResponse{Scenarios: closureMembersProto(closure.Scenarios), Resources: closureMembersProto(closure.Resources), HostTools: tools, Safeguards: safeguards, CatalogPaths: []string{"catalog/scenarios", "catalog/resources", "catalog/internal/tools", "catalog/internal/safeguards"}, ResourceModels: allResources, RequiredResources: requiredResources, OptionalResources: optionalResources, StandaloneResources: standaloneResources}, nil
}

func hasClosureProvenance(member closureMember, kind string) bool {
	for _, provenance := range member.Provenance {
		if provenance.Kind == kind {
			return true
		}
	}
	return false
}

func (s *Server) createHandoff(ctx context.Context, request *selectionv1.CreateHandoffRequest) (*selectionv1.CreateHandoffResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("handoff request is required")
	}
	selection, err := s.effectiveHandoffSelection(ctx, request)
	if err != nil {
		return nil, err
	}
	if target := strings.TrimSpace(request.GetTarget()); target != "" {
		if selectedTarget := strings.TrimSpace(selection.GetTarget()); selectedTarget != "" && selectedTarget != target {
			return nil, fmt.Errorf("selection target %q does not match handoff target %q", selectedTarget, target)
		}
		selection.Target = target
	}
	// The legacy selection projection remains available to local onboarding
	// callers. Cloud callers must supply the deployment and target fences so a
	// handoff can be resumed by identity rather than by an unowned URI.
	if strings.TrimSpace(request.GetDeploymentId()) == "" {
		return &selectionv1.CreateHandoffResponse{Selection: selection}, nil
	}
	if strings.TrimSpace(request.GetTarget()) == "" || strings.TrimSpace(request.GetMachineId()) == "" || strings.TrimSpace(request.GetNodeId()) == "" {
		return nil, fmt.Errorf("target, machine_id, and node_id are required for a durable handoff")
	}
	if request.GetEnrollmentGeneration() == 0 || request.GetDesiredRevision() == 0 {
		return nil, fmt.Errorf("enrollment_generation and desired_revision are required for a durable handoff")
	}
	actorScope, err := verifiedHandoffActor(ctx)
	if err != nil {
		return nil, err
	}
	selectionDigest, err := handoffSelectionDigest(selection)
	if err != nil {
		return nil, err
	}
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return nil, err
	}
	for _, existing := range state.Handoffs {
		if existing.RequestKey == strings.TrimSpace(request.GetRequestKey()) && existing.RequestKey != "" && existing.DeploymentID == strings.TrimSpace(request.GetDeploymentId()) && existing.ActorScope == actorScope {
			return &selectionv1.CreateHandoffResponse{Selection: selection, Handoff: handoffProto(existing)}, nil
		}
	}
	id, err := newHandoffID()
	if err != nil {
		return nil, err
	}
	now := operatorStateNow().UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	missing := sortedStrings(request.GetMissing())
	record := operatorstate.Handoff{
		ID: id, RequestKey: strings.TrimSpace(request.GetRequestKey()),
		DeploymentID: strings.TrimSpace(request.GetDeploymentId()), Target: strings.TrimSpace(request.GetTarget()),
		MachineID: strings.TrimSpace(request.GetMachineId()), NodeID: strings.TrimSpace(request.GetNodeId()), NodeKind: strings.TrimSpace(request.GetNodeKind()),
		EnrollmentGeneration: request.GetEnrollmentGeneration(), DesiredRevision: request.GetDesiredRevision(), ActorScope: actorScope,
		SelectionDigest: selectionDigest, Missing: missing, Selection: handoffSelection(selection), State: "pending", CreatedAt: now, UpdatedAt: now,
	}
	handoffs := make(map[string]operatorstate.Handoff, len(state.Handoffs)+1)
	for key, value := range state.Handoffs {
		handoffs[key] = value
	}
	handoffs[id] = record
	patch, err := json.Marshal(map[string]any{"handoffs": handoffs})
	if err != nil {
		return nil, err
	}
	if _, err := operatorStateService().ApplyAtRevision(ctx, operatorstate.Revision(state), patch); err != nil {
		return nil, fmt.Errorf("persist onboarding handoff: %w", err)
	}
	return &selectionv1.CreateHandoffResponse{Selection: selection, Handoff: handoffProto(record)}, nil
}

func (s *Server) effectiveHandoffSelection(ctx context.Context, request *selectionv1.CreateHandoffRequest) (*setupv1.Selection, error) {
	if request.GetDesiredSelection() != nil {
		selection := protoCloneSelection(request.GetDesiredSelection())
		if err := validateSelectionContract(selection); err != nil {
			return nil, err
		}
		normalizeHandoffSelection(selection)
		return selection, nil
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
	normalizeHandoffSelection(selection)
	return selection, nil
}

func verifiedHandoffActor(ctx context.Context) (string, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() || strings.TrimSpace(principal.Subject) == "" || strings.TrimSpace(string(principal.Source)) == "" {
		return "", fmt.Errorf("verified actor scope is required for a durable handoff")
	}
	return strings.TrimSpace(string(principal.Source)) + ":" + strings.TrimSpace(principal.Subject), nil
}

func newHandoffID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate handoff identity: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}

func handoffSelectionDigest(selection *setupv1.Selection) (string, error) {
	encoded, err := protojson.Marshal(selection)
	if err != nil {
		return "", fmt.Errorf("encode handoff selection: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func normalizeHandoffSelection(selection *setupv1.Selection) {
	selection.SchemaVersion = "v1"
	selection.Apply = true
	sort.Strings(selection.Scenarios)
	sort.Strings(selection.OptionalResources)
	sort.Strings(selection.HostTools)
	sort.Strings(selection.HostSafeguards)
	if len(selection.OperatingMode) == 0 {
		selection.OperatingMode = nil
	}
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func protoCloneSelection(selection *setupv1.Selection) *setupv1.Selection {
	if selection == nil {
		return nil
	}
	return proto.Clone(selection).(*setupv1.Selection)
}

func handoffSelection(selection *setupv1.Selection) operatorstate.HandoffSelection {
	return operatorstate.HandoffSelection{SchemaVersion: selection.GetSchemaVersion(), Scenarios: append([]string(nil), selection.GetScenarios()...), OptionalResources: append([]string(nil), selection.GetOptionalResources()...), HostTools: append([]string(nil), selection.GetHostTools()...), HostSafeguards: append([]string(nil), selection.GetHostSafeguards()...), OperatingMode: cloneStringMap(selection.GetOperatingMode())}
}

func selectionFromHandoff(value operatorstate.HandoffSelection) *setupv1.Selection {
	return &setupv1.Selection{SchemaVersion: value.SchemaVersion, Apply: true, Scenarios: append([]string(nil), value.Scenarios...), OptionalResources: append([]string(nil), value.OptionalResources...), HostTools: append([]string(nil), value.HostTools...), HostSafeguards: append([]string(nil), value.HostSafeguards...), OperatingMode: cloneStringMap(value.OperatingMode)}
}

func handoffProto(value operatorstate.Handoff) *selectionv1.Handoff {
	return &selectionv1.Handoff{Id: value.ID, Reference: "vrooli-onboarding://handoffs/" + value.ID, DeploymentId: value.DeploymentID, Target: value.Target, MachineId: value.MachineID, NodeId: value.NodeID, NodeKind: value.NodeKind, EnrollmentGeneration: value.EnrollmentGeneration, DesiredRevision: value.DesiredRevision, ActorScope: value.ActorScope, SelectionDigest: value.SelectionDigest, Missing: append([]string(nil), value.Missing...), State: value.State, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func (s *Server) getHandoff(ctx context.Context, request *selectionv1.GetHandoffRequest) (*selectionv1.GetHandoffResponse, error) {
	actorScope, err := verifiedHandoffActor(ctx)
	if err != nil {
		return nil, err
	}
	id := strings.TrimPrefix(strings.TrimSpace(request.GetReference()), "vrooli-onboarding://handoffs/")
	if id == "" || id == strings.TrimSpace(request.GetReference()) {
		return nil, fmt.Errorf("handoff reference is invalid")
	}
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return nil, err
	}
	record, ok := state.Handoffs[id]
	if !ok {
		return nil, fmt.Errorf("handoff %q was not found", id)
	}
	if record.State != "pending" || record.ActorScope != actorScope || record.DeploymentID != strings.TrimSpace(request.GetDeploymentId()) || record.Target != strings.TrimSpace(request.GetTarget()) || record.EnrollmentGeneration != request.GetEnrollmentGeneration() || record.DesiredRevision != request.GetDesiredRevision() || record.SelectionDigest != strings.TrimSpace(request.GetSelectionDigest()) {
		return nil, fmt.Errorf("handoff %q is stale or outside the authorized target revision", id)
	}
	return &selectionv1.GetHandoffResponse{Handoff: handoffProto(record), Selection: selectionFromHandoff(record.Selection)}, nil
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
