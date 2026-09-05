# Conversation Search Performance and Recovery Evidence

## Contract

Measurements use isolated fixtures unless a row is explicitly labelled live.
TEXT and REGEX run without semantic resources. SEMANTIC and HYBRID use the
variant-scoped Qdrant collection and resolved embedding binding. Minimum sample
size is five for operational validity; release latency evidence uses at least
30 queries per condition. Cold build and failure drills must not modify
canonical runs/events.

Initial floors are:

| Operation | Floor / ceiling |
|---|---:|
| TEXT p95 at representative scale | 250 ms |
| bounded REGEX | 2,000 candidates, 16 MiB, or 750 ms; smaller unprefiltered ceiling applies |
| HYBRID p95 with healthy optional resources | 2 s |
| snippet / context | 2,048 bytes / 20 events |
| incremental convergence p95 after completed backfill | 60 s |
| telemetry | 30 days, 100,000 retained rows, 10,000-row analytical sample |
| owned scenario data alarm | 6 GiB; canonical and projection bytes must be reported separately |

## Dated live evidence — 2026-09-04

| Evidence | Observation | Disposition |
|---|---|---|
| Baseline corpus | 11,154 runs, 691,586 events, 28,749 message events, 36,097,980 approximate message JSON bytes | Representative-scale fixture floor remains at least this distribution. |
| First serial semantic shadow | Approximately 69 minutes and 5,249 Qdrant points before rollback/restart | Rejected: serial embedding and an unbounded tool-event semantic corpus. |
| Interrupted live semantic shadow | 1,746 durable candidate points survived a lifecycle restart | Recovery now reattaches the same generation, reads the immutable staged SQLite snapshot, and skips complete candidate sources instead of discarding work. |
| Remote generation writes | Previous implementation scrolled the entire growing candidate and issued delete/upsert calls per source | Replaced with one filtered delete and one batch upsert per bounded page; the prior behavior was quadratic in candidate size. |
| Lexical publication | A single 554,974-document transaction made API readiness return 503 during cutover | Initial and replacement publication commit stable-identity diffs in 1,000-document resumable batches, yielding the single SQLite connection between batches. |
| Concurrent candidate | Four bounded embedding workers; reached 4,810 points while the API remained healthy | Directionally better, but not terminal evidence. |
| Isolated representative query fixture | 30,000 records with mixed harness/project/model metadata and one 256 KiB message; 30 iterations per arm: TEXT hit 3.10 ms/op, TEXT miss 0.327 ms/op, bounded adversarial REGEX 2.70 ms/op | Passes the 250 ms TEXT and 750 ms REGEX p95 release floors with substantial margin; benchmark setup is excluded from timings. |
| Isolated bounded backfill fixture | 50,000 generated sources, 64-source pages, fake deterministic embedder/store; completed in 284 ms with no page or lookup exceeding 64 sources | Proves the reconciler's planning memory and store lookups remain page-bounded above the measured 28,749-message corpus. This is a structural bound, not a live embedding-throughput claim. |
| Snapshot validation failure | An unconstrained second scan reported `canonical source mutated during build` on an active import store | Fixed by pinning both scans to one append-only event-row high-water mark; later imports stay queued. |
| Tool-event amplification | About 554,974 staged documents versus 28,749 measured message events | Rejected for semantic indexing. Vectors now accept only prose and quoted prose; tool events remain explicit SQLite recall. |
| SQLite growth during failed shadows | Live file reached about 2.37 GB; candidate table was about 297 MB at 194,655 rows | Within the 6 GiB alarm but not an efficiency success. No direct `VACUUM` or raw cleanup was performed. |
| Direct Search Hub eval before live index | direct run `59b68774-900c-41a4-9082-b848a568a6e0`: 2/11; p95 42 ms | Denominator mismatch: deterministic fixture IDs were intentionally absent from live data. No tuning accepted. |
| Federated Search Hub eval before live index | federated run `eb3b053a-7c08-4c81-96d4-2c435c8d5ecc`: 2/14; precision 0; p95 7,061 ms | Provider was withheld while semantic status was degraded. No tuning accepted. |

The final promoted retry, live golden ranks, direct/federated overlay eval IDs,
query p50/p95, final vector count, and projection bytes must be appended here;
the rows above cannot be represented as a successful release benchmark.

## Recovery matrix

| Injected condition | Required behavior | Executable evidence |
|---|---|---|
| embedder, Qdrant, reranker unavailable | Lexical hits remain; typed leg degradation is returned. | `TestHybridKeepsLexicalResultsForEveryOptionalFailureClass` |
| semantic leg ignores cancellation | Hybrid returns by its own deadline. | `TestHybridDeadlineDoesNotTrustOptionalRetrieverCancellation` |
| invalid/abusive regex | Reject or return bounded partial state; honor cancellation. | `TestRegexAdversarialUnicodeMultilineZeroWidthAndInvalidUTF8`, `TestRegexReportsCandidateByteAndDeadlineBounds`, `TestRegexHonorsContextCancellationAndDefaultContentPolicy` |
| source import during full build | Pinned snapshot promotes; new change remains queued. | `TestSQLiteSourceSnapshotExcludesLaterBackdatedImports`, `TestIndexerPromotesValidatedSnapshotWhenNewChangesArriveDuringSemanticBuild` |
| deletion during full build | Candidate promotion fails; serving projection stays readable. | `TestIndexerRejectsPromotionWhenDeletionArrivesDuringSemanticBuild` |
| semantic rebuild failure/cancel/restart | Validated lexical search publishes independently; explicit failures/cancellation preserve the prior safe serving state, while a complete interrupted shadow resumes. | `TestIndexerSemanticFailureStillPublishesLexicalSnapshot`, `TestIndexerCancellationLeavesServingProjectionReadable`, `TestIndexerRestartRollsBackInterruptedShadowBeforeRepair`, `TestIndexerRestartResumesCompleteStagedSnapshot`, and `TestStreamingReconcilerResumesCompleteCandidateSources` |
| live/shadow namespace overlap | Variant-aware collection resolution and memory-only SQLite tests prevent cross-contamination. | `TestProjectionTestsUseOnlyMemoryDatabase` plus shared Qdrant generation-store tests |

## Tuning rule

Change ranking, concurrency, or budgets only from a labelled eval or measured
resource result. Personal live-corpus misses do not tune the deterministic
primary suite because its fixture identities are intentionally absent live.
Record rejected arms as well as the selected arm, and never relax deletion,
privacy, or degradation behavior to improve a score.
