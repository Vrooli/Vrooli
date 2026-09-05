// Package summarize hosts the summarize domain's primitives and
// orchestrator. After the 2026-05-17 post-extraction audit (see
// docs/internal/DECISIONS.md) the two-file split is intentional:
//
//   - summarizer.go: low-level Ollama /api/chat client. Knows nothing
//     about provider routing, normalization, concurrency, or backoff.
//     Used both by SummarizationService here and by the local provider
//     in internal/ai/summarizechain.
//
//   - summarization_service.go: domain orchestrator. Normalizes input
//     via internal/text/normalizer, gates concurrency, applies the
//     auto-summarize failure cooldown, and dispatches via the summarize
//     chain. This is the type that handlers/summarize embeds.
//
// Provider routing (BYOK → Vrooli → Local) lives one level up in
// internal/ai/summarizechain; this package owns only the
// summarize-specific glue.
package summarize
