package scenarios

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios"
	scenariosconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios/scenarios_v1connect"
)

type handlers struct {
	core            *cliapp.ScenarioApp
	client          scenariosconnect.ScenariosServiceClient
	lifecycleClient scenariosconnect.ScenariosServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	lifecycleHTTPClient, lifecycleBaseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 3*time.Minute)
	return &handlers{
		core:            core,
		client:          scenariosconnect.NewScenariosServiceClient(httpClient, baseURL),
		lifecycleClient: scenariosconnect.NewScenariosServiceClient(lifecycleHTTPClient, lifecycleBaseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListScenarios(context.Background(), connect.NewRequest(&scenariosv1.ListScenariosRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list scenarios", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Scenarios))
	for _, s := range resp.Msg.Scenarios {
		results = append(results, fmt.Sprintf("%s | %s | flows=%d | %s", s.Id, s.DisplayName, s.FlowCount, s.Path))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d scenario(s) under %s.", len(resp.Msg.Scenarios), resp.Msg.VrooliRoot)},
		ResultsHeading: "Scenarios",
		Results:        results,
		RetrievalHints: []string{"`scenarios show <id>` — list the flows inside one scenario"},
	})
}

func (h *handlers) show(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetScenario(context.Background(), connect.NewRequest(&scenariosv1.GetScenarioRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("show scenario %q", id), err, nil)
	}
	s := resp.Msg.Scenario
	if s == nil || s.Summary == nil {
		return fmt.Errorf("server returned no scenario")
	}
	results := []string{
		fmt.Sprintf("id           = %s", s.Summary.Id),
		fmt.Sprintf("displayName  = %s", s.Summary.DisplayName),
		fmt.Sprintf("description  = %s", s.Summary.Description),
		fmt.Sprintf("path         = %s", s.Summary.Path),
		fmt.Sprintf("flowCount    = %d", s.Summary.FlowCount),
	}
	for _, f := range s.Flows {
		results = append(results, fmt.Sprintf("  - %s (%s)", f.FlowId, f.Language))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scenario %s", s.Summary.Id)},
		ResultsHeading: "Detail",
		Results:        results,
	})
}

func (h *handlers) lifecycles(ctx cliapp.RunContext) error {
	var huntIndex []byte
	if path := ctx.Flag("hunt-index"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read hunt index %q: %w", path, err)
		}
		huntIndex = data
	}
	huntCoverage := lifecycleHuntCoverage(huntIndex)
	resp, err := h.lifecycleClient.ListLifecycles(context.Background(), connect.NewRequest(&scenariosv1.ListLifecyclesRequest{
		ScenarioId: ctx.Flag("scenario"), HuntIndexJson: huntIndex,
	}))
	if err != nil {
		return cliapp.WrapAPIError("infer lifecycle candidates", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Candidates))
	for _, candidate := range resp.Msg.Candidates {
		states := make([]string, 0, len(candidate.States))
		for _, state := range candidate.States {
			states = append(states, state.Value)
		}
		huntIDs := make([]string, 0, len(candidate.HuntRecords))
		for _, record := range candidate.HuntRecords {
			huntIDs = append(huntIDs, record.EntryId)
		}
		results = append(results, fmt.Sprintf("%4d | %s | %s.%s | persistence=%s | preselection=%s | states=%s | flags=%d writes=%d guards=%d tests=%d flows=%d owners=%d modeled=%t hunt=%v | %s",
			candidate.BurdenScore, candidate.Confidence, candidate.TypeName, candidate.FieldName,
			candidate.PersistenceClass, candidate.PreselectionStatus, strings.Join(states, ","), len(candidate.RelatedFlags), len(candidate.WriteSites), len(candidate.GuardSites), len(candidate.TestFiles), len(candidate.ExplicitFlows), len(candidate.TransitionOwners), candidate.ExplicitlyModeled, huntIDs, candidate.Package))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scanned %d scenario(s), %d Go lifecycle candidates, %d eligible for source-backed preselection, %d withheld from ranking, %d Go parse error(s), %d flow-contract parse error(s). Preselection requires a complete %s hunt export, source-backed persisted class, state/write/owner evidence, and no prior hunt match; candidates still need caller and owner adjudication. Unknown and unresolved candidates remain in the unranked inventory. Persistence classification covers scenario API production Go only. Hunt evidence: %s (complete=%t).", resp.Msg.ScenarioCount, resp.Msg.CandidateCount, resp.Msg.PreselectionCandidateCount, max(0, int(resp.Msg.CandidateCount-resp.Msg.PreselectionCandidateCount)), resp.Msg.ParseErrorCount, resp.Msg.FlowContractParseErrorCount, resp.Msg.HuntCoverageScope, huntCoverage, resp.Msg.HuntCoverageComplete)},
		ResultsHeading: "Inferred lifecycle candidates (highest burden first)",
		Results:        results,
		RetrievalHints: []string{"`scenarios lifecycles --scenario <id>` — focus one owner", "`scenarios lifecycles --hunt-index <recall.json>` — attach review-only hunt record references using exact `candidate_ids` bindings or specific source identity terms"},
	})
}

func lifecycleHuntCoverage(data []byte) string {
	if len(data) == 0 {
		return "unavailable (no index supplied)"
	}
	var envelope struct {
		Coverage *struct {
			Complete   bool     `json:"complete"`
			Degraded   []string `json:"degraded"`
			EntryCount int      `json:"entry_count"`
			NextCursor *string  `json:"next_cursor"`
		} `json:"coverage"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "unknown (index has no readable coverage metadata)"
	}
	if envelope.Coverage == nil {
		return "unknown (index has no readable coverage metadata)"
	}
	c := envelope.Coverage
	if !c.Complete || len(c.Degraded) > 0 || c.NextCursor != nil {
		return fmt.Sprintf("partial/degraded (%d entries, complete=%t, next_cursor=%t, degraded=%v)", c.EntryCount, c.Complete, c.NextCursor != nil, c.Degraded)
	}
	return fmt.Sprintf("complete (%d entries)", c.EntryCount)
}
