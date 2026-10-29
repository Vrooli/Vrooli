# Debt and consolidation ledger

Each row is closed by deleting or unifying code, not by adding a layer. Record
the before/after measure and the evidence when closing a row. Paths are relative
to `scenarios/agent-manager/`.

## Baseline (2026-10-29)

| Measure | Baseline |
|---|---|
| API Go LOC (non-test / test) | 151,236 / 133,167 |
| Analytics core non-test LOC (runsignal, pricing, measures, runreport, invocationreadmodel, stats, metrics, tokenaccounting, repository_invocation_read_model, repository_stats, orchestration/stats, handlers/stats, token_cap) | ~17.6k |
| Event-sink constructions with differing hook sets | 6 |
| Distinct "total tokens for a run" computations | 8 |
| Capture mechanisms (attach, snapshot import, managed-skip, tombstone eviction, transcript copy) | 5 |

Re-measure with `find <dirs> -name '*.go' ! -name '*_test.go' | xargs wc -l` and
record the after values next to each row.

## W1 measures (non-test LOC, 2026-10-29, before → after)

| Area | Before | After |
|---|---|---|
| runsignal | 2,769 | 2,895 (`usage_selection.go` 106 → `usage_accounting.go` 239) |
| invocationreadmodel | 1,208 | 1,207 |
| runreport | 1,243 | 1,208 |
| tokenaccounting | 145 | 145 |
| codecs `codex.go` + `codex_interrupted_usage.go` | 1,641 | 1,663 |
| orchestration W1 files (invocation_read_model, token_cap, run_continuation, workflow_metering, workflow_terminal_receipt, run_report, run_lifecycle, recovery) | 3,915 | 3,970 |
| runner `interface.go` + `core/runner.go` | 2,250 | 2,238 |
| Distinct "total tokens for a run" computations | 8 | 1 |

Net is up ~+230 lines, mostly the D-4 durable queue (schema, repository,
sweep, marks) and the richer account (sessions, messages, per-turn view).

## A9 after-measures (2026-10-29, orchestrator, after W8a2)

| Measure | Baseline | After | Note |
|---|---|---|---|
| API Go LOC non-test / test | 151,236 / 133,167 | 155,968 / 136,200 | +4,732 non-test: new capability packages below (+5,180) and capture growth (D-6) outweigh deletions (REST stats −1,700, cohort/efficiency −600, delta machinery, pricing fallback) |
| Analytics core (baseline set) | ~17.6k | ~13.6k | baseline set minus deleted stats layer; −4.0k |
| New capability packages (usageforecast / usageattribution / transcriptindex) | 0 | 1,283 / 1,931 / 1,966 | forecast, attribution + rules, indexer; none existed |
| Event-sink constructions with differing hook sets | 6 | 1 | D-1 |
| Distinct "total tokens for a run" computations | 8 | 1 | D-5 |
| Capture mechanisms | 5 | 2 | D-6 (indexer; attach = identity only) |
| quota_observations rows (live) | 61.5k (pre back-fill) / 2.21M (post back-fill) | 133,895 | W4b dedupe + compaction |
| UI non-test TS/TSX LOC | 28,390 (pre-W6c) | 29,298 | Usage page, strip, run panel, health state, skeletons |

**A9 verdict:** every row D-1…D-16 is closed except D-15 (temporary repair code, delete once `transcript_usage_repairs` holds its marker and no legacy rows remain). Net non-test LOC is **below baseline in the pre-existing analytics core (−4.0k)** but **above baseline for the API as a whole (+4.7k)**, because the effort adds three capability packages that had no prior equivalent. The "net non-test LOC below baseline" bar is therefore met only for the areas that existed before; the operator should judge whether the new capability justifies the growth.

## Rows

