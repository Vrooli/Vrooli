package related

import (
	"context"
	"errors"
	"strings"

	"swarm-manager/internal/backlog"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

// ConnectService keeps transport conversion at the edge; engine types remain
// independent of generated code so future providers do not inherit wire debt.
type ConnectService struct{ engine *Engine }

func NewConnectService(engine *Engine) *ConnectService { return &ConnectService{engine} }
func RegisterRoutes(router *mux.Router, engine *Engine) {
	p, h := apiconnect.NewRelatedServiceHandler(NewConnectService(engine))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: p, Handler: h})
}

func (s *ConnectService) GetRelated(ctx context.Context, req *connect.Request[api.GetRelatedRequest]) (*connect.Response[api.GetRelatedResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	var target TargetRef
	switch t := req.Msg.Target.(type) {
	case *api.GetRelatedRequest_Backlog:
		if t.Backlog == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("backlog target is required"))
		}
		kind, err := backlog.ParseBacklogKind(strings.TrimSpace(t.Backlog.Kind))
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		target = TargetRef{Kind: TargetBacklog, BacklogKind: kind, Name: strings.TrimSpace(t.Backlog.Name)}
	case *api.GetRelatedRequest_Goal:
		if t.Goal == nil || strings.TrimSpace(t.Goal.Name) == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("goal name is required"))
		}
		target = TargetRef{Kind: TargetGoal, Name: strings.TrimSpace(t.Goal.Name)}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target is required"))
	}
	report, err := s.engine.Compute(ctx, target, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	allowed := map[string]bool{}
	for _, k := range req.Msg.EntityKinds {
		allowed[k] = true
	}
	groups := []Group{report.Linked, report.SameScope, report.Similar}
	out := &api.GetRelatedResponse{}
	for _, group := range groups {
		pg := &api.RelatedGroup{Name: string(group.Name), Degraded: group.Degraded}
		for _, entity := range group.Entities {
			if req.Msg.ExcludeHistorical && entity.Archived {
				continue
			}
			if len(allowed) > 0 && !allowed[string(entity.Kind)] {
				continue
			}
			row := &api.RelatedEntity{EntityKind: string(entity.Kind), Key: entity.Key, Title: entity.Title, Status: entity.Status, Archived: entity.Archived, Reasons: entity.Reasons}
			if entity.ScorePercent > 0 {
				score := int32(entity.ScorePercent)
				row.ScorePercent = &score
			}
			pg.Entities = append(pg.Entities, row)
		}
		out.Groups = append(out.Groups, pg)
	}
	return connect.NewResponse(out), nil
}
