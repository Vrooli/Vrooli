---
name: "improve-skill-authoring"
description: "Author scenario-specific improvement judgment: approved target references, truthful evidence, priorities, and authority-aware repair routes. Author a matching setpoint-read program when setup scope includes it."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["meta"]
  tags: ["skill", "authoring", "improve", "self-improvement", "control-loop", "setpoint", "meta-optimization"]
  icon: "gauge"
  status: "active"
  revision: 8
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-08T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "program-runtime", "measures-health"]
    commands: ["prompt-manager skill read", "measures-health validate scenario", "program-runtime bindings unbound", "prompt-manager skill-set validate", "vrooli scenario test"]
  origin:
    kind: "authored"
---
## Meta focus: Improve Skill Authoring

Author scenario-specific judgment against an approved target the agent cannot redefine.
The resulting skill selects priorities, interprets evidence, and names repair methods.
`scenario-improvement-campaign` owns execution inside a mandate; `goal-loop` selects
development or observation. Ordinary scenario usage does not load the improve role.

Required reading:
- `path:docs/agent-system/SCENARIO_DEVELOPMENT.md` — authority, target artifacts, and completion.
- `path:docs/agent-system/SKILL_AUTHORING.md` §"Scenario skill sets" and §"Universal quality bars".
- `path:docs/agent-system/TARGET_MODEL.md` §2 — the control chain this skill instantiates at scenario scale. Sensor implies no authority.
- `path:docs/agent-system/FRAMEWORK_HEALTH.md` §"Deadband rule" — a deadband states the target, never the current reading.
- `prompt-manager skill read improvement-do-and-dont` — the anti-gaming patterns D1, D2, D3 the authored skill cites by id.
- `path:scenarios/program-runtime/docs/guides/program-contracts.md` — §"The envelope" defines program output; §"Setpoint-read rows" defines the row shape, closed `reason` vocabulary, and status treatment.

Inputs from `skill-set-authoring`: approved outcome references, the evidence and
program inventories, corpora and their approved floors, open problems, and role
applicability evidence. Keep declaration, successful observation, missing target,
and unavailable measurement distinct.

### 1. Scope

**In scope:** the improve skill's eight sections, setpoint table, and repair routes.
Author or revise `scenarios/<scenario>/.vrooli/program-runtime/setpoint-read.{py,json}`
when program-source edits are authorized. Skill-only work records that prerequisite;
it does not claim that a matching executable board already exists.

**Out of scope:** defining sensors or implementing product repairs while authoring this role; the usage role. The authored skill may route authorized implementation through owning methods inside the current engagement. A work-ladder rung does not imply another backlog item.

### 2. The eight sections

Use these eight core sections in order. Each has a source and an authoring rule.
Add `Troubleshooting & Edge Cases` when the universal quality bars require it;
the core section order does not exempt an operational skill from failure guidance.

| # | Section | Source | Rule |
|---|---|---|---|
| 1 | Focus and scope | the scenario's PRD one-liner; the dependents count | Name the plant (what is regulated) and what this skill never touches |
| 2 | Setpoint | approved outcomes and evidence inventory | Link each required outcome to named rows, sensor commands or missing-measurement gaps, approved bands, and dated observations. Mark diagnostic-only rows separately. Do not equate a row with a Swarm goal. |
| 3 | Sensors | owner definitions and observed reads | Name commands, validity and comparability rules. Include applicable shared diagnostics without making them universal gates. An independent applicable measurement outranks an unsupported self-report, not evidence with stronger validity. |
| 4 | Golden corpora | `evals/*.json` | Cite suites, approved floors, comparability, and sample-count rules. Preserve holdouts. A failed protected floor takes repair priority; changing it requires target approval. |
| 5 | Routes | problems, sensor rows, active scope | Name the owning repair method and expected evidence change. Use §5 for implement-versus-propose disposition; do not hardcode backlog filing. |
| 6 | Anti-gaming | `improvement-do-and-dont` §1 | Cite D1 (loosened or deleted tagged test), D2 (deleted known-issue ledger), D3 (suppressed finding) by id, and §2 (the skeptic test); add the scenario's own gaming moves (a waiver on a domain the sensor needs, editing a requirements registry to match a claim, widening a floor, deleting a failing corpus case) |
| 7 | Evidence | active owner log and shared Memory contract | Preserve target identity, valid before/after evidence, changes, unmet obligations, and the next action. Reuse automatic capture; do not create a parallel ledger. |
| 8 | Stop rules | approved outcome contract and runtime limits | Required missing or unreliable evidence remains unmet. Route permanent gaps immediately within authority; use owner waits for pending work. Honor aggregate budget, cancellation, and amendment boundaries. Completion needs every required outcome and its validity/sample rules, not a universal two-cycle test. |

