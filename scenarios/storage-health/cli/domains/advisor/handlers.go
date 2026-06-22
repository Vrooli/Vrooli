package advisor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	advisorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-health/v1/advisor"
	advisorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-health/v1/advisor/advisor_v1connect"
)

// advisorTimeout is generous: a whole-fleet analysis runs static storage
// validation over every discovered scenario.
const advisorTimeout = 5 * time.Minute

type handlers struct {
	client advisorconnect.AdvisorServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, advisorTimeout)
	return &handlers{client: advisorconnect.NewAdvisorServiceClient(httpClient, baseURL)}
}

// migrations grades migration hygiene per scenario.
func (h *handlers) migrations(ctx cliapp.RunContext) error {
	resp, err := h.client.AnalyzeMigrations(context.Background(), connect.NewRequest(&advisorv1.AnalyzeMigrationsRequest{
		Scenarios: ctx.FlagValues("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("analyze migrations", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no migration response")
	}
	msg := resp.Msg
	summary := []string{fmt.Sprintf("%d scenario(s): %d with migrations/, %d carrying migration debt.",
		msg.GetScenarioCount(), msg.GetWithMigrationsCount(), msg.GetDebtCount())}
	var results []string
	for _, e := range msg.GetEntries() {
		if e.GetMigrationDebt() == 0 {
			continue
		}
		line := fmt.Sprintf("%s — stage=%s debt=%d", e.GetScenario(), e.GetStorageStage(), e.GetMigrationDebt())
		for _, note := range e.GetNotes() {
			line += "\n    · " + note
		}
		results = append(results, line)
	}
	if len(results) == 0 {
		results = append(results, "No migration debt across the analyzed scenarios.")
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Migration debt",
		Results:        results,
	})
}

// engines ranks Postgres→SQLite migration candidates.
func (h *handlers) engines(ctx cliapp.RunContext) error {
	resp, err := h.client.AdviseEngines(context.Background(), connect.NewRequest(&advisorv1.AdviseEnginesRequest{
		Scenarios: ctx.FlagValues("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("advise engines", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no advisor response")
	}
	msg := resp.Msg
	summary := []string{fmt.Sprintf("%d scenario(s) scored; %d Postgres→SQLite candidate(s).",
		msg.GetScenarioCount(), len(msg.GetCandidates()))}
	var results []string
	for _, c := range msg.GetCandidates() {
		line := fmt.Sprintf("%s — %s→%s fitness=%.2f", c.GetScenario(), c.GetCurrentEngine(), c.GetRecommendedEngine(), c.GetFitnessScore())
		if len(c.GetBlockers()) > 0 {
			line += "\n    blockers: " + strings.Join(c.GetBlockers(), "; ")
		}
		results = append(results, line)
	}
	if len(results) == 0 {
		results = append(results, "No engine-migration candidates.")
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Postgres→SQLite candidates (ranked by fitness)",
		Results:        results,
	})
}
