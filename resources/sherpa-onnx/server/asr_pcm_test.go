package main

import (
	"encoding/binary"
	"testing"
)

func TestDecodePCM16LEReusesScratchWithoutChangingSamples(t *testing.T) {
	pcm := make([]byte, 8)
	for i, sample := range []int16{-32768, -1, 0, 16384} {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(sample))
	}
	scratch := make([]float32, 0, 4)
	first := decodePCM16LE(scratch, pcm)
	want := []float32{-1, -1.0 / 32768, 0, 0.5}
	for i := range want {
		if first[i] != want[i] {
			t.Fatalf("sample %d = %v, want %v", i, first[i], want[i])
		}
	}
	if len(first) != cap(scratch) {
		t.Fatalf("decoded length %d, want scratch capacity %d", len(first), cap(scratch))
	}
	firstPointer := &first[0]
	second := decodePCM16LE(first, pcm[:4])
	if &second[0] != firstPointer {
		t.Fatal("decoder did not reuse the per-stream scratch allocation")
	}
	if len(second) != 2 || second[0] != -1 || second[1] != -1.0/32768 {
		t.Fatalf("reused decode = %v, want [-1, -1/32768]", second)
	}
}

func TestDecodePCM16LEDoesNotAllocateWithAdequateScratch(t *testing.T) {
	pcm := make([]byte, 6400)
	scratch := make([]float32, 0, len(pcm)/2)
	allocations := testing.AllocsPerRun(100, func() {
		scratch = decodePCM16LE(scratch, pcm)
	})
	if allocations != 0 {
		t.Fatalf("decode allocated %v objects per packet with sufficient scratch", allocations)
	}
}

func TestRetainPCMScratchReleasesOversizedPacketBuffer(t *testing.T) {
	ordinary := make([]float32, maxRetainedPCMScratchSamples)
	if retained := retainPCMScratch(ordinary); len(retained) != len(ordinary) {
		t.Fatalf("ordinary scratch length = %d, want %d", len(retained), len(ordinary))
	}
	oversized := make([]float32, maxRetainedPCMScratchSamples+1)
	if retained := retainPCMScratch(oversized); retained != nil {
		t.Fatalf("oversized packet retained %d float samples", len(retained))
	}
	largeCapacity := make([]float32, 1, maxRetainedPCMScratchSamples+1)
	if retained := retainPCMScratch(largeCapacity); retained != nil {
		t.Fatalf("small slice with oversized capacity retained %d float samples", cap(retained))
	}
}
