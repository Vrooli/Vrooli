# Implementation Plan — `chore/vrooli-emulator-documentation`

> **Round 2.** Round-1 decisions d1-d4 are settled (d1=B layout, d2=A external-url, d3=B baseline + follow-ups, d4=A full PRD). Round-2 raises d5-d8 (filename, per-route format, OT content, follow-up item list); see `workshop/round-002.json`.

## 1. Purpose

Produce the reference set future consumers (deployment-manager visual validation, scenario-to-desktop iframe embed, additional scenarios) read **before** integrating with the `vrooli-emulator` scenario. The documentation must cover:

1. The HTTP API contract under `/api/v1/sessions/`, including the `headless: true` mode and the required `app_path` field on `POST /control` `launch_app`.
2. The iframe embed protocol, including the `@vrooli/iframe-bridge` `SESSION` channel events emitted by the emulator UI and the `/embedded/emulator/external-url` endpoint that returns the open-in-new-tab URL.
3. The operator CLI surface (`vrooli-emulator session list|create|destroy|exec|logs`, plus `metrics tail`).
4. The PRD body (Overview, Operational Targets, Tech Snapshot, Dependencies, UX/Branding) absorbed from the retired `execute/vrooli-emulator-linux-first` umbrella — full pass per d4=A.
5. The operator runbook (start/stop service, create/destroy sessions, inspect metrics, tail logs, clean up stale sessions) absorbed from the same source.
6. The operational baseline values-as-docs per d3=B — *documenting the agreed defaults*, not enforcing them in code, with follow-up backlog items opened for each enforcement gap.

## 2. Required Reading

```bash
# Swarm-manager data model + plan authoring + skill discovery methodology
prompt-manager skill read swarm-manager-backlog-tools implementation-plan-authoring plan-skill-discovery

# Discovered domain skills (round-1)
prompt-manager skill read documentation-health api-steer scenario-generation
```

Repo-specific context:
- `swarm-manager initiatives get --name emulator-platform` — initiative rollup (13 items, 4 completed at round-2 time).
- `swarm-manager backlog get --kind execute --name scaffold-vrooli-emulator-scenario` — completed sibling that ships the API + CLI surface this doc describes.
- `swarm-manager backlog get --kind execute --name vrooli-emulator-standalone-ui` — completed sibling that ships the bridge-emitting UI.
- `swarm-manager backlog get --kind execute --name vrooli-emulator-external-url-endpoint` — sibling that will add the `session_id` query param to the existing external-url endpoint (still in `backlog`).
- `scenarios/swarm-manager/execute/vrooli-emulator-linux-first/plan.md` §6 + §8 Phase A — original spec for the absorbed PRD/runbook/baseline scope.
- `scenarios/vrooli-emulator/api/livedesktop/handler.go` — canonical API route table (RegisterRoutes, lines 24-34).
- `scenarios/vrooli-emulator/api/livedesktop/types.go` — `SessionConfig` (line 78), `SessionView` (line 105), `MetricsView` (line 88), `SessionState` (line 11). Note: line 87 has an existing `// DOC:` annotation pointing at `docs/reference/live-desktop-api.md#process-metrics`.
- `scenarios/vrooli-emulator/api/livedesktop/action.go` — control-action registry (the 14 actions surfaced by `POST /control`).
- `scenarios/vrooli-emulator/api/procmetrics/types.go` — also has a `// DOC: docs/reference/live-desktop-api.md#process-metrics` annotation on line 1.
- `scenarios/vrooli-emulator/cli/domains/sessions/sessions.go` — the operator CLI subcommand registration (`Register` at line 58).
- `scenarios/vrooli-emulator/cli/domains/metrics/metrics.go` — the `metrics tail` subcommand.
- `scenarios/vrooli-emulator/ui/server.js` + `@vrooli/api-base/server` — confirms `/embedded/emulator/external-url` is auto-mounted today.
- `scenarios/vrooli-emulator/docs/embed-protocol.md` — pre-existing bridge contract doc; per d1=B will be moved to `docs/concepts/embed-protocol.md` and integrated (not rewritten).
- `scenarios/vrooli-emulator/PRD.md` — current template scaffold; full pass per d4=A.
- `scenarios/vrooli-emulator/README.md` — currently the react-vite template scaffold; needs emulator-specific quickstart.

