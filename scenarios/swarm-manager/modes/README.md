# Operating modes (data SSOT)

Each subdirectory here is **one operating mode expressed as data** — the single
source of truth for a reusable, inspectable, testable methodology loop. One
generic engine under `../api/internal/operatingmode/` interprets these folders;
there is no Go file that declares a mode. Adding or changing a mode is a **data
edit**, not a code change or a redeploy.

Concept and vocabulary: [`../docs/concepts/EXECUTION-MODES.md`](../docs/concepts/EXECUTION-MODES.md).
Contract: [`.vrooli/schemas/operating-mode.schema.json`](../../../.vrooli/schemas/operating-mode.schema.json) (`$id: operating-mode/v1`).

## Folder layout

```
modes/<id>/
  mode.json          # the mode contract (schema kind: operating-mode)
  example-runs/      # simulation fixtures (schema kind: operating-mode-example-run)
```

- **`<id>`** is the mode id (kebab-case) and matches `mode.json`'s `id`.
- **`mode.json`** — identity + decision metadata, `target` (the unit of work:
  `backlog-item` | `initiative` | `plan-execution` | `scenario`), `run_strategy`, the
  `phase_graph` (phases, guarded `transitions`, per-phase declared `reads` and
  `declared_output`),
  and the `prompt` / `artifact` / `profile` / `backlog_sync` / `metrics` /
  `lock` / `ui` policy blocks. Validated against `operating-mode/v1`.
- **Prompt skills** — `mode.json` declares `prompt.catalog_prefix`, and each
  phase can declare prompt metadata plus an optional suffix. The engine resolves
  that data to a prompt-manager SkillID (for example,
  `swarm-manager-phased-plan-execute-next`) and renders the skill through the
  shared prompt seam. Prompt bodies stay in prompt-manager; they are not copied
  into the mode folder.
- **`example-runs/`** — one JSON file per simulation preset. Each seeds phase
  outputs and asserts the `expected_path` the **real** generic guard evaluator
  produces, so a mode is tested before it is trusted. These are also the data
  behind the UI Flow tab's simulation presets. They never spawn agents, acquire
  locks, or persist state. One example-run per file, ordered happy-path first;
  the reserved `happy-path` id is the simulator's default preset. Author one step
  per non-terminal phase visit in order (the terminal phase takes no step); a
  step's `output` overrides the contract-derived scaffolding and feeds the real
  guards, and a guarded stop (a matched guard with no target, e.g. a `blocked`
  decision) ends the walk. The loader replays every example-run at startup and
  rejects one whose walked path no longer equals its `expected_path`.

The resolved phase SkillID is part of the prompt catalog projection. Use
`swarm-manager operating-mode get --mode <id> --json` to inspect the mode data
and the prompt catalog commands to read the underlying skill body when needed.

## The shipped modes

Fifteen operating modes ship as folders here — the ten `backlog-*` operations
(clarify, conclude, evidence, finalize, fixup, followup, research, review,
revision, workshop) that implement per-item work on a `backlog-item` target,
`execution-drain` and `phased-plan-drain` on a `plan-execution` target,
`holistic-loop` and `initiative-review-loop` on an `initiative` target, and
`scenario-spec-sync` on a `scenario` target. Each is one folder, one `mode.json`.
The two most illustrative:

| Folder | Target | Run strategy | Shape |
|--------|--------|--------------|-------|
| [`holistic-loop/`](holistic-loop/) | `initiative` | `operator_gated_loop` | `investigate → plan → execute → review → reconcile`. `plan` authors and binds the plan-manager plan; `execute` is `executed_by: phased-plan-drain` (composes the generic drain), routing `progress=complete → review` and `progress=blocked → investigate` (the composed replan); `review` loops back to `execute` when `verdict=changes_requested`. |
| [`phased-plan-drain/`](phased-plan-drain/) | `plan-execution` | `sequential_handoff` | The generic plan-first drain: a single `execute` phase loops on itself via one classified edge deriving `progress` (continue → execute, complete / blocked → guarded stop). No terminal phase — every stop is a guarded stop. |

Initiatives that run no methodology loop use the **member-item workflow
strategy** instead: each member item runs through its own operation and the
initiative only provides scheduling strategy. That strategy is *not* an operating
mode — it has no folder here and no `mode.json`. It survives only as a sentinel
value on the initiative's persisted `mode` field: the string `item-level` (or a
blank value, which has always meant the same thing). The loader rejects any
folder claiming id `item-level`; it is member-item strategy configuration, never
a selectable methodology.

## Authoring a new mode

Authoring is self-serve and writes data — scaffold a folder, validate it against
the schema and semantics, simulate it against its example-runs, and run it, with
zero Go edits and no redeploy. See
[`../docs/internal/OPERATING-MODE-AUTHORING.md`](../docs/internal/OPERATING-MODE-AUTHORING.md).

## Validating mode data

Every `mode.json` and example-run validates against `operating-mode/v1`. Quick
structural check with any Draft 2020-12 validator, e.g.:

```bash
python3 - <<'PY'
import json
from jsonschema import Draft202012Validator
s = json.load(open('.vrooli/schemas/operating-mode.schema.json'))
v = Draft202012Validator(s)
for f in ['scenarios/swarm-manager/modes/holistic-loop/mode.json']:
    errs = list(v.iter_errors(json.load(open(f))))
    print(f, 'OK' if not errs else [e.message for e in errs])
PY
```

The engine's loader/validator adds the semantic checks JSON Schema cannot express
(reference resolution across phases, guard field-paths resolving against declared
output, prompt skill/catalog resolution, and example-run path assertions).
