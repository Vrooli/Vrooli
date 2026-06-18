package presence_test

import (
	"testing"
	"time"

	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

func fixedNow() time.Time { return time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC) }

// [REQ:BRG-P0-004] The control plane pushes a typed job frame down a node's held
// dial-out channel; the SSE handler drains conn.Out(). A push to an online node
// reaches its connection.
func TestHub_PushDeliversToOnlineNode(t *testing.T) {
	hub := presence.NewHub(mocks.NewFakeClock(fixedNow()))
	conn := hub.Connect("n1")
	defer conn.Close()

	delivered := hub.Push("n1", []byte(`{"job":{}}`))
	require.Equal(t, 1, delivered)

	select {
	case got := <-conn.Out():
		require.Equal(t, `{"job":{}}`, string(got))
	default:
		t.Fatal("expected the pushed frame on conn.Out()")
	}
}

// [REQ:BRG-P0-004] A push to an offline node reaches nothing (delivered 0), so
// dispatch treats it as non-delivery.
func TestHub_PushToOfflineNodeDeliversNothing(t *testing.T) {
	hub := presence.NewHub(mocks.NewFakeClock(fixedNow()))
	require.Equal(t, 0, hub.Push("ghost", []byte("x")))
}

// [REQ:BRG-P0-004] A push fans out to every live connection a node holds.
func TestHub_PushFansOutToAllConnections(t *testing.T) {
	hub := presence.NewHub(mocks.NewFakeClock(fixedNow()))
	c1 := hub.Connect("n1")
	c2 := hub.Connect("n1")
	defer c1.Close()
	defer c2.Close()

	require.Equal(t, 2, hub.Push("n1", []byte("frame")))
}

// [REQ:BRG-P0-004] A wedged node whose buffer is full does not stall the push
// path: once its buffer fills, further pushes report it as undelivered.
func TestHub_PushNonBlockingWhenBufferFull(t *testing.T) {
	hub := presence.NewHub(mocks.NewFakeClock(fixedNow()))
	conn := hub.Connect("n1")
	defer conn.Close()

	// Fill the buffer without draining conn.Out().
	delivered := 0
	for i := 0; i < 1000; i++ {
		delivered += hub.Push("n1", []byte("x"))
	}
	require.Greater(t, delivered, 0, "some pushes land in the buffer")
	require.Less(t, delivered, 1000, "a full buffer reports non-delivery rather than blocking")
}
