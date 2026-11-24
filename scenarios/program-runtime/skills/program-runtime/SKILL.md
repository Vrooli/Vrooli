---
name: "program-runtime"
description: "Use for high-arity work: fan out independent calls across scenarios, replace recurring tool-call loops with governed programs, return bounded results, or make a bounded AI decision over retrieved candidates."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["skill","program","python","session","binding","typed inference","governance","runtime","high arity","multi-scenario","cross-scenario","fan-out","bounded results","tool-call compression","library","shape","nomination","delegation","bounded AI decision","subprogram"]
  icon: "terminal"
  status: "active"
  revision: 20
  createdAt: "2026-08-06T00:00:00Z"
  updatedAt: "2026-11-24T02:51:00Z"
  requires:
    scenarios: ["program-runtime", "prompt-manager", "vrooli-memory", "ai-gateway", "agent-manager", "search-hub"]
    commands: ["program-runtime bindings", "program-runtime sessions", "program-runtime programs", "program-runtime library", "program-runtime space", "vrooli-memory recall", "vrooli-memory journal", "prompt-manager skill read", "vrooli scenario"]
  learning:
    scope: "program-runtime-usage"
    capture: "every attempt"
  origin:
    kind: "authored"
---
## Tools focus: Program Runtime

Use `program-runtime` for high-arity work: fan out independent operations across scenario owners, reduce large results to bounded evidence, or make a bounded AI decision over candidates an owning scenario retrieved. For repeated task-time work, use `program-discovery` to find the owner skill in Search Hub and an exact Program Runtime contract; keep one-off operations on the owner CLI. Package uncovered recurring sequences as governed programs. Model calls go through AI Gateway; check the usage receipt before assuming a route is cheap. Reuse promoted and scenario-owned programs when their bounded contract fits, preserving their statuses and errors. This skill holds the judgment `program-runtime <group> help` does not print. Before authoring or editing a program read `path:scenarios/program-runtime/docs/guides/program-construction.md` (rules) and, for a scenario-owned program, `program-contracts.md` (contract and envelope standard) in the same directory.

Required reading before authoring or editing a program:
- `path:scenarios/program-runtime/docs/guides/program-construction.md`
- For a scenario-owned program, also read `path:scenarios/program-runtime/docs/guides/program-contracts.md`.

**Scope.** In: choosing between a CLI step, a runtime verb, a declared contract and a new program; sessions, grants, provenance, spend ceilings; envelopes and failure shapes; recurring shapes. Out: direct process execution; ambient shell, model or agent access; scenario dependencies and generated approvals; regulating program-runtime itself (`prompt-manager skill read program-runtime-improve`).

### Before acting

- For a discovered contract, `program-runtime library run program-runtime.prepare-operation --input name=<scenario>.<program>` reads its inputs and owning skill together (up to 40,000 UTF-8 bytes); a partial result means read the indicated source before execution. Preparation performs no domain action.
- `vrooli-memory recall wake --scope program-runtime-usage`, apply a pinned note; for one binding or task `recall recall "<id or task>" --scope program-runtime-usage --limit 5`.
- A temporary program over an operator-authorized directory outside the repo: `sessions create --sandbox-workspace <absolute-directory> --json`, then `programs submit --session-id <id> --source-file <path> --provenance agent`; reclaim after. The directory is the working directory, not a security sandbox (reads are unconfined), so limit the program to authorized paths and track effects.

### Which route fits

For a flagged scenario, use `program-runtime library run program-runtime.scenario-triage --input scenario=<slug>` for bounded log classification; it retains only fixed category markers, never log lines. If exact log text is needed, inspect it through the owner CLI without copying it into evidence. For an exact status/health read, use `program-runtime library run program-runtime.supervision-sweep --input scenarios=<slug>[,<slug>...]` and read `signals.scenarios` in request order; `absent` means no fleet row and `unknown` with `source_truncated` means incomplete. The returned named rows are byte-bounded; `partial`/`truncated` means read fewer slugs. For a repeated full-fleet status read, use the same program with `--input all_scenarios=true --input page_offset=0 --input page_size=20`; append each page's `signals.scenarios`, advance to `next_offset`, and stop only at null. After start/restart, use `program-runtime library run program-runtime.scenario-wait --input scenario=<slug> --input timeout_seconds=<1..90>`; it reads status once, attaches to wait once, and returns a typed verdict and exit code. Choose the ceiling from the expected duration plus 75% buffer. Exit 124 means the wait detached and the start continues; reconcile state before one later reattach. Use `program-runtime library run program-runtime.scenario-deploy-report --input scenario=<slug>` for a bounded pre-restart report containing only provenance/status, capped repo-relative changed paths, build-key names and environment-switch names; it never returns build or environment values. `vrooli/agent/list` remains owner CLI because its inventory exposes session, process, scope, owner and resource details; use `--session-id` for one known lease and do not retain broad output. Treat `partial` or `source_truncated` as incomplete. For other multi-step work, confirm a governed binding exists for every operation (`program-runtime bindings list`; `bindings unbound` gives the closed-set reason); repair in scope or retain the gap (`report-bug`). Then:

