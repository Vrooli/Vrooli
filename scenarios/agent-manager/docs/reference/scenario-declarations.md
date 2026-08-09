# Scenario-owned agent declarations

Scenarios declare their agent-manager assets — coding-agent **profiles** and
**workflows** — in one canonical location, registered through one reconcile
entry point. This is the unified declaration layer introduced by the
declared-run doctrine. There is **no legacy fallback**: the old
`.vrooli/agent-profiles/` and `.vrooli/agent-workflows/` directories and the old
`config.profiles` / `config.workflows` service.json blocks are rejected with
actionable diagnostics.

## File location and format

All declaration files live under `.vrooli/agent-manager/`. Each file is
discriminated by its top-level `schemaVersion`:

| `schemaVersion`     | Kind     | Lifecycle                                             |
| ------------------- | -------- | ---------------------------------------------------- |
| `agent-profile/v1`  | Profile  | Mutable posture with `update_if_unmodified` drift tracking |
| `agent-workflow/v1` | Workflow | Digest-pinned immutable revisions (atomic activation) |

`schemaVersion` on a profile file is a file-format marker only. The reconcile
reader peeks it to route the file, then strips it before the strict
`AgentProfile` proto parse — it never appears on the runtime profile entity, its
DB row, or the public API. Workflow files already carry
`schemaVersion: "agent-workflow/v1"`; that value is part of the digested
definition and is unchanged.

## Service manifest block

### Cross-scenario command semantics

A CLI manifest may opt into generic cross-scenario investigation by adding a
`semantics` object to a command. Its required `kind` is one of `query`,
`verify`, `guidance`, or `mutate`; an optional `verdict` identifies pass/fail
fields in command output. The block is optional, so undeclared commands remain
available but their semantic measures are unavailable rather than guessed.

## Canonical tool restrictions

Profiles use the runner-neutral canonical tools `read`, `write`, `edit`,
`glob`, `grep`, `shell`, `web_search`, and `web_fetch`; native runner names
are never valid declarations. Reconcile rejects an unknown token with its
nearest canonical suggestion. The editor schema is
[CODE: scenarios/agent-manager/schemas/agent-profile/v1.schema.json].

An allowlist is enforced by default. If role routing selects a runner that
cannot enforce it, launch fails closed. Set
`"toolRestrictionPolicy": "advisory"` only when the caller deliberately
accepts that the runner cannot apply the restriction. Do not combine an
allowlist with `skipPermissionPrompt: true`: reconcile surfaces a warning,
and such profiles are not suitable for an enforceable restriction.

Use `agent-manager runner tools` to inspect the codec-sourced native mappings
and enforcement status on the running service.

A scenario declares its sources under
`dependencies.scenarios.agent-manager.config.declarations`:

```json
"agent-manager": {
  "enabled": true,
  "required": true,
  "startup_policy": "must_start",
  "config": {
    "declarations": {
      "reconcile": true,
      "profileMode": "update_if_unmodified",
      "sources": [
        ".vrooli/agent-manager/default.json",
        ".vrooli/agent-manager/deep-work.json",
        ".vrooli/agent-manager/backlog-workshop-round.json"
      ]
    }
  }
}
```

- `reconcile` (required): whether the sources reconcile on this scenario.
- `profileMode` (optional, default `update_if_unmodified`): reconcile mode for
  the profile-kind sources — one of `create_only`, `update_if_unmodified`,
  `force`. Only profiles carry a mode; workflows are always digest-pinned.
- `sources` (required, ≥1, unique): target-relative paths that must live under
  `.vrooli/agent-manager/`.

A scenario that requests only portable roles at runtime declares the
agent-manager dependency with **no `config`** and omits the block entirely.

## Reconcile entry points

One entry point reconciles both kinds in a single call, fanning out per source
by `schemaVersion` and preserving each kind's semantics (profiles reconcile
per-source with drift tracking; workflows activate as one atomic batch — if any
workflow source fails validation, the whole workflow batch is withheld).

```bash
agent-manager declarations reconcile-scenario --scenario <slug> [--dry-run] [--validate-only]
agent-manager declarations plan --scenario <slug>
```

The legacy `profile reconcile-scenario` and `workflow reconcile-scenario` verbs
still work as thin projections over this unified reconcile, returning only their
respective kind's results. They read exclusively from the new block.

