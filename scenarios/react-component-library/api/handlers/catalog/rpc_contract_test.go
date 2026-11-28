package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	"react-component-library/internal/componenttests"
)

type captureDependencyReader struct{ components.DependencyReader }

type captureExecutor struct{}

func (captureExecutor) ExecuteStory(context.Context, string, string, string) (componenttests.StoryExecution, error) {
	return componenttests.StoryExecution{}, nil
}

func TestCoverageAndNextWorkRPCProjectCachedReport(t *testing.T) {
	root := t.TempDir()
	fingerprintValue, err := fingerprint(root)
	require.NoError(t, err)
	rows := []catalogcoverage.Row{
		{AssetID: "promote", Name: "Promote", Bucket: catalogcoverage.BucketPlannedBuilt, Achieved: catalogcoverage.RungVerified, FailedGates: []string{"accessibility"}, Weight: 2, AssetScore: .7},
		{AssetID: "build", Name: "Build", Bucket: catalogcoverage.BucketPlannedUnbuilt, Achieved: catalogcoverage.RungMissing, Weight: 1},
	}
	report := &catalogcoverage.Report{
		Rows: rows, Totals: map[catalogcoverage.Bucket]int{catalogcoverage.BucketPlannedBuilt: 1},
		ByDomain:   map[string]catalogcoverage.DomainCount{"forms": {Planned: 2, Built: 1}},
		ByPriority: map[string]catalogcoverage.DomainCount{"P1": {Planned: 2, Built: 1}},
		Maturity:   catalogcoverage.MaturityCoverage{Total: 2, AtOrAboveTarget: 1, ByRung: map[catalogcoverage.AchievedRung]int{catalogcoverage.RungVerified: 1}, CatalogCompletion: catalogcoverage.CoverageMetric{Numerator: 1, Denominator: 2, Ratio: .5}, MandatoryGateCoverage: catalogcoverage.CoverageMetric{Numerator: 1, Denominator: 2, Ratio: .5}, WeightedQuality: catalogcoverage.CoverageMetric{Numerator: 1, Denominator: 2, Ratio: .5}, ProductionReadyCoverage: catalogcoverage.CoverageMetric{Numerator: 0, Denominator: 2}},
		ByGate:     map[string]catalogcoverage.ScoreBreakdown{"accessibility": {Passed: 1, Applicable: 2, Score: .5}}, ByRungScore: map[catalogcoverage.AchievedRung]catalogcoverage.ScoreBreakdown{catalogcoverage.RungVerified: {Passed: 1, Applicable: 2, Score: .5}},
		Corpus: []catalogcoverage.CorpusStatus{{Gate: "release", Result: "pass", FindingCount: 2}}, CompositionScores: map[string]float64{"promote": .8}, CompositionMedian: .8, BespokeEscapeCount: 1,
	}
	cache := reportCache{report: report, fingerprint: fingerprintValue, fpAt: time.Now(), fpValue: fingerprintValue}
	h := &handler{repoRoot: root}
	h.reports.report = cache.report
	h.reports.fingerprint = cache.fingerprint
	h.reports.fpAt = cache.fpAt
	h.reports.fpValue = cache.fpValue

	coverage, err := h.GetCoverage(context.Background(), connect.NewRequest(&catalogv1.GetCoverageRequest{}))
	require.NoError(t, err)
	require.EqualValues(t, 2, coverage.Msg.Report.Maturity.Total)
	require.EqualValues(t, 1, coverage.Msg.Report.Totals[string(catalogcoverage.BucketPlannedBuilt)])
	require.Equal(t, .8, coverage.Msg.Report.CompositionScores["promote"])
	require.Equal(t, "release", coverage.Msg.Report.Maturity.Corpus[0].Gate)

	for _, lane := range []string{"promote", "build", "all", "unknown"} {
		response, listErr := h.ListNextWork(context.Background(), connect.NewRequest(&catalogv1.ListNextWorkRequest{Lane: lane}))
		require.NoError(t, listErr)
		wantLane := lane
		if wantLane == "unknown" {
			wantLane = "promote"
		}
		require.Equal(t, wantLane, response.Msg.Lane)
		require.Len(t, response.Msg.Promote, 1)
		require.Len(t, response.Msg.Build, 1)
	}
}

