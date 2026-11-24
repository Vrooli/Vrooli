package sketch

import (
	"context"
	"fmt"
	"strings"
	"time"

	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	experienceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/experience/experience_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	workspaceassistconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspaceassist/workspaceassist_v1connect"
	workspacecandidateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacecandidate/workspacecandidate_v1connect"
)

type handlers struct {
	assistClient     workspaceassistconnect.WorkspaceAssistServiceClient
	inferenceClient  workspaceassistconnect.WorkspaceAssistServiceClient
	candidateClient  workspacecandidateconnect.WorkspaceCandidateServiceClient
	experienceClient experienceconnect.ExperienceServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 2*time.Minute)
	inferenceHTTP, inferenceURL := cliapp.NewConnectHTTPClientWithTimeout(core, 4*time.Minute)
	return &handlers{
		assistClient:     workspaceassistconnect.NewWorkspaceAssistServiceClient(httpClient, baseURL),
		inferenceClient:  workspaceassistconnect.NewWorkspaceAssistServiceClient(inferenceHTTP, inferenceURL),
		candidateClient:  workspacecandidateconnect.NewWorkspaceCandidateServiceClient(httpClient, baseURL),
		experienceClient: experienceconnect.NewExperienceServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*sketchv1.GetSketchResponse, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	resp, err := h.experienceClient.GetSketch(context.Background(), connect.NewRequest(&sketchv1.GetSketchRequest{Target: target}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get sketch", err, nil)
	}
	return resp.Msg, nil
}

func getHuman(ctx cliapp.RunContext, response *sketchv1.GetSketchResponse) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Sketch: %s/%s", target.Scenario, target.Page), fmt.Sprintf("Document: %s", response.GetDocumentPath())}})
}

func (h *handlers) setCall(ctx cliapp.OperationContext) (*sketchv1.PutSketchResponse, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	get, err := h.experienceClient.GetSketch(context.Background(), connect.NewRequest(&sketchv1.GetSketchRequest{Target: target}))
	if err != nil {
		return nil, cliapp.WrapAPIError("read sketch before set", err, nil)
	}
	doc := get.Msg.GetSketch()
	if doc == nil {
		doc = &sharedv1.Sketch{}
	}
	if ctx.FlagProvided("template") {
		doc.Template = &sharedv1.AssetReference{Asset: ctx.Flag("template"), Version: ctx.Flag("version")}
	}
	if ctx.FlagProvided("viewport") {
		doc.Viewport = ctx.Flag("viewport")
	}
	resp, err := h.experienceClient.PutSketch(context.Background(), connect.NewRequest(&sketchv1.PutSketchRequest{Target: target, Sketch: doc, ExpectedContentHash: get.Msg.GetContentHash()}))
	if err != nil {
		return nil, cliapp.WrapAPIError("set sketch", err, nil)
	}
	return resp.Msg, nil
}

func setHuman(ctx cliapp.RunContext, response *sketchv1.PutSketchResponse) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Sketch written: %s/%s", target.Scenario, target.Page), fmt.Sprintf("Document: %s", response.GetDocumentPath())}})
}

func (h *handlers) verifyCall(ctx cliapp.OperationContext) (*sketchv1.VerifySketchResponse, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	resp, err := h.experienceClient.VerifySketch(context.Background(), connect.NewRequest(&sketchv1.VerifySketchRequest{Target: target}))
	if err != nil {
		return nil, cliapp.WrapAPIError("verify sketch", err, nil)
	}
	return resp.Msg, nil
}

func verifyReport(ctx cliapp.OperationContext, response *sketchv1.VerifySketchResponse) cliapp.ListReport {
	target, _ := parseTarget(ctx.Positional("target"))
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Sketch verification: %s/%s", target.GetScenario(), target.GetPage()), fmt.Sprintf("Proven source bindings: %d/%d (%d custom)", response.GetCoverage().GetResolved(), response.GetCoverage().GetTotal(), response.GetCoverage().GetResolvedLocal()), fmt.Sprintf("Built coverage: %.2f%%", response.GetCoverage().GetBuiltPercent()), fmt.Sprintf("Declared regions: %d library-backed, %d local", response.GetCoverage().GetLibraryBacked(), response.GetCoverage().GetLocal()), fmt.Sprintf("Passes: %t", response.GetPasses())}}
}

