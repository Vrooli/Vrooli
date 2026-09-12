package audiotools

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
)

type resolverFunc func() (string, error)

func (f resolverFunc) Resolve() (string, error) { return f() }

func TestResolverClientAndCredentials(t *testing.T) {
	t.Setenv("AUDIO_TOOLS_URL", "http://example.test")
	if got, err := (EnvResolver{EnvVar: "AUDIO_TOOLS_URL"}).Resolve(); err != nil || got != "http://example.test" {
		t.Fatalf("env resolver=%q err=%v", got, err)
	}
	calls := 0
	inner := resolverFunc(func() (string, error) { calls++; return "http://audio", nil })
	cached := &CachedResolver{Inner: inner, TTL: time.Minute}
	if _, err := cached.Resolve(); err != nil {
		t.Fatal(err)
	}
	if _, err := cached.Resolve(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("resolver calls=%d", calls)
	}
	cached.Invalidate()
	c, err := New(cached, Policy{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	if !c.Resolved() || c.BaseURL() != "http://audio" {
		t.Fatal("client not resolved")
	}
	req := connect.NewRequest(&struct{}{})
	AttachCredentials(req, Credentials{BYOKProvider: "ollama", BYOKKey: "key", LPBSToken: "token", UserIdentity: "user"})
	if req.Header().Get("X-Audio-BYOK-Key") != "key" {
		t.Fatal("credentials missing")
	}
	c.HandleTransportFailure()
	if c.Resolved() {
		t.Fatal("transport failure did not invalidate")
	}
	if err := c.PingContext(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestResolversAndErrorNormalization(t *testing.T) {
	t.Setenv("AUDIO_TOOLS_URL", " ")
	if got, err := (EnvResolver{EnvVar: "AUDIO_TOOLS_URL", Default: "http://fallback"}).Resolve(); err != nil || got != "http://fallback" {
		t.Fatalf("default resolver=%q err=%v", got, err)
	}
	if _, err := (EnvResolver{EnvVar: "MISSING_AUDIO_TOOLS"}).Resolve(); err == nil {
		t.Fatal("missing resolver should fail")
	}
	if !errors.Is(NormalizeError(context.DeadlineExceeded), ErrTimeout) {
		t.Fatal("deadline should normalize to timeout")
	}
	if !errors.Is(NormalizeError(connect.NewError(connect.CodeUnavailable, errors.New("down"))), ErrUnavailable) {
		t.Fatal("connect unavailable should normalize")
	}
	if !errors.Is(NormalizeError(connect.NewError(connect.CodeInvalidArgument, errors.New("bad"))), ErrInvalidArgument) {
		t.Fatal("connect invalid argument should normalize")
	}
}
