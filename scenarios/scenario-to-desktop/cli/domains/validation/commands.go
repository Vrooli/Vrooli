// Package validation provides a concise, agent-facing validation runner.
package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"scenario-to-desktop/cli/domains/matrix"
	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	httpClient interface {
		DoWithContext(context.Context, string, string, url.Values, interface{}) ([]byte, error)
	}
	Commands struct {
		client httpClient
		prefix string
	}
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	app := deps.ScenarioApp()
	client := cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: app.APIBaseOptions(), Timeout: app.HTTPClient.Timeout()})
	c := &Commands{client: client, prefix: app.APIPrefix()}
	return cliapp.SubcommandGroup{Name: "validation", Description: "Run desktop validation matrices", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "run", Description: "Create, start, and wait for validation", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scenario", Required: true}, {Name: "selection-file", Description: "JSON MatrixSelection file"}, {Name: "artifact-digest", Description: "Artifact digest when building a selection from flags"}, {Name: "artifact-path", Description: "Artifact path when building a selection from flags"}, {Name: "platform", Description: "Target platform or OS (for example mac or windows)"}, {Name: "journey", Description: "Journey id"}, {Name: "profile", Default: "normal", Description: "Environment profile"}, {Name: "json", Bool: true}}}, RunCtx: c.run},
		{Name: "status", Description: "Read one validation run", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "run-id", Required: true}}, Flags: []cliapp.Flag{{Name: "json", Bool: true}}}, RunCtx: c.status},
	}}
}

func (c *Commands) run(ctx cliapp.RunContext) error {
	selectionPath := strings.TrimSpace(ctx.Flag("selection-file"))
	body := map[string]any{}
	var err error
	if selectionPath != "" {
		raw, readErr := os.ReadFile(selectionPath)
		if readErr != nil {
			return fmt.Errorf("read matrix selection: %w", readErr)
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return fmt.Errorf("decode matrix selection: %w", err)
		}
	} else {
		body, err = c.selectionFromFlags(ctx)
		if err != nil {
			return err
		}
	}
	scenarioName, _ := body["scenario_name"].(string)
	if strings.TrimSpace(scenarioName) == "" {
		body["scenario_name"] = strings.TrimSpace(ctx.Flag("scenario"))
	}
	created, err := c.request(ctx, "POST", "/validation/matrices", body)
	if err != nil {
		return err
	}
	runID := stringField(created, "run_id")
	if runID == "" {
		return fmt.Errorf("validation create returned no run_id")
	}
	if _, err := c.request(ctx, "POST", "/validation/matrices/"+url.PathEscape(runID)+"/start", nil); err != nil {
		return err
	}
	result, err := c.request(ctx, "GET", "/validation/matrices/"+url.PathEscape(runID)+"/wait", nil)
	if err != nil {
		return err
	}
	return render(ctx, result, "Validation run "+runID)
}

func (c *Commands) selectionFromFlags(ctx cliapp.RunContext) (map[string]any, error) {
	body := map[string]any{}
	if value := strings.TrimSpace(ctx.Flag("artifact-digest")); value != "" {
		body["artifact_digest"] = value
	}
	if value := strings.TrimSpace(ctx.Flag("artifact-path")); value != "" {
		body["artifact_path"] = value
	}
	if body["artifact_digest"] == nil && body["artifact_path"] == nil {
		return nil, fmt.Errorf("--selection-file or --artifact-digest/--artifact-path is required")
	}
	if value := strings.TrimSpace(ctx.Flag("journey")); value != "" {
		body["journeys"] = []any{map[string]any{"journey_id": value, "display_name": value, "required": true}}
	} else {
		return nil, fmt.Errorf("--journey is required when building a selection from flags")
	}
	profile, err := matrix.ProfileValue(ctx.Flag("profile"))
	if err != nil {
		return nil, err
	}
	body["environment_profiles"] = []any{profile}
	platform := strings.TrimSpace(ctx.Flag("platform"))
	if platform == "" {
		return nil, fmt.Errorf("--platform is required when building a selection from flags")
	}
	target, err := matrix.FindTarget(ctx, c.client, c.prefix, platform)
	if err != nil {
		return nil, err
	}
	body["targets"] = []any{target}
	return body, nil
}

func (c *Commands) status(ctx cliapp.RunContext) error {
	value, err := c.request(ctx, "GET", "/validation/matrices/"+url.PathEscape(ctx.Positional("run-id")), nil)
	if err != nil {
		return err
	}
	return render(ctx, value, "Validation run "+ctx.Positional("run-id"))
}

func (c *Commands) request(_ cliapp.OperationContext, method, path string, body any) (*structpb.Struct, error) {
	raw, err := c.client.DoWithContext(context.Background(), method, strings.TrimRight(c.prefix, "/")+path, nil, body)
	if err != nil {
		return nil, err
	}
	value := &structpb.Struct{}
	if err := protojson.Unmarshal(raw, value); err != nil {
		return nil, fmt.Errorf("decode validation response: %w", err)
	}
	return value, nil
}

func render(ctx cliapp.RunContext, value *structpb.Struct, summary string) error {
	raw, _ := protojson.Marshal(value)
	if ctx.JSON() {
		return ctx.RenderMutation(cliapp.MutationReport{Result: []string{string(raw)}})
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{summary}, Results: []string{string(raw)}, ResultsHeading: "Result"})
}

func stringField(value *structpb.Struct, field string) string {
	if value == nil {
		return ""
	}
	result, _ := value.AsMap()[field].(string)
	return strings.TrimSpace(result)
}
