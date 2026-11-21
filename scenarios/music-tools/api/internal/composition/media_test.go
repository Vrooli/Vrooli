package composition

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func mediaFixture() []byte {
	raw := make([]byte, 48)
	copy(raw, "RIFF")
	binary.LittleEndian.PutUint32(raw[4:], 40)
	copy(raw[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(raw[16:], 16)
	binary.LittleEndian.PutUint16(raw[20:], 1)
	binary.LittleEndian.PutUint16(raw[22:], 2)
	binary.LittleEndian.PutUint32(raw[24:], 48000)
	binary.LittleEndian.PutUint32(raw[28:], 192000)
	binary.LittleEndian.PutUint16(raw[32:], 4)
	binary.LittleEndian.PutUint16(raw[34:], 16)
	copy(raw[36:], "data")
	binary.LittleEndian.PutUint32(raw[40:], 4)
	return raw
}

func TestValidatedWAVChecksCompleteStream(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		ok   bool
	}{{"complete", mediaFixture(), true}, {"truncated", mediaFixture()[:47], false}, {"extra", append(mediaFixture(), 0), false}, {"header only", mediaFixture()[:12], false}} {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := ValidatedWAV(bytes.NewReader(tc.data))
			if err == nil {
				_, err = io.Copy(io.Discard, reader)
			}
			if (err == nil) != tc.ok {
				t.Fatalf("validation %v", err)
			}
		})
	}
}
