package strategy

import "audio-tools/internal/ai/sttchain"

func (o *OverlapAgree) Kind() sttchain.StrategyKind { return sttchain.StrategyOverlapAgree }

func (o *OverlapAgree) applyDefaults() {
	if o.SampleRate == 0 {
		o.SampleRate = 16000
	}
	if o.WindowMs == 0 {
		o.WindowMs = 2000
	}
	if o.CommitRuns < 2 {
		o.CommitRuns = 2
	}
	if o.AdvanceMs == 0 {
		o.AdvanceMs = o.WindowMs / 2
	}
	if o.MaxWindowMs == 0 {
		o.MaxWindowMs = 10000
	}
	if o.MaxAgreedTokens == 0 {
		o.MaxAgreedTokens = 30
	}
	if o.Trigger == "" {
		o.Trigger = TriggerVAD
	}
	if o.SilenceMs == 0 {
		// OverlapAgree settles at SHORTER pauses than VADSegment by
		// design — the whole point of the strategy is to emit Segments
		// mid-utterance, not wait for end-of-turn. 500ms catches natural
		// inter-phrase pauses without firing inside words.
		o.SilenceMs = 500
	}
	if o.SilenceRMS == 0 {
		o.SilenceRMS = 250
	}
	if o.FrameMs == 0 {
		o.FrameMs = 20
	}
}
