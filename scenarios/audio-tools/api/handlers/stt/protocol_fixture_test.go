package stt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type browserProtocolFixture struct {
	Magic       string            `json:"magic"`
	HeaderBytes int               `json:"headerBytes"`
	Messages    map[string]string `json:"messages"`
}

func serverProtocolFixture() browserProtocolFixture {
	return browserProtocolFixture{
		Magic:       "ATV2",
		HeaderBytes: wsV2AudioHeaderBytes,
		Messages: map[string]string{
			"partial":         wsMsgPartial,
			"final":           wsMsgFinal,
			"segmentFinal":    wsMsgSegmentFinal,
			"segmentRejected": wsMsgSegmentRejected,
			"error":           wsMsgError,
			"done":            wsMsgDone,
			"status":          wsMsgStatus,
			"vadState":        wsMsgVadState,
		},
	}
}

func protocolFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join("../../../../../packages/audio-capture-browser/src/__fixtures__/audio-tools-atv2.json"))
}

func TestBrowserProtocolFixtureMatchesServerConstants(t *testing.T) {
	path := protocolFixturePath(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read browser protocol fixture: %v (regenerate with AUDIO_PROTOCOL_UPDATE_FIXTURE=1 go test ./handlers/stt)", err)
	}
	var got browserProtocolFixture
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode browser protocol fixture: %v", err)
	}
	want := serverProtocolFixture()
	updateFixture, configured := os.LookupEnv("AUDIO_PROTOCOL_UPDATE_FIXTURE")
	if configured && updateFixture != "1" {
		t.Fatalf("AUDIO_PROTOCOL_UPDATE_FIXTURE must be empty or 1, got %q", updateFixture)
	}
	if updateFixture == "1" {
		encoded, err := json.MarshalIndent(want, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("write browser protocol fixture: %v", err)
		}
		got = want
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("browser fixture differs from server constants: got=%#v want=%#v; regenerate with AUDIO_PROTOCOL_UPDATE_FIXTURE=1 go test ./handlers/stt", got, want)
	}
}
