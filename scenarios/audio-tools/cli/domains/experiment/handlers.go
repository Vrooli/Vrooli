package experiment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client experimentconnect.ExperimentServiceClient
}

var experimentProtoJSONOptions = protojson.MarshalOptions{
	UseProtoNames:     true,
	Multiline:         true,
	Indent:            "  ",
	EmitDefaultValues: true,
	EmitUnpopulated:   false,
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: experimentconnect.NewExperimentServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetExperiment(context.Background(), connect.NewRequest(&experimentv1.GetExperimentRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-get", err, nil)
	}
	return renderExperiment(ctx, resp.Msg, "get")
}

func (h *handlers) wait(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	// Experiments are server-owned and may intentionally run through the
	// 60-minute qualification rung. The scenario's normal 180-second HTTP
	// timeout must not turn `experiment wait` into a misleading client abort
	// while the server continues correctly in the background.
	//
	// All non-wait RPCs retain h.client's ordinary timeout; this is the one
	// command whose purpose is explicitly to block until a durable terminal
	// state exists.
	waitHTTPClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(h.core, 0)
	waitClient := experimentconnect.NewExperimentServiceClient(waitHTTPClient, baseURL)
	resp, err := waitClient.WaitExperiment(context.Background(), connect.NewRequest(&experimentv1.WaitExperimentRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-wait", err, nil)
	}
	// Build one stable {experiment, report, runs} envelope so `wait --json`
	// parses to a fixed shape regardless of whether the run produced a report.
	envelope := &experimentv1.GetExperimentReportResponse{
		Experiment: resp.Msg.GetExperiment(),
		Runs:       resp.Msg.GetRuns(),
	}
	reportReady := false
	if experimentTerminal(resp.Msg.GetExperiment()) && resp.Msg.GetExperiment().GetResultRef() != "" {
		reportResp, reportErr := h.client.GetExperimentReport(context.Background(), connect.NewRequest(&experimentv1.GetExperimentReportRequest{Id: id}))
		if reportErr == nil && reportResp != nil && reportResp.Msg != nil && reportResp.Msg.GetReport() != nil {
			envelope.Experiment = reportResp.Msg.GetExperiment()
			envelope.Report = reportResp.Msg.GetReport()
			envelope.Runs = reportResp.Msg.GetRuns()
			reportReady = true
		}
	}
	if ctx.JSON() {
		return printExperimentWaitJSON(ctx.Stdout(), envelope)
	}
	if reportReady {
		return renderReport(ctx, envelope)
	}
	if experimentTerminal(resp.Msg.GetExperiment()) && resp.Msg.GetExperiment().GetResultRef() != "" {
		fmt.Fprintf(ctx.Stderr(), "Report is not available yet; use `audio-tools experiment report %s` to re-attach.\n", id)
	}
	return renderExperiment(ctx, resp.Msg, "wait")
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	req := &experimentv1.ListExperimentsRequest{}
	if v := strings.TrimSpace(ctx.Flag("status")); v != "" {
		status, err := statusFromFlag(v)
		if err != nil {
			return err
		}
		req.Status = status
	}
	if v := strings.TrimSpace(ctx.Flag("limit")); v != "" {
		n, err := parseIntFlag("limit", v)
		if err != nil {
			return err
		}
		req.Limit = n
	}
	if v := strings.TrimSpace(ctx.Flag("offset")); v != "" {
		n, err := parseIntFlag("offset", v)
		if err != nil {
			return err
		}
		req.Offset = n
	}
	resp, err := h.client.ListExperiments(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("experiment-list", err, nil)
	}
	if ctx.JSON() {
		return printExperimentProtoJSON(ctx.Stdout(), resp.Msg)
	}
	experiments := resp.Msg.GetExperiments()
	if len(experiments) == 0 {
		fmt.Fprintln(ctx.Stdout(), "No experiments found.")
		return nil
	}
	fmt.Fprintf(ctx.Stdout(), "%-38s  %-10s  %-22s  %s\n", "ID", "STATUS", "CREATED", "NAME")
	for _, exp := range experiments {
		fmt.Fprintf(ctx.Stdout(), "%-38s  %-10s  %-22s  %s\n",
			exp.GetId(), statusLabel(exp.GetStatus()), timestampLabel(exp.GetCreatedAt()), exp.GetName())
	}
	printExperimentErrors(ctx, experiments)
	return nil
}

func (h *handlers) cancel(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.CancelExperiment(context.Background(), connect.NewRequest(&experimentv1.CancelExperimentRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-cancel", err, nil)
	}
	exp := resp.Msg.GetExperiment()
	if exp == nil {
		return fmt.Errorf("server returned no experiment")
	}
	return renderExperimentProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Canceled experiment %s (%s).", exp.GetId(), statusLabel(exp.GetStatus()))},
		NextCommand: []string{
			fmt.Sprintf("audio-tools experiment get %s --json", exp.GetId()),
		},
	})
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	if !ctx.BoolFlag("yes") {
		return fmt.Errorf("refusing to delete experiment without --yes confirmation")
	}
	id := ctx.Positional("id")
	resp, err := h.client.DeleteExperiment(context.Background(), connect.NewRequest(&experimentv1.DeleteExperimentRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-delete", err, nil)
	}
	if ctx.JSON() {
		return printExperimentProtoJSON(ctx.Stdout(), resp.Msg)
	}
	report := "no report blob"
	if resp.Msg.GetDeletedReport() {
		report = "report blob deleted"
	}
	return ctx.RenderMutation(cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Deleted experiment %s (%s).", resp.Msg.GetId(), report)},
	})
}

func (h *handlers) watch(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	stream, err := h.client.StreamExperimentEvents(context.Background(), connect.NewRequest(&experimentv1.StreamExperimentEventsRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-watch", err, nil)
	}
	defer stream.Close()
	for stream.Receive() {
		ev := stream.Msg()
		if ctx.JSON() {
			if err := printExperimentProtoJSON(ctx.Stdout(), ev); err != nil {
				return err
			}
			continue
		}
		fmt.Fprintf(ctx.Stdout(), "%s %3d%% %s\n", statusLabel(ev.GetStatus()), ev.GetProgress(), ev.GetMessage())
	}
	if err := stream.Err(); err != nil {
		return cliapp.WrapAPIError("experiment-watch receive", err, nil)
	}
	return nil
}

func (h *handlers) report(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetExperimentReport(context.Background(), connect.NewRequest(&experimentv1.GetExperimentReportRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-report", err, nil)
	}
	if ctx.JSON() {
		return printExperimentProtoJSON(ctx.Stdout(), resp.Msg)
	}
	return renderReport(ctx, resp.Msg)
}

func (h *handlers) compare(ctx cliapp.RunContext) error {
	// Accept both space-separated positionals (`compare a b`) and comma-separated
	// forms (`compare a,b`), flatten, and dedupe while preserving order.
	ids := dedupeStrings(flattenCSV(ctx.Positionals("ids")))
	if len(ids) < 2 {
		return fmt.Errorf("compare requires at least two experiment ids (space- or comma-separated)")
	}
	resp, err := h.client.CompareExperiments(context.Background(), connect.NewRequest(&experimentv1.CompareExperimentsRequest{Ids: ids}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-compare", err, nil)
	}
	if ctx.JSON() {
		return printExperimentProtoJSON(ctx.Stdout(), resp.Msg)
	}
	printComparison(ctx, resp.Msg.GetExperiments())
	return nil
}

func (h *handlers) recordEvidence(ctx cliapp.RunContext) error {
	kind, err := qualificationKindFromFlag(ctx.Flag("kind"))
	if err != nil {
		return err
	}
	passed, err := qualificationOutcomeFromFlag(ctx.Flag("outcome"))
	if err != nil {
		return err
	}
	evidence := &experimentv1.QualificationEvidence{
		EngineId:      strings.TrimSpace(ctx.Flag("engine-id")),
		ModelId:       strings.TrimSpace(ctx.Flag("model-id")),
		Strategy:      strings.TrimSpace(ctx.Flag("strategy")),
		PolicyProfile: strings.TrimSpace(ctx.Flag("policy-profile")),
		Kind:          kind,
		FaultProfile:  strings.TrimSpace(ctx.Flag("fault-profile")),
		Passed:        passed,
		ArtifactRef:   strings.TrimSpace(ctx.Flag("artifact-ref")),
		Notes:         strings.TrimSpace(ctx.Flag("notes")),
		MachineJson:   strings.TrimSpace(ctx.Flag("machine-json")),
	}
	resp, err := h.client.RecordQualificationEvidence(context.Background(), connect.NewRequest(&experimentv1.RecordQualificationEvidenceRequest{Evidence: evidence}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-record-evidence", err, nil)
	}
	return renderExperimentProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Recorded %s qualification evidence for %s/%s/%s (%s).", ctx.Flag("kind"), evidence.GetEngineId(), evidence.GetModelId(), evidence.GetStrategy(), ctx.Flag("outcome"))},
		NextCommand: []string{"audio-tools experiment list-evidence --json"},
	})
}

func (h *handlers) listEvidence(ctx cliapp.RunContext) error {
	resp, err := h.client.ListQualificationEvidence(context.Background(), connect.NewRequest(&experimentv1.ListQualificationEvidenceRequest{
		EngineId: strings.TrimSpace(ctx.Flag("engine-id")), ModelId: strings.TrimSpace(ctx.Flag("model-id")), Strategy: strings.TrimSpace(ctx.Flag("strategy")),
		PolicyProfile: strings.TrimSpace(ctx.Flag("policy-profile")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-list-evidence", err, nil)
	}
	if ctx.JSON() {
		return printExperimentProtoJSON(ctx.Stdout(), resp.Msg)
	}
	if len(resp.Msg.GetEvidence()) == 0 {
		fmt.Fprintln(ctx.Stdout(), "No qualification evidence found.")
		return nil
	}
	fmt.Fprintf(ctx.Stdout(), "%-20s  %-20s  %-16s  %-15s  %-8s  %s\n", "ENGINE", "MODEL", "STRATEGY", "KIND", "OUTCOME", "ARTIFACT")
	for _, evidence := range resp.Msg.GetEvidence() {
		outcome := "failed"
		if evidence.GetPassed() {
			outcome = "passed"
		}
		kind := strings.TrimPrefix(strings.ToLower(evidence.GetKind().String()), "qualification_evidence_kind_")
		if evidence.GetFaultProfile() != "" {
			kind += ":" + evidence.GetFaultProfile()
		}
		fmt.Fprintf(ctx.Stdout(), "%-20s  %-20s  %-16s  %-15s  %-8s  %s\n", evidence.GetEngineId(), evidence.GetModelId(), evidence.GetStrategy(), kind, outcome, evidence.GetArtifactRef())
	}
	return nil
}

func renderExperiment(ctx cliapp.RunContext, msg interface {
	GetExperiment() *experimentv1.Experiment
	GetRuns() []*experimentv1.ExperimentRun
}, op string,
) error {
	switch m := msg.(type) {
	case *experimentv1.GetExperimentResponse:
		if ctx.JSON() {
			return printExperimentProtoJSON(ctx.Stdout(), m)
		}
	case *experimentv1.WaitExperimentResponse:
		if ctx.JSON() {
			return printExperimentProtoJSON(ctx.Stdout(), m)
		}
	}
	exp := msg.GetExperiment()
	if exp == nil {
		return fmt.Errorf("server returned no experiment")
	}
	runs := msg.GetRuns()
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Experiment %s is %s.", exp.GetId(), statusLabel(exp.GetStatus())),
			fmt.Sprintf("name=%q created=%s started=%s finished=%s", exp.GetName(), timestampLabel(exp.GetCreatedAt()), timestampLabel(exp.GetStartedAt()), timestampLabel(exp.GetFinishedAt())),
		},
		ResultsHeading: "Runs",
		RetrievalHints: []string{
			fmt.Sprintf("audio-tools experiment report %s --json", exp.GetId()),
			fmt.Sprintf("audio-tools experiment %s %s --json", op, exp.GetId()),
		},
	}
	if errMsg := strings.TrimSpace(exp.GetError()); errMsg != "" {
		report.Summary = append(report.Summary, fmt.Sprintf("error=%s", errMsg))
	}
	for _, run := range runs {
		report.Results = append(report.Results, formatRunStatus(run))
	}
	if len(runs) == 0 {
		report.Results = append(report.Results, "(none yet)")
	}
	return ctx.RenderList(report)
}

// printExperimentWaitJSON emits a stable {experiment, report, runs} object.
// protojson omits nil singular message fields, so when the run has not produced
// a report yet the `report` key would silently disappear and the parsed shape
// would shift by server timing. We normalize through a generic map and inject an
// explicit report:null so an agent can always parse `.report` unconditionally.
func printExperimentWaitJSON(w io.Writer, env *experimentv1.GetExperimentReportResponse) error {
	body, err := experimentProtoJSONOptions.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal wait envelope: %w", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return fmt.Errorf("normalize wait envelope: %w", err)
	}
	if _, ok := obj["report"]; !ok {
		obj["report"] = nil
	}
	out, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Errorf("encode wait envelope: %w", err)
	}
	out = append(out, '\n')
	_, err = w.Write(out)
	return err
}

func printExperimentProtoJSON(w io.Writer, msg proto.Message) error {
	body, err := experimentProtoJSONOptions.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal experiment proto json: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	if len(body) == 0 || body[len(body)-1] != '\n' {
		_, err = w.Write([]byte{'\n'})
	}
	return err
}

func renderExperimentProtoMutation(ctx cliapp.RunContext, payload proto.Message, human cliapp.MutationReport) error {
	if ctx.JSON() {
		return printExperimentProtoJSON(ctx.Stdout(), payload)
	}
	return ctx.RenderMutation(human)
}
