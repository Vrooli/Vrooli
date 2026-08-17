package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog/catalog_v1connect"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/catalogexperience"
	"react-component-library/internal/gates"
	"react-component-library/internal/module"
)

type handler struct {
	repoRoot string
	evidence *catalogcoverage.EvidenceStore
	reports  reportCache
}

// Module exposes the same live coverage projection used by the component-test
// phase. No CLI-only computation or second catalog join is permitted.
func Module(repoRoot string, dbs ...*sql.DB) module.Module {
	var evidence *catalogcoverage.EvidenceStore
	if len(dbs) > 0 && dbs[0] != nil {
		evidence = catalogcoverage.NewEvidenceStore(dbs[0])
	}
	h := &handler{repoRoot: repoRoot, evidence: evidence}
	// Warm the coverage report in the background at startup. The first
	// computation costs ~45s because it runs the full gate suite including the
	// toolchain-spawning `types` runner; paying that on a user's first page
	// view is what made the coverage page appear broken. Detached from startup
	// so it never delays the health check — until it lands, a cold request
	// still computes synchronously and is correct, just slow.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_, _ = h.report(ctx)
	}()
	path, service := catalogconnect.NewCatalogServiceHandler(h)
	return module.Module{
		Name:      "catalog",
		Mount:     func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: service}) },
		Endpoints: Endpoints,
	}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "catalog_coverage", Path: catalogconnect.CatalogServiceGetCoverageProcedure, Method: "POST", Summary: "Report achieved catalog maturity", Category: "catalog"},
	{ID: "catalog_next", Path: catalogconnect.CatalogServiceListNextWorkProcedure, Method: "POST", Summary: "Rank catalog next work", Category: "catalog"},
	{ID: "catalog_gate", Path: catalogconnect.CatalogServiceRunGateProcedure, Method: "POST", Summary: "Run a declarative catalog gate", Category: "catalog"},
	{ID: "catalog_graph", Path: catalogconnect.CatalogServiceGetAssetRelationshipsProcedure, Method: "POST", Summary: "Read catalog asset relationships", Category: "catalog"},
	{ID: "catalog_structure", Path: catalogconnect.CatalogServiceGetCatalogStructureProcedure, Method: "POST", Summary: "Read catalog structure", Category: "catalog"},
	{ID: "catalog_reconcile", Path: catalogconnect.CatalogServiceReconcileGraphProcedure, Method: "POST", Summary: "Reconcile catalog dependency graphs", Category: "catalog"},
	{ID: "catalog_ports", Path: catalogconnect.CatalogServiceGetAssetPortContractProcedure, Method: "POST", Summary: "Read asset host obligations", Category: "catalog"},
}

// report serves the coverage projection through the revision-keyed cache. The
// underlying computation executes every gate runner including the toolchain-
// spawning `types` gate, so it must not run once per request; see
// report_cache.go for the measurement that motivated this.
func (h *handler) report(ctx context.Context) (*catalogcoverage.Report, error) {
	return h.reports.get(ctx, h.repoRoot, h.computeReport)
}

func (h *handler) computeReport(ctx context.Context) (*catalogcoverage.Report, error) {
	assets, err := catalogcoverage.LoadCatalog(filepath.Join(h.repoRoot, "scenarios", "react-component-library", "catalog"))
	if err != nil {
		return nil, err
	}
	impls, err := catalogcoverage.LoadImplementations(filepath.Join(h.repoRoot, "scenarios", "react-component-library", "library"))
	if err != nil {
		return nil, err
	}
	gates, err := catalogcoverage.LoadGateDefinitions(filepath.Join(h.repoRoot, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		return nil, err
	}
	evidence, err := catalogcoverage.MergeExperienceEvidence(ctx, h.repoRoot, h.evidence, catalogexperience.Fetcher(h.repoRoot))
	if err != nil {
		return nil, err
	}
	return func() *catalogcoverage.Report {
		r := catalogcoverage.ComputeWithEvidence(assets, impls, evidence, gates)
		return &r
	}(), nil
}

func (h *handler) GetCoverage(ctx context.Context, _ *connect.Request[catalogv1.GetCoverageRequest]) (*connect.Response[catalogv1.GetCoverageResponse], error) {
	report, err := h.report(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("compute catalog coverage: %w", err))
	}
	return connect.NewResponse(&catalogv1.GetCoverageResponse{Report: toProto(report)}), nil
}

func (h *handler) ListNextWork(ctx context.Context, req *connect.Request[catalogv1.ListNextWorkRequest]) (*connect.Response[catalogv1.ListNextWorkResponse], error) {
	report, err := h.report(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("compute catalog next work: %w", err))
	}
	rows := catalogcoverage.NextWork(*report, int(req.Msg.GetLimit()))
	out := make([]*catalogv1.CoverageRow, 0, len(rows))
	for i := range rows {
		out = append(out, rowProto(rows[i]))
	}
	return connect.NewResponse(&catalogv1.ListNextWorkResponse{Rows: out, Maturity: maturityProto(report.Maturity)}), nil
}

