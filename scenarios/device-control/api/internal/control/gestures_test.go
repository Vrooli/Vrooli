package control

import (
	"context"
	"testing"

	"device-control/strategy"

	"github.com/stretchr/testify/require"
)

func TestGestureVocabularyBuildsTypedPointerEvents(t *testing.T) {
	tests := []struct {
		name  string
		step  Step
		calls int
	}{
		{name: "long press", step: Step{ID: "hold", Kind: "long-press", Arguments: map[string]any{"x": .2, "y": .3}}, calls: 1},
		{name: "double tap", step: Step{ID: "double", Kind: "double-tap", Arguments: map[string]any{"x": .2, "y": .3}}, calls: 2},
		{name: "drag", step: Step{ID: "drag", Kind: "drag", Arguments: map[string]any{"start_x": .1, "start_y": .2, "end_x": .8, "end_y": .9}}, calls: 1},
		{name: "fling", step: Step{ID: "fling", Kind: "fling", Arguments: map[string]any{"start_x": .1, "start_y": .8, "end_x": .1, "end_y": .1}}, calls: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var events []strategy.Actuation
			err := executeGesture(context.Background(), func(_ context.Context, event strategy.Actuation) error {
				events = append(events, event)
				return nil
			}, tc.step)
			require.NoError(t, err)
			require.Len(t, events, tc.calls)
			for _, event := range events {
				require.NotNil(t, event.Pointer)
				require.True(t, event.Pointer.Normalized)
			}
		})
	}
}
