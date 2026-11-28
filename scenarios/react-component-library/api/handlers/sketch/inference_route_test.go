package sketch

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	"react-component-library/internal/designinference"
	internal "react-component-library/internal/sketch"
)

func TestInferSketchMapsMissingValidatedAlternativesWithoutWritingPage(t *testing.T) {
	root := t.TempDir()
	pagePath := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(pagePath), 0o700))
	require.NoError(t, os.WriteFile(pagePath, []byte(`{"sketch":{"viewport":"desktop"}}`), 0o600))
	store := internal.NewStore(root)
	before, err := store.Read("demo", "home")
	require.NoError(t, err)
	h := NewConnectHandler(Deps{Store: store, ImportSearch: emptyCatalogSearch{}, InferenceFor: func(context.Context) (designinference.Repository, error) {
		t.Fatal("inference repository should not be acquired without validated candidates")
		return nil, nil
	}})
	request := connect.NewRequest(&sketchv1.InferSketchRequest{
		Proposal: &sharedv1.ProposeSketchRequest{
			Target:              &sharedv1.SketchTarget{Scenario: "demo", Page: "home"},
			ExpectedContentHash: before.ContentHash,
			CandidateLimit:      1,
			Intent: &sharedv1.DesignIntent{
				Intent: "Help an operator review current activity", Users: []string{"Operator"},
				PrimaryTasks: []string{"Review activity"}, Target: "react-vite", Kit: "vrooli-default", Viewports: []string{"desktop"},
			},
		},
		IdempotencyKey: "proposal-without-matches",
	})
	_, err = h.InferSketch(context.Background(), request)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), err)
	require.Contains(t, err.Error(), "one to three validated catalog alternatives")
	after, readErr := store.Read("demo", "home")
	require.NoError(t, readErr)
	require.Equal(t, before.ContentHash, after.ContentHash, "inference preparation must not mutate the authored page")
}

func TestInferSketchRejectsTestRoutingBeforePreparingProposal(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.InferSketch(apidb.WithTestMode(context.Background()), connect.NewRequest(&sketchv1.InferSketchRequest{}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Contains(t, err.Error(), "test-routing support")
}
