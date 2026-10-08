package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
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

// recoveryEvidenceSnapshot is deliberately metadata-only. Data Backup Manager
// owns backup and restore execution; Secrets Manager only projects the evidence
// needed to explain whether recovery is currently proven.
type recoveryEvidenceSnapshot struct {
	BackupStatus  string
	RestoreStatus string
	RecoveryReady bool
	Evidence      []recoveryEvidenceReference
	Remediation   string
	UpdatedAt     time.Time
}

type recoveryEvidenceReference struct {
	Kind             string    `json:"kind"`
	ArtifactIdentity string    `json:"artifact_identity"`
	SourceGeneration string    `json:"source_generation,omitempty"`
	Checksum         string    `json:"checksum,omitempty"`
	ObservedAt       time.Time `json:"observed_at"`
	Verified         bool      `json:"verified"`
	Remediation      string    `json:"remediation,omitempty"`
}

func productionRecoveryEvidence(ctx context.Context) recoveryEvidenceSnapshot {
	if err := registerRecoveryTargets(ctx); err != nil {
		return recoveryEvidenceSnapshot{
			BackupStatus:  "unavailable",
			RestoreStatus: "unavailable",
			Remediation:   err.Error(),
			UpdatedAt:     time.Now().UTC(),
		}
	}
	evidence, err := fetchRecoveryEvidence(ctx)
	if err != nil {
		return recoveryEvidenceSnapshot{
			BackupStatus:  "unavailable",
			RestoreStatus: "unavailable",
			Remediation:   err.Error(),
			UpdatedAt:     time.Now().UTC(),
		}
	}

	backupStatus, restoreStatus := "not_configured", "not_run"
	for _, evidence := range evidence.Evidence {
		switch evidence.Kind {
		case "durable-backup-coverage":
			if evidence.Verified {
				backupStatus = "verified"
			} else if backupStatus == "not_configured" {
				backupStatus = "configured"
			}
		case "recovery-drill":
			if evidence.Verified {
				restoreStatus = "verified"
			} else if restoreStatus == "not_run" {
				restoreStatus = "incomplete"
			}
		}
	}

	return recoveryEvidenceSnapshot{
		BackupStatus:  backupStatus,
		RestoreStatus: restoreStatus,
		RecoveryReady: evidence.Ready && backupStatus == "verified" && restoreStatus == "verified",
		Evidence:      evidence.Evidence,
		Remediation:   evidence.Remediation,
		UpdatedAt:     evidence.UpdatedAt,
	}
}

// registerRecoveryTargets asks Data Backup Manager to converge the targets it
// can derive from Secrets Manager's service declaration. Registration is
// idempotent and remains owned by Data Backup Manager; this call only starts
// that owner-level convergence before recovery evidence is evaluated.
func registerRecoveryTargets(ctx context.Context) error {
	baseURL, err := recoveryManagerURL()
	if err != nil {
		return err
	}
	client := safetyconnect.NewSafetyServiceClient(&http.Client{Timeout: 5 * time.Second}, baseURL)
	response, err := client.RegisterScenarioTargets(ctx, connect.NewRequest(&safetyv1.RegisterScenarioTargetsRequest{Scenario: "secrets-manager"}))
	if err != nil {
		return fmt.Errorf("register secrets-manager recovery targets: %w", err)
	}
	if response == nil || response.Msg == nil {
		return fmt.Errorf("data-backup-manager returned no recovery target registration")
	}
	return nil
}

type fetchedRecoveryEvidence struct {
	Ready       bool
	Evidence    []recoveryEvidenceReference
	Remediation string
	UpdatedAt   time.Time
}

