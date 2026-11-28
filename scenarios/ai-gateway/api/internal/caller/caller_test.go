package caller

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestScenariosNameTheScenarioACallerScopeBelongsTo(t *testing.T) {
	scenarios := NewScenarios([]string{"agent", "agent-manager", "program-runtime", " "})
	cases := map[string]string{
		"scenario:agent-manager-start-api/agent-manager-api": "agent-manager",
		"scenario:program-runtime-lab-start-api/api":         "program-runtime", // a variant keeps its scenario
		"scenario:agent-start-api/agent-api":                 "agent",
		"scenario:retired-thing-start-api/api":               "retired-thing-start-api",
		"agent:codex/ai-gateway":                             "agent:codex",
		"resource:ollama-companion/resource-ollama":          "resource:ollama-companion",
		Unknown: Unknown,
		"":      Unknown,
	}
	for label, want := range cases {
		if got := scenarios.Of(label); got != want {
			t.Errorf("Of(%q) = %q, want %q", label, got, want)
		}
	}
}

func TestLabelledContextIsReadyAtOnce(t *testing.T) {
	ctx := WithLabel(context.Background(), "agent:codex/ai-gateway")
	if label, ok := Ready(ctx); !ok || label != "agent:codex/ai-gateway" {
		t.Fatalf("Ready = %q, %v", label, ok)
	}
	if Pending(ctx) {
		t.Fatal("a known caller is not pending")
	}
	if got := Await(WithLabel(context.Background(), " "), time.Second); got != Unknown {
		t.Fatalf("blank label = %q, want unknown", got)
	}
}

func TestContextWithoutAnObservationHasNoCaller(t *testing.T) {
	ctx := context.Background()
	if _, ok := Ready(ctx); ok || Pending(ctx) {
		t.Fatal("no observation must read as neither ready nor pending")
	}
	if got := Await(ctx, time.Millisecond); got != Unknown {
		t.Fatalf("Await = %q, want unknown", got)
	}
}

// The test process is both ends of the connection, and identification never
// names the server itself, so a real loopback peer resolves to unknown; what
// matters here is that the observation always settles.
func TestObserveSettlesForARealLoopbackPeer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	ctx := context.WithValue(context.Background(), http.LocalAddrContextKey, server.LocalAddr())
	ctx = Observe(ctx, server.RemoteAddr().String())
	if got := Await(ctx, 5*time.Second); got != Unknown {
		t.Fatalf("self-connection caller = %q, want unknown", got)
	}
	if Pending(ctx) {
		t.Fatal("a settled observation is not pending")
	}
}

func TestObserveWithoutAServerAddressIsUnknown(t *testing.T) {
	ctx := Observe(context.Background(), "127.0.0.1:40000")
	if label, ok := Ready(ctx); !ok || label != Unknown {
		t.Fatalf("Ready = %q, %v; want unknown at once", label, ok)
	}
	ctx = context.WithValue(context.Background(), http.LocalAddrContextKey, &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1})
	if label, _ := Ready(Observe(ctx, "not-an-address")); label != Unknown {
		t.Fatalf("bad peer address = %q, want unknown", label)
	}
}