## 3. Greenfield Declaration

This is a **greenfield documentation pass**. The scenario is new (initialized 2026-04-18), all docs except `embed-protocol.md` and the seed `PROGRESS.md` are absent or template-default, and the PRD has never been filled. There is no legacy doc structure to maintain compatibility with, no deprecated reference paths to redirect from, and no external consumers depending on a specific filename or anchor today. The chore writes a clean docs/ tree from scratch.

**No code changes.** This chore writes only documentation files (`docs/**`, `README.md`, `PRD.md`). It does not touch API handlers, CLI commands, UI components, service.json, or tests. If documentation gaps surface code defects (e.g., missing limits, undocumented behavior), those become follow-up backlog items per d3=B / d8, not in-place code fixes inside this chore.

**Pre-existing source coupling.** Two source files already carry `// DOC:` annotations pointing at `docs/reference/live-desktop-api.md#process-metrics` (see §2 Required Reading). d5 decides whether the new API reference file honors that path (no code changes) or supersedes it (requires editing those annotations, which conflicts with the no-code-changes rule above and would have to spawn a follow-up).

## 4. Problem Statement

External consumers cannot integrate with `vrooli-emulator` without reading source code. There is no API reference, no CLI reference, no operator runbook, and no PRD. Specifically:

- **API contract is implicit.** Routes are defined in `livedesktop/handler.go` (10 routes), request/response shapes in `livedesktop/types.go`, and the 14 control actions in `livedesktop/action.go`, but no document maps method+path → request body → response shape → status codes → error semantics. Future consumers (deployment-manager visual validation, scenario-to-desktop iframe embed, third-party scenarios) currently must grep handlers to learn the surface.
- **Embed protocol is half-documented.** `docs/embed-protocol.md` covers the bridge `SESSION` channel events, but does not cover the `/embedded/emulator/external-url` endpoint or the recommended iframe loading pattern for hosts.
- **CLI surface is undocumented.** Operators must run `vrooli-emulator session --help` to discover commands. No reference for flags, output formats, or the relationship between `session exec` actions and the API control-action registry.
- **PRD is template-only.** `PRD.md` is the react-vite scaffold with placeholder text in every section (verified in round-2). No Operational Targets exist; no Tech Snapshot or Dependencies are recorded; no UX/Branding direction is set.
- **Runbook absent.** Operators have no documented procedure for routine tasks (start/stop, create/destroy sessions, inspect metrics, tail logs, clean up stale sessions).
- **Operational baselines are tribal knowledge.** Defaults are scattered across code: `service.go` defaults width/height to 1280x720; `main.go` configures the janitor at 30s check / 30min idle; there is no documented max-session cap or per-session resource ceiling.

The retired umbrella `execute/vrooli-emulator-linux-first` formally rehomed the PRD/runbook/operational-baseline deliverables into this chore (round 2 d1=A on that item), so closing this chore also closes the rehoming loop on the retired umbrella.

## 5. Scope

### In scope
- Author the API reference (filename per d5) covering every route in `livedesktop/handler.go` (POST/GET/DELETE `/api/v1/sessions[/...]`, `/heartbeat`, `/launch`, `/control`, `/metrics`, `/files/{filename}`, `/ws`). Per-route format per d6.
- Author `docs/reference/cli.md` for `vrooli-emulator session list|create|destroy|exec|logs` and the `metrics tail` group, including flag tables and example outputs.
- Author `docs/concepts/embed-protocol.md` (move existing `docs/embed-protocol.md` per d1=B) covering: (a) the `SESSION` channel events, (b) the existing `/embedded/emulator/external-url` endpoint behavior, (c) the planned `session_id` deep-link param marked as TBD owned by `execute/vrooli-emulator-external-url-endpoint` per d2=A.
- Author `docs/guides/runbook.md` covering start/stop, session create/destroy, metrics inspection, log tailing, stale-session cleanup.
- Author `docs/internal/operational-baseline.md` documenting the current code defaults verbatim per d3=B (1280x720 default resolution, 30s janitor / 30min idle, no max-session cap, no per-session ceilings, Xvfb display range delegated to screenrecording.DisplayManager).
- Fill `PRD.md` per d4=A — Overview, Operational Targets (P0/P1/P2 framing per d7), Tech Snapshot, Dependencies, UX/Branding.
- Replace `README.md` with an emulator-specific quickstart that links into `docs/`.
- Add `docs/manifest.json` listing every doc file in navigation order so the docs tree is navigable in the UI shell.
- File the d3=B follow-up backlog items per d8 (recommended: max-session cap, per-session resource ceilings, explicit Xvfb display range, broader source-code `// DOC:` annotations).

