package sketch

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	databasetest "github.com/vrooli/api-core/databasetest"
	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	"react-component-library/internal/designcapture"
	"react-component-library/internal/designcritique"
	"react-component-library/internal/preview"
	internal "react-component-library/internal/sketch"
)

type acceptanceRendererStub struct {
	response *sharedv1.RenderCompositionResponse
	err      error
}

func (r acceptanceRendererStub) RenderPrepared(context.Context, preview.PreparedComposition, *previewv1.RenderCompositionRequest) (*connect.Response[sharedv1.RenderCompositionResponse], error) {
	if r.err != nil {
		return nil, r.err
	}
	return connect.NewResponse(r.response), nil
}

func acceptanceHandlerFixture(t *testing.T, renderer CompositionRenderer, mutate func(*Deps)) (*connectHandler, *internal.Store, *sharedv1.CandidateReference) {
	t.Helper()
	root := t.TempDir()
	pagePath := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(pagePath), 0o700))
	require.NoError(t, os.WriteFile(pagePath, []byte(`{"sketch":{}}`), 0o600))
	store := internal.NewStore(root)
	base, err := store.Read("demo", "home")
	require.NoError(t, err)
	doc := internal.Document{
		Template: &internal.AssetRef{Asset: "templates.collection-page", Version: "1.7.0"},
		Regions:  []internal.Region{{ID: "content"}},
		Render:   &internal.RenderSettings{TemplateExport: "CollectionPage", Regions: []internal.RenderRegion{{ID: "content", Slot: []string{"content"}}}, Bindings: map[string]any{"$template": map[string]any{}}},
	}
	candidate, err := store.SaveCandidate("demo", "home", "design-one", base.ContentHash, doc)
	require.NoError(t, err)
	deps := Deps{Store: store, Renderer: renderer}
	if mutate != nil {
		mutate(&deps)
	}
	return NewConnectHandler(deps), store, &sharedv1.CandidateReference{Scenario: "demo", DesignId: candidate.DesignID, Hash: candidate.Hash}
}

func acceptanceRender(ref *sharedv1.CandidateReference) *sharedv1.RenderCandidateRequest {
	return &sharedv1.RenderCandidateRequest{Candidate: ref, MissingLabel: "Missing", FailedLabel: "Failed", Kit: "vrooli-default", Theme: "light", Direction: "ltr"}
}

func acceptanceRendererResponse(gaps ...*sharedv1.CompositionGap) *sharedv1.RenderCompositionResponse {
	return &sharedv1.RenderCompositionResponse{
		RenderHash: "render-current",
		Target:     &sharedv1.CompositionRenderTarget{InputHashes: map[string]string{"appearance": "appearance-hash", "bundle": "bundle-hash"}},
		Bundle:     &sharedv1.GetCompositionBundleResponse{Gaps: gaps},
	}
}

type acceptanceCaptureRepositoryStub struct {
	designcapture.Repository
	operation designcapture.Operation
	err       error
}

func (r acceptanceCaptureRepositoryStub) Get(context.Context, string) (designcapture.Operation, error) {
	return r.operation, r.err
}

type acceptanceScreenshotResolverStub struct{ url string }

func (r acceptanceScreenshotResolverStub) ResolveScreenshot(_ context.Context, op designcapture.Operation, _ designcapture.Artifact) (designcapture.Screenshot, error) {
	return designcapture.Screenshot{URL: r.url, Width: op.Request.Width, Height: op.Request.Height, ContentType: "image/png"}, nil
}

type acceptanceCritiqueRepo struct {
	store *databasetest.SliceRepo[designcritique.Record]
}

func (acceptanceCritiqueRepo) Create(context.Context, string, designcritique.Review, designcritique.Assessment) (designcritique.Record, error) {
	return designcritique.Record{}, errors.New("unexpected create")
}

func (acceptanceCritiqueRepo) List(context.Context, designcritique.Target, string) ([]designcritique.Record, string, error) {
	return nil, "", errors.New("unexpected list")
}

func (r acceptanceCritiqueRepo) Get(ctx context.Context, id string) (designcritique.Record, error) {
	return r.store.Get(ctx, id)
}

func acceptanceCritiqueRepository(record designcritique.Record, err error) *acceptanceCritiqueRepo {
	repo := databasetest.NewSliceRepo(
		func(record designcritique.Record) string { return record.ID },
		func(record *designcritique.Record, id string) { record.ID = id },
		func(id string) error { return sql.ErrNoRows },
	)
	record.ID = "review-1"
	repo.Items = []designcritique.Record{record}
	repo.GetErr = err
	return &acceptanceCritiqueRepo{store: repo}
}

