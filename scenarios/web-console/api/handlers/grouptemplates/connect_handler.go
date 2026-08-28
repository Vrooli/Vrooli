package grouptemplates

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	grouptemplatesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/grouptemplates"

	gtdomain "web-console/internal/grouptemplates"
)

// Deps wires the seams the Connect group-templates handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// GroupTemplatesServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// mapError maps the domain sentinel onto a Connect code. A blank name or an
// unknown start mode is a caller mistake, so it must not read as an internal
// fault in the client's error handling.
func (h *connectHandler) mapError(op string, err error) error {
	if errors.Is(err, gtdomain.ErrInvalidTemplate) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	h.deps.Logger.Printf("grouptemplates.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}

func (h *connectHandler) ListTemplates(ctx context.Context, _ *connect.Request[grouptemplatesv1.ListTemplatesRequest]) (*connect.Response[grouptemplatesv1.ListTemplatesResponse], error) {
	templates, err := h.deps.Service.List(ctx)
	if err != nil {
		return nil, h.mapError("ListTemplates", err)
	}
	out := make([]*grouptemplatesv1.Template, 0, len(templates))
	for _, t := range templates {
		out = append(out, templateToProto(t))
	}
	return connect.NewResponse(&grouptemplatesv1.ListTemplatesResponse{Templates: out}), nil
}

func (h *connectHandler) UpsertTemplate(ctx context.Context, req *connect.Request[grouptemplatesv1.UpsertTemplateRequest]) (*connect.Response[grouptemplatesv1.UpsertTemplateResponse], error) {
	in := UpsertRequest{
		ID:          req.Msg.GetId(),
		Name:        req.Msg.GetName(),
		Color:       req.Msg.GetColor(),
		Roles:       rolesFromProto(req.Msg.GetRoles()),
		UseCount:    int(req.Msg.GetUseCount()),
		HasUseCount: req.Msg.GetHasUseCount(),
	}
	t, err := h.deps.Service.Upsert(ctx, in)
	if err != nil {
		return nil, h.mapError("UpsertTemplate", err)
	}
	return connect.NewResponse(&grouptemplatesv1.UpsertTemplateResponse{Template: templateToProto(t)}), nil
}

func (h *connectHandler) DeleteTemplate(ctx context.Context, req *connect.Request[grouptemplatesv1.DeleteTemplateRequest]) (*connect.Response[grouptemplatesv1.DeleteTemplateResponse], error) {
	if _, err := h.deps.Service.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, h.mapError("DeleteTemplate", err)
	}
	return connect.NewResponse(&grouptemplatesv1.DeleteTemplateResponse{}), nil
}

func templateToProto(t Template) *grouptemplatesv1.Template {
	roles := make([]*grouptemplatesv1.TemplateRole, 0, len(t.Roles))
	for _, r := range t.Roles {
		roles = append(roles, &grouptemplatesv1.TemplateRole{
			Label:          r.Label,
			Command:        r.Command,
			WorkingDir:     r.WorkingDir,
			IncomingPrompt: r.IncomingPrompt,
			Backend:        r.Backend,
			TargetId:       r.TargetID,
			StartMode:      r.StartMode,
		})
	}
	return &grouptemplatesv1.Template{
		Id:        t.ID,
		Name:      t.Name,
		Color:     t.Color,
		Roles:     roles,
		UseCount:  int32(t.UseCount),
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func rolesFromProto(in []*grouptemplatesv1.TemplateRole) []TemplateRole {
	out := make([]TemplateRole, 0, len(in))
	for _, r := range in {
		out = append(out, TemplateRole{
			Label:          r.GetLabel(),
			Command:        r.GetCommand(),
			WorkingDir:     r.GetWorkingDir(),
			IncomingPrompt: r.GetIncomingPrompt(),
			Backend:        r.GetBackend(),
			TargetID:       r.GetTargetId(),
			StartMode:      r.GetStartMode(),
		})
	}
	return out
}
