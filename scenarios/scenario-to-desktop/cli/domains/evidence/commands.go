// Package evidence exposes durable desktop-validation evidence to operators.
package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scenario-to-desktop/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	offersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type evidenceRPC interface {
	ListEvidenceCaptures(context.Context, *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.ListEvidenceCapturesResponse], error)
	GetEvidenceCapture(context.Context, *connect.Request[domainv1.GetEvidenceCaptureRequest]) (*connect.Response[domainv1.GetEvidenceCaptureResponse], error)
	GetEvidenceCapturesSummary(context.Context, *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.EvidenceCapturesSummary], error)
	VoidEvidenceCapture(context.Context, *connect.Request[domainv1.VoidEvidenceCaptureRequest]) (*connect.Response[domainv1.EvidenceCapture], error)
}

type gatesRPC interface {
	AddFact(context.Context, *connect.Request[offersv1.AddFactRequest]) (*connect.Response[offersv1.AddFactResponse], error)
}

type Commands struct {
	rpc       evidenceRPC
	gates     gatesRPC
	apiPrefix string
	http      interface {
		DoWithContext(context.Context, string, string, url.Values, interface{}) ([]byte, error)
	}
}

func New(deps support.Dependencies) *Commands {
	app := deps.ScenarioApp()
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	restClient := cliutil.NewHTTPClient(cliutil.HTTPClientOptions{
		BaseOptions: app.APIBaseOptions(),
		Timeout:     app.HTTPClient.Timeout(),
	})
	var gates gatesRPC
	if offerDeskURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OFFER_DESK_API_BASE_URL")), "/"); offerDeskURL != "" {
		gates = offersconnect.NewGatesServiceClient(httpClient, offerDeskURL)
	}
	return &Commands{rpc: domainconnect.NewEvidenceServiceClient(httpClient, baseURL), gates: gates, apiPrefix: app.APIPrefix(), http: restClient}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	c := New(deps)
	scenarioArgs := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario name"}}}
	return cliapp.SubcommandGroup{Name: "evidence", Description: "Inspect durable desktop validation evidence", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "list", Description: "List persisted evidence captures", Args: cliapp.ArgSchema{Positionals: scenarioArgs.Positionals, Flags: []cliapp.Flag{{Name: "pipeline"}, {Name: "session"}, {Name: "kind"}}}}).WithPrimitive(c.listPrimitive()),
		(cliapp.Command{Name: "show", Description: "Export or inspect one evidence capture", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}, {Name: "capture-id", Required: true}}, Flags: []cliapp.Flag{{Name: "output"}}}}).WithPrimitive(c.showPrimitive()),
		(cliapp.Command{Name: "export", Description: "Export all evidence captures with checksums", Args: cliapp.ArgSchema{Positionals: scenarioArgs.Positionals, Flags: []cliapp.Flag{{Name: "pipeline"}, {Name: "output", Required: true}}}}).WithPrimitive(c.exportPrimitive()),
		(cliapp.Command{Name: "journey", Description: "Print the latest desktop journey steps and dispositions", Args: cliapp.ArgSchema{Positionals: scenarioArgs.Positionals, Flags: []cliapp.Flag{{Name: "pipeline"}}}}).WithPrimitive(c.journeyPrimitive()),
		(cliapp.Command{Name: "manifest", Description: "Retrieve an evidence manifest without exposing its host path", Args: cliapp.ArgSchema{Positionals: scenarioArgs.Positionals, Flags: []cliapp.Flag{{Name: "pipeline"}, {Name: "run"}, {Name: "output"}}}}).WithPrimitive(c.manifestPrimitive()),
		(cliapp.Command{Name: "summary", Description: "Summarize persisted evidence captures", Args: scenarioArgs}).WithPrimitive(c.summaryPrimitive()),
		(cliapp.Command{Name: "void", Description: "Void evidence without deleting its capture file", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}, {Name: "capture-id", Required: true}}, Flags: []cliapp.Flag{{Name: "reason", Required: true}, {Name: "superseded-by"}}}}).WithPrimitive(c.voidPrimitive()),
		(cliapp.Command{Name: "publish-offer-fact", Description: "Publish a producer-owned release fact to Offer Desk: publish-offer-fact <trigger-id> --scenario <name> [--observed-at RFC3339]", Args: publishFactArgs()}).WithPrimitive(c.publishOfferFactPrimitive()),
	}}
}

