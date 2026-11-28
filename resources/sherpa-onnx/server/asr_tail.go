package main

const (
	defaultStreamingFinishTailMS = 1200
	maxStreamingFinishTailMS     = 5000
)

// boundedFinishTailMS keeps end-of-input padding finite even when an operator
// supplies an excessive value. Zero intentionally disables synthetic silence.
func boundedFinishTailMS(tailMS int) int {
	if tailMS < 0 {
		return 0
	}
	if tailMS > maxStreamingFinishTailMS {
		return maxStreamingFinishTailMS
	}
	return tailMS
}

// finishOnlineInput lets online decoders consume a bounded silence tail before
// InputFinished. Some models otherwise discard final tokens when the client
// stops immediately after speech. The callback seam keeps ordering testable
// without requiring native libraries or model weights.
func finishOnlineInput(tailMS, sampleRate int, acceptSilence func([]float32), inputFinished func()) {
	tailMS = boundedFinishTailMS(tailMS)
	if tailMS > 0 && sampleRate > 0 && acceptSilence != nil {
		tail := make([]float32, tailMS*sampleRate/1000)
		acceptSilence(tail)
	}
	if inputFinished != nil {
		inputFinished()
	}
}
