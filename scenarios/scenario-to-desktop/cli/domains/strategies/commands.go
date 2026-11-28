package strategies

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type Commands struct {
	apiPrefix string
	http      interface {
		DoWithContext(context.Context, string, string, url.Values, interface{}) ([]byte, error)
	}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	app := deps.ScenarioApp()
	c := &Commands{apiPrefix: app.APIPrefix(), http: cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: app.APIBaseOptions(), Timeout: app.HTTPClient.Timeout()})}
	return cliapp.SubcommandGroup{Name: "strategies", Description: "List and explain registered packaging strategies", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "list", Description: "List registered packaging strategies as JSON"}).WithPrimitive(c.listPrimitive()),
		(cliapp.Command{Name: "explain", Description: "Explain eligibility and selection for a scenario", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}, Flags: []cliapp.Flag{{Name: "strategy", Description: "Explicit strategy override"}, {Name: "platform", Description: "Target platform (default linux-x64)"}}}}).WithPrimitive(c.explainPrimitive()),
	}}
}

func (c *Commands) listPrimitive() cliapp.PrimitiveHandler {
	return c.get("list strategies", "/strategies", nil)
}

func (c *Commands) explainPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*structpb.Struct, error) {
		path := "/strategies/" + url.PathEscape(strings.TrimSpace(ctx.Positional("scenario"))) + "/explain"
		query := url.Values{}
		if value := strings.TrimSpace(ctx.Flag("strategy")); value != "" {
			query.Set("strategy", value)
		}
		if value := strings.TrimSpace(ctx.Flag("platform")); value != "" {
			query.Set("platform", value)
		}
		return c.fetch(ctx, path, query)
	}, func(_ cliapp.OperationContext, result *structpb.Struct) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Packaging strategy selection"}, Results: []string{fmt.Sprintf("Selected: %v", result.AsMap()["selected"])}}
	})
}

func (c *Commands) get(operation, path string, query url.Values) cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*structpb.Struct, error) { return c.fetch(ctx, path, query) }, func(_ cliapp.OperationContext, result *structpb.Struct) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{operation}, Results: []string{fmt.Sprintf("Default: %v", result.AsMap()["default_strategy"])}}
	})
}

func (c *Commands) fetch(ctx cliapp.OperationContext, path string, query url.Values) (*structpb.Struct, error) {
	if c.http == nil {
		return nil, fmt.Errorf("scenario API HTTP client is unavailable")
	}
	raw, err := c.http.DoWithContext(context.Background(), "GET", strings.TrimRight(c.apiPrefix, "/")+path, query, nil)
	if err != nil {
		return nil, cliapp.WrapAPIError("read packaging strategies", err, nil)
	}
	result := &structpb.Struct{}
	if err := protojson.Unmarshal(raw, result); err != nil {
		return nil, fmt.Errorf("decode strategy response: %w", err)
	}
	return result, nil
}
