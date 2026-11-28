// Package styles is the CLI surface for container styles and product lines.
package styles

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	stylesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/styles"
	stylesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/styles/styles_v1connect"
)

// GroupName is the command group name.
const GroupName = "styles"

type handlers struct {
	client stylesconnect.StylesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: stylesconnect.NewStylesServiceClient(httpClient, baseURL)}
}

// Register builds the styles command group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	cmd := func(name, desc string, flags []cliapp.Flag, run func(cliapp.RunContext) error) cliapp.Command {
		return cliapp.Command{Name: name, Description: desc, NeedsAPI: true, Args: cliapp.ArgSchema{Flags: flags}, RunCtx: run}
	}
	return cliapp.SubcommandGroup{
		Name:        GroupName,
		Description: "Container styles and product lines",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			cmd("list", "List container styles", nil, h.list),
			cmd("get", "Get a container style by id", []cliapp.Flag{{Name: "id", Required: true, Description: "Style id"}}, h.get),
			cmd("create", "Create a container style", []cliapp.Flag{
				{Name: "name", Required: true, Description: "Style name"},
				{Name: "shape", Description: "rounded_square|square"},
				{Name: "corner-ratio", Description: "Corner radius fraction"},
			}, h.create),
			cmd("update", "Update a container style", []cliapp.Flag{
				{Name: "id", Required: true, Description: "Style id"},
				{Name: "name", Description: "New name"},
				{Name: "corner-ratio", Description: "New corner ratio"},
			}, h.update),
			cmd("lines", "List product lines", nil, h.lines),
		},
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListContainerStyles(context.Background(), connect.NewRequest(&stylesv1.ListContainerStylesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list container styles", err, nil)
	}
	rows := make([]string, 0, len(resp.Msg.GetStyles()))
	for _, s := range resp.Msg.GetStyles() {
		rows = append(rows, fmt.Sprintf("%s  %-22s %s  corner=%.4f scale=%.2f accent=%s", s.GetId(), s.GetName(), s.GetShape(), s.GetCornerRatio(), s.GetMarkScale(), s.GetAccentColor()))
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("%d container style(s).", len(rows))}, ResultsHeading: "Styles", Results: rows})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.GetContainerStyle(context.Background(), connect.NewRequest(&stylesv1.GetContainerStyleRequest{Id: ctx.Flag("id")}))
	if err != nil {
		return cliapp.WrapAPIError("get container style", err, nil)
	}
	s := resp.Msg.GetStyle()
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Style %s (%s)", s.GetId(), s.GetName())}, ResultsHeading: "Style", Results: []string{fmt.Sprintf("shape=%s corner=%.4f bg=%s/%s mark_scale=%.2f maskable=%.2f accent=%s", s.GetShape(), s.GetCornerRatio(), s.GetBackgroundTop(), s.GetBackgroundBottom(), s.GetMarkScale(), s.GetMaskableScale(), s.GetAccentColor())}})
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	resp, err := h.client.CreateContainerStyle(context.Background(), connect.NewRequest(&stylesv1.CreateContainerStyleRequest{
		Name:        ctx.Flag("name"),
		Shape:       ctx.Flag("shape"),
		CornerRatio: f64OrZero(ctx.Flag("corner-ratio")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("create container style", err, nil)
	}
	return ctx.RenderMutation(cliapp.MutationReport{Result: []string{fmt.Sprintf("Created style %s", resp.Msg.GetStyle().GetId())}})
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	resp, err := h.client.UpdateContainerStyle(context.Background(), connect.NewRequest(&stylesv1.UpdateContainerStyleRequest{
		Id:          ctx.Flag("id"),
		Name:        ctx.Flag("name"),
		CornerRatio: f64OrZero(ctx.Flag("corner-ratio")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("update container style", err, nil)
	}
	return ctx.RenderMutation(cliapp.MutationReport{Result: []string{fmt.Sprintf("Updated style %s", resp.Msg.GetStyle().GetId())}})
}

func (h *handlers) lines(ctx cliapp.RunContext) error {
	resp, err := h.client.ListProductLines(context.Background(), connect.NewRequest(&stylesv1.ListProductLinesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list product lines", err, nil)
	}
	rows := make([]string, 0, len(resp.Msg.GetLines()))
	for _, l := range resp.Msg.GetLines() {
		rows = append(rows, fmt.Sprintf("%s  %-14s style=%s products=%v", l.GetId(), l.GetName(), l.GetContainerStyleId(), l.GetProducts()))
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("%d product line(s).", len(rows))}, ResultsHeading: "Product lines", Results: rows})
}

func f64OrZero(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
