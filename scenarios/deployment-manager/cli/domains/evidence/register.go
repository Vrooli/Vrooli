package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	evidenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
	offersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const GroupName = "evidence"

type handlers struct {
	client      evidenceconnect.EvidenceServiceClient
	offerClient offersconnect.GatesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	offerBaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OFFER_DESK_API_BASE_URL")), "/")
	return &handlers{
		client:      evidenceconnect.NewEvidenceServiceClient(httpClient, baseURL),
		offerClient: offersconnect.NewGatesServiceClient(httpClient, offerBaseURL),
	}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*evidencev1.ListTargetVerdictsResponse, error) {
	response, err := h.client.ListTargetVerdicts(context.Background(), connect.NewRequest(&evidencev1.ListTargetVerdictsRequest{
		ProfileId:     ctx.Positional("profile_id"),
		GitCommitHash: ctx.Positional("git_commit_hash"),
		PageSize:      1000,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list evidence verdicts", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no evidence response")
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, response *evidencev1.ListTargetVerdictsResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetVerdicts()))
	for _, verdict := range response.GetVerdicts() {
		target := verdict.GetTarget()
		if target == nil {
			results = append(results, fmt.Sprintf("%s — target unspecified", verdict.GetDisposition().String()))
			continue
		}
		results = append(results, fmt.Sprintf("%s — %s/%s (%s)", verdict.GetDisposition().String(), target.GetRamp(), target.GetPlatform(), target.GetOs()))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d evidence verdict(s).", response.GetCount())},
		ResultsHeading: "Target verdicts",
		Results:        results,
		RetrievalHints: []string{"The API response includes producer-owned evidence references and checksums."},
	}
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"EvidenceService.ListTargetVerdicts": cliapp.ProtoList(h.listCall, h.listReport),
		"publish-report-fact": cliapp.ExternalDelegation(
			cliapp.ProtoMutation(h.publishReportFact, h.publishReportFactReport).Run,
		),
	})
}

type deploymentReport struct {
	GeneratedAt string `json:"generated_at"`
}

func defaultReportPath() string {
	if configured := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_REPORT_PATH")); configured != "" {
		return configured
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".vrooli", "data", "vrooli", "deployment-manager", "deployment", "deployment-report.json")
	}
	return filepath.Join(home, ".vrooli", "data", "vrooli", "deployment-manager", "deployment", "deployment-report.json")
}

func readReportFact(path string, now time.Time, staleAfter time.Duration) (*offersv1.Fact, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read deployment report: %w", err)
	}
	var report deploymentReport
	if err := json.Unmarshal(contents, &report); err != nil {
		return nil, fmt.Errorf("decode deployment report: %w", err)
	}
	generated, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(report.GeneratedAt))
	if err != nil {
		return nil, fmt.Errorf("parse deployment report generated_at: %w", err)
	}
	generated = generated.UTC()
	now = now.UTC()
	value := 1.0
	if generated.After(now) || now.Sub(generated) > staleAfter {
		value = 0
	}
	return &offersv1.Fact{
		Name:           "deployment_report_fresh",
		Value:          value,
		ObservedAt:     timestamppb.New(generated),
		StaleAfterDays: int32(staleAfter / (24 * time.Hour)),
		Dimension:      "producer:deployment-manager",
	}, nil
}

func (h *handlers) publishReportFact(ctx cliapp.OperationContext) (*offersv1.AddFactResponse, error) {
	if h.offerClient == nil || strings.TrimSpace(os.Getenv("OFFER_DESK_API_BASE_URL")) == "" {
		return nil, fmt.Errorf("offer-desk is unavailable; deployment report fact was not published")
	}
	staleDays := 30
	if raw := strings.TrimSpace(ctx.Flag("stale-after-days")); raw != "" {
		if _, err := fmt.Sscanf(raw, "%d", &staleDays); err != nil {
			return nil, fmt.Errorf("parse --stale-after-days: %w", err)
		}
		if staleDays <= 0 {
			return nil, fmt.Errorf("parse --stale-after-days: value must be positive")
		}
	}
	reportPath := strings.TrimSpace(ctx.Flag("report-path"))
	if reportPath == "" {
		reportPath = defaultReportPath()
	}
	fact, err := readReportFact(reportPath, time.Now().UTC(), time.Duration(staleDays)*24*time.Hour)
	if err != nil {
		return nil, err
	}
	response, err := h.offerClient.AddFact(context.Background(), connect.NewRequest(&offersv1.AddFactRequest{Fact: fact}))
	if err != nil {
		return nil, cliapp.WrapAPIError("publish deployment report fact", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) publishReportFactReport(_ cliapp.OperationContext, response *offersv1.AddFactResponse) cliapp.MutationReport {
	fact := response.GetFact()
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Producer fact published: %s value=%g observed_at=%s", fact.GetName(), fact.GetValue(), fact.GetObservedAt().AsTime().Format(time.RFC3339))}}
}
