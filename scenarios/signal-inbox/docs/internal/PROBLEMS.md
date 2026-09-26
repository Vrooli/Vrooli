# Problems — Signal Inbox

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Append evidence-backed entries as they appear.

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
### 2026-07-27 — Scenario Auditor cannot rebuild in the current toolchain cache

**Symptom:** `scenario-auditor standards scan signal-inbox --wait` cannot start its
local API. Its stale CLI attempts an auto-rebuild, which fails before scanning with
Go standard-library object version mismatches (`go1.21.13` cached objects versus
the active Go `1.25.12` toolchain).

**Root cause:** Lifecycle startup injects an incompatible local `GOROOT` while
the selected compiler is Go `1.25.12`; that older tree lacks packages such as
`math/rand/v2`. Signal Inbox code is not implicated; the scanner is also stopped.

**Workaround:** `env -u GOROOT make endpoints` works for direct scenario-local Go
commands. Keep using local API/UI/CLI validation and Test Genie evidence; do not claim
the standards scan passed while lifecycle still restores the incompatible variable.

**Real fix:** Correct lifecycle's Go environment, then rebuild and start
`scenario-auditor` with the active Go toolchain and address any Signal-Inbox-owned
findings from a new standards scan.

**Owner:** Scenario Auditor / development-toolchain maintainers.

**Refs:** `scenario-auditor standards scan signal-inbox --wait`, 2026-07-27.

### 2026-07-28 — Proto Health retains a pre-migration shared-type finding

**Symptom:** After moving the reusable `Signal` message and `SourceKind` enum
from `signal-inbox/v1/signals/signals.proto` to
`signal-inbox/v1/shared/signals.proto`, regenerating all proto artifacts, and
passing both API and CLI Go suites, `proto-health --auto-start validate scenario
signal-inbox --json` still reports the old location
`signal-inbox/v1/signals/signals.proto#Signal`.

**Root cause:** Proto Health's lifecycle-managed Code Facts analysis is not
refreshing to the generated schema state. Its running analysis path shares the
incompatible lifecycle Go environment described above, so the report cannot be
treated as a current source snapshot.

**Workaround:** Treat the source schema, regenerated artifacts, and passing
`api`/`cli` suites as the current local evidence; do not mark Proto Health green
until its lifecycle is rebuilt and it reports the shared location itself.

**Real fix:** Correct lifecycle toolchain injection, restart Proto Health/Code
Facts, and rerun the provider validation against the current descriptor.

**Owner:** Proto Health / Code Facts maintainers.

**Refs:** `packages/proto/schemas/signal-inbox/v1/shared/signals.proto`,
`proto-health --auto-start validate scenario signal-inbox --json`, 2026-07-28.

### 2026-07-27 - Resolved: Proto Health descriptor stale after schema migration

The shared Signal migration was correct in source and generated artifacts, but a running Proto Health process retained a pre-migration descriptor image and reported signals.Signal. Restarting Proto Health through vrooli scenario restart proto-health reloaded the descriptor; proto-health validate scenario signal-inbox then passed with shared.Signal clean. Final Test Genie run 20260728-031239-3f8b6b5e passed all 20 phases.

### 2026-07-28 - Test Genie UI-health RPC returns unexpected EOF despite direct provider pass

Full runs 20260728-131830-b919b673 and 20260728-132357-41dba7cc each failed only the ui-health phase. The latter phase command artifact artifact_93d7b474f314c02862ff1bffbaa58f2e records: unavailable: unexpected EOF. After a supported make start rebuild, direct ui-health validate scenario signal-inbox --json returned VALIDATION_STATUS_PASSED, including bundle freshness and desktop/mobile render plus iframe-handshake evidence. This is reproducible Test Genie-to-UI-Health RPC infrastructure drift, not established Signal Inbox UI behavior. Scenario QA bug knw-1785245290471420146 was filed. Keep full-suite-only validation evidence and affected requirement statuses planned until the owner resolves the RPC path; continue using direct provider evidence for scoped UI verification.
## Work ladder

- Rung: W3 / R1
- Evidence: W0 re-measured on 2026-07-28: goal `bookmark-intelligence-hub-rework-and-ideation` directs manual/tier-0 capture, immutable storage, operator-defined/confirmed categories, corpus-wide retrieval independent of category or disposition, and anomalous-adapter disable/no retry; these agree with `OT-P0-001` through `OT-P0-014`. D-030 and D-031 refine source-adapter configuration and local sensitive-content processing without contradicting a P0 target. W1 passed with `business-health validate scenario signal-inbox`; W2 passed with `vrooli scenario requirements validate signal-inbox --json`. The comprehensive run `20260729-014913-fd10f41e` kept contracts L4, API L3, architecture L3, dependencies L3, business L3, workflow L5, proto L3, and templates L4. It failed the UI-health freshness gate because source changed after the prior lifecycle rebuild.
- Blocker: R0 is re-opened by `freshness_ui_bundle_stale`; rebuild through scenario lifecycle, then re-run the full suite before acting on lower maturity findings. Security has advisory vulnerability intelligence debt, and `SEARCH_EVAL_CORPUS_INADEQUATE` remains an honest evidence limit until enough reviewed real signals exist.
- Measured: 2026-07-29

