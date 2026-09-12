package sketch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	sketchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch/sketch_v1connect"
	"google.golang.org/protobuf/proto"
)

type handlers struct {
	client          sketchconnect.SketchServiceClient
	inferenceClient sketchconnect.SketchServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 2*time.Minute)
	inferenceHTTP, inferenceURL := cliapp.NewConnectHTTPClientWithTimeout(core, 4*time.Minute)
	return &handlers{client: sketchconnect.NewSketchServiceClient(httpClient, baseURL), inferenceClient: sketchconnect.NewSketchServiceClient(inferenceHTTP, inferenceURL)}
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	resp, err := h.client.GetSketch(context.Background(), connect.NewRequest(&sketchv1.GetSketchRequest{Target: target}))
	if err != nil {
		return cliapp.WrapAPIError("get sketch", err, nil)
	}
	return render(ctx, resp.Msg, []string{fmt.Sprintf("Sketch: %s/%s", target.Scenario, target.Page), fmt.Sprintf("Document: %s", resp.Msg.GetDocumentPath())})
}

func (h *handlers) set(ctx cliapp.RunContext) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	get, err := h.client.GetSketch(context.Background(), connect.NewRequest(&sketchv1.GetSketchRequest{Target: target}))
	if err != nil {
		return cliapp.WrapAPIError("read sketch before set", err, nil)
	}
	doc := get.Msg.GetSketch()
	if doc == nil {
		doc = &sketchv1.Sketch{}
	}
	if ctx.FlagProvided("template") {
		doc.Template = &sketchv1.AssetReference{Asset: ctx.Flag("template"), Version: ctx.Flag("version")}
	}
	if ctx.FlagProvided("viewport") {
		doc.Viewport = ctx.Flag("viewport")
	}
	resp, err := h.client.PutSketch(context.Background(), connect.NewRequest(&sketchv1.PutSketchRequest{Target: target, Sketch: doc, ExpectedContentHash: get.Msg.GetContentHash()}))
	if err != nil {
		return cliapp.WrapAPIError("set sketch", err, nil)
	}
	return render(ctx, resp.Msg, []string{fmt.Sprintf("Sketch written: %s/%s", target.Scenario, target.Page), fmt.Sprintf("Document: %s", resp.Msg.GetDocumentPath())})
}

func (h *handlers) verify(ctx cliapp.RunContext) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	resp, err := h.client.VerifySketch(context.Background(), connect.NewRequest(&sketchv1.VerifySketchRequest{Target: target}))
	if err != nil {
		return cliapp.WrapAPIError("verify sketch", err, nil)
	}
	if err := render(ctx, resp.Msg, []string{fmt.Sprintf("Sketch verification: %s/%s", target.Scenario, target.Page), fmt.Sprintf("Proven source bindings: %d/%d (%d custom)", resp.Msg.GetCoverage().GetResolved(), resp.Msg.GetCoverage().GetTotal(), resp.Msg.GetCoverage().GetResolvedLocal()), fmt.Sprintf("Built coverage: %.2f%%", resp.Msg.GetCoverage().GetBuiltPercent()), fmt.Sprintf("Declared regions: %d library-backed, %d local", resp.Msg.GetCoverage().GetLibraryBacked(), resp.Msg.GetCoverage().GetLocal()), fmt.Sprintf("Passes: %t", resp.Msg.GetPasses())}); err != nil {
		return err
	}
	if !resp.Msg.GetPasses() {
		return exitError{message: "sketch verification found drifted or required missing regions", code: 2}
	}
	return nil
}

func (h *handlers) importPage(ctx cliapp.RunContext) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return err
	}
	resp, err := h.client.ImportPage(context.Background(), connect.NewRequest(&sketchv1.ImportPageRequest{Target: target, ExpectedContentHash: expected, Write: ctx.BoolFlag("write")}))
	if err != nil {
		return cliapp.WrapAPIError("import page", err, nil)
	}
	return render(ctx, resp.Msg, []string{fmt.Sprintf("Imported: %s/%s", target.Scenario, target.Page), fmt.Sprintf("Items: %d; templates: %d; written: %t", len(resp.Msg.GetItems()), len(resp.Msg.GetTemplates()), resp.Msg.GetWritten())})
}

func (h *handlers) place(ctx cliapp.RunContext) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return err
	}
	resp, err := h.client.Place(context.Background(), connect.NewRequest(&sketchv1.PlaceRequest{Target: target, ExpectedContentHash: expected, Region: ctx.Positional("region"), Asset: ctx.Positional("asset"), Version: ctx.Flag("version"), Note: ctx.Flag("note")}))
	if err != nil {
		return cliapp.WrapAPIError("place sketch asset", err, nil)
	}
	return render(ctx, resp.Msg, []string{fmt.Sprintf("Placed %s in %s", ctx.Positional("asset"), ctx.Positional("region"))})
}

func (h *handlers) placeholder(ctx cliapp.RunContext) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return err
	}
	resp, err := h.client.Placeholder(context.Background(), connect.NewRequest(&sketchv1.PlaceholderRequest{Target: target, ExpectedContentHash: expected, Region: ctx.Positional("region"), Placeholder: ctx.Positional("placeholder"), Intent: ctx.Flag("intent"), Note: ctx.Flag("note")}))
	if err != nil {
		return cliapp.WrapAPIError("place sketch placeholder", err, nil)
	}
	return render(ctx, resp.Msg, []string{fmt.Sprintf("Placeholder %s recorded in %s", ctx.Positional("placeholder"), ctx.Positional("region"))})
}

