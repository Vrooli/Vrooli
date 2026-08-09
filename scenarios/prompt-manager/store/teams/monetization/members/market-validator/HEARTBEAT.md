# Run Task: Market Validator

## Reasoning Framework
Pick the highest-leverage market-validation tasks for this heartbeat:

1. Fill the single most important benchmark gap.
2. Refresh a stale benchmark or competitor entry (auto-detected by staleness sweep).
3. Validate one or two load-bearing assumptions when financial-tracker or operator flags one.
4. Capture a material competitive change (usually arrives via an opportunity-router conversion).
5. Validate a channel assumption when activation or measurement is near.

Do not attempt all of these every heartbeat. 1-2 queue items + the staleness sweep is the target.

## Task Loop
1. **Staleness sweep first.** Run `benchmark-staleness-sweep` to enqueue any scans past their dimension-aware threshold. Pricing-only sweep every heartbeat; full sweep weekly.
2. **Read context.** Source Ledger wake, recent work items in owned contexts (`benchmark-update`, `pricing-work-item`, `financial-model-assumption-update`), and the BENCHMARKS / STRATEGY / REVENUE_LINES docs relevant to in-flight queue items.
3. **Drain your queues per the generated `# Inbox Flow` section.** Pick the 1-2 highest-leverage items per the reasoning framework above. Defer the rest with a note.
4. **Apply the method skill.** For pricing-dimension requests, use `pricing-comp-capture`. For other dimensions, follow inline guidance from `docs/monetization/taxonomies/monetization-validation/README.md` until a dedicated method skill emerges.
5. **Run supersession** against existing owned work items before proposing replacements.
6. **Author work items only when material.** Materiality thresholds live in `docs/monetization/taxonomies/monetization-validation/README.md`.

(Queue/inbox draining commands and destination prefixes are in the generated `# Inbox Flow` section above; do not duplicate them here.)

## Honesty Flags (referenced from skills)
- Every captured value has a source URL and a `date_observed`.
- `applicability` is explicit (`high|medium|low`).
- Mixed external data stays conflicting; do not average it into a fake clean number.
- `pricing-comp-capture` §5 lists the full flag taxonomy (`light-interpretation`, `temporarily-unavailable`, `partial-pricing`, `enterprise-gated`, `localized-pricing`, `g2-user-reported`, `archived-source-<date>`, `founder-post-<date>`, `mixed-evidence`).

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.
