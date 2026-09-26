package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"prompt-manager/handlers/transportbridge"
	"prompt-manager/internal/projection"
	domain "prompt-manager/internal/skills"
	"prompt-manager/internal/store"
)

type connectHandler struct {
	skillsconnect.UnimplementedSkillsServiceHandler
	legacy     *domain.Handlers
	variants   *domain.VariantHandlers
	imports    domain.ImportService
	projection *projection.Service
}

func NewConnectMountWithProjection(legacy *domain.Handlers, variants *domain.VariantHandlers, imports domain.ImportService, projector *projection.Service) (string, http.Handler) {
	return skillsconnect.NewSkillsServiceHandler(&connectHandler{legacy: legacy, variants: variants, imports: imports, projection: projector})
}

func (h *connectHandler) RefreshProjection(ctx context.Context, req *connect.Request[skillsv1.RefreshProjectionRequest]) (*connect.Response[skillsv1.RefreshProjectionResponse], error) {
	if h.projection == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("projection is not configured"))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result, err := h.projection.Refresh(projection.RefreshRequest{Runtime: req.Msg.GetRuntime(), Skills: req.Msg.GetSkills(), Apply: req.Msg.GetApply(), ExpectedDigest: req.Msg.GetExpectedDigest(), AdoptLegacy: req.Msg.GetAdoptLegacy()})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	out := &skillsv1.RefreshProjectionResponse{Digest: result.Digest}
	for _, row := range result.Rows {
		out.Rows = append(out.Rows, &skillsv1.ProjectionRow{Runtime: row.Runtime, Skill: row.Skill, Status: row.Status, SourceHash: row.SourceHash, InstalledHash: row.InstalledHash, BaselineHash: row.BaselineHash, ReceiptHash: row.ReceiptHash, Error: row.Error, BackupPath: row.BackupPath, Applied: row.Applied})
	}
	return connect.NewResponse(out), nil
}

func NewConnectMount(legacy *domain.Handlers, variants *domain.VariantHandlers, imports ...domain.ImportService) (string, http.Handler) {
	var importer domain.ImportService
	if len(imports) > 0 {
		importer = imports[0]
	}
	return skillsconnect.NewSkillsServiceHandler(&connectHandler{legacy: legacy, variants: variants, imports: importer})
}

func (h *connectHandler) ImportSkill(ctx context.Context, req *connect.Request[skillsv1.ImportSkillRequest]) (*connect.Response[skillsv1.ImportSkillResponse], error) {
	if h.imports == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("skill import is unavailable"))
	}
	tools := make([]store.ExternalTool, 0, len(req.Msg.GetExternalTools()))
	for _, tool := range req.Msg.GetExternalTools() {
		tools = append(tools, store.ExternalTool{Name: tool.GetName(), URL: tool.GetUrl(), Purpose: tool.GetPurpose()})
	}
	skill, err := h.imports.ImportSkill(ctx, store.ImportRequest{SourceDir: req.Msg.GetSourceDir(), SourceURL: req.Msg.GetSourceUrl(), Commit: req.Msg.GetCommit(), License: req.Msg.GetLicense(), Checksum: req.Msg.GetChecksum(), ImportedBy: req.Msg.GetImportedBy(), UpstreamVersion: req.Msg.GetUpstreamVersion(), ID: req.Msg.GetId(), ExternalTools: tools})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&skillsv1.ImportSkillResponse{Id: skill.ID, Pack: skill.Pack, Status: skill.Status, Checksum: skill.Origin.Checksum, ReviewVerdict: skill.Origin.Review.Verdict, ImportedAt: skill.Origin.ImportedAt, TreeChecksum: skill.Origin.TreeChecksum}), nil
}

