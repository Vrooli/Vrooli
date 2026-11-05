package sketch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	"react-component-library/internal/designcapture"
	"react-component-library/internal/preview"
	internal "react-component-library/internal/sketch"
)

type captureRenderer struct{}

func (captureRenderer) RenderPrepared(_ context.Context, p preview.PreparedComposition, _ *previewv1.RenderCompositionRequest) (*connect.Response[previewv1.RenderCompositionResponse], error) {
	bindings, _ := json.Marshal(p.Bindings)
	html := "<main>exact fixture</main>" + string(bindings)
	digest := sha256.Sum256([]byte(html))
	hash := hex.EncodeToString(digest[:])
	return connect.NewResponse(&previewv1.RenderCompositionResponse{Html: html, RenderHash: hash, Target: &previewv1.CompositionRenderTarget{Revision: p.Composition.Revision, RenderHash: hash, HtmlSha256: hash, InputsSha256: hash, Kind: "preview", Kit: "vrooli-default", Theme: "light", Direction: "ltr"}}), nil
}

type captureDispatch struct {
	calls         int
	observations  int
	cancellations int
}

func (d *captureDispatch) Start(context.Context, designcapture.Operation) (string, error) {
	d.calls++
	return "bas-fixture", nil
}
func (d *captureDispatch) Cancel(context.Context, designcapture.Operation) error {
	d.cancellations++
	return nil
}
func (d *captureDispatch) ResolveScreenshot(_ context.Context, op designcapture.Operation, artifact designcapture.Artifact) (designcapture.Screenshot, error) {
	return designcapture.Screenshot{Reference: artifact.Reference, URL: "http://bas.local/image.png", Width: op.Request.Width, Height: op.Request.Height, ContentType: "image/png"}, nil
}
func (d *captureDispatch) Observe(_ context.Context, op designcapture.Operation) (designcapture.Observation, error) {
	d.observations++
	return designcapture.Observation{State: designcapture.Completed, Artifacts: []designcapture.Artifact{{Kind: "screenshot", Reference: "bas:fixture:screenshot"}, {Kind: "target", Reference: "bas:fixture:target", Evidence: &designcapture.TargetEvidence{RenderHash: op.Request.Target.RenderHash, Width: float64(op.Request.Width), Height: float64(op.Request.Height), Regions: []designcapture.RegionGeometry{{Region: "body", Width: 390, Height: 100}}}}}}, nil
}
func TestCaptureCandidateRequiresExactRenderAndPreservesRetry(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, []byte(`{"sketch":{}}`), 0600))
	store := internal.NewStore(root)
	base, err := store.Read("demo", "home")
	require.NoError(t, err)
	candidate, err := store.SaveCandidate("demo", "home", "draft", base.ContentHash, internal.Document{Template: &internal.AssetRef{Asset: "templates.page", Version: "1.0.0"}, Render: &internal.RenderSettings{TemplateExport: "Page", Bindings: map[string]any{"$template": map[string]any{}}}})
	require.NoError(t, err)
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(designcapture.Schema)))
	repo := designcapture.NewSQLiteRepository(db)
	dispatcher := &captureDispatch{}
	h := NewConnectHandler(Deps{Store: store, Renderer: captureRenderer{}, CapturesFor: func(context.Context) (designcapture.Repository, error) { return repo, nil }, CaptureDispatcher: dispatcher})
	render := &sketchv1.RenderCandidateRequest{Candidate: &sketchv1.CandidateReference{Scenario: "demo", DesignId: "draft", Hash: candidate.Hash}, MissingLabel: "Missing", FailedLabel: "Failed"}
	request := &sketchv1.CaptureCandidateRequest{Render: render, ExpectedRenderHash: "stale", IdempotencyKey: "capture-one", Width: 390, Height: 844}
	_, err = h.CaptureCandidate(ctx, connect.NewRequest(request))
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	require.Zero(t, dispatcher.calls)
	output, err := h.RenderCandidate(ctx, connect.NewRequest(render))
	require.NoError(t, err)
	request.ExpectedRenderHash = output.Msg.RenderHash
	op, err := h.CaptureCandidate(ctx, connect.NewRequest(request))
	require.NoError(t, err)
	require.Equal(t, "running", op.Msg.State)
	retry, err := h.CaptureCandidate(ctx, connect.NewRequest(request))
	require.NoError(t, err)
	require.Equal(t, op.Msg.Id, retry.Msg.Id)
	require.Equal(t, 1, dispatcher.calls)
	read, err := h.GetCapture(ctx, connect.NewRequest(&sketchv1.GetCaptureRequest{Id: op.Msg.Id}))
	require.NoError(t, err)
	require.Equal(t, "bas-fixture", read.Msg.ProducerId)
	pending, err := h.CancelCapture(ctx, connect.NewRequest(&sketchv1.GetCaptureRequest{Id: op.Msg.Id}))
	require.NoError(t, err)
	require.Equal(t, "cancel_requested", pending.Msg.State)
	require.Equal(t, "bas-fixture", pending.Msg.ProducerId)
	require.Equal(t, 1, dispatcher.cancellations)
	attached, err := h.AttachCapture(ctx, connect.NewRequest(&sketchv1.GetCaptureRequest{Id: op.Msg.Id}))
	require.NoError(t, err)
	require.Equal(t, "completed", attached.Msg.State)
	require.Equal(t, "body", attached.Msg.Artifacts[1].Evidence.Regions[0].Region)
	screenshot, err := h.GetCaptureScreenshot(ctx, connect.NewRequest(&sketchv1.GetCaptureScreenshotRequest{Id: op.Msg.Id, Reference: "bas:fixture:screenshot"}))
	require.NoError(t, err)
	require.Equal(t, "http://bas.local/image.png", screenshot.Msg.Url)
	require.Equal(t, int32(390), screenshot.Msg.Width)
	_, err = h.GetCaptureScreenshot(ctx, connect.NewRequest(&sketchv1.GetCaptureScreenshotRequest{Id: op.Msg.Id, Reference: "bas:other:screenshot"}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

	_, err = h.AttachCapture(ctx, connect.NewRequest(&sketchv1.GetCaptureRequest{Id: op.Msg.Id}))
	require.NoError(t, err)
	require.Equal(t, 1, dispatcher.observations)
	terminal, err := h.CancelCapture(ctx, connect.NewRequest(&sketchv1.GetCaptureRequest{Id: op.Msg.Id}))
	require.NoError(t, err)
	require.Equal(t, "completed", terminal.Msg.State)
	require.Equal(t, 1, dispatcher.cancellations)

	_, err = h.CaptureCandidate(apidb.WithTestMode(ctx), connect.NewRequest(request))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Equal(t, 1, dispatcher.calls)
}

func TestRetryCaptureHandlerRefusesTestModeAndUnknownOperation(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(designcapture.Schema)))
	repo := designcapture.NewSQLiteRepository(db)
	dispatcher := &captureDispatch{}
	h := NewConnectHandler(Deps{CapturesFor: func(context.Context) (designcapture.Repository, error) { return repo, nil }, CaptureDispatcher: dispatcher})
	request := connect.NewRequest(&sketchv1.RetryCaptureRequest{Id: "missing", IdempotencyKey: "new-attempt"})
	_, err := h.RetryCapture(apidb.WithTestMode(ctx), request)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	_, err = h.RetryCapture(ctx, request)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	require.Zero(t, dispatcher.calls)
}

