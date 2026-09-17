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
	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"scenario-to-desktop/cli/internal/support"
)

// Register exposes the provider-neutral validation matrix HTTP contract. The
// matrix service is intentionally REST-shaped today, so these commands stay
// thin and return the server's structured result without duplicating its state.
func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	app := deps.ScenarioApp()
	client := cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: app.APIBaseOptions(), Timeout: app.HTTPClient.Timeout()})
	prefix := app.APIPrefix()
	return cliapp.SubcommandGroup{
		Name:        "matrix",
		Description: "Create and inspect desktop validation matrices",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "create", Description: "Create a validation matrix", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}, Flags: []cliapp.Flag{{Name: "selection-file", Description: "JSON MatrixSelection file"}, {Name: "artifact-digest", Description: "Artifact digest when building a selection from flags"}, {Name: "artifact-path", Description: "Artifact path when building a selection from flags"}, {Name: "platform", Description: "Target platform or OS (for example mac or windows)"}, {Name: "journey", Description: "Journey id"}, {Name: "profile", Default: "normal", Description: "Environment profile"}}}, RunCtx: func(ctx cliapp.RunContext) error {
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
				if value := strings.TrimSpace(ctx.Flag("artifact-digest")); value != "" {
					body["artifact_digest"] = value
				}
				if value := strings.TrimSpace(ctx.Flag("artifact-path")); value != "" {
					body["artifact_path"] = value
				}
				if value := strings.TrimSpace(ctx.Flag("journey")); value != "" {
					body["journeys"] = []any{map[string]any{"journey_id": value, "display_name": value, "required": true}}
				}
				if value := strings.TrimSpace(ctx.Flag("profile")); value != "" {
					profile, err := profileValue(value)
					if err != nil {
						return err
					}
					body["environment_profiles"] = []any{profile}
				}
				if platform := strings.TrimSpace(ctx.Flag("platform")); platform != "" {
					target, err := findTarget(ctx, client, prefix, platform)
					if err != nil {
						return err
					}
					body["targets"] = []any{target}
				}
				value, err := request(ctx, client, prefix, http.MethodPost, "/validation/matrices", body)
				if err != nil {
					return err
				}
				return ctx.RenderMutation(cliapp.MutationReport{Result: []string{"Validation matrix created", matrixID(value)}})
			}},
			{Name: "start", Description: "Start a validation matrix", Args: idArgs(), RunCtx: func(ctx cliapp.RunContext) error {
				value, err := request(ctx, client, prefix, http.MethodPost, "/validation/matrices/"+ctx.Positional("run-id")+"/start", nil)
				if err != nil {
					return err
				}
				return ctx.RenderMutation(cliapp.MutationReport{Result: []string{"Validation matrix started", matrixID(value)}})
			}},
			{Name: "wait", Description: "Wait for a validation matrix", Args: idArgs(), RunCtx: func(ctx cliapp.RunContext) error {
				value, err := request(ctx, client, prefix, http.MethodGet, "/validation/matrices/"+ctx.Positional("run-id")+"/wait", nil)
				if err != nil {
					return err
				}
				return ctx.RenderList(cliapp.ListReport{Summary: []string{"Validation matrix result", matrixID(value)}, Results: []string{matrixState(value)}, ResultsHeading: "State"})
			}},
			{Name: "show", Description: "Show a validation matrix", Args: idArgs(), RunCtx: func(ctx cliapp.RunContext) error {
				value, err := request(ctx, client, prefix, http.MethodGet, "/validation/matrices/"+ctx.Positional("run-id"), nil)
				if err != nil {
					return err
				}
				return ctx.RenderList(cliapp.ListReport{Summary: []string{"Validation matrix", matrixID(value)}, Results: []string{matrixState(value)}, ResultsHeading: "State"})
			}},
			{Name: "list", Description: "List validation matrices", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "json", Bool: true}}}, RunCtx: func(ctx cliapp.RunContext) error {
				raw, err := client.DoWithContext(context.Background(), http.MethodGet, strings.TrimRight(prefix, "/")+"/validation/matrices", nil, nil)
				if err != nil {
					return err
				}
				if ctx.JSON() {
					_, err = ctx.Stdout().Write(append(raw, '\n'))
					return err
				}
				return renderMatrixList(ctx, raw)
			}},
			{Name: "compare", Description: "Compare two validation matrices", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "run-id", Required: true}, {Name: "prior-run-id", Required: true}}, Flags: []cliapp.Flag{{Name: "json", Bool: true}}}, RunCtx: func(ctx cliapp.RunContext) error {
				path := "/validation/matrices/" + url.PathEscape(ctx.Positional("run-id")) + "/compare/" + url.PathEscape(ctx.Positional("prior-run-id"))
				value, err := request(ctx, client, prefix, http.MethodGet, path, nil)
				if err != nil {
					return err
				}
				if ctx.JSON() {
					_, err = ctx.Stdout().Write(append(mustJSON(value), '\n'))
					return err
				}
				return ctx.RenderList(cliapp.ListReport{Summary: []string{"Validation matrix comparison"}, Results: []string{string(mustJSON(value))}, ResultsHeading: "Comparison"})
			}},
		},
	}
}

