# Legacy → declarative-operations mapping

> Historical migration record. The runtime described below is retired; this
> mapping remains only to explain legacy records. Current behavior is defined
> by [Target Operating Model](../../concepts/TARGET-OPERATING-MODEL.md).

The authoritative map from **every** legacy identity found in the Phase-1
inventory to its destination in the declarative agent-operations model
(EXECUTION-MODES.md D1–D8, [AGENT-OPERATIONS.md](../../concepts/AGENT-OPERATIONS.md)).
Phases 8 (state migration) and 9 (legacy removal) execute against this document;
it is the reconciliation contract. Sources: the Phase-1
[inventory](inventory/inventory-phase1-summary.md) (22 object classes, 7
expected-absent entries, 6 referential findings), the
[agent cutover ledger](../../internal/AGENT-CUTOVER-LEDGER.md) (14 target-bound
behaviors), and the [safety runbook](RUNBOOK.md) storage layout.

> **This is a mapping, not a move plan.** Most artifacts stay exactly where they
> are on disk; the mapping records their *logical* destination in the new model
> (which workflow instance / operation execution correlates them). Where a Phase-8
> staged rewrite is required (folder-path renames, key canonicalization), the row
> says so.

## 1. Target kinds

| Legacy target kind | Destination | Migration action |
|---|---|---|
| `plan-manager-plan` | `plan-execution` (renamed; provider-neutral) | Rename in all vocabulary + rename on-disk folder segment `mode-targets/plan-manager-plan/` → `mode-targets/plan-execution/` (Phase 8 staged rewrite). Proto wire number unchanged (1). |
| `plan-ref` (unmanaged workspace-file plan target) | **Removed** — no successor target kind | 0 live instances in inventory (`plan-refs.unmanaged = 0`), so nothing to migrate. Any stray `mode-targets/plan-ref/**` round dir (none observed) is quarantined, not migrated. Proto enum number 2 is `reserved`. |
| `initiative` | `initiative` (unchanged) | None. |
| — (new) | `backlog-item` | New pinned vocabulary; no legacy instances (item-level ran under the `initiative` substrate). Adapter is a later phase. |

**Domain `plan_ref` field (NOT a target kind).** The 144 managed plan_refs
(`plan-refs.managed = 144`, `unmanaged = 0`) on backlog items and initiatives are
an associated Plan Manager reference. They are **unchanged** and stay as domain
fields; the `plan_ref_sweep_manifest` artifact below is a separate historical
record, not a target.

## 2. Operation contracts (the 14 target-bound agent spawns)

Every `(a)` call site in the cutover ledger maps to one named operation
contract (`api/internal/agentops` vocabulary).

| Ledger call site | Behavior | Operation contract |
|---|---|---|
| `backlog/research.go:567` | Autonomous research pass over an item | `research-refine` |
| `backlog/workshop_save.go:421` | Workshop synthesis spawn | `workshop-round` |
| `backlog/clarification.go:202` | Start a clarification thread | `clarification-start` |
| `backlog/clarification.go:353` | Continue the clarification run | `clarification-continue` |
| `backlog/clarification_service.go:166` | Re-enter workshop after clarification | `workshop-round` (clarification-triggered) |
| `execution/service_queue.go:281` | Queue/start the primary execution run | `execution-run` |
| `execution/service_control.go:117` | Start a queued execution run | `execution-run` |
| `execution/retry.go:147` | Retry a failed execution (new attempt) | `execution-retry` |
| `execution/followup.go:125` | Fixup run after review found issues | `execution-fixup` |
| `execution/followup.go:330` | Follow-up run after completion | `execution-followup` |
| `execution/followup.go:306` | Continue parent as follow-up | `execution-followup` (continuation) |
| `review/service.go:234` | Autonomous review of a completed item | `review-round` |
| `review/rounds.go:197` | Gather more evidence for a review round | `evidence-request` |
| `initiativereview/trigger.go:191` | Autonomous initiative-level review | `initiative-review` |

The `workshop-finalize` and `revision` contracts are also pinned in the
vocabulary (workshop finalization / targeted spec revision) even though the
ledger folds them into workshop/clarification call sites; they get their own
contracts so the finalize-and-bind-plan and request-revision behaviors are
first-class in Phases 4–5. `(b)` interactive-session and `(d)` capture-classify
call sites are **not** operations (see §7).

## 3. Modes and member-item coordination

