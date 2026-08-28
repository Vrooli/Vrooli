# Roles and handoffs — captured baseline

The state of `web-console` before and after the roles-and-handoffs work, so a
later reader can tell which failures this plan inherited from which it caused.

Plan: `web-console-roles-and-handoffs-group-first-sessions-waiting`
Design record: [`../ROLES-AND-HANDOFFS-UX.md`](../ROLES-AND-HANDOFFS-UX.md)

---

## 1. Scenario suite baseline

**Run id:** `20260827-194738-d489accf`
**Verdict:** FAIL — 12 phases passed, 10 failed, 1 skipped, 439s.

| Phase | Result |
|---|---|
| portability, structure, contracts, dependencies, quality, integration, business, measures, branding, monetization-conformance, tidiness, config | passed |
| ui-health, docs, soak, performance, unit, storage, workflow, experience, security, proto | **failed** |
| templates | skipped |

**Caveat on this capture, stated plainly:** the run started at 19:47:56 and
finished ~19:55:15. Contract-only edits (the three protos, the regenerated
bindings, and `schema.sql`) landed at 19:49–19:52, inside that window. The
suite was therefore started before any edit but did not complete before the
first ones landed, so late phases (`proto` at position 20 of 23, `templates` at
23) may have seen partial work. Earlier phases did not. A reader should treat
the `proto` and `templates` results as unreliable and the rest as a true
baseline. No UI or Go source had been edited at any point during the run.

## 2. UI test baseline

Every failure below was present before this plan's UI work and remains after
it. All of them are assertions on **react-component-library overlay
rendering** — panel class names, focus trapping, and Escape resolution in
`ResponsiveDialog`, `DrawerShell`, and `FullPageDrawer`. The library is on this
plan's change boundary deny list and was not touched.

| File | Failing | Cause |
|---|---|---|
| `src/__tests__/MessagesPane.test.tsx` | 4 | Mermaid viewer renders two close controls; `rcl-full-page-drawer__panel` no longer carries `--wc-safe-top`; export drawer renders two close controls |
| `src/__tests__/composer-attachments.test.tsx` | 3 | Focus trap and backdrop dismissal in the RCL drawer |
| `src/__tests__/full-screen-composer.test.tsx` | 3 | Same drawer; draft round-trip through minimize/expand |
| `src/__tests__/ArchiveDrawer.test.tsx` | 1 | `rcl-responsive-dialog__panel` no longer carries `md:max-w-md` |
| `src/__tests__/MessageExportDrawer.test.tsx` | 1 | Two elements match the close label |
| `src/components/__tests__/MessagesMermaidViewer.test.tsx` | 1 | Two elements match the close label |
| `src/__tests__/ai-component.test.ts` | 1 | Compact drawer dialog semantics and autofocus |
| **Total** | **14** | |

These are filed to scenario-qa rather than repaired here: they belong to the
library adoption, not to roles and handoffs.
Bug: `knw-1787866179985247848` — `bug-inbox/regression/web-console-ui-overlay-tests-fail-against-current`.

**Module 06's stale statuses.** `requirements/06-new-terminal-launcher-with-configurable/module.json`
recorded several launcher validations as `"status": "failing"`. Those tests
**passed** before this plan's work (`pnpm test -- terminal-launcher shortcuts`
→ 37 passed). The status field was stale, not a live signal.

## 3. Go test baseline

`cd api && go test ./...` passes, but the root `web-console` package carries a
**pre-existing family of parallel-scheduling flakes**: a different session/PTY
test fails on roughly every other full run and every one of them passes in
isolation. Observed across five consecutive full runs:

| Run | Failure |
|---|---|
| 1 | `TestSubscribe_SnapshotIsSelfContained` |
| 2 | *(clean)* |
| 3 | `TestSession_OfflineSnapshotIncludesPriorOutput` |
| 4 | `TestStandardBackend_AnswersClaudeStartupProbeSequence` |
| 5 | *(clean)* |

Each passes at `-count=5` on its own. The rotating identity is the tell: these
are timing-sensitive under parallel scheduling, not a defect in any one test.
Unrelated to this work, and filed to scenario-qa rather than repaired here.
Bug: `knw-1787866194218231030` — `bug-inbox/code-defect/web-console-go-session-tests-flake-under-parallel`.

`golangci-lint run` reported **10 pre-existing issues** (unused functions and
two gosimple suggestions) across files this plan does not touch. It did not
exit zero before this work and does not now; this plan added zero new issues.

## 4. After this plan

| Check | Result |
|---|---|
| `go test ./...` (api) | passes; zero new failures |
| `go test ./...` (cli) | passes |
| `golangci-lint run` | 10 pre-existing issues; **zero** in files this plan added or changed |
| `pnpm type-check` | exits zero |
| `pnpm strings:check` | exits zero |
| `pnpm test` | 14 pre-existing failures above; **zero** new |
| Three generalization greps | all return no match |

### Tests this plan added

| File | Tests |
|---|---|
| `api/internal/workspace/store_roles_test.go` | 8 (×2 stores) |
| `api/internal/grouptemplates/store_test.go` | 4 (×2 stores) |
| `api/internal/handoffrules/store_test.go` | 4 (×2 stores) |
| `api/handlers/workspace/connect_handler_test.go` | +2 |
| `api/handlers/grouptemplates/connect_handler_test.go` | 3 |
| `api/handlers/handoffrules/connect_handler_test.go` | 2 |
| `ui/src/lib/handoff.test.ts` | 11 |
| `ui/src/lib/captureRules.test.ts` | 19 |
| `ui/src/lib/workspaceNavigation.roles.test.ts` | 10 |
| `ui/src/stores/useWorkspaceStore-roles.test.ts` | 13 |
| `ui/src/__tests__/handoff-send.test.ts` | 15 |
| `ui/src/__tests__/handoff-composer.test.tsx` | 12 |
| `ui/src/__tests__/handoff-suggestions.test.tsx` | 4 |
| `ui/src/__tests__/group-auto-close.test.ts` | 15 |
| `ui/src/__tests__/group-templates.test.tsx` | 11 |
| `ui/src/__tests__/machine-picker.test.tsx` | 9 |
| `ui/src/components/launcher/agentGrid.test.ts` | 9 |
| `ui/src/__tests__/terminal-launcher.test.tsx` | +11 |
| `ui/src/__tests__/manage-groups-drawer.test.tsx` | rewritten, 15 |
| BAS | 2 new cases |

## 5. The operator's workflow, measured

The friction that motivated this plan, counted as operator actions.

**Before — nine steps:**

1. Open the new-session dialog.
2. Pick the planning agent.
3. *(plan)*
4. Copy the plan path by hand.
5. Open the dialog again.
6. Pick the implementing agent.
7. Create a group; name it; choose its colour.
8. Assign both sessions to it.
9. Paste the path plus an instruction. Later: clean up the dead group.

**After — three steps:**

1. Open the launcher, pick `Plan → Implement`, type the task name. *(One trip:
   the group, its roles, and the planning session all exist; the implementer
   waits and costs nothing.)*
2. *(plan)* Open the plan from the Messages view and press **Hand off**. *(The
   implementer starts and receives `Implement the plan at <path>`.)*
3. Close both sessions. *(The group closes itself, with an undo.)*

No clipboard. No leftover group.

> **Not yet measured live.** The step count above is derived from the shipped
> control flow, not from a timed session against a running scenario. The manual
> confirmation the plan asks for is outstanding; see the closeout note in the
> work record.
