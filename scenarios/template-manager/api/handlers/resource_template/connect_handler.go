package resourcetemplate

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	resourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/resource_template"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templateengine"
)

type connectHandler struct {
	engine *templateengine.Engine
}

func NewConnectHandler(engine *templateengine.Engine) *connectHandler {
	return &connectHandler{engine: engine}
}

func (h *connectHandler) ListResourceTemplates(ctx context.Context, _ *connect.Request[resourcev1.ListResourceTemplatesRequest]) (*connect.Response[resourcev1.ListResourceTemplatesResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	items, err := engine.ListResourceTemplates(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &resourcev1.ListResourceTemplatesResponse{}
	for _, item := range items {
		resp.Templates = append(resp.Templates, infoToProto(item))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetResourceTemplate(ctx context.Context, req *connect.Request[resourcev1.GetResourceTemplateRequest]) (*connect.Response[resourcev1.GetResourceTemplateResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	item, err := engine.ShowResourceTemplate(ctx, req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&resourcev1.GetResourceTemplateResponse{Template: infoToProto(item)}), nil
}

func (h *connectHandler) ValidateResourceTemplates(ctx context.Context, _ *connect.Request[resourcev1.ValidateResourceTemplatesRequest]) (*connect.Response[resourcev1.ValidateResourceTemplatesResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	report, err := engine.ValidateResourceTemplates(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &resourcev1.ValidateResourceTemplatesResponse{Count: int32(report.Count)}
	for _, item := range report.Templates {
		resp.Templates = append(resp.Templates, summaryToProto(item))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GenerateResourceTemplate(ctx context.Context, req *connect.Request[resourcev1.GenerateResourceTemplateRequest]) (*connect.Response[resourcev1.GenerateResourceTemplateResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	report, err := engine.GenerateResourceTemplate(ctx, templateengine.ResourceTemplateGenerateRequest{
		TemplateName:  req.Msg.Template,
		BlueprintName: req.Msg.FromBlueprint,
		Destination:   req.Msg.Destination,
		Force:         req.Msg.Force,
		DryRun:        req.Msg.DryRun,
		Values:        req.Msg.Values,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&resourcev1.GenerateResourceTemplateResponse{
		Template:      summaryToProto(report.Template),
		BlueprintName: report.BlueprintName,
		Destination:   report.Destination,
		Values:        report.Values,
		Files:         report.Files,
		DryRun:        report.DryRun,
	}), nil
}

func (h *connectHandler) requireEngine() (*templateengine.Engine, error) {
	if h.engine == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("template engine unavailable"))
	}
	return h.engine, nil
}

func infoToProto(info templateengine.ResourceTemplateInfo) *resourcev1.ResourceTemplateInfo {
	return &resourcev1.ResourceTemplateInfo{
		Name:     info.Name,
		Path:     info.Path,
		Manifest: manifestToProto(info.Manifest),
	}
}

func manifestToProto(manifest templateengine.ResourceTemplateManifest) *resourcev1.ResourceTemplateManifest {
	return &resourcev1.ResourceTemplateManifest{
		Name:                 manifest.Name,
		DisplayName:          manifest.DisplayName,
		Description:          manifest.Description,
		Driver:               manifest.Driver,
		RequiredVars:         varsToProto(manifest.RequiredVars),
		OptionalVars:         varsToProto(manifest.OptionalVars),
		Docs:                 manifest.Docs,
		PlatformExpectations: manifest.PlatformExpectations,
		Transitional:         manifest.Transitional,
	}
}

func varsToProto(vars map[string]templateengine.ResourceTemplateVar) map[string]*resourcev1.ResourceTemplateVar {
	if vars == nil {
		return nil
	}
	out := make(map[string]*resourcev1.ResourceTemplateVar, len(vars))
	for key, item := range vars {
		out[key] = &resourcev1.ResourceTemplateVar{Flag: item.Flag, Description: item.Description, Default: item.Default}
	}
	return out
}

func summaryToProto(summary templateengine.ResourceTemplateSummary) *resourcev1.ResourceTemplateSummary {
	return &resourcev1.ResourceTemplateSummary{
		Name:         summary.Name,
		DisplayName:  summary.DisplayName,
		Driver:       summary.Driver,
		Transitional: summary.Transitional,
		Description:  summary.Description,
	}
}