### Out of scope
- Any code changes inside `api/`, `cli/`, `ui/`, `.vrooli/`, or `requirements/`. If gaps surface, they are recorded as new backlog items per d8, not fixed here.
- Implementing the `session_id` query-param deep-link on `/embedded/emulator/external-url` — owned by `execute/vrooli-emulator-external-url-endpoint`.
- Implementing per-session resource ceilings, max-session caps, or explicit Xvfb display ranges — these are values-as-docs only (d3=B); enforcement is opened as follow-up items per d8, not done here.
- Editing the existing `// DOC:` annotations in `api/livedesktop/types.go:87` and `api/procmetrics/types.go:1` — out of scope per §3; covered by the d5 decision (option A avoids the issue entirely).
- Reopening or revising the round-2 absorbed-scope decision from the retired umbrella.
- Modifying `archive/` on this item (reserved for user-provided context).
- Modifying any sibling backlog item's spec or plan.

## 6. Current Technical Context

**Surface area to document (verified against code at round-2 time):**

| Surface | Source | Notes |
|---|---|---|
| API routes | `api/livedesktop/handler.go:24-34` `RegisterRoutes` | 10 routes. Verified at round-2: `POST /api/v1/sessions`, `GET /api/v1/sessions`, `GET /api/v1/sessions/{id}`, `POST /api/v1/sessions/{id}/heartbeat`, `POST /api/v1/sessions/{id}/launch`, `POST /api/v1/sessions/{id}/control`, `GET /api/v1/sessions/{id}/metrics`, `GET /api/v1/sessions/{id}/files/{filename}`, `DELETE /api/v1/sessions/{id}`, `GET /api/v1/sessions/{id}/ws` (WS proxy). |
| Request types | `api/livedesktop/types.go:78` `SessionConfig` | `width`, `height`, `scenario_name`, `app_path` (omitempty), `platform` (omitempty), `headless` (omitempty). `app_path` required only on `launch_app` action — session creation can omit it. |
| Response types | `api/livedesktop/types.go:88,105` `SessionView`, `MetricsView` | Includes `metrics` snapshot inline when a monitor is attached. `MetricsView` has a `// DOC:` annotation at line 87 referencing `docs/reference/live-desktop-api.md#process-metrics`. |
| Control actions | `api/livedesktop/action.go:21-42` `actionRegistry` | 14 actions verified: `launch_app`, `quit_app`, `screenshot`, `start_recording`, `stop_recording`, `offline_mode`, `slow_connection`, `inject_env`, `resize_display`, `clipboard_read`, `clipboard_write`, `dark_mode`, `locale`. |
| CLI subcommands | `cli/domains/sessions/sessions.go:58` + `cli/domains/metrics/metrics.go` | `session {list,create,destroy,exec,logs}` + `metrics {tail}`. |
| Embed events | `ui/src/lib/bridge.ts` + `docs/embed-protocol.md` | `SESSION` channel, four event types (`session.created`, `session.state_changed`, `session.error`, `session.destroyed`), envelope `v: 1`. |
| External-url endpoint | `ui/server.js` → `@vrooli/api-base/server` | `GET /embedded/emulator/external-url` auto-mounted, returns context-aware UI URL. The dedicated execute item adds a `session_id` deep-link param. |