func TestCaptureCandidateRejectsHashFromDifferentPreviewState(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, []byte(`{"sketch":{}}`), 0600))
	store := internal.NewStore(root)
	base, err := store.Read("demo", "home")
	require.NoError(t, err)
	candidate, err := store.SaveCandidate("demo", "home", "states", base.ContentHash, internal.Document{
		Template: &internal.AssetRef{Asset: "templates.page", Version: "1.0.0"},
		Render: &internal.RenderSettings{TemplateExport: "Page", Bindings: map[string]any{
			"$template": map[string]any{}, "$preview": map[string]any{
				"initial": "list", "states": map[string]any{"list": map[string]any{}, "detail": map[string]any{}}, "actions": map[string]any{},
			}}}})
	require.NoError(t, err)
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(designcapture.Schema)))
	repo := designcapture.NewSQLiteRepository(db)
	dispatcher := &captureDispatch{}
	h := NewConnectHandler(Deps{Store: store, Renderer: captureRenderer{}, CapturesFor: func(context.Context) (designcapture.Repository, error) { return repo, nil }, CaptureDispatcher: dispatcher})
	render := &sketchv1.RenderCandidateRequest{Candidate: &sketchv1.CandidateReference{Scenario: "demo", DesignId: "states", Hash: candidate.Hash}, MissingLabel: "Missing", FailedLabel: "Failed"}
	initial, err := h.RenderCandidate(ctx, connect.NewRequest(render))
	require.NoError(t, err)
	render.PreviewState = "detail"
	detail, err := h.RenderCandidate(ctx, connect.NewRequest(render))
	require.NoError(t, err)
	require.NotEqual(t, initial.Msg.RenderHash, detail.Msg.RenderHash)
	request := &sketchv1.CaptureCandidateRequest{Render: render, ExpectedRenderHash: initial.Msg.RenderHash, IdempotencyKey: "detail", Width: 390, Height: 844}
	_, err = h.CaptureCandidate(ctx, connect.NewRequest(request))
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	require.Zero(t, dispatcher.calls)
	request.ExpectedRenderHash = detail.Msg.RenderHash
	captured, err := h.CaptureCandidate(ctx, connect.NewRequest(request))
	require.NoError(t, err)
	require.Equal(t, detail.Msg.RenderHash, captured.Msg.Target.RenderHash)
	require.Equal(t, 1, dispatcher.calls)
	render.PreviewState = "absent"
	_, err = h.RenderCandidate(ctx, connect.NewRequest(render))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
