package codex

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

type writeCloser struct{ io.Writer }

func (writeCloser) Close() error { return nil }

func TestClientForksThroughVerifiedTurn(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"result":{"thread":{"id":"forked-thread"}}}` + "\n"
	var sent strings.Builder
	c := NewClient(writeCloser{&sent}, strings.NewReader(input))
	out, err := c.ForkThread(context.Background(), "thread-1", "turn-4")
	if err != nil {
		t.Fatal(err)
	}
	if out.ID() != "forked-thread" {
		t.Fatalf("thread id = %q", out.ThreadID)
	}
	if !strings.Contains(sent.String(), `"method":"thread/fork"`) || !strings.Contains(sent.String(), `"lastTurnId":"turn-4"`) {
		t.Fatalf("request did not bind native fork boundary: %s", sent.String())
	}
}

func TestClientReturnsProviderError(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"error":{"code":-1,"message":"unsupported"}}` + "\n"
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(input))
	if err := c.Initialize(context.Background(), "web-console", "test"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("error = %v", err)
	}
}

func TestInitializeAnnouncesReadyAfterHandshake(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"result":{}}` + "\n"
	var sent strings.Builder
	c := NewClient(writeCloser{&sent}, strings.NewReader(input))
	if err := c.Initialize(context.Background(), "web-console", "test"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sent.String(), `"method":"initialized"`) {
		t.Fatalf("initialized notification missing: %s", sent.String())
	}
}

func TestInitializeCapturesProviderVersion(t *testing.T) {
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":{"userAgent":"codex-app-server/0.153.4"}}`+"\n"))
	if err := c.Initialize(context.Background(), "web-console", "test"); err != nil {
		t.Fatal(err)
	}
	if c.ServerVersion() != "codex-app-server/0.153.4" {
		t.Fatalf("server version = %q", c.ServerVersion())
	}
}

func TestClientExposesProviderNotificationsWithoutDroppingResponses(t *testing.T) {
	input := "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}\n" + "{\"method\":\"thread/started\",\"params\":{\"thread\":{\"id\":\"t1\"}}}\n"
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(input))
	if err := c.Initialize(context.Background(), "web-console", "test"); err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-c.Notifications():
		if msg.Method != "thread/started" {
			t.Fatalf("notification = %q", msg.Method)
		}
	case <-time.After(time.Second):
		t.Fatal("thread notification was dropped")
	}
}

func TestClientStartsTurnWithNativeThreadBinding(t *testing.T) {
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":{"turn":{"id":"turn-7"}}}`+"\n"))
	turn, err := c.StartTurn(context.Background(), "thread-1", "hello")
	if err != nil || turn.ID() != "turn-7" {
		t.Fatalf("turn = %+v, err=%v", turn, err)
	}
}

func TestClientSeparatesProviderRequestsFromResponses(t *testing.T) {
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(`{"jsonrpc":"2.0","id":"approval-1","method":"item/commandApproval/request","params":{"threadId":"thread-1"}}`+"\n"))
	c.startReader()
	select {
	case request := <-c.Requests():
		if request.Method != "item/commandApproval/request" || string(request.ID) != `"approval-1"` {
			t.Fatalf("request = %+v", request)
		}
	case <-time.After(time.Second):
		t.Fatal("provider request was dropped")
	}
}

func TestClientCloseIsIdempotentAfterReaderEOF(t *testing.T) {
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(""))
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
