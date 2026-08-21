package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/skills"
)

type connectHandler struct {
	skillsconnect.UnimplementedSkillsServiceHandler
	legacy   *domain.Handlers
	variants *domain.VariantHandlers
}

func NewConnectMount(legacy *domain.Handlers, variants *domain.VariantHandlers) (string, http.Handler) {
	return skillsconnect.NewSkillsServiceHandler(&connectHandler{legacy: legacy, variants: variants})
}

func (h *connectHandler) ListSkills(ctx context.Context, req *connect.Request[skillsv1.ListSkillsRequest]) (*connect.Response[skillsv1.ListSkillsResponse], error) {
	query := url.Values{}
	if req.Msg.GetFolder() != "" {
		query.Set("folder", req.Msg.GetFolder())
	}
	if req.Msg.GetTag() != "" {
		query.Set("tag", req.Msg.GetTag())
	}
	if len(req.Msg.GetModes()) > 0 {
		query.Set("modes", strings.Join(req.Msg.GetModes(), ","))
	}
	if req.Msg.GetWithoutProgrammaticHome() {
		query.Set("withoutProgrammaticHome", "true")
	}
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.List, http.MethodGet, "/skills?"+query.Encode(), nil, nil)
	if err != nil {
		return nil, err
	}
	out := &skillsv1.ListSkillsResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "skills", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetSkill(ctx context.Context, req *connect.Request[skillsv1.GetSkillRequest]) (*connect.Response[skillsv1.GetSkillResponse], error) {
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Get, http.MethodGet, "/skills/"+url.PathEscape(req.Msg.GetId()), nil, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	out := &skillsv1.GetSkillResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "skill", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ReadSkills(ctx context.Context, req *connect.Request[skillsv1.ReadSkillsRequest]) (*connect.Response[skillsv1.ReadSkillsResponse], error) {
	payload, err := protoObject(req.Msg)
	if err != nil {
		return nil, err
	}
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Read, http.MethodPost, "/skills/read", payload, nil)
	if err != nil {
		return nil, err
	}
	out := &skillsv1.ReadSkillsResponse{}
	if err := transportbridge.Decode(result.Body, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateSkill(ctx context.Context, req *connect.Request[skillsv1.CreateSkillRequest]) (*connect.Response[skillsv1.CreateSkillResponse], error) {
	payload, err := protoObject(req.Msg)
	if err != nil {
		return nil, err
	}
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Create, http.MethodPost, "/skills", payload, nil)
	if err != nil {
		return nil, err
	}
	out := &skillsv1.CreateSkillResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "skill", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) UpdateSkill(ctx context.Context, req *connect.Request[skillsv1.UpdateSkillRequest]) (*connect.Response[skillsv1.UpdateSkillResponse], error) {
	payload := map[string]any{}
	copyOptionalString(payload, "file", req.Msg.File)
	copyOptionalString(payload, "name", req.Msg.Name)
	copyOptionalString(payload, "description", req.Msg.Description)
	copyOptionalString(payload, "content", req.Msg.Content)
	if req.Msg.GetReplaceModes() {
		payload["modes"] = req.Msg.GetModes()
	}
	if req.Msg.GetReplaceTags() {
		payload["tags"] = req.Msg.GetTags()
	}
	copyOptionalString(payload, "icon", req.Msg.Icon)
	copyOptionalString(payload, "targetToolId", req.Msg.TargetToolId)
	copyOptionalString(payload, "defaultScope", req.Msg.DefaultScope)
	if req.Msg.GetReplaceTargetDimensions() {
		payload["targetDimensions"] = req.Msg.GetTargetDimensions()
	}
	copyOptionalString(payload, "programmaticHome", req.Msg.ProgrammaticHome)
	if req.Msg.GetClearProgrammaticHome() {
		payload["clearProgrammaticHome"] = true
	}
	if req.Msg.Draft != nil {
		payload["draft"] = req.Msg.GetDraft()
	}
	copyOptionalString(payload, "folder", req.Msg.Folder)
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Update, http.MethodPut, "/skills/"+url.PathEscape(req.Msg.GetId()), payload, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	out := &skillsv1.UpdateSkillResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "skill", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) DeleteSkill(ctx context.Context, req *connect.Request[skillsv1.DeleteSkillRequest]) (*connect.Response[skillsv1.DeleteSkillResponse], error) {
	_, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Delete, http.MethodDelete, "/skills/"+url.PathEscape(req.Msg.GetId()), nil, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&skillsv1.DeleteSkillResponse{Id: req.Msg.GetId(), Deleted: true}), nil
}