func (c *Commands) manifestPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*structpb.Struct, error) {
		if c.http == nil {
			return nil, fmt.Errorf("scenario API HTTP client is unavailable")
		}
		identity := strings.TrimSpace(ctx.Flag("pipeline"))
		if identity == "" {
			identity = strings.TrimSpace(ctx.Flag("run"))
		}
		if identity == "" {
			return nil, fmt.Errorf("--pipeline or --run is required")
		}
		path := strings.TrimRight(c.apiPrefix, "/") + "/captures/" + url.PathEscape(ctx.Positional("scenario")) + "/manifest"
		value, err := c.http.DoWithContext(context.Background(), "GET", path, url.Values{"pipeline": []string{identity}}, nil)
		if err != nil {
			return nil, cliapp.WrapAPIError("retrieve evidence manifest", err, nil)
		}
		if output := strings.TrimSpace(ctx.Flag("output")); output != "" {
			if err := os.WriteFile(output, value, 0o600); err != nil {
				return nil, err
			}
			return &structpb.Struct{Fields: map[string]*structpb.Value{"output": structpb.NewStringValue(output)}}, nil
		}
		result := &structpb.Struct{}
		if err := protojson.Unmarshal(value, result); err != nil {
			return nil, fmt.Errorf("decode evidence manifest: %w", err)
		}
		return result, nil
	}, func(ctx cliapp.OperationContext, response *structpb.Struct) cliapp.MutationReport {
		if output := strings.TrimSpace(ctx.Flag("output")); output != "" {
			return cliapp.MutationReport{Result: []string{"Evidence manifest written: " + output}}
		}
		return cliapp.MutationReport{Result: []string{string(mustJSON(response))}}
	})
}

func mustJSON(value *structpb.Struct) []byte {
	data, _ := protojson.Marshal(value)
	return data
}

func (c *Commands) voidPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*domainv1.EvidenceCapture, error) {
		response, err := c.rpc.VoidEvidenceCapture(context.Background(), connect.NewRequest(&domainv1.VoidEvidenceCaptureRequest{ScenarioName: ctx.Positional("scenario"), CaptureId: ctx.Positional("capture-id"), Reason: ctx.Flag("reason"), SupersededBy: optionalString(ctx.Flag("superseded-by"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("void evidence capture", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.EvidenceCapture) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Evidence capture voided: %s", response.GetCaptureId())}}
	})
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func publishFactArgs() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "trigger-id", Required: true, Description: "Offer Desk trigger identifier"}},
		Flags: []cliapp.Flag{
			{Name: "scenario", Required: true, Description: "Producer scenario whose release fact is being reported"},
			{Name: "stale-after-days", Default: "30", Description: "Fact freshness window"},
		},
	}
}

type desktopBuildRecord struct {
	ScenarioName string `json:"scenario_name"`
	UpdatedAt    string `json:"updated_at"`
}

func desktopRecordsPath() string {
	if configured := strings.TrimSpace(os.Getenv("SCENARIO_TO_DESKTOP_RECORDS_PATH")); configured != "" {
		return configured
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".vrooli", "data", "vrooli", "scenario-to-desktop", "desktop_records_v2.json")
	}
	return filepath.Join(home, ".vrooli", "data", "vrooli", "scenario-to-desktop", "desktop_records_v2.json")
}

