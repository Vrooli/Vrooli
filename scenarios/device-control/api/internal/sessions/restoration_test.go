package sessions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateManagerRestoresReverseOrderAndReportsFailures(t *testing.T) {
	manager := &StateManager{}
	order := []string{}
	manager.Push("orientation", func(context.Context) error { order = append(order, "orientation"); return nil })
	manager.Push("network", func(context.Context) error { order = append(order, "network"); return errors.New("radio unavailable") })
	events := manager.Restore(context.Background())
	require.Equal(t, []string{"network", "orientation"}, order)
	require.Equal(t, "failed", events[0].Status)
	require.Equal(t, "restored", events[1].Status)
}