func (h *connectHandler) SyncSkills(ctx context.Context, req *connect.Request[skillsv1.SyncSkillsRequest]) (*connect.Response[skillsv1.SyncSkillsResponse], error) {
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Sync, http.MethodGet, "/skills/sync", nil, nil)
	if err != nil {
		return nil, err
	}
	out := &skillsv1.SyncSkillsResponse{}
	if err := transportbridge.Decode(result.Body, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RateSkill(ctx context.Context, req *connect.Request[skillsv1.RateSkillRequest]) (*connect.Response[skillsv1.RateSkillResponse], error) {
	payload := map[string]any{"rating": req.Msg.GetRating()}
	if req.Msg.Notes != nil {
		payload["notes"] = req.Msg.GetNotes()
	}
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.SetRating, http.MethodPut, "/skills/"+url.PathEscape(req.Msg.GetId())+"/rating", payload, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	out := &skillsv1.RateSkillResponse{Id: req.Msg.GetId()}
	if err := transportbridge.Decode(result.Body, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RecordSkillUsage(ctx context.Context, req *connect.Request[skillsv1.RecordSkillUsageRequest]) (*connect.Response[skillsv1.RecordSkillUsageResponse], error) {
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.RecordUsage, http.MethodPost, "/skills/"+url.PathEscape(req.Msg.GetId())+"/use", nil, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	out := &skillsv1.RecordSkillUsageResponse{Id: req.Msg.GetId()}
	if err := transportbridge.Decode(result.Body, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListSkillVersions(ctx context.Context, req *connect.Request[skillsv1.ListSkillVersionsRequest]) (*connect.Response[skillsv1.ListSkillVersionsResponse], error) {
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.GetVersions, http.MethodGet, "/skills/"+url.PathEscape(req.Msg.GetId())+"/versions", nil, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	out := &skillsv1.ListSkillVersionsResponse{}
	if err := transportbridge.Decode(result.Body, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RevertSkill(ctx context.Context, req *connect.Request[skillsv1.RevertSkillRequest]) (*connect.Response[skillsv1.RevertSkillResponse], error) {
	version := strconv.Itoa(int(req.Msg.GetVersion()))
	result, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.RevertToVersion, http.MethodPost, "/skills/"+url.PathEscape(req.Msg.GetId())+"/revert/"+version, map[string]any{}, map[string]string{"id": req.Msg.GetId(), "version": version})
	if err != nil {
		return nil, err
	}
	out := &skillsv1.RevertSkillResponse{}
	if err := transportbridge.Decode(result.Body, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListSkillVariants(ctx context.Context, req *connect.Request[skillsv1.ListSkillVariantsRequest]) (*connect.Response[skillsv1.ListSkillVariantsResponse], error) {
	result, err := transportbridge.Invoke(ctx, req.Header(), h.variants.ListVariants, http.MethodGet, "/skills/"+url.PathEscape(req.Msg.GetSkillId())+"/variants", nil, map[string]string{"id": req.Msg.GetSkillId()})
	if err != nil {
		return nil, err
	}
	out := &skillsv1.ListSkillVariantsResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "variants", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetSkillVariant(ctx context.Context, req *connect.Request[skillsv1.GetSkillVariantRequest]) (*connect.Response[skillsv1.GetSkillVariantResponse], error) {
	result, err := transportbridge.Invoke(ctx, req.Header(), h.variants.GetVariant, http.MethodGet, "/skills/"+url.PathEscape(req.Msg.GetSkillId())+"/variants/"+url.PathEscape(req.Msg.GetVariantId()), nil, map[string]string{"id": req.Msg.GetSkillId(), "vid": req.Msg.GetVariantId()})
	if err != nil {
		return nil, err
	}
	out := &skillsv1.GetSkillVariantResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "variant", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateSkillVariant(ctx context.Context, req *connect.Request[skillsv1.CreateSkillVariantRequest]) (*connect.Response[skillsv1.CreateSkillVariantResponse], error) {
	payload := map[string]any{"id": req.Msg.GetId(), "name": req.Msg.GetName(), "description": req.Msg.GetDescription(), "content": req.Msg.GetContent()}
	result, err := transportbridge.Invoke(ctx, req.Header(), h.variants.CreateVariant, http.MethodPost, "/skills/"+url.PathEscape(req.Msg.GetSkillId())+"/variants", payload, map[string]string{"id": req.Msg.GetSkillId()})
	if err != nil {
		return nil, err
	}
	out := &skillsv1.CreateSkillVariantResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "variant", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) UpdateSkillVariant(ctx context.Context, req *connect.Request[skillsv1.UpdateSkillVariantRequest]) (*connect.Response[skillsv1.UpdateSkillVariantResponse], error) {
	payload := map[string]any{}
	copyOptionalString(payload, "name", req.Msg.Name)
	copyOptionalString(payload, "description", req.Msg.Description)
	copyOptionalString(payload, "content", req.Msg.Content)
	result, err := transportbridge.Invoke(ctx, req.Header(), h.variants.UpdateVariant, http.MethodPut, "/skills/"+url.PathEscape(req.Msg.GetSkillId())+"/variants/"+url.PathEscape(req.Msg.GetVariantId()), payload, map[string]string{"id": req.Msg.GetSkillId(), "vid": req.Msg.GetVariantId()})
	if err != nil {
		return nil, err
	}
	out := &skillsv1.UpdateSkillVariantResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "variant", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) DeleteSkillVariant(ctx context.Context, req *connect.Request[skillsv1.DeleteSkillVariantRequest]) (*connect.Response[skillsv1.DeleteSkillVariantResponse], error) {
	_, err := transportbridge.Invoke(ctx, req.Header(), h.variants.DeleteVariant, http.MethodDelete, "/skills/"+url.PathEscape(req.Msg.GetSkillId())+"/variants/"+url.PathEscape(req.Msg.GetVariantId()), nil, map[string]string{"id": req.Msg.GetSkillId(), "vid": req.Msg.GetVariantId()})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&skillsv1.DeleteSkillVariantResponse{SkillId: req.Msg.GetSkillId(), VariantId: req.Msg.GetVariantId(), Deleted: true}), nil
}

func protoObject(message proto.Message) (map[string]any, error) {
	raw, err := protojson.Marshal(message)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode protobuf request: %w", err))
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode protobuf request: %w", err))
	}
	return payload, nil
}

func copyOptionalString(target map[string]any, key string, value *string) {
	if value != nil {
		target[key] = *value
	}
}
