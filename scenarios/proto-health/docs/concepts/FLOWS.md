# Flows — Proto Health

Proto Health's current product behavior is read-oriented: validate proto
contracts, describe a scenario's proto surface, compare a surface to a git
baseline, and report runtime health. These operations do not create or mutate
scenario-owned business records, so there is no active domain workflow or
state machine to model.

The former generated notes CRUD and attachment-upload flow were removed from
the runtime when the validation and proto-surface domains shipped. Older
flow-verifier artifacts that describe that example are historical references,
not current production behavior.

If a future capability introduces durable state or asynchronous transitions,
add its domain-owned flow here and keep the transition contract beside the
implementation. Do not model ordinary request/response validation as a
stateful workflow.

## Current operations

| Operation | Owner | Observable result | State change |
|---|---|---|---|
| Validate scenario | validation | Structured findings for one scenario's proto contract. | None. |
| Describe proto surface | protosurface | Descriptor-backed files, services, messages, imports, and transport facts. | None. |
| Compare impact | impact | Compatibility findings against a selected git baseline. | None. |
| Report health | health | API readiness and dependency status. | None. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — capability ownership
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md) — owner test suites
