package planning

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"connectrpc.com/connect"

	planningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning"
	planningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning/planning_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client planningconnect.PlanningServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: planningconnect.NewPlanningServiceClient(httpClient, baseURL)}
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	resp, err := h.client.CreatePlannedScenario(context.Background(), connect.NewRequest(&planningv1.CreatePlannedScenarioRequest{
		Slug:            ctx.Positional("slug"),
		DisplayName:     ctx.Flag("display-name"),
		Sector:          ctx.Flag("sector"),
		Tier:            ctx.Flag("tier"),
		TargetStability: ctx.Flag("stability"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("create planned scenario", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{scenarioLine(resp.Msg)},
		Changes:     []string{fmt.Sprintf("Planned scenario %s saved.", resp.Msg.GetSlug())},
		NextCommand: []string{fmt.Sprintf("tech-tree-designer plan add %s %s/v1/api/service.proto --from-file <file>", resp.Msg.GetSlug(), resp.Msg.GetSlug())},
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListPlannedScenarios(context.Background(), connect.NewRequest(&planningv1.ListPlannedScenariosRequest{
		Sector: ctx.Flag("sector"),
		Tier:   ctx.Flag("tier"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list planned scenarios", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetScenarios()))
	for _, scenario := range resp.Msg.GetScenarios() {
		results = append(results, scenarioLine(scenario))
	}
	if len(results) == 0 {
		results = append(results, "(no planned scenarios)")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d planned scenario(s).", len(resp.Msg.GetScenarios()))},
		ResultsHeading: "Scenarios",
		Results:        results,
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.GetPlannedScenario(context.Background(), connect.NewRequest(&planningv1.GetPlannedScenarioRequest{Slug: ctx.Positional("slug")}))
	if err != nil {
		return cliapp.WrapAPIError("get planned scenario", err, nil)
	}
	args := ctx.Args()
	if len(args) >= 2 {
		path := args[1]
		for _, file := range resp.Msg.GetFiles() {
			if file.GetPath() == path {
				if ctx.JSON() {
					return cliapp.PrintProtoJSON(ctx.Stdout(), file)
				}
				_, err := fmt.Fprint(ctx.Stdout(), file.GetText())
				return err
			}
		}
		return fmt.Errorf("planned proto file %q not found", path)
	}
	paths := make([]string, 0, len(resp.Msg.GetFiles()))
	for _, file := range resp.Msg.GetFiles() {
		paths = append(paths, file.GetPath())
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		paths = append(paths, "(no files)")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{scenarioLine(resp.Msg)},
		ResultsHeading: "Files",
		Results:        paths,
		RetrievalHints: []string{
			fmt.Sprintf("`tech-tree-designer plan tree %s <path>` — print file text", resp.Msg.GetSlug()),
			fmt.Sprintf("`tech-tree-designer plan validate %s` — validate stored protos", resp.Msg.GetSlug()),
		},
	})
}

func (h *handlers) put(ctx cliapp.RunContext) error {
	text, err := readProtoText(ctx.Flag("from-file"))
	if err != nil {
		return err
	}
	resp, err := h.client.PutPlannedProtoFile(context.Background(), connect.NewRequest(&planningv1.PutPlannedProtoFileRequest{
		Slug: ctx.Positional("slug"),
		Path: ctx.Positional("path"),
		Text: text,
	}))
	if err != nil {
		return cliapp.WrapAPIError("put planned proto file", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{resp.Msg.GetPath()},
		Changes:     []string{fmt.Sprintf("Stored %s.", resp.Msg.GetPath())},
		NextCommand: []string{fmt.Sprintf("tech-tree-designer plan validate %s", ctx.Positional("slug"))},
	})
}

func (h *handlers) remove(ctx cliapp.RunContext) error {
	resp, err := h.client.DeletePlannedProtoFile(context.Background(), connect.NewRequest(&planningv1.DeletePlannedProtoFileRequest{
		Slug: ctx.Positional("slug"),
		Path: ctx.Positional("path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("remove planned proto file", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{ctx.Positional("path")},
		Changes: []string{fmt.Sprintf("Deleted: %t.", resp.Msg.GetDeleted())},
	})
}

func (h *handlers) validate(ctx cliapp.RunContext) error {
	resp, err := h.client.ValidatePlannedScenario(context.Background(), connect.NewRequest(&planningv1.ValidatePlannedScenarioRequest{Slug: ctx.Positional("slug")}))
	if err != nil {
		return cliapp.WrapAPIError("validate planned scenario", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetFindings()))
	for _, finding := range resp.Msg.GetFindings() {
		results = append(results, fmt.Sprintf("%s %s %s: %s", finding.GetSeverity().String(), finding.GetCode(), finding.GetLocation(), finding.GetMessage()))
	}
	if len(results) == 0 {
		results = append(results, "(no findings)")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Validation passed: %t.", resp.Msg.GetPassed())},
		ResultsHeading: "Findings",
		Results:        results,
	})
}

func (h *handlers) materialize(ctx cliapp.RunContext) error {
	resp, err := h.client.MaterializePlannedScenario(context.Background(), connect.NewRequest(&planningv1.MaterializePlannedScenarioRequest{Slug: ctx.Positional("slug")}))
	if err != nil {
		return cliapp.WrapAPIError("materialize planned scenario", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  resp.Msg.GetWrittenPaths(),
		Changes: []string{fmt.Sprintf("Materialized %s; generated=%t.", resp.Msg.GetSlug(), resp.Msg.GetGenerated())},
	})
}

func readProtoText(fromFile string) (string, error) {
	fromFile = strings.TrimSpace(fromFile)
	if fromFile == "" || fromFile == "-" {
		data, err := os.ReadFile("/dev/stdin")
		if err != nil {
			return "", fmt.Errorf("read proto text from stdin: %w", err)
		}
		return string(data), nil
	}
	data, err := os.ReadFile(fromFile)
	if err != nil {
		return "", fmt.Errorf("read proto text from %s: %w", fromFile, err)
	}
	return string(data), nil
}

func scenarioLine(s *planningv1.PlannedScenario) string {
	return fmt.Sprintf("%s [%s %s] files=%d", s.GetSlug(), s.GetSector(), s.GetTier(), len(s.GetFiles()))
}