func acceptanceRequirement(t *testing.T, result *reviewv1.CandidateAcceptanceCheck, code string) *reviewv1.AcceptanceRequirement {
	t.Helper()
	for _, requirement := range result.GetRequirements() {
		if requirement.GetCode() == code {
			return requirement
		}
	}
	t.Fatalf("acceptance requirement %q missing from %+v", code, result.GetRequirements())
	return nil
}

func TestValidateAcceptanceCandidateRequiresAndResolvesImmutableReference(t *testing.T) {
	h, _, ref := acceptanceHandlerFixture(t, acceptanceRendererStub{response: acceptanceRendererResponse()}, nil)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(h.ValidateAcceptanceCandidate(context.Background(), nil)))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(h.ValidateAcceptanceCandidate(context.Background(), &reviewv1.CheckCandidateAcceptanceRequest{Render: &sharedv1.RenderCandidateRequest{}})))
	require.NoError(t, h.ValidateAcceptanceCandidate(context.Background(), &reviewv1.CheckCandidateAcceptanceRequest{Render: acceptanceRender(ref)}))
}

func TestEvaluateAcceptanceReportsCurrentPageRenderAndExplicitUnmetProof(t *testing.T) {
	h, _, ref := acceptanceHandlerFixture(t, acceptanceRendererStub{response: acceptanceRendererResponse()}, nil)
	result, err := h.EvaluateAcceptance(context.Background(), &reviewv1.CheckCandidateAcceptanceRequest{Render: acceptanceRender(ref), ExpectedRenderHash: "render-current"})
	require.NoError(t, err)
	require.Equal(t, internal.AcceptancePolicy, result.GetPolicyVersion())
	require.Equal(t, "render-current", result.GetRenderHash())
	require.Equal(t, "appearance-hash", result.GetInputHashes()["appearance"])
	require.Equal(t, "passed", acceptanceRequirement(t, result, "page_freshness").GetStatus())
	require.Equal(t, "passed", acceptanceRequirement(t, result, "render_freshness").GetStatus())
	require.Equal(t, "passed", acceptanceRequirement(t, result, "render_completeness").GetStatus())
	require.Equal(t, "blocked", acceptanceRequirement(t, result, "visual_review").GetStatus())
	require.Equal(t, "unavailable", acceptanceRequirement(t, result, "behavior_evidence").GetStatus())
	require.Equal(t, "unavailable", acceptanceRequirement(t, result, "rubric_calibration").GetStatus())
	require.Equal(t, "unavailable", acceptanceRequirement(t, result, "independent_review").GetStatus())
}

func TestEvaluateAcceptanceBlocksStalePageRenderHashAndRequiredGaps(t *testing.T) {
	for _, tc := range []struct {
		name       string
		changePage bool
		expected   string
		gaps       []*sharedv1.CompositionGap
		wantPage   string
		wantRender string
		wantBundle string
	}{
		{name: "page changed", changePage: true, expected: "render-current", wantPage: "blocked", wantRender: "passed", wantBundle: "passed"},
		{name: "render changed", expected: "render-reviewed", wantPage: "passed", wantRender: "blocked", wantBundle: "passed"},
		{name: "required region unresolved", expected: "render-current", gaps: []*sharedv1.CompositionGap{{Region: "content", Code: "missing_asset", Required: true}, {Region: "optional", Code: "empty", Required: false}}, wantPage: "passed", wantRender: "passed", wantBundle: "blocked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, store, ref := acceptanceHandlerFixture(t, acceptanceRendererStub{response: acceptanceRendererResponse(tc.gaps...)}, nil)
			if tc.changePage {
				current, err := store.Read("demo", "home")
				require.NoError(t, err)
				_, err = store.Save("demo", "home", current.ContentHash, internal.Document{Viewport: "phone"})
				require.NoError(t, err)
			}
			result, err := h.EvaluateAcceptance(context.Background(), &reviewv1.CheckCandidateAcceptanceRequest{Render: acceptanceRender(ref), ExpectedRenderHash: tc.expected})
			require.NoError(t, err)
			require.Equal(t, tc.wantPage, acceptanceRequirement(t, result, "page_freshness").GetStatus())
			require.Equal(t, tc.wantRender, acceptanceRequirement(t, result, "render_freshness").GetStatus())
			require.Equal(t, tc.wantBundle, acceptanceRequirement(t, result, "render_completeness").GetStatus())
		})
	}
}

