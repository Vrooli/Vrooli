package looks

import (
	"context"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
	looksconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks/looks_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client looksconnect.LooksServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: looksconnect.NewLooksServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListLooks(context.Background(), connect.NewRequest(&looksv1.ListLooksRequest{Kind: parseKind(ctx.Flag("kind"))}))
	if err != nil {
		return cliapp.WrapAPIError("list looks", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetLooks()))
	for _, l := range resp.Msg.GetLooks() {
		results = append(results, formatLook(l))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d Look(s).", len(resp.Msg.GetLooks()))},
		ResultsHeading: "Looks",
		Results:        results,
		RetrievalHints: []string{
			"`looks get <id>` — show one Look in detail",
			"`looks compile <id> --subject \"…\"` — resolve a Look into request shapes",
			"`looks render-preview <id>` — render its thumbnail",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetLook(context.Background(), connect.NewRequest(&looksv1.GetLookRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get look %q", id), err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Look %s.", resp.Msg.GetLook().GetId())},
		ResultsHeading: "Look",
		Results:        []string{formatLookDetail(resp.Msg.GetLook())},
	})
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	look, err := readLookFile(ctx.Flag("file"))
	if err != nil {
		return err
	}
	resp, err := h.client.CreateLook(context.Background(), connect.NewRequest(&looksv1.CreateLookRequest{Look: look}))
	if err != nil {
		return cliapp.WrapAPIError("create look", err, nil)
	}
	return ctx.RenderMutation(cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created Look %s", resp.Msg.GetLook().GetId())},
		Changes: []string{formatLookDetail(resp.Msg.GetLook())},
	})
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	look, err := readLookFile(ctx.Flag("file"))
	if err != nil {
		return err
	}
	if look.GetId() == "" {
		return fmt.Errorf("the Look file must include an \"id\" to update")
	}
	resp, err := h.client.UpdateLook(context.Background(), connect.NewRequest(&looksv1.UpdateLookRequest{Look: look}))
	if err != nil {
		return cliapp.WrapAPIError("update look", err, nil)
	}
	return ctx.RenderMutation(cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated Look %s", resp.Msg.GetLook().GetId())},
		Changes: []string{formatLookDetail(resp.Msg.GetLook())},
	})
}

func (h *handlers) del(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	_, err := h.client.DeleteLook(context.Background(), connect.NewRequest(&looksv1.DeleteLookRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("delete look %q", id), err, nil)
	}
	return ctx.RenderMutation(cliapp.MutationReport{Result: []string{fmt.Sprintf("Deleted Look %s", id)}})
}

func (h *handlers) compile(ctx cliapp.RunContext) error {
	resp, err := h.client.CompileLook(context.Background(), connect.NewRequest(&looksv1.CompileLookRequest{
		LookId:   ctx.Positional("look_id"),
		Subject:  ctx.Flag("subject"),
		Prompt:   ctx.Flag("prompt"),
		HasInput: ctx.BoolFlag("has-input"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("compile look", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetSteps()))
	for i, s := range resp.Msg.GetSteps() {
		results = append(results, fmt.Sprintf("%d. %-14s [%s] %s", i+1, s.GetOperation(), stepKindName(s.GetKind()), formatParams(s.GetParams())))
	}
	summary := []string{
		fmt.Sprintf("%d step(s); requires_image=%t requires_mask=%t", len(resp.Msg.GetSteps()), resp.Msg.GetRequiresImage(), resp.Msg.GetRequiresMask()),
	}
	if p := resp.Msg.GetPrimaryPrompt(); p != "" {
		summary = append(summary, "prompt: "+p)
	}
	for _, w := range resp.Msg.GetWarnings() {
		summary = append(summary, "warning: "+w)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Compiled steps",
		Results:        results,
	})
}

func (h *handlers) renderPreview(ctx cliapp.RunContext) error {
	id := ctx.Positional("look_id")
	resp, err := h.client.RenderPreview(context.Background(), connect.NewRequest(&looksv1.RenderPreviewRequest{LookId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("render preview %q", id), err, nil)
	}
	changes := []string{fmt.Sprintf("thumbnail: %s", resp.Msg.GetThumbnailRef())}
	if d := resp.Msg.GetDeferredSteps(); len(d) > 0 {
		changes = append(changes, "deferred (need a backend): "+strings.Join(d, ", "))
	}
	return ctx.RenderMutation(cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Rendered preview for %s", id)},
		Changes: changes,
	})
}

// readLookFile loads a Look from a protojson file (the create/update input).
func readLookFile(path string) (*looksv1.Look, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("--file is required (a JSON file describing the Look)")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read look file %q: %w", path, err)
	}
	look := &looksv1.Look{}
	if err := protojson.Unmarshal(raw, look); err != nil {
		return nil, fmt.Errorf("parse look file %q: %w", path, err)
	}
	return look, nil
}

// parseKind maps a friendly --kind token to the proto enum (unknown → unspecified,
// i.e. no filter).
func parseKind(s string) looksv1.LookKind {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "style":
		return looksv1.LookKind_LOOK_KIND_STYLE
	case "film":
		return looksv1.LookKind_LOOK_KIND_FILM
	case "camera":
		return looksv1.LookKind_LOOK_KIND_CAMERA
	case "enhance":
		return looksv1.LookKind_LOOK_KIND_ENHANCE
	case "custom":
		return looksv1.LookKind_LOOK_KIND_CUSTOM
	default:
		return looksv1.LookKind_LOOK_KIND_UNSPECIFIED
	}
}
