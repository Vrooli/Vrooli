package wireproto

import (
	"encoding/json"
	"testing"
)

func TestTerminalMessageCumulativeOffsetFieldsRoundTrip(t *testing.T) {
	input := TerminalMessage{
		Type:            MsgTypeStdin,
		Data:            "é",
		Offset:          12,
		AcceptedThrough: 14,
		HaveThrough:     12,
		ProtocolVersion: ProtocolVersion,
	}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var got TerminalMessage
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Offset != input.Offset || got.AcceptedThrough != input.AcceptedThrough || got.HaveThrough != input.HaveThrough || got.ProtocolVersion != ProtocolVersion {
		t.Fatalf("round trip = %+v", got)
	}
}

func TestControlAndResyncAreDistinctWireTypes(t *testing.T) {
	if MsgTypeControl == MsgTypeResync {
		t.Fatal("control and resync must remain distinct")
	}
}

func TestMouseModeFieldsRoundTrip(t *testing.T) {
	input := TerminalMessage{Type: MsgTypeSessionReady, MouseMode: false, MouseModeKnown: true}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var got TerminalMessage
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if !got.MouseModeKnown || got.MouseMode {
		t.Fatalf("mouse mode round trip = %+v", got)
	}
}
