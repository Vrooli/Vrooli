# Trials Fixture Authoring

The `trials` domain proves local-coding-agent readiness **empirically**: it
dispatches a sandboxed agent at a task and then decides whether the task was
actually solved. "What counts as solved" is defined by a committed **fixture**,
one per suite family, so the verdict is deterministic and the trend is
comparable over time. The Guide space defines *which families exist*; fixtures
define *what solved means*.

Corpus lives at `trials/fixtures/<family>/`. Families: `add-feature`,
`research`, `comprehend`, `bugfix`, `negative`.

## Layout

```
trials/fixtures/<family>/
  fixture.json   metadata (see below)
  spec.md        the agent prompt — the concrete task to perform on target/
  check.sh       the deterministic oracle (exit 0 = solved). Omit for negatives.
  target/        the minimal codebase the agent edits (the agent's scope-path)
```

The agent's sandbox scope is **`target/` only**. `spec.md` is handed to the
agent as the task description; `check.sh` lives OUTSIDE `target/` so the agent
cannot read or game the oracle.

## fixture.json

| field         | meaning                                                         |
|---------------|----------------------------------------------------------------|
| `family`      | suite family this fixture represents                           |
| `negative`    | `true` for honesty/abstention fixtures (no oracle)            |
| `prompt_file` | the spec file (default `spec.md`)                             |
| `oracle`      | the check command, e.g. `["bash","check.sh"]`; `[]` for negatives |
| `target_dir`  | the agent's scope subdir (default `target`)                  |

## How evaluation runs

1. The Runner dispatches the agent at `spec.md`, scoped to `target/`, and
   collects the produced diff + token/time metrics (agent-manager, sandboxed).
2. The Evaluator copies `target/`, applies the agent's diff, and runs the
   oracle there.
   - **cwd of the oracle is the diff-applied copy of `target/`** — so a check
     inspects the agent's result with plain relative paths (`grep -q DONE foo`).
   - Any oracle argument that names a file in the fixture dir (e.g. `check.sh`)
     is resolved to its absolute path, so the script is loaded from the fixture
     dir but runs against the copy.
   - Exit `0` → PASS; clean non-zero exit → FAIL; could-not-run → ERROR.
3. **Negative fixtures** have no oracle: PASS = correct abstention (the agent
   made no substantive change), FAIL = it fabricated a diff.

## Authoring a new family fixture

1. `mkdir -p trials/fixtures/<family>/target`.
2. Write `target/` with the minimal starting codebase.
3. Write `spec.md` — a self-contained task that references only files the agent
   can see under `target/`. Tell it which files to edit / create.
4. Write `check.sh` so that **exit 0 ⇔ the task is solved**. Keep it
   hermetic (only the tools you can rely on: bash + coreutils + git). Make it
   robust to reasonable solution variation (assert behaviour, not exact text).
5. Add `fixture.json`.
6. The fixture **revision** is a content hash of `spec.md` + the oracle command
   + every file under `target/`. Editing any of them changes the rev, which
   invalidates the `(task, model, fixture-rev)` idempotency key so the trend
   never compares runs across incompatible fixture versions. This is automatic —
   nothing to maintain by hand.

## Invariants

- A missing/oracle-less family degrades that one run to an honest
  `VerdictError`; it never blocks the rest of the suite and never fabricates a
  pass.
- Oracles must be deterministic and self-contained — no network, no host state.
- The agent must never be able to read the oracle (keep `check.sh` out of
  `target/`).
