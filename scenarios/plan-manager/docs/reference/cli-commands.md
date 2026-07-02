# CLI Commands — Plan Manager

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/plan-manager`, and rebuilt automatically when its
sources change (cli-core's stale-detection rebuilds before any command
that touches the API).

## Source of truth: `cli/manifest.json`

The CLI's command surface (groups, commands, positionals, flags,
RPC bindings, governance metadata) is declared in
[`cli/manifest.json`](../../cli/manifest.json) and validated against
[`.vrooli/schemas/cli-manifest.schema.json`](../../../../.vrooli/schemas/cli-manifest.schema.json)
(schema id `cli-manifest/v1`). The manifest is loaded at startup by
`cliapp.LoadFromManifest`, which:

- builds each domain's `SubcommandGroup` from its manifest group
- wires each command's `binding.method` (e.g.
  `<Domain>Service.List<Entity>`) to a handler registered in the
  domain's `register.go` bindings map
- fails loudly on missing handlers, dead handlers, or unknown groups

Per-domain tests use `cliapp.RequireProtoServiceCoverage` to assert
that every RPC on the bound proto service either has a manifest command
binding or appears in the manifest's `omitted[]` list with a reason —
adding a new RPC without exposing it as a CLI command (or explicitly
omitting it) fails the test.

The manifest's `governance` block (`effect`, `run_eligible`,
`permissions`, `requires_confirmation`) is consumed by prompt-manager
to derive action certainty automatically; scenarios that adopt the
manifest don't need hand-classified action-safety lists.

`binding.kind` is currently `connect-rpc` only. REST-exception
commands (for example, a multipart file-upload command whose request
body carries opaque bytes) are appended to the loaded group outside the
manifest path in the domain's `register.go` and documented in the
manifest's `omitted[]` array.

For environment-variable precedence and CLI config-file shape, see
[`configuration.md`](configuration.md).

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them
in scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start plan-manager` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

**Placement:** global flags must come **before** the subcommand, e.g.
`plan-manager --auto-start author start --title …` (not
`plan-manager author start --auto-start …`). A global flag placed after the
subcommand now yields a clear placement hint instead of a bare "unknown option".

## Built-in commands (auto-provided by `cli-core`)

