package experiment

import (
	"context"
	"testing"

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
			name: "negative snr",
			recipe: &experimentv1.ExperimentRecipe{Augmentation: &experimentv1.AugmentationRecipe{
				SnrDb: []float64{12, -3},
			}},
			want: "augmentation.snr_db[1]",
		},
		{
			name: "speaker without profile",
			recipe: &experimentv1.ExperimentRecipe{Speaker: &experimentv1.SpeakerExperimentRecipe{
				ExtractionEnabled: true,
			}},
			want: "target_profile_id",
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

func TestStartExperimentEstimatedSecondsOverrideWins(t *testing.T) {
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
	require.Equal(t, int32(12), resp.Msg.GetEstimatedSeconds())
	require.Equal(t, 12, mgr.spec.EstimatedSeconds)
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
