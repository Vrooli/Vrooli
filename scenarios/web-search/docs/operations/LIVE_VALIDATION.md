# Live Validation Runbook

Live agent behavior requires Agent Manager, SearXNG, Ollama, and reachable sources. The script collects execution-specific artifacts; an answered result is not itself proof of factual correctness.

```bash
make -C scenarios/web-search validate-live
# After an incomplete wait, attach to the same execution:
LIVE_VALIDATE_RUN_ID=<id> LIVE_VALIDATE_IDEMPOTENCY_KEY=<original-key> scenarios/web-search/scripts/live-validate.sh
```

`LIVE_VALIDATE_WAIT_SECONDS` defaults to 60 (maximum 90). The script performs one owner wait and exits with status 2 on timeout or terminal failure. It preserves the execution ID and partial output. It never starts a replacement when an execution ID is supplied. Workflow budgets and Agent Manager own execution lifetimes.

Artifacts under `coverage/manual-validations/artifacts/<key>` include start/result payloads, workflow trace, exact node attempts, findings named by the result, and 20 warm latency samples. The nearest-rank p95 is sample 19 after sorting. No API-global log count or time-window finding list is used to attribute traffic to this run.

For REQ-P1-002, inspect the run IDs from the execution attempts and their tool evidence: verify research actions and an actual gap that caused further research. Two unrelated tool calls do not establish this. For REQ-P1-004, inspect the exact returned finding IDs and corresponding creation events, sources, and citations; a result can name a pre-existing finding, so IDs alone do not prove auto-capture. For REQ-P0-004, require p95 <=500ms plus independent zero-external-call evidence from the hermetic routing tests.

After review, use `vrooli scenario requirements manual-log` for only the criteria supported by these artifacts, under the evidence owner's expiry policy. The script never automatically marks requirements passed. Missing evidence remains pending; abstention and partial answers remain distinguishable from answered research.

## Research maturity acceptance

For OT-P0-009, compare an explicitly current question with an exact stored-answer reuse request. Verify the complete answer, claim-specific citations, retrieval dates, abstention reason, and live-call count. Repeat with stale and disputed findings and an unreachable upstream.

For OT-P0-010, start one bounded L3 request and attach once to the owner wait operation. Retain the run identity across a wait timeout; verify the final structured evidence and distinguish research quality from agent completion.

For OT-P0-011, record successful, failed, and unavailable attempts through Vrooli Memory. Compare fixed windows with the same operation and context. Empty cohorts and absent quality evidence must remain unknown.

For OT-P0-012, run the versioned comparison method against recorded independent sources, changed facts, contradictions, and insufficient evidence. Preserve prior revisions and evidence when changing a method; require citation support and freshness before comparing effort.

For the historical REQ-P2-002 warm routing budget, warm Search Hub and its configured routing model, then measure representative natural-language external-intent queries end to end. Retain timing samples and routing provenance. A p95 above two seconds fails the historical budget; cold-start and provider-unavailable observations are separate. No current evidence is claimed by this protocol.

The 2026-09-05 bounded PostgreSQL smoke produced a valid structured child result but exhausted the original 40,000 aggregate-token workflow budget (146,328 reported tokens). Revision 1.0.2 scopes the task to this scenario and allocates 250,000 aggregate tokens, retaining the 600-second, 30-turn, one-child, one-attempt limits. This is an observed execution-envelope correction, not a quality improvement; the failed execution remains in the evidence.

Revision 1.0.3 explicitly records new findings with `--source l3`. Live execution artifacts `web-search-maturity-smoke-v103-20260905` show a completed answered result and finding `7a229205-c2a7-4475-b3b3-3f90508b8004` with `FINDING_SOURCE_L3`. Revision 1.0.2 had inherited the findings CLI L2 default; that observation is retained rather than relabelled.

## Storage migration evidence

On 2026-09-05, backup run `48681764-fdcf-4091-8343-0d1c0c756789` completed successfully with snapshot `ffca4a72dc8313117b5fcda50ca749ac`. With web-search stopped, a one-off script copied 11 findings and the engine metadata point from the legacy vector collection to the canonical live namespace. Verification preserved every finding ID, payload and vector. The metadata ID was then re-derived from the canonical collection name, as required by the engine; its payload and vector were preserved. A live findings search subsequently returned `method: dense`, confirming semantic retrieval after the migration. The old collection remains available for rollback; committed code contains no legacy-name fallback. The namespace regression test verifies separate live and shadow collections.

## Acceptance result — 2026-09-05

Comprehensive run `20260905-072147-42f80abd` passed all 27 phases, with none skipped, in 279 seconds. The passing unit phase includes 200 UI tests across 39 files and the original 85% coverage thresholds. Browser workflow acceptance includes a schema-complete leased findings database; its regression test also verifies no write reaches the primary database.

The evidence owner reports 22/22 requirements complete after syncing qualified test references. Manual attestations for REQ-P0-009 through REQ-P0-012 describe the exact live and controlled observations; artifacts are retained under `coverage/manual-validations/artifacts/web-search-maturity-acceptance-20260905`. The final suite result is preserved there too. Earlier failures remain available.

This is acceptance of the implemented behavior, not a claim that every maturity dimension is at its ceiling. Security reports 43 advisory warnings; the pinned host Go toolchain is filed to the control-plane owner as `knw-1788591604850346261`. Branding and other advisory improvements remain in the durable findings report. The bounded prune-count conversion is checked against `math.MaxInt32` despite a remaining scanner warning. Comparable operator learning baselines remain unknown; cache/governor readings are available as bounded fixed-window owner measures and return unknown when their observation history is empty or truncated.
