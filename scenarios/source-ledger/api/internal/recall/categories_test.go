package recall

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"source-ledger/internal/classification"
)

type categoryIndexFixture struct {
	categories []classification.Category
	matches    map[string][]string
}

func (f categoryIndexFixture) Categories(context.Context, string) ([]classification.Category, error) {
	return f.categories, nil
}

func (f categoryIndexFixture) MatchingEntries(context.Context, string, []string) (map[string][]string, error) {
	return f.matches, nil
}

type queryFixture func(string) ([]float64, error)

func (f queryFixture) EmbedQuery(_ context.Context, q string) ([]float64, error) {
	return f(q)
}

func TestCategoryRecallUsesSummaryAndKeepsGeneralRecallBroad(t *testing.T) {
	nodes := source{
		{ID: "setup", EntryID: "setup", Text: "Browser capture setup instructions", Vectors: [][]float64{{1, 0}}},
		{ID: "summary", Text: "Browser capture fixed locally. Remote browser capture still times out.", Summary: true, Vectors: [][]float64{{.8, .6}}},
		{ID: "first", EntryID: "first", ParentID: "summary", Text: "Browser capture is broken", Vectors: [][]float64{{.92, .38}}},
		{ID: "fix", EntryID: "fix", ParentID: "summary", Text: "Local capture fixed; remote capture still times out."},
		{ID: "printing", EntryID: "printing", Text: "Printer paper sensor has a defect"},
	}
	svc := NewService(nodes, queryFixture(func(q string) ([]float64, error) {
		assert.Contains(t, q, "browser capture")
		assert.NotContains(t, q, "bug reports")
		return []float64{1, 0}, nil
	}), Config{})
	svc.SetClassificationIndex(categoryIndexFixture{
		categories: []classification.Category{{ID: "defect", Name: "Defects", Aliases: []string{"bug reports"}}},
		matches:    map[string][]string{"first": {"defect"}, "printing": {"defect"}},
	})

	result, err := svc.Recall(context.Background(), "browser capture setup", 1)
	require.NoError(t, err)
	require.Equal(t, "setup", result.Hits[0].Node.ID)

	result, err = svc.Recall(context.Background(), "browser capture bug reports", 3)
	require.NoError(t, err)
	require.Equal(t, "summary", result.Hits[0].Node.ID)
	require.Equal(t, "setup", result.Hits[1].Node.ID)
	require.Equal(t, []string{"defect"}, result.Hits[0].Node.CategoryIDs)

	result, err = svc.RecallCategory(context.Background(), "browser capture", 1, "bug reports")
	require.NoError(t, err)
	require.Equal(t, "summary", result.Hits[0].Node.ID)
	constituents, err := svc.Zoom(context.Background(), result.Hits[0].Node.ID)
	require.NoError(t, err)
	require.Len(t, constituents, 2)
	require.Equal(t, "fix", constituents[1].ID)

	result, err = svc.RecallCategory(context.Background(), "", 10, "Defects")
	require.NoError(t, err)
	require.Len(t, result.Hits, 2)
	_, err = svc.RecallCategory(context.Background(), "browser capture", 10, "unknown")
	require.ErrorIs(t, err, ErrUnknownCategory)
}

func TestCategoryRecallOutageStillMatchesTheTopic(t *testing.T) {
	svc := NewService(source{
		{ID: "browser", EntryID: "browser", Text: "Browser capture is broken"},
		{ID: "printer", EntryID: "printer", Text: "Printer paper sensor has a defect"},
	}, queryFixture(func(string) ([]float64, error) {
		return nil, errors.New("embedding unavailable")
	}), Config{})
	svc.SetClassificationIndex(categoryIndexFixture{
		categories: []classification.Category{{ID: "defect", Name: "Defects", Aliases: []string{"bugs"}}},
		matches:    map[string][]string{"browser": {"defect"}, "printer": {"defect"}},
	})
	result, err := svc.Recall(context.Background(), "browser capture bugs", 10)
	require.NoError(t, err)
	require.True(t, result.Lexical)
	require.Len(t, result.Hits, 1)
	require.Equal(t, "browser", result.Hits[0].Node.ID)
}

func TestRecallDoesNotTurnAnUnverifiedMentionIntoAnOpenCase(t *testing.T) {
	svc := NewService(source{
		{ID: "observation", EntryID: "observation", Text: "Browser capture times out."},
		{ID: "summary", EntryID: "summary", Summary: true, Text: "Browser capture was discussed."},
		{ID: "case", EntryID: "case", CaseID: "defect-capture", Summary: true, Open: true, Text: "Remote browser capture still times out."},
	}, downEmbedder{}, Config{})
	svc.SetClassificationIndex(categoryIndexFixture{
		categories: []classification.Category{{ID: "defect-report", Name: "Bug reports", Aliases: []string{"bugs"}}},
		matches:    map[string][]string{"observation": {"defect-report"}, "summary": {"defect-report"}},
	})
	result, err := svc.Recall(t.Context(), "open browser capture bugs", 5)
	require.NoError(t, err)
	require.True(t, result.UnverifiedCaseMatches)
	require.Len(t, result.Hits, 1)
	require.Equal(t, "case", result.Hits[0].Node.ID)
	require.Equal(t, []string{"defect-report"}, result.Hits[0].Node.CategoryIDs)

	result, err = svc.Recall(t.Context(), "bug reports", 5)
	require.NoError(t, err)
	require.Len(t, result.Hits, 2, "category listings contain case accounts and observations")
}

func TestTopicalRecallFavorsFocusedSetupEvidenceOverDiffuseMentions(t *testing.T) {
	nodes := source{
		{ID: "technical-audit", Text: "Browser architecture and settings. " +
			strings.Repeat("Other work completed across the stack. ", 80) + "Audio capture and database setup."},
		{ID: "setup", Text: "Browser capture setup. Select the recorder configuration before recording."},
	}
	svc := NewService(nodes, downEmbedder{}, Config{})
	result, err := svc.Recall(t.Context(), "how do I set up browser capture", 5)
	require.NoError(t, err)
	require.NotEmpty(t, result.Hits)
	require.Equal(t, "setup", result.Hits[0].Node.ID)
}

func TestRecallPrefersSpecificCategoryPhrase(t *testing.T) {
	categories := []classification.Category{
		{ID: "defect-report", Name: "Bug reports", Aliases: []string{"bug"}},
		{ID: "resolution-update", Name: "Resolution updates", Aliases: []string{"bug fixes"}},
	}
	selection, err := resolveCategoryQuery("browser capture bug fixes", "", categories)
	require.NoError(t, err)
	require.Equal(t, []string{"resolution-update"}, selection.ids)
	require.Equal(t, "browser capture", selection.topic)
}

func TestSlowSemanticLegFallsBackToCategoryAwareTextRecall(t *testing.T) {
	svc := NewService(source{{ID: "first", EntryID: "first", Text: "Browser capture times out."}},
		queryFixture(func(string) ([]float64, error) {
			<-t.Context().Done()
			return nil, t.Context().Err()
		}), Config{})
	svc.SetClassificationIndex(categoryIndexFixture{
		categories: []classification.Category{{ID: "defect", Name: "Bug reports"}},
		matches:    map[string][]string{"first": {"defect"}},
	})
	result, err := svc.Recall(t.Context(), "browser capture bug reports", 5)
	require.NoError(t, err)
	require.True(t, result.Lexical)
	require.Len(t, result.Hits, 1)
	require.Equal(t, "first", result.Hits[0].Node.ID)
}
