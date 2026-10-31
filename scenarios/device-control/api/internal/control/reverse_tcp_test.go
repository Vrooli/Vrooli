package control

import (
	"context"
	"errors"
	"testing"

	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"

	"github.com/stretchr/testify/require"
)

type reverseFlowStrategy struct {
	*fakes.Strategy
	cancel          context.CancelFunc
	cleanupErr      error
	cleanupContexts []error
	active          bool
}

func (s *reverseFlowStrategy) OpenReverseTCP(ctx context.Context, _, _ int) (func(context.Context) error, error) {
	s.active = true
	if s.cancel != nil {
		s.cancel()
	}
	return func(cleanupCtx context.Context) error {
		s.cleanupContexts = append(s.cleanupContexts, cleanupCtx.Err())
		s.active = false
		return s.cleanupErr
	}, ctx.Err()
}

func reverseStep() Step {
	return Step{ID: "route", Kind: "reverse-tcp", Target: "24965", Arguments: map[string]any{"host_port": float64(24965)}}
}

func TestReverseTCPFlowRestoresOnSuccessFailureAndCancellation(t *testing.T) {
	for _, mode := range []string{"success", "downstream-failure", "cancelled", "cleanup-failure"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			a := &reverseFlowStrategy{Strategy: fakes.New("fake", strategy.StatusAvailable, strategy.CapReverseTCP, strategy.CapInput)}
			steps := []Step{reverseStep()}
			switch mode {
			case "downstream-failure":
				a.ActuateErr = errors.New("input failed")
				steps = append(steps, Step{ID: "input", Kind: "key", Target: "HOME"})
			case "cancelled":
				a.cancel = cancel
			case "cleanup-failure":
				a.cleanupErr = errors.New("mapping removal unverified")
			}
			svc := New(strategyregistry.New(a))
			result, err := svc.Run(ctx, Flow{Steps: steps}, "fake", "operator")
			require.NoError(t, err)
			require.Equal(t, []error{nil}, a.cleanupContexts, "cleanup retains authority after request cancellation")
			require.False(t, a.active)
			require.Len(t, result.Restoration, 1)
			if mode == "success" {
				require.Equal(t, "passed", result.Disposition)
			} else {
				require.NotEqual(t, "passed", result.Disposition)
			}
			if mode == "cleanup-failure" {
				require.Equal(t, "failed", result.Restoration[0].Status)
			} else {
				require.Equal(t, "restored", result.Restoration[0].Status)
			}
		})
	}
}

func TestReverseTCPValidationRefusesMissingCapabilityAndInvalidPortSets(t *testing.T) {
	for _, steps := range [][]Step{
		{{ID: "a", Kind: "reverse-tcp", Target: "80", Arguments: map[string]any{"host_port": float64(24965)}}},
		{{ID: "a", Kind: "reverse-tcp", Target: "24965", Arguments: map[string]any{"host_port": float64(22)}}},
		{{ID: "a", Kind: "reverse-tcp", Target: "24965", Arguments: map[string]any{"host_port": 24965}}},
		{reverseStep(), {ID: "b", Kind: "reverse-tcp", Target: "24965", Arguments: map[string]any{"host_port": float64(20002)}}},
	} {
		require.Error(t, validateFlowContract(Flow{Steps: steps}))
	}
	svc := New(strategyregistry.New(fakes.New("fake", strategy.StatusAvailable, strategy.CapInput)))
	require.False(t, svc.Validate(context.Background(), Flow{Steps: []Step{reverseStep()}}, "fake").Runnable)
}

func (s *reverseFlowStrategy) RemoveReverseTCP(ctx context.Context, _, _ int) error {
	s.active = false
	s.cleanupContexts = append(s.cleanupContexts, ctx.Err())
	return s.cleanupErr
}

func TestReverseTCPRecoveryUsesLeaseScopedExactDestinationOperation(t *testing.T) {
	a := &reverseFlowStrategy{Strategy: fakes.New("fake", strategy.StatusAvailable, strategy.CapReverseTCP), active: true}
	svc := New(strategyregistry.New(a))
	step := reverseStep()
	step.Kind = "reverse-tcp-remove"
	result, err := svc.Run(context.Background(), Flow{Steps: []Step{step}}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.False(t, a.active)
	require.Empty(t, result.Restoration, "explicit removal must not recreate the removed mapping")
}
