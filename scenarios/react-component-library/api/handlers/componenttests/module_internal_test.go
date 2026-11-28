package componenttests

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	componenttestsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests"
	"react-component-library/internal/components"
	domain "react-component-library/internal/componenttests"
)

type reportListStub struct{ reports []domain.Report }

func (r reportListStub) Save(context.Context, domain.Report) error { return nil }
func (r reportListStub) Get(context.Context, string) (domain.Report, error) {
	return domain.Report{}, sql.ErrNoRows
}

func (r reportListStub) List(context.Context, string, string, int) ([]domain.Report, error) {
	return r.reports, nil
}

func TestErrorMapsValidationAndMissingReportToStableConnectCodes(t *testing.T) {
	handler := &connectHandler{logger: log.New(io.Discard, "", 0)}

	validation := handler.error(domain.ValidationError{Code: "INVALID_VERSION", Field: "version", Detail: "must be pinned"})
	var connectErr *connect.Error
	require.ErrorAs(t, validation, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())

	missing := handler.error(sql.ErrNoRows)
	require.ErrorAs(t, missing, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())

	internal := handler.error(os.ErrPermission)
	require.ErrorAs(t, internal, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())
}

func TestStoryContractPathRequiresAnExistingContractAndNonDraftVersion(t *testing.T) {
	root := t.TempDir()
	handler := &connectHandler{sourceRoot: root}
	version := components.ComponentVersion{ComponentID: "component-id", Version: "1.0.0", Status: components.VersionStatusReleased, SourcePath: "components/Button/versions/1.0.0/Button.tsx"}

	_, ok := handler.storyContractPath(version)
	require.False(t, ok, "missing story contract must not be treated as runnable evidence")
	storyPath := filepath.Join(root, "components", "Button", "versions", "1.0.0", "story.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(storyPath), 0o755))
	require.NoError(t, os.WriteFile(storyPath, []byte(`{"stories":[]}`), 0o644))
	resolved, ok := handler.storyContractPath(version)
	require.True(t, ok)
	require.Equal(t, storyPath, resolved)
	require.True(t, handler.sweepableVersion(version))

	version.Status = components.VersionStatusDraft
	require.False(t, handler.sweepableVersion(version), "drafts are not corpus evidence targets")
	version.SourcePath = " "
	require.False(t, handler.sweepableVersion(version), "versions without source placement are not sweepable")
}

func TestStoryEvidenceFreshUsesLatestMatchingReportAndContractTime(t *testing.T) {
	root := t.TempDir()
	storyPath := filepath.Join(root, "components", "Button", "versions", "1.0.0", "story.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(storyPath), 0o755))
	require.NoError(t, os.WriteFile(storyPath, []byte(`{"stories":[]}`), 0o644))
	version := components.ComponentVersion{ComponentID: "component-id", Version: "1.0.0", Status: components.VersionStatusReleased, SourcePath: "components/Button/versions/1.0.0/Button.tsx"}
	created := time.Now().Add(-time.Minute)
	service := domain.NewService(domain.Runner{}, reportListStub{reports: []domain.Report{{CreatedAt: created}}})
	handler := &connectHandler{sourceRoot: root, service: service}
	require.False(t, handler.storyEvidenceFresh(context.Background(), version), "a prior report must not cover a newer contract")
	created = time.Now().Add(time.Minute)
	handler.service = domain.NewService(domain.Runner{}, reportListStub{reports: []domain.Report{{CreatedAt: created}}})
	require.True(t, handler.storyEvidenceFresh(context.Background(), version), "a report newer than the contract is fresh")

	handler.service = domain.NewService(domain.Runner{}, reportListStub{})
	require.False(t, handler.storyEvidenceFresh(context.Background(), version), "freshness requires an existing report")
}

func TestToProtoPreservesReportEvidenceAndArtifacts(t *testing.T) {
	created := time.Date(2026, 11, 20, 12, 30, 0, 0, time.UTC)
	report := domain.Report{
		ID: "report-1", RootLibraryID: "rcl:Button", RootVersion: "1.0.0", IncludeClosure: true,
		CreatedAt: created, Verdict: domain.VerdictPassed, SourceRevision: "abc123",
		Results:   []domain.Result{{Stage: domain.StageDeclared, AssetLibraryID: "rcl:Button", Version: "1.0.0", Subject: "idle", Verdict: domain.VerdictPassed, Evidence: []domain.Evidence{{Kind: "console", Artifacts: []string{"capture.png"}}}}},
		Artifacts: []domain.Artifact{{Kind: "screenshot", Label: "Idle state", StoryID: "idle", AssetLibraryID: "rcl:Button", Version: "1.0.0", Reference: "capture.png"}},
	}

	encoded := toProto(report)
	require.Equal(t, report.ID, encoded.GetId())
	require.Equal(t, "abc123", encoded.GetSourceRevision())
	require.Equal(t, created, encoded.GetCreatedAt().AsTime())
	require.True(t, encoded.GetIncludeClosure())
	require.Len(t, encoded.GetResults(), 1)
	require.Equal(t, "idle", encoded.GetResults()[0].GetSubject())
	require.Len(t, encoded.GetResults()[0].GetEvidence(), 1)
	var evidence map[string]any
	require.NoError(t, json.Unmarshal([]byte(encoded.GetResults()[0].GetEvidence()[0].GetJson()), &evidence))
	// Evidence payload is intentionally JSON so clients can evolve alongside evidence kinds.
	require.Equal(t, "console", evidence["kind"])
	require.Equal(t, []any{"capture.png"}, evidence["artifacts"])
	require.Len(t, encoded.GetArtifacts(), 1)
	require.Equal(t, "capture.png", encoded.GetArtifacts()[0].GetReference())
}

type rpcReportRepository struct {
	getID           string
	listComponentID string
	listVersion     string
	listLimit       int
	getReport       domain.Report
	listReports     []domain.Report
	getErr, listErr error
}

func (r *rpcReportRepository) Save(context.Context, domain.Report) error { return nil }
func (r *rpcReportRepository) Get(_ context.Context, id string) (domain.Report, error) {
	r.getID = id
	return r.getReport, r.getErr
}

func (r *rpcReportRepository) List(_ context.Context, componentID, version string, limit int) ([]domain.Report, error) {
	r.listComponentID, r.listVersion, r.listLimit = componentID, version, limit
	return r.listReports, r.listErr
}

func TestReportRPCsForwardFiltersAndReturnDurableReportFields(t *testing.T) {
	created := time.Date(2026, 11, 20, 12, 30, 0, 0, time.UTC)
	report := domain.Report{ID: "report-42", RootLibraryID: "rcl:Button", RootVersion: "2.1.0", IncludeClosure: true, CreatedAt: created, Verdict: domain.VerdictPassed, SourceRevision: "revision-42"}
	repository := &rpcReportRepository{getReport: report, listReports: []domain.Report{report}}
	handler := &connectHandler{service: domain.NewService(domain.Runner{}, repository)}

	got, err := handler.GetComponentTestReport(context.Background(), connect.NewRequest(&componenttestsv1.GetComponentTestReportRequest{Id: "report-42"}))
	require.NoError(t, err)
	require.Equal(t, "report-42", repository.getID)
	require.Equal(t, "report-42", got.Msg.GetReport().GetId())
	require.Equal(t, "rcl:Button", got.Msg.GetReport().GetRootLibraryId())
	require.Equal(t, "2.1.0", got.Msg.GetReport().GetRootVersion())
	require.Equal(t, "revision-42", got.Msg.GetReport().GetSourceRevision())
	require.True(t, got.Msg.GetReport().GetIncludeClosure())
	require.Equal(t, created, got.Msg.GetReport().GetCreatedAt().AsTime())

	listed, err := handler.ListComponentTestReports(context.Background(), connect.NewRequest(&componenttestsv1.ListComponentTestReportsRequest{ComponentId: "component-7", Version: "2.1.0", Limit: 4}))
	require.NoError(t, err)
	require.Equal(t, "component-7", repository.listComponentID)
	require.Equal(t, "2.1.0", repository.listVersion)
	require.Equal(t, 4, repository.listLimit)
	require.Len(t, listed.Msg.GetReports(), 1)
	require.Equal(t, report.ID, listed.Msg.GetReports()[0].GetId())
	require.Equal(t, "passed", listed.Msg.GetReports()[0].GetVerdict())
}

func TestReportRPCsMapRepositoryFailuresToConnectErrors(t *testing.T) {
	repository := &rpcReportRepository{getErr: sql.ErrNoRows, listErr: os.ErrPermission}
	handler := &connectHandler{service: domain.NewService(domain.Runner{}, repository), logger: log.New(io.Discard, "", 0)}

	_, err := handler.GetComponentTestReport(context.Background(), connect.NewRequest(&componenttestsv1.GetComponentTestReportRequest{Id: "missing"}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())
	require.Equal(t, "missing", repository.getID)

	_, err = handler.ListComponentTestReports(context.Background(), connect.NewRequest(&componenttestsv1.ListComponentTestReportsRequest{ComponentId: "component-7", Version: "2.1.0", Limit: 101}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())
	require.Equal(t, "component-7", repository.listComponentID)
	require.Equal(t, "2.1.0", repository.listVersion)
	require.Equal(t, 101, repository.listLimit)
}

func TestSweepRPCRequiresRegistryWiring(t *testing.T) {
	handler := &connectHandler{}
	_, err := handler.SweepComponentTests(context.Background(), connect.NewRequest(&componenttestsv1.SweepComponentTestsRequest{}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
	require.Contains(t, connectErr.Message(), "not wired to the component registry")
}
