// Package graph is the Connect-RPC surface for the graph domain.
// Translates between the proto wire types and the domain types in
// internal/graph; applies the error-mapping policy; honours X-Dry-Run
// for ClearGraphSnapshots.
package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"architecture-cartographer/internal/archetype"
	"architecture-cartographer/internal/attest"
	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/layering"
	intdomains "architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	intslice "architecture-cartographer/internal/slice"
	"architecture-cartographer/internal/zones"

	"connectrpc.com/connect"
	domainsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph/graph_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler implements graph_v1connect.GraphServiceHandler.
type Handler struct {
	graph_v1connect.UnimplementedGraphServiceHandler
	svc        graph.Service
	domainsSvc intdomains.Service
}

// NewHandler constructs the Connect handler.
func NewHandler(svc graph.Service, domainsSvc ...intdomains.Service) *Handler {
	h := &Handler{svc: svc}
	if len(domainsSvc) > 0 {
		h.domainsSvc = domainsSvc[0]
	}
	return h
}

var _ graph_v1connect.GraphServiceHandler = (*Handler)(nil)

func (h *Handler) ExtractGraph(ctx context.Context, req *connect.Request[graphv1.ExtractGraphRequest]) (*connect.Response[graphv1.ExtractGraphResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	in := graph.ExtractGraphInput{
		Scenario:       scenario,
		IdempotencyKey: req.Msg.GetIdempotencyKey(),
	}
	for _, l := range req.Msg.GetLanguages() {
		in.Languages = append(in.Languages, protoToLanguage(l))
	}
	snap, fromCache, err := h.svc.ExtractGraph(ctx, in)
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&graphv1.ExtractGraphResponse{
		Snapshot:  snapshotToProto(snap),
		FromCache: fromCache,
	}), nil
}

func (h *Handler) GetGraphSnapshot(ctx context.Context, req *connect.Request[graphv1.GetGraphSnapshotRequest]) (*connect.Response[graphv1.GetGraphSnapshotResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	snap, err := h.svc.GetSnapshot(ctx, id)
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&graphv1.GetGraphSnapshotResponse{Snapshot: snapshotToProto(snap)}), nil
}

