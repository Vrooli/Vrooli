package activityedge

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
)

func TestConnPeerResolvesOncePerConnection(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	release := make(chan struct{})
	peer := &connPeer{resolve: func() string {
		<-release
		mu.Lock()
		calls++
		mu.Unlock()
		return "scenario:demo/demo-api"
	}}
	got := make(chan string, 3)
	peer.observe(func(caller string) { got <- caller })
	peer.observe(func(caller string) { got <- caller })
	close(release)
	for i := 0; i < 2; i++ {
		if caller := <-got; caller != "scenario:demo/demo-api" {
			t.Fatalf("caller = %q", caller)
		}
	}
	peer.observe(func(caller string) { got <- caller })
	if caller := <-got; caller != "scenario:demo/demo-api" {
		t.Fatalf("resolved caller = %q", caller)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("resolve calls = %d, want 1 per connection", calls)
	}
	var none *connPeer
	none.observe(func(caller string) { got <- caller })
	if caller := <-got; caller != modelretention.UnknownCaller {
		t.Fatalf("a request without a connection peer = %q, want unknown", caller)
	}
}

func TestPeerContextCarriesTheConnectionAddresses(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	ctx := withPeer(context.Background(), server, func(local, remote net.Addr) string {
		if local == nil || remote == nil {
			return ""
		}
		return "host:pipe/test"
	})
	done := make(chan string, 1)
	peerFrom(ctx).observe(func(caller string) { done <- caller })
	select {
	case caller := <-done:
		if caller != "host:pipe/test" {
			t.Fatalf("caller = %q", caller)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("peer resolution did not finish")
	}
}
