package main

import "testing"

type syntheticTailDecoder struct {
	events       []string
	finalText    string
	pendingWords bool
	invalidTail  bool
}

func (d *syntheticTailDecoder) acceptSilence(samples []float32) {
	d.events = append(d.events, "silence-tail")
	for _, sample := range samples {
		if sample != 0 {
			d.invalidTail = true
			break
		}
	}
	// Simulate a decoder whose buffered final word is emitted only after
	// receiving trailing acoustic context before InputFinished.
	d.pendingWords = len(samples) > 0
}

func (d *syntheticTailDecoder) inputFinished() {
	d.events = append(d.events, "input-finished")
	if d.pendingWords {
		d.finalText = "Ask what you can do for your country."
	} else {
		d.finalText = "Ask what you can do for you. As."
	}
}

func TestFinishOnlineInputTailFlushesBufferedFinalWords(t *testing.T) {
	const sampleRate = 16_000
	cases := []struct {
		name          string
		tailMS        int
		wantSamples   int
		wantFinalText string
		wantEvents    []string
	}{
		{
			name:          "zero tail reproduces dropped final words",
			wantFinalText: "Ask what you can do for you. As.",
			wantEvents:    []string{"input-finished"},
		},
		{
			name:          "positive tail flushes final words before input finishes",
			tailMS:        1200,
			wantSamples:   sampleRate * 12 / 10,
			wantFinalText: "Ask what you can do for your country.",
			wantEvents:    []string{"silence-tail", "input-finished"},
		},
		{
			name:          "excessive tail is bounded",
			tailMS:        60_000,
			wantSamples:   sampleRate * maxStreamingFinishTailMS / 1000,
			wantFinalText: "Ask what you can do for your country.",
			wantEvents:    []string{"silence-tail", "input-finished"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decoder := &syntheticTailDecoder{}
			gotSamples := 0
			finishOnlineInput(tc.tailMS, sampleRate, func(samples []float32) {
				gotSamples = len(samples)
				decoder.acceptSilence(samples)
			}, decoder.inputFinished)
			if gotSamples != tc.wantSamples {
				t.Errorf("silence samples = %d, want %d", gotSamples, tc.wantSamples)
			}
			if decoder.finalText != tc.wantFinalText {
				t.Errorf("final text = %q, want %q", decoder.finalText, tc.wantFinalText)
			}
			if decoder.invalidTail {
				t.Error("finish tail contained nonzero samples; it must be silence")
			}
			if len(decoder.events) != len(tc.wantEvents) {
				t.Fatalf("finalization events = %v, want %v", decoder.events, tc.wantEvents)
			}
			for i := range tc.wantEvents {
				if decoder.events[i] != tc.wantEvents[i] {
					t.Errorf("finalization events = %v, want %v", decoder.events, tc.wantEvents)
					break
				}
			}
		})
	}
}
