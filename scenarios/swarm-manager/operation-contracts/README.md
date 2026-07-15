# Operation catalog (agent-operations data SSOT)

This directory tree is the **authored data** the declarative agent-operations
layer loads at startup (`api/internal/opscatalog`). It is the *what* half of the
model — the *how* (operating modes) lives under [`../modes/`](../modes/), and the
concept SSOT is
[`../docs/concepts/AGENT-OPERATIONS.md`](../docs/concepts/AGENT-OPERATIONS.md).

```
operation-contracts/   # one JSON per operation contract (id + version)
target-capabilities/   # the capability each target kind provides (registry mirror)
../bindings/           # system-default bindings: which mode implements an operation
../policy/             # transition policies: which domain action fires on an outcome
```

Each document validates against an `api/internal/agentops/schemas/*.json` schema
plus the Go semantic validators. The loader fails **closed**: any invalid,
duplicate, or version-conflicting document aborts the whole load. Every document
is pinned by a canonical content digest so an execution records exactly which
bytes decided it.

- **operation-contracts/** — provider-neutral `id@version` contracts declaring
  required target capabilities, typed caller inputs, result shape + closed
  outcomes, evidence expectations, and cancellation/retry. Regenerate from
  `SeedOperationContracts` with `go run ./api/cmd/genopscatalog <scenario-root>`
  (the Go function is the SSOT — edit it, not the JSON). Every contract is bound
  to an implementing operating mode by a system-default binding under
  [`../bindings/`](../bindings/); the mode's `target.kind` must be compatible with
  the operation's target or binding resolution fails closed.
- **target-capabilities/** — a mirror of the `agentops.TargetCapabilities()` Go
  registry (the SSOT the runner actually consults); shipped as data for
  inspection and external tooling.
- **../bindings/** — system-default bindings only (one per operation).
  Item/initiative/invocation overrides live in domain storage, never here. A
  binding pins a mode revision; a deleted revision is a typed error, not a silent
  re-resolution. The operation→mode map: research-refine→backlog-research,
  workshop-round→backlog-workshop, workshop-finalize→backlog-finalize,
  clarification-{start,continue}→backlog-clarify, execution-{run,retry}→
  execution-drain, execution-fixup→backlog-fixup, execution-followup→
  backlog-followup, review-round→backlog-review, evidence-request→backlog-evidence,
  revision→backlog-revision, initiative-review→initiative-review-loop.
- **../policy/** — transition policies over the closed domain-action registry. A
  policy can name only a registered action, never code, a path, or a command.
