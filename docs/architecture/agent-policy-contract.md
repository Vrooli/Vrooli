# Agent Policy Contract v1

`agent-policy/v1` is the stable boundary between native coding-agent adapters,
the installed policy runtime, and scenario-owned providers. JSON is the wire
encoding so a hook can run offline on Linux, macOS, or Windows without importing
a scenario or Agent Manager.

## Event

`ToolEvent` carries `runner`, `tool`, structured `arguments`, optional shell
provenance, working directory, target, timestamp, and opaque context. The
runtime never reparses a shell-rendered display string as argv. Empty or future
contract versions are rejected with explicit evidence. If `event_id` is absent,
the runtime derives a deterministic digest from runner, tool, argv, shell,
working directory, and target.

## Provider snapshot

A snapshot declares one provider's version, capabilities, scope, health,
readiness, evidence state, expiry, policy rules, and provenance. Declared
maturity is independent from runtime health:

| Maturity | Meaning |
| --- | --- |
| experimental | contract or capability is not ready for reliance |
| advisory | observations and guidance are available |
| guided | explicit confirmation is available for risky work |
| guarded | selected high-risk work can be blocked safely |
| enforcing | conformance and canary evidence supports fail-closed use |

Providers cannot elevate their own maturity at evaluation time. Promotion and
withdrawal are operator/audit operations that publish a new bundle.

## Bundle

The snapshot store contains one `snapshot-bundle.json`, not one mutable file
per provider. Its generation, publication time, provider map, and SHA-256
integrity are replaced together. A provider bridge adds or withdraws one entry
and republishes the complete bundle. A malformed, tampered, future-clock, or
expired bundle is unavailable evidence; it is never clean evidence.

## Decision and fallback

| Profile | Healthy enforcing evidence | Missing/stale/unavailable evidence |
| --- | --- | --- |
| advisory | allow with evidence | allow with degraded evidence |
| guided | allow low-risk; ask high-risk below maturity | ask high-risk |
| guarded | allow low-risk; ask below guarded maturity | deny high-risk |
| enforcing | deny high-risk below enforcing maturity | deny high-risk |

Every ask, rewrite, route, repair, deny, and unavailable result includes a
reason and evidence. Unknown, opaque, unsupported, stale, and unavailable are
distinct states. Low-risk inspection and frozen reproduction remain usable
under degraded advisory/guided operation.

## Repair contract

A repair plan must identify an owner, operation, scenario-relative target root,
explicit file scope, preview digest, transaction id, expiry, rollback metadata,
validator, and idempotence. Preview does not mutate. Apply must match the
preview digest and target scope, write atomically, preserve the original
finding/provenance, and rerun the authoritative validator. Package changes
always route through SDA; a textual shell command is never a repair primitive.

## Exit codes

The runner emits JSON before returning a stable hook status: `0` continue,
`10` ask/confirm, `20` deny, and `30` unavailable or invalid runner state.
Native adapters must preserve their own allow/deny controls when a runner is
unavailable or a hook is unsupported.