func (h *handler) RunGate(ctx context.Context, req *connect.Request[catalogv1.RunGateRequest]) (*connect.Response[catalogv1.RunGateResponse], error) {
	gate := strings.TrimSpace(req.Msg.GetGate())
	var (
		result gates.Result
		err    error
	)
	switch gate {
	case "types":
		result, err = gates.ValidateTypes(h.repoRoot)
	case "api":
		result, err = gates.ValidateAPI(h.repoRoot)
	case "tokens":
		result, err = gates.ValidateTokens(h.repoRoot)
	case "token-vocabulary":
		result, err = gates.ValidateTokenVocabulary(h.repoRoot)
	case "token-ramp-complete":
		result, err = gates.ValidateTokenRampComplete(h.repoRoot)
	case "released-version-immutable":
		result, err = gates.ValidateReleasedVersionImmutable(h.repoRoot)
	case "lifecycle":
		result, err = gates.ValidateLifecycle(h.repoRoot)
	case "fixture-adversarial":
		result, err = gates.ValidateFixtures(h.repoRoot)
	case "examples":
		result, err = gates.ValidateExamples(h.repoRoot)
	case "graph-reconciled":
		result, err = gates.ValidateGraphReconciled(h.repoRoot)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown catalog gate %q", gate))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("run catalog gate %q: %w", gate, err))
	}
	// Severity is the gate's declared blocking flag, not a constant. Reporting
	// every finding as "error" made the non-blocking gates (graph-reconciled,
	// forced-colors, documentation, migration) indistinguishable from the
	// blocking ones, so a reader had no way to tell reported drift from a
	// release-stopping defect.
	severity := "error"
	if definitions, defErr := catalogcoverage.LoadGateDefinitions(filepath.Join(h.repoRoot, "scenarios", "react-component-library", "catalog", "config.json")); defErr == nil {
		for _, definition := range definitions {
			if definition.ID == gate && !definition.Blocking {
				severity = "warning"
				break
			}
		}
	}
	response := &catalogv1.RunGateResponse{Gate: gate, InspectedFiles: int32(result.Inspected), Findings: make([]*catalogv1.GateFinding, 0, len(result.Findings))}
	for _, finding := range result.Findings {
		response.Findings = append(response.Findings, &catalogv1.GateFinding{
			Code:        finding.Code,
			Message:     finding.Message,
			AssetId:     finding.AssetID,
			Severity:    severity,
			File:        finding.File,
			Line:        int32(finding.Line),
			Remediation: finding.Remediation,
			DocsRef:     finding.DocsRef,
		})
	}
	return connect.NewResponse(response), nil
}

func toProto(report *catalogcoverage.Report) *catalogv1.CoverageReport {
	out := &catalogv1.CoverageReport{Totals: map[string]int32{}, Maturity: maturityProto(report.Maturity)}
	for _, row := range report.Rows {
		out.Rows = append(out.Rows, rowProto(row))
	}
	for key, value := range report.Totals {
		out.Totals[string(key)] = int32(value)
	}
	for key, value := range report.ByDomain {
		out.ByDomain = append(out.ByDomain, &catalogv1.Rollup{Key: key, Planned: int32(value.Planned), Built: int32(value.Built)})
	}
	for key, value := range report.ByPriority {
		out.ByPriority = append(out.ByPriority, &catalogv1.Rollup{Key: key, Planned: int32(value.Planned), Built: int32(value.Built)})
	}
	return out
}

func rowProto(row catalogcoverage.Row) *catalogv1.CoverageRow {
	return &catalogv1.CoverageRow{AssetId: row.AssetID, Name: row.Name, Domain: row.Domain, Kind: row.Kind, Priority: row.Priority, Bucket: string(row.Bucket), Platform: row.Platform, Target: row.Target, Achieved: string(row.Achieved), Implementation: row.Implementation, BlocksDownstream: int32(row.BlocksDownstream), Rung: int32(row.Rung), RungName: row.RungName, DomainOrder: int32(row.DomainOrder)}
}

func maturityProto(m catalogcoverage.MaturityCoverage) *catalogv1.MaturitySummary {
	out := &catalogv1.MaturitySummary{Total: int32(m.Total), AtOrAboveTarget: int32(m.AtOrAboveTarget), ByRung: map[string]int32{}}
	for key, value := range m.ByRung {
		out.ByRung[string(key)] = int32(value)
	}
	out.CatalogCompletion = metricProto(m.CatalogCompletion)
	out.MandatoryGateCoverage = metricProto(m.MandatoryGateCoverage)
	out.WeightedQuality = metricProto(m.WeightedQuality)
	out.ProductionReadyCoverage = metricProto(m.ProductionReadyCoverage)
	return out
}

func metricProto(metric catalogcoverage.CoverageMetric) *catalogv1.CoverageMetric {
	return &catalogv1.CoverageMetric{Numerator: int32(metric.Numerator), Denominator: int32(metric.Denominator), Ratio: metric.Ratio}
}
