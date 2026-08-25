package audiotools

import (
	"context"
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
