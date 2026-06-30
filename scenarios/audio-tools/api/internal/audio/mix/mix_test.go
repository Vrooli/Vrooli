package mix

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestOverlayAtSNRHitsTarget(t *testing.T) {
	signal := constantPCM(12000, 16000)
	overlay := constantPCM(6000, 1600)
	out, stats, err := OverlayAtSNR(signal, overlay, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(signal) {
		t.Fatalf("len(out)=%d want %d", len(out), len(signal))
	}
	if math.Abs(stats.ActualSNRDB-12) > 0.1 {
		t.Fatalf("actual SNR %.3f, want ~12", stats.ActualSNRDB)
	}
	if stats.ClippedCount != 0 {
		t.Fatalf("unexpected clipping: %d", stats.ClippedCount)
	}
}

func TestOverlayAtSNRSaturates(t *testing.T) {
	signal := constantPCM(maxInt16-1, 4)
	overlay := constantPCM(maxInt16-1, 4)
	out, stats, err := OverlayAtSNR(signal, overlay, -20)
	if err != nil {
		t.Fatal(err)
	}
	if stats.ClippedCount == 0 {
		t.Fatalf("expected clipping")
	}
	for i := 0; i < len(out); i += 2 {
		if got := int16(binary.LittleEndian.Uint16(out[i:])); got != maxInt16 {
			t.Fatalf("sample %d=%d want saturated max", i/2, got)
		}
	}
}

func TestOverlayAtSNRRejectsBadLengths(t *testing.T) {
	if _, _, err := OverlayAtSNR([]byte{1}, []byte{0, 0}, 0); err == nil {
		t.Fatalf("expected odd signal length error")
	}
	if _, _, err := OverlayAtSNR([]byte{0, 0}, nil, 0); err == nil {
		t.Fatalf("expected empty overlay error")
	}
}

func constantPCM(v int, samples int) []byte {
	out := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(int16(v)))
	}
	return out
}
