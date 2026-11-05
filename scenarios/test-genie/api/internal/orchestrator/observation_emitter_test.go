package orchestrator

import (
	"bytes"
	"testing"
)

func TestObservationEmitterBoundsNewlineFreeInput(t *testing.T) {
	var log bytes.Buffer
	emitter := &observationEmitter{underlying: &log, phase: "unit", emit: func(ExecutionEvent) {}}
	if _, err := emitter.Write(bytes.Repeat([]byte("x"), maxObservationLineBytes*3)); err != nil {
		t.Fatal(err)
	}
	if len(emitter.buffer) != maxObservationLineBytes {
		t.Fatalf("buffer bytes = %d, want %d", len(emitter.buffer), maxObservationLineBytes)
	}
}
