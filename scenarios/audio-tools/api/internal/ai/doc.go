// Package ai groups the canonical per-capability provider-routing
// chains for the audio-tools scenario. After the 2026-05-17 audit (see
// docs/internal/DECISIONS.md), the layering invariant is:
//
//	internal/ai/{sttchain,ttschain,summarizechain} — orchestrators that
//	  own provider selection across the BYOK → Vrooli → Local tiers.
//	  Consumed by handlers/{stt,tts,summarize} and bootstrap.
//
//	internal/{stt,tts,summarize} — domain primitives (segmenter,
//	  strategy, cache, normalizer, summarizer). Used by the chains as
//	  building blocks; never call back into the chains.
//
//	internal/ai/chains — generic per-capability chain primitives shared
//	  across stt/tts/summarize (capability registry, error wrapping).
//
// If a primitive package starts importing from internal/ai/*chain the
// layering invariant is broken — fix the import direction, do not relax
// the rule.
package ai
