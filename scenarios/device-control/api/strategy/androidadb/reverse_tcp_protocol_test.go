package androidadb

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReverseTCPProtocolUsesExactTransportLoopbackAndNoRebind(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(fmt.Sprint(reject), func(t *testing.T) {
			client, server := net.Pipe()
			defer client.Close()
			defer server.Close()
			requests := make(chan []string, 1)
			go func() {
				defer server.Close()
				var seen []string
				for i := 0; i < 2; i++ {
					var prefix [4]byte
					if _, err := io.ReadFull(server, prefix[:]); err != nil {
						return
					}
					n, err := strconv.ParseInt(string(prefix[:]), 16, 32)
					if err != nil {
						return
					}
					body := make([]byte, n)
					if _, err := io.ReadFull(server, body); err != nil {
						return
					}
					seen = append(seen, string(body))
					if _, err := io.WriteString(server, "OKAY"); err != nil {
						return
					}
				}
				requests <- seen
				if reject {
					_, _ = io.WriteString(server, "FAIL")
				} else {
					_, _ = io.WriteString(server, "OKAY")
				}
			}()
			err := createReverseTCPProtocol(context.Background(), client, "192.168.1.158:5555", 24965, 24965)
			if reject {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, []string{"host:transport:192.168.1.158:5555", "reverse:forward:norebind:tcp:localhost:24965;tcp:localhost:24965"}, <-requests)
		})
	}
}

func TestReverseTCPProtocolCancellationClosesStalledConnection(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- createReverseTCPProtocol(ctx, client, "tv", 24965, 24965) }()
	cancel()
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("cancel did not close the connection")
	}
}
