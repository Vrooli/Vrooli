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
  prompts/           # phase prompt templates with {{VARIABLE}} slots
  example-runs/      # simulation fixtures (schema kind: operating-mode-example-run)
```

- **`<id>`** is the mode id (kebab-case) and matches `mode.json`'s `id`.
- **`mode.json`** — identity + decision metadata, `scope`, `run_strategy`, the
  `phase_graph` (phases, guarded `transitions`, per-phase `declared_output`),
  and the `prompt` / `artifact` / `profile` / `backlog_sync` / `metrics` /
  `lock` / `ui` policy blocks. Validated against `operating-mode/v1`.
- **`prompts/`** — the prompt template a phase runs, referenced from a phase's
  `prompt.template`. Templates carry `{{VARIABLE}}` slots the engine fills
  through the shared render seam (byte-parity with the live spawn path).
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

> `prompts/` for the three shipped modes are populated as the rebuild lands
> (prompt extraction phase). A mode's `mode.json` may reference a
> `prompt.template` before the file exists at the schema level; the engine's
> semantic validator is what requires the referenced file to resolve.

## The three shipped modes

| Folder | Scope | Run strategy | Shape |
|--------|-------|--------------|-------|
| [`item-level/`](item-level/) | `backlog_item` | `existing_item_flow` | Default. Each item drains through the existing item pipeline; no mode rounds. |
| [`holistic-loop/`](holistic-loop/) | `initiative` | `operator_gated_loop` | `investigate → plan → execute → review → reconcile`, with `execute` looping back to `investigate` when `replan_needed`, and `review` looping back to `execute` when `verdict=changes_requested`. |
| [`phased-plan-drain/`](phased-plan-drain/) | `initiative` | `sequential_handoff` | `prepare_plan → execute_next → classify_progress → …`, branching on the progress decision (continue / replan / complete / blocked). |

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
output, prompt-template existence, and example-run path assertions).