func (h *Handler) ListGraphSnapshots(ctx context.Context, req *connect.Request[graphv1.ListGraphSnapshotsRequest]) (*connect.Response[graphv1.ListGraphSnapshotsResponse], error) {
	filter := graph.ListSnapshotsFilter{
		Scenario: strings.TrimSpace(req.Msg.GetScenario()),
		PageSize: int(req.Msg.GetPageSize()),
	}
	page, err := h.svc.ListSnapshots(ctx, filter)
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	out := &graphv1.ListGraphSnapshotsResponse{}
	for _, s := range page.Snapshots {
		out.Snapshots = append(out.Snapshots, snapshotToProto(s))
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) ClearGraphSnapshots(ctx context.Context, req *connect.Request[graphv1.ClearGraphSnapshotsRequest]) (*connect.Response[graphv1.ClearGraphSnapshotsResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	dryRun := req.Msg.GetDryRun() || req.Header().Get("X-Dry-Run") == "true"
	deleted, dry, err := h.svc.ClearSnapshots(ctx, scenario, dryRun)
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&graphv1.ClearGraphSnapshotsResponse{
		Deleted: int32(deleted),
		DryRun:  dry,
	}), nil
}

func (h *Handler) ExportGraph(ctx context.Context, req *connect.Request[graphv1.ExportGraphRequest]) (*connect.Response[graphv1.ExportGraphResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	snap, err := h.svc.GetSnapshot(ctx, id)
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	payload, err := json.Marshal(snap)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&graphv1.ExportGraphResponse{
		Payload:     payload,
		ContentType: "application/json",
	}), nil
}

func (h *Handler) GetZoneMap(ctx context.Context, req *connect.Request[graphv1.GetZoneMapRequest]) (*connect.Response[graphv1.GetZoneMapResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	if h.domainsSvc == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("domains service is required for zone map"))
	}
	snap, err := h.snapshotForGraphView(ctx, scenario, strings.TrimSpace(req.Msg.GetSnapshotId()))
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	domainMap, err := h.domainsSvc.GetDomainMap(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(intdomains.ErrorToConnectCode(err), err)
	}
	zoneConfig := zones.LoadForScenarioName(scenario)
	declared := zones.LoadDeclaredZoneMapForScenario(scenario)
	importConsistent := zoneImportConsistency(snap, zoneConfig, domainMap)
	resp := &graphv1.GetZoneMapResponse{ZoneMap: &graphv1.ZoneMap{
		Scenario:   scenario,
		SnapshotId: snap.ID,
	}}
	anyDrift := false
	for _, pkg := range sortedPackages(snap.Packages) {
		if strings.TrimSpace(pkg.RepoPath) == "" {
			continue
		}
		info := zoneConfig.Classify(pkg.RepoPath, domainMap)
		var consistent *bool
		if c, ok := importConsistent[pkg.ID]; ok {
			consistent = &c
		}
		conv := zones.Converge(pkg.RepoPath, info, declared, consistent)
		zp := &graphv1.ZonePackage{
			PackageId:     pkg.ID,
			ImportPath:    pkg.ImportPath,
			RepoPath:      pkg.RepoPath,
			Zone:          info.Zone,
			Domain:        info.Domain,
			Archetype:     info.Archetype,
			Declared:      info.Declared,
			Confidence:    conv.Confidence,
			DeclaredLayer: conv.DeclaredLayer,
			Drift:         conv.Drift,
		}
		for _, e := range conv.Evidence {
			zp.Evidence = append(zp.Evidence, &graphv1.ZoneEvidence{Kind: e.Kind, Detail: e.Detail, Locator: e.Locator})
		}
		resp.ZoneMap.Packages = append(resp.ZoneMap.Packages, zp)
		if conv.Drift {
			anyDrift = true
			resp.ZoneMap.Violations = append(resp.ZoneMap.Violations, &graphv1.ZoneViolation{
				Kind:      "zone_drift",
				Subtype:   "declared_vs_derived",
				Severity:  sharedv1.Severity_SEVERITY_WARN,
				Locations: []string{pkg.RepoPath},
				Domains:   nonEmpty(info.Domain),
				Summary:   fmt.Sprintf("ARCHITECTURE.md declares layer %q (%s) but the code-derived zone is %q", conv.DeclaredLayer, conv.DeclaredZone, info.Zone),
			})
		}
	}
	// Per-scenario overlay deviations from the template SSOT (Phase 0.2).
	for _, dev := range zoneConfig.Deviations {
		resp.ZoneMap.Violations = append(resp.ZoneMap.Violations, &graphv1.ZoneViolation{
			Kind:     "zone_overlay_deviation",
			Subtype:  dev.Field,
			Severity: sharedv1.Severity_SEVERITY_INFO,
			Summary:  fmt.Sprintf("%s overridden by .vrooli/architecture.json: template=%v overlay=%v", dev.Field, dev.TemplateValue, dev.OverlayValue),
		})
	}
	violations, err := layering.New().Detect(ctx, conflicts.DetectInput{
		Scenario:  scenario,
		Snapshot:  snap,
		DomainMap: domainMap,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("detect layering violations: %w", err))
	}
	for _, v := range violations {
		resp.ZoneMap.Violations = append(resp.ZoneMap.Violations, &graphv1.ZoneViolation{
			Kind:      v.Type,
			Subtype:   v.Subtype,
			Severity:  severityToProto(v.Severity),
			Locations: append([]string(nil), v.Locations...),
			Domains:   append([]string(nil), v.Domains...),
			Summary:   firstEvidenceSummary(v),
		})
	}
	resp.ZoneMap.AuthorityConfidence = graphv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_LOW
	if declared.Present {
		resp.ZoneMap.AuthorityConfidence = graphv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_HIGH
	}
	resp.ZoneMap.Attestation = zoneMapAttestation(scenario, len(resp.ZoneMap.Packages), declared.Present, anyDrift)
	return connect.NewResponse(resp), nil
}