At startup agent-manager sweeps every scenario declaring the block and
reconciles it, isolating per-scenario failures so one broken manifest never
blocks readiness.

## The declared-run doctrine

An agent interaction **must** be a declared workflow when **code composes the
prompt and code consumes the output**. That is the whole test. If a scenario
assembles a prompt from typed inputs and then parses the agent's result back into
typed state, that middle — prompt assembly, prompt-manager fetching, execution,
extraction, classification, retries, looping — belongs in a definition, not in
hand-rolled glue.

- **Chat carve-out.** Interactive and conversational surfaces (web-console
  sessions, agent-manager interactive mode, the operator CLI) stay on plain runs.
  There is no code composing a prompt and consuming a structured result there; a
  human is in the loop. The raw run API also remains the substrate *under*
  workflow nodes and internal plumbing — declaring a workflow does not remove the
  run API, it layers on top of it.
- **Two ends in code.** A scenario adopting the doctrine keeps exactly two ends
  in its own code: building the typed input snapshot, and applying the typed
  result to domain state. Everything between those two ends is the definition's
  job. Any residue beyond those two ends (a bespoke extractor, a poll loop, a
  substring-sniffed failure classifier) is a sign the doctrine is not yet fully
  applied.
- **A one-run feature is still a workflow file.** The value being purchased is
  declaration plus registration-time validation, not orchestration. Do not
  invent a new entity kind for "just one run": a declared run is a single-node
  workflow (see sugar below), never a third kind of thing.

## Minimal declared run (single-node sugar)

A definition whose only node is a `run` node may omit `entryNode`, `edges`, and
the terminal `end` node. The catalog canonicalizes the shorthand into the full
form **before** digesting — it synthesizes the entry, an implied `end` that maps
the run node's structured result to `output.result`, and the single unconditional
edge — so the runtime and the digest see one representation. An equivalent
explicit definition canonicalizes to identical bytes and therefore an identical
digest.

```json
{
  "schemaVersion": "agent-workflow/v1",
  "owner": "my-scenario",
  "key": "my-scenario/summarize",
  "version": "1.0.0",
  "inputSchema": { "type": "object", "additionalProperties": true },
  "outputSchema": {
    "type": "object",
    "properties": { "result": { "type": "object", "additionalProperties": true } },
    "required": ["result"],
    "additionalProperties": false
  },
  "nodes": [
    {
      "id": "summarize",
      "kind": "run",
      "run": {
        "roleRef": "code.default",
        "promptTemplate": "Summarize the change below.\n\n{{.diff}}",
        "resultSpec": { "version": "result-spec/v1", "kind": "json_schema", "extractionMode": "deterministic_only", "schema": { "type": "object", "additionalProperties": true } },
        "bindings": [
          { "name": "diff", "source": "workflow_input", "selector": "$.diff", "limit": 1, "maxBytes": 8192, "renderAs": "text", "missingPolicy": "error" }
        ]
      }
    }
  ],
  "budgets": { "wallTimeSeconds": 600, "maxTurns": 8, "maxTokens": 40000, "maxChargeMicroUsd": 5000000, "maxNodeAttempts": 2, "maxChildren": 1, "maxConcurrency": 1, "maxRecursion": 1, "maxRetries": 1, "maxWaitSeconds": 60 }
}
```

The implied end is exactly:

```json
{ "id": "end", "kind": "end", "end": { "status": "succeeded", "bindings": [
  { "name": "result", "source": "structured_result", "selector": "$.value", "order": "desc", "limit": 1, "maxBytes": 16384, "renderAs": "json", "missingPolicy": "error" } ] } }
```

with `entryNode` set to the run node and one edge from the run node to `end`. The
single run node may not itself be named `end`.

## Referencing a prompt-manager skill (`promptRef`)

A `run` or `continue` node may resolve its prompt from a prompt-manager skill
instead of inlining it. Author **exactly one** of `promptTemplate` or `promptRef`.

```json
"run": {
  "roleRef": "code.default",
  "promptRef": { "skillId": "my-scenario-process-thing", "variables": { "MODE": "strict" } },
  "resultSpec": { "...": "..." },
  "bindings": [ "..." ]
}
```