### 3. Writing the setpoint

A setpoint row is a sentence an agent can check:

```
| discovery-floor | program-runtime discovery eval --suite evals/discovery.primary.json | met >= floor | 41/45 (floor 43) 2026-09-02 |
```

Rules:
- **Band references approved intent.** Never substitute the current reading or an invented trend for a missing target. Mark the target undecided; an approved trend needs a comparison window and acceptance rule. The program keeps `target` and `in_band` null until that rule exists.
- **Rows preserve outcome meaning.** Use one owner-derived reading per row. Keep composite acceptance in its owning evaluator; splitting inputs into rows must not replace a required relationship with unrelated passing values.
- **Readings are dated.** They are observations, not state; the next cycle re-reads them.
- **The recursive row** belongs to scenarios whose improve loop installs capability elsewhere (program-runtime, prompt-manager): "share of <targets> with a conformant skill set". Use `prompt-manager skill-set validate <target>` for structural findings and the `prompt-manager.skill-set-read` program for its declared inventory readings. Keep these readings distinct from skill quality, sensor reality, and executed program behavior. Verify a governed binding before including either in the sensor program; a local-only validator becomes `no_governed_binding`, and an unimplemented quality sensor remains `pending_telemetry`. The row's actuator routes authoring repairs to the owning scenario.

### 4. Authoring `setpoint-read`

The table is judgment, not executable code or a generator. When program work is
authorized, read `program-runtime` and follow its construction/contract guidance.
Reuse existing owner reads and sensor programs before adding composition.
`program-contracts.md` §"Setpoint-read rows" owns the envelope, reason vocabulary,
and status semantics; do not copy that protocol into the improve role.

Keep the board read-only with no inference or agent delegation. Compose only
children whose effects, budgets, and validity contracts fit that boundary. A
separately budgeted program may supply evidence through `read_elsewhere:<program>`;
the reference is not evidence until the caller consumes a valid result.

Preserve required rows even when their sensor or band is missing. Check authored
row IDs and band semantics against the program. Validate the executed fixtures
through `vrooli scenario test <scenario> --phases programs`, not just preflight.
Cover valid evidence, missing targets, permanent gaps, and failed/incomplete child
results. Assert row validity and unresolved obligations as well as parent status.
Declare expected statuses from the fixture inputs, never from today's accidental
outage. Live smoke is a separate qualification with its dependencies disclosed.

### 5. Routes

A route table row:

```
| agent-failure-rate above band with kernel_runtime naming a forbidden import | ladder W3: kernel guard for the name, then retire the prose | agent-failure-rate |
```

**Two sources, not one.** §2 lists this section's sources as *problems, sensor rows, active
scope*. Sensor rows are the easy half. The problems ledger
(`scenarios/<scenario>/docs/internal/PROBLEMS.md`) is the half that gets skipped, and skipping
it is silent: a cycle that reads only `setpoint-read` looks complete and misses everything the
ledger holds.

The two surfaces answer different questions, and the difference decides where a gap belongs:

| | A board row | A ledger entry |
|---|---|---|
| Has a target and a sensor | yes, both | neither |
| Says | "this reading is out of band, or unmeasured, and why" | "this capability does not exist, and here is what is wanted" |
| Example | `promotion-latency` at `pending_telemetry` | "the data exists in `catalog_audit` and no command reads it" |

A **wanted capability** — state that is persisted and unreadable, an absent corpus, an
operation nobody exposes — has no target and no sensor, so it can never be out of band and can
never appear on the board. It lives in the ledger and nowhere else. Do not promote it into the
Setpoint table to make it visible: a row invented without an approved target invites a band
derived from the current reading, which is §7's first anti-pattern.

