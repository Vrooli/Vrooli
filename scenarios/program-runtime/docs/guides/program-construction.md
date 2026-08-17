# Constructing a program

Program Runtime is a small, governed programming surface for agents. Write a
program when one task needs several scenario capabilities, bounded shaping,
or a result that should be retained as evidence. Start with a fresh session,
write the smallest program that returns a `Handle`, submit it with agent
provenance, inspect the result, and reclaim the session when the work is done.

The construction surface has three forms:

- Scenario bindings are flat top-level names: `search_hub.query.query(...)`.
- The project control plane is `vrooli.scenario.status(...)`.
- Runtime verbs are top-level: `discover(...)`, `gather(...)`, `ai`, `agent`,
  `recall`, `guide`, `validate`, `capture`, `describe`, `reachable`, and `lib`.

`__vrooli__` is the stable root escape hatch when a local variable shadows a
scenario name. Runtime-owned names cannot be overwritten. Use `describe()`
before calling an unfamiliar binding and use `rows="fieldName"` when a
response has multiple repeated fields.

Task-shaped examples:

- [Namespace and contracts](../construction/namespace-and-contracts.md)
- [Handle shaping](../construction/handles.md)
- [Inference and discovery](../construction/inference-and-discovery.md)
- [Delegation](../construction/delegation.md)
- [Failure recovery](../construction/failure-recovery.md)
