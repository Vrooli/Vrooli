# Problems — Tech Tree Designer

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-06-14 - AI strategic analysis deferred

**Symptom:** The old implementation included AI-flavored strategic analysis concepts, but the regenerated scenario has no `ollama` or `openrouter` dependency.

**Root cause:** The deterministic graph and contract-first planning surface must ship before analysis layers are useful.

**Workaround:** Keep AI analysis out of service dependencies and UI/API scope.

**Real fix:** Implement follow-up `tech-tree-designer-ai-strategic-analysis` after graph, planning, ontology, and UI are stable.

**Owner:** unassigned.

**Refs:** `PRD.md` OT-P2-002.

### 2026-06-14 - SDA GraphSource deferred

**Status:** Resolved on 2026-06-15. TTD now consumes `scenario-dependency-analyzer` `DescribeInterfaceGraph` through `SDASource`.

**Root cause:** Earlier graph work landed before SDA's Connect graph contract existed.

**Outcome:** The old proto-health source was deleted; SDA is now the canonical live graph source for proto and Go import evidence.

**Owner:** unassigned.

**Refs:** `PRD.md` OT-P2-001.

### 2026-06-14 - Scenario scaffold generation from plans deferred

**Symptom:** Planned scenarios can be materialized as proto schemas, but this phase will not scaffold a full scenario from a planned node.

**Root cause:** The generator integration is a separate governance and lifecycle concern from proto materialization.

**Workaround:** `plan materialize` writes validated proto text only; agents run `vrooli scenario generate` separately when implementation starts.

**Real fix:** Design and implement the TTD-to-generator seam after planned proto validation is proven.

**Owner:** unassigned.

**Refs:** `PRD.md` OT-P0-002.

### 2026-06-14 - Premature experimental proto import guard belongs in proto-health

**Symptom:** After a planned proto is materialized, other scenarios could import an experimental cross-scenario proto before it is ready.

**Root cause:** Import policy enforcement belongs in proto-health or shared proto governance, not in TTD's planning UI.

**Workaround:** Keep materialization explicit and validation-gated.

**Real fix:** Add a proto-health guard that detects or rejects premature imports of another scenario's experimental cross-scenario proto.

**Owner:** unassigned.

**Refs:** `PRD.md` OT-P0-002.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## 2026-09-09 — expanded target and qualification debt

The current ProtoFile planner/materializer does not implement general artifact bundles. New requirement modules and experience specs describe intended behavior only. Required gaps include durable immutable revisions, cross-repository safe scope, owner/grant adapters, partial-apply recovery, publication isolation and bounded scale measurements.

Business Health validation reports TTD-GRAPH-001 complete without a logged manual attestation and TTD-ONTOLOGY-001/TTD-SDA-001 complete without a requirements-sync snapshot. These historical records were preserved during the documentation expansion. Reconcile through the evidence owner; do not infer a fresh passing suite or manufacture attestations.

Numeric performance cohorts/thresholds, retention windows and backup recovery objectives remain qualification decisions. Remote/multi-user support and draft experiments need separate identity and effect-isolation proof. Scenario-specific improvement skills and governed sensors must be discovered and completed through the shared setup method; prose alone does not establish they exist.

Installed experience-manager validation passed, but later CLI auto-rebuild failed on a missing protovalidate go.sum entry and fell back to a stale binary. structure-health likewise reported a CLI rebuild dependency issue. These are tooling qualification limitations, not authorization to edit dependencies in this documentation-only task.

## Work ladder

- Rung: W2 evidence and scoped W3 development setup; governing expansion remains unchanged.
- Measured: 2026-09-09, subsequent implementation task (not the documentation-only review above).
- Reconciled: TTD-GRAPH-001, TTD-ONTOLOGY-001 and TTD-SDA-001 now state `in_progress`, following Business Health's remediation. Historical assertions and validation refs remain recorded. Ontology refs now name real tests; tagged foundation regressions pass. No manual attestation or full sync was fabricated.
- Implemented: declared usage/improve skills, governed diagnostic reader, outcome inventory, regression tests and an unapproved development review packet. See PROGRESS.md for evidence and limitations.
- Product outcomes remain unqualified; generic proposal/review/apply work is still outstanding.

