package ingress

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"switchboard/internal/channels"
)

// [REQ:SWBD-P0-005]
func TestDeduplicatesByChannelAndRemoteID(t *testing.T) {
	s := New()
	e := channels.Envelope{ChannelID: "a", RemoteMessageID: "1"}
	r, err := s.Accept(e)
	require.NoError(t, err)
	require.Equal(t, Accepted, r)
	r, err = s.Accept(e)
	require.NoError(t, err)
	require.Equal(t, AlreadySeen, r)
	r, err = s.Accept(channels.Envelope{ChannelID: "b", RemoteMessageID: "1"})
	require.NoError(t, err)
	require.Equal(t, Accepted, r)
}

func TestBurstHasOneAccepted(t *testing.T) {
	s := New()
	e := channels.Envelope{ChannelID: "a", RemoteMessageID: "1"}
	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, _ := s.Accept(e)
			if r == Accepted {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	require.Equal(t, 1, accepted)
	require.Equal(t, 1, s.Count())
}
