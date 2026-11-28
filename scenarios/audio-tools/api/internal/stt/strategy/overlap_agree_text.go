package strategy

import (
	"strings"

	voice "audio-tools/internal/stt/pipeline"
)

// mergeAgreed merges a newly-agreed run-prefix into the committed text.
//
// Returns (newCommit, tail, ok).
//
//   - ok=false: divergence — agreed is non-empty, does not extend committed
//     as a prefix, AND shares no word-suffix↔prefix overlap with
//     committed. Caller emits Partial only.
//   - newCommit > committed: caller emits Segment with `tail` and stores
//     newCommit.
//   - newCommit unchanged: caller may emit Partial.
//
// Three accepted cases (ok=true):
//
//  1. agreed=="" — no new agreement; returns committed, "", true.
//  2. committed is a prefix of agreed (the classic LocalAgreement
//     growth case) — returns the trimmed agreed text. The candidate
//     tail is run through DeduplicateOverlap before emitting because
//     Whisper sometimes regurgitates the initial_prompt (= committed)
//     at the start of its output; without dedupe the tail would be a
//     literal duplicate of committed.
//  3. committed's tail-words overlap agreed's head-words — happens when
//     the cursor didn't word-advance (no word timestamps) and the next
//     hypothesis covers the same audio. Delegates to
//     pipeline.DeduplicateOverlap.
func mergeAgreed(committed, agreed string) (newCommit, tail string, ok bool) {
	if agreed == "" {
		return committed, "", true
	}
	if committed == "" {
		nc := strings.TrimSpace(agreed)
		return nc, nc, true
	}
	if strings.HasPrefix(agreed, committed) {
		nc := strings.TrimSpace(agreed)
		if len(nc) <= len(committed) {
			return committed, "", true
		}
		candidate := nc[len(committed):]
		candidateTrimmed := strings.TrimSpace(candidate)
		if candidateTrimmed == "" {
			return committed, "", true
		}
		merged := voice.DeduplicateOverlapBounded(committed, candidateTrimmed, maxPromptOverlapWords)
		if len(merged) <= len(committed) {
			return committed, "", true
		}
		return merged, strings.TrimPrefix(merged, committed), true
	}
	if strings.HasPrefix(committed, agreed) {
		return committed, "", true
	}
	merged := voice.DeduplicateOverlapBounded(committed, agreed, maxPromptOverlapWords)
	if merged == committed+" "+agreed {
		return committed, "", false
	}
	if len(merged) <= len(committed) {
		return committed, "", true
	}
	return merged, strings.TrimPrefix(merged, committed), true
}

// appendAfterAdvance is the post-cursor-advance merge function.
//
// When committedAudioBytes has just moved forward via word-aligned
// advance, the next hypothesis covers GENUINELY NEW audio that
// follows what committed represents. mergeAgreed's divergence
// detector would reject this expected state ("no overlap, no
// prefix relationship → wander → REJECT"). appendAfterAdvance
// instead accepts the append, with only the prompt-regurgitation
// defense applied (Whisper occasionally echoes its initial_prompt
// at the start of post-advance output, which DeduplicateOverlap
// will strip).
//
// Returns (newCommit, tail, ok). ok is always true — the caller's
// divergence-reject branch is intentionally unreachable for
// post-advance commits.
//
// Three accepted shapes:
//
//  1. agreed == "" → no change.
//  2. committed == "" → first commit; trimmed agreed becomes
//     newCommit; tail equals newCommit.
//  3. otherwise → DeduplicateOverlap(committed, agreed) merges
//     prompt-regurg overlap if present; pure append otherwise.
//     tail is the new content beyond committed.
func appendAfterAdvance(committed, agreed string) (newCommit, tail string, ok bool) {
	if agreed == "" {
		return committed, "", true
	}
	if committed == "" {
		nc := strings.TrimSpace(agreed)
		return nc, nc, true
	}
	// Whisper can regurgitate the initial_prompt in two shapes:
	//
	//  (a) `committed + " " + newWords`   — full prefix echo
	//  (b) `lastFewCommittedWords + " " + newWords` — partial overlap
	//
	// And sometimes BOTH at once:
	//  `committed + " " + committed + " " + newWords` — full echo
	//  followed by another partial overlap on the second copy.
	//
	// Strip the prefix echo first (shape a), then dedupe overlap on
	// the remainder (shape b). This mirrors mergeAgreed's case-2
	// defense but unconditionally — no divergence rejection.
	candidate := strings.TrimSpace(agreed)
	if strings.HasPrefix(candidate, committed) {
		tail := strings.TrimSpace(candidate[len(committed):])
		if tail == "" {
			return committed, "", true
		}
		merged := voice.DeduplicateOverlapBounded(committed, tail, maxPromptOverlapWords)
		if len(merged) <= len(committed) {
			return committed, "", true
		}
		return merged, strings.TrimPrefix(merged, committed), true
	}
	merged := voice.DeduplicateOverlapBounded(committed, candidate, maxPromptOverlapWords)
	if len(merged) <= len(committed) {
		return committed, "", true
	}
	return merged, strings.TrimPrefix(merged, committed), true
}

// longestAgreedPrefix returns the longest token-level prefix common to
// every transcript in `runs`, when at least `commitRuns` runs are
// present. Token-level (whitespace-separated) so partial words don't
// commit until the next iteration confirms them.
//
// Comparison is normalized via pipeline.NormalizeToken (case-insensitive
// + trailing-punctuation stripped) because Whisper jitters
// capitalization and punctuation across calls even when the underlying
// audio is identical (it re-decides sentence boundaries as more audio
// arrives). Without normalization, "Hello world" vs "hello world."
// would yield zero agreement and the algorithm would never commit.
// The returned prefix uses the FIRST run's tokens verbatim so committed
// text preserves Whisper's chosen capitalization/punctuation rather
// than an arbitrary normalized form.
//
// When maxTokens > 0 the comparison is capped at that many tokens — the
// agreement walk only considers the first maxTokens positions. This
// bounds variance accumulation on long uncommitted buffers: each
// settle attempt asks Whisper to be self-consistent over a fixed text
// length regardless of utterance duration. maxTokens <= 0 means
// unbounded (entire prefix is considered).
func longestAgreedPrefix(runs []string, commitRuns int, maxTokens int) string {
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
	if maxTokens > 0 && minLen > maxTokens {
		minLen = maxTokens
	}
	var prefix []string
	for i := 0; i < minLen; i++ {
		head := voice.NormalizeToken(tokens[0][i])
		ok := true
		for j := 1; j < len(tokens); j++ {
			if voice.NormalizeToken(tokens[j][i]) != head {
				ok = false
				break
			}
		}
		if !ok {
			break
		}
		prefix = append(prefix, tokens[0][i])
	}
	return strings.Join(prefix, " ")
}
