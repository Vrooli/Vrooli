package vroolicli

import (
	"io"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// cliVersionMessage maps the inline versionOutput struct onto the vrooli.cli.v1
// wire contract. A proto field rename breaks this mapping at compile time.
func cliVersionMessage(out versionOutput) *cliv1.CliVersion {
	return &cliv1.CliVersion{
		CliVersion:      out.CLIVersion,
		PlatformVersion: out.PlatformVersion,
		Root:            out.Root,
	}
}

// cliSupervisorStatusMessage maps runtimesupervisor.StatusReport onto the
// vrooli.cli.v1 wire contract.
func cliSupervisorStatusMessage(report runtimesupervisor.StatusReport) *cliv1.CliSupervisorStatus {
	var pid int32
	if report.PID != nil {
		pid = int32(*report.PID)
	}
	return &cliv1.CliSupervisorStatus{
		SupervisorId:                  report.SupervisorID,
		Status:                        report.Status,
		StatusReason:                  report.StatusReason,
		HostBootId:                    report.HostBootID,
		HostSessionId:                 report.HostSessionID,
		Pid:                           pid,
		LastHeartbeatAt:               formatRFC3339Nano(report.LastHeartbeatAt),
		HeartbeatDeadlineAt:           formatRFC3339Nano(report.HeartbeatDeadlineAt),
		SupervisedInstanceCount:       int32(report.SupervisedInstanceCount),
		UnverifiedInstanceCount:       int32(report.UnverifiedInstanceCount),
		EffectiveRenewInterval:        int64(report.EffectiveRenewInterval),
		EffectiveLeaseTtl:             int64(report.EffectiveLeaseTTL),
		EffectiveHealthInterval:       int64(report.EffectiveHealthInterval),
		EffectiveMaxHealthConcurrency: int32(report.EffectiveMaxHealthConcurrency),
		EffectiveBatchSize:            int32(report.EffectiveBatchSize),
		LastTick: &cliv1.CliSupervisorTick{
			SupervisorId:     report.LastTick.SupervisorID,
			Renewed:          int32(report.LastTick.Renewed),
			Expired:          int32(report.LastTick.Expired),
			Unverified:       int32(report.LastTick.Unverified),
			HealthProbeCount: int32(report.LastTick.HealthProbeCount),
		},
	}
}

// cliSupervisorServiceResultMessage maps runtimesupervisor.ServiceInstallResult
// (shared by install + uninstall) onto the vrooli.cli.v1 wire contract.
func cliSupervisorServiceResultMessage(result runtimesupervisor.ServiceInstallResult) *cliv1.CliSupervisorServiceResult {
	return &cliv1.CliSupervisorServiceResult{
		UnitName: result.UnitName,
		UnitPath: result.UnitPath,
		Scope:    result.Scope,
		Active:   result.Active,
	}
}

// formatRFC3339Nano renders a time.Time as RFC3339Nano, mapping the zero time
// to "" so the wire contract distinguishes "unset" from a real timestamp.
func formatRFC3339Nano(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

func writeCliVersionJSON(w io.Writer, out versionOutput) error {
	return marshalCliRuntime(w, cliVersionMessage(out))
}

func writeCliSupervisorStatusJSON(w io.Writer, report runtimesupervisor.StatusReport) error {
	return marshalCliRuntime(w, cliSupervisorStatusMessage(report))
}

func writeCliSupervisorServiceResultJSON(w io.Writer, result runtimesupervisor.ServiceInstallResult) error {
	return marshalCliRuntime(w, cliSupervisorServiceResultMessage(result))
}

func marshalCliRuntime(w io.Writer, msg proto.Message) error {
	return cliout.WriteProtoJSON(w, msg)
}