func (h *connectHandler) ReviewImportedSkill(ctx context.Context, req *connect.Request[skillsv1.ReviewImportedSkillRequest]) (*connect.Response[skillsv1.ReviewImportedSkillResponse], error) {
	if h.imports == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("skill review is unavailable"))
	}
	if err := h.imports.ReviewImportedSkill(ctx, req.Msg.GetId(), req.Msg.GetReviewer(), req.Msg.GetVerdict()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&skillsv1.ReviewImportedSkillResponse{Id: req.Msg.GetId(), Verdict: req.Msg.GetVerdict(), Reviewer: req.Msg.GetReviewer()}), nil
}

func (h *connectHandler) ReportImportedSkillStaleness(ctx context.Context, req *connect.Request[skillsv1.ReportImportedSkillStalenessRequest]) (*connect.Response[skillsv1.ReportImportedSkillStalenessResponse], error) {
	if h.imports == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("skill staleness reporting is unavailable"))
	}
	recorded, stale, err := h.imports.ImportedSkillStaleness(ctx, req.Msg.GetId(), req.Msg.GetUpstreamVersion())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&skillsv1.ReportImportedSkillStalenessResponse{Id: req.Msg.GetId(), RecordedVersion: recorded, CurrentVersion: req.Msg.GetUpstreamVersion(), Stale: stale}), nil
}