func readDesktopBuildFact(path, scenario string, staleDays int32) (*offersv1.Fact, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read desktop build records: %w", err)
	}
	var records []desktopBuildRecord
	if err := json.Unmarshal(contents, &records); err != nil {
		return nil, fmt.Errorf("decode desktop build records: %w", err)
	}
	var latest time.Time
	for _, record := range records {
		if record.ScenarioName != scenario {
			continue
		}
		updated, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(record.UpdatedAt))
		if parseErr != nil {
			return nil, fmt.Errorf("parse desktop build timestamp for %q: %w", scenario, parseErr)
		}
		if updated.After(latest) {
			latest = updated
		}
	}
	if latest.IsZero() {
		return nil, fmt.Errorf("no desktop build record exists for %q", scenario)
	}
	return &offersv1.Fact{
		Name:           "release_gate_passed." + scenario,
		Value:          1,
		ObservedAt:     timestamppb.New(latest.UTC()),
		StaleAfterDays: staleDays,
		Dimension:      "producer:scenario-to-desktop",
	}, nil
}

func (c *Commands) publishOfferFactPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*offersv1.AddFactResponse, error) {
		if c.gates == nil {
			return nil, fmt.Errorf("offer-desk is unavailable; producer fact was not published")
		}
		stale := int32(30)
		if raw := strings.TrimSpace(ctx.Flag("stale-after-days")); raw != "" {
			if _, err := fmt.Sscanf(raw, "%d", &stale); err != nil {
				return nil, fmt.Errorf("parse --stale-after-days: %w", err)
			}
		}
		if stale <= 0 {
			return nil, fmt.Errorf("parse --stale-after-days: value must be positive")
		}
		scenario := strings.TrimSpace(ctx.Flag("scenario"))
		fact, err := readDesktopBuildFact(desktopRecordsPath(), scenario, stale)
		if err != nil {
			return nil, err
		}
		request := &offersv1.AddFactRequest{Fact: fact}
		response, err := c.gates.AddFact(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("publish producer fact", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *offersv1.AddFactResponse) cliapp.MutationReport {
		fact := response.GetFact()
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Producer fact published: %s value=%g dimension=%s observed_at=%s", fact.GetName(), fact.GetValue(), fact.GetDimension(), fact.GetObservedAt().AsTime().Format(time.RFC3339))}}
	})
}

type journeyReport struct {
	Disposition           string `json:"disposition"`
	DegradedReason        string `json:"degraded_reason"`
	IsolationObservations []struct {
		StateRoot           string `json:"state_root"`
		SocketPath          string `json:"socket_path"`
		DatabasePath        string `json:"database_path"`
		AdoptedSessionCount int    `json:"adopted_session_count"`
		Source              string `json:"source"`
	} `json:"isolation_observations"`
	Selection *struct {
		Capability string   `json:"capability"`
		Reason     string   `json:"reason"`
		Skipped    []string `json:"skipped"`
	} `json:"selection,omitempty"`
	WindowManager string `json:"window_manager"`
	Titlebar      bool   `json:"titlebar"`
	Steps         []struct {
		Name            string `json:"name"`
		Action          string `json:"action"`
		Disposition     string `json:"disposition"`
		BeforeCaptureID string `json:"before_capture_id"`
		AfterCaptureID  string `json:"after_capture_id"`
		DegradedReason  string `json:"degraded_reason"`
		Error           string `json:"error"`
	} `json:"steps"`
}

