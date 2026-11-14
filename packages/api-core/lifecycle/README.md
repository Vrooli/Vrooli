# Provider lifecycle fence

`api-core/lifecycle` is the shared transport, ingress gate, and provider layer
for `common.v1.LifecycleMaintenanceService`.

The standard `api-core/server.Run` path mounts a provider automatically when
the lifecycle environment identifies a scenario. It tracks in-flight HTTP
requests, keeps health probes and loopback lifecycle RPCs available while
admission is closed, and supplies `Status`, `Prepare`, `Drain`, and `Resume`.
Scenario code only supplies `Hooks` when work can outlive its request: an
inventory snapshot plus paired quiesce/resume hooks. Custom HTTP servers can
use `NewProviderFromEnv` and `Provider.Mount`; domain owners with external
processes or richer work inventories can continue to implement the Connect
service and matching admission middleware themselves.

A control-plane replacement uses one durable operation ID to:

1. ask the live provider to close admission;
2. observe complete inventory and drain it when required;
3. retain the returned fence token and revision while replacing the process;
4. resume the provider only after the replacement is healthy.

The standing declares `fence_persistence`. The shared provider uses
`PROCESS_LOCAL`: its replacement starts open and must not receive an old
process token. A durable owner uses `OPERATION_DURABLE` and retains the same
operation, token, revision and interlock in its own store. After replacement,
the control plane resumes that exact fence only with a fresh complete, drained
inventory and no pending admission or quiescence. A known process-local fence
cannot be converted into a durable one. An older unspecified predecessor can
be continued only when its successor explicitly supplies that exact durable
proof. A new provider generation by itself is not proof of open admission.

Mutation calls are mounted behind `LoopbackOnly` because the current control
plane invokes them over loopback. A deployment that proxies a scenario must
provide an equivalent authenticated local boundary before exposing the
lifecycle service. Missing contracts and incomplete inventory fail closed by
default. An operator can use the control plane's restart-only
`--force-lifecycle --lifecycle-override-reason ...` option to accept that
replacement risk explicitly; the control plane records the reason and warns
that in-flight or detached work may be interrupted. This override does not
permit replacement when process identity is non-authoritative.

`emergency_override` remains an explicitly reasoned operator decision. It may
override eligible blockers. It never substitutes for identifying the target
process in the control plane.

The loopback `Drain` RPC follows its operation/request context, including a
request timeout when supplied. Its transport clears the ordinary HTTP response
write deadline so a long quiescence wait can return its receipt after ingress
has closed. Other requests retain the server write deadline. This exception
preserves client cancellation and the owner must still await every tracked
worker before reporting quiescence.
