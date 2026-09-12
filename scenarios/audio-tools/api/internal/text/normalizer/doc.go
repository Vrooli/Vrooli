// Package normalizer rewrites raw input text into a TTS-friendly /
// summarize-friendly form before it crosses a chain boundary. It is a
// genuinely shared utility: callers must not assume a single owner.
//
// Consumers (kept in sync with `rg -n "text/normalizer" --type go`):
//   - audio-tools/handlers/tts/connect_handler.go — applies normalization
//     to caller-supplied text before synthesis.
//   - audio-tools/internal/summarize/summarization_service.go — applies
//     normalization to the input window before invoking the summarize chain.
//
// If a third consumer appears outside `tts` or `summarize`, revisit the
// 2026-05-17 cleanup decision in docs/internal/DECISIONS.md and consider
// promoting this package (e.g., to `internal/text/` proper) or splitting
// it into per-consumer rule sets.
package normalizer
