# Resource Interface Standards

### Managed-service companions

A managed-service companion is launched with the supervised resource PID in
`--parent-pid`; it must not infer ownership from the short-lived control-plane
process or from `os.Getppid()`. The companion exits when that resource process
exits. Crash attempts are retained in the companion state directory separately
from the reap-able `.pid` file, so status inspection cannot erase crash-loop
evidence.

Companion declarations support `required: true`. A down required companion
makes the resource unhealthy. An optional companion remains visible in status
but does not change the resource health verdict. Whisper's activity edge is
required because it owns the canonical client port 8090.

An acceleration claim of `resource_kind: vram` is valid only when the resource
declares a non-CPU accelerator backend. CPU-only resources must not reserve
video memory they cannot use.
This page describes the current expectations for resource-facing platform surfaces.

## Current Truth

The canonical project-level resource surface is the `vrooli resource` command group.

Resource-specific wrappers may still exist in some places, but they should not be treated as the default design center for new resource work unless explicitly retained.

## Expectations

At the platform level, a well-behaved resource should have:

- a clear canonical name
- discoverable metadata
- manifest-backed configuration where implemented
- status and validation behavior
- lifecycle behavior that is consistent with the control plane

Shared React components also expose a restyle seam. A reusable component must
accept an optional `className`, merge it with the published `twMerge`-semantic
helper so consumer utilities win conflicts, and forward the result to its
outermost rendered element. Consumers should link the governed package and
compose this seam at the call site; copying the component is reserved for a
reasoned ejection when the public contract is insufficient.

## Useful Commands

```bash
vrooli resource list
vrooli resource info "<name>"
vrooli resource status
vrooli resource validate "<name>"
vrooli resource logs "<name>"
vrooli resource restart "<name>"
```

## Guidance

- prefer consistency with the root CLI
- avoid inventing special-case interfaces without strong reason
- keep compatibility shims isolated and documented as transitional when they still exist
- keep resource-specific wrappers subordinate to the canonical `vrooli resource ...` surface
- keep resource-local entrypoints thin when shared control-plane packages can own the behavior
- keep source-build tooling separate from deployed artifact delivery; an
  end-user resource path must not depend on Go or Bash
- expose target limitations, host requirements, and fallbacks before scenario
  deployment/runtime attempts the resource

## Health Checks Are Readiness Probes

The manifest `health_checks` entries are what orchestration ordering, status,
and consumer auto-recovery believe. They must therefore mean "able to serve
the primary capability" — a service whose model has not loaded, whose first
request would block on a large download, or whose admission path is closed is
not healthy, even if its process answers a liveness endpoint.

Declare semantics explicitly with the `kind` field (`readiness` or
`liveness`). Manifest-level checks are treated as readiness by default; point
them at a real readiness endpoint (for example `/ready`, not `/health`) when
the service distinguishes the two.

## Declared Degradation

A resource that can run in more than one mode (GPU/CPU device, model size,
engine fallback) must:

- expose the active mode on an info/status surface an operator and a consumer
  can query
- report running below its configured mode as visible degraded status, not as
  silent fallback

Silent mode switches are the most expensive failure class in fleet history:
they keep health green while quality or correctness quietly collapses.

## Acceleration and Capacity

A resource that runs work on an accelerator declares it once, in the
`acceleration` block. There is no separate `gpu` block, no `requirements.gpu`,
and no top-level `capacity` block: those three surfaces gave two contradictory
answers to "does this resource use the GPU", and the control plane rejects any
manifest that still carries one.

```json
"acceleration": {
  "backends": ["cuda", "cpu"],
  "require": "preferred",
  "cuda": { "min_compute": "8.9", "env": { "MY_DEVICE": "cuda" } },
  "cpu":  { "env": { "MY_DEVICE": "cpu" } },
  "claim": { "resource_kind": "vram", "...": "..." }
}
```

The claim lives *inside* the block, so a resource cannot reserve VRAM without
declaring the backend it needs it on — which is what used to hide a resource
from the broker's planning. A `vram` claim must carry `yield_when_idle` and a
`profile` whose last step equals `floor_bytes`; without them the broker can
never ask the resource to step down, and the reservation is a number nobody can
act on.

Every resource answers the broker with the same verb:

```
<resource-cli> capacity degrade --to <label>
```

What a rung means is the resource's own business. How the broker asks is not.

`internal/accel` owns the rest: which backends the host can reach, which one a
resource is given, and which one it actually landed on. No resource carries a
private host probe, and no resource is trusted about its own placement.

## Startup Timeouts

`orchestration.startup_timeout_seconds` must budget the worst normal case,
including first-run model/image downloads. Use `startup_time_estimate` to
describe the warm-start experience; never tune the enforced timeout to the
warm case, or the orchestrator will kill healthy first starts.

## Related

- [architecture.md](architecture.md)
- [deployment-contract.md](deployment-contract.md)
- [storage.md](storage.md)