func TestCatalogReportRPCsMapWarmingStateToTransportErrors(t *testing.T) {
	previousGrace := reportWarmGrace
	reportWarmGrace = 0
	t.Cleanup(func() { reportWarmGrace = previousGrace })

	h := &handler{repoRoot: t.TempDir()}
	h.reports.refreshing = true
	h.reports.ready = make(chan struct{})
	ctx := context.Background()

	_, err := h.GetCoverage(ctx, connect.NewRequest(&catalogv1.GetCoverageRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))

	_, err = h.ListNextWork(ctx, connect.NewRequest(&catalogv1.ListNextWorkRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestReadinessAndSimpleCatalogRPCContracts(t *testing.T) {
	h := &handler{repoRoot: catalogTestRepoRoot(t)}
	ctx := context.Background()
	readiness, err := h.GetReadiness(ctx, connect.NewRequest(&catalogv1.GetReadinessRequest{}))
	require.NoError(t, err)
	require.NotNil(t, readiness.Msg.Coverage)
	require.NotEmpty(t, readiness.Msg.Config.DeclaredFloor)
	require.NotEmpty(t, readiness.Msg.Verdict)

	_, err = h.GetReadiness(ctx, connect.NewRequest(&catalogv1.GetReadinessRequest{Floor: "not-a-rung"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	history, err := (&handler{}).GetScoreHistory(ctx, connect.NewRequest(&catalogv1.GetScoreHistoryRequest{}))
	require.NoError(t, err)
	require.Empty(t, history.Msg.Points)

	_, err = (&handler{}).CaptureEvidence(ctx, connect.NewRequest(&catalogv1.CaptureEvidenceRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestScoreHistoryRPCProjectsPersistedHistory(t *testing.T) {
	ctx := context.Background()
	database := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, database, apidb.SchemaProviderFunc(catalogcoverage.Schema)))
	store := catalogcoverage.NewEvidenceStore(database)
	require.NoError(t, store.Save(ctx, []catalogcoverage.GateEvidence{{
		AssetID: "react-component-library:NotificationBadge", Target: "react-vite", Gate: "accessibility", Version: "1.0.0", Result: "pass", SourceRevision: "rev-1",
	}}))
	h := &handler{repoRoot: catalogTestRepoRoot(t), evidence: store}
	history, err := h.GetScoreHistory(ctx, connect.NewRequest(&catalogv1.GetScoreHistoryRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, history.Msg.Points)
	require.NotEmpty(t, history.Msg.Points[0].RecordedAt)
	require.NotEmpty(t, history.Msg.Points[0].SourceRevision)
	_, err = h.GetScoreHistory(ctx, connect.NewRequest(&catalogv1.GetScoreHistoryRequest{Since: "last-week"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestHealthOverviewRPCProjectsCachedHealthGraph(t *testing.T) {
	root := catalogTestRepoRoot(t)
	fingerprintValue, err := fingerprint(root)
	require.NoError(t, err)
	cache := reportCache{
		report: &catalogcoverage.Report{
			Rows:     []catalogcoverage.Row{{AssetID: "react-component-library:NotificationBadge", Bucket: catalogcoverage.BucketPlannedBuilt, AssetScore: .95, Weight: 2, NewestEvidence: time.Now().UTC().Format(time.RFC3339Nano), VisualEvidence: true}},
			Maturity: catalogcoverage.MaturityCoverage{Total: 1, AtOrAboveTarget: 1},
		},
		fingerprint: fingerprintValue, fpAt: time.Now(), fpValue: fingerprintValue,
	}
	h := &handler{repoRoot: root, quarantined: map[string]bool{"visual": true}}
	h.reports.report = cache.report
	h.reports.fingerprint = cache.fingerprint
	h.reports.fpAt = cache.fpAt
	h.reports.fpValue = cache.fpValue
	response, err := h.GetHealthOverview(context.Background(), connect.NewRequest(&catalogv1.GetHealthOverviewRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, response.Msg.Nodes)
	require.NotEmpty(t, response.Msg.Edges)
	require.Contains(t, response.Msg.QuarantinedGates, "visual")
	var found bool
	for _, node := range response.Msg.Nodes {
		if node.Asset.GetAssetId() == "react-component-library:NotificationBadge" {
			found = true
			require.Equal(t, "healthy", node.Health)
			require.True(t, node.VisualCurrent)
		}
	}
	require.True(t, found)
}

func TestReadinessReportsRepositoryAndConfigErrors(t *testing.T) {
	ctx := context.Background()
	_, err := (&handler{repoRoot: t.TempDir()}).GetReadiness(ctx, connect.NewRequest(&catalogv1.GetReadinessRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))

	root := t.TempDir()
	base := filepath.Join(root, "scenarios", "react-component-library")
	require.NoError(t, os.MkdirAll(filepath.Join(base, "catalog"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(base, "library"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(base, "catalog", "config.json"), []byte("{"), 0o600))
	_, err = (&handler{repoRoot: root}).GetReadiness(ctx, connect.NewRequest(&catalogv1.GetReadinessRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestCatalogRPCErrorCodesForRequestValidation(t *testing.T) {
	h := &handler{repoRoot: catalogTestRepoRoot(t)}
	_, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	_, err = h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "react-component-library:NotificationBadge", Profile: "unknown"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	h.captureRunner = &componenttests.Runner{Assets: captureDependencyReader{}, Executor: captureExecutor{}}
	_, err = h.CaptureEvidence(context.Background(), connect.NewRequest(&catalogv1.CaptureEvidenceRequest{AssetId: " "}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCoverageProtoIncludesQualityGateOptOutCount(t *testing.T) {
	report := &catalogcoverage.Report{QualityGateOptOutCount: 3}
	got := maturityProto(report).GetQualityGateOptOutCount()
	require.Equal(t, int32(3), got)
}
