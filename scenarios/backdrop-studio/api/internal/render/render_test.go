package render

import (
	"backdrop-studio/internal/catalog"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

type fakeExecutor struct{ calls int }

func (f *fakeExecutor) Apply(_ context.Context, input []byte, treatments []string, _ map[string]string) ([]byte, error) {
	f.calls++
	if len(treatments) == 0 {
		return nil, fmt.Errorf("expected treatment chain")
	}
	return append(append([]byte(nil), input...), 0x42), nil
}

func TestSubmitIsReproducibleAndRequiresSelection(t *testing.T) {
	store := NewStore(&fakeExecutor{})
	style := catalog.Style{ID: "horizon", Strategy: "procedural-treated", Subject: "horizon", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"}}
	a, err := store.Submit(style, "full_bleed", 7, 1)
	require.NoError(t, err)
	b, err := store.Submit(style, "full_bleed", 7, 1)
	require.NoError(t, err)
	require.Equal(t, a.Candidates[0].PNG, b.Candidates[0].PNG)
	require.Empty(t, a.SelectedCandidateID)
	_, err = store.Select(a.ID, a.Candidates[0].ID, "operator")
	require.NoError(t, err)
}
