// Package targets exposes the provider-neutral validation target inventory.
package targets

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

type httpClient interface {
	DoWithContext(context.Context, string, string, url.Values, interface{}) ([]byte, error)
}

type Commands struct {
	client httpClient
	prefix string
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	app := deps.ScenarioApp()
	client := cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: app.APIBaseOptions(), Timeout: app.HTTPClient.Timeout()})
	c := &Commands{client: client, prefix: app.APIPrefix()}
	return cliapp.SubcommandGroup{Name: "targets", Description: "Inspect desktop validation targets", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "list", Description: "List validation targets", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "json", Bool: true, Description: "Emit the endpoint response as JSON"}}}, RunCtx: c.list},
	}}
}

func (c *Commands) list(ctx cliapp.RunContext) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("scenario API HTTP client is unavailable")
	}
	raw, err := c.client.DoWithContext(context.Background(), "GET", strings.TrimRight(c.prefix, "/")+"/validation/targets", nil, nil)
	if err != nil {
		return err
	}
	if ctx.JSON() {
		_, err = ctx.Stdout().Write(append(raw, '\n'))
		return err
	}
	value := &structpb.Struct{}
	if err := protojson.Unmarshal(raw, value); err != nil {
		return fmt.Errorf("decode validation targets: %w", err)
	}
	human := humanSummary(value)
	if len(human) == 0 {
		return ctx.RenderList(cliapp.ListReport{Summary: []string{"Validation targets"}, ResultsHeading: "Targets"})
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{human[0]}, Results: human[1:], ResultsHeading: "Targets"})
}

func mustJSON(value *structpb.Struct) []byte {
	raw, _ := protojson.Marshal(value)
	return raw
}

func humanSummary(value *structpb.Struct) []string {
	if value == nil {
		return []string{"Validation targets unavailable"}
	}
	items, _ := value.AsMap()["targets"].([]any)
	if len(items) == 0 {
		return []string{"No validation targets discovered"}
	}
	result := []string{fmt.Sprintf("Validation targets: %d", len(items))}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		descriptor, _ := item["descriptor"].(map[string]any)
		id := stringValue(descriptor["target_id"])
		name := stringValue(descriptor["display_name"])
		available := descriptor["available"] == true
		platform := stringValue(item["os"])
		arch := stringValue(item["architecture"])
		reason := stringValue(item["reason"])
		if reason == "" {
			reason = stringValue(descriptor["reason"])
		}
		result = append(result, fmt.Sprintf("%s (%s) platform=%s/%s available=%t reason=%s", firstNonEmpty(name, id), id, platform, arch, available, firstNonEmpty(reason, "ready")))
		if !available {
			result = append(result, "  missing capability or next action: "+firstNonEmpty(stringValue(descriptor["missing_capability"]), "inspect target reason and remediate host capability"))
		}
	}
	return result
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
