package handlers

import (
	"encoding/binary"
	"testing"
)

func TestDecodeDriverFrame(t *testing.T) {
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	payload, header, err := decodeDriverFrame(jpeg)
	if err != nil || header != nil || string(payload) != string(jpeg) {
		t.Fatalf("JPEG decoding = payload=%v header=%v err=%v", payload, header, err)
	}

	headerJSON := []byte(`{"frame_id":"frame-1","capture_ms":4}`)
	packet := make([]byte, 4+len(headerJSON)+len(jpeg))
	binary.BigEndian.PutUint32(packet[:4], uint32(len(headerJSON)))
	copy(packet[4:], headerJSON)
	copy(packet[4+len(headerJSON):], jpeg)
	payload, header, err = decodeDriverFrame(packet)
	if err != nil || header == nil || header.FrameID != "frame-1" || string(payload) != string(jpeg) {
		t.Fatalf("header decoding = payload=%v header=%+v err=%v", payload, header, err)
	}
}