**Code defaults (per d3=B documented verbatim, not enforced):**
- Default resolution: 1280x720 (`livedesktop/service.go:StartSession`).
- Janitor: check every 30s, idle timeout 30min (`api/main.go:118`).
- Max concurrent sessions: **uncapped** (no enforcement in `livedesktop/store.go`). → d8 follow-up.
- Per-session CPU/memory/disk ceilings: **none** (no `rlimit`, no `cgroups` wiring in `platform_linux.go`). → d8 follow-up.
- Xvfb display number: delegated to `screenrecording.NewDisplayManager()` — no explicit range documented in the emulator code. → d8 follow-up.
- Bridge target origin: `VITE_PARENT_ORIGIN` env var, falls back to `document.referrer`'s origin, otherwise `"*"` (`ui/src/lib/bridge.ts:35`).

**External-url endpoint state:** `GET /embedded/emulator/external-url` is **already live** today via `@vrooli/api-base/server` — verified in `ui/server.js`. The dedicated execute backlog item (`execute/vrooli-emulator-external-url-endpoint`, status `backlog`) adds a `session_id` query param for deep-linking; that param does not exist yet. Per d2=A the doc records the current behavior plus a forward reference.

**Doc-tree layout (settled by d1=B):**
```
docs/
├── reference/
│   ├── <api-filename>.md     # filename TBD by d5: live-desktop-api.md (recommended) or api.md
│   └── cli.md
├── guides/
│   └── runbook.md
├── concepts/
│   └── embed-protocol.md     # moved from docs/embed-protocol.md
├── internal/
│   └── operational-baseline.md
├── manifest.json
└── PROGRESS.md               # untouched
PRD.md                        # filled per d4=A
README.md                     # replaced with emulator-specific quickstart
```

**Acceptance allow posture:** `scenarios/vrooli-emulator/docs/**`, `scenarios/vrooli-emulator/README.md`, `scenarios/vrooli-emulator/PRD.md` (PRD added in round 1). Verified to cover every file written in §5 In scope.

## 7. Target End State

- `docs/` tree matches the d1=B layout shown in §6, with every file populated.
- API reference filename matches d5; if d5=A, the existing `// DOC:` annotations remain accurate and no source code is touched.
- Per-route documentation matches the depth chosen in d6.
- `PRD.md` filled to the depth chosen in d4=A; OT-P0/P1/P2 reflect the framing chosen in d7. No template placeholders remain in Overview, Operational Targets, Tech Snapshot, Dependencies, UX/Branding.
- `README.md` replaced with emulator-specific quickstart content; no react-vite template boilerplate remains.
- `docs/internal/operational-baseline.md` documents every code default listed in §6.
- `docs/concepts/embed-protocol.md` integrates the existing bridge contract content and adds the external-url endpoint section per d2=A.
- The d8-chosen follow-up backlog items exist and are tagged `emulator`.
- `docs/manifest.json` exists and lists every doc file.
- `validation-report.json` `passed: true` (all 13 mandatory plan sections present, no warnings).
- `swarm-manager backlog get --kind chore --name vrooli-emulator-documentation` shows `status: completed`.
- Initiative `emulator-platform` rollup: completed count incremented by 1 (4 → 5).

## 8. Implementation Strategy

**Phase A — Resolve outstanding decisions (workshop, this round + next).**
- Round 1 settled d1-d4. Round 2 surfaces d5-d8. Round 3 (post-user-input on d5-d8) synthesizes the remaining decisions into §9 Contract Decisions and finalizes §11 / the doc-writing tasks.

**Phase B — Author the doc tree.** *(executed only after all eight decisions are settled and the plan is queued for execution.)*

