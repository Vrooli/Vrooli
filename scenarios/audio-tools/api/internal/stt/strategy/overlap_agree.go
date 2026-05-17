package strategy

import (
	"context"
	"fmt"
	"strings"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/clock"
)

// OverlapAgree is the LocalAgreement-style streaming strategy
// (Macháček et al. 2023) for batch-only providers. It runs sliding
// overlapping windows over the incoming chunk stream, transcribing
// each window with Provider.Transcribe and committing a prefix to a
// Segment only when it has been observed identically across
// CommitRuns consecutive runs. Uncommitted text is emitted as a
// Partial so consumers see live progress.
//
// The strategy trades CPU per partial for lower end-of-utterance
// latency and live partials on local Whisper. Operators configure
// WindowMs (default 2000) and CommitRuns (default 2) via the lever
// table in docs/reference/configuration.md.
type OverlapAgree struct {
	Provider sttchain.Provider

	// WindowMs is the sliding-window size used for each transcription
	// call. Matches StreamConfig.OverlapWindowMs (default 2000;
	// range 1000–5000).
	WindowMs int
	// CommitRuns is how many consecutive runs must agree on a prefix
	// before it commits Partial → Segment. Matches
	// StreamConfig.OverlapCommitRuns (default 2; range 2–4).
	CommitRuns int
	// SampleRate of the inbound PCM. Default 16000.
	SampleRate int
	// AdvanceMs is the stride between window starts. Default WindowMs/2.
	AdvanceMs int

	// Clock is the wall-clock seam used for per-window latency
	// measurement. Defaults to clock.System{}.
	Clock clock.Clock
}

// Kind reports the strategy kind for selector enforcement.
func (o *OverlapAgree) Kind() sttchain.StrategyKind { return sttchain.StrategyOverlapAgree }

// Run consumes chunks, transcribes overlapping windows, emits Partial
// + Segment events under LocalAgreement, and a terminal Done.
func (o *OverlapAgree) Run(
	ctx context.Context,
	start sttchain.StreamStart,
	chunks <-chan sttchain.AudioChunk,
	events chan<- sttchain.StreamEvent,
) error {
	if o.Provider == nil {
		err := fmt.Errorf("audio-tools/stt/strategy: OverlapAgree requires a Provider")
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
		return err
	}
	o.applyDefaults()

	const sampleBytes = 2
	windowBytes := o.SampleRate * o.WindowMs / 1000 * sampleBytes
	advanceBytes := o.SampleRate * o.AdvanceMs / 1000 * sampleBytes
	if advanceBytes <= 0 {
		advanceBytes = windowBytes / 2
	}

	var buf []byte
	cursor := 0         // start offset of next window to transcribe
	committed := ""     // prefix already emitted as Segments
	var recent []string // last (CommitRuns-1) full-window transcripts for agreement
	var lastTier sttchain.ProviderTier
	var lastProviderID, lastModelID string
	var totalLatencyMs float64

	emitDone := func() {
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{
			FinalText:  committed,
			LockedTier: lastTier,
			ProviderID: lastProviderID,
			ModelID:    lastModelID,
			LatencyMs:  totalLatencyMs,
		}}
	}

	transcribe := func(audio []byte) (*sttchain.Result, error) {
		req := sttchain.Request{
			Audio:                   audio,
			Language:                start.Language,
			InitialPrompt:           committed,
			SkipSpeakerVerification: start.SkipSpeakerVerification,
			BYOKProvider:            start.BYOKProvider,
			BYOKKey:                 start.BYOKKey,
			LPBSToken:               start.LPBSToken,
			UserIdentity:            start.UserIdentity,
		}
		clk := o.Clock
		if clk == nil {
			clk = clock.System{}
		}
		t0 := clk.Now()
		res, err := o.Provider.Transcribe(ctx, req)
		totalLatencyMs += float64(clk.Now().Sub(t0).Milliseconds())
		return res, err
	}

	processWindow := func(end int) {
		if end-cursor < windowBytes {
			return
		}
		audio := make([]byte, end-cursor)
		copy(audio, buf[cursor:end])
		res, err := transcribe(audio)
		if err != nil {
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
			cursor += advanceBytes
			return
		}
		lastTier = res.Tier
		lastProviderID = res.ProviderID
		lastModelID = res.ModelID
		recent = append(recent, res.Text)
		if len(recent) > o.CommitRuns {
			recent = recent[len(recent)-o.CommitRuns:]
		}
		// Find the longest token-prefix agreement across the last
		// CommitRuns runs.
		agreed := longestAgreedPrefix(recent, o.CommitRuns)
		if agreed != "" && !strings.HasPrefix(committed, agreed) && !strings.HasPrefix(agreed, committed) {
			// Diverged on the committed prefix — keep prior commit,
			// emit current window as Partial only.
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventPartial, Partial: &sttchain.PartialEvent{Text: res.Text}}
			cursor += advanceBytes
			return
		}
		if len(agreed) > len(committed) {
			newCommit := strings.TrimSpace(agreed)
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
				Text:             newCommit[len(committed):],
				DetectedLanguage: res.DetectedLanguage,
				ProviderTier:     res.Tier,
				ProviderID:       res.ProviderID,
				ModelID:          res.ModelID,
				LatencyMs:        float64(res.Latency.Milliseconds()),
			}}
			committed = newCommit
		} else if res.Text != "" {
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventPartial, Partial: &sttchain.PartialEvent{Text: res.Text}}
		}
		cursor += advanceBytes
	}

	for {
		select {
		case <-ctx.Done():
			emitDone()
			return ctx.Err()
		case ch, ok := <-chunks:
			if !ok {
				// Final transcribe over the remaining tail (if any).
				if len(buf)-cursor > 0 {
					tail := make([]byte, len(buf)-cursor)
					copy(tail, buf[cursor:])
					res, err := transcribe(tail)
					if err == nil && res.Text != "" {
						newCommit := strings.TrimSpace(res.Text)
						if len(newCommit) > len(committed) {
							events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
								Text:         newCommit[len(committed):],
								ProviderTier: res.Tier,
								ProviderID:   res.ProviderID,
								ModelID:      res.ModelID,
							}}
							committed = newCommit
						}
					}
				}
				emitDone()
				return nil
			}
			buf = append(buf, ch.Audio...)
			for len(buf)-cursor >= windowBytes {
				processWindow(cursor + windowBytes)
			}
		}
	}
}

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
}

// longestAgreedPrefix returns the longest token-level prefix common to
// every transcript in `runs`, when at least `commitRuns` runs are
// present. Token-level (whitespace-separated) so partial words don't
// commit until the next window confirms them.
func longestAgreedPrefix(runs []string, commitRuns int) string {
	if len(runs) < commitRuns {
		return ""
	}
	tokens := make([][]string, len(runs))
	for i, r := range runs {
		tokens[i] = strings.Fields(r)
	}
	minLen := len(tokens[0])
	for _, t := range tokens {
		if len(t) < minLen {
			minLen = len(t)
		}
	}
	var prefix []string
	for i := 0; i < minLen; i++ {
		tok := tokens[0][i]
		ok := true
		for j := 1; j < len(tokens); j++ {
			if tokens[j][i] != tok {
				ok = false
				break
			}
		}
		if !ok {
			break
		}
		prefix = append(prefix, tok)
	}
	return strings.Join(prefix, " ")
}
