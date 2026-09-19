package generators

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/generators"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/generators/generators_v1connect"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewGeneratorsServiceClient(httpClient, baseURL)

	author := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*v1.AuthorGeneratorResponse, error) {
		resp, err := client.AuthorGenerator(context.Background(), connect.NewRequest(&v1.AuthorGeneratorRequest{
			Id:    strings.TrimSpace(ctx.Flag("id")),
			Brief: ctx.Flag("brief"),
			Store: truthy(ctx.Flag("store")),
		}))
		if err != nil {
			return nil, cliapp.WrapAPIError("author generator", err, nil)
		}
		return resp.Msg, nil
	}, func(_ cliapp.OperationContext, msg *v1.AuthorGeneratorResponse) cliapp.MutationReport {
		g := msg.GetGenerator()
		report := g.GetValidation()
		lines := []string{fmt.Sprintf("Generator %q authored by %s.", g.GetId(), modelOrUnknown(g.GetModelId()))}
		// Every check is printed, passed and failed. A verdict that showed only
		// its failures would make a generator that passed look unexamined, and
		// the whole point of storing the report is that someone can read what
		// was actually tested.
		for _, check := range report.GetChecks() {
			mark := "ok  "
			if !check.GetPassed() {
				mark = "FAIL"
			}
			lines = append(lines, fmt.Sprintf("  %s %-22s %s", mark, check.GetName(), check.GetDetail()))
		}
		switch {
		case !report.GetPassed():
			lines = append(lines, "Refused; nothing was stored. Re-ask with a brief that addresses the failures above.")
		case msg.GetStored():
			lines = append(lines, fmt.Sprintf("Stored. Bind a style's scaffold preset to %q to render it.", g.GetId()))
		default:
			lines = append(lines, "Passed validation. Re-run with --store to keep it.")
		}
		return cliapp.MutationReport{Result: lines}
	})

	list := cliapp.ProtoList(func(cliapp.OperationContext) (*v1.ListGeneratorsResponse, error) {
		resp, err := client.ListGenerators(context.Background(), connect.NewRequest(&v1.ListGeneratorsRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list generators", err, nil)
		}
		return resp.Msg, nil
	}, func(_ cliapp.OperationContext, msg *v1.ListGeneratorsResponse) cliapp.ListReport {
		rows := make([]string, 0, len(msg.GetGenerators()))
		for _, g := range msg.GetGenerators() {
			names := make([]string, 0, len(g.GetParams()))
			for _, p := range g.GetParams() {
				names = append(names, fmt.Sprintf("%s[%g..%g]", p.GetName(), p.GetMin(), p.GetMax()))
			}
			rows = append(rows, fmt.Sprintf("%s %q authored-by=%s params=%s inks=%v",
				g.GetId(), g.GetName(), modelOrUnknown(g.GetModelId()), strings.Join(names, ","), g.GetInks()))
		}
		return cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("%d stored generator(s).", len(rows))},
			ResultsHeading: "Generators",
			Results:        rows,
		}
	})

	remove := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*v1.DeleteGeneratorResponse, error) {
		resp, err := client.DeleteGenerator(context.Background(), connect.NewRequest(&v1.DeleteGeneratorRequest{
			Id: ctx.Positional("id"),
		}))
		if err != nil {
			return nil, cliapp.WrapAPIError("delete generator", err, nil)
		}
		return resp.Msg, nil
	}, func(ctx cliapp.OperationContext, _ *v1.DeleteGeneratorResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{
			fmt.Sprintf("Deleted %q. A style still bound to it now fails by name at its next render.", ctx.Positional("id")),
		}}
	})

	group, err := cliapp.LoadFromManifestPrimitives(manifest, "generators", map[string]cliapp.PrimitiveHandler{
		"GeneratorsService.AuthorGenerator": author,
		"GeneratorsService.ListGenerators":  list,
		"GeneratorsService.DeleteGenerator": remove,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("generators: load manifest: %w", err)
	}
	return group, nil
}

// truthy reads a boolean flag. The manifest surfaces flags as strings, and an
// unset flag is the empty string — so an absent --store previews rather than
// stores, which is the safe default for a call that spends money.
func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "t", "true", "y", "yes":
		return true
	default:
		return false
	}
}

// modelOrUnknown keeps an unnamed model visible rather than printing an empty
// column. A generator whose authoring model went unrecorded cannot be disclosed
// and the operator should see that, not a blank.
func modelOrUnknown(model string) string {
	if strings.TrimSpace(model) == "" {
		return "(unrecorded)"
	}
	return model
}