### `plan-manager status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
plan-manager status
plan-manager status --json
```

### `plan-manager configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
plan-manager configure api_base http://localhost:15001/api/v1
plan-manager configure token <token>
```

Read values back without an argument:

```bash
plan-manager configure api_base
```

## Scenario commands — `<domain>`

Each product domain exposes its commands as a subcommand group
(`plan-manager <domain> <verb>`). Every command calls a single API
endpoint and renders the result through one of the three output
contracts below. Document your domain's commands here as you build
them, one row/section per command, mirroring the endpoints they call
in [`api-endpoints.md`](api-endpoints.md).

The scaffold ships one fully worked CRUD command group as a copyable
reference (see the fenced example below); `vrooli scenario detemplate
<scenario>` removes it once your real domains are green.

## Primary Agent Loops

Small agents should prefer the continue-loop commands and use the lower-level
commands only for the specific action returned by the API-owned `GuidedStep`.

| Command | RPC | Purpose |
|---|---|---|
| `plan-manager author continue <session>` | `AuthoringService.ContinueAuthoring` | Returns one recommended next authoring action across section, phase, validation-review, and finalize states. |
| `plan-manager exec continue <plan-or-execution>` | `ExecutionService.ContinueExecution` | Resumes or starts execution and returns one recommended next runner action without advancing the phase pointer. |

`plan-manager exec transition <execution> <phase> --status done` is guarded by
the execution service: it requires the last stored phase validation to be
`pass` + `fresh`, or an explicit `--validation-override-reason` for degraded or
offline completion. Prefer `exec continue` so the API recommends validation
before the done transition.

## `author` — the guided composer wizard

`author status <session>` is an alias of `author preview`. Global flags
(`--auto-start`, `--api-base`, `--instance`, `--dry-run`) go **before** the
subcommand (`plan-manager --auto-start author continue <s>`); a misplaced one
after the subcommand gets a placement hint at the point of error.

The authoring response contract is **focused for small/local models**: normal
mutations (`section-submit`, `phase-submit`, `phase-add`, `autofill`,
`context-submit`, `context-update`, `context-remove`, `context-accept`,
`context-reject`, `context-discover`, `context-apply`, `suggest-references`,
`reference-accept`, `reference-reject`, `reference-apply`) **do not echo the
full session**. With
`--json` you get a compact `AuthoringProgress` (current section/phase,
mandatory-sections + phases counts, `remaining_required_inputs[]`,
`ready_to_finalize`), an `AuthoringMutationSummary` (what changed), the single
changed object, any structure violations, and the next guided step — not the
whole `AuthoringSession`. To read the whole session graph, ask for it explicitly:

| Command | RPC | Purpose |
|---|---|---|
| `plan-manager author start --title <title> [--slug <slug>]` | `AuthoringService.StartSession` | Starts a session with a readable title-derived handle when `--slug` is omitted. UUIDs remain supported, but human guidance prefers the readable handle. |
| `plan-manager author get-session <session>` | `AuthoringService.GetSession` | **Explicit full-state read.** Returns the whole `AuthoringSession`; use it for read-after-write when you need the full graph (mutations no longer echo it). `author preview` / `plans render` return the full rendered markdown. |
| `plan-manager author context-submit <session> --kind command --label <label> --command <command> [--argv-json <json-array>]` | `AuthoringService.SubmitRelevantContextItem` | Records executable setup context. Use `--argv-json '["search-hub","query","shared drift hygiene","--type","record,doc","--json"]'` for quoting-sensitive commands; command-only fallback uses shell-compatible parsing. |
| `plan-manager author context-update <session> <item> --kind <k> [--phase --label --reason --instruction --command --argv-json --argv --target --required --repeat]` | `AuthoringService.UpdateRelevantContextItem` | Replace one accepted relevant-context item in place (by id from `author context-list`) so a bad item discovered in `author preview` is corrected **without deleting the phase/session**. Legal only before finalize. |
| `plan-manager author context-remove <session> <item> [--phase]` | `AuthoringService.RemoveRelevantContextItem` | Remove one accepted relevant-context item (by id) before finalize; removal recomputes structure violations so any resulting gate (e.g. removing the only phase context) is reported with its recovery action. |
| `plan-manager author phase-move <session> <phase> --before <phase>` / `--after <phase>` | `AuthoringService.MovePhase` | Reorders a structured phase draft without rewriting its id, title, intent, steps, context, or acceptance content. |
| `plan-manager author finalize <session> [--full] [--workspace <root>]` | `AuthoringService.Finalize` | Validates, persists, verifies read-back through the plans domain, and reports the honest persistence state: the physical SQLite store path, the workspace stamped on the plan (default: the resolved repo root), and the **computed** mirror publish result (`fresh` / a loud `write_failed` warning — never a default `unknown`). Re-running finalize on a finalized session says `Already finalized at <ts>` explicitly. |

### Batch submission — the form, not the wizard

The session is a **form with a derived cursor, not a wizard with state**: every
guided step carries a full-disclosure `checklist` (every requirement for the
touched scope with `filled`/`missing`/`violation` status), fields may be
submitted **in any order**, and one call can carry one field, a whole phase, or
the whole plan. An agent that already knows its content authors a complete
N-phase plan in ≤ 3+N mutation calls: `start` → one `submit` sections batch →
one `phase-add --set …` per phase → `finalize`.

| Command | RPC | Purpose |
|---|---|---|
| `plan-manager author submit <session> --set <section_key>=<content> --set <phase-ref>.<field>=<content> …` | `AuthoringService.SubmitFields` | Session-scope batch: repeated `--set` pairs land in ONE call under one session lock/save. Items apply independently (never all-or-nothing); each returns an accepted/rejected line naming exactly what was parsed or why it was rejected. `--set-file <key>=<path>` reads long content from a file. |
| `plan-manager author phase-submit <session> <phase> --set <field>=<content> …` | `AuthoringService.SubmitFields` | Phase-scope batch: keys are bare phase field names. The single-field `--field`/`--content` form remains for one-off edits (same apply path — no drift). |
| `plan-manager author phase-add <session> --title <t> --intent <i> --set <field>=<content> …` | `AuthoringService.AddPhase` + `SubmitFields` | Add+fill in one command: the optional `--set` pairs apply to the just-created phase in one batch. The response checklist shows all 7 phase fields with live status the moment the phase exists. |

A rejected item (unknown section/phase/field, unparsable references markup, or
acceptance duplicating validation) is **not applied** — the line says so and the
rest of the batch still lands. For an already-written plan **document**, `plans
import` remains the whole-document path; batch submission is for composing
through the authoring session.

### `context-discover` — server-side discovery execution

`plan-manager author context-discover <session> [--concepts "<c1>,<c2>"] [--complexity <minor|moderate|major|architectural>] [--refresh]`
EXECUTES the discovery probes itself and returns a curated proposal — the agent
supplies concepts and judgment; the code runs the probes, deduplicates,
curates, and formats final setup lines. Per (concept, probe) pair it runs,
concurrently and each bounded by a per-probe timeout (default 20s) with capped
fan-out. `author start` launches a best-effort title-derived prefetch; with no
`--concepts`, `context-discover` can reuse that batch immediately. Explicit
concepts or `--refresh` force a fresh run that supersedes/merges through the
normal no-resurrection path.

- `prompt-manager discover <concept> --type skill --json [--complexity <c>]` — curated skill candidates (bare-slug targets),
- `prompt-manager discover <concept> --type all --json` — executable action candidates (their `showCommand` verbatim),
- `search-hub query <concept> --type record,skill,doc --json` — prior-work records (as `swarm-manager records get` commands), docs, and skills.

Results are deduplicated by (kind, target) across the pending batch and prior
accepted/rejected history, so rediscovery does not duplicate or resurrect a
rejected target. The response shows a shortlist with handles (`c1`, `c2`, ...),
tier, score, origin, size when known, hit/corroboration count, setup line,
evidence, tags, snippets, and batch-level probe notes. Lower-ranked longlist
items are auditable with `context-list --all`, but they are not
disposition-required. A failed/slow/unparseable probe degrades
**independently** into a typed note on the batch, not a candidate — the step
always returns and the wizard never blocks on a down dependency.

### `context-apply` — one-call context disposition

`plan-manager author context-apply <session> [--batch <id>] [--take <handle[:phase]>]... [--drop <handle=reason>]... [--take-all] [--note <text>]`

Applies the latest pending curated context batch as one decision. Take only the
items whose concrete implementation improvement you can articulate; omitted
shortlist items are swept as not taken, and longlist items are implicitly swept.
Use `--drop <handle=reason>` only when a high-confidence candidate should not
land, or when recording an explicit reason is useful. `--take c1:phase-id`
accepts a candidate as phase-scoped setup; bare `--take c1` accepts it as
global setup. `--take-all` is available for trusted tiny batches, but it should
not replace review.

`context-accept` and `context-reject` remain the small-agent one-item lane and
accept either the stable candidate id or the batch handle. They share the same
internal disposition path as `context-apply`; when the last shortlist item is
handled one by one, the batch auto-closes. Prefer `context-apply` for normal
authoring because the context checkpoint is budgeted as at most two calls:
`context-discover` (or zero calls when the prefetch is sufficient) plus one
`context-apply`.

**Skill checkpoint gate v3:** finalize requires the latest context discovery
batch to be applied, or an explicit `NO_SKILL_CONTEXT: <reason>` /
`NO_CONTEXT: <reason>` skip when no relevant setup exists. Legacy sessions with
pre-batch pending candidates still satisfy the gate through the one-item lane.
No minimum skill count is imposed: the gate demands evidence of a sweep, not a
quota.

### `suggest-references` / `reference-apply` — reviewed code/document locators

`plan-manager author suggest-references <session>` queries search-hub's Answer
projection, keeps only `[CODE:]`, `[DOC:]`, and `[REQ:]` locator-shaped hits,
and returns a curated reference proposal with handles (`r1`, `r2`, ...),
shortlist/longlist tiers, scores/evidence where available, and batch metadata.
Search-hub down or empty results leave the author with manual reference submit
or an honest `NO_CODE_REFS: <reason>` path; degraded lookup metadata never
becomes a fake reference.

`plan-manager author reference-apply <session> [--batch <id>] [--take <handle>]... [--drop <handle=reason>]... [--take-all] [--note <text>]`
applies the pending reference batch in one call. Taken locators enter the plan
or phase references; the rest are swept not-taken, with reasons required only
for high-confidence drops. `reference-accept` / `reference-reject` remain the
one-item lane and share the same batch disposition path.

`--drop <handle=reason>` is repeatable. Drop reasons are preserved as one flag
value, so spaces, commas, semicolons, and punctuation are safe:

```bash
plan-manager author context-apply sess-1 --drop "c2=too broad, stale, and not actionable."
plan-manager author reference-apply sess-1 --drop "r4=obsolete path, replaced by docs/reference/api.md."
```

### `decisions` — pinned plan-time contract decisions (optional)

`author section-submit <session> --section decisions --content "<title>: <statement>"`
records pinned plan-time decisions, one per line as `<title>: <statement>`
(rendered `D1..Dn` under **Approach & Decisions**). Assumption lines submitted
to the `assumptions` section may carry an `-> <mitigation>` suffix ("if wrong →
then"); those render as the two-column table under **Assumptions & Risks**.
Both are optional — empty renders nothing. Execution-time decisions are
different: capture those with `plan-manager log decision-add` as they happen.

Free-form phase `relevant_context` is classified as **notes only**:
`author phase-submit <session> <phase> --field relevant_context` records every
prose line as a note (`NO_CONTEXT:` reasons preserved) — it never turns prose
into an executable `prompt-manager skill read …` command. Executable setup
context must go through typed `author context-submit` / candidate acceptance.

### Change boundary

The `acceptance_boundary` section is **mandatory** and authored before references
and the regression anchor. Submit it with `author section-submit <session>
--section acceptance_boundary --content <boundary>`. The content accepts several
shapes:

- keyed lists:
  ```
  acceptance_allow:
  - scenarios/plan-manager/**
  - packages/proto/**
  acceptance_deny:
  - scenarios/swarm-manager/**
  ```
- a JSON object: `{"acceptance_allow":["scenarios/plan-manager/**"],"acceptance_deny":[]}`
- an `OPERATOR_ONLY: <reason>` escape for genuinely no-code / operator-only plans
  (the boundary analogue of `NO_CODE_REFS`).

There is **no `scope` flag and no primary-scenario flag** — scenario identity is
derived from the `acceptance_allow` globs. Finalize rejects unresolved
`<placeholder>` tokens in the boundary or anchor.

Because the anchor is **boundary-native**, anchor autofill (`author autofill
<session> --sources regression_anchor`) derives the affected scenarios and the
tiered baseline/diff commands from the change boundary — submit the boundary
first. The derived anchor block is `Strategy: change_boundary` / `Baseline name:`
/ `HEAD sha: <captured at execution start>` plus tiered commands (scenario
baseline **oracle** pairs and an **informational** repo/path diff); it carries no
`<scenario>` placeholder. The legacy `scenario_baseline` / `head_sha_allowlist`
anchor strategies remain import/read-only for pre-cutover plans.

## Persisted Plan Repair

Finalized plans can be repaired without archiving/re-authoring and without
editing the rendered markdown mirror. These commands mutate the structured
record, preserve phase identity, recompute plan metadata, and republish the
mirror from SQLite:

| Command | RPC | Purpose |
|---|---|---|
| `plan-manager plans context-list <plan> [--phase <phase>] [--workspace <root>]` | `PlansService.ListRelevantContext` | List global or phase-scoped relevant-context items. Legacy records without stored ids are exposed with stable effective ids such as `item-1`. |
| `plan-manager plans context-update <plan> <item> [--phase <phase>] --kind <k> [--label --reason --instruction --command --target --required --repeat]` | `PlansService.UpdateRelevantContext` | Replace one structured setup item. Missing/synthetic ids are replaced with durable ids on write. |
| `plan-manager plans context-remove <plan> <item> [--phase <phase>]` | `PlansService.RemoveRelevantContext` | Remove one setup item from the selected scope. |
| `plan-manager plans reference-list <plan> [--phase <phase>] [--workspace <root>]` | `PlansService.ListReferences` | List global or phase-scoped connected references. |
| `plan-manager plans reference-update <plan> <reference> [--phase <phase>] --kind <code|doc|req> --target <locator> [--future] [--note <text>]` | `PlansService.UpdateReference` | Replace one connected reference and clear computed resolution/staleness annotations so validation recomputes them. |
| `plan-manager plans reference-remove <plan> <reference> [--phase <phase>]` | `PlansService.RemoveReference` | Remove one connected reference from the selected scope. |

For a draft authoring session, use the authoring repair lane
(`author context-list/update/remove`, `author reference-accept/reject/apply`) and
run `plan-manager author validate <session>`. `plan-manager validate run <plan>`
is for persisted plans; if it is pointed at a draft-only handle, the CLI hints at
the authoring validation command.

## `phase` — direct phases on a persisted plan

`plan-manager phase add` and `phase update` expose the **canonical phase fields**
directly (not only via the authoring wizard): `--affected-areas`, `--steps`,
`--expected-outputs`, `--validation`, `--risks-hazards`, `--handoff-notes`, in
addition to `--title`, `--intent`, `--acceptance`, `--context`, `--reminders`,
`--baseline-scope`, and (on `update`) `--status`. `phase update` is
**full-replace**: the caller owns all fields it sends.

`plans update` preserves a plan's `import_provenance` and
`preserved_legacy_sections` when the caller omits them, so a routine
authored-field update never drops governance lineage.

`plan-manager plans list/get/render/archive` accept `--workspace <path>` so slug
lookup and list/archive/render behavior can be scoped consistently with the API.
`plan-manager plans get`, mutation outputs, and `plans render` surface the
rendered markdown mirror path when the API returns mirror metadata; render also
returns the resolved plan metadata in JSON. Use
`plan-manager plans import --source docs/plans/example.md --workspace <path>` to
resolve a relative source path from a workspace root. Use `plan-manager plans
reconcile --dry-run --workspace <path>` to preview missing/stale mirror repairs
and source intake. Run `plan-manager plans reconcile --repair-mirrors` to
repair projections from SQLite, or add `--source-intake` to import markdown from
the documented source locations. Add `--retire-sources` only when
sources that are already canonical, newly imported, or skipped as
duplicates should be removed after provenance/content checks. Dry-runs report
`source retirement planned`; apply runs report `source removed` for successful
retirement. Reconcile resolves repo `docs/plans` and `plans` scans from the
workspace root when provided, defaulting to the discovered Vrooli repo root in
the scenario CLI. Parse failures and conflicts are never removed by cleanup;
repair those files until `plans import` can parse them or move non-plan notes
out of plan source locations, then rerun reconcile.

## `log` — the execution-log ledger

The `log` group is the single durable home for the typed work products an agent
produces while executing a plan: decisions, candidate findings, filed bug
reports, reusable records, and notes (DISTINCT concepts — a finding is
unvalidated, a bug report is filed to the issue tracker, a record is reusable
learning). The CLI is **flat** (`plan-manager log decision-add`, not nested
`log decision add`). Bugs and records are forwarded downstream **internally** by
Plan Manager (bug → scenario-qa, record → swarm-manager); agents never invoke an
external scenario CLI from the plan workflow.

> **Moved from `exec`.** The former `exec decision-add`, `exec finding-add`,
> `exec findings`, and `exec triage` commands (and the `RecordDecision`/
> `RecordFinding`/`ListCandidateFindings`/`TriageFinding` RPCs) were **removed**.
> Use `log decision-add`, `log finding-add`, `log list --type finding`, and
> `log update --triage` / `log promote` respectively.

| Command | RPC | Purpose |
|---|---|---|
| `log decision-add <plan-or-execution> --title <t> [--phase --detail --evidence --source-command --idempotency-key --run-id]` | `LogService.AddDecision` | Record an in-flow design decision (feeds the handoff). |
| `log finding-add <plan-or-execution> --title <t> [--severity --phase --detail --evidence --source-command --idempotency-key --run-id]` | `LogService.AddFinding` | Record a CANDIDATE finding (a possible bug; never auto-promoted). |
| `log bug-add <plan-or-execution> --title <t> [--severity --phase --detail --evidence --source-command --idempotency-key --run-id]` | `LogService.AddBug` | File a bug report; forwarded to the issue tracker (scenario-qa) through an internal seam. v1 default is a pending stub (production adapter deferred), so the entry persists `pending` and is retried via `log sync`. |
| `log record-add <plan-or-execution> --title <t> [--phase --detail --evidence --source-command --idempotency-key --run-id]` | `LogService.AddRecord` | Capture a reusable record; forwarded to Swarm Manager through an internal seam. v1 default is a pending stub (production adapter deferred), so the entry persists `pending` and is retried via `log sync`. |
| `log note-add <plan-or-execution> --title <t> [--phase --detail --evidence --source-command --idempotency-key --run-id]` | `LogService.AddNote` | Record a lightweight progress/context note (local-only). |
| `log list [<plan-or-execution>] [--phase --type --triage --sync-status]` | `LogService.ListEntries` | List ledger entries with a compact summary. `--type` = `decision\|finding\|bug_report\|record\|note`; `--triage` = `candidate\|promoted\|dismissed`; `--sync-status` = `local\|pending\|synced\|sync_failed`. |
| `log get <id>` | `LogService.GetEntry` | Get one ledger entry by id, including its downstream reference. |
| `log update <id> [--title --detail --severity --triage --add-evidence]` | `LogService.UpdateEntry` | Update mutable fields; empty/unspecified leaves a field unchanged; `--add-evidence` appends. |
| `log promote <id> --to <bug\|record> [--title --detail --severity]` | `LogService.PromoteEntry` | Promote a finding into a bug report or record, preserving the original finding (marked promoted) and linking the new entry back to it. |
| `log sync <id>` | `LogService.SyncEntry` | Retry downstream forwarding for a `pending`/`sync_failed` bug or record. |

`--idempotency-key` makes retries safe (a retry with the same key returns the
existing entry); findings/decisions also dedup by
(execution, attribution run id, type, normalized title). A failed/pending
downstream sync is never fatal — the entry stays durable and is retried with
`log sync`.

## Output contracts

Every scenario command should render through one of three human
contracts. Proto-backed commands should use `cliapp.RenderProtoList`
or `cliapp.RenderProtoMutation`: human consumers see the report, while
`--json` consumers receive the proto JSON response shape.

| Contract | Used by | Structure |
|---|---|---|
| **Operational** | `status`, `health`, `audit`, `validate`, `doctor` | Status → Triage → Next Steps |
| **Data Retrieval** | `list`, `get`, `view`, `search` | Summary → Results → Retrieval Hints |
| **Mutation** | `create`, `update`, `delete`, `start`, `stop` | Result → What Changed → Next Command |

For commands that aggregate multiple API calls or produce a
non-proto report, use the `RunContext` render helpers directly
(`ctx.RenderList`, `ctx.RenderMutation`, or the operational report
helpers).

## Adding a new command

For a new domain, copy the worked CRUD command group in the fenced
example above first, then replace it once your real domain is green.

For a command inside an existing domain:

1. If the command needs a new API endpoint (RPC), add it first per
   [`api-endpoints.md`](api-endpoints.md#adding-a-new-endpoint). The
   manifest's coverage test will fail otherwise on the next CLI build.
2. Add a command entry to the matching group in
   [`cli/manifest.json`](../../cli/manifest.json): `name`, optional
   `description`, `positionals` / `flags`, the `binding` (service +
   method), and the `governance` block (`effect`, `run_eligible`,
   `permissions`, optional `requires_confirmation`). The schema in
   `.vrooli/schemas/cli-manifest.schema.json` is authoritative.
3. Implement the handler in `cli/domains/<domain>/handlers.go` (or a
   focused sibling file) with signature
   `func(ctx cliapp.RunContext) error`. Read values with
   `ctx.Flag(...)`, `ctx.BoolFlag(...)`, `ctx.Positional(...)`, and
   `ctx.JSON()`.
4. Add the handler to the bindings map in
   `cli/domains/<domain>/register.go` keyed by `"<Service>.<Method>"`
   so `cliapp.LoadFromManifest` can wire it. Missing handler or
   dead handler both fail at startup.
5. Handler implementation should:
   - Construct generated Connect clients with
     `cliapp.NewConnectHTTPClient(core)` for proto-typed operations.
   - Use `cliapp.UploadFile` only for documented multipart REST
     exceptions (append those outside the manifest path in
     `register.go` and document them in the manifest's `omitted[]`).
   - Render proto-backed responses with `cliapp.RenderProtoList` or
     `cliapp.RenderProtoMutation`.
6. Add endpoint metadata in the API handler module and bind the method
   (or list it in `omitted[]`) in `cli/manifest.json`. Then run
   `make endpoints`; do not edit [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json)
   by hand.
7. Add a row to this document.
8. Add a handler test in
   `cli/domains/<domain>/handlers_test.go` using `clitest.NewTestApp`
   + `clitest.NewAPIServer` + `clitest.CaptureStdout` (see
   [`../internal/TESTING.md`](../internal/TESTING.md)). Driving the
   handler via `cliapp.NewTestRunContextFromArgs` against the manifest's
   schema gives the closest parity with the dispatched path.

## Command structure principles

- **Subcommand groups** (`<domain> list`, `<domain> create`) over flat
  verbs (`list-<entity>`, `create-<entity>`). Discoverability via
  `--help` is the goal.
- **Positional for required, flags for optional.** `<domain> get <id>`
  not `<domain> get --id <id>`.
- **One command per API endpoint.** If you find yourself making two
  endpoint calls, the API is missing a use-case.
- **Error messages must be actionable.** "API unreachable" is bad;
  "API unreachable at http://localhost:15001 — try `--auto-start` or
  `vrooli scenario start plan-manager`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
