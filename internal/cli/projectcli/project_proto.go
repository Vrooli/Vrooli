package projectcli

import (
	"encoding/json"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/templatevalidation"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ---------------------------------------------------------------------------
// shared field mappers
// ---------------------------------------------------------------------------

func projectFormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

func projectFormatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return projectFormatTime(*t)
}

// projectBoolPtrValue maps a *bool tri-state onto a *structpb.Value: nil ->
// null, else a bool value.
func projectBoolPtrValue(b *bool) *structpb.Value {
	if b == nil {
		return nil
	}
	return structpb.NewBoolValue(*b)
}

// projectIntPtrValue maps a *int tri-state onto a *structpb.Value: nil -> null,
// else a number value.
func projectIntPtrValue(i *int) *structpb.Value {
	if i == nil {
		return nil
	}
	return structpb.NewNumberValue(float64(*i))
}

// projectAnyValue maps an arbitrary interface{} payload onto a *structpb.Value
// (errors -> nil, i.e. JSON null).
func projectAnyValue(v any) *structpb.Value {
	if v == nil {
		return nil
	}
	val, err := cliout.NewJSONValue(v)
	if err != nil {
		return nil
	}
	return val
}

// projectRawValue maps a json.RawMessage-style payload onto a *structpb.Value
// by decoding the raw JSON; nil/empty -> null.
func projectRawValue(raw []byte) *structpb.Value {
	if len(raw) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	return projectAnyValue(decoded)
}

func projectResource(item resources.Resource) *cliv1.Resource {
	return &cliv1.Resource{
		Name:       item.Name,
		Path:       item.Path,
		Exists:     item.Exists,
		Registered: item.Registered,
		Enabled:    item.Enabled,
		Required:   item.Required,
		HasCli:     item.HasCLI,
		Config: &cliv1.ResourceConfig{
			Enabled:     item.Config.Enabled,
			Required:    item.Config.Required,
			Description: item.Config.Description,
		},
		ControlMode:  item.ControlMode,
		Driver:       item.Driver,
		Template:     item.Template,
		ManifestPath: item.ManifestPath,
	}
}

func projectSystemProcess(p maintenance.SystemProcess) *cliv1.ProjectSystemProcess {
	return &cliv1.ProjectSystemProcess{
		Pid:     int32(p.PID),
		Ppid:    int32(p.PPID),
		Command: p.Command,
	}
}

func projectStopReport(report control.StopReport) *cliv1.ProjectStopReport {
	out := &cliv1.ProjectStopReport{Message: report.Message}
	for _, item := range report.Stopped {
		out.Stopped = append(out.Stopped, projectStopItem(item))
	}
	for _, item := range report.Failed {
		out.Failed = append(out.Failed, projectStopItem(item))
	}
	return out
}

func projectStopItem(item control.ResultItem) *cliv1.ProjectStopItem {
	return &cliv1.ProjectStopItem{
		Name:    item.Name,
		Message: item.Message,
		Error:   item.Error,
	}
}

func projectRuntimeClaim(claim maintenance.RuntimeClaimInfo) *cliv1.ProjectRuntimeClaim {
	return &cliv1.ProjectRuntimeClaim{
		ClaimId:                     claim.ClaimID,
		InstanceId:                  claim.InstanceID,
		Scenario:                    claim.Scenario,
		Generation:                  claim.Generation,
		PortName:                    claim.PortName,
		EnvVar:                      claim.EnvVar,
		Port:                        int32(claim.Port),
		BindHost:                    claim.BindHost,
		Url:                         claim.URL,
		ClaimStatus:                 claim.ClaimStatus,
		InstanceStatus:              claim.InstanceStatus,
		SupervisorId:                claim.SupervisorID,
		SupervisorStatus:            claim.SupervisorStatus,
		SupervisorFresh:             projectBoolPtrValue(claim.SupervisorFresh),
		LeaseFresh:                  projectBoolPtrValue(claim.LeaseFresh),
		HeartbeatDeadline:           projectFormatTimePtr(claim.HeartbeatDeadline),
		SupervisorHeartbeatDeadline: projectFormatTimePtr(claim.SupervisorDeadline),
		HealthStatus:                claim.HealthStatus,
		HealthReady:                 projectBoolPtrValue(claim.HealthReady),
		Reconciliation:              string(claim.Reconciliation),
		ReconcileReason:             claim.ReconcileReason,
		Authoritative:               projectBoolPtrValue(claim.Authoritative),
		CreatedAt:                   projectFormatTime(claim.CreatedAt),
		UpdatedAt:                   projectFormatTime(claim.UpdatedAt),
		ExpiresAt:                   projectFormatTimePtr(claim.ExpiresAt),
		LastBoundAt:                 projectFormatTimePtr(claim.LastBoundAt),
		LastListenerCheckAt:         projectFormatTimePtr(claim.LastListenerCheckAt),
		LastListenerSeenAt:          projectFormatTimePtr(claim.LastListenerSeenAt),
		FirstUnboundAt:              projectFormatTimePtr(claim.FirstUnboundAt),
		ConsecutiveListenerMisses:   int32(claim.ConsecutiveListenerMisses),
		ListenerStatus:              claim.ListenerStatus,
		ListenerPid:                 projectIntPtrValue(claim.ListenerPID),
		ListenerProcessLabel:        claim.ListenerProcessLabel,
		RecommendationCode:          claim.RecommendationCode,
		RecommendationConfidence:    claim.RecommendationConfidence,
		RecommendationRationale:     claim.RecommendationRationale,
	}
}

