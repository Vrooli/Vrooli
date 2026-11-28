package motions

import (
	"connectrpc.com/connect"
	"context"
	"device-studio/cli/domains/componentrevisions"
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/motions"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/motions/motions_v1connect"
	"google.golang.org/protobuf/proto"
)

const GroupName = "motions"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	hc, base := cliapp.NewConnectHTTPClient(core)
	c := api.NewMotionsServiceClient(hc, base)
	return componentrevisions.Register(core, manifest, componentrevisions.Spec{Group: GroupName, Service: "MotionsService", CandidateName: "Motion", SaveMethod: "SaveMotionCandidate", GetMethod: "GetMotionRevision", HeadMethod: "GetMotionHead", RevisionsMethod: "ListMotionRevisions", CandidatesMethod: "ListMotionCandidates", Candidate: func() proto.Message { return &v1.MotionCandidate{} },
		Save: func(ctx context.Context, m proto.Message, n uint32) (proto.Message, error) {
			r, e := c.SaveMotionCandidate(ctx, connect.NewRequest(&v1.SaveMotionCandidateRequest{Candidate: m.(*v1.MotionCandidate), ExpectedHead: n}))
			if e != nil {
				return nil, componentrevisions.APIError("save motion candidate", GroupName, e)
			}
			return r.Msg, e
		},
		Get: func(ctx context.Context, id string, n uint32) (proto.Message, error) {
			r, e := c.GetMotionRevision(ctx, connect.NewRequest(&v1.GetMotionRevisionRequest{MotionId: id, Revision: n}))
			if e != nil {
				return nil, componentrevisions.APIError("get motion revision", GroupName, e)
			}
			return r.Msg, e
		},
		Head: func(ctx context.Context, id string) (proto.Message, error) {
			r, e := c.GetMotionHead(ctx, connect.NewRequest(&v1.GetMotionHeadRequest{MotionId: id}))
			if e != nil {
				return nil, componentrevisions.APIError("get motion head", GroupName, e)
			}
			return r.Msg, e
		},
		Revisions: func(ctx context.Context, id string, a, p uint32) (proto.Message, error) {
			r, e := c.ListMotionRevisions(ctx, connect.NewRequest(&v1.ListMotionRevisionsRequest{MotionId: id, AfterRevision: a, PageSize: p}))
			if e != nil {
				return nil, componentrevisions.APIError("list motion revisions", GroupName, e)
			}
			return r.Msg, e
		},
		Candidates: func(ctx context.Context, after string, p uint32) (proto.Message, error) {
			r, e := c.ListMotionCandidates(ctx, connect.NewRequest(&v1.ListMotionCandidatesRequest{AfterMotionId: after, PageSize: p}))
			if e != nil {
				return nil, componentrevisions.APIError("list motion candidates", GroupName, e)
			}
			return r.Msg, nil
		},
		CandidateRows: func(m proto.Message) ([]proto.Message, string) {
			r := m.(*v1.ListMotionCandidatesResponse)
			out := make([]proto.Message, 0, len(r.Candidates))
			for _, v := range r.Candidates {
				out = append(out, v)
			}
			return out, r.NextAfterMotionId
		},
		RevisionFromSave: func(m proto.Message) proto.Message { return m.(*v1.SaveMotionCandidateResponse).Revision }, RevisionFromGet: func(m proto.Message) proto.Message { return m.(*v1.GetMotionRevisionResponse).Revision }, RevisionFromHead: func(m proto.Message) proto.Message { return m.(*v1.GetMotionHeadResponse).Revision },
		RevisionRows: func(m proto.Message) ([]proto.Message, uint32) {
			r := m.(*v1.ListMotionRevisionsResponse)
			out := make([]proto.Message, 0, len(r.Revisions))
			for _, v := range r.Revisions {
				out = append(out, v)
			}
			return out, r.NextAfterRevision
		}, RevisionID: func(m proto.Message) string { return m.(*v1.MotionRevision).MotionId }, RevisionNumber: func(m proto.Message) uint32 { return m.(*v1.MotionRevision).Revision }, DescribeRevision: func(m proto.Message) string {
			r := m.(*v1.MotionRevision)
			return fmt.Sprintf("motion=%s revision=%d state=%s sha256=%s", r.MotionId, r.Revision, r.State, r.Sha256)
		}})
}
