package codex

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/envkit-go"
)

type writeCloser struct{ io.Writer }

func (writeCloser) Close() error { return nil }

func TestFilteredAppServerEnvironmentDropsAmbientAgentIdentity(t *testing.T) {
	got := appServerEnvironment([]string{"PATH=/bin", "VROOLI_AGENT_IDENTITY_TOKEN=ambient", "HOME=/tmp"}, nil, envkit.Platform{})
	if strings.Contains(strings.Join(got, "\n"), "VROOLI_AGENT_IDENTITY_TOKEN=") {
		t.Fatalf("ambient Agent Manager identity leaked to managed app-server: %v", got)
	}
	if !strings.Contains(strings.Join(got, "\n"), "PATH=/bin") || !strings.Contains(strings.Join(got, "\n"), "HOME=/tmp") {
		t.Fatalf("unrelated launch environment was dropped: %v", got)
	}
}

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

func TestAppServerEnvironmentIsIndependentOfHostServiceAndCaller(t *testing.T) {
	parent := []string{"PATH=/bin", "HOME=/home/operator", "VROOLI_ROOT=/repo", "VROOLI_HOME=/runtime", "OPENAI_API_KEY=fixture-key", "CODEX_AGENT_TAG=host-tag", "CLAUDE_AGENT_TAG=other-host-tag", "VROOLI_RUN_ID=host-run", "VROOLI_HARNESS_SESSION_ID=host-session", "VROOLI_AGENT_SESSION=host-agent", "WC_WEB_CONSOLE_SESSION_ID=host-pane", "SCENARIO_NAME=web-console", "SCENARIO_PATH=/web-console", "API_PORT=8000", "API_BASE_URL=http://host-service", "VROOLI_RUNTIME_INSTANCE_ID=host-instance", "VROOLI_PROCESS_ID=host-process", "CODEX_THREAD_ID=host-thread", "VROOLI_AGENT_IDENTITY_TOKEN=host-token", "CODEX_HOME=/host-home", "VROOLI_AGENT_MANAGER_STORE_ID=host-store"}
	child := []string{"CODEX_AGENT_TAG=child-tag", "CODEX_HOME=/child-home", "VROOLI_AGENT_MANAGER_STORE_ID=child-store", "VROOLI_AGENT_IDENTITY_TOKEN=verified-child"}
	for _, platform := range []envkit.Platform{{}, {CaseInsensitive: true}} {
		got := appServerEnvironment(parent, child, platform)
		values := map[string]string{}
		for _, entry := range got {
			key, value, _ := strings.Cut(entry, "=")
			if _, exists := values[key]; exists {
				t.Fatalf("duplicate environment key %s", key)
			}
			values[key] = value
		}
		for _, key := range []string{"CLAUDE_AGENT_TAG", "VROOLI_RUN_ID", "VROOLI_HARNESS_SESSION_ID", "VROOLI_AGENT_SESSION", "WC_WEB_CONSOLE_SESSION_ID", "SCENARIO_NAME", "SCENARIO_PATH", "API_PORT", "API_BASE_URL", "VROOLI_RUNTIME_INSTANCE_ID", "VROOLI_PROCESS_ID", "CODEX_THREAD_ID"} {
			if _, exists := values[key]; exists {
				t.Errorf("provider inherited host attribution %s", key)
			}
		}
		for key, want := range map[string]string{"CODEX_AGENT_TAG": "child-tag", "CODEX_HOME": "/child-home", "VROOLI_AGENT_MANAGER_STORE_ID": "child-store", "VROOLI_AGENT_IDENTITY_TOKEN": "verified-child", "PATH": "/bin", "HOME": "/home/operator", "VROOLI_ROOT": "/repo", "VROOLI_HOME": "/runtime", "OPENAI_API_KEY": "fixture-key"} {
			if values[key] != want {
				t.Errorf("child launch lost %s", key)
			}
		}
	}
}
