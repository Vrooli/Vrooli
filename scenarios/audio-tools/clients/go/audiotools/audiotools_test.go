package audiotools

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type erroringResolver struct{ err error }

func (e erroringResolver) ResolveURL(_ context.Context) (string, error) { return "", e.err }

type sequenceResolver struct {
	urls []string
	idx  int
}

func (s *sequenceResolver) ResolveURL(_ context.Context) (string, error) {
	if s.idx >= len(s.urls) {
		return "", errors.New("exhausted")
	}
	u := s.urls[s.idx]
	s.idx++
	return u, nil
}

func TestNew_RequiredPropagatesResolverError(t *testing.T) {
	_, err := New(erroringResolver{err: errors.New("port not bound")}, Policy{Required: true})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnavailable))
}

func TestNew_LazyResolveDeferredToFirstCall(t *testing.T) {
	c, err := New(FixedURLResolver("http://localhost:9999"), Policy{})
	require.NoError(t, err)
	require.Empty(t, c.BaseURL())
	url, err := c.Ensure(context.Background())
	require.NoError(t, err)
	require.Equal(t, "http://localhost:9999", url)
	require.Equal(t, "http://localhost:9999", c.BaseURL())
}

func TestNew_RequiredBindsAtConstruction(t *testing.T) {
	c, err := New(FixedURLResolver("http://localhost:8080"), Policy{Required: true})
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080", c.BaseURL())
	require.NotNil(t, c.STT)
	require.NotNil(t, c.TTS)
	require.NotNil(t, c.Audio)
	require.NotNil(t, c.Session)
	require.NotNil(t, c.Settings)
	require.NotNil(t, c.Summarize)
	require.NotNil(t, c.Usage)
}

func TestHandleTransportFailure_ClearsBaseURL(t *testing.T) {
	r := &sequenceResolver{urls: []string{"http://first:1", "http://second:2"}}
	c, err := New(r, Policy{Required: true})
	require.NoError(t, err)
	require.Equal(t, "http://first:1", c.BaseURL())
	c.HandleTransportFailure()
	require.Empty(t, c.BaseURL())
	url, err := c.Ensure(context.Background())
	require.NoError(t, err)
	require.Equal(t, "http://second:2", url)
}

func TestHealthURL(t *testing.T) {
	c, err := New(FixedURLResolver("http://localhost:8080"), Policy{Required: true})
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080/health", c.HealthURL())
}

func TestNew_NilResolverErrors(t *testing.T) {
	_, err := New(nil, Policy{})
	require.Error(t, err)
}
