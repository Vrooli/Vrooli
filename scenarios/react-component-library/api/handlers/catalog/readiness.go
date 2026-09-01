package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/storage"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	"react-component-library/internal/catalogconfig"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/gates"
)

type readinessConfigFile struct {
	AdoptionMaturityFloor string `json:"adoptionMaturityFloor"`
	Gates                 []struct {
		ID          string   `json:"id"`
		Blocking    bool     `json:"blocking"`
		Attribution string   `json:"attribution"`
		AppliesTo   []string `json:"appliesTo"`
	} `json:"gates"`
}

type readinessFindingsFile struct {
	RunID       string `json:"runId"`
	CompletedAt string `json:"completedAt"`
}

type readinessManifestFile struct {
	RunID       string `json:"run_id"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
}

type readinessIndexEntry struct {
	RunID     string `json:"run_id"`
	StartedAt string `json:"started_at"`
}

func (h *handler) GetReadiness(ctx context.Context, req *connect.Request[catalogv1.GetReadinessRequest]) (*connect.Response[catalogv1.GetReadinessResponse], error) {
	report, err := h.readinessReport()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("compute catalog readiness: %w", err))
	}
	configPath := filepath.Join(h.repoRoot, "scenarios", "react-component-library", "catalog", "config.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read catalog readiness config: %w", err))
	}
	var config readinessConfigFile
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("parse catalog readiness config: %w", err))
	}
	floor, err := catalogconfig.DeclaredMaturityFloor(configPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if req != nil && req.Msg != nil && strings.TrimSpace(req.Msg.GetFloor()) != "" {
		floor = strings.TrimSpace(req.Msg.GetFloor())
	}
	if !validReadinessRung(floor) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported readiness floor %q", floor))
	}
	triage, triageOmitted := readinessTriageWithOmitted(report)
	run := readinessRun(h.repoRoot)
	verdict := "READY"
	if !run.Completed || report.Maturity.MandatoryGateCoverage.Ratio < 1 {
		verdict = "NOT_READY"
	}
	configProjection := readinessConfigProjection(config, report, floor)
	h.quarantineMu.RLock()
	configProjection.QuarantinedGates = int32(len(h.quarantined))
	h.quarantineMu.RUnlock()
	return connect.NewResponse(&catalogv1.GetReadinessResponse{
		Coverage:           toProto(report),
		Run:                run,
		Config:             configProjection,
		Triage:             triage,
		NextSteps:          readinessNextSteps(triage),
		Verdict:            verdict,
		TriageOmittedCount: int32(triageOmitted),
	}), nil
}

// readinessReport intentionally never triggers the full live gate suite. A
// readiness query is an operational summary backed by the durable run below;
// if the report cache is cold, a zero-evidence projection still tells the
// caller the declared floor and that the catalog is not yet certified.
func (h *handler) readinessReport() (*catalogcoverage.Report, error) {
	if report := h.reports.peek(); report != nil {
		return report, nil
	}
	base := filepath.Join(h.repoRoot, "scenarios", "react-component-library")
	assets, err := catalogcoverage.LoadCatalog(filepath.Join(base, "catalog"))
	if err != nil {
		return nil, err
	}
	impls, err := catalogcoverage.LoadImplementations(filepath.Join(base, "library"))
	if err != nil {
		return nil, err
	}
	definitions, err := catalogcoverage.LoadGateDefinitions(filepath.Join(base, "catalog", "config.json"))
	if err != nil {
		return nil, err
	}
	report := catalogcoverage.ComputeWithEvidence(assets, impls, nil, definitions)
	return &report, nil
}

func readinessConfigProjection(config readinessConfigFile, report *catalogcoverage.Report, floors ...string) *catalogv1.ReadinessConfig {
	floor := config.AdoptionMaturityFloor
	if len(floors) > 0 && strings.TrimSpace(floors[0]) != "" {
		floor = floors[0]
	}
	achieved := achievedRung(report)
	attributable, corpus := 0, 0
	for _, definition := range gates.Definitions() {
		if definition.Run == nil {
			continue
		}
		if definition.CorpusScoped {
			corpus++
		} else {
			attributable++
		}
	}
	return &catalogv1.ReadinessConfig{
		DeclaredFloor:     floor,
		AchievedRung:      achieved,
		RungGap:           rungGap(floor, achieved),
		BlockingGates:     int32(countGates(config.Gates, func(g readinessConfigFileGate) bool { return g.Blocking })),
		AdvisoryGates:     int32(countGates(config.Gates, func(g readinessConfigFileGate) bool { return !g.Blocking })),
		QuarantinedGates:  int32(0),
		AttributableGates: int32(attributable),
		CorpusGates:       int32(corpus),
	}
}

type readinessConfigFileGate struct {
	ID          string
	Blocking    bool
	Attribution string
}

func countGates(gates []struct {
	ID          string   `json:"id"`
	Blocking    bool     `json:"blocking"`
	Attribution string   `json:"attribution"`
	AppliesTo   []string `json:"appliesTo"`
}, predicate func(readinessConfigFileGate) bool,
) int {
	n := 0
	for _, gate := range gates {
		if predicate(readinessConfigFileGate{ID: gate.ID, Blocking: gate.Blocking, Attribution: gate.Attribution}) {
			n++
		}
	}
	return n
}

func achievedRung(report *catalogcoverage.Report) string {
	if report == nil {
		return "scaffolded"
	}
	if report.Maturity.ProductionReadyCoverage.Denominator > 0 && report.Maturity.ProductionReadyCoverage.Ratio >= 1 {
		return "production-ready"
	}
	if report.Maturity.MandatoryGateCoverage.Denominator > 0 && report.Maturity.MandatoryGateCoverage.Ratio >= 1 {
		return "verified"
	}
	if report.Maturity.CatalogCompletion.Denominator > 0 && report.Maturity.CatalogCompletion.Ratio >= 1 {
		return "implemented"
	}
	return "scaffolded"
}

func rungGap(floor, achieved string) int32 {
	ranks := map[string]int{"scaffolded": 0, "implemented": 1, "verified": 2, "production-ready": 3}
	return int32(ranks[floor] - ranks[achieved])
}

func validReadinessRung(value string) bool {
	switch value {
	case "scaffolded", "implemented", "verified", "production-ready":
		return true
	default:
		return false
	}
}

func readinessRun(repoRoot string) *catalogv1.ReadinessRun {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return &catalogv1.ReadinessRun{EvidenceAge: "invalid"}
	}
	gateRoot, err := resolver.ArtifactPath(storage.Options{ScenarioID: "react-component-library"}, storage.ArtifactRef{
		Owner: "react-component-library", Domain: "gates", Class: storage.ClassState,
	})
	if err != nil {
		return &catalogv1.ReadinessRun{EvidenceAge: "invalid"}
	}
	findingsData, findingsErr := os.ReadFile(filepath.Join(gateRoot, "latest", "findings.json"))
	manifestData, manifestErr := os.ReadFile(filepath.Join(gateRoot, "latest", "manifest.json"))
	if findingsErr != nil && manifestErr != nil {
		return &catalogv1.ReadinessRun{EvidenceAge: "missing"}
	}
	var latest readinessFindingsFile
	if findingsErr == nil {
		if json.Unmarshal(findingsData, &latest) != nil {
			return &catalogv1.ReadinessRun{EvidenceAge: "invalid"}
		}
	}
	var manifest readinessManifestFile
	if manifestErr == nil && json.Unmarshal(manifestData, &manifest) != nil {
		return &catalogv1.ReadinessRun{EvidenceAge: "invalid"}
	}
	// The manifest is the authoritative run identity. Test Genie can update it
	// before the findings projection is copied to latest/, so a stale findings
	// file must not make readiness point at an older run.
	runID := manifest.RunID
	if runID == "" {
		runID = latest.RunID
	}
	completedAt := manifest.CompletedAt
	if completedAt == "" && latest.RunID == runID {
		completedAt = latest.CompletedAt
	}
	startedAt := manifest.StartedAt
	if startedAt == "" {
		startedAt = readinessIndexStartedAt(gateRoot, runID)
	}
	completed := runID != "" && completedAt != "" && !strings.HasPrefix(completedAt, "0001-")
	age := "unknown"
	if parsed, err := time.Parse(time.RFC3339, completedAt); err == nil {
		if elapsed := time.Since(parsed); elapsed >= 0 {
			age = elapsed.Round(time.Minute).String() + " old"
		} else {
			age = "future"
		}
	}
	return &catalogv1.ReadinessRun{RunId: runID, StartedAt: startedAt, CompletedAt: completedAt, Completed: completed, EvidenceAge: age}
}

func readinessIndexStartedAt(coverageDir, runID string) string {
	data, err := os.ReadFile(filepath.Join(coverageDir, "runs.index.json"))
	if err != nil {
		return ""
	}
	var entries []readinessIndexEntry
	if json.Unmarshal(data, &entries) != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.RunID == runID {
			return entry.StartedAt
		}
	}
	return ""
}

func readinessTriage(report *catalogcoverage.Report) []*catalogv1.ReadinessTriageRow {
	rows, _ := readinessTriageWithOmitted(report)
	return rows
}

func readinessTriageWithOmitted(report *catalogcoverage.Report) ([]*catalogv1.ReadinessTriageRow, int) {
	rows := make([]*catalogv1.ReadinessTriageRow, 0)
	for _, row := range report.Rows {
		for _, gate := range row.FailedGates {
			rows = append(rows, &catalogv1.ReadinessTriageRow{Gate: gate, AssetId: row.AssetID, Message: fmt.Sprintf("%s failed %s", row.Name, gate), NearestBlockingGate: row.NearestBlockingGate, BlocksDownstream: int32(row.BlocksDownstream), Weight: row.Weight})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		left := float64(rows[i].BlocksDownstream) * rows[i].Weight
		right := float64(rows[j].BlocksDownstream) * rows[j].Weight
		if left != right {
			return left > right
		}
		return rows[i].AssetId < rows[j].AssetId
	})
	omitted := 0
	if len(rows) > 50 {
		omitted = len(rows) - 50
		rows = rows[:50]
	}
	return rows, omitted
}

func readinessNextSteps(rows []*catalogv1.ReadinessTriageRow) []string {
	if len(rows) == 0 {
		return []string{"react-component-library catalog readiness --json"}
	}
	return []string{"react-component-library catalog gates " + rows[0].Gate + " --json", "react-component-library catalog readiness --json"}
}