// ---------------------------------------------------------------------------
// `vrooli status --json`
// ---------------------------------------------------------------------------

// ProjectStatusResponseMessage maps the internal status report onto the
// vrooli.cli.v1 wire contract. A proto field rename breaks this at compile time.
func ProjectStatusResponseMessage(report project.StatusReport) *cliv1.ProjectStatusResponse {
	out := &cliv1.ProjectStatusReport{Summary: map[string]int32{}}
	for _, item := range report.Resources {
		out.Resources = append(out.Resources, projectResourceStatus(item))
	}
	for _, item := range report.Scenarios {
		out.Scenarios = append(out.Scenarios, projectScenarioStatus(item))
	}
	if report.Maintenance != nil {
		out.Maintenance = projectProcessSnapshot(*report.Maintenance)
	}
	for k, v := range report.Summary {
		out.Summary[k] = int32(v)
	}
	return &cliv1.ProjectStatusResponse{Success: true, Status: out}
}

func projectResourceStatus(s resources.Status) *cliv1.ProjectResourceStatus {
	return &cliv1.ProjectResourceStatus{
		Resource:   projectResource(s.Resource),
		Installed:  s.Installed,
		Running:    s.Running,
		Healthy:    projectBoolPtrValue(s.Healthy),
		Health:     s.Health,
		StatusCode: s.StatusCode,
		Message:    s.Message,
		ProbeError: s.ProbeError,
		Raw:        projectRawValue(s.Raw),
	}
}

func projectScenarioStatus(v orchestrator.ScenarioView) *cliv1.ProjectScenarioStatus {
	ports := map[string]int32{}
	for k, p := range v.Ports {
		ports[k] = int32(p)
	}
	return &cliv1.ProjectScenarioStatus{
		Name:         v.Name,
		DisplayName:  v.DisplayName,
		Description:  v.Description,
		Tags:         append([]string(nil), v.Tags...),
		Status:       v.Status,
		Processes:    int32(v.Processes),
		StartedAt:    projectFormatTimePtr(v.StartedAt),
		Runtime:      v.Runtime,
		Ports:        ports,
		HealthStatus: projectAnyValue(v.Health),
	}
}

func projectProcessSnapshot(s maintenance.ProcessSnapshot) *cliv1.ProjectProcessSnapshot {
	out := &cliv1.ProjectProcessSnapshot{
		TrackedProcesses: int32(s.TrackedProcesses),
		RunningTracked:   int32(s.RunningTracked),
		ChildProcesses:   int32(s.ChildProcesses),
		TotalProcesses:   int32(s.TotalProcesses),
		ZombieProcesses:  int32(s.ZombieProcesses),
		OrphanProcesses:  int32(s.OrphanProcesses),
	}
	for _, p := range s.Orphans {
		out.Orphans = append(out.Orphans, projectSystemProcess(p))
	}
	return out
}

// ---------------------------------------------------------------------------
// `vrooli doctor --json`
// ---------------------------------------------------------------------------

