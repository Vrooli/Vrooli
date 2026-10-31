package push

import (
	"encoding/json"
	"testing"
)

func TestDeclarativePayloadIsTheWebKitEnvelope(t *testing.T) {
	raw := DeclarativePayload(Message{Title: "Agent needs your input", Body: "Codex · tab 2", ID: "n-1"}, "https://web-console.example.test/sessions/abc")
	var got struct {
		WebPush      int `json:"web_push"`
		Notification struct {
			Title, Body, Tag, Navigate string
		} `json:"notification"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.WebPush != 8030 || got.Notification.Title != "Agent needs your input" || got.Notification.Body != "Codex · tab 2" ||
		got.Notification.Tag != "n-1" || got.Notification.Navigate != "https://web-console.example.test/sessions/abc" {
		t.Fatalf("unexpected envelope: %s", raw)
	}
}
