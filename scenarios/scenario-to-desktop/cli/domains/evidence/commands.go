// Package evidence exposes durable desktop-validation evidence to operators.
package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"scenario-to-desktop/cli/internal/support"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
)

type evidenceRPC interface {
	ListEvidenceCaptures(context.Context, *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.ListEvidenceCapturesResponse], error)
	GetEvidenceCapture(context.Context, *connect.Request[domainv1.GetEvidenceCaptureRequest]) (*connect.Response[domainv1.GetEvidenceCaptureResponse], error)
	GetEvidenceCapturesSummary(context.Context, *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.EvidenceCapturesSummary], error)
}

type Commands struct{ rpc evidenceRPC }

func New(deps support.Dependencies) *Commands {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(deps.ScenarioApp())
	return &Commands{rpc: domainconnect.NewEvidenceServiceClient(httpClient, baseURL)}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	c := New(deps)
	scenarioArgs := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario name"}}}
	return cliapp.SubcommandGroup{Name: "evidence", Description: "Inspect durable desktop validation evidence", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "list", Description: "List persisted evidence captures", Args: scenarioArgs}).WithPrimitive(c.listPrimitive()),
		(cliapp.Command{Name: "journey", Description: "Print the latest desktop journey steps and dispositions", Args: scenarioArgs}).WithPrimitive(c.journeyPrimitive()),
		(cliapp.Command{Name: "summary", Description: "Summarize persisted evidence captures", Args: scenarioArgs}).WithPrimitive(c.summaryPrimitive()),
	}}
}

type journeyReport struct {
	Disposition    string `json:"disposition"`
	DegradedReason string `json:"degraded_reason"`
	WindowManager  string `json:"window_manager"`
	Titlebar       bool   `json:"titlebar"`
	Steps          []struct {
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
		list, err := c.rpc.ListEvidenceCaptures(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{ScenarioName: scenario}))
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

func (c *Commands) listPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.ListEvidenceCapturesResponse, error) {
		response, err := c.rpc.ListEvidenceCaptures(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list evidence captures", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.ListEvidenceCapturesResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Evidence captures retrieved"}, Results: []string{fmt.Sprintf("Captures: %d", len(response.GetCaptures()))}}
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
