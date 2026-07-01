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
