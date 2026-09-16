package matrix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"scenario-to-desktop/cli/internal/support"
)

// Register exposes the provider-neutral validation matrix HTTP contract. The
// matrix service is intentionally REST-shaped today, so these commands stay
// thin and return the server's structured result without duplicating its state.
func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	client := deps.ScenarioApp().HTTPClient
	return cliapp.SubcommandGroup{
		Name:        "matrix",
		Description: "Create and inspect desktop validation matrices",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "create", Description: "Create a validation matrix", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}, Flags: []cliapp.Flag{{Name: "selection-file", Description: "JSON MatrixSelection file"}}}, RunCtx: func(ctx cliapp.RunContext) error {
				body := map[string]any{"scenario_name": strings.TrimSpace(ctx.Positional("scenario"))}
				if path := strings.TrimSpace(ctx.Flag("selection-file")); path != "" {
					raw, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					if err := json.Unmarshal(raw, &body); err != nil {
						return fmt.Errorf("decode matrix selection: %w", err)
					}
				}
				value, err := request(ctx, client, http.MethodPost, "/validation/matrices", body)
				if err != nil {
					return err
				}
				return ctx.RenderMutation(cliapp.MutationReport{Result: []string{"Validation matrix created", matrixID(value)}})
			}},
			{Name: "start", Description: "Start a validation matrix", Args: idArgs(), RunCtx: func(ctx cliapp.RunContext) error {
				value, err := request(ctx, client, http.MethodPost, "/validation/matrices/"+ctx.Positional("run-id")+"/start", nil)
				if err != nil {
					return err
				}
				return ctx.RenderMutation(cliapp.MutationReport{Result: []string{"Validation matrix started", matrixID(value)}})
			}},
			{Name: "wait", Description: "Wait for a validation matrix", Args: idArgs(), RunCtx: func(ctx cliapp.RunContext) error {
				value, err := request(ctx, client, http.MethodGet, "/validation/matrices/"+ctx.Positional("run-id")+"/wait", nil)
				if err != nil {
					return err
				}
				return ctx.RenderList(cliapp.ListReport{Summary: []string{"Validation matrix result", matrixID(value)}, Results: []string{matrixState(value)}, ResultsHeading: "State"})
			}},
			{Name: "show", Description: "Show a validation matrix", Args: idArgs(), RunCtx: func(ctx cliapp.RunContext) error {
				value, err := request(ctx, client, http.MethodGet, "/validation/matrices/"+ctx.Positional("run-id"), nil)
				if err != nil {
					return err
				}
				return ctx.RenderList(cliapp.ListReport{Summary: []string{"Validation matrix", matrixID(value)}, Results: []string{matrixState(value)}, ResultsHeading: "State"})
			}},
		},
	}
}

func idArgs() cliapp.ArgSchema {
	return cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "run-id", Required: true, Description: "Matrix run identifier"}}}
}

func request(_ cliapp.OperationContext, client interface {
	DoWithContext(context.Context, string, string, url.Values, interface{}) ([]byte, error)
}, method, path string, body any) (*structpb.Struct, error) {
	raw, err := client.DoWithContext(context.Background(), method, path, nil, body)
	if err != nil {
		return nil, err
	}
	value := &structpb.Struct{}
	if err := protojson.Unmarshal(raw, value); err != nil {
		return nil, fmt.Errorf("decode matrix response: %w", err)
	}
	return value, nil
}

func matrixID(value *structpb.Struct) string {
	if value == nil {
		return ""
	}
	if id, ok := value.AsMap()["run_id"].(string); ok && id != "" {
		return "run_id=" + id
	}
	return "run_id unavailable"
}

func matrixState(value *structpb.Struct) string {
	if value == nil {
		return "state unavailable"
	}
	if state, ok := value.AsMap()["state"].(string); ok && state != "" {
		return state
	}
	return "state unavailable"
}
