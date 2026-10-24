package advisor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	advisorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/advisor"
	advisorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/advisor/advisor_v1connect"
)

type handlers struct {
	client advisorconnect.AdvisorServiceClient
}

const advisorTimeout = 20 * time.Minute

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, advisorTimeout)
	return &handlers{client: advisorconnect.NewAdvisorServiceClient(httpClient, baseURL)}
}

func (h *handlers) migrations(ctx cliapp.RunContext) error {
	resp, err := h.client.AnalyzeMigrations(context.Background(), connect.NewRequest(&advisorv1.AnalyzeMigrationsRequest{Scenarios: ctx.FlagValues("scenario")}))
	if err != nil {
		return cliapp.WrapAPIError("analyze storage migrations", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no migration response")
	}
	results := make([]string, 0, len(resp.Msg.GetEntries()))
	for _, entry := range resp.Msg.GetEntries() {
		if entry.GetMigrationDebt() > 0 {
			results = append(results, fmt.Sprintf("%s — stage=%s debt=%d notes=%s", entry.GetScenario(), entry.GetStorageStage(), entry.GetMigrationDebt(), strings.Join(entry.GetNotes(), "; ")))
		}
	}
	if len(results) == 0 {
		results = append(results, "No migration debt found.")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Analyzed %d scenario(s); %d with migration debt.", resp.Msg.GetScenarioCount(), resp.Msg.GetDebtCount())}, ResultsHeading: "Migration debt", Results: results})
}

func (h *handlers) engines(ctx cliapp.RunContext) error {
	resp, err := h.client.AdviseEngines(context.Background(), connect.NewRequest(&advisorv1.AdviseEnginesRequest{Scenarios: ctx.FlagValues("scenario")}))
	if err != nil {
		return cliapp.WrapAPIError("advise storage engines", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no engine response")
	}
	results := make([]string, 0, len(resp.Msg.GetCandidates()))
	for _, candidate := range resp.Msg.GetCandidates() {
		results = append(results, fmt.Sprintf("%s — %s to %s fitness=%.2f blockers=%s", candidate.GetScenario(), candidate.GetCurrentEngine(), candidate.GetRecommendedEngine(), candidate.GetFitnessScore(), strings.Join(candidate.GetBlockers(), "; ")))
	}
	if len(results) == 0 {
		results = append(results, "No engine-migration candidates.")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Scored %d scenario(s); %d candidate(s).", resp.Msg.GetScenarioCount(), len(results))}, ResultsHeading: "Engine candidates", Results: results})
}
