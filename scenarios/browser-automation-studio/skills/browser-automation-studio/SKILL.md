---
name: "browser-automation-studio"
description: "Decide how to run a browser task with Browser Automation Studio: a single-page capture Action, an existing typed workflow through smoke-flow, a draft through author-flow, an intent through navigate-intent, or the whole task through do-task. Recall before and journal after every attempt in the bas-usage scope."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["skill", "browser", "automation", "bas", "workflow", "screenshot", "capture", "e2e", "ui", "selector", "vision-navigation", "smoke test", "console logs", "network"]
  icon: "play"
  status: "active"
  revision: 48
  createdAt: "2026-01-25T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: ["browser-automation-studio", "program-runtime", "vrooli-memory", "prompt-manager", "workflow-health"]
    commands: ["browser-automation-studio capture", "browser-automation-studio workflows", "browser-automation-studio executions", "browser-automation-studio session-profiles", "browser-automation-studio observability", "browser-automation-studio vision-navigation", "browser-automation-studio schema", "browser-automation-studio status", "program-runtime sessions", "program-runtime programs submit", "vrooli-memory recall", "vrooli-memory journal note", "vrooli-memory facets", "prompt-manager action", "prompt-manager skill read"]
  learning:
    scope: "bas-usage"
    capture: "every attempt"
  origin:
    kind: "authored"
---
## Tools focus: Browser Automation Studio

Use Browser Automation Studio (BAS) to observe or drive a real browser against a running scenario or a site: one-page captures, persisted typed workflows, ad hoc drafts, and AI navigation by intent. This skill is the judgment that `--help` does not carry: which leaf to take, how to read a program envelope, what to remember. Command syntax lives in `browser-automation-studio <group> help`.

**In scope:** choosing and running the leaf for a browser task; reading and journaling the outcome. **Out of scope:** e2e strategy, `bas/` asset layout, selector registries, and validation cases (`prompt-manager skill read e2e-testing`); regulating BAS itself (`browser-automation-studio-improve`); memory mechanics (`prompt-manager skill read vrooli-memory`).

Required reading: `prompt-manager skill read vrooli-memory` (scopes, pins, rules), `prompt-manager skill read program-runtime` (how a program step runs).

### Running a program step

A leaf that reads `run browser-automation-studio.<program>` means: prepend `inputs = {...}` to `scenarios/browser-automation-studio/.vrooli/program-runtime/<program>.py` in a scratch file, then

```
program-runtime sessions create --name <task> --json          # fresh session per run: inputs persist inside a session
program-runtime programs submit --session-id <id> --source-file <scratch.py> --provenance operator --json
program-runtime sessions delete <id> --reason "<task> done"
```

Read the envelope from `.program.stdout`. Branch on `status` first, then `errors[0].class`. The contract beside the program (`<program>.json`) is the vocabulary; this skill only names its values.

### Before acting (recall)

1. `vrooli-memory recall wake --scope bas-usage` for the ambient set. [S1]
2. `vrooli-memory recall recall "<site or task>" --scope bas-usage --limit 5` for the target. [S1]
3. A `site-note` that names a wait, selector, profile, or viewport → apply that in-use setting (table below) before choosing a leaf.

### The tree

```
What does the task need?
├─ One page, no interaction (does it load, what does it log, what does it request)
│   ├─ desktop screenshot ............................ Action bas.screenshot            [S2]
│   ├─ mobile viewport screenshot .................... Action bas.screenshot.mobile     [S2]
│   ├─ console output only ........................... Action bas.console-logs          [S2]
│   ├─ network requests only ......................... Action bas.network               [S2]
│   ├─ screenshot + console + network in one load .... Action bas.audit                 [S2]
│   ├─ a readiness condition or viewport the Actions do not fix
│   │     browser-automation-studio capture --url <url> --capture <csv> --wait-for <css|networkidle|ms> --dimensions <preset> [S1]
│   └─ is BAS itself up ................................ Action bas.status                [S2]
├─ Interaction, or more than one page
│   ├─ You know the workflow id ....................... run browser-automation-studio.smoke-flow      [S3]
│   ├─ You do not ..................................... run browser-automation-studio.find-flows      [S3]
│   │     ├─ candidates[0].fit is strong and runnable_by_id ... smoke-flow with that id           [S3]
│   │     ├─ the candidate is a bas/ asset (runnable_by_id false) ... it is a validation case: e2e-testing [S0]
│   │     ├─ no candidate, you can write the flow ....... author-flow                              [S3]
│   │     └─ no candidate, a live browser session exists . navigate-intent                          [S3]
│   ├─ You hold a V2 flow object not yet persisted ..... run browser-automation-studio.author-flow     [S3]
│   ├─ You want the whole task done with memory ........ run browser-automation-studio.do-task         [S4]
│   └─ Inline steps for one quick check ............... browser-automation-studio workflows execute-adhoc --flow-file <f> --wait [S1]
└─ Requirements evidence in scenarios/<target>/bas/ .... e2e-testing, not this tree                [S0]
```