func verifyOutcome(response *sketchv1.VerifySketchResponse) error {
	if !response.GetPasses() {
		return exitError{message: "sketch verification found drifted or required missing regions", code: 2}
	}
	return nil
}

func (h *handlers) importPageCall(ctx cliapp.OperationContext) (*sketchv1.ImportPageResponse, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return nil, err
	}
	resp, err := h.experienceClient.ImportPage(context.Background(), connect.NewRequest(&sketchv1.ImportPageRequest{Target: target, ExpectedContentHash: expected, Write: ctx.BoolFlag("write")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("import page", err, nil)
	}
	return resp.Msg, nil
}

func importPageHuman(ctx cliapp.RunContext, response *sketchv1.ImportPageResponse) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Imported: %s/%s", target.Scenario, target.Page), fmt.Sprintf("Items: %d; templates: %d; written: %t", len(response.GetItems()), len(response.GetTemplates()), response.GetWritten())}})
}

func (h *handlers) placeCall(ctx cliapp.OperationContext) (*sketchv1.PutSketchResponse, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return nil, err
	}
	resp, err := h.experienceClient.Place(context.Background(), connect.NewRequest(&sketchv1.PlaceRequest{Target: target, ExpectedContentHash: expected, Region: ctx.Positional("region"), Asset: ctx.Positional("asset"), Version: ctx.Flag("version"), Note: ctx.Flag("note")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("place sketch asset", err, nil)
	}
	return resp.Msg, nil
}

func placeHuman(ctx cliapp.RunContext, response *sketchv1.PutSketchResponse) error {
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Placed %s in %s", ctx.Positional("asset"), ctx.Positional("region"))}})
}

func (h *handlers) placeholderCall(ctx cliapp.OperationContext) (*sketchv1.PutSketchResponse, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return nil, err
	}
	resp, err := h.experienceClient.Placeholder(context.Background(), connect.NewRequest(&sketchv1.PlaceholderRequest{Target: target, ExpectedContentHash: expected, Region: ctx.Positional("region"), Placeholder: ctx.Positional("placeholder"), Intent: ctx.Flag("intent"), Note: ctx.Flag("note")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("place sketch placeholder", err, nil)
	}
	return resp.Msg, nil
}

func placeholderHuman(ctx cliapp.RunContext, response *sketchv1.PutSketchResponse) error {
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Placeholder %s recorded in %s", ctx.Positional("placeholder"), ctx.Positional("region"))}})
}

func (h *handlers) noteCall(ctx cliapp.OperationContext) (*sketchv1.PutSketchResponse, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return nil, err
	}
	resp, err := h.experienceClient.AddNote(context.Background(), connect.NewRequest(&sketchv1.AddNoteRequest{Target: target, ExpectedContentHash: expected, Scope: ctx.Flag("scope"), Text: ctx.Flag("text")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("add sketch note", err, nil)
	}
	return resp.Msg, nil
}

func noteHuman(ctx cliapp.RunContext, _ *sketchv1.PutSketchResponse) error {
	return ctx.RenderList(cliapp.ListReport{Summary: []string{"Sketch note recorded."}})
}

func (h *handlers) unplaceCall(ctx cliapp.OperationContext) (*sketchv1.PutSketchResponse, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return nil, err
	}
	resp, err := h.experienceClient.Unplace(context.Background(), connect.NewRequest(&sketchv1.UnplaceRequest{Target: target, ExpectedContentHash: expected, Region: ctx.Positional("region"), Reason: ctx.Flag("reason")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("unplace sketch region", err, nil)
	}
	return resp.Msg, nil
}

func unplaceHuman(ctx cliapp.RunContext, _ *sketchv1.PutSketchResponse) error {
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Region %s moved to unplaced.", ctx.Positional("region"))}})
}

func (h *handlers) setTemplateCall(ctx cliapp.OperationContext) (*sketchv1.SetTemplateResponse, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return nil, err
	}
	remaps := make([]*sketchv1.RegionRemap, 0, len(ctx.FlagValues("remap")))
	for _, value := range ctx.FlagValues("remap") {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid --remap %q: expected old=new", value)
		}
		remaps = append(remaps, &sketchv1.RegionRemap{From: parts[0], To: parts[1]})
	}
	resp, err := h.experienceClient.SetTemplate(context.Background(), connect.NewRequest(&sketchv1.SetTemplateRequest{Target: target, ExpectedContentHash: expected, Asset: ctx.Positional("asset"), Version: ctx.Flag("version"), Remap: remaps, ConfirmUnplaced: ctx.BoolFlag("confirm-unplaced"), Preview: ctx.BoolFlag("preview")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("set sketch template", err, nil)
	}
	return resp.Msg, nil
}

func setTemplateHuman(ctx cliapp.RunContext, response *sketchv1.SetTemplateResponse) error {
	status := "Template unchanged"
	if response.GetChanged() {
		status = "Template updated"
	} else if ctx.BoolFlag("preview") {
		status = "Template preview"
	}
	if response.GetRequiresConfirmation() {
		status = "Template preview: confirm displaced placements before applying"
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{status, fmt.Sprintf("Newly unplaced: %d", len(response.GetNewlyUnplaced()))}})
}

func (h *handlers) briefCall(ctx cliapp.OperationContext) (*sketchv1.BuildBriefResponse, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	resp, err := h.experienceClient.BuildBrief(context.Background(), connect.NewRequest(&sketchv1.BuildBriefRequest{Target: target}))
	if err != nil {
		return nil, cliapp.WrapAPIError("build sketch brief", err, nil)
	}
	return resp.Msg, nil
}

func briefHuman(ctx cliapp.RunContext, response *sketchv1.BuildBriefResponse) error {
	_, err := fmt.Fprintln(ctx.Stdout(), response.GetMarkdown())
	return err
}

func parseTarget(value string) (*sharedv1.SketchTarget, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid target %q: expected <scenario>/<page>", value)
	}
	return &sharedv1.SketchTarget{Scenario: parts[0], Page: parts[1]}, nil
}

type exitError struct {
	message string
	code    int
}

func (e exitError) Error() string { return e.message }
func (e exitError) ExitCode() int { return e.code }

var _ error = exitError{}

func (h *handlers) currentHash(target *sharedv1.SketchTarget) (string, error) {
	response, err := h.experienceClient.GetSketch(context.Background(), connect.NewRequest(&sketchv1.GetSketchRequest{Target: target}))
	if err != nil {
		return "", cliapp.WrapAPIError("read current sketch revision", err, nil)
	}
	return response.Msg.GetContentHash(), nil
}

func (h *handlers) historyCall(ctx cliapp.OperationContext) (*sketchv1.GetHistoryResponse, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	response, err := h.experienceClient.GetHistory(context.Background(), connect.NewRequest(&sketchv1.GetHistoryRequest{Target: target}))
	if err != nil {
		return nil, cliapp.WrapAPIError("read design history", err, nil)
	}
	return response.Msg, nil
}

func historyHuman(ctx cliapp.RunContext, response *sketchv1.GetHistoryResponse) error {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return err
	}
	lines := []string{fmt.Sprintf("Design history: %s/%s", target.Scenario, target.Page)}
	for _, revision := range response.GetRevisions() {
		lines = append(lines, fmt.Sprintf("%s · %s · current: %t", revision.GetContentHash(), revision.GetCreatedAt(), revision.GetCurrent()))
	}
	return ctx.RenderList(cliapp.ListReport{Summary: lines})
}

func (h *handlers) recoverCall(ctx cliapp.OperationContext) (*sketchv1.PutSketchResponse, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	expected, err := h.currentHash(target)
	if err != nil {
		return nil, err
	}
	response, err := h.experienceClient.RecoverSketch(context.Background(), connect.NewRequest(&sketchv1.RecoverSketchRequest{Target: target, ExpectedContentHash: expected}))
	if err != nil {
		return nil, cliapp.WrapAPIError("recover interrupted design apply", err, nil)
	}
	return response.Msg, nil
}

func recoverHuman(ctx cliapp.RunContext, response *sketchv1.PutSketchResponse) error {
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Revision: %s; recovered: %t", response.GetContentHash(), response.GetChanged())}})
}