The corollary matters for the usage skill too. An agent *using* the scenario is who discovers
the fifth missing capability, so the usage skill should name the known limits and point at the
ledger rather than let each agent rediscover them — citing, never restating, per
`path:docs/agent-system/LAYERS.md`. Adding an entry is the Observe exit
(`path:docs/agent-system/TARGET_MODEL.md` §8): one typed entry, no decision requested.

Note the asymmetry when you rely on this: the ledger's *existence* is a template artifact, but
nothing gates its currency or completeness. An empty or stale ledger is indistinguishable from
a scenario with no known gaps, so treat a silent ledger as unknown rather than clean.

Choose the smallest intervention that advances an approved outcome without
violating a protected floor. Diagnostic score movement alone is not success:

| If the fix is | Route | Who does it |
|---|---|---|
| A data or policy change the scenario already exposes (promote, supersede, gc, set-current, a config knob) | curation move, done in-cycle | the agent running the improve skill |
| A missing sensor (`pending_telemetry`) | `measures-adoption` and the owning implementation method | implement within the mandate; otherwise report the prerequisite |
| A missing binding (`no_governed_binding`) | owning binding method; use the work ladder when layer selection is needed | implement within the mandate; otherwise propose authorized follow-up |
| A contract, obligation, evidence, or implementation defect | owning method selected through `scenario-work-ladder` | repair within authority; a target amendment still needs approval |
| A dependency defect | dependency owner's method | repair only when included in the grant; otherwise report through the existing owner flow |

Read-only callers report these routes without performing them. Curation is an
effect too; require its actual grant. A cross-owner edit is not intrinsically
forbidden, but sensor evidence alone never authorizes it.

### 6. Convergence patterns

Two agents authoring the improve skill for the same scenario from the same inventory must produce the same setpoint rows, the same `reason` marks, and the same routes. The sources that decide it: the PRD's `OT-*` list, the inventory's `measured` column, the corpora's floors, the reason vocabulary in `program-contracts.md`, and the route table in §5.

### 7. Anti-patterns

| Anti-pattern | Why it fails | Instead |
|---|---|---|
| A band equal to the last reading | Reads in band while the defect stands; can only detect growth | Reference the approved target or retain an undecided target |
| A sensor the skill computes by hand from files | The skill becomes the instrument and can be gamed by editing | Cite owner evidence and route missing instrumentation under the active grant |
| Retry logic in the routes | Hides the failure class the next cycle needs | Keep retries in the owning operation; re-measure after a coherent intervention |
| A route that treats a sensor as authority to edit a dependency | The instrument grants itself scope | Check the mandate and use the dependency owner; otherwise report the missing authority |
| Prose goals with no sensor | The loop reports unsupported progress | Keep `pending_telemetry` and repair within authority; do not count the row as complete |
| Interpret board `ok` as product acceptance | Declared permanent gaps can coexist with `ok` | Inspect every required outcome using the owner row contract |

### 8. Output expectations

You may: author the improve role and its sensor program within setup authority. Report or file remaining work only through the caller's authorized path.

You must: distinguish existing sensors from measurement gaps; date readings; cite
anti-gaming by id; use the owner's wire vocabulary; verify table/program agreement
when authored; pass `skill-validation` §3.3 on the routes table. Report absent or
unexecuted programs as setup gaps, not successful behavioral validation.

Do not implement measures while authoring this role. Do not close product findings
solely because guidance was written. A row needs owner evidence or an explicit
measurement gap, not an adjective presented as a sensor.

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Every row is `pending_telemetry` | Measurements are absent | `measures-health validate scenario <scenario>` | Preserve the target with pending rows; report non-executable setup and route repair by authority |
| A corpus has no floor field | Target not established | Inspect the corpus contract | Propose a floor with comparable evidence and obtain approval; keep the target undecided, separately from sensor validity |
| `setpoint-read` cannot call a sensor because it is a local CLI command | The command has `binding.kind: local` | `program-runtime bindings unbound` | The row carries `unavailable: true`, reason `no_governed_binding`; route to the binding owner under the active grant |
| The scenario owns a projection but `space --projection` fails | Space contract unimplemented | `<scenario> space --json` | Keep coverage unverified; inspect the owning space contract and select its repair method under the active grant |
| Routes table has two rows for one reading | A C4 defect the divergence probe will catch | Run the probe | Merge or add a discriminating predicate |