func (c *Commands) journeyPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.GetEvidenceCaptureResponse, error) {
		scenario := strings.TrimSpace(ctx.Positional("scenario"))
		pipelineID := strings.TrimSpace(ctx.Flag("pipeline"))
		list, err := c.rpc.ListEvidenceCaptures(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{ScenarioName: scenario, PipelineId: optionalString(pipelineID)}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list journey captures", err, nil)
		}
		var latest *domainv1.EvidenceCapture
		for _, item := range list.Msg.GetCaptures() {
			if item.GetKind() == "journey" {
				latest = item
			}
		}
		if latest == nil {
			return nil, cliapp.WrapAPIError("find journey capture", fmt.Errorf("no journey capture exists for %q", scenario), nil)
		}
		response, err := c.rpc.GetEvidenceCapture(context.Background(), connect.NewRequest(&domainv1.GetEvidenceCaptureRequest{ScenarioName: scenario, CaptureId: latest.GetCaptureId()}))
		if err != nil {
			return nil, cliapp.WrapAPIError("read journey capture", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.GetEvidenceCaptureResponse) cliapp.ListReport {
		var report journeyReport
		if err := json.Unmarshal(response.GetContent(), &report); err != nil {
			return cliapp.ListReport{Summary: []string{"Journey capture is unreadable"}, Results: []string{err.Error()}}
		}
		results := []string{fmt.Sprintf("Disposition: %s", report.Disposition), fmt.Sprintf("Window manager: %s (titlebar=%t)", report.WindowManager, report.Titlebar)}
		if report.DegradedReason != "" {
			results = append(results, "Degraded reason: "+report.DegradedReason)
		}
		if report.Selection != nil {
			results = append(results, fmt.Sprintf("Capability selected: %s (reason=%s)", report.Selection.Capability, report.Selection.Reason))
			if len(report.Selection.Skipped) > 0 {
				results = append(results, "Skipped capabilities: "+strings.Join(report.Selection.Skipped, ", "))
			}
		}
		for _, observation := range report.IsolationObservations {
			results = append(results, fmt.Sprintf("Isolation observation: source=%s adopted_sessions=%d state_root=%s socket_path=%s database_path=%s", observation.Source, observation.AdoptedSessionCount, observation.StateRoot, observation.SocketPath, observation.DatabasePath))
		}
		for _, step := range report.Steps {
			line := fmt.Sprintf("%s [%s] %s", step.Name, step.Action, step.Disposition)
			if step.BeforeCaptureID != "" && step.AfterCaptureID != "" {
				line += fmt.Sprintf(" screenshots=%s,%s", step.BeforeCaptureID, step.AfterCaptureID)
			}
			if step.DegradedReason != "" {
				line += " reason=" + step.DegradedReason
			}
			if step.Error != "" {
				line += " error=" + step.Error
			}
			results = append(results, line)
		}
		return cliapp.ListReport{Summary: []string{"Desktop journey retrieved"}, ResultsHeading: "Steps", Results: results}
	})
}

func (c *Commands) exportPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.ListEvidenceCapturesResponse, error) {
		scenario := strings.TrimSpace(ctx.Positional("scenario"))
		output := strings.TrimSpace(ctx.Flag("output"))
		if output == "" {
			return nil, fmt.Errorf("--output is required")
		}
		if err := os.MkdirAll(output, 0o700); err != nil {
			return nil, fmt.Errorf("create export directory: %w", err)
		}
		list, err := c.rpc.ListEvidenceCaptures(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{ScenarioName: scenario, PipelineId: optionalString(strings.TrimSpace(ctx.Flag("pipeline")))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list evidence captures", err, nil)
		}
		lines := make([]string, 0, len(list.Msg.GetCaptures()))
		index := []string{"# Scenario-to-desktop evidence", "", "| Capture ID | Kind | File | Checksum |", "|---|---|---|---|"}
		for _, item := range list.Msg.GetCaptures() {
			response, err := c.rpc.GetEvidenceCapture(context.Background(), connect.NewRequest(&domainv1.GetEvidenceCaptureRequest{ScenarioName: scenario, CaptureId: item.GetCaptureId()}))
			if err != nil {
				return nil, cliapp.WrapAPIError("read evidence capture", err, nil)
			}
			name := item.GetCaptureId() + "-" + filepath.Base(item.GetFilename())
			if name == "." || name == "" {
				name = item.GetCaptureId()
			}
			if err := os.WriteFile(filepath.Join(output, name), response.Msg.GetContent(), 0o600); err != nil {
				return nil, fmt.Errorf("write capture %s: %w", item.GetCaptureId(), err)
			}
			lines = append(lines, fmt.Sprintf("%s  %s", item.GetChecksum(), name))
			index = append(index, fmt.Sprintf("| `%s` | %s | `%s` | `%s` |", item.GetCaptureId(), item.GetKind(), name, item.GetChecksum()))
		}
		index = append(index, "", "SHA256SUMS records producer checksums.")
		if err := os.WriteFile(filepath.Join(output, "README.md"), []byte(strings.Join(index, "\n")+"\n"), 0o600); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(output, "SHA256SUMS"), []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
			return nil, err
		}
		return list.Msg, nil
	}, func(ctx cliapp.OperationContext, response *domainv1.ListEvidenceCapturesResponse) cliapp.ListReport {
		results := make([]string, 0, len(response.GetCaptures()))
		for _, capture := range response.GetCaptures() {
			results = append(results, fmt.Sprintf("%s  %s  %s  %s", capture.GetCaptureId(), capture.GetKind(), capture.GetFilename(), capture.GetChecksum()))
		}
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("Evidence exported: %d capture(s) to %s", len(response.GetCaptures()), ctx.Flag("output"))}, ResultsHeading: "Captures", Results: results, ResultCount: len(results), ListShaped: true}
	})
}

