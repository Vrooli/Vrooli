package experiment

import (
	"context"
	"math"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	intexp "audio-tools/internal/experiment"
	"audio-tools/internal/logx"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

func TestValidateRecipeRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name   string
		recipe *experimentv1.ExperimentRecipe
		want   string
	}{
		{
			name: "unknown strategy",
			recipe: &experimentv1.ExperimentRecipe{Strategies: []*evalv1.EvalStrategy{{
				Kind: "mystery",
			}}},
			want: "strategies[0].kind",
		},
		{
			name:   "negative seed",
			recipe: &experimentv1.ExperimentRecipe{Seed: -1},
			want:   "seed",
		},
		{
			name: "negative target duration",
			recipe: &experimentv1.ExperimentRecipe{LongForm: &experimentv1.LongFormRecipe{
				TargetDurationSeconds: -1,
			}},
			want: "long_form.target_duration_seconds",
		},
		{
			name: "negative sweep duration",
			recipe: &experimentv1.ExperimentRecipe{LongForm: &experimentv1.LongFormRecipe{
				SweepDurationsSeconds: []int32{30, -60},
			}},
			want: "long_form.sweep_durations_seconds[1]",
		},
		{
			name: "snr out of range",
			recipe: &experimentv1.ExperimentRecipe{Augmentation: &experimentv1.AugmentationRecipe{
				SnrDb: []float64{12, 500},
			}},
			want: "augmentation.snr_db[1]",
		},
		{
			name: "unknown noise type",
			recipe: &experimentv1.ExperimentRecipe{Augmentation: &experimentv1.AugmentationRecipe{
				NoiseTypes: []string{"white", "whooosh"},
			}},
			want: "augmentation.noise_types[1]",
		},
		{
			name: "speaker without profile",
			recipe: &experimentv1.ExperimentRecipe{Speaker: &experimentv1.SpeakerExperimentRecipe{
				ExtractionEnabled: true,
			}},
			want: "target_profile_id",
		},
		{
			name: "cell without engine",
			recipe: &experimentv1.ExperimentRecipe{Cells: []*experimentv1.EvaluationCell{{
				Strategy: "passthrough", ReplayLane: experimentv1.ReplayLane_REPLAY_LANE_REALTIME, RepeatCount: 1,
			}}},
			want: "cells[0].engine_id",
		},
		{
			name: "realtime cell with tail approximation",
			recipe: &experimentv1.ExperimentRecipe{LatencyTailSeconds: 5, Cells: []*experimentv1.EvaluationCell{{
				EngineId: "kyutai", Strategy: "passthrough", ReplayLane: experimentv1.ReplayLane_REPLAY_LANE_REALTIME, RepeatCount: 1,
			}}},
			want: "cannot use latency_tail_seconds",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRecipe(tc.recipe)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestValidateRecipeAllowsDefaultsAndOverlapStallDefault(t *testing.T) {
	err := validateRecipe(&experimentv1.ExperimentRecipe{Strategies: []*evalv1.EvalStrategy{{
		Kind:                   "overlap_agree",
		OverlapMaxStallRejects: -1,
	}}})
	require.NoError(t, err)
}

func TestValidateRecipeAllowsProviderNeutralCell(t *testing.T) {
	err := validateRecipe(&experimentv1.ExperimentRecipe{Cells: []*experimentv1.EvaluationCell{{
		EngineId: "kyutai", Strategy: "passthrough", PolicyProfile: "speaker-filter",
		ReplayLane: experimentv1.ReplayLane_REPLAY_LANE_PRODUCT_PATH, FaultProfile: "dropped_connection", RepeatCount: 2,
	}}})
	require.NoError(t, err)
}

// Zero and negative SNR (noise as loud as / louder than speech) are the
// canonical hard conditions the robustness lab exists to measure and must be
// accepted, not rejected.
func TestValidateRecipeAllowsZeroAndNegativeSNR(t *testing.T) {
	for _, snr := range []float64{0, -5, -10, minSNRDB, maxSNRDB} {
		err := validateRecipe(&experimentv1.ExperimentRecipe{Augmentation: &experimentv1.AugmentationRecipe{
			NoiseTypes: []string{"white"},
			SnrDb:      []float64{snr},
		}})
		require.NoErrorf(t, err, "snr_db=%g should be accepted", snr)
	}
}

func TestValidateRecipeRejectsNonFiniteSNR(t *testing.T) {
	err := validateRecipe(&experimentv1.ExperimentRecipe{Augmentation: &experimentv1.AugmentationRecipe{
		SnrDb: []float64{math.Inf(1)},
	}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "finite")
}

func TestValidateRecipeAllowsKnownNoiseAliases(t *testing.T) {
	err := validateRecipe(&experimentv1.ExperimentRecipe{Augmentation: &experimentv1.AugmentationRecipe{
		NoiseTypes: []string{"white", "fan", "percussive", "music", "constant_fan", "dynamic", "music_like"},
	}})
	require.NoError(t, err)
}

// A dry run validates and resolves the recipe but must not enqueue anything.
func TestStartExperimentDryRunDoesNotSubmit(t *testing.T) {
	mgr := &captureSubmitManager{}
	h := NewConnectHandler(Deps{
		Logger:  logx.Std{},
		Manager: mgr,
		Service: &intexp.Service{},
	})

	resp, err := h.StartExperiment(context.Background(), connect.NewRequest(&experimentv1.StartExperimentRequest{
		Name:   "preview",
		DryRun: true,
		Recipe: &experimentv1.ExperimentRecipe{
			Strategies:   []*evalv1.EvalStrategy{{Kind: "batch"}},
			Augmentation: &experimentv1.AugmentationRecipe{NoiseTypes: []string{"white"}, SnrDb: []float64{-5}},
		},
	}))

	require.NoError(t, err)
	require.True(t, resp.Msg.GetDryRun())
	require.Empty(t, mgr.spec.Name, "dry run must not call Submit")
	require.Empty(t, resp.Msg.GetExperiment().GetId(), "preview has no persisted id")
	require.Equal(t, "preview", resp.Msg.GetExperiment().GetName())
	require.Equal(t, []float64{-5}, resp.Msg.GetExperiment().GetRecipe().GetAugmentation().GetSnrDb())
}

func TestStartExperimentDryRunStillValidates(t *testing.T) {
	mgr := &captureSubmitManager{}
	h := NewConnectHandler(Deps{Logger: logx.Std{}, Manager: mgr, Service: &intexp.Service{}})

	_, err := h.StartExperiment(context.Background(), connect.NewRequest(&experimentv1.StartExperimentRequest{
		DryRun: true,
		Recipe: &experimentv1.ExperimentRecipe{Augmentation: &experimentv1.AugmentationRecipe{NoiseTypes: []string{"whooosh"}}},
	}))

	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Contains(t, err.Error(), "noise_types")
	require.Empty(t, mgr.spec.Name, "invalid dry run must not submit")
}

type noOpManager struct{}

func (noOpManager) Submit(context.Context, intexp.SubmitSpec) (intexp.Experiment, error) {
	return intexp.Experiment{}, nil
}

func (noOpManager) Get(context.Context, string) (intexp.Experiment, error) {
	return intexp.Experiment{}, nil
}

func (noOpManager) Wait(context.Context, string) (intexp.Experiment, error) {
	return intexp.Experiment{}, nil
}

func (noOpManager) List(context.Context, intexp.ListFilter) ([]intexp.Experiment, error) {
	return nil, nil
}
func (noOpManager) Cancel(string) error { return nil }
func (noOpManager) Subscribe(string) (<-chan intexp.ProgressEvent, func(), error) {
	ch := make(chan intexp.ProgressEvent)
	close(ch)
	return ch, func() {}, nil
}

type captureSubmitManager struct {
	noOpManager
	spec intexp.SubmitSpec
}

func (m *captureSubmitManager) Submit(_ context.Context, spec intexp.SubmitSpec) (intexp.Experiment, error) {
	m.spec = spec
	return intexp.Experiment{
		ID:          "exp-1",
		Name:        spec.Name,
		Status:      intexp.StatusQueued,
		RecipeJSON:  spec.RecipeJSON,
		MachineJSON: spec.MachineJSON,
	}, nil
}

type getOnlyManager struct {
	noOpManager
	exp intexp.Experiment
}

func (m getOnlyManager) Get(context.Context, string) (intexp.Experiment, error) {
	return m.exp, nil
}

func TestListExperimentsRejectsNegativePagination(t *testing.T) {
	h := NewConnectHandler(Deps{Logger: logx.Std{}, Manager: noOpManager{}})

	_, err := h.ListExperiments(context.Background(), connect.NewRequest(&experimentv1.ListExperimentsRequest{Limit: -1}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Contains(t, err.Error(), "limit")

	_, err = h.ListExperiments(context.Background(), connect.NewRequest(&experimentv1.ListExperimentsRequest{Offset: -1}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Contains(t, err.Error(), "offset")
}

func TestStartExperimentRejectsMissingSpeakerProfileAsFailedPrecondition(t *testing.T) {
	h := NewConnectHandler(Deps{
		Logger:  logx.Std{},
		Manager: noOpManager{},
		Service: &intexp.Service{},
	})

	_, err := h.StartExperiment(context.Background(), connect.NewRequest(&experimentv1.StartExperimentRequest{
		Recipe: &experimentv1.ExperimentRecipe{Speaker: &experimentv1.SpeakerExperimentRecipe{
			VerificationEnabled: true,
		}},
	}))

	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Contains(t, err.Error(), "speaker-enroll")
	require.Contains(t, err.Error(), "speaker-status")
}

func TestStartExperimentPersistsMachineProvenance(t *testing.T) {
	mgr := &captureSubmitManager{}
	h := NewConnectHandler(Deps{
		Logger:  logx.Std{},
		Manager: mgr,
		Service: &intexp.Service{},
	})

	resp, err := h.StartExperiment(context.Background(), connect.NewRequest(&experimentv1.StartExperimentRequest{
		Name:   "provenance",
		Recipe: &experimentv1.ExperimentRecipe{Seed: 42},
	}))

	require.NoError(t, err)
	require.NotEmpty(t, mgr.spec.MachineJSON)
	require.NotEqual(t, "{}", string(mgr.spec.MachineJSON))
	require.Contains(t, string(mgr.spec.MachineJSON), "recipe_sha256")
	require.Equal(t, string(mgr.spec.MachineJSON), resp.Msg.GetExperiment().GetMachineJson())
}

func TestStartExperimentComputesEstimatedSeconds(t *testing.T) {
	mgr := &captureSubmitManager{}
	h := NewConnectHandler(Deps{
		Logger:  logx.Std{},
		Manager: mgr,
		Service: &intexp.Service{},
	})

	resp, err := h.StartExperiment(context.Background(), connect.NewRequest(&experimentv1.StartExperimentRequest{
		Name: "estimate",
		Recipe: &experimentv1.ExperimentRecipe{
			Strategies:      []*evalv1.EvalStrategy{{Kind: "batch"}, {Kind: "vad_segment"}},
			RealtimeRepeats: 1,
			LongForm:        &experimentv1.LongFormRecipe{Enabled: true, SweepDurationsSeconds: []int32{30, 60}},
			Augmentation:    &experimentv1.AugmentationRecipe{NoiseTypes: []string{"white", "fan"}, SnrDb: []float64{6}},
			Speaker:         &experimentv1.SpeakerExperimentRecipe{TargetProfileId: "profile-1", AblationEnabled: true},
		},
	}))

	require.NoError(t, err)
	require.Equal(t, int32(2880), resp.Msg.GetEstimatedSeconds())
	require.Equal(t, 2880, mgr.spec.EstimatedSeconds)
}

func TestStartExperimentComputesEstimatedSecondsFromClipIDs(t *testing.T) {
	mgr := &captureSubmitManager{}
	h := NewConnectHandler(Deps{
		Logger:  logx.Std{},
		Manager: mgr,
		Service: &intexp.Service{},
		EstimateClipSeconds: func(_ context.Context, clipIDs []string) (int32, error) {
			require.Equal(t, []string{"clip-a", "clip-b"}, clipIDs)
			return 11, nil
		},
	})

	resp, err := h.StartExperiment(context.Background(), connect.NewRequest(&experimentv1.StartExperimentRequest{
		Name: "estimate-clip-ids",
		Recipe: &experimentv1.ExperimentRecipe{
			ClipIds:         []string{"clip-a", "clip-b"},
			Strategies:      []*evalv1.EvalStrategy{{Kind: "batch"}},
			RealtimeRepeats: 1,
		},
	}))

	require.NoError(t, err)
	require.Equal(t, int32(22), resp.Msg.GetEstimatedSeconds())
	require.Equal(t, 22, mgr.spec.EstimatedSeconds)
}

func TestStartExperimentEstimatedSecondsCannotUnderstateRecipe(t *testing.T) {
	mgr := &captureSubmitManager{}
	h := NewConnectHandler(Deps{
		Logger:  logx.Std{},
		Manager: mgr,
		Service: &intexp.Service{},
	})

	resp, err := h.StartExperiment(context.Background(), connect.NewRequest(&experimentv1.StartExperimentRequest{
		EstimatedSeconds: 12,
		Recipe: &experimentv1.ExperimentRecipe{
			Strategies: []*evalv1.EvalStrategy{{Kind: "batch"}},
			LongForm:   &experimentv1.LongFormRecipe{Enabled: true, TargetDurationSeconds: 600},
		},
	}))

	require.NoError(t, err)
	require.Equal(t, int32(600), resp.Msg.GetEstimatedSeconds())
	require.Equal(t, 600, mgr.spec.EstimatedSeconds)
}

func TestStartExperimentRejectsExcessiveComputedDuration(t *testing.T) {
	mgr := &captureSubmitManager{}
	h := NewConnectHandler(Deps{Logger: logx.Std{}, Manager: mgr, Service: &intexp.Service{}})
	_, err := h.StartExperiment(context.Background(), connect.NewRequest(&experimentv1.StartExperimentRequest{
		Recipe: &experimentv1.ExperimentRecipe{
			Cells:    []*experimentv1.EvaluationCell{{EngineId: "kyutai", Strategy: "passthrough", ReplayLane: experimentv1.ReplayLane_REPLAY_LANE_REALTIME, RepeatCount: 2}},
			LongForm: &experimentv1.LongFormRecipe{Enabled: true, TargetDurationSeconds: 1_801},
		},
	}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "qualification ceiling")
	require.Nil(t, mgr.spec.RecipeJSON)
}

func TestExperimentRuntimeBudgetBoundsKnownAndUnknownDurations(t *testing.T) {
	require.Equal(t, 18*time.Minute+45*time.Second, experimentRuntimeBudget(900))
	require.Equal(t, experimentRuntimeUnknown, experimentRuntimeBudget(0))
}

func TestDeleteExperimentRejectsRunningExperiment(t *testing.T) {
	h := NewConnectHandler(Deps{
		Logger:  logx.Std{},
		Manager: getOnlyManager{exp: intexp.Experiment{ID: "exp-running", Status: intexp.StatusRunning}},
		Service: &intexp.Service{},
	})

	_, err := h.DeleteExperiment(context.Background(), connect.NewRequest(&experimentv1.DeleteExperimentRequest{Id: "exp-running"}))

	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Contains(t, err.Error(), "status is running")
}
