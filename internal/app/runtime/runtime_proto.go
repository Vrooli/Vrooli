package runtimeapp

import (
	"encoding/json"
	"io"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func supervisorStatusMessage(report runtimesupervisor.StatusReport) *cliv1.CliSupervisorStatus {
	var pid int32
	if report.PID != nil {
		pid = int32(*report.PID)
	}
	return &cliv1.CliSupervisorStatus{SupervisorId: report.SupervisorID, Status: report.Status, StatusReason: report.StatusReason, HostBootId: report.HostBootID, HostSessionId: report.HostSessionID, Pid: pid, LastHeartbeatAt: formatRFC3339Nano(report.LastHeartbeatAt), HeartbeatDeadlineAt: formatRFC3339Nano(report.HeartbeatDeadlineAt), SupervisedInstanceCount: int32(report.SupervisedInstanceCount), UnverifiedInstanceCount: int32(report.UnverifiedInstanceCount), EffectiveRenewInterval: int64(report.EffectiveRenewInterval), EffectiveLeaseTtl: int64(report.EffectiveLeaseTTL), EffectiveHealthInterval: int64(report.EffectiveHealthInterval), EffectiveMaxHealthConcurrency: int32(report.EffectiveMaxHealthConcurrency), EffectiveBatchSize: int32(report.EffectiveBatchSize), LastTick: &cliv1.CliSupervisorTick{SupervisorId: report.LastTick.SupervisorID, Renewed: int32(report.LastTick.Renewed), Expired: int32(report.LastTick.Expired), Unverified: int32(report.LastTick.Unverified), HealthProbeCount: int32(report.LastTick.HealthProbeCount)}}
}

func supervisorServiceResultMessage(result runtimesupervisor.ServiceInstallResult) *cliv1.CliSupervisorServiceResult {
	return &cliv1.CliSupervisorServiceResult{UnitName: result.UnitName, UnitPath: result.UnitPath, Scope: result.Scope, Active: result.Active}
}

func formatRFC3339Nano(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

func writeSupervisorStatusJSON(w io.Writer, report runtimesupervisor.StatusReport) error {
	return cliout.WriteProtoJSON(w, supervisorStatusMessage(report))
}

// WriteSupervisorStatusJSON exposes the typed supervisor status renderer to
// the compatibility test seam and any future runtime CLI adapter.
func WriteSupervisorStatusJSON(w io.Writer, report runtimesupervisor.StatusReport) error {
	return writeSupervisorStatusJSON(w, report)
}

func writeSupervisorServiceResultJSON(w io.Writer, result runtimesupervisor.ServiceInstallResult) error {
	return cliout.WriteProtoJSON(w, supervisorServiceResultMessage(result))
}

// WriteSupervisorServiceResultJSON exposes the shared install/uninstall shape.
func WriteSupervisorServiceResultJSON(w io.Writer, result runtimesupervisor.ServiceInstallResult) error {
	return writeSupervisorServiceResultJSON(w, result)
}

// writeRuntimeJSON bridges legacy address-shaped recovery objects through the
// protobuf JSON implementation until each recovery object has a stable public
// cliv1 schema. It retains keys and never handles secrets.
func writeRuntimeJSON(w io.Writer, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return err
	}
	payload, err := structpb.NewStruct(object)
	if err != nil {
		return err
	}
	return cliout.WriteProtoJSON(w, payload)
}