// zoneMapAttestation is the Q7 map-level honesty contract: DERIVED when zones
// are computed from code and the doc agrees (or is absent); CONTRADICTED when a
// declared layer disagrees with the code-derived zone.
func zoneMapAttestation(scenario string, pkgCount int, declaredPresent, anyDrift bool) *commonv1.AttestedAnswer {
	basis := commonv1.Basis_BASIS_DERIVED
	if declaredPresent && anyDrift {
		basis = commonv1.Basis_BASIS_CONTRADICTED
	} else if declaredPresent {
		basis = commonv1.Basis_BASIS_VALIDATED
	}
	suff := commonv1.Sufficiency_SUFFICIENCY_PARTIAL
	if declaredPresent {
		suff = commonv1.Sufficiency_SUFFICIENCY_FULL
	}
	b := attest.New(fmt.Sprintf("zone map for %q: %d package(s) classified", scenario, pkgCount)).
		Basis(basis).
		Sufficiency(suff).
		CiteCode("api/internal/zones/", "template-manifest zone classifier")
	if declaredPresent {
		b.CiteDoc(zones.ArchitectureDocPath, "declared Zone Map")
	} else {
		b.Gap("no ARCHITECTURE.md Zone Map to converge against; zones are derived-only")
		b.FollowUp("add a `## Zone Map` table to docs/concepts/ARCHITECTURE.md to raise authority confidence")
	}
	a := b.Build()
	if attest.Validate(a) != nil {
		a.Basis = commonv1.Basis_BASIS_ABSENT
	}
	return a
}

// zoneImportConsistency computes, per package id, whether its outbound imports
// stay within same-or-lower zones (the import-graph reality signal). A package
// that reaches a more surface-level zone than its own is flagged inconsistent;
// composition-root is exempt (it wires everything). Packages with an unknown
// zone are omitted.
func zoneImportConsistency(snap graph.GraphSnapshot, cfg zones.Config, domainMap intdomains.DerivedDomainMap) map[string]bool {
	zoneByPkg := make(map[string]string, len(snap.Packages))
	for _, pkg := range snap.Packages {
		if strings.TrimSpace(pkg.RepoPath) == "" {
			continue
		}
		zoneByPkg[pkg.ID] = cfg.Classify(pkg.RepoPath, domainMap).Zone
	}
	pkgByFile := make(map[string]string)
	for _, f := range snap.Files {
		pkgByFile[f.ID] = f.PackageID
	}
	out := make(map[string]bool, len(zoneByPkg))
	for id, z := range zoneByPkg {
		if z != zones.Unknown {
			out[id] = true // optimistic; demote on the first illegal edge
		}
	}
	for _, e := range snap.Imports {
		fromPkg := pkgByFile[e.From]
		fromZone, ok := zoneByPkg[fromPkg]
		if !ok || fromZone == zones.Unknown || fromZone == zones.CompositionRoot {
			continue
		}
		toZone, ok := zoneByPkg[e.ToPackageID]
		if !ok || toZone == zones.Unknown {
			continue
		}
		if zoneOrder(toZone) > zoneOrder(fromZone) {
			out[fromPkg] = false
		}
	}
	return out
}

// zoneOrder ranks zones from foundational (low) to surface (high). A package
// should depend only on same-or-lower zones.
func zoneOrder(zone string) int {
	switch zone {
	case zones.Substrate:
		return 0
	case zones.Domain:
		return 1
	case zones.Transport, zones.CLI, zones.UI:
		return 2
	default:
		return 1
	}
}

func nonEmpty(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return []string{s}
}