func TestEvaluateAcceptanceReportsRendererAndVisualOwnerAvailability(t *testing.T) {
	t.Run("renderer unavailable", func(t *testing.T) {
		h, _, ref := acceptanceHandlerFixture(t, nil, nil)
		result, err := h.EvaluateAcceptance(context.Background(), &reviewv1.CheckCandidateAcceptanceRequest{Render: acceptanceRender(ref), ExpectedRenderHash: "reviewed"})
		require.NoError(t, err)
		require.Equal(t, "blocked", acceptanceRequirement(t, result, "render_completeness").GetStatus())
		require.Equal(t, "unavailable", acceptanceRequirement(t, result, "render_freshness").GetStatus())
		require.Contains(t, acceptanceRequirement(t, result, "render_completeness").GetDetail(), "composition renderer unavailable")
	})
	t.Run("critique requested without capture verifier", func(t *testing.T) {
		h, _, ref := acceptanceHandlerFixture(t, acceptanceRendererStub{response: acceptanceRendererResponse()}, nil)
		result, err := h.EvaluateAcceptance(context.Background(), &reviewv1.CheckCandidateAcceptanceRequest{Render: acceptanceRender(ref), ExpectedRenderHash: "render-current", CritiqueIds: []string{"review-1"}})
		require.NoError(t, err)
		require.Equal(t, "unavailable", acceptanceRequirement(t, result, "visual_review").GetStatus())
	})
}