| ID | Debt | Target | WS | Status |
|---|---|---|---|---|
| D-1 | Six sink variants; only `runEventSink` applies hooks (`run_operations.go:604`, `run_executor.go:717`, `run_execution.go:456`, `recovery.go:400`, `import_transcript.go:1042`, `run_operations.go:484`) | One constructor that always applies the hooks; tests exercise the real executor, interactive and recovery paths | W3 | closed 2026-10-29: `newRunEventSink(hooks *Orchestrator, …)` is the only sink constructor and always applies `afterRunEventPersisted` (quota, nudge, token cap); executor gets `hooks` in `newRunExecutor`, Reconciler in `SetReconciler`; `interactiveEventSink` deleted; `appendAndBroadcastEvents` applies the same set; import applies quota only (read-only runs). Sink variants 6 → 1. Tests: `TestEveryLiveSinkPathRecordsProviderQuotaObservations`, `TestTokenCapNudgesOnceAt85PercentThenStopsWithTokenCapReason/{headless executor,interactive tail,recovery}`, `TestImportRecordsQuotaButNeverActsOnImportedRuns`, `TestImportTranscriptRecordsCodexQuotaFrames` |
| D-2 | Quota frames stored as `EventTypeError` (`domain/run_configuration.go:432-450`); polluted event counts and the `type=error` filter | Own event type plus a migration of ~40k rows; consumers split by type, not payload sniffing | W3 | closed 2026-10-29: `EventTypeQuota`/`RUN_EVENT_TYPE_QUOTA`; start-up `migrateQuotaEvents` moves the 63,340 live legacy rows (39,995 tagged + 23,345 untagged) and back-fills `quota_observations` in one idempotent transaction; deleted `decodeErrorPayload`, `IsOperationalErrorEvent`, `PayloadKindQuota`, run-report error breakdown (proto field 31 reserved) |
| D-3 | Codex delta/baseline machinery (`rolloutBaselineKnown`, `rolloutReceiptUsage`, `SetContinuationUsageBaseline`, `continuationCodexUsageBaseline`, `CodexUsageSnapshot`, receipt-vs-delta authority) | Raw cumulative counters plus one derivation (SPEC T-2); delete the listed code | W1 | closed 2026-10-29: all listed code deleted (plus `ReconciliationComplete`, `CodexUsageBaseline`, `ContinueRequest.CodexUsageBaseline`, `rolloutUsageDelta`, `codexUsageDelta`, `codexThreadTotalDelta`, `SelectCanonicalUsage`, `CanonicalUsageProjection`). Usage events carry `sessionId` + `cumulative`; `runsignal.AccountUsage` derives totals. Kept: `ReconciliationAuthority` as the invocation-closed marker workflow metering needs, and the print-mode Claude receipt rule (a real invocation total). `terminalWorkflowReceiptProjection` now feeds charge completeness only. Proof: `codecs/usage_accounting_test.go`, `interactive/usage_accounting_test.go`, `orchestration/usage_accounting_test.go`; live `TestLiveUsageComparison` shows fresh and per-turn re-tail account = disk final for 2d230f9d (745,410,641), 1ced66dd (1,147,786,266), aee95b01 (284,138,869) |
| D-4 | Fire-and-forget projection (`invocation_read_model.go:543-552`); Reconciler terminal writes skip projection | Durable needs-projection mark plus an idempotent sweep | W1 | closed 2026-10-29: goroutine deleted; `invocation_read_model_pending` (generation, attempts, backoff) marked by lifecycle, executor observer, Reconciler writes (`recovery.go` success/failure/interruption, sandbox approve/reject, `markRunFailed`) and recovered terminal receipts; reconcile tick runs `ProjectPendingInvocationReadModels`, which also enqueues ended runs with a missing or pre-`invocation-fact.v20` watermark (re-projects history after restart). Needs a restart to take effect |
| D-5 | 8 token-total computations: `invocationreadmodel/project.go:513`, `runsignal/time_accounting.go:103`, `runsignal/episode.go:110` (no dedup), `runreport/report.go:266,290,455,487`, `orchestration/token_cap.go:163`, `workflow_execution.go:235`, `supervision/effort_service.go:597`, `protoconv/entities_events.go:144` | One accounting function; the others call it or are deleted | W1 | closed 2026-10-29: read model, time accounting, episodes (computed once per run), report total and goal-before-first-met (was double-added), incurred attribution, token-cap recount, workflow metering, `RunAccounting`, terminal run summary (lifecycle recomputes `TokensUsed`) and effort usage (`EffortSpendReader.RunTokens`) read `runsignal.AccountUsage`. Kept by decision: `protoconv` maps one event's own amount (a per-event view, not a run total); the token-cap live estimate stays a per-event upper bound that triggers the exact recount |
| D-6 | Snapshot import + tombstones + managed-session skip (`transcript_scheduler.go:302,460`) + transcript copies (5.6 GB) + eviction measured on the whole DB (`reconciler_storage.go:55-99`) | Transcript Indexer (SPEC T-1); recall budget measured on its own tables | W2 | closed 2026-10-29 (live-unverified): `transcript_scheduler.go` deleted; `ImportTranscript` is now a manual call into the indexer (`transcriptindex.Indexer.IndexPath`); snapshot copy, `rehydrateImportedTranscript`, `importEventSink`, managed-session skip, web-console budget env, `evictImportedCorpus`/`reclaimSearchIndex`/`liveBytesFn` and the repository methods `ListImportedRunIDsOldestFirst`/`DeleteImportedRun`/`DeleteEmptyImportedRuns` deleted. Recall budget (4 GiB) measured on run_events + search rows of indexed runs (`TranscriptIndexRepository.RecallBytes`). Legacy imported runs are relinked to their source and their copies removed. Capture mechanisms 5 → 2 (indexer; attach = identity only). Capture non-test LOC ≈1,890 → 3,147 (+1,257: durable usage rows, AM-3 facts, coverage, recall rebuild are new capability). Tests: `transcriptindex/indexer_test.go` |
| D-7 | Attached runs marked `failed` on every normal exit (13,799 failed) | Exit is `complete`; failure only for real failures | W2 | closed 2026-10-29 (live-unverified): reconciler `completeAttachedRun` (complete, token revoked, `attached_run_exited` event); start-up `migrateAttachedExitStatus` corrects the 13,813 live rows whose only failure was the exit and queues their re-projection; other failures stay. Tests: `TestAttachedRunLivenessSweepOnlyObservesHarmlessProcess`, `TestMigrateAttachedExitStatusCompletesNormalExitsOnly` |
| D-8 | Session keying collides fork files and Claude sidechains on the parent id | Keyed by file identity, with parent linkage kept | W2 | closed 2026-10-29: `transcript_files.file_key = dev:inode`; fork/sidechain rows keep `parent_session_id`/`parent_file_key`, their runs get `parentRunId`; provenance session id is the file's own (fork child id / sidechain agent id). Test: `TestIndexerTailsGrowingFilesIncrementallyByFileKey` |
| D-9 | REST stats layer (`handlers/stats.go`, `orchestration/stats.go`, `repository_stats.go`, mocks; ~1.8k LOC, 14 routes) duplicates Measures; the UI `STATS_ENDPOINTS` constant is dead | Delete after a fleet consumer search (SPEC D4); `run stats` reads Measures | W7 | closed 2026-10-29 (W7): REST stats layer deleted after a fleet consumer search (no callers outside AM); `repository_stats.go` reduced to the 2 pricing model-ranking reads; `run stats` reads 8 Measures procedures. ≈ −1,700 non-test / −770 test LOC |
| D-10 | Measures missing from `agentReadPOSTRoute` (`handlers/authorization_routes.go:354-414`) → 403 for agents | Read procedures declared once, so the allowlist is derived rather than hand-listed if feasible | W7 | closed 2026-10-29 (W7): agent read access derived from the MeasuresService descriptor; `TestEveryMeasuresProcedureIsAgentReadable` covers every procedure incl. the 6 usage measures |
| D-11 | Overlapping CLI families: `run invocation-*`, `replay/refresh-invocation-*` (6), cohort commands (`cohort-report`, `goal-cohort`, `cohort-compare`, `episode-cohort`, `cohort`), `run efficiency`, `run tokens` | Fold into `measures` / `usage` with one cohort selector; delete superseded commands and handlers | W7 | closed 2026-10-29 (W7b + W8a parts 1–2); history: W7b: invocation-aggregate/cohort/metrics, cohort-report, goal-cohort, cohort-compare, episode-cohort, efficiency deleted (CLI+REST); Measures CohortReport/CompareEpisodeCohorts + selector (`--cohort|--run-ids|--filter-json`) on select-cohort/episode-cohort/cohort-report/episode-compare. W8a 2026-10-29: `run cohort define|list|show|delete` folded into `measures cohort define|list|show|delete` (old command, `RunService` cohort methods and manifest entry deleted; the `/api/v1/runs/cohorts` REST routes stay because the new command uses them and `measures.proto` has no definition RPCs), selector precedence/validation unit-tested (`api/internal/measures/cohort_test.go`). W8a2 closed 2026-10-29: `agent-manager usage run <id> [--children] [--weights] [--facts]` replaces `run tokens` and `run invocation-facts`; `agent-manager usage reindex [<run-id>] [filters] [--full]` (refresh by default) replaces `run replay-invocation-facts`, `run refresh-invocation-facts` and `run replay-invocation-corpus`. Deleted: the five `run` commands and manifest entries, CLI `RunService.ReplayInvocationFacts/RefreshInvocationFacts`, REST `POST /api/v1/runs/{id}/invocation-facts/replay|refresh` with their handlers and the `InvocationFactService.RefreshInvocationFacts` interface method; the corpus route takes `run_id`. `GET /runs/{id}/invocation-facts` stays (`usage run --facts`, `run cross-scenario`). Skills, docs, manifest, primitive evidence and endpoints.json updated; `run import-sweep` text names the indexer pass. Tests: `TestUsageReindexQuery`, `TestUsageGroupOwnsRunUsageCommands`, single-run corpus refresh/full in `TestReplayInvocationFacts_RebuildsRetainedAndPreservesPruned` |
| D-12 | Claude codec detects quota by error text only (`claude_events.go:638-650`) | Parse `quotaLimits`; statusline feed per SPEC D3 | W3 | closed 2026-10-29: Claude codec emits `quota` events from `quotaLimits` (pools `five_hour`/`seven_day`); `resource-claude-code statusline run|install|uninstall|status`, reconciled by `install-direct`; `pricing.ClaudeStatuslineReader` ingests samples; `docs/RATE_LIMIT_TRACKING.md` deleted |
| D-13 | Docs contradict code: `docs/concepts/DATA.md:119-123`, `docs/reference/effort-supervision.md:454-468`; requirements 22/23 status | Fixed with the code that changes them | W0/W8 | closed 2026-10-29 (W0; confirmed W8): DATA.md, effort-supervision.md, EVENT_TAXONOMY.md corrected; module 22 folded/removed, 23 re-statused; docs phase re-run in W8 (A10) |
| D-14 | `baseline/` scripts in this folder | Delete when `usage coverage` and the accounting regression replace them | W8 | closed 2026-10-29: folder deleted under operator deletion grant deletion-grant-8857de941a2392383683a201de96f0d9. Replacements: `usage coverage` (A1 scan `transcriptindex/live_coverage_test.go`), `usage burners`, `usage forecast --backtest`, A2 regressions (`codecs/usage_accounting_test.go`, `TestLiveUsageComparison`) |
| D-15 | Usage rows indexed before 2026-10-29 carry the indexing time as `observed_at` and `unknown` as model (2.2M rows; Codex codec stamped usage with parse time and never read `turn_context.model`, Claude ignored `message.model`) | Codecs stamp line time and the turn's model (fixed, W5); a one-time start-up repair rewrote legacy rows from the transcript by line offset (160,573 rows repaired on this host) | W5 | closed in Pass 2 (P2-W6): the repair, its repository methods and the marker table declaration are deleted (the 109 remaining `unknown` rows have no model in their transcripts); see the Pass 2 D-15 row |
| D-16 | Attribution reads two sources: indexed rows for operator sessions and `run_events` for Agent Manager runs, because run-owned transcripts produce no `transcript_usage` rows; Codex run events written before cumulative counters (2026-10-29) overcount up to 50× and are excluded, so their window share is unattributed (17.4 of 34 points in the current Codex window) | The indexer records usage rows (summaries only) for run-owned files too; attribution reads `transcript_usage` alone and `runTurns` is deleted | W2/W8 | closed 2026-10-29 (W8a2; live-unverified: needs restart): run-owned files are tailed for usage only (no events, quota, run updates or recall drops), keyed to the owning run; a run-owned row never indexed is read once from offset 0; `runTurns`, `runEventRow` and `legacyCodexUsage` deleted; `UsageTurns` reads `transcript_usage` alone. Live read-only dry run (`USAGE_ATTRIBUTION_LIVE=1`): 81 run-owned files, 1.16 GB, 34,529 rows, 0 host writes; Codex weekly 34 = 16.58 + 17.42 unattributed → 30.99 + 3.01 (sums equal the observed increase; Claude windows unchanged). Tests: `TestIndexerRecordsUsageOnlyForRunOwnedFiles`, `TestUsageTurnsCountEachReportOnceFromTranscriptUsage` |


