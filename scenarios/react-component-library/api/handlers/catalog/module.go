package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

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
}

// Module exposes the same live coverage projection used by the component-test
// phase. No CLI-only computation or second catalog join is permitted.
func Module(repoRoot string, dbs ...*sql.DB) module.Module {
	var evidence *catalogcoverage.EvidenceStore
	if len(dbs) > 0 && dbs[0] != nil {
		evidence = catalogcoverage.NewEvidenceStore(dbs[0])
	}
	path, service := catalogconnect.NewCatalogServiceHandler(&handler{repoRoot: repoRoot, evidence: evidence})
	return module.Module{
		Name:  "catalog",
		Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: service}) },
		Endpoints: []module.EndpointDescriptor{
			{ID: "catalog_coverage", Path: catalogconnect.CatalogServiceGetCoverageProcedure, Method: "POST", Summary: "Report achieved catalog maturity", Category: "catalog"},
			{ID: "catalog_next", Path: catalogconnect.CatalogServiceListNextWorkProcedure, Method: "POST", Summary: "Rank catalog next work", Category: "catalog"},
			{ID: "catalog_gate", Path: catalogconnect.CatalogServiceRunGateProcedure, Method: "POST", Summary: "Run a declarative catalog gate", Category: "catalog"},
		},
	}
}

func (h *handler) report(ctx context.Context) (*catalogcoverage.Report, error) {
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
	case "lifecycle":
		result, err = gates.ValidateLifecycle(h.repoRoot)
	case "fixture-adversarial":
		result, err = gates.ValidateFixtures(h.repoRoot)
	case "examples":
		result, err = gates.ValidateExamples(h.repoRoot)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown catalog gate %q", gate))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("run catalog gate %q: %w", gate, err))
	}
	response := &catalogv1.RunGateResponse{Gate: gate, InspectedFiles: int32(result.Inspected), Findings: make([]*catalogv1.GateFinding, 0, len(result.Findings))}
	for _, finding := range result.Findings {
		response.Findings = append(response.Findings, &catalogv1.GateFinding{Code: finding.Code, Message: finding.Message, AssetId: finding.AssetID, Severity: "error"})
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
	return &catalogv1.CoverageRow{AssetId: row.AssetID, Name: row.Name, Domain: row.Domain, Kind: row.Kind, Priority: row.Priority, Bucket: string(row.Bucket), Platform: row.Platform, Target: row.Target, Achieved: string(row.Achieved), Implementation: row.Implementation, BlocksDownstream: int32(row.BlocksDownstream)}
}

func maturityProto(m catalogcoverage.MaturityCoverage) *catalogv1.MaturitySummary {
	out := &catalogv1.MaturitySummary{Total: int32(m.Total), AtOrAboveTarget: int32(m.AtOrAboveTarget), ByRung: map[string]int32{}}
	for key, value := range m.ByRung {
		out.ByRung[string(key)] = int32(value)
	}
	return out
}