1. **Layout setup.** `mkdir -p scenarios/vrooli-emulator/docs/{reference,guides,concepts,internal}`. `git mv scenarios/vrooli-emulator/docs/embed-protocol.md scenarios/vrooli-emulator/docs/concepts/embed-protocol.md`. Verify no other repo paths reference the old `docs/embed-protocol.md` location with `rg "docs/embed-protocol\.md"` — update any in-scope references; out-of-scope hits are recorded but not modified.
2. **API reference.** Write `scenarios/vrooli-emulator/docs/reference/<d5-filename>.md` covering all 10 routes from `handler.go:24-34` and all 14 actions from `action.go:21-42`. Per-route format per d6. Include the `#process-metrics` anchor that the existing `// DOC:` annotations point at. Cross-reference `docs/concepts/embed-protocol.md` from the embed/external-url section.
3. **CLI reference.** Write `scenarios/vrooli-emulator/docs/reference/cli.md` covering every subcommand registered by `cli/domains/sessions/sessions.go:58` and `cli/domains/metrics/metrics.go`. Document the human-output report contract (Status/Triage/Next Steps for operational, Summary/Results/Retrieval Hints for list, Result/What Changed/Next Command for mutation) and the `--json` flag where applicable.
4. **Embed protocol concepts doc.** Refresh `docs/concepts/embed-protocol.md` (already moved in step 1) — keep the bridge-contract content, add a new "External-url endpoint" section per d2=A documenting the existing default behavior with a forward reference to `execute/vrooli-emulator-external-url-endpoint` for the planned `session_id` deep-link.
5. **Runbook.** Write `docs/guides/runbook.md` covering: start/stop the API + UI, create a session, launch an app, fetch metrics, tail logs, destroy a session, what the janitor does and when it kicks in, how to clean up stale sessions manually if it doesn't.
6. **Operational baseline.** Write `docs/internal/operational-baseline.md` listing every default in §6 verbatim, with file:line citations, and a note that enforcement gaps are tracked as the follow-up items filed in step 9.
7. **PRD fill-out.** Update `PRD.md` Overview, Operational Targets (per d7-chosen framing), Tech Snapshot, Dependencies, UX/Branding sections. Leave `📎 Appendix` as-is.
8. **README replacement.** Replace `README.md` with emulator-specific quickstart: install, start the scenario, create a session via CLI, embed the UI in an iframe, links into `docs/reference/`, `docs/guides/`, `docs/concepts/`.
9. **Manifest.** Write `docs/manifest.json` listing every doc file with title and order.
10. **File follow-ups.** Run `swarm-manager backlog create` for each item identified by d8.
11. **Cross-link verification.** `rg "docs/reference/<d5-filename>"` should hit the existing source `// DOC:` annotations (if d5=A) and any new cross-references in the docs tree itself.

**Phase C — Validate and queue.**
1. Run `swarm-manager backlog get` to confirm `acceptance_allow` covers every file written.
2. Run the automated coverage greps from §10 against the new docs.
3. Run `swarm-manager backlog file-upload` for `plan.md` and `validation-report.json`; confirm `passed: true`.
4. Queue the chore for review.

## 9. Contract Decisions

**Settled in round 1:**
- **d1=B (Documentation file layout):** Standard layout per documentation-health skill — `docs/reference/{api,cli}.md`, `docs/guides/runbook.md`, `docs/concepts/embed-protocol.md`, `docs/internal/operational-baseline.md`, `docs/manifest.json`. The existing `docs/embed-protocol.md` is moved (not rewritten) into `docs/concepts/`.
- **d2=A (External-url endpoint coverage):** Document the existing default `/embedded/emulator/external-url` behavior now; mark the planned `session_id` query param as TBD owned by `execute/vrooli-emulator-external-url-endpoint`. Do not add that sibling as a `depends_on` of this chore.
- **d3=B (Operational baseline values-as-docs):** Document current defaults verbatim AND open follow-up backlog items for the missing limits (concrete list decided by d8). No enforcement in this chore.
- **d4=A (PRD fill-out depth):** Full PRD pass — Overview, Operational Targets (P0/P1/P2 framing per d7), Tech Snapshot, Dependencies, UX/Branding.

**Pending in round 2:**
- **d5 (API reference filename):** TBD — recommended A (`docs/reference/live-desktop-api.md` to honor existing `// DOC:` annotations).
- **d6 (Per-route documentation format):** TBD — recommended A (verbose markdown with curl examples and field tables).
- **d7 (PRD Operational Targets framing):** TBD — recommended A (capability-focused).
- **d8 (d3=B follow-up backlog items list):** TBD — recommended A (file 4 follow-ups: max-session cap, per-session resource ceilings, explicit Xvfb display range, broader `// DOC:` source annotations).

## 10. Testing Plan

Documentation chores have no runtime tests; verification is by structural and content checks.