## W5 measures (non-test LOC, 2026-10-29)

| Area | Before | After |
|---|---|---|
| usageattribution (new) | 0 | 1,788 |
| measures `usage_attribution.go` (new) | 0 | 362 |
| database `repository_usage_attribution.go` (new; includes D-15 repair methods) | 0 | 542 |
| transcriptindex `repair.go` (new, D-15, temporary) | 0 | 212 |
| CLI `domains/usage/attribution.go` (new) | 0 | 263 |

All new capability (T-5, T-6, A1 report, A11 compare). Nothing is superseded:
`friction.token_attribution` attributes tokens to capabilities and command
paths from invocation facts, which burners (quota points by session and
dimension) do not replace; it stays.

## W8a part 2 measures (non-test LOC, 2026-10-29)

| Area | Before | After |
|---|---|---|
| database `repository_usage_attribution.go` (D-16: `runTurns` deleted) | 558 | 455 |
| CLI `cmd_invocation_facts.go` (four commands deleted) | 532 | 434 |
| CLI `cmd_run_tokens.go` → `cmd_usage_run.go` (`usage run` + `usage reindex`) | 143 | 223 |
| handlers `invocation_replay.go` (per-run replay/refresh deleted, `run_id` added) | ≈225 | 206 |
| measures `usage_registry.go` (new, F-W4-4) | 0 | 231 |
| usageattribution `rules.go` (F-ORCH-8 projection + basis; handoff skip deleted) | 685 | 748 |
| transcriptindex, codecs, forecast, runsignal (D-16 owned tail, F-W2-2, F-W5-5, F-W1-2, F-ORCH-7, F-W5-4) | — | ≈ +110 |

