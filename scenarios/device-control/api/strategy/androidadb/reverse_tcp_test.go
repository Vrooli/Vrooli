package androidadb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type reverseRunner struct {
	bindings   map[string]string
	writes     []string
	failRemove bool
	cancel     context.CancelFunc
	normalize  bool
}

func (r *reverseRunner) Run(ctx context.Context, _ string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	args = args[2:] // exact selected device: -s tv
	switch args[1] {
	case "--list":
		var out strings.Builder
		for device, host := range r.bindings {
			fmt.Fprintf(&out, "tv %s %s\n", device, host)
		}
		return []byte(out.String()), nil
	case "--no-rebind":
		if _, exists := r.bindings[args[2]]; exists {
			return nil, errors.New("cannot rebind")
		}
		r.writes = append(r.writes, strings.Join(args, " "))
		key := args[2]
		if r.normalize {
			key = "tcp:" + key[strings.LastIndex(key, ":")+1:]
		}
		r.bindings[key] = args[3]
		if r.cancel != nil {
			r.cancel()
			return nil, context.Canceled
		}
	case "--remove":
		r.writes = append(r.writes, strings.Join(args, " "))
		if r.failRemove {
			return nil, errors.New("transport unavailable")
		}
		delete(r.bindings, args[2])
	default:
		return nil, errors.New("unexpected command")
	}
	return nil, nil
}

func (r *reverseRunner) createReverseTCP(ctx context.Context, selector string, devicePort, hostPort int) error {
	_, err := r.Run(ctx, "adb", "-s", selector, "reverse", "--no-rebind", fmt.Sprintf("tcp:localhost:%d", devicePort), fmt.Sprintf("tcp:localhost:%d", hostPort))
	return err
}

func TestReverseTCPPreservesExistingBindingsAndRemovesOnlyOwnedRoute(t *testing.T) {
	r := &reverseRunner{bindings: map[string]string{"tcp:3000": "tcp:4000"}}
	cleanup, err := NewWithRunner(r, "tv").OpenReverseTCP(context.Background(), 24965, 24965)
	require.NoError(t, err)
	require.Equal(t, "tcp:localhost:24965", r.bindings["tcp:localhost:24965"])
	require.NoError(t, cleanup(context.Background()))
	require.NoError(t, cleanup(context.Background()))
	require.Equal(t, map[string]string{"tcp:3000": "tcp:4000"}, r.bindings)
	require.Equal(t, []string{"reverse --no-rebind tcp:localhost:24965 tcp:localhost:24965", "reverse --remove tcp:localhost:24965"}, r.writes)
}

func TestReverseTCPRefusesExistingPortAndInvalidEndpointsWithoutWrites(t *testing.T) {
	r := &reverseRunner{bindings: map[string]string{"tcp:24965": "tcp:localhost:24965"}}
	a := NewWithRunner(r, "tv")
	for _, ports := range [][2]int{{24965, 24965}, {80, 24965}, {24965, 22}, {65536, 24965}} {
		cleanup, err := a.OpenReverseTCP(context.Background(), ports[0], ports[1])
		require.Error(t, err)
		require.Nil(t, cleanup)
	}
	require.Empty(t, r.writes)
}

func TestReverseTCPCleansPartiallyAcknowledgedSetupAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &reverseRunner{bindings: map[string]string{}, cancel: cancel}
	cleanup, err := NewWithRunner(r, "tv").OpenReverseTCP(ctx, 24965, 24965)
	require.Error(t, err)
	require.NotNil(t, cleanup)
	require.NoError(t, cleanup(context.Background()))
	require.Empty(t, r.bindings)
}

func TestReverseTCPCleanupRefusesChangedBindingAndReportsRemovalFailure(t *testing.T) {
	r := &reverseRunner{bindings: map[string]string{}}
	cleanup, err := NewWithRunner(r, "tv").OpenReverseTCP(context.Background(), 24965, 24965)
	require.NoError(t, err)
	r.bindings["tcp:localhost:24965"] = "tcp:localhost:20002"
	require.ErrorContains(t, cleanup(context.Background()), "unowned")
	require.Len(t, r.writes, 1)
	r.bindings["tcp:localhost:24965"] = "tcp:localhost:24965"
	r.failRemove = true
	require.ErrorContains(t, cleanup(context.Background()), "remove owned")
	require.NotEmpty(t, r.bindings)
}

func TestReverseTCPHandlesAndroid11NormalizedNamesAndExactRecovery(t *testing.T) {
	r := &reverseRunner{bindings: map[string]string{}, normalize: true}
	a := NewWithRunner(r, "tv")
	cleanup, err := a.OpenReverseTCP(context.Background(), 24965, 24965)
	require.NoError(t, err)
	require.Equal(t, "tcp:localhost:24965", r.bindings["tcp:24965"])
	require.NoError(t, cleanup(context.Background()))
	require.Empty(t, r.bindings)
	r.bindings["tcp:24965"] = "tcp:localhost:24965"
	require.Error(t, a.RemoveReverseTCP(context.Background(), 24965, 20002))
	require.Len(t, r.bindings, 1)
	require.NoError(t, a.RemoveReverseTCP(context.Background(), 24965, 24965))
	require.Empty(t, r.bindings)
}