func (h *connectHandler) ListSkills(ctx context.Context, req *connect.Request[skillsv1.ListSkillsRequest]) (*connect.Response[skillsv1.ListSkillsResponse], error) {
	items, err := h.legacy.ListSkills(ctx, domain.FilterOptions{Folder: req.Msg.GetFolder(), Tag: req.Msg.GetTag(), Modes: req.Msg.GetModes(), WithoutProgrammaticHome: req.Msg.GetWithoutProgrammaticHome()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &skillsv1.ListSkillsResponse{}
	if err := decodeSkillJSON(map[string]any{"skills": items}, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetSkill(ctx context.Context, req *connect.Request[skillsv1.GetSkillRequest]) (*connect.Response[skillsv1.GetSkillResponse], error) {
	item, err := h.legacy.GetSkill(ctx, req.Msg.GetId())
	if err != nil {
		code := connect.CodeInternal
		if err.Error() == "Skill not found" {
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}
	out := &skillsv1.GetSkillResponse{}
	if err := decodeSkillJSON(map[string]any{"skill": item}, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func decodeSkillJSON(value any, target proto.Message) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, target); err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	return nil
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
	result, err := h.legacy.CreateSkill(ctx, domain.CreateRequest{
		ID:               req.Msg.GetId(),
		Name:             req.Msg.GetName(),
		Description:      req.Msg.GetDescription(),
		Content:          req.Msg.GetContent(),
		Modes:            req.Msg.GetModes(),
		Tags:             req.Msg.GetTags(),
		Icon:             req.Msg.GetIcon(),
		TargetToolID:     req.Msg.TargetToolId,
		DefaultScope:     req.Msg.GetDefaultScope(),
		TargetDimensions: req.Msg.GetTargetDimensions(),
		ProgrammaticHome: req.Msg.ProgrammaticHome,
		Draft:            req.Msg.GetDraft(),
		Folder:           req.Msg.GetFolder(),
	})
	if err != nil {
		code := connect.CodeInternal
		switch err.Error() {
		case "folder must be one of: local, drafts, core", "Name and content are required":
			code = connect.CodeInvalidArgument
		case "Skill with this ID already exists":
			code = connect.CodeAlreadyExists
		}
		return nil, connect.NewError(code, err)
	}
	out := &skillsv1.CreateSkillResponse{}
	if err := decodeSkillJSON(map[string]any{"skill": result}, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) UpdateSkill(ctx context.Context, req *connect.Request[skillsv1.UpdateSkillRequest]) (*connect.Response[skillsv1.UpdateSkillResponse], error) {
	update := domain.UpdateRequest{
		File:                  req.Msg.File,
		Name:                  req.Msg.Name,
		Description:           req.Msg.Description,
		Content:               req.Msg.Content,
		Icon:                  req.Msg.Icon,
		TargetToolID:          req.Msg.TargetToolId,
		DefaultScope:          req.Msg.DefaultScope,
		ProgrammaticHome:      req.Msg.ProgrammaticHome,
		ClearProgrammaticHome: req.Msg.GetClearProgrammaticHome(),
		Draft:                 req.Msg.Draft,
		Folder:                req.Msg.Folder,
	}
	if req.Msg.GetReplaceModes() {
		update.Modes = req.Msg.GetModes()
	}
	if req.Msg.GetReplaceTags() {
		update.Tags = req.Msg.GetTags()
	}
	if req.Msg.GetReplaceTargetDimensions() {
		update.TargetDimensions = req.Msg.GetTargetDimensions()
	}
	result, err := h.legacy.UpdateSkill(ctx, req.Msg.GetId(), update)
	if err != nil {
		code := connect.CodeInternal
		message := err.Error()
		switch {
		case message == "Skill not found":
			code = connect.CodeNotFound
		case strings.HasPrefix(message, "Cannot edit vendored skill") || message == "Cannot update core skills":
			code = connect.CodePermissionDenied
		case message == "Cannot move skill to non-writable folder" || strings.HasPrefix(message, "content removes existing template variables:") || strings.Contains(message, "invalid skill ID format"):
			code = connect.CodeInvalidArgument
		case strings.Contains(message, "already exists"):
			code = connect.CodeAlreadyExists
		}
		return nil, connect.NewError(code, err)
	}
	out := &skillsv1.UpdateSkillResponse{}
	if err := decodeSkillJSON(map[string]any{"skill": result}, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) DeleteSkill(ctx context.Context, req *connect.Request[skillsv1.DeleteSkillRequest]) (*connect.Response[skillsv1.DeleteSkillResponse], error) {
	if err := h.legacy.DeleteSkill(ctx, req.Msg.GetId()); err != nil {
		code := connect.CodeInternal
		switch err.Error() {
		case "Skill not found":
			code = connect.CodeNotFound
		case "Cannot delete core skills":
			code = connect.CodePermissionDenied
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&skillsv1.DeleteSkillResponse{Id: req.Msg.GetId(), Deleted: true}), nil
}

func (h *connectHandler) SyncSkills(ctx context.Context, req *connect.Request[skillsv1.SyncSkillsRequest]) (*connect.Response[skillsv1.SyncSkillsResponse], error) {
	result, err := h.legacy.SyncSkills(ctx, "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &skillsv1.SyncSkillsResponse{}
	if err := decodeSkillJSON(result, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RateSkill(ctx context.Context, req *connect.Request[skillsv1.RateSkillRequest]) (*connect.Response[skillsv1.RateSkillResponse], error) {
	if err := h.legacy.SetSkillRating(ctx, req.Msg.GetId(), int(req.Msg.GetRating()), req.Msg.Notes); err != nil {
		code := connect.CodeInternal
		switch err.Error() {
		case "Rating must be between 1 and 5":
			code = connect.CodeInvalidArgument
		case "Skill not found":
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}
	out := &skillsv1.RateSkillResponse{Id: req.Msg.GetId(), Rating: req.Msg.GetRating(), Status: "rating updated"}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RecordSkillUsage(ctx context.Context, req *connect.Request[skillsv1.RecordSkillUsageRequest]) (*connect.Response[skillsv1.RecordSkillUsageResponse], error) {
	usageCount, lastUsed, err := h.legacy.RecordSkillUsage(ctx, req.Msg.GetId())
	if err != nil {
		code := connect.CodeInternal
		if err.Error() == "Skill not found" {
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}
	lastUsedValue := ""
	if !lastUsed.IsZero() {
		lastUsedValue = lastUsed.Format(time.RFC3339Nano)
	}
	out := &skillsv1.RecordSkillUsageResponse{Id: req.Msg.GetId(), Status: "usage recorded", UsageCount: int32(usageCount), LastUsed: lastUsedValue}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListSkillVersions(ctx context.Context, req *connect.Request[skillsv1.ListSkillVersionsRequest]) (*connect.Response[skillsv1.ListSkillVersionsResponse], error) {
	result, err := h.legacy.ListSkillVersions(ctx, req.Msg.GetId())
	if err != nil {
		code := connect.CodeInternal
		if strings.Contains(err.Error(), "not found") {
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}
	out := &skillsv1.ListSkillVersionsResponse{}
	if err := decodeSkillJSON(result, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RevertSkill(ctx context.Context, req *connect.Request[skillsv1.RevertSkillRequest]) (*connect.Response[skillsv1.RevertSkillResponse], error) {
	result, err := h.legacy.RevertSkillVersion(ctx, req.Msg.GetId(), int(req.Msg.GetVersion()))
	if err != nil {
		code := connect.CodeInternal
		switch err.Error() {
		case "Skill not found":
			code = connect.CodeNotFound
		case "Cannot revert core skills":
			code = connect.CodePermissionDenied
		}
		return nil, connect.NewError(code, err)
	}
	out := &skillsv1.RevertSkillResponse{}
	if err := decodeSkillJSON(result, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListSkillVariants(ctx context.Context, req *connect.Request[skillsv1.ListSkillVariantsRequest]) (*connect.Response[skillsv1.ListSkillVariantsResponse], error) {
	result, err := h.variants.ListSkillVariants(ctx, req.Msg.GetSkillId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	out := &skillsv1.ListSkillVariantsResponse{}
	if err := decodeSkillJSON(map[string]any{"variants": result}, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetSkillVariant(ctx context.Context, req *connect.Request[skillsv1.GetSkillVariantRequest]) (*connect.Response[skillsv1.GetSkillVariantResponse], error) {
	result, err := h.variants.GetSkillVariant(ctx, req.Msg.GetSkillId(), req.Msg.GetVariantId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	out := &skillsv1.GetSkillVariantResponse{}
	if err := decodeSkillJSON(map[string]any{"variant": result}, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateSkillVariant(ctx context.Context, req *connect.Request[skillsv1.CreateSkillVariantRequest]) (*connect.Response[skillsv1.CreateSkillVariantResponse], error) {
	result, err := h.variants.CreateSkillVariant(ctx, req.Msg.GetSkillId(), domain.CreateVariantRequest{ID: req.Msg.GetId(), Name: req.Msg.GetName(), Description: req.Msg.GetDescription(), Content: req.Msg.GetContent()})
	if err != nil {
		code := connect.CodeAlreadyExists
		if err.Error() == "id, name, and content are required" {
			code = connect.CodeInvalidArgument
		}
		return nil, connect.NewError(code, err)
	}
	out := &skillsv1.CreateSkillVariantResponse{}
	if err := decodeSkillJSON(map[string]any{"variant": result}, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) UpdateSkillVariant(ctx context.Context, req *connect.Request[skillsv1.UpdateSkillVariantRequest]) (*connect.Response[skillsv1.UpdateSkillVariantResponse], error) {
	result, err := h.variants.UpdateSkillVariant(ctx, req.Msg.GetSkillId(), req.Msg.GetVariantId(), domain.UpdateVariantRequest{Name: req.Msg.Name, Description: req.Msg.Description, Content: req.Msg.Content})
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	out := &skillsv1.UpdateSkillVariantResponse{}
	if err := decodeSkillJSON(map[string]any{"variant": result}, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) DeleteSkillVariant(ctx context.Context, req *connect.Request[skillsv1.DeleteSkillVariantRequest]) (*connect.Response[skillsv1.DeleteSkillVariantResponse], error) {
	if err := h.variants.DeleteSkillVariant(ctx, req.Msg.GetSkillId(), req.Msg.GetVariantId()); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
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