func TestEvaluateAcceptanceRejectsInvalidAndDuplicateCritiqueIDs(t *testing.T) {
	h, _, ref := acceptanceHandlerFixture(t, acceptanceRendererStub{response: acceptanceRendererResponse()}, func(deps *Deps) {
		deps.CaptureDispatcher = screenshotDispatchStub{}
		deps.CritiquesFor = func(context.Context) (designcritique.Repository, error) {
			return acceptanceCritiqueRepository(designcritique.Record{}, errors.New("review missing")), nil
		}
		deps.CapturesFor = func(context.Context) (designcapture.Repository, error) { return nil, nil }
	})
	for _, ids := range [][]string{{""}, {"review-1", "review-1"}} {
		_, err := h.EvaluateAcceptance(context.Background(), &reviewv1.CheckCandidateAcceptanceRequest{Render: acceptanceRender(ref), ExpectedRenderHash: "render-current", CritiqueIds: ids})
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
	_, err := h.EvaluateAcceptance(context.Background(), &reviewv1.CheckCandidateAcceptanceRequest{Render: acceptanceRender(ref), ExpectedRenderHash: "render-current", CritiqueIds: make([]string, 17)})
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestEvaluateAcceptanceMapsProducerFailuresAndRejectsInvalidRequests(t *testing.T) {
	h, _, ref := acceptanceHandlerFixture(t, acceptanceRendererStub{response: acceptanceRendererResponse()}, nil)
	for _, req := range []*reviewv1.CheckCandidateAcceptanceRequest{
		{},
		{Render: acceptanceRender(ref)},
		{Render: acceptanceRender(ref), ExpectedRenderHash: "", CritiqueIds: []string{}},
	} {
		_, err := h.EvaluateAcceptance(context.Background(), req)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
	h, _, ref = acceptanceHandlerFixture(t, acceptanceRendererStub{response: acceptanceRendererResponse()}, func(deps *Deps) {
		deps.StoreFor = func(context.Context) (internal.Repository, error) {
			return nil, errors.New("scenario store unavailable")
		}
	})
	_, err := h.EvaluateAcceptance(context.Background(), &reviewv1.CheckCandidateAcceptanceRequest{Render: acceptanceRender(ref), ExpectedRenderHash: "render-current"})
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestEvaluateAcceptanceFailsClosedAcrossVisualReviewOwnerBoundaries(t *testing.T) {
	validDispatcher := screenshotDispatchStub{}
	for _, tc := range []struct {
		name        string
		critiqueErr error
		captureErr  error
		record      designcritique.Record
		wantCode    uint32
		wantVisual  string
	}{
		{name: "critique repository unavailable", critiqueErr: errors.New("critique store down"), wantCode: uint32(connect.CodeUnavailable)},
		{name: "capture repository unavailable", captureErr: errors.New("capture store down"), wantCode: uint32(connect.CodeUnavailable)},
		{name: "review missing", record: designcritique.Record{}, wantVisual: "blocked"},
		{name: "review targets different render", record: designcritique.Record{Review: designcritique.Review{Target: designcritique.Target{Scenario: "demo", DesignID: "another-design", Revision: "different", RenderHash: "different"}}}, wantVisual: "blocked"},
		{name: "invalid review evidence contract", record: designcritique.Record{Review: designcritique.Review{Target: designcritique.Target{Scenario: "demo", DesignID: "design-one"}}}, wantVisual: "blocked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _, ref := acceptanceHandlerFixture(t, acceptanceRendererStub{response: acceptanceRendererResponse()}, func(deps *Deps) {
				deps.CaptureDispatcher = validDispatcher
				deps.CritiquesFor = func(context.Context) (designcritique.Repository, error) {
					if tc.critiqueErr != nil {
						return nil, tc.critiqueErr
					}
					return acceptanceCritiqueRepository(tc.record, nil), nil
				}
				deps.CapturesFor = func(context.Context) (designcapture.Repository, error) {
					if tc.captureErr != nil {
						return nil, tc.captureErr
					}
					return nil, nil
				}
			})
			result, err := h.EvaluateAcceptance(context.Background(), &reviewv1.CheckCandidateAcceptanceRequest{Render: acceptanceRender(ref), ExpectedRenderHash: "render-current", CritiqueIds: []string{"review-1"}})
			if tc.wantCode != 0 {
				require.Equal(t, tc.wantCode, uint32(connect.CodeOf(err)))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantVisual, acceptanceRequirement(t, result, "visual_review").GetStatus())
		})
	}
}

func TestEvaluateAcceptancePassesRevalidatedVisualReviewEvidence(t *testing.T) {
	var imageBytes bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	img.Set(50, 50, color.RGBA{R: 20, G: 80, B: 160, A: 255})
	require.NoError(t, png.Encode(&imageBytes, img))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(imageBytes.Bytes())
	}))
	defer server.Close()

	renderHash := strings.Repeat("a", 64)
	renderResponse := acceptanceRendererResponse()
	renderResponse.RenderHash = renderHash
	h, _, ref := acceptanceHandlerFixture(t, acceptanceRendererStub{response: renderResponse}, nil)
	target := designcritique.Target{Scenario: ref.GetScenario(), DesignID: ref.GetDesignId(), Revision: ref.GetHash(), RenderHash: renderHash}
	evidence := designcritique.Evidence{RenderHash: renderHash, CaptureID: "capture-1", Artifact: "shot-1", Region: "$page", State: "default", Width: 100, Height: 100}
	review := designcritique.Review{Target: target, RubricVersion: designcritique.RubricVersion, PolicyVersion: designcritique.PolicyVersion, Critic: designcritique.Critic{Kind: "human", ID: "reviewer-1", Version: "review-process/1"}}
	for _, dimension := range designcritique.Dimensions() {
		review.Ratings = append(review.Ratings, designcritique.Rating{Dimension: dimension, Score: 4, Rationale: "Evidence supports the accepted visual floor.", Evidence: []designcritique.Evidence{evidence}})
	}
	operation := designcapture.Operation{
		ID: "capture-1", State: designcapture.Completed, ProducerID: "producer-1",
		Request:   designcapture.Request{Target: designcapture.Target{Scenario: target.Scenario, DesignID: target.DesignID, Revision: target.Revision, RenderHash: target.RenderHash}, Width: 100, Height: 100},
		Artifacts: []designcapture.Artifact{{Kind: "screenshot", Reference: "shot-1"}},
	}
	h.deps.CritiquesFor = func(context.Context) (designcritique.Repository, error) {
		return acceptanceCritiqueRepository(designcritique.Record{ID: "review-1", Hash: "review-hash", Review: review}, nil), nil
	}
	h.deps.CapturesFor = func(context.Context) (designcapture.Repository, error) {
		return acceptanceCaptureRepositoryStub{operation: operation}, nil
	}
	// The capture owner supplies screenshot bytes through its artifact resolver.
	h.deps.CaptureDispatcher = acceptanceDispatcherWithScreenshot{url: server.URL}
	result, err := h.EvaluateAcceptance(context.Background(), &reviewv1.CheckCandidateAcceptanceRequest{Render: acceptanceRender(ref), ExpectedRenderHash: renderHash, CritiqueIds: []string{"review-1"}})
	require.NoError(t, err)
	visual := acceptanceRequirement(t, result, "visual_review")
	require.Equal(t, "passed", visual.GetStatus(), visual.GetDetail())
	require.Contains(t, visual.GetEvidenceIds(), "review-hash")
}

type acceptanceDispatcherWithScreenshot struct{ url string }

func (acceptanceDispatcherWithScreenshot) Start(context.Context, designcapture.Operation) (string, error) {
	return "unused", nil
}

func (d acceptanceDispatcherWithScreenshot) ResolveScreenshot(_ context.Context, op designcapture.Operation, _ designcapture.Artifact) (designcapture.Screenshot, error) {
	return acceptanceScreenshotResolverStub(d).ResolveScreenshot(context.Background(), op, designcapture.Artifact{})
}
