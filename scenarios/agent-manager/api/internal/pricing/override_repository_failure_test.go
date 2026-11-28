package pricing

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type overrideReadFailureRepository struct {
	Repository
	err error
}

func (r overrideReadFailureRepository) GetOverridesForModel(context.Context, string) ([]*ManualPriceOverride, error) {
	return nil, r.err
}

func newOverrideReadFailureService(t *testing.T) (Service, error) {
	t.Helper()
	ctx := context.Background()
	base := NewMemoryRepository()
	inputPrice, outputPrice := 2e-6, 8e-6
	require.NoError(t, base.UpsertPricing(ctx, &ModelPricing{
		CanonicalModelName: "vendor/model",
		Provider:           "openrouter",
		InputTokenPrice:    &inputPrice,
		OutputTokenPrice:   &outputPrice,
		InputTokenSource:   SourceProviderAPI,
		OutputTokenSource:  SourceProviderAPI,
	}))
	readErr := errors.New("override store unavailable")
	repo := overrideReadFailureRepository{Repository: base, err: readErr}
	return NewServiceWithModelResolver(repo, nil, logrus.New(), testModelResolver{}), readErr
}

// A configured manual price is authoritative for attribution. When its store
// cannot be read, every price-facing consumer must avoid presenting the lower-
// priority provider price as settled. Cost events must retain a distinct reason
// so operators can tell dependency uncertainty from a genuinely unpriced model.
func TestPriceConsumersFailClosedWhenOverrideReadFails(t *testing.T) {
	ctx := context.Background()

	t.Run("pricing lookup returns the repository error", func(t *testing.T) {
		svc, readErr := newOverrideReadFailureService(t)
		_, err := svc.GetModelPricing(ctx, "vendor/model", "codex")
		require.ErrorIs(t, err, readErr)
	})

	t.Run("model listing does not fall back to provider prices", func(t *testing.T) {
		svc, readErr := newOverrideReadFailureService(t)
		_, err := svc.ListModelsWithPricing(ctx)
		require.ErrorIs(t, err, readErr)
	})

	t.Run("cost calculation is unpriced with an uncertainty reason", func(t *testing.T) {
		svc, _ := newOverrideReadFailureService(t)
		calc, err := svc.CalculateCost(ctx, CostRequest{
			Model: "vendor/model", RunnerType: "codex", InputTokens: 1000, OutputTokens: 1000,
		})
		require.NoError(t, err)
		require.Equal(t, "unpriced", calc.CostSource)
		require.Equal(t, "override_lookup_failed", calc.ChargeReason)
		require.Zero(t, calc.TotalCostUSD)
	})
}
