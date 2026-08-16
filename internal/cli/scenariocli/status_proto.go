package scenariocli

import (
	"io"
	"time"

	"github.com/vrooli/vrooli/internal/lifecycle"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/resources"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// formatTimePtr maps a *time.Time to RFC3339Nano, returning "" for nil.
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

// formatTime maps a time.Time to RFC3339Nano, returning "" for the zero value.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

// copyInt32Map maps a map[string]int onto the proto map<string,int32> shape.
func copyInt32Map(src map[string]int) map[string]int32 {
	if len(src) == 0 {
		return map[string]int32{}
	}
	dup := make(map[string]int32, len(src))
	for k, v := range src {
		dup[k] = int32(v)
	}
	return dup
}

// scenarioPortMessages maps internal ListPortOutput slices onto the shared
// ScenarioPort message (defined in cli/v1/scenario_list.proto).
func scenarioPortMessages(ports []ListPortOutput) []*cliv1.ScenarioPort {
	out := make([]*cliv1.ScenarioPort, 0, len(ports))
	for _, port := range ports {
		out = append(out, &cliv1.ScenarioPort{
			Key:            port.Key,
			Step:           port.Step,
			Port:           int32(port.Port),
			ListenerStatus: port.ListenerStatus,
		})
	}
	return out
}

// scenarioStatusItem maps a StatusItemOutput onto its proto message. Health is
// an arbitrary JSON value; a structpb conversion error degrades to nil (null),
// preserving the additive contract rather than failing the whole render.
func scenarioStatusItem(item StatusItemOutput) *cliv1.ScenarioStatusItem {
	var health *structpb.Value
	if v, err := structpb.NewValue(item.Health); err == nil {
		health = v
	}
	return &cliv1.ScenarioStatusItem{
		Name:           item.Name,
		DisplayName:    item.DisplayName,
		Description:    item.Description,
		Tags:           item.Tags,
		Status:         item.Status,
		Processes:      int32(item.Processes),
		Runtime:        item.Runtime,
		StartedAt:      formatTimePtr(item.StartedAt),
		Ports:          copyInt32Map(item.Ports),
		PortBindings:   scenarioPortMessages(item.PortBindings),
		HealthStatus:   health,
		HealthError:    item.HealthError,
		StartOperation: ScenarioStartOperationMessage(item.StartOperation),
	}
}

// ScenarioStartOperationMessage maps a start-operation view onto its proto
// message; nil maps to nil (absent). Exported for the wait/lifecycle
// response builders.
func ScenarioStartOperationMessage(view *lifecycle.StartOperationView) *cliv1.ScenarioStartOperation {
	if view == nil {
		return nil
	}
	steps := make([]*cliv1.ScenarioStartOperationStep, 0, len(view.Steps))
	for _, step := range view.Steps {
		ended := ""
		if step.EndedAt != nil {
			ended = step.EndedAt.Format(time.RFC3339Nano)
		}
		steps = append(steps, &cliv1.ScenarioStartOperationStep{
			Name:      step.Name,
			Status:    step.Status,
			StartedAt: step.StartedAt.Format(time.RFC3339Nano),
			EndedAt:   ended,
		})
	}
	finished := ""
	if view.FinishedAt != nil {
		finished = view.FinishedAt.Format(time.RFC3339Nano)
	}
	return &cliv1.ScenarioStartOperation{
		OperationId:                 view.OperationID,
		Scenario:                    view.Scenario,
		Variant:                     view.Variant,
		Operation:                   view.Operation,
		Status:                      view.Status,
		Verdict:                     view.Verdict,
		Error:                       view.Error,
		CurrentStep:                 view.CurrentStep,
		DependencyCurrent:           view.DependencyCurrent,
		DependencyIndex:             int32(view.DependencyIndex),
		DependencyTotal:             int32(view.DependencyTotal),
		StartedAt:                   view.StartedAt.Format(time.RFC3339Nano),
		FinishedAt:                  finished,
		ElapsedSeconds:              int32(view.ElapsedSeconds),
		Steps:                       steps,
		EtaKnown:                    view.ETAKnown,
		EtaSeconds:                  int32(view.ETASeconds),
		RecommendedNextCheckSeconds: int32(view.RecommendedNextCheckSeconds),
		InitiatorPid:                int32(view.InitiatorPID),
	}
}

// scenarioPortSummary maps a scenariomodel.PortSummary onto its proto message.
// FixedPort is a *int that is nil when a range is used; nil maps to 0.
func scenarioPortSummary(p scenariomodel.PortSummary) *cliv1.ScenarioInfoPortSummary {
	fixed := int32(0)
	if p.FixedPort != nil {
		fixed = int32(*p.FixedPort)
	}
	return &cliv1.ScenarioInfoPortSummary{
		Name:        p.Name,
		EnvVar:      p.EnvVar,
		Description: p.Description,
		Range:       p.Range,
		FixedPort:   fixed,
	}
}

// scenarioGenerationMetadata maps an optional *GenerationMetadata onto its
// proto message; nil maps to nil (absent / null).
func scenarioGenerationMetadata(g *scenariomodel.GenerationMetadata) *cliv1.ScenarioGenerationMetadata {
	if g == nil {
		return nil
	}
	return &cliv1.ScenarioGenerationMetadata{
		Template: &cliv1.ScenarioGenerationTemplate{
			Id:      g.Template.ID,
			Version: g.Template.Version,
		},
		GeneratedAt: g.GeneratedAt,
		Design: &cliv1.ScenarioGenerationDesign{
			Id:      g.Design.ID,
			Version: g.Design.Version,
			Adapter: g.Design.Adapter,
		},
		ManifestSha: g.ManifestSha,
		ContentSha:  g.ContentSha,
	}
}

// scenarioInfoData maps an InfoScenarioData onto its proto message.
func scenarioInfoData(info InfoScenarioData) *cliv1.ScenarioInfoData {
	msg := &cliv1.ScenarioInfoData{
		Name:              info.Name,
		DisplayName:       info.DisplayName,
		Description:       info.Description,
		Version:           info.Version,
		Type:              info.Type,
		Category:          info.Category,
		Tags:              info.Tags,
		Path:              info.Path,
		ServicePath:       info.ServicePath,
		SandboxRedirected: info.SandboxRedirect,
		ConfigVersion:     info.ConfigVersion,
		LifecycleVersion:  info.LifecycleVersion,
		Generation:        scenarioGenerationMetadata(info.Generation),
	}
	for _, p := range info.Ports {
		msg.Ports = append(msg.Ports, scenarioPortSummary(p))
	}
	for _, ph := range info.Phases {
		msg.Phases = append(msg.Phases, &cliv1.ScenarioInfoPhaseSummary{
			Name:        ph.Name,
			Description: ph.Description,
			Steps:       int32(ph.Steps),
			Defined:     ph.Defined,
		})
	}
	return msg
}

// scenarioProcessRecord maps a process.Record onto its proto message.
func scenarioProcessRecord(r process.Record) *cliv1.ScenarioProcessRecord {
	return &cliv1.ScenarioProcessRecord{
		Pid:        int32(r.PID),
		Pgid:       int32(r.PGID),
		ProcessId:  r.ProcessID,
		Phase:      r.Phase,
		Scenario:   r.Scenario,
		Step:       r.Step,
		Command:    r.Command,
		WorkingDir: r.WorkingDir,
		LogFile:    r.LogFile,
		Port:       int32(r.Port),
		StartedAt:  formatTime(r.StartedAt),
		Status:     r.Status,
	}
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// ScenarioEnvValidationResponse maps a ScenarioEnvValidationReport onto the
// wire contract (cliout.WriteFieldsWithSuccess; success mirrors report.Passed).
func ScenarioEnvValidationResponse(report resources.ScenarioEnvValidationReport) *cliv1.ScenarioEnvValidationResponse {
	reportMsg := &cliv1.ScenarioEnvValidationReport{
		Scenario: report.Scenario,
		Values:   copyStringMap(report.Values),
		Passed:   report.Passed,
	}
	for _, issue := range report.Issues {
		reportMsg.Issues = append(reportMsg.Issues, &cliv1.ScenarioValidationIssue{
			Severity: issue.Severity,
			Message:  issue.Message,
		})
	}
	for _, rr := range report.ResourceReports {
		reportMsg.ResourceReports = append(reportMsg.ResourceReports, &cliv1.ScenarioResourceReport{
			Name:         rr.Name,
			ManifestPath: rr.Manifest,
			Values:       copyStringMap(rr.Values),
			Warnings:     rr.Warnings,
		})
	}
	return &cliv1.ScenarioEnvValidationResponse{
		Success: report.Passed,
		Report:  reportMsg,
	}
}

func writeScenarioEnvValidationJSON(w io.Writer, report resources.ScenarioEnvValidationReport) error {
	return marshalScenarioStatus(w, ScenarioEnvValidationResponse(report))
}

// scenarioRuntimeData maps an InfoRuntimeData onto its proto message.
func scenarioRuntimeData(rt InfoRuntimeData) *cliv1.ScenarioRuntimeData {
	msg := &cliv1.ScenarioRuntimeData{
		Status:      rt.Status,
		Processes:   int32(rt.Processes),
		Runtime:     rt.Runtime,
		StartedAt:   formatTimePtr(rt.StartedAt),
		Ports:       copyInt32Map(rt.Ports),
		ListPorts:   scenarioPortMessages(rt.ListPorts),
		HealthError: rt.HealthError,
	}
	for _, r := range rt.ProcessInfo {
		msg.ProcessRecords = append(msg.ProcessRecords, scenarioProcessRecord(r))
	}
	return msg
}

// -----------------------------------------------------------------------------
// `scenario status` (list form)
// -----------------------------------------------------------------------------

// ScenarioStatusListResponse maps the status list payload onto its wire
// contract (cliout.WriteSuccessFields envelope).
func ScenarioStatusListResponse(items []StatusItemOutput, failures []discovery.Failure) *cliv1.ScenarioStatusListResponse {
	running := 0
	for _, item := range items {
		if item.Status == "running" {
			running++
		}
	}
	resp := &cliv1.ScenarioStatusListResponse{
		Success: true,
		Summary: &cliv1.ScenarioStatusSummary{
			TotalScenarios: int32(len(items)),
			Running:        int32(running),
			Stopped:        int32(len(items) - running),
		},
	}
	for _, item := range items {
		resp.Scenarios = append(resp.Scenarios, scenarioStatusItem(item))
	}
	for _, failure := range failures {
		resp.DiscoveryFailures = append(resp.DiscoveryFailures, &cliv1.DiscoveryFailure{
			Kind:  failure.Kind,
			Name:  failure.Name,
			Path:  failure.Path,
			Stage: failure.Stage,
			Error: failure.Error,
		})
	}
	return resp
}

func writeScenarioStatusListJSON(w io.Writer, items []StatusItemOutput, failures []discovery.Failure) error {
	return marshalScenarioStatus(w, ScenarioStatusListResponse(items, failures))
}

// -----------------------------------------------------------------------------
// `scenario status <name>` (single form)
// -----------------------------------------------------------------------------

// ScenarioStatusSingleResponse maps the single-status payload onto its wire
// contract (bare cliout.WriteJSON payload).
func ScenarioStatusSingleResponse(out StatusSingleOutput) *cliv1.ScenarioStatusSingle {
	return &cliv1.ScenarioStatusSingle{
		Success:  out.Success,
		Scenario: scenarioStatusItem(out.Scenario),
		Info:     scenarioInfoData(out.Info),
		Runtime:  scenarioRuntimeData(out.Runtime),
	}
}

func writeScenarioStatusSingleJSON(w io.Writer, out StatusSingleOutput) error {
	return marshalScenarioStatus(w, ScenarioStatusSingleResponse(out))
}

// -----------------------------------------------------------------------------
// `scenario info`
// -----------------------------------------------------------------------------

// ScenarioInfoResponse maps the info payload onto its wire contract (bare
// cliout.WriteJSON payload).
func ScenarioInfoResponse(out InfoOutput) *cliv1.ScenarioInfoResponse {
	return &cliv1.ScenarioInfoResponse{
		Success:  out.Success,
		Scenario: scenarioInfoData(out.Scenario),
		Runtime:  scenarioRuntimeData(out.Runtime),
	}
}

func writeScenarioInfoJSON(w io.Writer, out InfoOutput) error {
	return marshalScenarioStatus(w, ScenarioInfoResponse(out))
}

// -----------------------------------------------------------------------------
// `scenario port` / `scenario ports`
// -----------------------------------------------------------------------------

// ScenarioPortSingleResponse maps a PortSingleOutput onto its wire contract.
func ScenarioPortSingleResponse(out PortSingleOutput) *cliv1.ScenarioPortSingle {
	return &cliv1.ScenarioPortSingle{
		Success:  out.Success,
		Scenario: out.Scenario,
		PortName: out.PortName,
		Step:     out.Step,
		Port:     int32(out.Port),
		Error:    out.Error,
	}
}

func writeScenarioPortSingleJSON(w io.Writer, out PortSingleOutput) error {
	return marshalScenarioStatus(w, ScenarioPortSingleResponse(out))
}

// ScenarioPortListResponse maps a PortListOutput onto its wire contract.
func ScenarioPortListResponse(out PortListOutput) *cliv1.ScenarioPortList {
	return &cliv1.ScenarioPortList{
		Success:  out.Success,
		Scenario: out.Scenario,
		Ports:    scenarioPortMessages(out.Ports),
		Metadata: copyInt32Map(out.Metadata),
		Error:    out.Error,
	}
}

func writeScenarioPortListJSON(w io.Writer, out PortListOutput) error {
	return marshalScenarioStatus(w, ScenarioPortListResponse(out))
}

// marshalScenarioStatus marshals a status-domain proto message and writes it
// (newline-terminated) to w.
func marshalScenarioStatus(w io.Writer, msg proto.Message) error {
	return cliout.WriteProtoJSON(w, msg)
}
