//go:build linux

package hostfs

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTemporaryActivityUsesSetupBrokerWithNoCallerSelectedTarget(t *testing.T) {
	observer := &temporaryActivityObserver{exchange: func(_ context.Context, request, response any) error {
		encoded, err := json.Marshal(request)
		if err != nil {
			return err
		}
		var req map[string]any
		_ = json.Unmarshal(encoded, &req)
		if len(req) != 4 || req["action"] != "storage.temporary-activity.observe" || len(req["subject"].(map[string]any)) != 0 {
			t.Fatalf("unbounded observation request: %s", encoded)
		}
		snapshot, _ := json.Marshal(map[string]any{"uid": os.Getuid(), "complete": true, "observed_at": time.Now().UTC(), "roots": []string{"/tmp"}, "paths": []string{"/tmp/held", "/outside/private"}})
		*(response.(*temporaryActivityResponse)) = temporaryActivityResponse{Version: "v1", RequestID: req["request_id"].(string), Action: req["action"].(string), Status: "completed", Snapshot: snapshot}
		return nil
	}}
	paths, err := observer.ObserveTemporary(context.Background(), "/tmp")
	if err != nil || len(paths) != 1 {
		t.Fatalf("paths=%v err=%v", paths, err)
	}
}

func TestTemporaryActivityRejectsUnmatchedOrRefusedBrokerReceipt(t *testing.T) {
	for _, test := range []string{"wrong request", "wrong action", "wrong version", "refused", "changed", "no evidence", "partial"} {
		t.Run(test, func(t *testing.T) {
			observer := &temporaryActivityObserver{exchange: func(_ context.Context, request, response any) error {
				encoded, _ := json.Marshal(request)
				var req map[string]any
				_ = json.Unmarshal(encoded, &req)
				snapshot, _ := json.Marshal(map[string]any{"uid": os.Getuid(), "complete": true, "observed_at": time.Now().UTC(), "roots": []string{"/tmp"}, "paths": []string{"/tmp/held"}})
				out := temporaryActivityResponse{Version: "v1", RequestID: req["request_id"].(string), Action: req["action"].(string), Status: "completed", Snapshot: snapshot}
				switch test {
				case "wrong request":
					out.RequestID = "other"
				case "wrong action":
					out.Action = "other"
				case "wrong version":
					out.Version = "v2"
				case "refused":
					out.Status = "failed"
					out.Code = "action_not_allowed"
				case "changed":
					out.Changed = true
				case "no evidence":
					out.Snapshot = nil
				case "partial":
					out.Snapshot = []byte(`{"complete":false}`)
				}
				*(response.(*temporaryActivityResponse)) = out
				return nil
			}}
			if paths, err := observer.ObserveTemporary(context.Background(), "/tmp"); err == nil || paths != nil {
				t.Fatalf("accepted %s: paths=%v err=%v", test, paths, err)
			} else if test == "refused" && !strings.Contains(err.Error(), "vrooli setup --sudo-mode=ask") {
				t.Fatalf("missing setup recovery instruction: %v", err)
			}
		})
	}
}

func TestTemporaryActivityDecoderRequiresFreshCompleteMatchingUser(t *testing.T) {
	now := time.Now().UTC()
	for _, test := range []struct {
		name     string
		uid      int
		at       time.Time
		complete bool
		valid    bool
	}{
		{"valid", 1000, now, true, true}, {"wrong user", 0, now, true, false}, {"stale", 1000, now.Add(-time.Minute), true, false}, {"incomplete", 1000, now, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw, _ := json.Marshal(map[string]any{"uid": test.uid, "observed_at": test.at, "complete": test.complete, "roots": []string{"/tmp"}, "paths": []string{"/tmp/held", "/tmpfoo/private", "/private"}})
			paths, err := decodeTemporaryActivity(string(raw), "/tmp", 1000, now)
			if test.valid {
				if err != nil || len(paths) != 1 {
					t.Fatalf("paths=%v err=%v", paths, err)
				}
			} else if err == nil {
				t.Fatal("accepted invalid activity proof")
			}
		})
	}
}

func TestTemporaryActivityRefusesUnobservedTemporaryNamespace(t *testing.T) {
	now := time.Now().UTC()
	raw, _ := json.Marshal(map[string]any{"uid": 1000, "complete": true, "observed_at": now, "roots": []string{"/tmp"}, "paths": []string{}})
	for _, root := range []string{"/", "/tmp-other", "/home/user/custom-temp"} {
		if paths, err := decodeTemporaryActivity(string(raw), root, 1000, now); err == nil || paths != nil {
			t.Fatalf("unobserved root %s accepted: %v %v", root, paths, err)
		}
	}
}

func TestTemporaryActivityPreservesRedactedProcessFailure(t *testing.T) {
	observer := &temporaryActivityObserver{exchange: func(_ context.Context, request, response any) error {
		raw, _ := json.Marshal(request)
		var req map[string]any
		_ = json.Unmarshal(raw, &req)
		out := temporaryActivityResponse{Version: "v1", RequestID: req["request_id"].(string), Action: req["action"].(string), Status: "failed", Code: "temporary_activity_incomplete"}
		out.Evidence.Detail = "process 123 environ: permission denied"
		*(response.(*temporaryActivityResponse)) = out
		return nil
	}}
	paths, err := observer.ObserveTemporary(context.Background(), "/tmp")
	if paths != nil || err == nil || !strings.Contains(err.Error(), "process 123 environ") || strings.Contains(err.Error(), "run vrooli setup") {
		t.Fatalf("process failure lost or falsely attributed to setup: %v %v", paths, err)
	}
}
