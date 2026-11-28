package handlers

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
	"google.golang.org/protobuf/proto"
)

func TestDecodeProgramEventBase64Envelope(t *testing.T) {
	want := &telemetryv1.ProgramEvent{EventId: "event-1", ProgramId: "program-1", SessionId: "session-1"}
	raw, err := proto.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]string{"encoding": "base64", "data": base64.StdEncoding.EncodeToString(raw)})
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := decodeProgramEvent(payload)
	if err != nil {
		t.Fatal(err)
	}
	if got.GetEventId() != want.GetEventId() || got.GetProgramId() != want.GetProgramId() {
		t.Fatalf("got=%+v", got)
	}
}

func TestProgramEventSignatureRejectsTampering(t *testing.T) {
	if validProgramEventSignature([]byte("body"), "bad", "secret") {
		t.Fatal("bad signature accepted")
	}
}
