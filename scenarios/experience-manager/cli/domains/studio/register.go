// Package studio owns the authoring-session CLI surface.
package studio

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
	contractconnect "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract/contract_v1connect"
	"google.golang.org/protobuf/proto"
)

const GroupName = "author"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"StudioSessionService.StartAuthoringSession": h.start,
		"StudioSessionService.SubmitPage":            h.submit,
		"StudioSessionService.PreviewSession":        h.preview,
		"StudioSessionService.ApplySession":          h.apply,
		"StudioSessionService.DiscardSession":        h.discard,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("studio: load from manifest: %w", err)
	}
	return group, nil
}

type handlers struct {
	client contractconnect.StudioSessionServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: contractconnect.NewStudioSessionServiceClient(httpClient, baseURL)}
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	resp, err := h.client.StartAuthoringSession(context.Background(), connect.NewRequest(&contractv1.StartAuthoringSessionRequest{
		Scenario: ctx.Positional("scenario"),
		Path:     ctx.Flag("path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("start authoring session", err, nil)
	}
	session := resp.Msg.GetSession()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{fmt.Sprintf("session %s for %s", session.GetId(), session.GetScenario())},
		Results: []string{
			"status: " + session.GetStatus(),
			"target: " + session.GetTargetPath(),
		},
	})
}

func (h *handlers) submit(ctx cliapp.RunContext) error {
	page, err := pageFormFromFlags(ctx)
	if err != nil {
		return err
	}
	resp, err := h.client.SubmitPage(context.Background(), connect.NewRequest(&contractv1.SubmitPageRequest{
		SessionId: ctx.Positional("session"),
		Page:      page,
	}))
	if err != nil {
		return cliapp.WrapAPIError("submit page draft", err, nil)
	}
	draft := resp.Msg.GetPage()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{fmt.Sprintf("drafted %s at %s", draft.GetId(), draft.GetPath())},
	})
}

func (h *handlers) preview(ctx cliapp.RunContext) error {
	resp, err := h.client.PreviewSession(context.Background(), connect.NewRequest(&contractv1.PreviewSessionRequest{
		SessionId: ctx.Positional("session"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("preview authoring session", err, nil)
	}
	return renderDiffs(ctx, resp.Msg, resp.Msg.GetValidation(), resp.Msg.GetDiffs())
}

func (h *handlers) apply(ctx cliapp.RunContext) error {
	resp, err := h.client.ApplySession(context.Background(), connect.NewRequest(&contractv1.ApplySessionRequest{
		SessionId: ctx.Positional("session"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("apply authoring session", err, nil)
	}
	return renderDiffs(ctx, resp.Msg, resp.Msg.GetValidation(), resp.Msg.GetDiffs())
}

func (h *handlers) discard(ctx cliapp.RunContext) error {
	resp, err := h.client.DiscardSession(context.Background(), connect.NewRequest(&contractv1.DiscardSessionRequest{
		SessionId: ctx.Positional("session"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("discard authoring session", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("discarded %s", resp.Msg.GetSessionId())}})
}

func renderDiffs(ctx cliapp.RunContext, msg proto.Message, validation *contractv1.ValidateScenarioResponse, diffs []*contractv1.FileDiff) error {
	results := make([]string, 0, len(diffs))
	for _, diff := range diffs {
		results = append(results, fmt.Sprintf("%s %s", diff.GetAction(), diff.GetPath()))
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("validation: %s", validation.GetStatus())},
		ResultsHeading: "Diffs",
		Results:        results,
	})
}

func pageFormFromFlags(ctx cliapp.RunContext) (*contractv1.PageForm, error) {
	if file := strings.TrimSpace(ctx.Flag("file")); file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read page form %q: %w", file, err)
		}
		var page contractv1.PageForm
		if err := json.Unmarshal(data, &page); err != nil {
			return nil, fmt.Errorf("parse page form %q: %w", file, err)
		}
		return &page, nil
	}
	page := &contractv1.PageForm{
		Id:      ctx.Flag("id"),
		Title:   ctx.Flag("title"),
		Purpose: ctx.Flag("purpose"),
		Routes:  splitList(ctx.Flag("routes")),
		PrdRefs: splitList(ctx.Flag("prd-refs")),
		Status:  ctx.Flag("status"),
	}
	for _, p := range splitSemi(ctx.Flag("priorities")) {
		page.Priorities = append(page.Priorities, &contractv1.PriorityForm{Statement: p})
	}
	for _, item := range splitSemi(ctx.Flag("states")) {
		parts := splitFields(item, 2)
		page.States = append(page.States, &contractv1.StateForm{Id: parts[0], Description: parts[1]})
	}
	for _, item := range splitSemi(ctx.Flag("elements")) {
		parts := splitFields(item, 4)
		page.Elements = append(page.Elements, &contractv1.ElementForm{Id: parts[0], Role: parts[1], Name: parts[2], Description: parts[3]})
	}
	for _, item := range splitSemi(ctx.Flag("claims")) {
		parts := splitFields(item, 6)
		page.Claims = append(page.Claims, &contractv1.ClaimForm{
			Id:        parts[0],
			Type:      parts[1],
			Tier:      parts[2],
			Statement: parts[3],
			Elements:  splitList(parts[4]),
			States:    splitList(parts[5]),
		})
	}
	for _, item := range splitSemi(ctx.Flag("bindings")) {
		parts := splitFields(item, 3)
		page.Bindings = append(page.Bindings, &contractv1.BindingForm{ElementId: parts[0], Testid: parts[1], Selector: parts[2]})
	}
	for _, item := range splitSemi(ctx.Flag("sketch")) {
		parts := splitFields(item, 2)
		page.SketchRegions = append(page.SketchRegions, &contractv1.SketchRegionForm{Id: parts[0], Elements: splitList(parts[1])})
	}
	return page, nil
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func splitSemi(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func splitFields(value string, count int) []string {
	parts := strings.Split(value, ":")
	out := make([]string, count)
	for i := 0; i < count && i < len(parts); i++ {
		out[i] = strings.TrimSpace(parts[i])
	}
	return out
}