| Legacy identity | Destination | Migration action |
|---|---|---|
| `item-level` operating mode (phase-less) | **Member-item workflow strategy** (`member-item-strategy.schema.json`, default `parallel-items` + `execution-run`) on the initiative workflow instance | Not an operating mode. Phase 8 rewrites each initiative's `item-level`/default mode selection into an initiative workflow-instance `strategy`; deletion of the `item-level` mode folder + alias is Phase 9. |
| Blank / sentinel mode value (no explicit mode) | Same member-item strategy default | Treated identically to `item-level`; canonicalized to an explicit strategy in Phase 8. |
| `holistic-loop` operating mode | Unchanged operating mode (`initiative` target) | None — a genuine phase-graph loop. |
| `phased-plan-drain` operating mode | Unchanged operating mode (`plan-execution` target) | None (already renamed via the target vocabulary change). |

## 4. Object classes (every inventoried class)

All 22 inventoried object classes. "Correlation" = linked to a workflow
instance / operation execution as history/evidence; the bytes stay in place
unless a Phase-8 staged rewrite is noted.

| Object class | Count | Destination | Migration action |
|---|---:|---|---|
| `backlog_item` (primary) | 600 | Backlog-item domain truth (unchanged) + a `backlog-item` workflow instance correlating its operations | Stays authoritative for status/plan_ref. Phase 8 opens a workflow instance per item with active operations. |
| `initiative` (primary) | 68 | Initiative domain truth (unchanged) + an `initiative` workflow instance carrying the member-item strategy | Stays authoritative for membership/criteria/plan_ref. Phase 8 opens a workflow instance + strategy. |
| `goal` (primary) | 5 | Unchanged domain truth | None (goals are not operation targets). |
| `record` (primary) | 1611 | Unchanged (learning-loop records) | None. |
| `capture` (primary) | 9 | Out of scope (ingest/triage) | Excluded — see §7 (ledger `(d)` scope decision). |
| `workshop_round` (artifact) | 283 | Correlated to `workshop-round` operation executions on the item workflow | Stays on disk; linked as operation output/evidence. |
| `workshop_clarification` (artifact) | 3 | Correlated to `clarification-start`/`clarification-continue` executions | Stays; linked. |
| `backlog_clarify_artifact` (artifact) | 1 | Correlated to clarification operation execution | Stays; linked. |
| `backlog_review_artifact` (artifact) | 185 | Correlated to `review-round` operation executions | Stays; linked. |
| `backlog_evidence_artifact` (artifact) | 9 | Correlated to `evidence-request` executions + evidence expectations | Stays; linked. |
| `acceptance_validation` (artifact) | 56 | Correlated to review/execution outcomes as evidence | Stays; linked. |
| `item_doc` (artifact) | 225 | Spec/plan document behind `provides-spec-document` | Stays; referenced by operations, not moved. |
| `item_swarm_artifact` (artifact) | 136 | Operation-execution working artifacts | Stays; linked to executions. |
| `initiative_context_file` (artifact) | 84 | Initiative-adapter context (unchanged) | None. |
| `initiative_graph` (artifact) | 68 | Initiative domain artifact (unchanged) | None. |
| `initiative_review_round` (artifact) | 1 | Correlated to `initiative-review` operation execution | Stays; linked. |
| `om_round` (artifact) | 3 | Operating-mode round → operation-execution history on the workflow instance | Stays; provenance digests referenced from workflow `operations[]`. Folder segment rename if under `mode-targets/plan-manager-plan/` (§6). |
| `agent_activities` (state) | 1 | Activity chokepoint (unchanged; `(c)` boundary) | None — remains the tracking chokepoint operations route through. |
| `settings_config` (artifact) | 1 | Binding defaults + transition-policy inputs | Phase 8 reads legacy settings that selected modes/auto-drain into `system-default` bindings + policy revisions. |
| `plan_ref_sweep_manifest` (artifact) | 1 | Legacy sweep record — retained as-is (archival) | Not a target and not correlated; kept or archived, never remapped. |
| `eventlog_sqlite` (opaque) | 3 | Opaque event DB (unchanged) | WAL-checkpointed backup only; never hand-edited or remapped (RUNBOOK §0). |
| `foreign_deployment_report` (foreign) | 1 | **Excluded** (foreign artifact) | scenario-dependency-analyzer output under swarm-manager's data root; excluded from migration + reconciliation (`ambiguous_ownership`). |

## 5. Expected-but-absent state (all 7)

| Expected-absent file | Destination | Migration action |
|---|---|---|
| `state/execution-runs.json` | Workflow operation history (`operations[]` on the item/initiative workflow instance) | When present, each entry becomes a correlated operation-execution record with a provenance digest; absent means no runs to correlate. |
| `state/queue.json` | Execution scheduling state under the member-item strategy | Rehydrated into the workflow scheduler; absent = empty queue. |
| `state/circuit-breaker.json` | Per-item failure guard on the item workflow (blocked-state + policy) | Modeled as a `blocked` workflow state + `escalate-needs-attention` transition; absent = no trips. |
| `state/engagement-owners.json` | Ownership/exclusivity → workflow + provenance ownership | Carried into the provenance `binding.owner` + workflow ownership; absent = no open engagements. |
| `data/operating-mode-run-owners/run-owners.json` | Run-owner index → workflow `operations[]` + provenance ownership | Global index folded into per-instance correlation; absent = no indexed mode-round runs. |
| `data/auto-drain.json` | Auto-drain flag → binding/policy input | Becomes a `system-default` binding + policy toggle; absent = disabled (default). |
| `data/autofiler/dismissed_findings.json` | Auto-filer dismissals (unrelated to operations) | Unchanged domain state; absent = none dismissed. |

