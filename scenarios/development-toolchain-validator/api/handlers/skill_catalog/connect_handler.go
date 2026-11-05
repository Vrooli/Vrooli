package skill_catalog

import (
	"context"
	"log"

	skillcatalog "development-toolchain-validator/internal/skill_catalog"

	"connectrpc.com/connect"

	skillcatalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/skill_catalog"
)

// Deps wires the seams the Connect skill_catalog handler needs.
type Deps struct {
	Service skillcatalog.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect handler for SkillCatalogService.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Sync(ctx context.Context, _ *connect.Request[skillcatalogv1.SyncRequest]) (*connect.Response[skillcatalogv1.SyncResponse], error) {
	result, err := h.deps.Service.Sync(ctx)
	if err != nil {
		connectErr := skillcatalog.ToConnectError(err)
		if code := connect.CodeOf(connectErr); code == connect.CodeInternal || code == connect.CodeUnavailable {
			h.deps.Logger.Printf("skill_catalog.Sync: %v", err)
		}
		return nil, connectErr
	}
	resp := &skillcatalogv1.SyncResponse{
		Skills:  make([]*skillcatalogv1.Skill, 0, len(result.Skills)),
		Added:   int32(result.Added),
		Updated: int32(result.Updated),
		Removed: int32(result.Removed),
	}
	for _, s := range result.Skills {
		resp.Skills = append(resp.Skills, domainToProto(s))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListSkills(ctx context.Context, _ *connect.Request[skillcatalogv1.ListSkillsRequest]) (*connect.Response[skillcatalogv1.ListSkillsResponse], error) {
	skills, err := h.deps.Service.List(ctx)
	if err != nil {
		h.deps.Logger.Printf("skill_catalog.ListSkills: %v", err)
		return nil, skillcatalog.ToConnectError(err)
	}
	resp := &skillcatalogv1.ListSkillsResponse{Skills: make([]*skillcatalogv1.Skill, 0, len(skills))}
	for _, s := range skills {
		resp.Skills = append(resp.Skills, domainToProto(s))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetSkill(ctx context.Context, req *connect.Request[skillcatalogv1.GetSkillRequest]) (*connect.Response[skillcatalogv1.GetSkillResponse], error) {
	s, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		connectErr := skillcatalog.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("skill_catalog.GetSkill(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&skillcatalogv1.GetSkillResponse{Skill: domainToProto(s)}), nil
}
