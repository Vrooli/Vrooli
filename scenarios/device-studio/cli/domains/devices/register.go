package devices

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"device-studio/cli/domains/componentrevisions"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices/devices_v1connect"
	"google.golang.org/protobuf/proto"
)

const GroupName = "devices"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	hc, base := cliapp.NewConnectHTTPClient(core)
	c := api.NewDevicesServiceClient(hc, base)
	return componentrevisions.Register(core, manifest, componentrevisions.Spec{Group: GroupName, Service: "DevicesService", CandidateName: "Device", SaveMethod: "SaveDeviceCandidate", GetMethod: "GetDeviceRevision", HeadMethod: "GetDeviceHead", RevisionsMethod: "ListDeviceRevisions", CandidatesMethod: "ListDeviceCandidates", Candidate: func() proto.Message { return &v1.DeviceCandidate{} },
		Save: func(ctx context.Context, m proto.Message, n uint32) (proto.Message, error) {
			r, e := c.SaveDeviceCandidate(ctx, connect.NewRequest(&v1.SaveDeviceCandidateRequest{Candidate: m.(*v1.DeviceCandidate), ExpectedHead: n}))
			if e != nil {
				return nil, componentrevisions.APIError("save device candidate", GroupName, e)
			}
			return r.Msg, e
		},
		Get: func(ctx context.Context, id string, n uint32) (proto.Message, error) {
			r, e := c.GetDeviceRevision(ctx, connect.NewRequest(&v1.GetDeviceRevisionRequest{DeviceId: id, Revision: n}))
			if e != nil {
				return nil, componentrevisions.APIError("get device revision", GroupName, e)
			}
			return r.Msg, e
		},
		Head: func(ctx context.Context, id string) (proto.Message, error) {
			r, e := c.GetDeviceHead(ctx, connect.NewRequest(&v1.GetDeviceHeadRequest{DeviceId: id}))
			if e != nil {
				return nil, componentrevisions.APIError("get device head", GroupName, e)
			}
			return r.Msg, e
		},
		Revisions: func(ctx context.Context, id string, a, p uint32) (proto.Message, error) {
			r, e := c.ListDeviceRevisions(ctx, connect.NewRequest(&v1.ListDeviceRevisionsRequest{DeviceId: id, AfterRevision: a, PageSize: p}))
			if e != nil {
				return nil, componentrevisions.APIError("list device revisions", GroupName, e)
			}
			return r.Msg, e
		},
		Candidates: func(ctx context.Context, after string, p uint32) (proto.Message, error) {
			r, e := c.ListDeviceCandidates(ctx, connect.NewRequest(&v1.ListDeviceCandidatesRequest{AfterDeviceId: after, PageSize: p}))
			if e != nil {
				return nil, componentrevisions.APIError("list device candidates", GroupName, e)
			}
			return r.Msg, nil
		},
		CandidateRows: func(m proto.Message) ([]proto.Message, string) {
			r := m.(*v1.ListDeviceCandidatesResponse)
			out := make([]proto.Message, 0, len(r.Candidates))
			for _, v := range r.Candidates {
				out = append(out, v)
			}
			return out, r.NextAfterDeviceId
		},
		RevisionFromSave: func(m proto.Message) proto.Message { return m.(*v1.SaveDeviceCandidateResponse).Revision }, RevisionFromGet: func(m proto.Message) proto.Message { return m.(*v1.GetDeviceRevisionResponse).Revision }, RevisionFromHead: func(m proto.Message) proto.Message { return m.(*v1.GetDeviceHeadResponse).Revision },
		RevisionRows: func(m proto.Message) ([]proto.Message, uint32) {
			r := m.(*v1.ListDeviceRevisionsResponse)
			out := make([]proto.Message, 0, len(r.Revisions))
			for _, v := range r.Revisions {
				out = append(out, v)
			}
			return out, r.NextAfterRevision
		},
		RevisionID: func(m proto.Message) string { return m.(*v1.DeviceRevision).DeviceId }, RevisionNumber: func(m proto.Message) uint32 { return m.(*v1.DeviceRevision).Revision }, DescribeRevision: func(m proto.Message) string {
			r := m.(*v1.DeviceRevision)
			return fmt.Sprintf("device=%s revision=%d state=%s sha256=%s", r.DeviceId, r.Revision, r.State, r.Sha256)
		}})
}
