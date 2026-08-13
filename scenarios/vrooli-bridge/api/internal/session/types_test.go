package session

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/audit"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
)

type fakeAudit struct{ records []audit.Record }

func (f *fakeAudit) Append(_ context.Context, r audit.Record) (audit.Record, error) {
	f.records = append(f.records, r)
	return r, nil
}

func openRequest() OpenRequest {
	return OpenRequest{ID: "s1", NodeID: "n1", OwnerID: "o1", Scopes: []string{Scope}, Reauth: true, Window: 2, Idle: time.Minute, MaxLifetime: time.Hour}
}

func TestManagerRefusesScopeAndOwnerReauth(t *testing.T) {
	c := scheduletest.New(time.Unix(100, 0))
	m := NewManager(c, nil)
	_, err := m.Open(context.Background(), OpenRequest{ID: "s1", NodeID: "n1", OwnerID: "o1", Reauth: true})
	require.ErrorIs(t, err, ErrScopeDenied)
	r := openRequest()
	r.Reauth = false
	_, err = m.Open(context.Background(), r)
	require.ErrorIs(t, err, ErrOwnerReauth)
}

func TestManagerSequenceGapAndBackpressure(t *testing.T) {
	c := scheduletest.New(time.Unix(100, 0))
	m := NewManager(c, nil)
	_, err := m.Open(context.Background(), openRequest())
	require.NoError(t, err)
	_, err = m.AcceptData(context.Background(), "s1", 1, []byte("out of order"))
	require.ErrorIs(t, err, ErrSequenceGap)
	_, err = m.AcceptData(context.Background(), "s1", 0, []byte("one"))
	require.NoError(t, err)
	_, err = m.AcceptData(context.Background(), "s1", 1, []byte("two"))
	require.NoError(t, err)
	_, err = m.AcceptData(context.Background(), "s1", 2, []byte("three"))
	require.ErrorIs(t, err, ErrWindowFull)
	require.NoError(t, m.Acknowledge("s1", 1))
	_, err = m.AcceptData(context.Background(), "s1", 2, []byte("three"))
	require.NoError(t, err)
}

func TestManagerIdleAndLifetimeAreHardBoundsAndAudited(t *testing.T) {
	c := scheduletest.New(time.Unix(100, 0))
	a := &fakeAudit{}
	m := NewManager(c, a)
	r := openRequest()
	r.Idle = 2 * time.Second
	r.MaxLifetime = 10 * time.Second
	_, err := m.Open(context.Background(), r)
	require.NoError(t, err)
	c.Advance(2 * time.Second)
	_, err = m.AcceptData(context.Background(), "s1", 0, []byte("late"))
	require.Error(t, err)
	require.Len(t, a.records, 2)
	require.Equal(t, audit.ActionSessionClose, a.records[1].Action)

	c = scheduletest.New(time.Unix(100, 0))
	m = NewManager(c, nil)
	r = openRequest()
	r.Idle = time.Hour
	r.MaxLifetime = 2 * time.Second
	_, err = m.Open(context.Background(), r)
	require.NoError(t, err)
	c.Advance(2 * time.Second)
	require.NoError(t, m.Expire(context.Background(), "s1"))
}

func TestManagerKillClosesDoneSignalIdempotently(t *testing.T) {
	m := NewManager(nil, nil)
	state, err := m.Open(context.Background(), OpenRequest{ID: "kill-me", NodeID: "node-1", OwnerID: "owner-1", Scopes: []string{Scope}, Reauth: true})
	require.NoError(t, err)
	done, err := m.Done(state.ID)
	require.NoError(t, err)
	require.NoError(t, m.Kill(context.Background(), state.ID))
	select {
	case <-done:
	default:
		t.Fatal("kill did not close the session done signal")
	}
	require.NoError(t, m.Kill(context.Background(), state.ID))
}

func TestManagerCloseByNodeTerminatesRemoteSessions(t *testing.T) {
	a := &fakeAudit{}
	m := NewManager(nil, a)
	first := openRequest()
	first.ID = "s1"
	_, err := m.Open(context.Background(), first)
	require.NoError(t, err)
	second := first
	second.ID = "s2"
	second.NodeID = "n2"
	_, err = m.Open(context.Background(), second)
	require.NoError(t, err)

	m.CloseByNode(context.Background(), "n1", "node_channel_lost")
	state, err := m.Get("s1")
	require.NoError(t, err)
	require.True(t, state.Closed)
	state, err = m.Get("s2")
	require.NoError(t, err)
	require.False(t, state.Closed)
	require.Equal(t, "node_channel_lost", a.records[len(a.records)-1].Detail)
}

func TestManagerDeliverOutputIsMonotonicAndFanoutSafe(t *testing.T) {
	m := NewManager(nil, nil)
	_, err := m.Open(context.Background(), openRequest())
	require.NoError(t, err)
	out, unsubscribe, err := m.SubscribeOutput("s1")
	require.NoError(t, err)
	defer unsubscribe()
	require.NoError(t, m.DeliverOutput(context.Background(), "s1", 0, []byte("prompt> ")))
	result := <-out
	require.Equal(t, uint64(0), result.Sequence)
	require.Equal(t, []byte("prompt> "), result.Data)
	require.ErrorIs(t, m.DeliverOutput(context.Background(), "s1", 0, []byte("duplicate")), ErrSequenceGap)
	require.NoError(t, m.DeliverOutput(context.Background(), "s1", 1, []byte("ok\n")))
}

func TestManagerReattachPreservesPTYSessionAndReplaysScrollback(t *testing.T) {
	clock := scheduletest.New(time.Unix(100, 0))
	m := NewManager(clock, nil)
	_, err := m.Open(context.Background(), openRequest())
	require.NoError(t, err)
	first, unsubscribe, err := m.SubscribeOutput("s1")
	require.NoError(t, err)
	require.NoError(t, m.DeliverOutput(context.Background(), "s1", 0, []byte("before disconnect\n")))
	require.Equal(t, []byte("before disconnect\n"), (<-first).Data)
	unsubscribe()
	require.NoError(t, m.DeliverOutput(context.Background(), "s1", 1, []byte("during disconnect\n")))

	reattached, err := m.Open(context.Background(), openRequest())
	require.NoError(t, err)
	require.Equal(t, "s1", reattached.ID)
	require.False(t, reattached.Closed)
	replay, unsubscribe, err := m.SubscribeOutput("s1")
	require.NoError(t, err)
	defer unsubscribe()
	require.Equal(t, []byte("before disconnect\n"), (<-replay).Data)
	require.Equal(t, []byte("during disconnect\n"), (<-replay).Data)
	clock.Advance(time.Second)
	reattached, err = m.Open(context.Background(), openRequest())
	require.NoError(t, err)
	require.True(t, reattached.LastActivity.Equal(clock.Now()), "reattach must refresh idle expiry")
}

func TestManagerAcceptsIdenticalOutputReplay(t *testing.T) {
	m := NewManager(nil, nil)
	_, err := m.Open(context.Background(), openRequest())
	require.NoError(t, err)
	require.NoError(t, m.DeliverOutput(context.Background(), "s1", 0, []byte("once\n")))
	require.NoError(t, m.DeliverOutput(context.Background(), "s1", 0, []byte("once\n")))
	require.ErrorIs(t, m.DeliverOutput(context.Background(), "s1", 0, []byte("different\n")), ErrSequenceGap)
}