## 6. Storage hazards (Phase-1 findings requiring explicit handling)

| Hazard | Resolution |
|---|---|
| **Unsanitized `scope_id` vs sanitized on-disk directory names** | The document's `scope_id` may differ from its sanitized folder name (`sanitizeOwnershipToken`). Phase 8 keys reconciliation on the in-document `scope_id`, deriving the folder via the same sanitizer, so the id—not the folder name—is authoritative. |
| **Three inconsistent `(kind,name)` key spellings under `execution/`** | Canonicalize to one spelling on migration (the sanitized `<kind>/<name>` form). Phase 8 records the canonical mapping; all three legacy spellings resolve to the single canonical workflow-instance/operation key. |
| **Initiative-target rounds under `initiatives/<name>/modes/`** | This is a *different* root than `mode-targets/`. Both hold mode rounds and both must be reconciled: initiative-target rounds stay under the initiative; plan-target rounds under `mode-targets/plan-execution/`. Phase 8 walks both roots. |
| **`mode-targets/plan-manager-plan/` folder segment** | Renamed to `mode-targets/plan-execution/` (staged rewrite; the target-kind string drives the path). |
| **`data/deployment/deployment-report.json` (foreign)** | Excluded from migration + reconciliation (`ambiguous_ownership`); written by scenario-dependency-analyzer. |
| **`events.db` (+ `-wal`/`-shm`) live SQLite** | Opaque; backed up with WAL checkpointed; never hand-edited or remapped. |

## 7. Referential findings (all 6) and out-of-scope

| Finding | Resolution |
|---|---|
| `ambiguous_ownership`: `data/deployment/deployment-report.json` → foreign | **Exclude** from migration + reconciliation. |
| `dangling_dependency`: `fix/dtv-cli-validate-and-report` → `fix/dtv-validation-api` (missing) | **Quarantine** the dangling edge: Phase 8 drops the unresolvable `depends_on` into a quarantine report rather than migrating a broken reference. |
| `dangling_dependency`: `fix/dtv-report-generation` → `fix/dtv-validation-api` (missing) | Same — quarantine the dangling edge. |
| `dangling_initiative_item`: `initiative/dtv-meta-optimization-readiness` → `fix/dtv-validation-api` (missing) | **Quarantine** the membership entry: the initiative workflow instance records the member set minus the missing item, with the dropped ref in the quarantine report. |
| `dangling_initiative_item`: `initiative/swarm-manager-graph-workspace` → `chore/swarm-manager-remove-legacy-tabbed-surfaces` (missing) | Same — quarantine the membership entry. |
| `dangling_initiative_item`: `initiative/swarm-manager-graph-workspace` → `execute/swarm-manager-mobile-graph-interaction` (missing) | Same — quarantine the membership entry. |

**Out of scope (recorded scope decisions):**
- `capture` primary objects + `captures/classify.go:146` (`(d)`): capture
  classification is an ingest/triage step, not a target-bound operation. It
  stays a direct spawn; re-examined in Phase 6 closeout.
- `(b)` interactive Agent Session continuations (`agentsessions/*`,
  `handler.go:242`): a session boundary, not an operation. Unchanged.

## 8. Coverage assertion

- **Object classes:** 22 / 22 mapped (§4) — 20 with a destination in the new
  model, 2 explicitly excluded (`capture` = ledger `(d)`;
  `foreign_deployment_report` = foreign/ambiguous). `plan_ref_sweep_manifest`
  and `eventlog_sqlite` are retained-as-is (archival/opaque), which is a
  destination.
- **Expected-absent state:** 7 / 7 mapped (§5).
- **Referential findings:** 6 / 6 resolved (§7) — 1 excluded, 5 quarantined.
- **Target kinds:** 4 / 4 (rename, remove, unchanged, new) (§1).
- **Ledger `(a)` behaviors:** 14 / 14 mapped to operation contracts (§2);
  `(b)`×3 and `(d)`×1 explicitly out of scope (§7).

Every inventoried legacy identity therefore has a precise destination: a new-model
correlation, an unchanged domain field, a renamed path, an archival retention, an
explicit exclusion, or a quarantine.

## 9. Settings → policy controls (Phase-7 slice B equivalence table)

