package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage"
	coverageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage/coverage_v1connect"
	drillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/drills"
	drillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/drills/drills_v1connect"
	restoresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/restores"
	restoresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/restores/restores_v1connect"
	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety"
	safetyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety/safety_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type recoveryCoverageFixture struct {
	coverageconnect.UnimplementedCoverageServiceHandler
}

func (recoveryCoverageFixture) GetCoverageReport(context.Context, *connect.Request[coveragev1.GetCoverageReportRequest]) (*connect.Response[coveragev1.GetCoverageReportResponse], error) {
	return connect.NewResponse(&coveragev1.GetCoverageReportResponse{Report: &coveragev1.CoverageReport{Summary: &coveragev1.CoverageSummary{
		RegisteredCount: 1, PlannedCount: 1, BackedUpCount: 1, VerifiedCount: 1,
	}}}), nil
}

type recoveryDrillFixture struct {
	drillsconnect.UnimplementedRecoveryDrillsServiceHandler
}

func (recoveryDrillFixture) ListDrills(context.Context, *connect.Request[drillsv1.ListDrillsRequest]) (*connect.Response[drillsv1.ListDrillsResponse], error) {
	return connect.NewResponse(&drillsv1.ListDrillsResponse{Drills: []*drillsv1.RecoveryDrill{{
		Id: "drill-fixture", SnapshotId: "snapshot-fixture", RestoreId: "restore-fixture",
		Status: drillsv1.DrillStatus_DRILL_STATUS_VERIFIED, FinishedAt: timestamppb.New(time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)),
	}}}), nil
}

type recoveryRestoreFixture struct {
	restoresconnect.UnimplementedRestoresServiceHandler
}

func (recoveryRestoreFixture) GetRestore(context.Context, *connect.Request[restoresv1.GetRestoreRequest]) (*connect.Response[restoresv1.GetRestoreResponse], error) {
	return connect.NewResponse(&restoresv1.GetRestoreResponse{Restore: &restoresv1.Restore{
		Id: "restore-fixture", Status: restoresv1.RestoreStatus_RESTORE_STATUS_VERIFIED,
		Checksum: "fixture-checksum", FinishedAt: timestamppb.New(time.Date(2026, 9, 6, 12, 1, 0, 0, time.UTC)),
	}}), nil
}

type recoverySafetyFixture struct {
	safetyconnect.UnimplementedSafetyServiceHandler
	registered string
}

func (f *recoverySafetyFixture) RegisterScenarioTargets(_ context.Context, req *connect.Request[safetyv1.RegisterScenarioTargetsRequest]) (*connect.Response[safetyv1.RegisterScenarioTargetsResponse], error) {
	f.registered = req.Msg.GetScenario()
	return connect.NewResponse(&safetyv1.RegisterScenarioTargetsResponse{}), nil
}

func TestRegisterRecoveryTargetsUsesTypedBackupContract(t *testing.T) {
	mux := http.NewServeMux()
	fixture := &recoverySafetyFixture{}
	path, handler := safetyconnect.NewSafetyServiceHandler(fixture)
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("SECRETS_MANAGER_DATA_BACKUP_MANAGER_URL", server.URL)
	if err := registerRecoveryTargets(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fixture.registered != "secrets-manager" {
		t.Fatalf("registered scenario = %q, want secrets-manager", fixture.registered)
	}
}

func TestFetchRecoveryEvidenceUsesTypedBackupContracts(t *testing.T) {
	mux := http.NewServeMux()
	coveragePath, coverageHandler := coverageconnect.NewCoverageServiceHandler(recoveryCoverageFixture{})
	mux.Handle(coveragePath, coverageHandler)
	drillsPath, drillsHandler := drillsconnect.NewRecoveryDrillsServiceHandler(recoveryDrillFixture{})
	mux.Handle(drillsPath, drillsHandler)
	restoresPath, restoresHandler := restoresconnect.NewRestoresServiceHandler(recoveryRestoreFixture{})
	mux.Handle(restoresPath, restoresHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("SECRETS_MANAGER_DATA_BACKUP_MANAGER_URL", server.URL)
	evidence, err := fetchRecoveryEvidence(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !evidence.Ready || len(evidence.Evidence) != 2 {
		t.Fatalf("evidence = %+v, want ready coverage and drill evidence", evidence)
	}
	if evidence.Evidence[1].Checksum != "fixture-checksum" || !evidence.Evidence[1].Verified {
		t.Fatalf("drill evidence = %+v", evidence.Evidence[1])
	}
}