**Automated checks (run on the chore branch from repo root after Phase B completes):**
- `swarm-manager backlog files --kind chore --name vrooli-emulator-documentation` lists every expected file.
- **Route coverage** — for every route in `api/livedesktop/handler.go:24-34`, the API reference contains a heading or table row mentioning that path. Concretely:
  ```bash
  for path in /api/v1/sessions /api/v1/sessions/{id} /api/v1/sessions/{id}/heartbeat \
              /api/v1/sessions/{id}/launch /api/v1/sessions/{id}/control \
              /api/v1/sessions/{id}/metrics /api/v1/sessions/{id}/files/{filename} \
              /api/v1/sessions/{id}/ws; do
    rg --quiet -F "$path" scenarios/vrooli-emulator/docs/reference/<d5-filename>.md \
      || { echo "MISSING route: $path"; exit 1; }
  done
  ```
- **Action coverage** — every action key from `api/livedesktop/action.go:21-42` (`launch_app`, `quit_app`, `screenshot`, `start_recording`, `stop_recording`, `offline_mode`, `slow_connection`, `inject_env`, `resize_display`, `clipboard_read`, `clipboard_write`, `dark_mode`, `locale`) appears in the API reference.
- **CLI coverage** — every subcommand name from `cli/domains/sessions/sessions.go` (`list`, `create`, `destroy`, `exec`, `logs`) and `cli/domains/metrics/metrics.go` (`tail`) appears in `docs/reference/cli.md`.
- **Event coverage** — every `SessionEventType` (`session.created`, `session.state_changed`, `session.error`, `session.destroyed`) appears in `docs/concepts/embed-protocol.md`.
- **Anchor preservation** — `rg "live-desktop-api\.md#process-metrics" scenarios/vrooli-emulator/api/` returns the two existing `// DOC:` annotation hits (verifies d5=A wasn't undermined). If d5=B was chosen, this check is replaced by verification that the annotations were updated to the new filename.
- **Manifest completeness** — every `.md` file under `docs/` (except `PROGRESS.md` if intentionally excluded) is listed in `docs/manifest.json`.
- **No template residue** — `rg "\[Summarize the permanent capability|\[Outcome title\]|\[Who benefits\]|\[React, Go, etc\.\]" scenarios/vrooli-emulator/PRD.md` returns no hits.
- **Acceptance allow alignment** — every file written matches one of the `acceptance_allow` globs from `spec.json`. Computed by listing changed files and filtering against the globs.
- **Plan validity** — `validation-report.json` `passed: true` after final upload.

**Manual checks:**
- A reader unfamiliar with the codebase can, from the docs alone, write a minimal client that creates a session, launches an app, fetches metrics, and tears down — without reading source. (This is the operational test of d6's depth choice.)
- The PRD's Operational Targets (per d7) reference real, measurable outcomes — not template placeholder text.
- The d8-filed follow-up items are visible in `swarm-manager initiatives get --name emulator-platform` rollup.

## 11. Rollout / Validation Checklist

- [x] Round-1 workshop decisions d1-d4 settled.
- [ ] Round-2 workshop decisions d5-d8 settled.
- [ ] Plan §9 Contract Decisions reflects all eight settled choices.
- [ ] Phase B doc files written and uploaded via `swarm-manager backlog file-upload` AND written directly to the live scenario tree on disk (`scenarios/vrooli-emulator/docs/**`, `PRD.md`, `README.md`).
- [ ] All automated checks in §10 Testing Plan pass.
- [ ] d8-chosen follow-up backlog items exist (`swarm-manager backlog get` for each).
- [ ] `validation-report.json` `passed: true` after final upload.
- [ ] `acceptance_allow` covers every file written (re-verify after Phase B completes).
- [ ] Initiative rollup shows the expected completed-count increment (4 → 5).

## 12. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Documented API contract drifts from `handler.go` after this chore ships. | High over time | Medium — wrong docs erode trust. | The documentation-health pattern of source-side `// DOC:` annotations is already partially in use (`types.go:87`, `procmetrics/types.go:1`). Broader adoption is filed as one of the d8=A follow-ups so future code changes touching those files are visibly coupled to the docs that describe them. Final placement of `// DOC:` tags is itself a code change and is **out of scope for this chore**. |
| API doc filename mismatch with existing `// DOC:` annotations. | High if d5=B/C, Low if d5=A | Medium — readers get a 404 from the source-coupled link. | d5=A (recommended) honors the existing `docs/reference/live-desktop-api.md` path. d5=B requires editing two source annotations, which §3 forbids and would have to be filed as a separate follow-up before this chore can ship. d5=C creates new mismatch by splitting the file. |
| `vrooli-emulator-external-url-endpoint` ships a different deep-link contract than what we document under d2=A's TBD note. | Medium | Low — the TBD note is explicitly forward-looking. | Cross-reference the sibling backlog item by name in the doc; on its completion, update the doc as part of that item's plan. |
| d8=A follow-ups never get prioritized, leaving the documented "no cap" gap permanent. | Medium | Medium — operators may hit OOM in production without warning. | Open the follow-ups inside this chore's Phase B step 10 (not after) so they exist as concrete backlog items before this chore closes. Tag them with `emulator` so the initiative owner sees them. |
| PRD Operational Targets (d7) commit the initiative to outcomes that haven't been validated. | Medium | Medium — mis-set OTs distort future prioritization. | All three d7 framings derive OTs from existing initiative scope and completed sibling work — none commit to numerical SLOs (option C is explicitly flagged as that risk). The recommended option A maps OTs to verifiable code surfaces. |
| Moving `docs/embed-protocol.md` into `docs/concepts/` breaks an external link to its current path. | Low | Low — scenario is new (initialized 2026-04-18); no external links observed in repo. | Phase B step 1 runs `rg "docs/embed-protocol\.md"` before moving and updates any in-scope references; out-of-scope hits are recorded but not modified. |
| `acceptance_allow` regresses (e.g., we forget to add a new file's path before writing). | Low | Low — caught by `swarm-manager backlog get` validation in §11. | Re-verify after every Phase B sub-step. Currently set to `docs/**`, `README.md`, `PRD.md` — covers every file in §5 In scope. |
| Per-route format chosen in d6 turns out to be too terse to satisfy the §10 manual check. | Low if d6=A | Medium — readers fall back to source. | d6=A (recommended) is the verbose option specifically designed to satisfy the "write a client without reading source" manual check. d6=B/C either elide examples or add tooling overhead with no payoff. |

## 13. Non-goals / Prohibited Patterns

- **No code changes.** If documentation reveals a code defect, open a new backlog item per d8; do not fix in-place inside this chore.
- **No PRD changes outside the sections d4=A names.** Other PRD sections (📎 Appendix) remain template-default until a separate product-strategy round.
- **No new `// DOC:` source-code annotations in this chore** — those are code edits and live in the d8=A follow-up.
- **No editing the existing `// DOC:` annotations in `api/livedesktop/types.go:87` and `api/procmetrics/types.go:1`.** d5=A avoids this entirely; if d5=B is chosen, the edit must be filed as a separate follow-up before this chore can ship.
- **No backwards-compatibility shims** for the moved `embed-protocol.md` (greenfield — see §3).
- **No edits to `archive/`** — reserved for user-provided context per swarm-manager-backlog-tools.
- **No restart of the vrooli-emulator scenario from inside the chore** — see §15.

## 14. Definition of Done

- Workshop decisions d1-d8 all have `selected` populated; round-3 synthesis run.
- §9 Contract Decisions populated for all eight decisions.
- Every file listed in §7 Target End State exists with non-template content.
- Every automated check in §10 Testing Plan passes.
- d8-chosen follow-up backlog items exist and are tagged `emulator`.
- `validation-report.json` `passed: true`.
- `swarm-manager backlog update --kind chore --name vrooli-emulator-documentation --data '{"status":"completed"}'` succeeds; initiative rollup reflects the 4 → 5 increment.

## 15. Scenario Restart Note

This chore writes documentation only — no scenario code, no service.json, no lifecycle wiring. **No `vrooli scenario restart vrooli-emulator` is required** as part of execution. If a future round changes that (e.g., a follow-up adds a manifest the UI shell loads at runtime), this section will be updated to add the restart step. Per the user's standing guidance, even if a restart were required, this plan would only document that the user must run it manually — Claude Code does not restart the scenario it is operating inside.
