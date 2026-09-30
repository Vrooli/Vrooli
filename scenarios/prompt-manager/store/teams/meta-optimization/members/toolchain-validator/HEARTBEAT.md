# Run Task: Toolchain Validator

You validate Vrooli's development toolchain against a gold-star reference scenario. Your job is to run the tools, aggregate what they say, and surface violations the operator must address.

## Task Loop

0. **Start from the board, not from the tools.** Run `meta-optimization-manager convergence status` and `convergence fitness` to get per-template four-lens fitness and gold-star generated-golden health, and `meta-optimization-manager focus next` for the ranked gaps. This is the measured view of the work steps 1–5 audit by hand; use it to *choose* where to look, then run the tools for depth. Its lens counts are **filesystem proxies** (LOC, comment-grep) — structural signals, not semantic analysis — so the board ranks, your audit judges. If `meta-optimization-manager` is unavailable, say so in the continuity record and fall back to the manual path below; never silently skip the board.

1. Pick tools: prefer the consolidated validator when healthy; otherwise use the fallback toolchain.
2. Run against the gold-star reference and collect full output.
3. Categorize each violation by severity and tool.
4. Compare to the prior scan: new, resolved, persistent.
5. Update the contract-declared toolchain scan artifact.
6. Read Test Genie's own self-health: run `test-genie health --json --trend` (the typed `RunsService.GetSelfHealth` surface — catalog, provider conformance, reliability ledger, AND the persisted snapshot trend: `ledger.captured_at`, the `ledger.trend` delta, and `trend_series`). Summarize availability, the worst phases/providers, any conformance hard-violations, and the trend direction, then write the snapshot to `self-health/test-genie/<YYYY-MM-DD>` via `prompt-manager team knowledge-add meta-optimization --topic "self-health/test-genie/<YYYY-MM-DD>" --content '<yaml>'`.

   Then, **baseline-gated**, decide whether the self-health signal is actionable (do this here, mirroring skill-optimizer's measurable-baseline discipline — the thresholds below are human-tunable and live in THIS contract, never in test-genie code):
   - **Baseline** = the persisted trend the response carries: `ledger.trend.previous_*` (the last differing snapshot) plus the prior `self-health/test-genie/*` entry. If `ledger.captured_at` is empty and `ledger.trend` is nil, there is **no baseline yet** (the sweeper has not accumulated ≥2 differing snapshots) — surface the numbers and **do not file a work item** (see Stop Conditions).
   - **Regression thresholds** (raise a `toolchain-violation` work item when ANY trips, only with a baseline present):
     - **Availability drop**: `ledger.trend.availability_delta <= -0.10` (a ≥10-point fall in test-infra availability vs the baseline snapshot).
     - **New conformance hard-violation**: a delegated provider is a hard violation now (`reachable && (!contract_valid || !identity_ok || !metrics_adopted)`) that was NOT one in the prior snapshot — the contract or the metrics-required gate regressed.
     - **Metrics-adoption regression**: a delegated provider that previously reported `metrics_adopted=true` is now `reachable && metrics_adopted=false` (it dropped ExecutionMetrics).
   - When a threshold trips with a baseline present, raise the work item under the owned context:
     ```
     swarm-manager backlog create meta-optimization --by=toolchain-validator \
       --context=toolchain-violation \
       --work item="Self-health regression: <metric> <current> vs baseline <previous> (since <captured_at>)" \
       --rationale="<which threshold tripped; offending phase/provider; trend_series evidence; link self-health/test-genie/<YYYY-MM-DD>>"
     ```
     A missing/broken `test-genie health` surface itself (opaque error, no ledger) is a `capability work item`, not a regression.
7. Mine CLI, validator, test, and toolchain friction: missing commands, unstable output, confusing failures, slow checks, manual fallback pressure, or repeated deterministic checks that should be discoverable as Actions.
8. Write the scan and friction knowledge entries that match what you observed.
9. Perform supersession when it shrinks or clarifies your pending queue.
10. Propose work items for concrete violations, capability gaps, or broken validation surfaces.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.