func (c *Commands) listPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.ListEvidenceCapturesResponse, error) {
		pipeline, session, kind := "", "", ""
		if ctx.FlagProvided("pipeline") {
			pipeline = ctx.Flag("pipeline")
		}
		if ctx.FlagProvided("session") {
			session = ctx.Flag("session")
		}
		if ctx.FlagProvided("kind") {
			kind = ctx.Flag("kind")
		}
		response, err := c.rpc.ListEvidenceCaptures(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario")), PipelineId: optionalString(pipeline), SourceSessionId: optionalString(session), Kind: optionalString(kind)}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list evidence captures", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.ListEvidenceCapturesResponse) cliapp.ListReport {
		results := make([]string, 0, len(response.GetCaptures()))
		for _, capture := range response.GetCaptures() {
			results = append(results, fmt.Sprintf("%s  %s  %s  %d bytes  %s  %s", capture.GetCaptureId(), capture.GetKind(), capture.GetCreatedAt().AsTime().Format(time.RFC3339), capture.GetFileSizeBytes(), capture.GetSourceSessionId(), capture.GetFilename()))
		}
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("Evidence captures retrieved: %d", len(results))}, Results: results}
	})
}

func (c *Commands) showPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.GetEvidenceCaptureResponse, error) {
		response, err := c.rpc.GetEvidenceCapture(context.Background(), connect.NewRequest(&domainv1.GetEvidenceCaptureRequest{ScenarioName: ctx.Positional("scenario"), CaptureId: ctx.Positional("capture-id")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("show evidence capture", err, nil)
		}
		if output := strings.TrimSpace(ctx.Flag("output")); output != "" {
			if err := os.WriteFile(output, response.Msg.GetContent(), 0o600); err != nil {
				return nil, fmt.Errorf("write evidence capture: %w", err)
			}
		}
		return response.Msg, nil
	}, func(ctx cliapp.OperationContext, response *domainv1.GetEvidenceCaptureResponse) cliapp.ListReport {
		capture := response.GetCapture()
		if output := strings.TrimSpace(ctx.Flag("output")); output != "" {
			return cliapp.ListReport{Summary: []string{"Evidence capture exported"}, Results: []string{output}}
		}
		return cliapp.ListReport{Summary: []string{"Evidence capture metadata"}, Results: []string{fmt.Sprintf("%s  %s  %s  %d bytes", capture.GetCaptureId(), capture.GetKind(), capture.GetFilename(), capture.GetFileSizeBytes())}}
	})
}

func (c *Commands) summaryPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.EvidenceCapturesSummary, error) {
		response, err := c.rpc.GetEvidenceCapturesSummary(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get evidence captures summary", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.EvidenceCapturesSummary) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Evidence capture summary retrieved"}, Results: []string{fmt.Sprintf("Captures: %d", response.GetCount()), fmt.Sprintf("Bytes: %d", response.GetTotalBytes())}}
	})
}