Refines the `settings_config` row in §4. Phase 7 slice B introduced the typed
`agentops.PolicyControls` projection (`api/internal/agentops/policy_controls.go`)
derived from persisted settings by `settings.ProjectPolicyControls`
(`api/internal/settings/policy_controls.go`); all orchestration consumers now
read through that seam (or the execution provider seams derived from it), never
raw settings fields. **No persisted field was renamed, dropped, or rewritten** —
`config/settings.json` loads exactly as before. Phase 8 uses this table when
migrating persisted settings into `system-default` bindings and
transition-policy revisions; the same table is served publicly as
`SettingsResponse.policy_projection` (proto `SettingsPolicyProjection`).

Retained user preferences (per the plan): auto-advance + delay, retry/fixup
limits, review thresholds. They remain user-facing settings; only the *read
path* moved behind the policy-controls seam.

| Legacy settings field (JSON) | Role | PolicyControls destination | Consumer(s) after slice B | Phase-8 action |
|---|---|---|---|---|
| `default_mode` | policy control | `execution.default_mode` | `execution.PolicyProvider` (queue-mode default) | Fold into system-default binding's execution posture. |
| `auto_fixup` | policy control | `retry.auto_fixup` | `execution` finalization (fixup spawn decision) | Transition-policy control (fail → fixup transition guard). |
| `max_fixup_attempts` | policy control (retained pref) | `retry.max_fixup_attempts` | `execution` finalization | Transition-policy retry-limit parameter. |
| `review_agent_enabled` | policy control | `review.agent_enabled` | `execution` finalization (`checkReviewAgentEnabled`, review trigger) | Transition-policy control gating `review-round`/evidence spawning. |
| `max_auto_rounds` | policy control | `auto_advance.max_auto_rounds` | `backlog` workshop auto-advance via `Handler.loadPolicyControls` | Transition-policy round cap. |
| `auto_initialize_workshop` | policy control | `auto_advance.auto_initialize` | `backlog` create path (`maybeAutoWorkshop`) | Transition-policy control (created → workshop transition). |
| `auto_advance_workshop` | policy control (retained pref) | `auto_advance.enabled` | `backlog` workshop save (`computeAutoAdvance`) | Transition-policy auto-advance toggle. |
| `auto_cascade_workshop` | policy control | `auto_advance.cascade` | `backlog` update path (`maybeCascadeWorkshop`) | Transition-policy control (dependency-unblock cascade). |
| `auto_advance_delay_seconds` | policy control (retained pref) | `auto_advance.delay_seconds` | `backlog` scheduler-intent creation (`scheduleDeferredAdvance` not-before) | Scheduler-intent delay parameter on the advance transition. |
| `agent_max_turns` | policy control | `budgets.max_turns` | `execution` governance cost estimation (via `GovernanceProvider`, derived from projection) | Binding-default agent budget. |
| `agent_timeout_seconds` | **dormant** (no runtime reader; spawn timeouts come from the agent profile) | `budgets.timeout_seconds` | none (CLI/UI display only) | Migrate alongside `max_turns` into the binding-default budget or retire with an explicit decision; field stays persisted until Phase 9. |
| `review_code_quality_min_score` | policy control (retained pref) | `review.code_quality_min_score` | `execution.ReviewThresholdsProvider` → GCT review request | Review-transition threshold parameter. |
| `review_test_min_pass_rate` | policy control (retained pref) | `review.test_min_pass_rate` | same | same |
| `review_max_blocking_violations` | policy control (retained pref) | `review.max_blocking_violations` | same | same |
| `review_max_warnings` | policy control (retained pref) | `review.max_warnings` | same | same |
| `review_require_screenshots` | policy control (retained pref) | `review.require_screenshots` | same | same |
| `review_require_tests` | policy control (retained pref) | `review.require_tests` | same | same |

**Not policy controls (stay in settings unchanged):** `theme`,
`search_debounce_ms`, `toast_duration_ms`, `delete_confirmation_levels` (pure
UI preferences); `lane_concurrency_limits`, `max_queue_depth`,
`circuit_breaker_*`, `execution_cost_cap_per_run`, `cost_per_turn_estimate`,
`fix_before_feature`, `auto_filer.*` (system governance — consumed through the
existing governance/lane seams, not transition policies).

**Equivalence guarantee for Phase 8:** `agentops.DefaultPolicyControls()` ==
`ProjectPolicyControls(settings.DefaultSettings())` (asserted by
`TestDefaultPolicyControlsEqualsDefaultSettingsProjection`), and the legacy
`execution.Policy` / `execution.ReviewThresholds` adapters are derived from the
projection (asserted by `TestLegacyAdaptersDerivedFromProjection`), so a
migration that reproduces the projected values in bindings/policies reproduces
today's behavior exactly.