| pricing historical-average fallback (`applyHistorical`, `hasNilComponents`, `HistoricalPricing`, `SourceHistoricalAverage`, `GetHistoricalAverages` query; 0 matching rows ever on this host) | ≈95 | 0 |

Net ≈ +240 non-test LOC (with the `PriceBook`, ≈ +125, and the pricing deletion, ≈ −95). The deletions (D-16 second source, five CLI commands,
two REST routes, the separate Codex receipt charge) are larger than the code
they leave behind; the growth is the measures registry for the six usage
procedures (new capability) and the stated recommendation basis.

Add rows when a worker finds debt in its area. Record a row's closure in the
same change as the code.

## Pass 2 baseline (2026-10-29, before any pass-2 change)

| Measure | Baseline |
|---|---|
| API Go non-test / test LOC (`find api -name '*.go' ! -name '*_test.go' ! -path '*/gen/*'`) | 155,996 / 136,237 |
| CLI Go non-test / test LOC | 16,311 / 8,663 |
| UI non-test / test LOC (`ui/src` .ts/.tsx excluding tests, test-utils, __tests__) | 29,272 / 12,726 |
| eslint errors in `ui/src` | 21 (8 test files) |
| AgentManagerService RPCs without an AM CLI/UI caller | 55 of 104 |
| REST routes in `handlers.go RegisterRoutes` | 120 |
| Details 7-day tokens vs `transcript_usage` | 286–337 B vs 31–32 B (≈9×) |
| Fixture runs from web-console tests | 3,633 |