func (h *Handler) GetSlice(ctx context.Context, req *connect.Request[graphv1.GetSliceRequest]) (*connect.Response[graphv1.GetSliceResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	domain := strings.TrimSpace(req.Msg.GetDomain())
	if domain == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("domain is required"))
	}
	if h.domainsSvc == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("domains service is required for slice"))
	}
	snap, err := h.snapshotForGraphView(ctx, scenario, strings.TrimSpace(req.Msg.GetSnapshotId()))
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	domainMap, err := h.domainsSvc.GetDomainMap(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(intdomains.ErrorToConnectCode(err), err)
	}
	if !domainExists(domainMap, domain) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("domain %q not found in scenario %q", domain, scenario))
	}
	derived := intslice.Build(
		sliceSnapshot(snap),
		sliceDomainMap(domainMap),
		sliceClassifier{cfg: zones.LoadForScenarioName(scenario), domains: domainMap},
		domain,
	)
	return connect.NewResponse(&graphv1.GetSliceResponse{Slice: sliceToProto(derived)}), nil
}

// InferArchetype infers each domain's archetype from graph signals and
// converges it against the declared DOMAINS.md value (Q20).
func (h *Handler) InferArchetype(ctx context.Context, req *connect.Request[graphv1.InferArchetypeRequest]) (*connect.Response[graphv1.InferArchetypeResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	if h.domainsSvc == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("domains service is required for archetype inference"))
	}
	snap, err := h.snapshotForGraphView(ctx, scenario, strings.TrimSpace(req.Msg.GetSnapshotId()))
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	domainMap, err := h.domainsSvc.GetDomainMap(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(intdomains.ErrorToConnectCode(err), err)
	}
	want := strings.TrimSpace(req.Msg.GetDomain())
	resp := &graphv1.InferArchetypeResponse{Scenario: scenario, SnapshotId: snap.ID}
	for _, d := range domainMap.Domains {
		if want != "" && d.Name != want {
			continue
		}
		resp.Reports = append(resp.Reports, archetypeReport(snap, d))
	}
	if want != "" && len(resp.Reports) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("domain %q not found in scenario %q", want, scenario))
	}
	return connect.NewResponse(resp), nil
}

func archetypeReport(snap graph.GraphSnapshot, d intdomains.DerivedDomain) *graphv1.ArchetypeReport {
	in := graph.BuildArchetypeSignals(snap, d.Name, d.Paths)
	inferred := archetype.Infer(in)

	report := &graphv1.ArchetypeReport{Domain: d.Name}
	declaredPrimary := ""
	for _, a := range d.Archetypes {
		if a.Source != intdomains.ArchetypeSourceDeclared {
			continue
		}
		if declaredPrimary == "" {
			declaredPrimary = a.Name
		}
		report.Archetypes = append(report.Archetypes, &domainsv1.DomainArchetype{
			Archetype:     graphArchetypeEnum(a.Name),
			Source:        domainsv1.ArchetypeSource_ARCHETYPE_SOURCE_DECLARED,
			Confidence:    a.Confidence,
			Evidence:      append([]string(nil), a.Evidence...),
			DeclaredLabel: a.DeclaredLabel,
		})
	}
	inferredPrimary := ""
	for i, r := range inferred {
		if i == 0 {
			inferredPrimary = r.Name
		}
		report.Archetypes = append(report.Archetypes, &domainsv1.DomainArchetype{
			Archetype:  graphArchetypeEnum(r.Name),
			Source:     domainsv1.ArchetypeSource_ARCHETYPE_SOURCE_INFERRED,
			Confidence: r.Confidence,
			Evidence:   append([]string(nil), r.Evidence...),
		})
	}

	hasDoc := declaredPrimary != ""
	hasCode := inferredPrimary != ""
	agree := hasDoc && hasCode && declaredPrimary == inferredPrimary
	report.ConvergenceDrift = hasDoc && hasCode && !agree

	b := attest.New(fmt.Sprintf("domain %q archetype: declared=%q inferred=%q", d.Name, orDash(declaredPrimary), orDash(inferredPrimary))).
		Basis(attest.ConvergenceBasis(hasCode, hasDoc, agree)).
		Sufficiency(commonv1.Sufficiency_SUFFICIENCY_FULL)
	if hasDoc {
		b.CiteDoc(intdomains.DomainsDocPath, "declared Primary Archetype")
	}
	for _, r := range inferred {
		for _, ev := range r.Evidence {
			b.CiteCode(firstPath(d.Paths), ev)
		}
	}
	if report.ConvergenceDrift {
		b.Gap(fmt.Sprintf("declared %q disagrees with inferred %q", declaredPrimary, inferredPrimary))
	}
	a := b.Build()
	if attest.Validate(a) != nil {
		a.Basis = commonv1.Basis_BASIS_ABSENT
	}
	report.Attestation = a
	return report
}

