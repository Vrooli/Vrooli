package categories

import (
	"context"
	"errors"
	"testing"
	"time"

	localdb "signal-inbox/internal/database"
	"signal-inbox/internal/inference"
	"signal-inbox/internal/signals"
	"signal-inbox/internal/testutil/mocks"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/scheduletest"
)

func newCategoryService(t *testing.T, fake *mocks.FakeInference) (*Service, signals.Service) {
	t.Helper()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(signals.Schema),
		apidb.SchemaProviderFunc(Schema),
	))
	clk := scheduletest.New(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	return NewService(NewSQLiteRepository(database), clk, fake), signals.NewService(signals.NewSQLiteRepository(database, clk), clk)
}

func captureText(t *testing.T, svc signals.Service) signals.Signal {
	t.Helper()
	result, err := svc.Capture(context.Background(), signals.CaptureInput{Text: "durable category evidence"})
	require.NoError(t, err)
	return result.Signal
}

// [REQ:SIG-P0-004] Categories are operator-created runtime data; only the
// required reserved fallback has a fixed identity.
func TestCategoryCreateRenameAndReservedProtection(t *testing.T) {
	t.Log("[REQ:SIG-P0-004]")
	svc, _ := newCategoryService(t, &mocks.FakeInference{})
	reserved, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)
	created, err := svc.Create(context.Background(), "Future work", "Things to review")
	require.NoError(t, err)
	renamed, err := svc.Rename(context.Background(), created.ID, "Future directions", "Updated operator definition")
	require.NoError(t, err)
	require.Equal(t, "Future directions", renamed.Name)
	_, err = svc.Rename(context.Background(), reserved.ID, "anything", "")
	require.ErrorAs(t, err, new(ErrReservedCategory))
	_, err = svc.Retire(context.Background(), reserved.ID)
	require.ErrorAs(t, err, new(ErrReservedCategory))
}

// [REQ:SIG-P0-005] A proposal is not a confirmed assignment. Confirmation
// appends a second classification record, retaining the model proposal.
func TestProposalDoesNotBecomeConfirmedUntilOperatorConfirms(t *testing.T) {
	t.Log("[REQ:SIG-P0-005]")
	fake := &mocks.FakeInference{}
	svc, journal := newCategoryService(t, fake)
	_, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)
	category, err := svc.Create(context.Background(), "Research queue", "Evidence worth investigating")
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), "Personal", "Personal notes")
	require.NoError(t, err)
	fake.ClassifyOut = inference.ClassificationResult{CategoryID: category.ID, Confidence: 0.82, Model: "decision-model"}
	signal := captureText(t, journal)
	require.NoError(t, svc.Enrich(context.Background(), signal))
	require.Equal(t, signal.ExtractedContent, fake.ClassifyInput)
	require.Len(t, fake.ClassifyChoices, 2)
	var includedCategory bool
	for _, choice := range fake.ClassifyChoices {
		if choice.ID == category.ID && choice.Description == "Research queue: Evidence worth investigating" {
			includedCategory = true
		}
	}
	require.True(t, includedCategory, "active category ID and definition must be options in the closed-set request")
	proposal, found, err := svc.GetClassification(context.Background(), signal.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, StateProposed, proposal.State)
	require.Empty(t, proposal.ConfirmedCategoryID)

	confirmed, err := svc.Confirm(context.Background(), signal.ID, category.ID)
	require.NoError(t, err)
	require.Equal(t, StateConfirmed, confirmed.State)
	require.Equal(t, category.ID, confirmed.ConfirmedCategoryID)
	require.Equal(t, proposal.ProposedCategoryID, confirmed.ProposedCategoryID)
	require.Equal(t, proposal.ProposedConfidence, confirmed.ProposedConfidence)
}

// [REQ:SIG-P0-004] [REQ:SIG-P0-005] Retiring a category preserves the signal
// and appends an override to the reserved fallback instead of deleting either.
func TestRetireReassignsConfirmedSignalsWithoutTouchingJournal(t *testing.T) {
	t.Log("[REQ:SIG-P0-004] [REQ:SIG-P0-005]")
	fake := &mocks.FakeInference{}
	svc, journal := newCategoryService(t, fake)
	reserved, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)
	category, err := svc.Create(context.Background(), "Temporary focus", "Short-lived category")
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), "Other", "Another category")
	require.NoError(t, err)
	fake.ClassifyOut = inference.ClassificationResult{CategoryID: category.ID, Confidence: 0.9, Model: "decision-model"}
	signal := captureText(t, journal)
	require.NoError(t, svc.Enrich(context.Background(), signal))
	_, err = svc.Confirm(context.Background(), signal.ID, category.ID)
	require.NoError(t, err)

	retired, err := svc.Retire(context.Background(), category.ID)
	require.NoError(t, err)
	require.False(t, retired.Active())
	classification, found, err := svc.GetClassification(context.Background(), signal.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, StateOverridden, classification.State)
	require.Equal(t, reserved.ID, classification.ConfirmedCategoryID)
	stored, err := journal.Get(context.Background(), signal.ID)
	require.NoError(t, err)
	require.Equal(t, signal.ContentHash, stored.ContentHash)
}

func TestInferenceFailureRecordsUncategorizedAndQueue(t *testing.T) {
	fake := &mocks.FakeInference{ClassifyErr: errors.New("gateway unavailable")}
	svc, journal := newCategoryService(t, fake)
	reserved, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), "Available category", "Exists so inference is attempted")
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), "Another category", "Second choice")
	require.NoError(t, err)
	signal := captureText(t, journal)
	require.NoError(t, svc.Enrich(context.Background(), signal))
	classification, found, err := svc.GetClassification(context.Background(), signal.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, StateUncategorized, classification.State)
	require.Equal(t, reserved.ID, classification.ProposedCategoryID)
	require.Contains(t, classification.Reason, "gateway unavailable")
}

func TestSingleActiveCategoryUsesDeterministicCodePath(t *testing.T) {
	fake := &mocks.FakeInference{}
	svc, journal := newCategoryService(t, fake)
	_, err := svc.Bootstrap(context.Background())
	require.NoError(t, err)
	category, err := svc.Create(context.Background(), "Only category", "No choice is needed")
	require.NoError(t, err)
	signal := captureText(t, journal)
	require.NoError(t, svc.Enrich(context.Background(), signal))
	proposal, found, err := svc.GetClassification(context.Background(), signal.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, category.ID, proposal.ProposedCategoryID)
	require.Equal(t, 1.0, proposal.ProposedConfidence)
	require.Equal(t, "deterministic.single_category", proposal.Model)
	require.Zero(t, fake.ClassifyCalls.Load(), "one remaining option is a code decision")
}