The table below records **open gaps carried out of the design workshop** and
observed validation limits. They are listed here rather than in
[`DECISIONS.md`](DECISIONS.md) because a decision records what was chosen, while
these record what is still unknown.

| Gap | Why it is open | Blocks | Resolution path |
|---|---|---|---|
| `video-downloader` has no transcript-only request | The capability is assumed by `SIG-P1-005` and does not exist today. The scenario must not work around it by downloading media and transcribing locally — that duplicates a whole problem domain and violates D-018. | `SIG-P1-005` | Confirm the contract with `video-downloader` before planning P1. Until then the requirement stays blocked rather than resolved another way. |
| Large archive imports lack a durable progress/resume job | The bounded multipart endpoint successfully imported the full operator X archive: 3,492 authored records and 152,294 likes. The likes run took almost three minutes, which can outlive an interactive client even though no platform request is made. | Operator experience for future large archives, not the completed X corpus import | Move large archive parsing/import into a durable server-owned job with progress, resume, and cancellation; preserve independently enabled authored and likes streams. Never ingest DMs, ads, contacts, profile/security/account data, or media merely because the ZIP carries them. |
| Source-stream configuration is not implemented | Current adapter state is one toggle per adapter. The approved model requires one source to expose independently enabled streams with distinct priority, local-processing policy, checkpoint, and credential reference. | Safe X bookmarks, archive, and like synchronization | Add stream-level state and operator controls before enabling any tier-1 X bookmarks integration. |
| Classification accuracy has no baseline | The classifier's real accuracy is unknown and cannot be estimated before a corpus exists. The predecessor asserted ">85%, >95% with learning" with no way to measure either. | Trusting `SIG-P0-005` proposals | `SIG-P1-008` measures against the override corpus. Until it reports, every proposal is reviewed rather than trusted. |
| The ambient-view budget default is unvalidated | Chosen by reasoning about attention, not measured against real consumption. Too small hides work; too large reintroduces the context tax the budget exists to prevent. | Nothing; it is tunable | Calibrate against real walk-prep consumption once signals exist. |
| Disposition keeps no history | Only the current value is stored, so "when was this marked done, and by whom" is unanswerable. | Triage auditing, if ever needed | Accepted deliberately to keep the model small. Add a transition log only if a real question demands it. |
| Multi-consumer disposition is undefined | Single-primary-category (D-010) avoids the problem rather than solving it. If two consumers ever share a signal, "handled" has no single answer. | `SIG-P2-003` | Define the ownership rule before relaxing the single-category constraint, not after. |
| No hard-delete path exists | Deletion is deliberately absent (D-006), but material captured in genuine error — a credential in a screenshot, say — currently has no removal path. | Nothing today | If added, it must be an explicit, logged, single-signal operation and never reachable from a filter-driven bulk path. |
| Blob-store growth is unbounded | Pasted images accumulate with no size budget and no measurement. | Nothing today | Measure after the first real corpus; add a budget only if it turns out to matter. |
| Phase 1 baseline comparison was not comparable | Git Control Tower validation operation `4c7ea8aa-41e2-435c-9c30-336b913a1da9` returned `signal-inbox:ready:regression` with the only detail `phase did not apply to one or both captured targets`; its two proto-path checks have no typed producer synchronization. The native inference tests, `ai-go/search` suite, and linter passed. | Automated phase-close evidence only; no requirement status was advanced. | Re-run the phase baseline comparison after the first signal-inbox proto/API slice produces a typed target; do not treat the current baseline verdict as behavioral evidence for a requirement. |
| Search eval corpus is intentionally inadequate | `.vrooli/search.json` declares the provider contract and a junk negative. One real saved Reddit URL is now present, but there are not yet enough reviewed, diverse real signals for honest positive cases or hard cases. Search Hub fast validation reports `SEARCH_EVAL_CORPUS_INADEQUATE`. | `SIG-P0-011` certification and final search phase | After the operator imports actual Chrome and X exports alongside the Reddit archive, review at least three real queries against their real signal ids and add them as reviewed positives. Do not fabricate fixture or placeholder ids. |

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Validation infrastructure | Scenario Auditor and Code Facts cannot currently start because lifecycle restores an incompatible Go root. | Architecture and code-fact findings cannot be independently re-measured by their owning services. | Correct the lifecycle Go environment, then rerun the owning scans and repair only the Signal-Inbox findings they produce. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
