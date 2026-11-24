package sketch

import (
	"context"
	"encoding/json"
	"testing"

	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"

	"react-component-library/internal/designinference"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	"google.golang.org/protobuf/proto"
)

func TestInferredAlternativesPreserveExactCatalogIdentityAndGroundQuotes(t *testing.T) {
	base := &sharedv1.ProposeSketchResponse{ContentHash: "page-hash", DesignSource: &sharedv1.DesignSourceSnapshot{Path: "DESIGN.md", ContentHash: "source-hash", Content: "Use dense rows. Preserve navigation."}, Candidates: []*sharedv1.DesignProposal{
		{Title: "Collection", Sketch: &sharedv1.Sketch{Template: &sharedv1.AssetReference{Asset: "templates.collection-page", Version: "1.3.0"}, Notes: []*sharedv1.SketchNote{{Scope: "page", Text: "Authored note"}}}},
		{Title: "Other", Sketch: &sharedv1.Sketch{Template: &sharedv1.AssetReference{Asset: "templates.other", Version: "2.0.0"}}},
	}}
	before := proto.Clone(base)
	request, err := inferenceRequest(&sharedv1.SketchTarget{Scenario: "demo", Page: "home"}, base)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(request.Schema)))
	require.Equal(t, &designinference.Scope{Scenario: "demo", Page: "home"}, request.Scope)
	response := &inferencev1.RunResponse{Validated: true, ValueJson: `{"alternatives":[{"candidateIndex":1,"rationale":"Offers a compact scan path.","requirements":[{"text":"Use compact rows for scanning.","sourceSection":"L1 Document"}]}]}`}
	got, err := inferredProposal(request, response)
	require.NoError(t, err)
	require.Len(t, got.Candidates, 1)
	require.Equal(t, "templates.other", got.Candidates[0].Sketch.Template.Asset)
	require.Equal(t, "2.0.0", got.Candidates[0].Sketch.Template.Version)
	require.Contains(t, got.Candidates[0].Sketch.Notes[0].Text, "Proposed design requirement:")
	require.True(t, proto.Equal(before, base), "inference mutated source candidates")
	for _, invalid := range []string{
		`{"alternatives":[{"candidateIndex":9,"rationale":"x","requirements":[]}]}`,
		`{"alternatives":[{"candidateIndex":0,"rationale":"x","requirements":[]},{"candidateIndex":0,"rationale":"y","requirements":[]}]}`,
		`{"alternatives":[{"candidateIndex":0,"rationale":"x","requirements":[{"text":"Invented","sourceSection":"Missing section"}]}]}`,
		`{"alternatives":[{"candidateIndex":0,"rationale":"x","requirements":[],"asset":"invented"}]}`,
		`{"alternatives":[{"rationale":"x","requirements":[]}]}`,
		`{"alternatives":[]}`,
	} {
		response.ValueJson = invalid
		_, err := inferredProposal(request, response)
		require.Error(t, err, invalid)
	}
}

func TestInferenceDispatchRefusesTestModeBeforeAnyGatewayOrRepositoryAccess(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.InferSketch(apidb.WithTestMode(context.Background()), connect.NewRequest(&sketchv1.InferSketchRequest{}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestSourceSectionsKeepExactSpansAndIgnoreFencedHeadings(t *testing.T) {
	content := "Preface\n## Layout\nUse dense rows.\n" + string([]byte{96, 96, 96}) + "\n## Example heading\n" + string([]byte{96, 96, 96}) + "\n## Layout\nPreserve context."
	sections := sourceSections(&sharedv1.ProposeSketchResponse{DesignSource: &sharedv1.DesignSourceSnapshot{Content: content}})
	require.Len(t, sections, 3)
	require.Equal(t, "L1 Document", sections[0].ID)
	require.Equal(t, "L2 Layout", sections[1].ID)
	require.Equal(t, 6, sections[1].EndLine)
	require.Contains(t, sections[1].Text, "## Example heading")
	require.Equal(t, "L7 Layout", sections[2].ID)
	require.Equal(t, 8, sections[2].EndLine)
	require.Equal(t, "## Layout\nPreserve context.", sections[2].Text)
}

func TestInferenceRecoveryByOriginalKeyDoesNotDispatch(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(designinference.Schema)))
	repo := designinference.NewSQLiteRepository(db)
	op, err := repo.Create(ctx, "lost-first-response", designinference.Request{Role: "extract.structured", MaxOutputTokens: 4096, Source: "source", Schema: `{"type":"object"}`, Instruction: "Extract", Context: []byte(`{}`)})
	require.NoError(t, err)
	h := NewConnectHandler(Deps{InferenceFor: func(context.Context) (designinference.Repository, error) { return repo, nil }})
	result, err := h.GetSketchInference(ctx, connect.NewRequest(&sketchv1.GetSketchInferenceRequest{IdempotencyKey: "lost-first-response"}))
	require.NoError(t, err)
	require.Equal(t, op.ID, result.Msg.Id)
	require.Equal(t, "prepared", result.Msg.State)
	_, err = h.GetSketchInference(ctx, connect.NewRequest(&sketchv1.GetSketchInferenceRequest{Id: op.ID, IdempotencyKey: "lost-first-response"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	_, err = h.GetSketchInference(ctx, connect.NewRequest(&sketchv1.GetSketchInferenceRequest{IdempotencyKey: "missing"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