func fetchRecoveryEvidence(ctx context.Context) (fetchedRecoveryEvidence, error) {
	baseURL, err := recoveryManagerURL()
	if err != nil {
		return fetchedRecoveryEvidence{}, err
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	coverageClient := coverageconnect.NewCoverageServiceClient(httpClient, baseURL)
	coverageResponse, err := coverageClient.GetCoverageReport(ctx, connect.NewRequest(&coveragev1.GetCoverageReportRequest{}))
	if err != nil {
		return fetchedRecoveryEvidence{}, fmt.Errorf("read data-backup-manager coverage: %w", err)
	}
	if coverageResponse.Msg.GetReport() == nil || coverageResponse.Msg.GetReport().GetSummary() == nil {
		return fetchedRecoveryEvidence{}, fmt.Errorf("data-backup-manager returned no coverage summary")
	}
	summary := coverageResponse.Msg.GetReport().GetSummary()
	coverageVerified := summary.GetRegisteredCount() > 0 &&
		summary.GetRegisteredCount() == summary.GetPlannedCount() &&
		summary.GetRegisteredCount() == summary.GetBackedUpCount() &&
		summary.GetRegisteredCount() == summary.GetVerifiedCount() &&
		summary.GetRecommendedCount() == 0

	now := time.Now().UTC()
	evidence := fetchedRecoveryEvidence{
		Ready:     coverageVerified,
		UpdatedAt: now,
		Evidence: []recoveryEvidenceReference{{
			Kind:             "durable-backup-coverage",
			ArtifactIdentity: "data-backup-manager/coverage",
			ObservedAt:       now,
			Verified:         coverageVerified,
		}},
	}

	drillClient := drillsconnect.NewRecoveryDrillsServiceClient(httpClient, baseURL)
	drillsResponse, err := drillClient.ListDrills(ctx, connect.NewRequest(&drillsv1.ListDrillsRequest{PageSize: 50}))
	if err != nil {
		return fetchedRecoveryEvidence{}, fmt.Errorf("read data-backup-manager recovery drills: %w", err)
	}
	drills := append([]*drillsv1.RecoveryDrill(nil), drillsResponse.Msg.GetDrills()...)
	sort.SliceStable(drills, func(i, j int) bool { return recoveryDrillTime(drills[i]).After(recoveryDrillTime(drills[j])) })
	if len(drills) == 0 {
		evidence.Ready = false
		evidence.Remediation = "run a recovery drill; a snapshot alone is not recovery evidence"
		return evidence, nil
	}

	latest := drills[0]
	drillEvidence := recoveryEvidenceReference{
		Kind:             "recovery-drill",
		ArtifactIdentity: "data-backup-manager/drill/" + latest.GetId(),
		SourceGeneration: latest.GetSnapshotId(),
		ObservedAt:       recoveryDrillTime(latest),
		Verified:         latest.GetStatus() == drillsv1.DrillStatus_DRILL_STATUS_VERIFIED,
		Remediation:      latest.GetStatus().String(),
	}
	if drillEvidence.Verified && latest.GetRestoreId() != "" {
		restoreClient := restoresconnect.NewRestoresServiceClient(httpClient, baseURL)
		restoreResponse, restoreErr := restoreClient.GetRestore(ctx, connect.NewRequest(&restoresv1.GetRestoreRequest{Id: latest.GetRestoreId()}))
		if restoreErr != nil {
			return fetchedRecoveryEvidence{}, fmt.Errorf("read verified restore for drill %q: %w", latest.GetId(), restoreErr)
		}
		if restore := restoreResponse.Msg.GetRestore(); restore != nil {
			drillEvidence.Checksum = restore.GetChecksum()
			if restore.GetFinishedAt() != nil {
				drillEvidence.ObservedAt = restore.GetFinishedAt().AsTime().UTC()
			}
		}
	}
	evidence.Evidence = append(evidence.Evidence, drillEvidence)
	evidence.Ready = evidence.Ready && drillEvidence.Verified && drillEvidence.Checksum != ""
	if !evidence.Ready {
		evidence.Remediation = "complete a successful backup and verified recovery drill for every registered source"
	}
	return evidence, nil
}

func recoveryManagerURL() (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("SECRETS_MANAGER_DATA_BACKUP_MANAGER_URL")), "/")
	if baseURL != "" {
		return baseURL, nil
	}
	port := strings.TrimSpace(cliutil.DetectPortFromVrooli("data-backup-manager", "API_PORT")())
	if port == "" {
		return "", fmt.Errorf("data-backup-manager API port is unavailable")
	}
	return "http://127.0.0.1:" + port, nil
}

func recoveryDrillTime(drill *drillsv1.RecoveryDrill) time.Time {
	if drill == nil {
		return time.Time{}
	}
	for _, candidate := range []*timestamppb.Timestamp{drill.GetFinishedAt(), drill.GetStartedAt(), drill.GetRequestedAt()} {
		if candidate != nil {
			return candidate.AsTime().UTC()
		}
	}
	return time.Time{}
}
