package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"web-console/backends/codex"
	"web-console/internal/sessionstore"
)

type continuityFixtureStdin struct{ output *io.PipeWriter }

func (continuityFixtureStdin) Write(p []byte) (int, error) { return len(p), nil }
func (s continuityFixtureStdin) Close() error              { return s.output.Close() }

func TestManagedCodexShutdownPreservesObservedTurnActivity(t *testing.T) {
	for _, activity := range []sessionstore.NativeTurnActivity{sessionstore.NativeTurnIdle, sessionstore.NativeTurnActive} {
		t.Run(string(activity), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			store := sessionstore.NewInMemory()
			row := sessionstore.Metadata{ID: "session", NativeThreadID: "thread",
				LaunchMode: sessionstore.LaunchModeCodexAppServer, ControlMode: sessionstore.ControlModeNativeCapable,
				LaunchDescriptorJSON: `{"agent":"codex","launchMode":"codex_app_server"}`}
			if err := store.Save(ctx, row); err != nil {
				t.Fatal(err)
			}

			reader, writer := io.Pipe()
			t.Cleanup(func() { _ = writer.Close() })
			initialResponse := strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":{}}` + "\n")
			client := codex.NewClient(continuityFixtureStdin{writer}, io.MultiReader(initialResponse, reader))
			if err := client.Call(ctx, "fixture", nil, nil); err != nil {
				t.Fatal(err)
			}
			owner, err := codex.NewManagedOwner(client, "generation", "thread")
			if err != nil {
				t.Fatal(err)
			}
			r := newManagedCodexRegistry(store)
			r.entries[row.ID] = managedCodexEntry{owner: owner}
			observed := make(chan string, 2)
			r.setEventSink(func(_ string, event codex.NativeEvent) { observed <- event.Kind })
			r.watch(row.ID, owner)
			send := func(method string) {
				t.Helper()
				if _, err := fmt.Fprintf(writer, "{\"method\":%q,\"params\":{\"threadId\":\"thread\",\"turn\":{\"id\":\"turn\"}}}\n", method); err != nil {
					t.Fatal(err)
				}
				select {
				case got := <-observed:
					if got != method {
						t.Fatalf("event = %q, want %q", got, method)
					}
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			send("turn/started")
			if activity == sessionstore.NativeTurnIdle {
				send("turn/completed")
			}
			if err := r.Shutdown(ctx); err != nil {
				t.Fatal(err)
			}
			got, err := store.Get(ctx, row.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != sessionstore.StatusLive || got.ControlMode != row.ControlMode ||
				got.NativeThreadID != row.NativeThreadID || got.NativeTurnActivity != activity {
				t.Fatalf("shutdown lost recovery evidence: %+v", got)
			}
			if _, ok := r.owner(row.ID); ok {
				t.Fatal("closed runtime still accepts work")
			}
			boot := newManagedCodexRegistry(store)
			var outbound strings.Builder
			boot.resume = func(_ context.Context, _, ownerID, _, _, threadID, _ string, _ []string, _ []string) (*codex.ManagedOwner, error) {
				if threadID != row.NativeThreadID {
					t.Fatalf("boot selected thread %q", threadID)
				}
				return codex.NewManagedOwner(codex.NewClient(testWriteCloser{&outbound}, strings.NewReader("")), ownerID, threadID)
			}
			if err := boot.Recover(ctx); err != nil {
				t.Fatal(err)
			}
			if _, ok := boot.owner(row.ID); !ok || outbound.Len() != 0 {
				t.Fatal("boot did not restore the exact session without submitting a turn")
			}
			if err := boot.Shutdown(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
}