// ProjectDoctorResponseMessage maps the doctor checks onto the wire contract.
func ProjectDoctorResponseMessage(checks []project.DoctorCheck) *cliv1.ProjectDoctorResponse {
	out := &cliv1.ProjectDoctorResponse{Success: true}
	for _, c := range checks {
		out.Checks = append(out.Checks, &cliv1.ProjectDoctorCheck{
			Name:    c.Name,
			Status:  c.Status,
			Message: c.Message,
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// stop / orphans-kill / locks-clean reports — WriteSuccessJSON(w, "data", report)
// ---------------------------------------------------------------------------

// ProjectStopResponseMessage maps a control.StopReport onto the "data" envelope.
func ProjectStopResponseMessage(report control.StopReport) *cliv1.ProjectStopResponse {
	return &cliv1.ProjectStopResponse{Success: true, Data: projectStopReport(report)}
}

// ---------------------------------------------------------------------------
// `vrooli orphans --json` (list mode)
// ---------------------------------------------------------------------------

// ProjectOrphansResponseMessage maps the orphan list onto the "orphans" envelope.
func ProjectOrphansResponseMessage(list []maintenance.SystemProcess) *cliv1.ProjectOrphansResponse {
	out := &cliv1.ProjectOrphansResponse{Success: true}
	for _, p := range list {
		out.Orphans = append(out.Orphans, projectSystemProcess(p))
	}
	return out
}

// ProjectOrphansDryRunResponseMessage maps the dry-run orphan list onto the
// nested "dry_run.orphans" envelope.
func ProjectOrphansDryRunResponseMessage(list []maintenance.SystemProcess) *cliv1.ProjectOrphansDryRunResponse {
	dry := &cliv1.ProjectOrphansDryRun{}
	for _, p := range list {
		dry.Orphans = append(dry.Orphans, projectSystemProcess(p))
	}
	return &cliv1.ProjectOrphansDryRunResponse{Success: true, DryRun: dry}
}

// ---------------------------------------------------------------------------
// `vrooli locks --json` (list mode) — WriteSuccessFields(registry_claims)
// ---------------------------------------------------------------------------

// ProjectLocksResponseMessage maps the registry-claim list onto the
// success-fields envelope.
func ProjectLocksResponseMessage(claims []maintenance.RuntimeClaimInfo) *cliv1.ProjectLocksResponse {
	out := &cliv1.ProjectLocksResponse{Success: true}
	for _, claim := range claims {
		out.RegistryClaims = append(out.RegistryClaims, projectRuntimeClaim(claim))
	}
	return out
}

// ---------------------------------------------------------------------------
// `vrooli template-validation cleanup --json` — WriteFieldsWithSuccess(success, {cleanup})
// NOTE: camelCase keys (json_name) — uses projectCamelJSONOptions.
// ---------------------------------------------------------------------------

// ProjectTemplateCleanupResponseMessage maps the cleanup result onto the
// "cleanup" envelope. success is len(Failures)==0, passed through by the caller.
func ProjectTemplateCleanupResponseMessage(result templatevalidation.CleanupResult) *cliv1.ProjectTemplateCleanupResponse {
	cleanup := &cliv1.ProjectTemplateCleanupResult{
		DryRun:             result.DryRun,
		OlderThan:          int64(result.OlderThan),
		IncludeRetained:    result.IncludeRetained,
		RunId:              result.RunID,
		NeedsProtoGenerate: result.NeedsProtoGenerate,
		ProtoGenerateRan:   result.ProtoGenerateRan,
		Message:            result.Message,
	}
	for _, run := range result.Eligible {
		cleanup.Eligible = append(cleanup.Eligible, projectTemplateRun(run))
	}
	for _, run := range result.Skipped {
		cleanup.Skipped = append(cleanup.Skipped, projectTemplateSkippedRun(run))
	}
	for _, run := range result.Failures {
		cleanup.Failures = append(cleanup.Failures, projectTemplateFailedRun(run))
	}
	for _, run := range result.Removed {
		cleanup.Removed = append(cleanup.Removed, projectTemplateRun(run))
	}
	return &cliv1.ProjectTemplateCleanupResponse{
		Success: len(result.Failures) == 0,
		Cleanup: cleanup,
	}
}

func projectTemplateRun(run templatevalidation.Run) *cliv1.ProjectTemplateRun {
	return &cliv1.ProjectTemplateRun{
		MarkerPath: run.MarkerPath,
		Marker:     projectTemplateRunMarker(run.Marker),
		Age:        run.Age,
	}
}

func projectTemplateRunMarker(m templatevalidation.RunMarker) *cliv1.ProjectTemplateRunMarker {
	return &cliv1.ProjectTemplateRunMarker{
		Version:             m.Version,
		RunId:               m.RunID,
		RepoRoot:            m.RepoRoot,
		Template:            m.Template,
		ScenarioId:          m.ScenarioID,
		ScenarioPath:        m.ScenarioPath,
		TempRoot:            m.TempRoot,
		CreatedAt:           projectFormatTime(m.CreatedAt),
		Retained:            m.Retained,
		CreatorPid:          int32(m.CreatorPID),
		Completed:           m.Completed,
		CleanupStatus:       m.CleanupStatus,
		RelocationArtifacts: append([]string(nil), m.RelocationArtifacts...),
	}
}

func projectTemplateSkippedRun(run templatevalidation.SkippedRun) *cliv1.ProjectTemplateSkippedRun {
	out := &cliv1.ProjectTemplateSkippedRun{
		Path:   run.Path,
		Reason: run.Reason,
	}
	if run.Run != nil {
		out.Run = projectTemplateRun(*run.Run)
	}
	return out
}

func projectTemplateFailedRun(run templatevalidation.FailedRun) *cliv1.ProjectTemplateFailedRun {
	out := &cliv1.ProjectTemplateFailedRun{
		Path:  run.Path,
		Error: run.Error,
	}
	if run.Run != nil {
		out.Run = projectTemplateRun(*run.Run)
	}
	return out
}

// ---------------------------------------------------------------------------
// `vrooli diagnose-port --json` — WriteSuccessJSON(w, "diagnostic", diagnostic)
// ---------------------------------------------------------------------------

// ProjectPortDiagnosticResponseMessage maps a maintenance.PortDiagnostic onto
// the "diagnostic" envelope.
func ProjectPortDiagnosticResponseMessage(d maintenance.PortDiagnostic) *cliv1.ProjectPortDiagnosticResponse {
	out := &cliv1.ProjectPortDiagnostic{
		Port:     int32(d.Port),
		Scenario: d.Scenario,
		InUse:    d.InUse,
		ListenerInspection: &cliv1.ProjectListenerInspection{
			Available: d.ListenerInspection.Available,
			Tool:      d.ListenerInspection.Tool,
			Reason:    d.ListenerInspection.Reason,
		},
		HostOrphanCount: int32(d.HostOrphanCount),
		Recommendations: append([]string(nil), d.Recommendations...),
		PortPolicy: &cliv1.ProjectPortPolicy{
			EphemeralMin:         int32(d.PortPolicy.EphemeralMin),
			EphemeralMax:         int32(d.PortPolicy.EphemeralMax),
			EphemeralSource:      d.PortPolicy.EphemeralSource,
			InsideEphemeralRange: d.PortPolicy.InsideEphemeralRange,
			CanonicalBand:        d.PortPolicy.CanonicalBand,
			AboveCanonicalMax:    d.PortPolicy.AboveCanonicalMax,
		},
	}
	for _, l := range d.Listeners {
		out.Listeners = append(out.Listeners, &cliv1.ProjectPortListener{
			Pid:     int32(l.PID),
			Command: l.Command,
			Zombie:  l.Zombie,
		})
	}
	for _, claim := range d.RegistryClaims {
		out.RegistryClaims = append(out.RegistryClaims, projectRuntimeClaim(claim))
	}
	for _, ref := range d.RegistryProcesses {
		out.RegistryProcesses = append(out.RegistryProcesses, projectRuntimeProcessRef(ref))
	}
	return &cliv1.ProjectPortDiagnosticResponse{Success: true, Diagnostic: out}
}

func projectRuntimeProcessRef(ref maintenance.RuntimeProcessRefInfo) *cliv1.ProjectRuntimeProcessRef {
	return &cliv1.ProjectRuntimeProcessRef{
		RefId:          ref.RefID,
		InstanceId:     ref.InstanceID,
		Scenario:       ref.Scenario,
		InstanceStatus: ref.InstanceStatus,
		Pid:            projectIntPtrValue(ref.PID),
		Pgid:           projectIntPtrValue(ref.PGID),
		ProcessId:      ref.ProcessID,
		Step:           ref.Step,
		Command:        ref.Command,
		Status:         ref.Status,
		PidRunning:     projectBoolPtrValue(ref.PIDRunning),
	}
}
