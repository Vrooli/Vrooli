package devices

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"device-studio/cli/domains/componentrevisions"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices/devices_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const GroupName = "devices"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	hc, base := cliapp.NewConnectHTTPClient(core)
	c := api.NewDevicesServiceClient(hc, base)
	g, err := componentrevisions.Register(core, manifest, componentrevisions.Spec{Group: GroupName, Service: "DevicesService", CandidateName: "Device", SaveMethod: "SaveDeviceCandidate", GetMethod: "GetDeviceRevision", HeadMethod: "GetDeviceHead", RevisionsMethod: "ListDeviceRevisions", CandidatesMethod: "ListDeviceCandidates", Candidate: func() proto.Message { return &v1.DeviceCandidate{} },
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
		}, AdditionalHandlers: map[string]cliapp.PrimitiveHandler{"review": reviewAction(c), "export-qualification-package": exportPackageAction(c), "import-qualification-package": importPackageAction(c)}})
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	return g, nil
}

func exportPackageAction(c api.DevicesServiceClient) cliapp.PrimitiveHandler {
	return cliapp.Action(func(ctx cliapp.OperationContext) (string, error) {
		revision, err := strconv.ParseUint(strings.TrimSpace(ctx.Flag("revision")), 10, 32)
		if err != nil || revision == 0 {
			return "", fmt.Errorf("revision must be a positive uint32")
		}
		projectID, deviceID, path := strings.TrimSpace(ctx.Flag("project-id")), strings.TrimSpace(ctx.Positional("id")), strings.TrimSpace(ctx.Flag("output-file"))
		if projectID == "" || deviceID == "" || path == "" {
			return "", fmt.Errorf("project-id, device id, and output-file are required")
		}
		response, err := c.ExportQualificationPackage(context.Background(), connect.NewRequest(&v1.ExportQualificationPackageRequest{ProjectId: projectID, DeviceId: deviceID, QualifiedRevision: uint32(revision)}))
		if err != nil {
			return "", componentrevisions.APIError("export qualification package", GroupName, err)
		}
		if len(response.Msg.PackageJson) == 0 || len(response.Msg.PackageJson) > 128<<20 {
			return "", fmt.Errorf("server returned an empty or oversized qualification package")
		}
		if err := os.WriteFile(path, response.Msg.PackageJson, 0o600); err != nil {
			return "", err
		}
		return path, nil
	}, func(_ cliapp.OperationContext, path string) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Wrote scoped qualification package to %s.", path)}, Changes: []string{"The package carries integrity hashes, not a reviewer signature."}}
	})
}

func importPackageAction(c api.DevicesServiceClient) cliapp.PrimitiveHandler {
	return cliapp.Action(func(ctx cliapp.OperationContext) (*v1.ReviewDeviceCandidateResponse, error) {
		data, err := os.ReadFile(strings.TrimSpace(ctx.Flag("package-file")))
		if err != nil {
			return nil, err
		}
		if len(data) == 0 || len(data) > 128<<20 {
			return nil, fmt.Errorf("package-file must contain 1 to 134217728 bytes")
		}
		head, err := strconv.ParseUint(strings.TrimSpace(ctx.Flag("expected-head")), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("expected-head must be a uint32")
		}
		response, err := c.ImportQualificationPackage(context.Background(), connect.NewRequest(&v1.ImportQualificationPackageRequest{PackageJson: data, ExpectedHead: uint32(head)}))
		if err != nil {
			return nil, componentrevisions.APIError("import qualification package", GroupName, err)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *v1.ReviewDeviceCandidateResponse) cliapp.MutationReport {
		revision := response.GetRevision()
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Transferred scoped review appended as device revision %d (%s).", revision.GetRevision(), revision.GetState())}, Changes: []string{fmt.Sprintf("preset=%s@%d; scope remains exact to the packaged setup", response.GetPresetId(), response.GetPresetRevision())}}
	})
}

func reviewAction(c api.DevicesServiceClient) cliapp.PrimitiveHandler {
	return cliapp.Action(func(ctx cliapp.OperationContext) (*v1.ReviewDeviceCandidateResponse, error) {
		data, err := os.ReadFile(ctx.Flag("review-file"))
		if err != nil {
			return nil, err
		}
		if len(data) == 0 || len(data) > 64<<10 {
			return nil, fmt.Errorf("review-file must contain 1 to 65536 bytes")
		}
		review := &v1.QualificationReview{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, review); err != nil {
			return nil, fmt.Errorf("decode qualification review: %w", err)
		}
		revisionValue, err := strconv.ParseUint(strings.TrimSpace(ctx.Flag("revision")), 10, 32)
		if err != nil || revisionValue == 0 {
			return nil, fmt.Errorf("revision must be a positive uint32")
		}
		headValue, err := strconv.ParseUint(strings.TrimSpace(ctx.Flag("expected-head")), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("expected-head must be a uint32")
		}
		id, projectID := strings.TrimSpace(ctx.Positional("id")), strings.TrimSpace(ctx.Flag("project-id"))
		if id == "" || projectID == "" || strings.TrimSpace(ctx.Flag("sha256")) == "" {
			return nil, fmt.Errorf("device id, candidate digest, and project-id are required")
		}
		response, err := c.ReviewDeviceCandidate(context.Background(), connect.NewRequest(&v1.ReviewDeviceCandidateRequest{DeviceId: id, CandidateRevision: uint32(revisionValue), CandidateSha256: strings.TrimSpace(ctx.Flag("sha256")), ExpectedHead: uint32(headValue), Review: review, ProjectId: projectID}))
		if err != nil {
			return nil, componentrevisions.APIError("review device candidate", GroupName, err)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *v1.ReviewDeviceCandidateResponse) cliapp.MutationReport {
		revision := response.GetRevision()
		changes := []string{"review decision recorded; candidate remains unqualified"}
		if response.GetPresetId() != "" {
			changes = []string{fmt.Sprintf("qualified setup %s revision %d", response.GetPresetId(), response.GetPresetRevision())}
		}
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("device revision %d state=%s sha256=%s", revision.GetRevision(), revision.GetState(), revision.GetSha256())}, Changes: changes, NextCommand: []string{fmt.Sprintf("device-studio devices get %s --revision %d", revision.GetDeviceId(), revision.GetRevision())}}
	})
}