func graphArchetypeEnum(name string) domainsv1.Archetype {
	switch archetype.Name(name) {
	case archetype.Reporting:
		return domainsv1.Archetype_ARCHETYPE_REPORTING
	case archetype.Service:
		return domainsv1.Archetype_ARCHETYPE_SERVICE
	case archetype.Mutation:
		return domainsv1.Archetype_ARCHETYPE_MUTATION
	case archetype.Classification:
		return domainsv1.Archetype_ARCHETYPE_CLASSIFICATION
	case archetype.Orchestration:
		return domainsv1.Archetype_ARCHETYPE_ORCHESTRATION
	case archetype.Scoring:
		return domainsv1.Archetype_ARCHETYPE_SCORING
	case archetype.Query:
		return domainsv1.Archetype_ARCHETYPE_QUERY
	default:
		return domainsv1.Archetype_ARCHETYPE_UNSPECIFIED
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func firstPath(paths []string) string {
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}

type sliceClassifier struct {
	cfg     zones.Config
	domains intdomains.DerivedDomainMap
}

func (c sliceClassifier) Classify(repoPath string) intslice.ZoneInfo {
	info := c.cfg.Classify(repoPath, c.domains)
	return intslice.ZoneInfo{
		Zone:   info.Zone,
		Domain: info.Domain,
	}
}

func sliceDomainMap(in intdomains.DerivedDomainMap) intslice.DomainMap {
	out := intslice.DomainMap{
		Scenario: in.Scenario,
		Domains:  make([]intslice.Domain, 0, len(in.Domains)),
	}
	for _, domain := range in.Domains {
		out.Domains = append(out.Domains, intslice.Domain{
			Name:  domain.Name,
			Paths: append([]string(nil), domain.Paths...),
		})
	}
	return out
}

func sliceSnapshot(s graph.GraphSnapshot) intslice.Snapshot {
	out := intslice.Snapshot{
		ID:       s.ID,
		Scenario: s.Scenario,
		Packages: make([]intslice.PackageNode, 0, len(s.Packages)),
		Imports:  make([]intslice.ImportEdge, 0, len(s.Imports)),
		Files:    make([]intslice.FileNode, 0, len(s.Files)),
		Symbols:  make([]intslice.SymbolNode, 0, len(s.Symbols)),
	}
	pkgByFile := make(map[string]string, len(s.Files))
	pathByFile := make(map[string]string, len(s.Files))
	for _, f := range s.Files {
		pkgByFile[f.ID] = f.PackageID
		pathByFile[f.ID] = f.Path
	}
	for _, pkg := range s.Packages {
		out.Packages = append(out.Packages, intslice.PackageNode{
			ID:         pkg.ID,
			ImportPath: pkg.ImportPath,
			RepoPath:   pkg.RepoPath,
		})
	}
	for _, edge := range s.Imports {
		out.Imports = append(out.Imports, intslice.ImportEdge{
			FromPackageID: pkgByFile[edge.From],
			ToPackageID:   edge.ToPackageID,
			TestOnly:      edge.TestOnly,
		})
	}
	for _, file := range s.Files {
		out.Files = append(out.Files, intslice.FileNode{
			Path:      file.Path,
			PackageID: file.PackageID,
			Lines:     file.Lines,
			IsTest:    file.IsTest,
		})
	}
	for _, sym := range s.Symbols {
		out.Symbols = append(out.Symbols, intslice.SymbolNode{
			Name:      sym.Name,
			Kind:      sym.Kind,
			PackageID: sym.PackageID,
			FilePath:  pathByFile[sym.FileID],
			Exported:  sym.Exported,
		})
	}
	return out
}

func (h *Handler) snapshotForGraphView(ctx context.Context, scenario, snapshotID string) (graph.GraphSnapshot, error) {
	if snapshotID != "" {
		return h.svc.GetSnapshot(ctx, snapshotID)
	}
	page, err := h.svc.ListSnapshots(ctx, graph.ListSnapshotsFilter{Scenario: scenario, PageSize: 1})
	if err != nil {
		return graph.GraphSnapshot{}, err
	}
	if len(page.Snapshots) > 0 {
		return page.Snapshots[0], nil
	}
	snap, _, err := h.svc.ExtractGraph(ctx, graph.ExtractGraphInput{Scenario: scenario})
	return snap, err
}

// -------------------------- proto<->domain --------------------------

func protoToLanguage(l graphv1.Language) graph.Language {
	switch l {
	case graphv1.Language_LANGUAGE_GO:
		return graph.LanguageGo
	case graphv1.Language_LANGUAGE_TYPESCRIPT:
		return graph.LanguageTypeScript
	default:
		return graph.LanguageUnspecified
	}
}

func languageToProto(l graph.Language) graphv1.Language {
	switch l {
	case graph.LanguageGo:
		return graphv1.Language_LANGUAGE_GO
	case graph.LanguageTypeScript:
		return graphv1.Language_LANGUAGE_TYPESCRIPT
	default:
		return graphv1.Language_LANGUAGE_UNSPECIFIED
	}
}

func snapshotToProto(s graph.GraphSnapshot) *graphv1.GraphSnapshot {
	out := &graphv1.GraphSnapshot{
		Id:           s.ID,
		Scenario:     s.Scenario,
		ContentHash:  s.ContentHash,
		ExtractionMs: s.ExtractionMS,
	}
	if !s.ExtractedAt.IsZero() {
		out.ExtractedAt = timestamppb.New(s.ExtractedAt)
	}
	for _, l := range s.Languages {
		out.Languages = append(out.Languages, languageToProto(l))
	}
	for _, f := range s.Files {
		out.Files = append(out.Files, &graphv1.FileNode{
			Id:        f.ID,
			Path:      f.Path,
			PackageId: f.PackageID,
			Language:  languageToProto(f.Language),
			Lines:     int32(f.Lines),
			IsTest:    f.IsTest,
		})
	}
	for _, p := range s.Packages {
		out.Packages = append(out.Packages, &graphv1.PackageNode{
			Id:         p.ID,
			ImportPath: p.ImportPath,
			RepoPath:   p.RepoPath,
			Language:   languageToProto(p.Language),
		})
	}
	for _, sym := range s.Symbols {
		out.Symbols = append(out.Symbols, &graphv1.SymbolNode{
			Id:        sym.ID,
			Name:      sym.Name,
			PackageId: sym.PackageID,
			FileId:    sym.FileID,
			Kind:      sym.Kind,
			Exported:  sym.Exported,
		})
	}
	for _, e := range s.Imports {
		out.Imports = append(out.Imports, &graphv1.ImportEdge{
			From:        e.From,
			ToPackageId: e.ToPackageID,
			SymbolIds:   append([]string(nil), e.SymbolIDs...),
			TestOnly:    e.TestOnly,
		})
	}
	out.ExtractionProfiles = append([]string(nil), s.ExtractionProfiles...)
	for _, omission := range s.OmittedInformation {
		out.OmittedInformation = append(out.OmittedInformation, &commonv1.CodeGraphOmission{
			Capability: omission.Capability,
			Reason:     omission.Reason,
		})
	}
	return out
}

func sortedPackages(in []graph.PackageNode) []graph.PackageNode {
	out := append([]graph.PackageNode(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].RepoPath != out[j].RepoPath {
			return out[i].RepoPath < out[j].RepoPath
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func firstEvidenceSummary(c conflicts.Conflict) string {
	for _, ev := range c.Evidence {
		if strings.TrimSpace(ev.Summary) != "" {
			return ev.Summary
		}
	}
	if len(c.Locations) > 0 {
		return strings.Join(c.Locations, " -> ")
	}
	return c.Subtype
}

func severityToProto(s conflicts.Severity) sharedv1.Severity {
	switch s {
	case conflicts.SeverityInfo:
		return sharedv1.Severity_SEVERITY_INFO
	case conflicts.SeverityWarn:
		return sharedv1.Severity_SEVERITY_WARN
	case conflicts.SeverityError:
		return sharedv1.Severity_SEVERITY_ERROR
	case conflicts.SeverityBlocker:
		return sharedv1.Severity_SEVERITY_BLOCKER
	default:
		return sharedv1.Severity_SEVERITY_UNSPECIFIED
	}
}

func sliceToProto(in intslice.DomainSlice) *graphv1.DomainSlice {
	out := &graphv1.DomainSlice{
		Scenario:   in.Scenario,
		Domain:     in.Domain,
		SnapshotId: in.SnapshotID,
		Surfaces:   append([]string(nil), in.Surfaces...),
	}
	presentRungs := 0
	for _, rung := range in.Rungs {
		pr := &graphv1.SliceRung{Name: rung.Name, Present: rung.Present}
		for _, e := range rung.Evidence {
			pr.Evidence = append(pr.Evidence, &graphv1.SliceEvidence{Kind: e.Kind, Value: e.Value, Source: e.Source})
		}
		for _, f := range rung.Files {
			pr.Files = append(pr.Files, &graphv1.SliceFile{Path: f.Path, Lines: int32(f.Lines), IsTest: f.IsTest})
		}
		for _, s := range rung.Symbols {
			pr.Symbols = append(pr.Symbols, &graphv1.SliceSymbol{Name: s.Name, Kind: s.Kind, File: s.File})
		}
		out.Rungs = append(out.Rungs, pr)
		if rung.Present {
			presentRungs++
		}
	}
	for _, e := range in.LayerEdges {
		out.LayerEdges = append(out.LayerEdges, &graphv1.SliceEdge{FromRung: e.FromRung, ToRung: e.ToRung, Kind: e.Kind})
	}
	out.Attestation = sliceAttestation(in, presentRungs)
	return out
}

// sliceAttestation is the Q16 honesty contract: DERIVED from the graph, with
// sufficiency PARTIAL while symbol-level edge resolution is deferred to the
// call/reference-edge phase. Citations are built from the rungs' typed evidence.
func sliceAttestation(in intslice.DomainSlice, presentRungs int) *commonv1.AttestedAnswer {
	b := attest.New(fmt.Sprintf("domain %q implementation slice: %d/%d rungs present", in.Domain, presentRungs, len(in.Rungs))).
		Basis(commonv1.Basis_BASIS_DERIVED).
		Sufficiency(commonv1.Sufficiency_SUFFICIENCY_PARTIAL).
		Gap("package-level edges only; symbol-level call/reference links pending (Q17)")
	for _, rung := range in.Rungs {
		for _, e := range rung.Evidence {
			kind := attest.KindCode
			if e.Source == "doc" {
				kind = attest.KindDoc
			}
			b.Cite(e.Value, kind, rung.Name+" rung "+e.Kind)
		}
	}
	a := b.Build()
	if attest.Validate(a) != nil {
		a.Basis = commonv1.Basis_BASIS_ABSENT
	}
	return a
}

func domainExists(m intdomains.DerivedDomainMap, name string) bool {
	for _, domain := range m.Domains {
		if domain.Name == name {
			return true
		}
	}
	return false
}
