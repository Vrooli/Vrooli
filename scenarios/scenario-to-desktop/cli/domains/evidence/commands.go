// Package evidence exposes durable desktop-validation evidence to operators.
package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scenario-to-desktop/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	offersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type evidenceRPC interface {
	ListEvidenceCaptures(context.Context, *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.ListEvidenceCapturesResponse], error)
	GetEvidenceCapture(context.Context, *connect.Request[domainv1.GetEvidenceCaptureRequest]) (*connect.Response[domainv1.GetEvidenceCaptureResponse], error)
	GetEvidenceCapturesSummary(context.Context, *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.EvidenceCapturesSummary], error)
}

type gatesRPC interface {
	AddFact(context.Context, *connect.Request[offersv1.AddFactRequest]) (*connect.Response[offersv1.AddFactResponse], error)
}

type Commands struct {
	rpc   evidenceRPC
	gates gatesRPC
}

func New(deps support.Dependencies) *Commands {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(deps.ScenarioApp())
	var gates gatesRPC
	if offerDeskURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OFFER_DESK_API_BASE_URL")), "/"); offerDeskURL != "" {
		gates = offersconnect.NewGatesServiceClient(httpClient, offerDeskURL)
	}
	return &Commands{rpc: domainconnect.NewEvidenceServiceClient(httpClient, baseURL), gates: gates}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	c := New(deps)
	scenarioArgs := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario name"}}}
	return cliapp.SubcommandGroup{Name: "evidence", Description: "Inspect durable desktop validation evidence", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "list", Description: "List persisted evidence captures", Args: scenarioArgs}).WithPrimitive(c.listPrimitive()),
		(cliapp.Command{Name: "journey", Description: "Print the latest desktop journey steps and dispositions", Args: scenarioArgs}).WithPrimitive(c.journeyPrimitive()),
		(cliapp.Command{Name: "summary", Description: "Summarize persisted evidence captures", Args: scenarioArgs}).WithPrimitive(c.summaryPrimitive()),
		(cliapp.Command{Name: "publish-offer-fact", Description: "Publish a producer-owned release fact to Offer Desk: publish-offer-fact <trigger-id> --scenario <name> [--observed-at RFC3339]", Args: publishFactArgs()}).WithPrimitive(c.publishOfferFactPrimitive()),
	}}
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