Actions are run with `prompt-manager action run <id>`; the six ids above are listed by `prompt-manager action list`.

### Branching on the envelope

**smoke-flow** (`workflow_id`; optional `session_profile`, `parameters` as an object matching the workflow's `ExecutionParameters`, `version`)

| status | errors[0].class | Next leaf |
|---|---|---|
| ok | — | Done. Capture a `workflow-verdict` pass. |
| failed | workflow_not_found | find-flows. |
| failed | profile_not_found | `browser-automation-studio session-profiles list`; create or refresh (settings table); rerun once. [S1] |
| failed | selector_not_found | Debug order steps 1–3; capture a `site-note` (facet selector). A `@selector/` reference → e2e-testing. |
| failed | timeout | Capture a `site-note` (facet wait). Raise the node's timeout or add a wait node in the flow → author-flow. |
| failed | auth_required | Session profile row in the settings table; rerun once with `parameters` carrying the credentials the flow declares. |
| failed | step_failed | Debug order. |
| partial | timeout (`outcome: still_running`) | `browser-automation-studio executions get <execution_id>` once by hand. Do not loop. [S1] |
| unavailable | scenario_unreachable | `vrooli scenario status browser-automation-studio`; stop until `running`. [S1] |
| refused | no_grant, not_run_eligible | Request the grant through the session path; stop. |

**find-flows** (`task`; optional `scenario`, `k`)

| status | errors[0].class or signal | Next leaf |
|---|---|---|
| ok or partial | `candidates[0].fit` strong, `runnable_by_id` true | smoke-flow with `candidates[0].id`. |
| ok or partial | `candidates[0].runnable_by_id` false | A `bas/` validation asset: e2e-testing. |
| ok or partial | `candidates` empty | author-flow, or navigate-intent with a session. |
| partial | search_unavailable | search-hub was unreachable; the candidates come from `workflows list` alone. Rerun when search-hub is running if the list is empty. |

**author-flow** (`flow` object; optional `name`, `folder`)

| status | errors[0].class | Next leaf |
|---|---|---|
| partial | no_governed_binding, `persistable: true` | Write the flow to a file; run `signals.persist_command` (`workflows create ... --folder-path candidates`). [S1] |
| failed | validation_failed | Fix the draft against `browser-automation-studio schema workflow --nodes <types>`; rerun. [S1] |
| failed | selector_not_found, timeout, auth_required, step_failed | As smoke-flow. |
| failed | no_governed_binding (on `ai_prompt`) | AI generation has no governed path from a program; write the flow yourself. This is permanent until BAS exposes the binding, so do not retry. |

**navigate-intent** (`session`, `prompt`, `model`; optional `max_steps` ≤ 25, `navigator`). `model` is required and is never defaulted: the binding carries no proto default, and the program fails in `validate` with `model_required` when you omit it. Read the model from `browser-automation-studio vision-navigation list-navigators` and pass it.

| status | errors[0].class or signal | Next leaf |
|---|---|---|
| ok | `outcome: reached` | Done. Capture a `task-record`. |
| partial | `outcome: in_progress` or `human_pause` | `browser-automation-studio vision-navigation status <navigation_id>` once by hand; `vision-navigation resume <id>` when it says `awaiting_human`. [S1] |
| failed | budget_exhausted | Narrow the prompt, or raise `max_steps` once (≤ 25). |
| failed | session_not_found | `browser-automation-studio observability sessions` for a live session id. [S1] |
| failed | navigation_failed | Debug order step 5 on the start URL; capture a `site-note` (facet failure). |
| refused | credits_required | `browser-automation-studio vision-navigation list-navigators`; stop. [S1] |
| failed | model_required | You omitted `model`. Read one from `vision-navigation list-navigators` and pass it; never hardcode a slug in a skill or program. |

**do-task** (`task`; optional `scenario`, `k`, `workflow_ids`, `session`, `recurrence_threshold`, `model`, resolved as in navigate-intent when the fallback runs). The contract budget is async: submit with `--async --wait-timeout 900s`.

| status | errors[0].class or signal | Next leaf |
|---|---|---|
| ok | — | Done. Memory already written by the program; do not double-write. |
| partial | navigation_pending | Status read as in navigate-intent. |
| partial | memory_unavailable | Done; capture the `task-record` by hand. |
| any | `prior_attempts` ≥ 2 | Read the recalled task-records (`vrooli-memory recall recall "<task>" --scope bas-usage`) before choosing again. [S1] |
| failed | no_candidates | author-flow, or navigate-intent with a session. |
| failed | selector_not_found, timeout, auth_required, step_failed | As smoke-flow, for `attempts[-1]`. |
| any | `author_recommended: true` | author-flow with a draft of the navigated path, then persist. |

### After acting, always (capture)

One note per attempt, unless do-task ran (it writes its own):

```
vrooli-memory journal note "task-record: <task> | <leaf> | <status>/<class>" --scope bas-usage --kind task-record \
  --trigger "<task>" --approach "<leaf>" --evidence "execution:<id>" --outcome "<status>/<class>"
```

Kinds (`--kind`): `task-record` (every attempt), `site-note` (a fact about a site: wait, selector, profile, dimensions, failure), `workflow-verdict` (`<workflow id> passed|failed <class>`).

Facets on this host: the `bas-usage` scope carries the six default facets (`vrooli-memory facets list --scope bas-usage`), so classification and corrections use those ids: a site-note is an `environment-fact`, a task-record an `episode`, a workflow-verdict an `entity-record`, a repeated failure a `gotcha`, and pinned advice a `standing-rule`. The BAS vocabulary (`bas-site`, `bas-flow`, `bas-selector`, `bas-wait`, `bas-profile`, `bas-dimensions`, `bas-failure`) needs a scope created with `--facets-json` because facets are fixed at creation; that is a later `scope-bootstrap` run, after which the declaration in `service.json` switches to the new scope.

Curation leaves, at the branch where the evidence appears: a site-note confirmed on its third attempt → `vrooli-memory facets pin <entry-id> --scope bas-usage` [S1]; advice that failed on retry → `vrooli-memory facets supersede <entry-id> --scope bas-usage --replacement-entry-id <new>` [S1]; a pattern repeated across sites → propose a rule with `run vrooli-memory.scope-bootstrap` [S3].

### In-use settings

| Symptom | Move | Journal |
|---|---|---|
| Login step fails, cookies expired, `auth_required` | `browser-automation-studio session-profiles list`; `session-profiles update <id> --browser-profile '<json>'` to change fingerprint settings; re-run the login flow to refresh stored state (e2e-testing) | `site-note`, facet profile |
| Page not ready at capture | `browser-automation-studio capture ... --wait-for networkidle`, `--wait-for '<css>'`, or `--wait-for <ms>` | `site-note`, facet wait |
| Layout differs by viewport | `browser-automation-studio capture ... --dimensions mobile` (or `tablet`, `desktop`, `--width/--height`) | `site-note`, facet dimensions |
| Failed runs crowd the artifact store | `browser-automation-studio executions retention-preview --max-age-days <n> --keep-latest <m>`, then `executions retention-run` with the same flags and `--confirm` | `task-record` |
| A run needs a trace or HAR for debugging | `browser-automation-studio workflows execute <id> --wait --requires-trace --requires-har` | `task-record` |
| Driver looks unhealthy | `browser-automation-studio observability status`; `observability sessions` | — |

No retention knob is exposed through `observability config-set`: `config-get` showed zero runtime overrides on 2026-09-02, and that surface configures the playwright-driver, not execution retention. Use the retention commands above.

### Debug order

1. `browser-automation-studio executions get <execution-id>` — status and the error string (`step N failed: ...`). [S1]
2. `browser-automation-studio executions timeline <execution-id>` — the failed entry and its action. [S1]
3. `browser-automation-studio executions screenshots <execution-id>` — the last frame; a failed run may retain none (`docs/PROBLEMS.md` 2026-07-27). [S1]
4. `browser-automation-studio executions recorded-traces <execution-id>` when the run requested a trace. [S1]
5. Re-observe the failing page: Action `bas.audit` on its URL. [S2]
6. Driver-side suspicion only: `browser-automation-studio drills list`, then `drills run --name <DRILL>` (development only). [S1]

### Safety

- Cases without `metadata.safety` are not run by an agent. Add the label or leave the case to a human.
- Never point a workflow at a production URL from a `test`-provenance session. Use a scenario target (`scenario=<name>,path=/`) or a site you own.
- Respect execution labels. `observer` workflows only navigate, screenshot, assert, extract, and wait; a `mutating` workflow needs `requires_confirmation` and `routed_isolation` and is refused by workflow-health otherwise (e2e-testing).
- Run workflows only against scenarios and sites you own or are permitted to test. Vision navigation spends credits under the navigator's policy: `browser-automation-studio vision-navigation list-navigators` before a first run.
- `executions retention-run` deletes rows and artifact directories; run `retention-preview` first.
- Never start a scenario from a program or a skill step; the lifecycle owns that.

### Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Every program returns `unavailable`/`scenario_unreachable` | BAS stopped or restarting; it restarts often under test | `vrooli scenario status browser-automation-studio` | Wait for `running`; rerun once |
| `no running runtime ports` mid-program | BAS restarted during the run | same | Rerun; journal nothing |
| `no proto field matches "flow_file"` | `--flow-file` is CLI-local | `program-runtime bindings describe browser-automation-studio/workflows/validate --json` | Pass the flow as an object (`flow` input); author-flow already does |
| `workflows execute` says multiple workflows match | Name-based execution | `browser-automation-studio workflows list` | Use the UUID |
| A timeout is classified `selector_not_found` | Playwright phrases selector waits as timeouts; the classifier prefers the selector class | timeline entry's action | Read the step action; treat as wait when the element exists |
| `recall wake` says scope not registered | `bas-usage` missing on this host | `vrooli-memory scopes list` | `vrooli-memory scopes create bas-usage --label "BAS usage learnings" --wake-budget 48 --max-entry-lines 2 --facets-json '[{"id":"bas-site","label":"Site"},{"id":"bas-flow","label":"Flow"},{"id":"bas-selector","label":"Selector"},{"id":"bas-wait","label":"Wait"},{"id":"bas-profile","label":"Profile"},{"id":"bas-dimensions","label":"Dimensions"},{"id":"bas-failure","label":"Failure"}]'` (facet ids are unique across scopes and fixed at creation; the scope on this host was created with the six defaults instead, see §After acting) [S1] |
| Screenshots list is empty for a failed run | Evidence gap (PROBLEMS.md 2026-07-27) | `executions screenshots <id>` | Rerun with `--requires-trace`; the improve skill tracks `failed-run-evidence` |
| `uxmetrics` returns an entitlement error | Pro-tier gate | `browser-automation-studio entitlement status` | Not a run failure; skip the metric |
| vision-navigation `model is required` | The binding has no proto default and the program does not invent one | `browser-automation-studio vision-navigation list-navigators` | Pass `model` explicitly on every call |
| A timeline read fails with `no determinable primary response field` | `executions/timeline` carries two repeated fields (`entries`, `logs`), so the projection is ambiguous | `program-runtime bindings describe browser-automation-studio/executions/timeline` | Pass `rows="entries"`: `executions.timeline(execution_id=<id>, rows="entries")` returns the step rows (`stepIndex`, `action`, `durationMs`, `context`) |
