package providers

import (
	"context"
	"encoding/json"
	"testing"
)

func TestStorageBrokerForwardsManagedLogSubject(t *testing.T) {
	client := &privilegeBrokerClient{exchange: func(_ context.Context, request, response any) error {
		raw, _ := json.Marshal(request)
		var wire map[string]any
		_ = json.Unmarshal(raw, &wire)
		log, ok := wire["log"].(map[string]any)
		if !ok || log["stanza"] != "vrooli-log-volume-bounds" {
			t.Fatalf("managed log subject lost in transport: %s", raw)
		}
		req := request.(privilegeBrokerRequest)
		*(response.(*privilegeBrokerResponse)) = privilegeBrokerResponse{Version: req.Version, Action: req.Action, RequestID: req.RequestID, Status: "completed", Changed: true}
		return nil
	}}
	if _, err := client.Do(context.Background(), "log.rotate.force", map[string]any{"log": map[string]any{"stanza": "vrooli-log-volume-bounds"}}); err != nil {
		t.Fatal(err)
	}
}

func TestStorageBrokerRejectsDroppedOrMixedSubjectsBeforeTransport(t *testing.T) {
	called := false
	client := &privilegeBrokerClient{exchange: func(context.Context, any, any) error { called = true; return nil }}
	for _, subject := range []map[string]any{
		{"log": map[string]any{"stanza": "vrooli-log-volume-bounds"}, "command": "anything"},
		{"docker": map[string]any{}},
		{"log": "anything"},
	} {
		_, err := client.Do(context.Background(), "log.rotate.force", subject)
		if err == nil || called {
			t.Fatalf("unsafe subject reached broker: %v err=%v", subject, err)
		}
	}
}

func TestStorageBrokerMutationRequiresMatchingTerminalReceipt(t *testing.T) {
	for _, test := range []string{"completed", "admitted", "wrong request", "wrong action", "wrong version"} {
		t.Run(test, func(t *testing.T) {
			client := &privilegeBrokerClient{exchange: func(_ context.Context, request, response any) error {
				req := request.(privilegeBrokerRequest)
				out := privilegeBrokerResponse{Version: req.Version, RequestID: req.RequestID, Action: req.Action, Status: "completed", Changed: true}
				switch test {
				case "admitted":
					out.Status = "running"
				case "wrong request":
					out.RequestID = "other"
				case "wrong action":
					out.Action = "other"
				case "wrong version":
					out.Version = "v2"
				}
				*(response.(*privilegeBrokerResponse)) = out
				return nil
			}}
			got, err := client.Do(context.Background(), "docker.prune.unused-images", map[string]any{"docker": map[string]any{}})
			if test == "completed" {
				if err != nil || !got.Changed {
					t.Fatalf("result=%v err=%v", got, err)
				}
			} else if err == nil || got.Changed {
				t.Fatalf("unverified %s receipt claimed an effect: %v", test, got)
			}
		})
	}
}
