# Verify the stage before entering it

Write `safety-report.md` before invoking lifecycle or population commands. Begin
with target, forbidden live targets, and the operator's remote control constraints.
Read `docs/architecture/presentation-instances.md` and the current implementation.
A named instance is not a sandbox for host effects.

| Boundary | Required evidence |
|---|---|
| Runtime | Live/presentation status, ports, process identities; read-only baseline |
| Builds | Effective source paths, component output paths, shared package provisioning, concurrent consumers |
| Startup graph | Required and optional dependencies, resource startup policies, unhealthy/stale restart behavior |
| Discovery | Follow list, explicit URL overrides, whether missing variants fail closed |
| Storage | Actual repositories and resolver paths; do not infer isolation from manifest labels |
| Host effects | Collectors, automatic investigations, hooks, remediation, background schedulers, GPU use |
| Privacy | Each filmed surface's data owner and sensitive fields; approved crops or omission |

Lifecycle dependency startup can target **live** even when dependency discovery
follows `presentation`. `try_start` is not a no-restart policy. Healthy dependency
reuse and unhealthy dependency restart are different branches. Shared build
admission protects same-scenario outputs; it does not prove that common package
builds are harmless to other scenarios.

Start a variant only when every effect in its actual startup path is within the
user's authority and protected services remain available. Do not alter critical
service manifests, disable safeguards, falsify freshness, or run binaries directly
to get around an unsafe path. A start refusal does not authorize restarting live.
Do not stop other GPU tenants to fit a music/image model.

Classify every source as presentation-owned fiction, real host observation, or
shared service data. Seed fictional application content through supported CLI/API,
UI, or ingestion interfaces. Never manufacture hardware facts and present them as
observations. Never stimulate an actual host incident for a promotional shot.
If no safe seed path exists, cut the shot or use a clearly labeled explanatory
visual, and leave the product-proof requirement unsatisfied when applicable.

Before the first write, confirm the exact presentation port via lifecycle status.
Before delivery, inspect captured frames for names, paths, identifiers, tokens,
private infrastructure, and errors. A label does not make sensitive data public.
After capture, stop only production-owned instances. Compare protected process
identities and health against the baseline without mutating them. Record exactly
what started, what stayed shared, and what was not attempted.

## Own the production tools

Record each new browser, preview server, encoder and model job before using it.
Keep PID and start-time identities for local tools. Keep the owner job handle for
governed services. Record the tool path/version, purpose, output directory and
cleanup action. A pre-existing process is not owned by this production.

Start local rendering with the owner's bounded profile and one capture worker.
Record the actual memory/task limits, deadline and CPU policy; relative CPU weight
is priority, not a quota, and capture-worker count does not bound native decoder
threads. Record sampled task pressure, process-tree peak RSS and elapsed time.
State the sampling method and unknown observations. Use the owner's cancellation
and diagnostics when pressure or stalls occur; do not keep raising ceilings.
Crop oversized source media before composition when the crop preserves the shot.
Disable persistent frame caches for one-off renders, or record their size and owner.

Close capture browsers and temporary servers in `finally`. On timeout or failure,
stop only the tool's owned process group and verified descendants. Compare PID
start times before terminating a retained descendant. Use the governing lifecycle
command for scenario/resource jobs; never kill a shared service to free capacity.
Verify that owned processes are terminal before delivery. Retain the final process
audit, protected-service comparison and install/disk register with the media.

These receipts prove ownership and cleanup, not a hard memory ceiling. Promote
repeated process supervision to the existing media/capture owner instead of
copying private daemon management into each production.

## Distinguish the three capture environments

- A named presentation instance isolates ports and supported data namespaces. It
  does not copy code or necessarily isolate dependencies and automatic effects.
- `git-control-tower baseline start` creates a development engagement and can
  stand up shadow/live instances and invoke validation. Inspect its effects; it
  is not a universal safe clone or an exemption from protected-service rules.
- An isolated frontend fixture capture copies the actual UI source/build, supplies
  authored responses at its existing network boundary, and denies all egress. It
  proves UI interactions only. Use it for operator-authorized mock-data work when
  relevant claims are strictly UI-scoped. Do not call it an instance or full-stack
  evidence. Validate responses with generated schemas, hash unmodified UI sources,
  and record the browser audit. Deny unfixtured calls rather than forwarding live.

This third path must not weaken normal product proof: no fabricated investigation
results, implied backend execution, or synthetic hardware presented as observed.
Record fictional-data provenance in the report and share context. Respect the
operator's rejection of persistent demo/mock badges. Narrow or omit a claim
whose meaning would otherwise mislead. Reuse installed toolchains and write
static build outputs only into the copied tree; no shared package provisioning.
