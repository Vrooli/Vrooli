package main

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/vrooli/secrets-manager-native-host/protocol"
)

func TestWriteFailureKeepsStdoutFramed(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = writer
	writeFailure(errors.New("malformed test input"))
	os.Stdout = original
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	frame, err := protocol.ReadFrame(reader)
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := json.Unmarshal(frame, &response); err != nil {
		t.Fatal(err)
	}
	if response["ok"] != "false" || response["protocol"] != protocol.ProtocolName {
		t.Fatalf("response = %#v", response)
	}
	if _, ok := response["message"]; !ok {
		t.Fatal("framed failure omitted its diagnostic message")
	}
}
