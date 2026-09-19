# Testing — Search Hub

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

Existing local entry points (choose the domain test for the behavior you change):

- [api/handlers/health/handler_test.go](../../api/handlers/health/handler_test.go)
- [ui/src/App.test.tsx](../../ui/src/App.test.tsx)
- [ui/src/features/health/HealthCard.test.tsx](../../ui/src/features/health/HealthCard.test.tsx)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## Coverage thresholds

### Retrieval trust seams

The composed router suite is generated from the registered provider-owned
corpora, so routing cases grow and shrink with registry state. Corpus-shape
validation is tested with descriptor-only fixtures, and federated evals retain
`routing_precision` separately from `retrieval_recall`. Lifecycle/evidence
tests assert that experimental or unproven providers remain explicitly
selectable but are excluded from automatic routing.

The persistent description index has tests for changed-descriptor
incrementality, restart reuse, embedding metadata invalidation, deterministic
fallback ordering, and isolated failed-entry drops. Long-running validation is
owned by Test Genie; start a run with `vrooli scenario test <name>` and block
once with `test-genie runs wait --json <name> <run-id>`.

For retrieval-correctness work, preserve the measurement boundary: collect the
declared baseline before editing, keep the reranker and Qdrant healthy, and do
not run a comparison concurrently with catalog changes. Use the guarded
comparison surface and retain its run IDs:

```bash
curl --fail-with-body --silent --show-error --max-time 1800 \
  -H 'Content-Type: application/json' -H 'Connect-Protocol-Version: 1' \
  --data '{"suiteId":"router.routing","strategyNames":["semantic-cross-encoder","lexical-cross-encoder","lexical-fallback"],"apply":false,"limit":10}' \
  http://localhost:19157/vrooli.search_hub.v1.eval.EvalService/CompareStrategies
```

Promotion is valid only when the returned arm holds the held-out gate and the
paired significance guard passes. A failed or incomparable comparison is
evidence to record, not a reason to edit the active strategy by hand.

The operational eviction drill is also part of the validation contract:

1. Stop the reranker with `vrooli resource stop reranker` and call
   `meta-optimization-manager focus next --json`; the condition surface must
   name `condition/substrate/reranker` and identify the reranker leg.
2. Restore it with `vrooli resource start reranker`, wait for the managed
   resource to report healthy, then call focus again; the substrate item must
   disappear and a healthy Search Hub must not set `degraded`.
3. Keep the resource running for the final Search Hub suite and record its
   active reranker leg in the evidence.

## Common patterns and anti-patterns

### Route-discriminative comparison evidence (2026-08-16)

After changing routing metadata or semantic representations, rebuild and
restart Search Hub through its lifecycle target before running a comparison;
an already-running API process can otherwise produce valid-looking runs from
the old binary. The authoritative follow-up command was:

```bash
search-hub strategy compare \
  --strategies semantic-cross-encoder,lexical-cross-encoder,lexical-fallback \
  --apply --json
```

Inspect the persisted `routing_trace` on the returned `router.routing` runs.
The trace is diagnostic only: it must not alter the evaluator's expected IDs
or grading labels. For each gradeable case, compare dense top-1/3/6 recall,
evidence-union membership, selected-provider membership, returned-evidence
state, and the final expected-hit rank. A dense miss rescued by lexical
evidence is still a dense miss; a selected owner whose expected item is absent
or outside declared top-K is a provider-retrieval failure. These are different
remediation owners and must remain separate in reports.

The 2026-08-16 comparison persisted complete traces for all 213 cases in each
arm. The semantic arm retained the expected owner in 173/179 evidence unions
and 160/179 selected sets, but its end-to-end retrieval recall was only
0.00625 versus 0.2625 for the lexical incumbent. Promotion was refused by the
paired-significance guard (delta mean 0, CI lower 0) despite a held-out routing
fold hold. See `internal/PERFORMANCE.md` for the immutable run IDs and stage
counts.

| ✅ DO | ❌ DON'T |
|---|---|
| `mocks.FakeClock` for time-dependent assertions | `time.Sleep(150 * time.Millisecond)` then assert on a fuzzy match |
| `httpx.NewLiveServer` for handler tests | `httptest.NewRecorder` (hides SSE-flusher bugs) |
| `getByTestId(selectors.x.y)` for stable selectors | `getByText("Save")` (breaks the moment copy changes) |
| `vi.mock("./api/health", async (importOriginal) => …)` inline at top of file | Helper-wrapped `vi.mock` (TDZ at hoist time) |
| `makeHealthResponse({ status: "degraded" })` for variants | Hardcoded literal payload in three different tests |
| Per-method error knob (`PingErr error`) on fakes | Single global "fail mode" boolean across the fake |
| `var _ Pinger = (*sql.DB)(nil)` to lock the contract at compile time | Runtime "does this satisfy" check in init |
| Adding new seams to [`SEAMS.md`](SEAMS.md) at the same commit | "We'll document it later" |