func (h *handlers) note(ctx cliapp.RunContext) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return err
	}
	resp, err := h.client.AddNote(context.Background(), connect.NewRequest(&sketchv1.AddNoteRequest{Target: target, ExpectedContentHash: expected, Scope: ctx.Flag("scope"), Text: ctx.Flag("text")}))
	if err != nil {
		return cliapp.WrapAPIError("add sketch note", err, nil)
	}
	return render(ctx, resp.Msg, []string{"Sketch note recorded."})
}

func (h *handlers) unplace(ctx cliapp.RunContext) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return err
	}
	resp, err := h.client.Unplace(context.Background(), connect.NewRequest(&sketchv1.UnplaceRequest{Target: target, ExpectedContentHash: expected, Region: ctx.Positional("region"), Reason: ctx.Flag("reason")}))
	if err != nil {
		return cliapp.WrapAPIError("unplace sketch region", err, nil)
	}
	return render(ctx, resp.Msg, []string{fmt.Sprintf("Region %s moved to unplaced.", ctx.Positional("region"))})
}

func (h *handlers) setTemplate(ctx cliapp.RunContext) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return err
	}
	remaps := make([]*sketchv1.RegionRemap, 0, len(ctx.FlagValues("remap")))
	for _, value := range ctx.FlagValues("remap") {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid --remap %q: expected old=new", value)
		}
		remaps = append(remaps, &sketchv1.RegionRemap{From: parts[0], To: parts[1]})
	}
	resp, err := h.client.SetTemplate(context.Background(), connect.NewRequest(&sketchv1.SetTemplateRequest{Target: target, ExpectedContentHash: expected, Asset: ctx.Positional("asset"), Version: ctx.Flag("version"), Remap: remaps, ConfirmUnplaced: ctx.BoolFlag("confirm-unplaced"), Preview: ctx.BoolFlag("preview")}))
	if err != nil {
		return cliapp.WrapAPIError("set sketch template", err, nil)
	}
	status := "Template unchanged"
	if resp.Msg.GetChanged() {
		status = "Template updated"
	} else if ctx.BoolFlag("preview") {
		status = "Template preview"
	}
	if resp.Msg.GetRequiresConfirmation() {
		status = "Template preview: confirm displaced placements before applying"
	}
	return render(ctx, resp.Msg, []string{status, fmt.Sprintf("Newly unplaced: %d", len(resp.Msg.GetNewlyUnplaced()))})
}

func (h *handlers) brief(ctx cliapp.RunContext) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	resp, err := h.client.BuildBrief(context.Background(), connect.NewRequest(&sketchv1.BuildBriefRequest{Target: target}))
	if err != nil {
		return cliapp.WrapAPIError("build sketch brief", err, nil)
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	_, err = fmt.Fprintln(ctx.Stdout(), resp.Msg.GetMarkdown())
	return err
}

func parseTarget(value string) (*sketchv1.SketchTarget, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid target %q: expected <scenario>/<page>", value)
	}
	return &sketchv1.SketchTarget{Scenario: parts[0], Page: parts[1]}, nil
}

func render(ctx cliapp.RunContext, message proto.Message, lines []string) error {
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), message)
	}
	return ctx.RenderList(cliapp.ListReport{Summary: lines})
}

type exitError struct {
	message string
	code    int
}

func (e exitError) Error() string { return e.message }
func (e exitError) ExitCode() int { return e.code }

var _ error = exitError{}

func (h *handlers) currentHash(target *sketchv1.SketchTarget) (string, error) {
	response, err := h.client.GetSketch(context.Background(), connect.NewRequest(&sketchv1.GetSketchRequest{Target: target}))
	if err != nil {
		return "", cliapp.WrapAPIError("read current sketch revision", err, nil)
	}
	return response.Msg.GetContentHash(), nil
}

func (h *handlers) history(ctx cliapp.RunContext) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	response, err := h.client.GetHistory(context.Background(), connect.NewRequest(&sketchv1.GetHistoryRequest{Target: target}))
	if err != nil {
		return cliapp.WrapAPIError("read design history", err, nil)
	}
	lines := []string{fmt.Sprintf("Design history: %s/%s", target.Scenario, target.Page)}
	for _, revision := range response.Msg.GetRevisions() {
		lines = append(lines, fmt.Sprintf("%s · %s · current: %t", revision.GetContentHash(), revision.GetCreatedAt(), revision.GetCurrent()))
	}
	return render(ctx, response.Msg, lines)
}
func (h *handlers) recover(ctx cliapp.RunContext) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return err
	}
	response, err := h.client.RecoverSketch(context.Background(), connect.NewRequest(&sketchv1.RecoverSketchRequest{Target: target, ExpectedContentHash: expected}))
	if err != nil {
		return cliapp.WrapAPIError("recover interrupted design apply", err, nil)
	}
	return render(ctx, response.Msg, []string{fmt.Sprintf("Revision: %s; recovered: %t", response.Msg.GetContentHash(), response.Msg.GetChanged())})
}