Use `code.smart` only for work that needs deep reasoning or substantial
multi-file changes. Profile declarations that retain it should include a
one-line `roleReason`; reconciliation reports a non-blocking warning when the
reason is missing.

Resolution happens at **reconcile time**, before the digest. The resolved content
is embedded into `promptTemplate` and pinned into the revision alongside its
provenance (`promptProvenance`: skill id, revision, variant, content hash), so a
revision's behavior is immutable even if the skill later changes. A changed skill
resolves to different content, a different digest, and therefore a **new revision
on the next reconcile** — never a silent behavior change under a fixed digest.
A resolution failure (missing skill, prompt-manager unreachable) fails that
source and, because workflow activation is an atomic batch, withholds the whole
batch: reconcile never registers a partially-resolved revision. `promptProvenance`
is engine-populated — authors do not write it.

## Workflow trigger policy

Each workflow may define a `trigger` block. Agent Manager enforces this policy
only when it starts an execution. Intermediary scenarios forward caller identity
but do not implement a second policy check.

```json
"trigger": {
  "initiators": ["human", "programmatic", "agent"],
  "selfTrigger": { "mode": "allow", "maxDepth": 2 }
}
```

`initiators` is optional. Its default allows `human`, `programmatic`, and
`agent`. `selfTrigger` defaults to `{ "mode": "deny" }`. Set `mode` to
`allow` only with a positive `maxDepth`; the depth counts existing executions
of the same workflow in the verified caller chain. `maxDepth` is not valid for
`deny`.

Agent Manager classifies a valid identity token as `agent`. A request without a
token is `programmatic`. A trusted UI or CLI session may mark a request
`human`. If a request presents itself as an agent but the token is missing,
invalid, revoked, expired, or cannot be verified, the trigger is denied and an
audit entry records `identity_unverified`; it is never reclassified as
programmatic. This prevents an identity-verification outage from becoming an
authorization grant. The policy does not prevent an agent from clearing its
environment; hard isolation needs sandbox network controls.

## Workflow run scope

A `run` node can set `scopePathTemplate`. It renders from that node's declared
bindings before Agent Manager creates the task workspace. Use it when workflow
input identifies the scenario that an agent may change.

```json
{
  "scopePathTemplate": "scenarios/{{.scope_scenario}}",
  "bindings": [
    { "name": "scope_scenario", "source": "workflow_input", "selector": "$.targetScenario", "renderAs": "text" }
  ]
}
```

The template can use only declared binding names. An undeclared or malformed
reference fails declaration reconciliation. The rendered path still receives
normal task scope validation, including rejection of path traversal.

## Prompt maturity gradient

An inline `promptTemplate` remains valid and produces an `inline_prompt`
reconcile warning. It is the on-ramp. A mature workflow uses `promptRef` for
every workflow prompt so prompt-manager can expose the prompt to operator
editing, discovery, and meta-optimization scrutiny. The warning does not block
reconcile.

## Journal helpers for edge conditions (`latest`, `count`)

Branch routing lives on edge `condition` strings (CEL). Two helpers over the
execution `journal` remove the repeated "newest successful structured result"
incantation:

- **`latest(journal)`** — the newest successful structured result across every
  node, as its entry map (carrying `.value`, `.status`). Returns an empty map
  when none exists yet, so guard with `has(latest(journal).value)` where the
  journal may be empty.
- **`latest(journal, nodeId)`** — the newest result-bearing entry for one node: a
  successful structured result (carrying `.value`) or a child-workflow completion
  (carrying `.output`). Empty map when the node has produced none.
- **`count(journal, nodeId)`** — how many times a node has been attempted (its
  traversal count).

```text
# Route to the correction node when the latest slice asked for a correction,
# or the latest independent review was not accepted:
(latest(journal).value.outcome == 'continue' && latest(journal).value.correctionRequired)
  || (has(latest(journal, 'review').output) && !latest(journal, 'review').output.review.accepted)

# Stop once the consumer's slice budget is spent:
latest(journal).value.outcome == 'continue' && count(journal, 'slice') >= input.constraints.maxSlices
```

The helpers read the engine's enriched journal projection (each entry carries
`nodeId` and `kind` alongside its payload fields) and are registered on the same
CEL environment the catalog compiles conditions against, so a condition that
validates at registration is one the engine can evaluate.
