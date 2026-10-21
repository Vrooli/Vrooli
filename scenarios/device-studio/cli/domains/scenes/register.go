package scenes

import (
	"connectrpc.com/connect"
	"context"
	"device-studio/cli/domains/componentrevisions"
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes/scenes_v1connect"
	"google.golang.org/protobuf/proto"
)

const GroupName = "scenes"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	hc, base := cliapp.NewConnectHTTPClient(core)
	c := api.NewScenesServiceClient(hc, base)
	return componentrevisions.Register(core, manifest, componentrevisions.Spec{Group: GroupName, Service: "ScenesService", CandidateName: "Scene", SaveMethod: "SaveSceneCandidate", GetMethod: "GetSceneRevision", HeadMethod: "GetSceneHead", RevisionsMethod: "ListSceneRevisions", CandidatesMethod: "ListSceneCandidates", Candidate: func() proto.Message { return &v1.SceneCandidate{} },
		Save: func(ctx context.Context, m proto.Message, n uint32) (proto.Message, error) {
			r, e := c.SaveSceneCandidate(ctx, connect.NewRequest(&v1.SaveSceneCandidateRequest{Candidate: m.(*v1.SceneCandidate), ExpectedHead: n}))
			if e != nil {
				return nil, componentrevisions.APIError("save scene candidate", GroupName, e)
			}
			return r.Msg, e
		},
		Get: func(ctx context.Context, id string, n uint32) (proto.Message, error) {
			r, e := c.GetSceneRevision(ctx, connect.NewRequest(&v1.GetSceneRevisionRequest{SceneId: id, Revision: n}))
			if e != nil {
				return nil, componentrevisions.APIError("get scene revision", GroupName, e)
			}
			return r.Msg, e
		},
		Head: func(ctx context.Context, id string) (proto.Message, error) {
			r, e := c.GetSceneHead(ctx, connect.NewRequest(&v1.GetSceneHeadRequest{SceneId: id}))
			if e != nil {
				return nil, componentrevisions.APIError("get scene head", GroupName, e)
			}
			return r.Msg, e
		},
		Revisions: func(ctx context.Context, id string, a, p uint32) (proto.Message, error) {
			r, e := c.ListSceneRevisions(ctx, connect.NewRequest(&v1.ListSceneRevisionsRequest{SceneId: id, AfterRevision: a, PageSize: p}))
			if e != nil {
				return nil, componentrevisions.APIError("list scene revisions", GroupName, e)
			}
			return r.Msg, e
		},
		Candidates: func(ctx context.Context, after string, p uint32) (proto.Message, error) {
			r, e := c.ListSceneCandidates(ctx, connect.NewRequest(&v1.ListSceneCandidatesRequest{AfterSceneId: after, PageSize: p}))
			if e != nil {
				return nil, componentrevisions.APIError("list scene candidates", GroupName, e)
			}
			return r.Msg, nil
		},
		CandidateRows: func(m proto.Message) ([]proto.Message, string) {
			r := m.(*v1.ListSceneCandidatesResponse)
			out := make([]proto.Message, 0, len(r.Candidates))
			for _, v := range r.Candidates {
				out = append(out, v)
			}
			return out, r.NextAfterSceneId
		},
		RevisionFromSave: func(m proto.Message) proto.Message { return m.(*v1.SaveSceneCandidateResponse).Revision }, RevisionFromGet: func(m proto.Message) proto.Message { return m.(*v1.GetSceneRevisionResponse).Revision }, RevisionFromHead: func(m proto.Message) proto.Message { return m.(*v1.GetSceneHeadResponse).Revision },
		RevisionRows: func(m proto.Message) ([]proto.Message, uint32) {
			r := m.(*v1.ListSceneRevisionsResponse)
			out := make([]proto.Message, 0, len(r.Revisions))
			for _, v := range r.Revisions {
				out = append(out, v)
			}
			return out, r.NextAfterRevision
		}, RevisionID: func(m proto.Message) string { return m.(*v1.SceneRevision).SceneId }, RevisionNumber: func(m proto.Message) uint32 { return m.(*v1.SceneRevision).Revision }, DescribeRevision: func(m proto.Message) string {
			r := m.(*v1.SceneRevision)
			return fmt.Sprintf("scene=%s revision=%d state=%s sha256=%s", r.SceneId, r.Revision, r.State, r.Sha256)
		}})
}
