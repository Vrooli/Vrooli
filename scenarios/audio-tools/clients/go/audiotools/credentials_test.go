package audiotools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health"
)

// Verify the interceptor writes the four X-Audio-* headers from
// context-bound credentials and omits absent ones.
func TestCredentialsInterceptor_PopulatesHeadersFromContext(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.Header().Set("Content-Type", "application/proto")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	interceptor := WithCredentialsInterceptor(DefaultCredentialsGetter)

	// Forge a unary call by constructing a Connect client against any
	// service; the test only inspects request headers, so we choose a
	// service whose pb.go ships in the proto module even though the
	// server doesn't implement it — the connection drops with an
	// error after the headers are written, which is exactly what we
	// want to inspect.
	type stubReq = healthv1.Response
	client := connect.NewClient[stubReq, stubReq](http.DefaultClient, srv.URL+"/stub.Service/M", connect.WithInterceptors(interceptor))

	ctx := WithCredentials(context.Background(), Credentials{
		BYOKProvider: "openai-whisper",
		BYOKKey:      "sk-abc",
		LPBSToken:    "lpbs-tok",
		UserIdentity: "user-42",
	})
	_, _ = client.CallUnary(ctx, connect.NewRequest(&stubReq{}))

	require.Equal(t, "openai-whisper", got.Get(HeaderProvider))
	require.Equal(t, "sk-abc", got.Get(HeaderKey))
	require.Equal(t, "lpbs-tok", got.Get(HeaderLPBSToken))
	require.Equal(t, "user-42", got.Get(HeaderUserIdentity))
}

func TestCredentialsInterceptor_AbsentContextWritesNoHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	interceptor := WithCredentialsInterceptor(DefaultCredentialsGetter)
	client := connect.NewClient[healthv1.Response, healthv1.Response](http.DefaultClient, srv.URL+"/stub.Service/M", connect.WithInterceptors(interceptor))
	_, _ = client.CallUnary(context.Background(), connect.NewRequest(&healthv1.Response{}))

	require.Empty(t, got.Get(HeaderProvider))
	require.Empty(t, got.Get(HeaderKey))
	require.Empty(t, got.Get(HeaderLPBSToken))
	require.Empty(t, got.Get(HeaderUserIdentity))
}

func TestCredentialsInterceptor_PartialCredentialsOnlySetsPopulated(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	interceptor := WithCredentialsInterceptor(DefaultCredentialsGetter)
	client := connect.NewClient[healthv1.Response, healthv1.Response](http.DefaultClient, srv.URL+"/stub.Service/M", connect.WithInterceptors(interceptor))

	ctx := WithCredentials(context.Background(), Credentials{LPBSToken: "tok-only"})
	_, _ = client.CallUnary(ctx, connect.NewRequest(&healthv1.Response{}))

	require.Empty(t, got.Get(HeaderProvider))
	require.Empty(t, got.Get(HeaderKey))
	require.Equal(t, "tok-only", got.Get(HeaderLPBSToken))
	require.Empty(t, got.Get(HeaderUserIdentity))
}

func TestCredentials_HasAny(t *testing.T) {
	require.False(t, Credentials{}.HasAny())
	require.True(t, Credentials{LPBSToken: "x"}.HasAny())
	require.True(t, Credentials{UserIdentity: "u"}.HasAny())
}

func TestFromContext_Default(t *testing.T) {
	_, ok := FromContext(context.Background())
	require.False(t, ok)
	ctx := WithCredentials(context.Background(), Credentials{LPBSToken: "x"})
	got, ok := FromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "x", got.LPBSToken)
}