| Work | Route |
|---|---|
| A bounded decision over retrieved candidates | Retrieve through the owning scenario, filter deterministically first, use a declared AI Gateway decision only for ambiguity; reuse a matching owner program; check child status and usage before accepting a result or claiming a cost |
| A label per text, closed set | `ai-gateway.classify-batch` (1-32 texts, caller's instruction) |
| Join, sort or aggregate first | Copy the shape of `program-runtime.handle-shaping` |
| Agent Manager workflows (bounded batch, not a durable supervisor) | One or two runs `delegated-run`; 3-8 plus an outcome label `watch-set`; `workflow_rejected` means the owner does not declare that key (fix the request, never retry it). Sustained supervision is Agent Manager's (`docs/guides/supervision-loops.md`) |
| Why programs keep failing | `failure-triage`; branch on `signals.top[0]`, inspect `signals.evidence_by_shape[shape]`; `partial` means use only the returned rows |

Read `signals.attention` and `signals.unavailable` (reads not performed: unknown, never zero) on sweeps; `fleet-fanout` unavailable means the scenario is down (journal, never estimate). A contract runs as `program-runtime library run <scenario>.<name> --input k=v`: the runner validates inputs, owns the session, waits once, prints the envelope and reclaims the session. Inputs are JSON-decoded when valid, so quote numeric, boolean or null strings (`--input 'window="7"'`). Branch only on envelope `status` and `errors[0].class`.

**Portfolio and usage.** `portfolio-audit` then `usage-triage` grade which programs need attention (branch on `status`, `fought_it`, `abandoned`, `wrong_program`; never spend inference or lower a band to improve a score); `programs portfolio --window-days N` counts use; `program-runtime.local-work-snapshot` is one bounded read-only observation of local work across Agent Manager, Prompt Manager teams, Swarm and retained research (`docs/guides/local-work-bridge.md`; normally `partial`).

### Runtime verbs and learning verbs

The Python verbs (`discover`, `recall`, `validate`, `capture`, `guide`, `lib.<name>()`) and when a binding beats each are in the construction guide's "Runtime verbs". A `discover` null verdict (empty `binding_id`, `unavailable=False`) means stop and read its reason; `unavailable=True` means the lookup failed, not that no capability exists: retry once after correcting an input, continue independent work from canonical sources, and never infer a binding or call an unregistered endpoint.

The ten learning verbs (`learn.task`, `step`, `recall`, `choose`, `note`, `outcome`, `infer`, `act`, `delegate`, `feedback`) are specified in "Adaptive authoring and delayed feedback". Use a stable task key; `learn.act` needs an independent `verify` and `verifier_revision` and takes `key="<owner>/<subject>/v1"` when correct code does not depend on the task; promotion is the separate `program-runtime.fragment-promote`. If Memory is unavailable, branch on its unavailable receipt; never fabricate a learning result. **Every learning program emits `signals.learning.feedback_ref`** with a declared `learning` output signal (a kernel test enforces it). When an output proves wrong, send a correction with that reference instead of re-running:

```bash
program-runtime library run program-runtime.learning-feedback --input 'feedback_ref=<signals.learning.feedback_ref>' --input disposition=contradicted --input dimension=usefulness --input 'evidence=["operator:the extracted list was last week"]'
```

Dimensions: `usefulness` (missed the need), `context` (requirements moved), `execution` or `verification` (confirmed code or postcondition defect; these two disqualify the exact implementation).

### Commands where judgment applies

| Command | Use it when | Not when |
|---|---|---|
| `bindings condition` | A program returned `unavailable` for a scenario `vrooli scenario status` says is running; the row says dormant, stale or degraded | You only need to know a binding exists |
| `bindings sweep` | You need the eligible set before a fleet read (keep `--dry-run`) | `--execute` on a shared host without an operator asking |
| `programs mine-refusals`, `mine-unresolved` | Programs end `refused` (`not_run_eligible`: no grant), or `unresolved_name` recurs (a near-miss is a typo; a real capability with no binding is W1 for its owner) | You know the binding id (`bindings describe`) |
| `library promote` | A succeeded program a second agent will call; `--reason` names the binding set it deduplicates | It printed rows instead of counts, or only an operator-provenance fixture succeeded |
| `discovery eval --suite evals/discovery.primary.json --mode judged`, `authoring eval` | An improve cycle reads the discovery or authoring floor | Inside a program (`no_governed_binding`; `kernel_invoke_budget` over the 100 s invoke budget): run from a shell and read `met`, `floor`, `missed` |

### Sessions, provenance and bounds

- One session per program call (variables persist across submissions, so a stale `inputs` is reused). A synchronous submission is bounded at 120 s; for longer work `programs submit ... --async --wait-timeout 300s` or `programs wait <program-id> --timeout 300s` once, never poll.
- `sessions create --inference-ceiling-micros N --delegation-ceiling-micros N` sets the spend ceilings; crossing one ends the program (`inference_spend_exceeded`, `delegated_run_spend_exceeded`), stopped not retried. Decide it from the contract's `budget`; raising it means a new session and a journal line why.
- A succeeded two-binding shape recurring across two non-test sessions is nominated: `shapes list --uncovered --min-occurrences 3`, then file the W1 obligation against the dominant scenario.
- `--provenance`: `operator` only for a direct operator instruction in this session; `agent` for heartbeats, goal-loop cycles and anything a human did not type; `test` for `programs`-phase fixtures; `replay` for diff replays (never promoted). Never relabel a run to quiet `programs mine`; never use `agent` on a destructive binding without a grant.
- Keep program output bounded: reduce in the kernel (`count()`, `agg(field, "sum")`, `group_by(key)`, `filter` and `join`), `head(n)` with `n` in the contract's `materialize_limit` (`materialize(limit)` only when rows are required), `meta()` for non-row fields, `raw()` only once in a scratch session, never inside an envelope (guide "The core discipline").

### Authoring a program that will recur

`library search "<intent>"` first (a promoted hit ends it); compose a matching scenario-owned workflow with `lib.<scenario>.<name>(...)`; otherwise copy the shape of `scenarios/program-runtime/.vrooli/program-runtime/setpoint-read.py` and follow the construction guide (`program.*` helpers are kernel-bound, and a locally defined `classify_transport`, `fail`, `guarded` or driver loop is reported as `programs.duplicated_helper`). Probe each binding's row keys and arguments in a scratch session first (`describe("<id>")`, `bindings describe <id> --json`). Write the contract beside the source, validated against `scenarios/program-runtime/schemas/program-contract.schema.json`; `programs submit --explain` in a fresh session until `diagnostics` is null; run every fixture in its own session (a stopped dependency is `unavailable`, not skipped); name the program in the owning usage skill as `run <scenario>.<name>` and in its `.vrooli/service.json` `skills` block; wrap a read that can fail alone so the row is `unavailable` and the envelope `partial`.

### Debug order and settings

Read `program-runtime programs get <id>` `failure_shape` (closed vocabulary; guide "Reading a failure") before `failure_detail`. `unresolved_name`, `unknown_field`: fix from the listed candidates, resubmit in a new session. `ambiguous_response`: choose `rows="<field>"` first; reconcile owner state before retrying. `effect_uncertain`, `deadline_exceeded`, `output_schema_mismatch`, `output_truncated`: effects may already have happened, so reconcile with the owning operation before any retry and journal program id, wall time and owner evidence. `unreachable_scenario`, `bridge_transport`: `vrooli scenario status <scenario>`, `bindings condition --scenario <scenario>`; never start a scenario inside a program. `refused_*`: `programs mine-refusals`; `sessions grant <id> --grants <exact grant>` only after an operator confirms (destructive calls also need confirmation in the request), journaling grant and confirmer. `inference_spend_exceeded`: new session with a higher `--inference-ceiling-micros` only if `budget.inference_calls` justifies it; journal both ceilings. A CLI answering about a stale port after an API restart is a discovery disagreement: inspect `vrooli scenario status program-runtime`, repair it in the shared discovery owner, and export `PROGRAM_RUNTIME_API_PORT` meanwhile. Kernel row keys are camelCase, CLI keys snake_case.

### Governance and safety

- Use only bindings `bindings list` returns; never call a method `bindings unbound` classifies `no_manifest`, `local_binding`, `omitted_rpc` or `external_tool_only`. Destructive operations are denied until the session has the exact grant and the request includes confirmation. `ai.*` and `agent.*` exist only inside a live session with the governed bridge. No retry or polling loop inside a program; retry is the caller's decision, informed by the envelope.
- A program may change only session, program, telemetry and declared library state owned by program-runtime, and files inside a declared sandbox workspace. `bindings sweep --execute` and `library promote` are the two writes this skill reaches, each with a stated reason. Edit the scenario file, never a library row; never mutate shared bindings, generated proto, dependency approvals or other scenarios.

### After acting, always

Leave a traceable session id, program id, provenance and envelope status, with one `vrooli-memory journal note --scope program-runtime-usage --kind task-record --trigger "<task>" --approach "<route, program or command>" --evidence "<session id, program id, envelope status, errors[0].class>" --outcome "<ok | partial: <gap> | unavailable: <scenario> | refused: <grant> | failed: <class>>" "<one line>"` per attempt (`--kind binding-note` for one binding).

### Troubleshooting & Edge Cases

| Symptom | Fix |
|---|---|
| A contract program ignores its inputs, or session not found | Session reused or reclaimed: use a fresh session per call, `inputs = {...}` prepended to the copied source |
| `programs get` denies a failure-triage sample | It belongs to another verified caller: keep the aggregate count as evidence, inspect another authorized sample or a source cause, never retry under another identity |
