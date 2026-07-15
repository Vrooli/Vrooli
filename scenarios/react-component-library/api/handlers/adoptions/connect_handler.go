package adoptions

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"connectrpc.com/connect"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/components"
	"react-component-library/internal/deps"

	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory"
)

// Deps wires the seams the Connect adoptions handler needs.
type Deps struct {
	Service adoptions.Service
	Logger  *log.Logger
	// Resolver computes adopted paths for ResolveAdoptionPath. Optional: when
	// nil, ResolveAdoptionPath returns connect.CodeUnimplemented.
	Resolver *adoptions.Resolver
	// SlotReader looks up a component's declared slot. Optional, paired with
	// Resolver.
	SlotReader SlotReader
	// Library is the component lookup the handler uses to fetch DisplayName
	// (for token substitution in ResolveAdoptionPath). Optional in tests
	// that don't exercise the resolver path.
	Library adoptions.LibraryReader
	// Suggestion dependencies are optional so existing focused handler tests do
	// not construct inventory/dependency services.
	Components    components.Service
	Dependencies  deps.Service
	Inventory     InventoryScanner
	ScenariosRoot string
}

// InventoryScanner is the concrete InventoryService.ScanScenario seam used by
// suggestions. Keeping the existing RPC as the seam ensures the ranking uses
// the same production inventory operators inspect elsewhere.
type InventoryScanner interface {
	ScanScenario(context.Context, *connect.Request[inventoryv1.ScanScenarioRequest]) (*connect.Response[inventoryv1.ScanScenarioResponse], error)
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.ListAdoptionsRequest]) (*connect.Response[adoptionsv1.ListAdoptionsResponse], error) {
	out, err := h.deps.Service.List(ctx, adoptions.ListQuery{
		ComponentID: req.Msg.ComponentId,
		Scenario:    req.Msg.Scenario,
		Limit:       int(req.Msg.Limit),
	})
	if err != nil {
		h.deps.Logger.Printf("adoptions.ListAdoptions: %v", err)
		return nil, adoptions.ToConnectError(err)
	}
	resp := &adoptionsv1.ListAdoptionsResponse{Adoptions: make([]*adoptionsv1.Adoption, 0, len(out))}
	for _, a := range out {
		resp.Adoptions = append(resp.Adoptions, domainToProto(a))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ApplyAdoption(ctx context.Context, req *connect.Request[adoptionsv1.ApplyAdoptionRequest]) (*connect.Response[adoptionsv1.ApplyAdoptionResponse], error) {
	result, err := h.deps.Service.Apply(ctx, adoptions.ApplyInput{
		ComponentID:        req.Msg.ComponentId,
		Scenario:           req.Msg.Scenario,
		AdoptedPath:        req.Msg.AdoptedPath,
		Version:            req.Msg.Version,
		ConfirmOverwrite:   req.Msg.ConfirmOverwrite,
		OverrideValidation: req.Msg.OverrideValidation,
		ReplaceExisting:    req.Msg.ReplaceExisting,
	})
	if err != nil {
		connectErr := adoptions.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("adoptions.ApplyAdoption: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&adoptionsv1.ApplyAdoptionResponse{Adoption: domainToProto(result.Adoption), WrittenPath: result.WrittenPath, ImportSites: result.ImportSites}), nil
}

func (h *connectHandler) ReapplyAdoption(ctx context.Context, req *connect.Request[adoptionsv1.ReapplyAdoptionRequest]) (*connect.Response[adoptionsv1.ReapplyAdoptionResponse], error) {
	got, writtenPath, err := h.deps.Service.Reapply(ctx, adoptions.ReapplyInput{
		ID:                    req.Msg.Id,
		Version:               req.Msg.Version,
		ConfirmLocalOverwrite: req.Msg.ConfirmLocalOverwrite,
		OverrideValidation:    req.Msg.OverrideValidation,
	})
	if err != nil {
		connectErr := adoptions.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("adoptions.ReapplyAdoption: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&adoptionsv1.ReapplyAdoptionResponse{Adoption: domainToProto(got), WrittenPath: writtenPath}), nil
}

func (h *connectHandler) DeleteAdoption(ctx context.Context, req *connect.Request[adoptionsv1.DeleteAdoptionRequest]) (*connect.Response[adoptionsv1.DeleteAdoptionResponse], error) {
	if err := h.deps.Service.Delete(ctx, req.Msg.Id); err != nil {
		connectErr := adoptions.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("adoptions.DeleteAdoption(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&adoptionsv1.DeleteAdoptionResponse{}), nil
}

func (h *connectHandler) RefreshAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.RefreshAdoptionsRequest]) (*connect.Response[adoptionsv1.RefreshAdoptionsResponse], error) {
	rows, summary, err := h.deps.Service.Refresh(ctx, req.Msg.ComponentId)
	if err != nil {
		h.deps.Logger.Printf("adoptions.RefreshAdoptions: %v", err)
		return nil, adoptions.ToConnectError(err)
	}
	resp := &adoptionsv1.RefreshAdoptionsResponse{
		Adoptions:         make([]*adoptionsv1.Adoption, 0, len(rows)),
		LibraryCurrent:    int32(summary.LibraryCurrent),
		LibraryBehind:     int32(summary.LibraryBehind),
		LibraryDeprecated: int32(summary.LibraryDeprecated),
		LibraryMissing:    int32(summary.LibraryMissing),
		LibraryUnknown:    int32(summary.LibraryUnknown),
		LocalClean:        int32(summary.LocalClean),
		LocalModified:     int32(summary.LocalModified),
		LocalMissing:      int32(summary.LocalMissing),
		LocalUnknown:      int32(summary.LocalUnknown),
	}
	for _, a := range rows {
		resp.Adoptions = append(resp.Adoptions, domainToProto(a))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ReconcileAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.ReconcileAdoptionsRequest]) (*connect.Response[adoptionsv1.ReconcileAdoptionsResponse], error) {
	out, err := h.deps.Service.Reconcile(ctx, adoptions.ReconcileInput{Apply: req.Msg.Apply})
	if err != nil {
		h.deps.Logger.Printf("adoptions.ReconcileAdoptions: %v", err)
		return nil, adoptions.ToConnectError(err)
	}
	resp := &adoptionsv1.ReconcileAdoptionsResponse{Scanned: int32(out.Scanned), AlreadyRecorded: int32(out.AlreadyRecorded), Created: int32(out.Created)}
	for _, finding := range out.Findings {
		resp.Findings = append(resp.Findings, &adoptionsv1.ReconcileFinding{Scenario: finding.Scenario, AdoptedPath: finding.AdoptedPath, LibraryId: finding.LibraryID, Version: finding.Version, Detail: finding.Detail})
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ReconvergeAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.ReconvergeAdoptionsRequest]) (*connect.Response[adoptionsv1.ReconvergeAdoptionsResponse], error) {
	out, err := h.deps.Service.Reconverge(ctx, adoptions.ReconvergeInput{Scenario: req.Msg.Scenario, Apply: req.Msg.Apply})
	if err != nil {
		h.deps.Logger.Printf("adoptions.ReconvergeAdoptions: %v", err)
		return nil, adoptions.ToConnectError(err)
	}
	resp := &adoptionsv1.ReconvergeAdoptionsResponse{
		Scanned:   int32(out.Scanned),
		Behind:    int32(out.Behind),
		Reapplied: int32(out.Reapplied),
		Flagged:   int32(out.Flagged),
		Skipped:   int32(out.Skipped),
		Errored:   int32(out.Errored),
	}
	for _, o := range out.Outcomes {
		resp.Outcomes = append(resp.Outcomes, reconvergeOutcomeToProto(o))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) DiscoverAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.DiscoverAdoptionsRequest]) (*connect.Response[adoptionsv1.DiscoverAdoptionsResponse], error) {
	out, err := h.deps.Service.Discover(ctx, adoptions.DiscoverInput{
		Scenario:      req.Msg.Scenario,
		MinSimilarity: req.Msg.MinSimilarity,
		Limit:         int(req.Msg.Limit),
	})
	if err != nil {
		h.deps.Logger.Printf("adoptions.DiscoverAdoptions: %v", err)
		return nil, adoptions.ToConnectError(err)
	}
	resp := &adoptionsv1.DiscoverAdoptionsResponse{Scanned: int32(out.Scanned), MinSimilarity: out.MinSimilarity}
	for _, c := range out.Candidates {
		resp.Candidates = append(resp.Candidates, &adoptionsv1.DiscoveryCandidate{
			Scenario:       c.Scenario,
			AdoptedPath:    c.AdoptedPath,
			ComponentId:    c.ComponentID,
			LibraryId:      c.LibraryID,
			Version:        c.Version,
			DisplayName:    c.DisplayName,
			Similarity:     c.Similarity,
			SharedLines:    int32(c.SharedLines),
			CandidateLines: int32(c.CandidateLines),
			SourceLines:    int32(c.SourceLines),
			BasenameMatch:  c.BasenameMatch,
			Evidence:       c.Evidence,
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ConfirmDiscovery(ctx context.Context, req *connect.Request[adoptionsv1.ConfirmDiscoveryRequest]) (*connect.Response[adoptionsv1.ConfirmDiscoveryResponse], error) {
	out, err := h.deps.Service.ConfirmDiscovery(ctx, adoptions.ConfirmDiscoveryInput{
		Scenario:    req.Msg.Scenario,
		AdoptedPath: req.Msg.AdoptedPath,
		ComponentID: req.Msg.ComponentId,
		Version:     req.Msg.Version,
	})
	if err != nil {
		connectErr := adoptions.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("adoptions.ConfirmDiscovery: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&adoptionsv1.ConfirmDiscoveryResponse{Adoption: domainToProto(out.Adoption), WrittenPath: out.WrittenPath, Similarity: out.Similarity}), nil
}

func (h *connectHandler) SuggestAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.SuggestAdoptionsRequest]) (*connect.Response[adoptionsv1.SuggestAdoptionsResponse], error) {
	if h.deps.Components == nil || h.deps.Dependencies == nil || h.deps.Inventory == nil || h.deps.ScenariosRoot == "" {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("adoption suggestions are not configured"))
	}
	scenarios, err := h.suggestionScenarios(req.Msg.Scenario)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	componentsList, err := h.deps.Components.List(ctx, components.SearchQuery{Limit: 500})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if componentID := strings.TrimSpace(req.Msg.ComponentId); componentID != "" {
		componentsList = slices.DeleteFunc(componentsList, func(component components.Component) bool {
			return component.ID != componentID
		})
		if len(componentsList) == 0 {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("component %q not found", componentID))
		}
	}
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 50
	}
	result := make([]*adoptionsv1.AdoptionSuggestion, 0)
	for _, scenario := range scenarios {
		inventory, err := h.deps.Inventory.ScanScenario(ctx, connect.NewRequest(&inventoryv1.ScanScenarioRequest{Scenario: scenario}))
		if err != nil {
			continue // scenarios without a compatible UI manifest are not candidates
		}
		adopted, err := h.deps.Service.List(ctx, adoptions.ListQuery{Scenario: scenario, Limit: 1024})
		if err != nil {
			return nil, adoptions.ToConnectError(err)
		}
		alreadyAdopted := make(map[string]bool, len(adopted))
		for _, row := range adopted {
			alreadyAdopted[row.ComponentID] = true
		}
		for _, component := range componentsList {
			if alreadyAdopted[component.ID] {
				continue
			}
			for _, surface := range inventory.Msg.Surfaces {
				match := suggestionMatch(component, surface.DisplayName, surface.FilePath)
				if match == "" {
					continue
				}
				version := firstVersion(component.LatestVersion, component.Version)
				depVerdict, err := h.deps.Dependencies.ValidateAdoption(ctx, component.ID, version, scenario)
				if err != nil || depVerdict.Kind == deps.VerdictBlock {
					continue
				}
				style, err := h.deps.Components.ValidateStyleFit(ctx, component.ID, version, scenario)
				if err != nil {
					continue
				}
				reasons := []string{fmt.Sprintf("inventory surface %q matches %s", surface.FilePath, match)}
				if style.Detail != "" {
					reasons = append(reasons, style.Detail)
				}
				if depVerdict.Kind == deps.VerdictWarn {
					reasons = append(reasons, "dependency compatibility has warnings")
				} else {
					reasons = append(reasons, "dependencies are compatible")
				}
				result = append(result, &adoptionsv1.AdoptionSuggestion{Scenario: scenario, ComponentId: component.ID, LibraryId: component.LibraryID, DisplayName: component.DisplayName, InventoryPath: surface.FilePath, Reasons: reasons})
				break
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Scenario != result[j].Scenario {
			return result[i].Scenario < result[j].Scenario
		}
		return result[i].LibraryId < result[j].LibraryId
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return connect.NewResponse(&adoptionsv1.SuggestAdoptionsResponse{Suggestions: result}), nil
}

func (h *connectHandler) suggestionScenarios(requested string) ([]string, error) {
	if requested = strings.TrimSpace(requested); requested != "" {
		if strings.ContainsAny(requested, "/\\") || strings.Contains(requested, "..") {
			return nil, fmt.Errorf("invalid scenario %q", requested)
		}
		return []string{requested}, nil
	}
	entries, err := os.ReadDir(h.deps.ScenariosRoot)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			out = append(out, entry.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func suggestionMatch(component components.Component, displayName, path string) string {
	haystack := normalizeSuggestionToken(displayName + " " + filepath.Base(path))
	for _, candidate := range suggestionTerms(component) {
		normalized := normalizeSuggestionToken(candidate)
		if len(normalized) > 2 && strings.Contains(haystack, normalized) {
			return candidate
		}
	}
	return ""
}

// suggestionTerms includes a component's identity plus the reusable surface
// named by an intentionally thin "*Shell" wrapper. This keeps suggestions
// explainable (DrawerShell -> drawer) without falling back to broad tags such
// as "layout" or "surface", which produced noisy recommendations.
func suggestionTerms(component components.Component) []string {
	terms := []string{component.Slug, component.DisplayName}
	for _, identity := range []string{component.Slug, component.DisplayName} {
		normalized := normalizeSuggestionToken(identity)
		if base, ok := strings.CutSuffix(normalized, "shell"); ok && len(base) > 2 {
			terms = append(terms, base)
		}
	}
	return terms
}

func normalizeSuggestionToken(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.ToLower(value))
}

func firstVersion(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