## Pass 2 rows

| ID | Debt | Target | WS | Status |
|---|---|---|---|---|
| P2-1 | Windowed totals sum per-run lifetime counters at `ended_at` (`invocationreadmodel/project.go:420`, readers `RunMetrics`, `ChargeByBasis`, `RunCostDistribution`, `RunBreakdown`, `ToolUsage`, `CapabilityUsage`, `TokenAttribution`, `CohortRows`, `repository_usage_attribution.go:372`); 247 duplicate imported runs (67 B); zero-token imported runs counted as runs; `runCycleTime` Denom unset (`measures.go:303`), `runCost` Denom = tokens | Windowed token/quota totals from `transcript_usage` only; AM-executed scope for success/throughput/time (P2-D8); explicit Denom per kind; duplicates explained and fixed | P2-W1 | closed 2026-10-29 (P2-W1; live after restart 7): windowed totals from `transcript_usage` by `observed_at` (7-day 30,893,007,112 = transcript sum); `RunScope`; Denom per kind, p50/p90; duplicates counted once; dollars only for metered runs. Follow-up fixes: duplicated run events repaired (two marker-gated background passes, 23,719 + 102 events) and prevented by deterministic event IDs (P2-W2) |
| P2-2 | Second stats engine `internal/stats` (+schema) behind REST `/stats/operational`, `/fallback`, `/stats/run-classes`, CLI ops, UI operationalClient/RunClassBreakdown | Fallback/model-failure insights as Measures (Health item, P2-D10); engine, routes, CLI services and UI clients deleted | P2-W1 / P2-W5a | closed 2026-10-29: `FallbackInsights` Measures RPC + Health "Fallbacks · last 7 days" card with partial index `idx_run_events_fallback` (0.7 s live); `internal/stats` (+schema), REST `/stats/operational`, `/fallback`, `/stats/run-classes`, CLI ops and UI operational/run-class clients deleted; `stats_checkpoint` dropped (marker-gated) |
| P2-3 | Findings: generic-text generator, never-used review fields (operator_decision, effectiveness, before/after), Findings page; friction shown in 4 places | Friction measure + Usage › Friction (P2-D1); generator, fields and page deleted | P2-W4 / P2-W5a | closed 2026-10-29: Usage › Friction replaces Findings (`/findings` redirects); per-pattern wasted-time cost, classifier v13 (self-pairs, pipeline reads, stall bounded 5 min–2 h within a turn); Report bug (scenario-qa, scenario/command owners) / Suggest a fix (agent behaviour) / Dismiss; generator, review fields and 888 generic rows deleted |
| P2-4 | Recommendations flat, savings double-counted across levers, raw metric dump in UI ("Still running 1 bool") | Server-side groups with combined saving and typed facts | P2-W4 | closed 2026-10-29: plans with combined, overlap-free and recoverable-before-reset savings; cross-cut friction plans never summed; UI renders groups and the focus band |
| P2-5 | Label helpers duplicated with different output (`usageforecast/alert.go` "5h", CLI `register.go`, UI 5 copies) | One owner per side, one rule (P2-D12) | P2-W3 / P2-W5b | closed 2026-10-29: `usagelabel` owns Go labels (CLI + alerts, ±2 min rounding); `lib/format.ts` owns UI labels (P2-D12) |
| P2-6 | Web-console test writes fixtures into the real sessions dir; 3,633 imported fixture runs | Test uses a temp dir; indexer skips test output; rows and dirs deleted under grant | P2-W2 | closed 2026-10-29 (web-console tests isolated; indexer skips test rollouts; 4,700 dirs and 4,700 runs deleted by the agent under operator-approved grants, see `p2-fixture-inventory.md` Outcome; derived rows of deleted runs are now removed with the run) |
| P2-7 | Routes with no caller: `/investigations/{id}/learning-retry`, `/runs/{id}/audit-transcript`, `/model-policy/canary/arm`, `/compare` | Deleted | P2-W6 | closed except `learning-retry`. Searched `scenarios/*`, `packages/*`, `.vrooli`, skills and docs: `/model-policy/canary/*` (handler, `CompareCanary`, metrics types), `/runs/{id}/audit-transcript` (handler; its one doc reference, TRIAGE.md, now points at `run messages`) and `/runs/investigate` (an always-400 "retired" stub; Prompt Manager has its own route of that path, not this one) are deleted with their tests and `endpoints.json` entries. **`learning-retry` stays**: it is the only way to replay a `pending` investigation learning capture (set on every completed investigation), the Agent Manager skill tells agents to call it, and no other path retries it. `invocationreadmodel.CanaryRuns/CanaryStore` are now unused (P2-W1 files) |
| P2-8 | 16 start-up `migrate*` functions (502 LOC) incl. pass-1 one-shots run every start | Declarative schema; one-shots gated by a marker or removed | P2-W6 | closed. `connection.go` 1,384 → 637 LOC. **Rule: a one-shot is removed only when every install that follows the branch has run it (at least two weeks old).** Kept, marker-gated: `migrateQuotaEvents` and `migrateAttachedExitStatus` (added 2026-10-29, so other installs such as minimouse have not run them) now run once per database through `runOnce` (`migrations_once.go`), which records the name in `schema_migrations_done` in the migration's own transaction; later starts do one marker lookup and no scan. The orphan `stats_checkpoint` and `transcript_usage_repairs` tables are dropped the same way. Removed (weeks old): `consolidateLegacyInvestigationProjections`, `removeRetiredInvocationReadModelColumns`, the fact segment-key rebuild, `migrateRunTaskIDNullability` (192 lines), the two watch-action migrations and the investigation-sentinel delete — every live table already has the target shape (legacy `investigation_*` projection tables gone, `runs.task_id` nullable, `cohort_watch_actions` has `state`, no `status`, nullable `decision_id`, `invocation_read_model_runs` without the retired columns) and a fresh install creates it from the declarative schema. The four `runs` indexes moved into `runmodel/schema.sql`. Kept: the additive column migrations as one table-driven list (`additiveColumnMigrations`), applied once and only to tables that exist, before `EnsureSchemas`; and `retireRunFindingReviewState` (the finding-column drop has not run on this host yet: it ships with P2-W4's restart). Tests: `TestQuotaEventMigrationRunsOncePerDatabase`, `TestAttachedExitStatusMigrationRunsOncePerDatabase`, `TestRetiredBookkeepingTablesAreDroppedOnce`, `TestAdditiveColumnMigrationAddsMissingRunColumns`, `TestInitSchemaCreatesRunIndexesAndUpgradesLegacyColumnsIdempotently`. **Consequence**: an install older than those cutovers (other fleet nodes' Agent Manager DBs, if any predate them) must first run a build that still has them; this host needs nothing |
| P2-9 | Three pricing paths (`pricing.Service`, `usageattribution.Price` fallbacks, PriceBook) | One owner; tier/cache defaults a named fallback source | P2-W6 | closed. `pricing/fallback.go` owns `FallbackTokenPrices`, the tier priors (`TierTokenPrices`) and the cache-ratio defaults (`WithDefaultCacheRatios`); `usageattribution` keeps its `Price` shape but holds no constants (`model.go` tier/fallback/ratio code now delegates), and `PriceBook` is a memo over `pricing.Service` |
| P2-10 | Codec classifiers ×3 hooks per runner; 3 outcome classifiers | One codec classifier, one outcome function | P2-W6 | closed where they truly overlapped. Each codec now has one session-loss detector (`sessionLoss` on `baseCodec`); `ClassifyTerminalError` and `Classify` both derive from it (codex, claude, opencode, grok; antigravity's no-op copies of base defaults deleted). Not merged: `ClassifyTerminalError` stays separate from `Classify` because `Classify` also types generic text such as "session expired, please reauthenticate", which must not become a typed session-loss stop. Behaviour change: OpenCode `Classify` is now case-insensitive like its terminal classifier (the live CLI prints "Session not found"). Outcome: `domain.ClassifyRunOutcome` (dead cancelled/timeout parameters removed) and `domain.ClassifyErrorOutcome` replace `phases.classifyOutcome` and `phases.ClassifyErrorOutcome`. `PostClassify` is a separate stage (result rewrite), not a classifier |
| P2-11 | Dead API funcs (NewControlPlaneInventory, WithGo*, CrossScenario, AllMeasureDefinitions, AppliedActionStats, SearchLexical, DeleteStagedRun, EventDocumentIDs, placeholder) and 26 unused test builders; 412 hand-written run literals | Deleted; one run builder adopted where tests are touched | P2-W6 | closed except two. Deleted: `NewControlPlaneInventory`, `WithGoBaseURL/TTL/HTTPClient` (+ the option type), `runsignal.CrossScenario` (+ file), `AppliedActionStats`, `DeleteStagedRun`, `EventDocumentIDs`, and 18 unused fixture options in `orchestration/testutil/fixtures` (`NewRunEvent` lost its option hook). **Kept**: `AllMeasureDefinitions` is a Connect RPC implementation (proto-backed) and `SearchLexical` implements `aisearch.LexicalSearcher` (grep showed no caller because the fusion type calls it through the interface). **Left to P2-W1**: `_ = workloadInstance` in `invocationreadmodel/project.go`. The run builder is `fixtures.NewRun` (already used by 9 files); the 412 literals were not churned |
| P2-12 | God functions: `createRun` 736, `runDurable` 372, `reconcileDiscovery` 306, `NewOrchestrator` 298 | Split into named phases | P2-W6 | closed. `createRun` 50 lines over `createRunFlow` (prepareRequest, replayAccepted, admitCapacity, loadTask, resolveConfiguration, evaluatePolicy, deriveRunMode, applySpawnPolicy, validateAdmission, resolveExistingSandbox, freezeRunContract, authorizeDelegation, reserveIdempotency, buildRun, persistAttempt, launchAdmittedRun, dispatch); `runDurable` 37 lines over `durableRun` (the live tail and the final drain now share one consumer configuration); `reconcileDiscovery` 25 lines over `discoveryScan`; `NewOrchestrator` extracted policy-catalog, config, spawn, sandbox-launcher, interactive-session, receipts, conversation-policy, supervision, model-health and usage-alert builders. One no-op removed: pre-reservation `failCreation()` calls, which could never fire because the idempotency key was not yet reserved, and the dead first `identities` list in discovery |
| P2-13 | REST/RPC twins: 55 RPCs without AM callers; CLI 103 REST vs 32 Connect call sites | One transport per operation after a fleet consumer search (P2-D11) | P2-W7 | closed except named exceptions, 2026-10-29. Finding: 38 of the 104 RPCs were `Unimplemented` stubs, not uncalled. Done: AM CLI and UI call the typed procedure for every operation whose twin was implemented; about 30 adapters implemented (declarations, workflow catalog and executions, role-policy, permission-policy ×7, probe, purge, GetRun, ReconcileScenarioProfiles); **30 REST routes + handlers deleted** (18 in step 1, 12 after the UI moved) after a fleet search found no caller outside AM; **22 stub RPCs deleted from `service.proto` (104 → 82)** after a reference search (record in `p2-transport-inventory.md`). Exceptions kept with reasons: REST routes with outside callers (profiles/tasks/runs/runners/execution-options/workflow-execution start-get-result-wait-trace-cancel-signals/investigations start+list/role-policy catalog), `runs/quiesce` + `runs/{id}/recover` (HTTP-bound run-initiated guards; REST only), CLI run events/report/list still REST (RPC implemented; inspection tests mock REST; ListRuns lacks `parent_run_id`), and the 6 RPCs with outside references (GetRun, ReconcileScenarioProfiles, GetProfile, ListProfiles, CreateTask, CreateRun) were implemented rather than deleted, so no RPC in the service answers `Unimplemented`; `CreateRun` runs the REST handler in process (its guards are HTTP-bound). Measures 2026-10-29: REST routes registered in `handlers.go` 119 → 89; endpoints.json 166 → 136 entries |
| P2-14 | UI parallel formatters (duration ×3, tokens ×3, provider ×5, "N/A" vs "—"), 3 private `apiRequest`, 9 raw fetches without deadlines, manual setInterval polling | `lib/format.ts`; one query client with deadlines | P2-W5b | closed 2026-10-29: `lib/format.ts`; `useApiQuery`/`fetchWithDeadline` (30 s) everywhere incl. features/usage and stats; no raw fetch without deadline |
| P2-15 | UI dead code (stats `calculations.ts`, `formatCompact`, chart helpers, `formatUsd`, `useModelAliases`, `useCreateAlias`, `useIsMobile`, unused barrel re-exports, CSS) | Deleted | P2-W5a / P2-W5b | closed 2026-10-29: W5b + W5a dead code deleted (≈60 stats files, FindingsPage, UsageRecommendations, formatters, calculations) |
| P2-16 | `RunDetail` 726-line function, tokens shown twice from two sources; ModelPricingTab duplicate dialog + sort header, 28px buttons; Dashboard `HealthItem` 174 lines | Split by tab; shared dialog/header; probe on Health only (P2-D13) | P2-W5b | closed (W5b) |
| P2-17 | Three card styles, local copies of RCL components on focus pages, raw tables | RCL Card/Tabs/Table on focus pages | P2-W5a / P2-W5b | closed 2026-10-29 on the focus pages: RCL Card/Tabs/Table/PageHeader; shared Dialog for the run mobile info dialog |
| P2-18 | 21 eslint errors in UI tests; 9 live-DB "tests" that never run under Test Genie | 0 errors; live checks moved to a program or marked manual | P2-W5b / P2-W6 | eslint 0 errors (W5b); live-DB checks (API): the six Go live-data tests carry `//go:build livecheck` plus their opt-in variables and are listed in `docs/internal/UNIT_TEST_ARCHITECTURE.md` (the seventh, `storagehealth` rehearsal, already had its own tag) — manual checks, never compiled by `go test ./...`; closed for the API |
| P2-19 | Stats panels duplicating burners (runner, tool, capability, cost distribution, run classes); time window held in a module global | Deleted (P2-D10); window in URL | P2-W5a | closed 2026-10-29: Details trimmed to tiles + model/profile tables; window and scope in the URL; module-global window deleted |
| D-15 | (pass 1) temporary transcript repair | Confirm the 109 unknown-model rows are not legacy, then delete repair + table | P2-W6 | closed. The 109 rows (108 Codex across 4 files, 1 Claude) carry real line times, not indexing times, and their transcripts contain no `turn_context`/model at all, so `unknown` is correct. Marker `usage-line-time-and-model-2026-10-29` is present. `transcriptindex/repair.go` (+test), the five repository methods, the interface assertion, the indexer call and the `transcript_usage_repairs` declaration are deleted; the table is dropped by the marker-gated retired-bookkeeping migration. `TransientStoreFailure` moved to `indexer.go`. Other fleet nodes never ran the repair; their pre-2026-10-29 rows keep `unknown` models and indexing-time stamps |

## Pass 2 after-measures (2026-10-29 17:35Z, orchestrator, after restart 12)

| Measure | Pass-2 baseline | After | Δ |
|---|---|---|---|
| API Go non-test / test LOC | 155,996 / 136,237 | 155,749 / 137,477 | −247 / +1,240 |
| CLI Go non-test / test LOC | 16,311 / 8,663 | 15,862 / 8,739 | −449 / +76 |
| UI non-test / test LOC | 29,272 / 12,726 | 24,925 / 10,898 | −4,347 / −1,828 |
| Total non-test LOC (API+CLI+UI) | 201,579 | 196,536 | **−5,043** |
| eslint errors in `ui/src` | 21 | 0 | −21 |
| `go vet` warnings (api, cli) | copylocks in 3 api + 3 cli tests | 0 | — |
| AgentManagerService RPCs | 104 (38 Unimplemented stubs) | 82 (0 stubs) | −22 |
| `.vrooli/endpoints.json` REST entries | 166 | 135 | −31 |
| Details 7-day tokens vs `transcript_usage` | ≈9–11× high | equal (30,893,007,112 = 30,893,007,112) | fixed |
| Fixture runs | 4,700 | 0 (deleted 2026-10-29 under the storage-manager deletion grant and the `run-deletion` authorization) | −4,700 |
| UI branch coverage | 84.7% (failing gate) | 85.9% | gate green |

API non-test LOC is roughly flat because pass 2 added capability (grouped plans, friction problems and
report/dismiss, four alert kinds with hysteresis, 25 Connect adapters replacing REST, deterministic event
IDs, the duplicate repair, run-once markers) while deleting `internal/stats`, the friction publisher and
finding review fields, 30 REST routes, 22 stub RPCs, one-shot migrations (`connection.go` 1,384 → 637),
the D-15 repair and dead functions; `createRun` 736 → 50 lines.

**How these LOC were counted.** Raw `wc -l` (blank and comment lines included) over non-test source only:
`find api cli -name '*.go' ! -name '*_test.go' ! -path '*/gen/*'` and `ui/src` `.ts`/`.tsx` without tests,
`test-utils` or `__tests__`. Docs, tests, requirements, SQL and generated code are excluded, and the
baseline is the pass-2 start, not the last commit. The staged diff in Git Control Tower (GetRepoStatus
`fileStats`, 2026-10-29) counts every file against commit `dc39df928a9` and reads +25,402 / −20,599 for
the scenario. By category:

| Category | Files | + | − | Net |
|---|---|---|---|---|
| API Go non-test | 117 | 6,186 | 6,351 | −165 |
| CLI Go non-test | 28 | 474 | 921 | −447 |
| UI TS non-test | 114 | 2,846 | 7,145 | −4,299 |
| Tests (Go and UI) | 159 | 4,578 | 5,011 | −433 |
| `p2-fixture-runs.txt` + `p2-fixture-dirs.txt` | 2 | 9,400 | 0 | +9,400 |
| Effort docs and mockup | 10 | 1,061 | 2 | +1,059 |
| Other (docs, requirements, `.vrooli`, CLI manifest, SQL, generated) | 46 | 857 | 1,169 | −312 |

The two fixture lists were deletion inputs, one line per item, not code. They and their two scripts
were removed after the deletion (2026-10-29). Once the scenario is restaged, its diff is about −4,600
lines net, docs and tests included.