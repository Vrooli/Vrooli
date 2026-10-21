package shots

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	shotsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client shotsconnect.ShotsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: shotsconnect.NewShotsServiceClient(httpClient, base)}
}

func readSnapshot(path string) (*shotsv1.ShotSnapshot, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("snapshot is required; pass a proto-JSON file path or - for stdin")
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("read snapshot: %w", err)
	}
	snapshot := &shotsv1.ShotSnapshot{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, snapshot); err != nil {
		return nil, fmt.Errorf("parse typed ShotSnapshot proto JSON: %w", err)
	}
	return snapshot, nil
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*shotsv1.CreateShotResponse, error) {
	snapshot, err := readSnapshot(ctx.Flag("snapshot"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.CreateShot(context.Background(), connect.NewRequest(&shotsv1.CreateShotRequest{ProjectId: ctx.Flag("project-id"), Name: ctx.Flag("name"), Snapshot: snapshot}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create shot", err, nil)
	}
	if resp.Msg.Shot == nil || resp.Msg.Revision == nil {
		return nil, fmt.Errorf("server returned an incomplete shot response")
	}
	return resp.Msg, nil
}

func (h *handlers) createReport(_ cliapp.OperationContext, msg *shotsv1.CreateShotResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created shot %s at revision %d.", msg.Shot.Id, msg.Revision.Revision)}, Changes: []string{formatRevision(msg.Revision)}, NextCommand: []string{fmt.Sprintf("`shots get %s` — inspect this shot", msg.Shot.Id)}}
}

func (h *handlers) saveCall(ctx cliapp.OperationContext) (*shotsv1.SaveShotResponse, error) {
	revision, err := strconv.ParseInt(ctx.Flag("expected-revision"), 10, 64)
	if err != nil || revision <= 0 {
		return nil, fmt.Errorf("expected-revision must be a positive integer")
	}
	snapshot, err := readSnapshot(ctx.Flag("snapshot"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.SaveShot(context.Background(), connect.NewRequest(&shotsv1.SaveShotRequest{ShotId: ctx.Positional("id"), ExpectedRevision: revision, Name: ctx.Flag("name"), Snapshot: snapshot}))
	if err != nil {
		if detail, ok := shotConflict(err); ok {
			return nil, detail
		}
		return nil, cliapp.WrapAPIError("save shot", err, nil)
	}
	if resp.Msg.Shot == nil || resp.Msg.Revision == nil {
		return nil, fmt.Errorf("server returned an incomplete shot response")
	}
	return resp.Msg, nil
}

func (h *handlers) saveReport(_ cliapp.OperationContext, msg *shotsv1.SaveShotResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Saved shot %s at revision %d.", msg.Shot.Id, msg.Revision.Revision)}, Changes: []string{formatRevision(msg.Revision)}}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*shotsv1.ListShotsResponse, error) {
	size := int32(50)
	if raw := strings.TrimSpace(ctx.Flag("page-size")); raw != "" {
		n, e := strconv.ParseInt(raw, 10, 32)
		if e != nil {
			return nil, fmt.Errorf("invalid page-size: %w", e)
		}
		size = int32(n)
	}
	resp, err := h.client.ListShots(context.Background(), connect.NewRequest(&shotsv1.ListShotsRequest{ProjectId: ctx.Flag("project-id"), PageSize: size, PageToken: ctx.Flag("page-token")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list shots", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(ctx cliapp.OperationContext, msg *shotsv1.ListShotsResponse) cliapp.ListReport {
	rows := make([]string, 0, len(msg.Shots))
	for _, s := range msg.Shots {
		rows = append(rows, formatShot(s))
	}
	hints := []string{}
	if msg.NextPageToken != "" {
		hints = append(hints, fmt.Sprintf("`shots list --project-id %s --page-token %s` — next page", ctx.Flag("project-id"), msg.NextPageToken))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d shot(s).", len(rows))}, ResultsHeading: "Shots", Results: rows, RetrievalHints: hints}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*shotsv1.GetShotResponse, error) {
	resp, err := h.client.GetShot(context.Background(), connect.NewRequest(&shotsv1.GetShotRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get shot", err, nil)
	}
	if resp.Msg.Shot == nil || resp.Msg.Head == nil {
		return nil, fmt.Errorf("server returned an incomplete shot response")
	}
	return resp.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, msg *shotsv1.GetShotResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Opened shot."}, ResultsHeading: "Shot", Results: []string{formatShot(msg.Shot), formatRevision(msg.Head)}}
}

func (h *handlers) revisionsCall(ctx cliapp.OperationContext) (*shotsv1.ListShotRevisionsResponse, error) {
	size := int32(50)
	if raw := strings.TrimSpace(ctx.Flag("page-size")); raw != "" {
		n, e := strconv.ParseInt(raw, 10, 32)
		if e != nil {
			return nil, fmt.Errorf("invalid page-size: %w", e)
		}
		size = int32(n)
	}
	resp, err := h.client.ListShotRevisions(context.Background(), connect.NewRequest(&shotsv1.ListShotRevisionsRequest{ShotId: ctx.Positional("id"), PageSize: size, PageToken: ctx.Flag("page-token")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list shot revisions", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) revisionsReport(_ cliapp.OperationContext, msg *shotsv1.ListShotRevisionsResponse) cliapp.ListReport {
	rows := make([]string, 0, len(msg.Revisions))
	for _, r := range msg.Revisions {
		rows = append(rows, formatRevision(r))
	}
	hints := []string{}
	if msg.NextPageToken != "" {
		hints = append(hints, fmt.Sprintf("`shots revisions %s --page-token %s` — next page", msg.Revisions[0].ShotId, msg.NextPageToken))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d revision(s).", len(rows))}, ResultsHeading: "Shot revisions", Results: rows, RetrievalHints: hints}
}

func (h *handlers) revisionCall(ctx cliapp.OperationContext) (*shotsv1.GetShotRevisionResponse, error) {
	revision, e := strconv.ParseInt(ctx.Flag("revision"), 10, 64)
	if e != nil || revision <= 0 {
		return nil, fmt.Errorf("revision must be a positive integer")
	}
	resp, err := h.client.GetShotRevision(context.Background(), connect.NewRequest(&shotsv1.GetShotRevisionRequest{ShotId: ctx.Positional("id"), Revision: revision}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get shot revision", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) revisionReport(_ cliapp.OperationContext, msg *shotsv1.GetShotRevisionResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Loaded immutable revision."}, ResultsHeading: "Shot revision", Results: []string{formatRevision(msg.Revision)}}
}

func (h *handlers) compileCall(ctx cliapp.OperationContext) (*shotsv1.CompileShotResponse, error) {
	revision, e := strconv.ParseInt(ctx.Flag("revision"), 10, 64)
	if e != nil || revision <= 0 {
		return nil, fmt.Errorf("revision must be a positive integer")
	}
	resp, err := h.client.CompileShot(context.Background(), connect.NewRequest(&shotsv1.CompileShotRequest{ShotId: ctx.Positional("id"), Revision: revision}))
	if err != nil {
		return nil, cliapp.WrapAPIError("compile shot", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) compileReport(_ cliapp.OperationContext, msg *shotsv1.CompileShotResponse) cliapp.ListReport {
	if msg.Compiled == nil {
		return cliapp.ListReport{Summary: []string{"Server returned no compiled shot."}}
	}
	state := "render eligible"
	if !msg.Compiled.RenderEligible {
		state = "render blocked"
	}
	return cliapp.ListReport{Summary: []string{state}, ResultsHeading: "Compiled shot", Results: []string{
		fmt.Sprintf("%s revision=%d profile=%s recipe=%s", msg.Compiled.ShotId, msg.Compiled.Revision, msg.Compiled.CompositorProfile, msg.Compiled.RecipeSha256),
		fmt.Sprintf("scene=%s (%s) presentation=%s (%s)", msg.Compiled.SceneSha256, msg.Compiled.SceneVersion, msg.Compiled.PresentationSha256, msg.Compiled.PresentationVersion),
		formatIssues(msg.Compiled.Issues),
	}}
}

func (h *handlers) validateCall(ctx cliapp.OperationContext) (*shotsv1.ValidateShotResponse, error) {
	snapshot, e := readSnapshot(ctx.Flag("snapshot"))
	if e != nil {
		return nil, e
	}
	resp, err := h.client.ValidateShot(context.Background(), connect.NewRequest(&shotsv1.ValidateShotRequest{ProjectId: ctx.Flag("project-id"), Snapshot: snapshot}))
	if err != nil {
		return nil, cliapp.WrapAPIError("validate shot", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) validateReport(_ cliapp.OperationContext, msg *shotsv1.ValidateShotResponse) cliapp.ListReport {
	state := "render eligible"
	if !msg.RenderEligible {
		state = "render blocked"
	}
	return cliapp.ListReport{Summary: []string{state}, ResultsHeading: "Validation", Results: []string{formatIssues(msg.Issues)}}
}

func shotConflict(err error) (error, bool) {
	var ce *connect.Error
	if !errors.As(err, &ce) || ce.Code() != connect.CodeAborted {
		return nil, false
	}
	for _, detail := range ce.Details() {
		value, e := detail.Value()
		if e != nil {
			continue
		}
		if conflict, ok := value.(*shotsv1.ShotConflict); ok {
			return fmt.Errorf("shot %s revision conflict: expected %d, current %d", conflict.ShotId, conflict.ExpectedRevision, conflict.CurrentRevision), true
		}
	}
	return nil, false
}

func formatShot(s *shotsv1.Shot) string {
	if s == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s [revision=%d]", s.Id, s.Name, s.HeadRevision)
}

func formatRevision(r *shotsv1.ShotRevision) string {
	if r == nil {
		return "(nil)"
	}
	return fmt.Sprintf("revision=%d name=%q eligible=%t recipe=%s", r.Revision, r.Name, r.RenderEligible, r.RecipeSha256)
}

func formatIssues(issues []*shotsv1.ShotIssue) string {
	if len(issues) == 0 {
		return "No blocking issues."
	}
	parts := make([]string, 0, len(issues))
	for _, i := range issues {
		parts = append(parts, fmt.Sprintf("%s: %s", i.FieldPath, i.Summary))
	}
	return strings.Join(parts, "; ")
}

var _ io.Reader = (*os.File)(nil)