### Remaining qualification boundaries after setup

| Boundary | Evidence / owner action |
| --- | --- |
| Swarm cannot fingerprint the complete selected target | Preview rejects canonical DESIGN.md. Bug `knw-1788971670711410441`; retain the file in the packet. Swarm owner repair and its active launch-readiness plan precede a full development launch. |
| Documentation owner reports a false duplicate | experience/README.md and root README receive the same implicit alias despite distinct paths/types. Bug `knw-1788972142771687273`. Do not rename or omit the experience contract to silence it. |
| Repository-relative skill references reported missing | Files exist, but scenario docs health resolves marked paths incorrectly. Bug `knw-1788972143021234067`; keep canonical path semantics. |
| Program fixtures cannot be freshly qualified after a historical timeout | The owner includes historical budget findings in the gate that decides whether to execute fixtures. Bug `knw-1788972348435843167`. Retain the failure and repair that owner gate; do not widen the budget, erase history or run repeated successes to dilute it. The latest direct live reader succeeds, which is separate evidence. |
| Existing UI maturity debt | Test Genie run `20260909-163227-f7c7c663`: UI coverage command fails; direct render bypasses renderWithProviders; coverage warnings and an API seam warning remain. No thresholds or warnings were weakened. |
| Experience validator freshness | The installed command reports pass, but current-source auto-build still fails on its own missing protovalidate checksum. This is not current-source certification. |
| Dependency governance debt | SDA reports existing unrecorded dependencies and version/scope warnings for UI tooling. The approved protobuf checksum repair in this task does not resolve those independent findings. |
| Fresh-agent development proof | No mandate was approved or started. Execute TESTING.md's fresh-session protocol after Swarm qualification and explicit target/grant/budget review. |

The numeric performance, retention, backup and execution limits in DECISIONS.md
and PERFORMANCE.md are review proposals. They are not silent defaults or new
authority. The machine-readable preview limits cannot express every monetary or
cross-owner constraint; field completeness would not make that packet launchable.

### 2026-09-09 — shared-owner repair follow-up

The operator authorized repair of the shared blockers above. The earlier table
records the pre-repair observations; this section records their disposition.

- W3: Swarm now fingerprints canonical DESIGN.md and preserves PRD outcome IDs
  such as OT-P0-001. The full packet returns a digest instead of an argument
  error. Untyped proposed effects and missing acceptance resolvers still produce
  findings. No proposal was approved or launched.
- W3: Knowledge Observatory now gives explicit identifiers precedence over
  optional filename aliases. Ambiguous filenames do not invalidate distinct
  documents; explicit identifier collisions still fail. Marked repository paths
  use the repository root, with traversal and escaping symlinks rejected.
  Scenario-relative documentation relationships retain their existing root.
- W3: Program Runtime now gates fixture execution on the current static contract,
  not historical portfolio findings. Fresh fixtures pass in run
  `20260909-170118-74e98643`; the historical budget finding remains failed.
  This resolves fixture starvation, not the historical performance debt.
- W3: The UI setup no longer shadows the canonical helper with a process-wide
  QueryClient. The identity/isolation regression failed before repair and passes
  afterward. All UI assertions pass, including error, empty, and warning states.
  Input and textarea tests now use the canonical render helper; the final unit
  phase no longer reports render-policy drift.
- Experience Manager's CLI now rebuilds from current source and validates this
  experience contract without stale-binary fallback. Its checksum repair used
  Scenario Dependency Analyzer's approved install route.

Remaining qualification work is real, not waived: UI coverage is below its
existing floor; the historical program timeout remains in portfolio evidence;
full requirements-sync, product acceptance resolvers, reviewed grants/budgets,
and fresh-agent development proof remain outstanding. Existing dependency
governance findings outside the approved checksum repair remain outstanding.
Program Runtime restarted with a healthy API but a degraded Memory dependency;
the successful read-only fixtures do not certify that learning dependency.

The docs phase passes without contract errors or broken marked paths, but CLI
Health still rejects existing group help and the Test Genie registry manifest
omits build arguments/effects. These separate owner findings remain visible.
The architecture diagram is explicitly fenced as text, not executable shell.
See PROGRESS.md for exact evidence and the new owner report IDs.