func renderMatrixList(ctx cliapp.RunContext, raw []byte) error {
	var matrices []struct {
		RunID string `json:"run_id"`
		State string `json:"state"`
	}
	if err := json.Unmarshal(raw, &matrices); err != nil {
		return fmt.Errorf("decode matrix list: %w", err)
	}
	results := make([]string, 0, len(matrices))
	for _, matrix := range matrices {
		results = append(results, matrix.RunID+" "+matrix.State)
	}
	if len(results) == 0 {
		results = []string{"(no validation matrices)"}
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Validation matrices: %d", len(matrices))}, Results: results, ResultsHeading: "Runs"})
}

// ProfileValue converts the agent-facing profile name into the API enum value.
func ProfileValue(raw string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "normal":
		return 1, nil
	case "offline":
		return 2, nil
	case "slow-network", "slow_network":
		return 3, nil
	default:
		return 0, fmt.Errorf("unknown validation profile %q (supported: normal, offline, slow-network)", raw)
	}
}

// FindTarget resolves a target by id, display name, or normalized operating system.
func FindTarget(ctx cliapp.OperationContext, client interface {
	DoWithContext(context.Context, string, string, url.Values, interface{}) ([]byte, error)
}, prefix, selector string) (map[string]any, error) {
	value, err := request(ctx, client, prefix, http.MethodGet, "/validation/targets", nil)
	if err != nil {
		return nil, fmt.Errorf("resolve validation target: %w", err)
	}
	for _, raw := range value.AsMap()["targets"].([]any) {
		target, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		descriptor, ok := target["descriptor"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("validation target %q has no descriptor", selector)
		}
		if targetSelectorMatches(selector, target, descriptor) {
			return map[string]any{"descriptor": descriptor, "kind": target["kind"]}, nil
		}
	}
	if isPlatformSelector(selector) {
		platform := normalizeTargetSelector(selector)
		return map[string]any{
			"descriptor": map[string]any{
				"target_id":    "unavailable-" + platform,
				"display_name": platform + " target",
				"available":    false,
				"reason":       "no registered target for platform " + platform,
			},
			"kind": "bridge",
		}, nil
	}
	return nil, fmt.Errorf("validation target %q was not found; use `targets list`", selector)
}

// The local command implementation keeps these helpers private to the package.
// Preserve the old names for the matrix command while allowing validation's
// ergonomic runner to use the exact same selector semantics.
func profileValue(raw string) (int, error) { return ProfileValue(raw) }

func findTarget(ctx cliapp.OperationContext, client interface {
	DoWithContext(context.Context, string, string, url.Values, interface{}) ([]byte, error)
}, prefix, selector string) (map[string]any, error) {
	return FindTarget(ctx, client, prefix, selector)
}

func targetSelectorMatches(selector string, target, descriptor map[string]any) bool {
	want := normalizeTargetSelector(selector)
	for _, value := range []any{
		target["target_id"], target["display_name"], target["os"],
		descriptor["target_id"], descriptor["display_name"], descriptor["os"],
	} {
		if normalizeTargetSelector(fmt.Sprint(value)) == want {
			return true
		}
	}
	return false
}

func normalizeTargetSelector(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "mac", "macos", "osx":
		return "darwin"
	case "win", "windows":
		return "windows"
	default:
		return normalized
	}
}

func isPlatformSelector(value string) bool {
	switch normalizeTargetSelector(value) {
	case "darwin", "windows", "linux":
		return true
	default:
		return false
	}
}

func idArgs() cliapp.ArgSchema {
	return cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "run-id", Required: true, Description: "Matrix run identifier"}}}
}

func request(_ cliapp.OperationContext, client interface {
	DoWithContext(context.Context, string, string, url.Values, interface{}) ([]byte, error)
}, prefix, method, path string, body any,
) (*structpb.Struct, error) {
	raw, err := client.DoWithContext(context.Background(), method, strings.TrimRight(prefix, "/")+path, nil, body)
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

func mustJSON(value *structpb.Struct) []byte {
	raw, _ := protojson.Marshal(value)
	return raw
}
