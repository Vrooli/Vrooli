package sttchain

import "testing"

func TestSentChunkQueueEvictsAcknowledgedAudio(t *testing.T) {
	var queue sentChunkQueue
	for i := 0; i < 1_000; i++ {
		queue.append(AudioChunk{Sequence: uint64(i), Audio: make([]byte, 32<<10)})
	}

	chunk, ok := queue.acknowledge(8)
	if !ok {
		t.Fatal("acknowledging the first processed window should resolve a chunk")
	}
	if chunk.Sequence != 7 {
		t.Fatalf("acknowledged cursor resolved sequence %d, want 7", chunk.Sequence)
	}
	if got := queue.retained(); got != 992 {
		t.Fatalf("queue retained %d acknowledged chunks, want 992 unacknowledged chunks", got)
	}

	chunk, ok = queue.acknowledge(1_000)
	if !ok {
		t.Fatal("acknowledging the remaining processed window should resolve a chunk")
	}
	if chunk.Sequence != 999 {
		t.Fatalf("final acknowledged cursor resolved sequence %d, want 999", chunk.Sequence)
	}
	if got := queue.retained(); got != 0 {
		t.Fatalf("queue retained %d chunks after complete acknowledgement, want 0", got)
	}
}
