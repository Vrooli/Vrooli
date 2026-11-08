# Qualify and Mature Vrooli VPS Deployment for `landing-page-business-suite`

This assignment supersedes the earlier VPS qualification goal. It is deliberately
kept as an editable repository artifact because the transport, onboarding, and
deployment integration may expose additional blockers during qualification.

## Objective

Use the disposable test VPS at:

- Host: `157.245.87.154`
- SSH user: `root`
- OS: Ubuntu 24.04.4 LTS
- Architecture: `x86_64`
- Provider: DigitalOcean
- Nominal disk: 25 GB
- Available RAM: approximately 961 MiB
- Current Vrooli installation: none

to repeatedly qualify, repair, consolidate, secure, and document the complete
Vrooli deployment lifecycle for `landing-page-business-suite`.

The final outcome must be a mature, reliable, maintainable workflow suitable for
deploying the same scenario to the real production VPS. A single successful
deployment is insufficient evidence.

This is an adaptive qualification and improvement mandate. Let measured evidence
determine whether the next repair belongs in the contract, requirements,
validation/evidence, implementation, CLI behavior, documentation, onboarding,
`vrooli-bridge`, `scenario-to-cloud`, `deployment-manager`, `tunnel-manager`,
credential authority, or the VPS environment.

This is a campaign-level assignment, not a promise that one uninterrupted agent
session can complete every phase. The receiving agent must make the work
resumable and split it into owner-local work packages or a governed plan when
the change crosses scenarios, shared packages, infrastructure, security, and
real-VPS qualification. A continuation, handoff, or budget limit is not success.
Only the evidence-backed gates in this assignment can establish completion.

## Autonomous execution during the Swarm transition

`swarm-manager` is currently under active change and its present execution path
may require human approval even when this assignment already supplies explicit
authority for the disposable VPS and the in-scope repository work. Do not make
that transitional approval UX an implicit prerequisite for this campaign.

First qualify the current Swarm and Agent Manager capabilities. Use Swarm's
autonomous route when it can represent the assignment's authority, scope,
acceptance, budget, identity, continuation and evidence contract. If the only
failure is that the current Swarm implementation cannot autonomously grant or
continue already-authorized work, use a qualified direct Agent Manager or
harness route within this assignment's boundary. Record the missing Swarm
receipt, lost guarantees, route identity and recovery evidence. This is a
temporary execution fallback, not permission to invent a second ledger,
unregistered workflow, duplicate run, or fake Swarm approval.

Pause only the specific effect that genuinely requires an owner decision or
grant, including new scope, production access, paid spend, private data, or an
unapproved product/security decision. Continue independently authorized
qualification, diagnosis, implementation, testing, cleanup and documentation.
Do not use `swarm-manager-meta-orchestrator` as the sole execution path: it is a
human-led shaping session and does not itself run this campaign. Do not wait for
its approval loop when the assignment already authorizes the work.

## Managed finite coordination

The `landing-page-vps-qualification` Prompt Manager team remains disabled.
Its current entry point is
`/home/matthalloran8/.vrooli/plan-artifacts/efforts/landing-page-business-suite-vps-qualification/README.md`;
the team member heartbeat provides the bounded reconciliation and assignment loop.
Keep this adaptive qualification model and its existing evidence, not a parallel
product home, campaign ledger or shell scheduler.

The temporary wake driver is stopped. Its original instructions are retained
under that workspace's `archive/legacy-coordination-20261024/`; their shell
start/resume commands are historical, not the current execution path. Do not
restart or clone the driver when a native operation is unavailable. Reconcile
retained owner identities, current qualification receipts and applicable
decisions before deliberate admission through the existing owner route.

Each approved wake assesses current gaps, allocates substantial owner work,
reviews returned evidence and checkpoints or hands back. Park for a named
producer; stop with a precise return condition when only a real external input
or decision remains. A goal may bound a worker assignment; it does not replace
the managed coordinator. This navigation update does not activate the team,
qualify a new dispatch route or widen the assignment's effect authority.

## Critical transport decision

Do not conflate SSH reachability with Bridge-agent reachability. Qualify these as
two explicit, independently selectable transports:

### SSH transport

The control-plane host or `scenario-to-cloud` reaches the VPS over SSH. The VPS
does not need to connect back to Bridge. This is the valid one-way model for
first-touch, recovery, and an explicitly SSH-bound deployment.

### Bridge-agent transport

The VPS agent initiates and maintains an outbound connection to Bridge. Bridge
dispatches typed operations, artifacts, credentials, and run control over that
held channel. Bridge does not independently SSH into the VPS for normal
Bridge-agent operations. The physical network path is:

```text
VPS agent --HTTPS/SSE--> Bridge API / channel endpoint
Bridge    --frames over the held connection--> VPS agent
```

The VPS therefore must reach Bridge for pairing, heartbeats, presence,
dispatch, artifact delivery, and acknowledgement. It does not need an inbound
Bridge port for this mode.

The recommended final architecture is hybrid:

- SSH for first-touch, repair, and break-glass recovery;
- Bridge-agent transport for normal managed operations;
- explicit transport selection in deployment state;
- no silent Bridge-to-SSH fallback after a target is Bridge-bound.

If the product decision is instead SSH-only, document that as a deliberate
transport mode and do not claim that Bridge-agent qualification has passed.

## Authority and scope

The test VPS is disposable and may be modified extensively.

Allowed target-side actions include installing/removing software, configuring
systemd, SSH, sudo, firewall rules, users, service accounts, storage, swap,
rebooting, stopping/removing test deployments, cleaning artifacts, testing
interruption/recovery, taking/restoring snapshots, and reimaging when necessary
for a clean lifecycle test.

Do not touch the production VPS, production DNS, production databases,
production credentials, unrelated local scenarios, or unrelated dirty worktree
changes.

Before destructive provider actions:

1. Verify the exact target identity.
2. Capture sanitized evidence.
3. Verify recovery SSH access.
4. Take a snapshot when practical.
5. Record the action and recovery plan.

## Credential handling

Credentials are supplied through the secure execution context. Never print,
commit, journal, receipt, or persist passwords or private keys. Never put secrets
in shell arguments, normal files, issue records, or logs.

Use the approved credential authority, SSH agent, masked prompts, password-stdin,
or a short-lived root-owned mode-0600 file only when unavoidable. Do not invent a
second credential store. After key-based recovery is proven, rotate or disable
the temporary root password where the recovery design permits.

Bridge owner credentials, node credentials, scenario-to-cloud credentials, and
the VPS SSH credential are separate authorities. Do not substitute one for
another.

## Required operating doctrine

Before adding code, scripts, state, routes, or configuration:

1. Search the repository and capability inventory for an existing solution.
2. Search commands, skills, docs, programs, APIs, handlers, tests, and state stores.
3. Identify the owning scenario/control-plane layer.
4. Read the existing implementation and tests.
5. Ask whether the proper existing mechanism or configuration already solves it.
6. Prefer the canonical owner and consolidate duplicate paths.
7. Refactor shared logic into the correct owner when duplication is real.
8. Add code only when an existing extension point is insufficient.
9. Add desired-behavior regression evidence for every behavioral repair.
10. Update canonical documentation with the implementation.
11. Record rejected approaches and why they were rejected.

Do not monkey-patch, manually edit deployment databases, create parallel
onboarding/state/receipt systems, add one-off scripts that duplicate CLI
behavior, use raw package managers, run scenario binaries directly, bypass
ownership boundaries, weaken Cloudflare Access merely to pass a test, or lower
acceptance criteria because the VPS is small.

Use the scenario lifecycle commands (`make setup`, `make start`, `make status`,
`make logs`, `make stop`) and official scenario CLIs. Follow `docs/TESTING.md`
for validation scope and the server-owned Test Genie wait protocol.

## Sources to inspect before implementation

Read the relevant canonical docs and code before changing anything, including:

- `scenarios/vrooli-bridge/docs/operations/DEPLOYMENT.md`
- `scenarios/vrooli-bridge/docs/internal/TESTING.md`
- `scenarios/vrooli-bridge/api/internal/onboard/`
- `scenarios/vrooli-bridge/agent/internal/channel/`
- `scenarios/vrooli-bridge/api/handlers/channel/`
- `scenarios/vrooli-bridge/api/internal/dispatch/`
- `scenarios/scenario-to-cloud/docs/`
- `scenarios/scenario-to-cloud/api/reach/`
- `scenarios/scenario-to-cloud/api/domain/manifest.go`
- `scenarios/scenario-to-cloud/api/persistence/deployment.go`
- `scenarios/scenario-to-cloud/api/handlers_deployment.go`
- the existing Bridge/scenario-to-cloud transport integration plan

Use the scenario work ladder to identify whether the first broken layer is W0
contract, W1 obligations, W2 evidence, or W3 implementation. Do not write
downstream tests against a false contract.

## Initial known facts

The local environment currently has `vrooli-bridge`, `scenario-to-cloud`, and
`tunnel-manager` installed and healthy. An existing Cloudflare tunnel exists and
the public Bridge hostname is protected by Cloudflare Access.

The current local repository is dirty:

- branch: `agi`;
- revision: `9c9cc93e3e28`;
- approximately 206 modified paths;
- approximately 51 untracked paths;
- approximately 25 deleted paths.

Preserve these changes. Do not reset, checkout, clean, or otherwise destroy
unrelated work. Do not assume the dirty tree is a release-quality source.

`landing-page-business-suite` currently has an approximately 1.3 GB checkout,
requires PostgreSQL, and has API/UI, credentials, billing, storage, and
deployment behavior that must be qualified rather than stubbed.

The observed VPS baseline is Ubuntu 24.04.4, approximately 22 GB free on a 24 GB
root filesystem, approximately 961 MiB RAM, no swap, inactive UFW, SSH on port
22, root password login enabled, no Vrooli/Docker/Go/Node/pnpm detected, and
approximately 133 pending Ubuntu updates. Re-measure; do not treat these facts as
immutable.

## External inputs and completion gates

The following inputs are not safely inferable and must be supplied through the
approved secure/operator context or resolved through an existing owner:

- the final machine-agent endpoint and its owning exposure/authentication path;
- Bridge owner/API authentication and the credential-authority route that supplies it;
- an approved test edge domain and its DNS/TLS authority;
- a clean, pushed revision suitable for release-like qualification;
- any decision to use private networking, public machine TLS, mTLS, or another
  supported transport boundary;
- any production promotion approval, which is separate from this disposable-VPS
  qualification.

The agent may perform read-only discovery, baseline capture, and an explicitly
SSH-only qualification while an input is unresolved. It must not claim the
corresponding Bridge, public-edge, release, or production-readiness gate passed.
Do not guess credentials, domains, endpoints, or security policy. Report the
missing input and its exact blocking gate.

The final report must provide separate verdicts for:

1. SSH transport qualification.
2. Bridge-agent transport qualification.
3. LPBS functional qualification.
4. Resource/performance optimization qualification.
5. VPS security/recovery qualification.
6. Public test-edge qualification.
7. Production-promotion readiness.

The seventh verdict cannot be `production-ready` solely because the test VPS
passed. Production remains untouched by this assignment and requires a separate
controlled run against the production target with fresh authorization, current
configuration, production backups/recovery evidence, and an explicit approval.

## Run-scoped evidence and state

First discover whether an existing governed qualification, receipt, or state
mechanism owns this workflow. Reuse it when suitable. Otherwise create a
run-scoped, mode-700 control-host workspace such as:

```text
~/.local/state/vrooli/vps-qualification/<target>/<run-id>/
```

Use a separate transient target directory such as:

```text
/var/tmp/vrooli-qualification/<run-id>/
```

Remove target-side temporary state during cleanup. Sanitize all retained
evidence. Store operation IDs, digests, receipts, redacted logs, issue links,
and checkpoints, never credential values.

Capture the target identity, machine/boot IDs, addresses, OS, architecture,
kernel, disk/filesystem, memory/swap, routes, DNS, listening ports, firewall,
SSH policy, users/sudo, services, failed services, installed tools, pending
updates, Vrooli state, Docker state, and provider metadata where available.

## Phase 0: resolve blockers before mutation

Before installing a node or deploying LPBS, establish the intended transport
contract and resolve or explicitly document blockers.

### Machine-agent endpoint

The Bridge agent uses the API/channel endpoint `/api/v1/channel/events`, not the
Bridge UI. Verify the selected endpoint reaches Bridge API port `18767` and does
not return a UI page or a human Cloudflare Access login redirect.

Investigate existing reachability owners in this order:

1. Existing Bridge/tunnel configuration.
2. Tunnel Manager.
3. An authenticated machine-to-machine route.
4. Private VPN/network connectivity.
5. Another existing Vrooli transport owner.

Prefer a private network or narrowly scoped authenticated machine endpoint. Do
not disable human Access globally or add an unauthenticated public Bridge API.
If a public endpoint is necessary, use stable TLS, machine authentication,
redacted proxy logs, explicit exposure documentation, and end-to-end tests.

An IP may be used for SSH and can be used for a private Bridge endpoint. A public
Bridge endpoint should normally use a DNS name with a certificate; do not use a
bare public IP with insecure HTTP as the permanent design.

The endpoint preflight must distinguish DNS, TCP, TLS, authentication, wrong
service/UI response, and protocol failures. Test it from the VPS, not only from
the Bridge host.

### Bridge owner authentication

Use the canonical Bridge CLI and inspect current help before invoking it. Use
the documented owner login or token authority; do not create a parallel token
store. Verify the authenticated session with read-only commands before mutation.

### Target binding

Inspect and qualify the seam between Bridge enrollment and scenario-to-cloud
deployment target resolution. A manifest-derived VPS target currently defaults
to SSH, while Bridge execution requires a durable node ID/machine ID binding.

Implement or repair the owning public CLI/API path if necessary so that:

- SSH and Bridge are explicit transports;
- the binding includes machine ID, node ID, and enrollment generation;
- binding is inspectable before plan/apply;
- manifest refresh does not silently overwrite an explicit Bridge binding;
- explicit rebind is required to change transport;
- revoked/offline Bridge targets do not silently fall back to SSH;
- no database edits are required for normal operation.

Before writing new code, determine whether an existing target-binding owner or
CLI already exists. Extend it rather than adding a competing mechanism.

## Transport qualification A: SSH-only lane

Prove the one-way model independently:

- Bridge/control-plane host reaches the VPS over SSH.
- The VPS does not need to reach Bridge.
- First-touch can install key-based recovery access.
- The VPS can be prepared and LPBS can deploy through an explicitly SSH-bound
  target.
- No Bridge-agent presence is required for SSH deployment.
- SSH host-key verification, user scope, sudo scope, command typing, receipts,
  retries, interruption handling, and cleanup are reliable.

If testing with VPS egress restricted, use a controlled, reversible diagnostic
restriction and restore recovery access afterward. Do not weaken the permanent
firewall merely to make the test pass.

## Transport qualification B: Bridge-agent lane

Use the control-plane-driven onboarding path after verifying the exact current
CLI syntax:

```bash
vrooli-bridge onboard start \
  --host 157.245.87.154 \
  --user root \
  --port 22 \
  --control-plane-url <MACHINE_AGENT_URL> \
  --password-stdin
```

Use a pushed, clean, pinned revision for release-like evidence. Use a dirty
working tree only for a labeled diagnostic run. Do not assume an option exists;
inspect `vrooli-bridge help` and the owning docs first.

Observe durable operations once using the supported watch/wait mechanism. Do not
poll repeatedly. Verify completion, node identity, enrollment generation,
scopes, online state, heartbeat freshness, protocol compatibility,
dispatchability, agent service ownership, service user, sudo policy, revision,
readiness, disk, and temporary-file cleanup.

Then test repeated onboarding, agent restart, VPS reboot, Bridge restart,
temporary endpoint loss, reconnect backoff, protocol mismatch, revocation, and
re-enrollment. Confirm that Bridge reports typed offline/revoked/protocol errors
instead of masking them as SSH success.

## Small-host qualification

The VPS has only 25 GB disk and approximately 1 GB RAM. This is not merely a
compatibility constraint or a reason to resize the machine immediately. Treat
the envelope as the performance and efficiency target that should inform the
deployment architecture, shared deployment code, release format, dependency
closure, runtime configuration, retention policy, and future scenario work.

The optimization objective is:

```text
Required LPBS functionality and reliability
  within the 25 GB / approximately 1 GB VPS envelope
  with measured headroom for updates, recovery, and routine operations
```

The first response to pressure must be measurement and optimization, not a
larger VPS, disabled functionality, or a VPS-specific monkey patch. Improvements
should benefit future scenario deployments wherever possible. LPBS-specific
exceptions are acceptable only when the scenario genuinely has a unique need and
the exception is documented.

### Establish explicit budgets before optimizing

Before large mutations, measure disk, memory, swap, CPU, closure, resources,
images, build artifacts, network transfer, startup time, health latency, and
temporary peak usage. Establish a written budget for:

- base operating system and security/update reserve;
- Vrooli control/agent footprint;
- scenario release and extracted runtime footprint;
- PostgreSQL data, indexes, WAL, and maintenance overhead;
- build, upload, extraction, migration, and rollback peaks;
- active deployment generations;
- logs, caches, temporary files, backups, and recovery points;
- steady-state memory and update/recovery headroom.

Maintain a provisional minimum of 5 GB free disk after steady state and after a
normal update/rollback operation. Replace this provisional value only with
measured evidence and a documented rationale. Define a memory target that leaves
operational headroom and avoids sustained OOM pressure, thrashing, or an
unbounded swap substitute.

The budget is not satisfied by measuring only the final state. Capture the peak
working set and peak disk usage during build, transfer, extraction, migration,
restart, update, rollback, and cleanup.

### Optimize in evidence-driven order

Use one measurable intervention at a time and compare it with the recorded
baseline. Prefer, in order:

1. Remove duplicate state, duplicate release copies, obsolete dependencies,
   unused services, redundant images, and unnecessary build output.
2. Build on the control host when the deployment contract permits it and transfer
   a bounded, reproducible runtime artifact.
3. Reuse content-addressed or otherwise canonical artifacts instead of copying
   the same release into multiple owners or generations.
4. Bound deployment generations, logs, caches, temporary files, backups, and
   rollback retention through the proper owner.
5. Reduce runtime memory and database overhead through measured configuration,
   connection limits, indexes, worker counts, and cache sizes.
6. Improve startup, health checks, readiness, and update behavior so resources
   are not held unnecessarily or duplicated during transitions.
7. Compress or deduplicate artifacts where this reduces total peak and steady
   state usage without increasing unsafe CPU, latency, or recovery behavior.

For every intervention record the measured before/after disk, memory, CPU,
latency, reliability, deployment time, and recovery impact. Search for and reuse
the existing artifact, cleanup, dependency, storage, and retention owners before
adding another implementation. If two owners retain the same data, consolidate
the responsibility rather than merely adding a cleanup cron job.

### Required optimization gates

Do not proceed to a broader lifecycle claim until the current optimization gate
passes:

- required LPBS journeys remain functional;
- the steady-state budget passes with the documented headroom;
- update and rollback peak usage passes;
- repeated deployments do not cause unbounded growth;
- restart and recovery do not require manual cleanup;
- no OOM, corruption, timeout, or unacceptable latency regression is introduced;
- the improvement is implemented in the reusable owning layer, not only on this
  VPS;
- the measured result is recorded against the baseline.

Adding swap, removing optional functionality, using a smaller runtime profile,
or resizing the VPS is a mitigation or fallback, not the first optimization.
Each must be labeled as such, justified by measured evidence, and evaluated for
whether it masks a fixable inefficiency in Vrooli or LPBS.

Measure Vrooli base, agent, LPBS bundle, PostgreSQL, build peak, runtime, logs,
caches, deployment generations, rollback, and cleanup across pristine,
repeated, reboot, interrupted, update, rollback, and re-enrollment cycles.

Do not claim success by skipping required LPBS functionality. If optimization
cannot bring the complete required workflow within the envelope, report the
remaining limiting component, the interventions attempted, their measurements,
the reusable fixes made, and the exact reason a mitigation or larger VPS is
required.

## VPS security qualification

After alternate recovery access is proven, use the proper owning workflow to
evaluate and configure SSH key-only access, root-login policy, password policy,
sudo scope, service users, systemd ownership, firewall rules, security updates,
time synchronization, unnecessary services, log permissions, temporary files,
pairing material, agent IPC, and credential residue.

Expected public exposure is normally SSH (preferably source-restricted) and the
LPBS HTTP/HTTPS edge. Do not expose arbitrary application ports or an inbound
Bridge port on the VPS unless a documented transport contract requires it.

Never disable the only recovery path before proving its replacement.

## Scenario-to-cloud qualification

Use LPBS for all realistic deployment tests. Use a separately approved test edge
domain; the VPS may be identified by IP, but normal public HTTP/TLS edge behavior
requires a hostname.

Inspect current CLI help before running commands. The intended lifecycle is:

```bash
scenario-to-cloud manifest init \
  --scenario landing-page-business-suite \
  --host 157.245.87.154 \
  --domain <TEST_DOMAIN> \
  --out <QUALIFICATION_WORKSPACE>/cloud-manifest.json

scenario-to-cloud manifest doctor <MANIFEST>
scenario-to-cloud manifest validate <MANIFEST>
scenario-to-cloud preflight run <MANIFEST> --json
scenario-to-cloud deployment create <MANIFEST> --json
scenario-to-cloud deployment resolve \
  --scenario landing-page-business-suite \
  --environment production
scenario-to-cloud deployment plan <DEPLOYMENT_ID> --show-commands
```

Resolve every preflight failure. Before apply, verify target identity,
transport, node/machine identity, enrollment generation, release/closure/config
digests, scopes, secret requirements, artifact size, cleanup, downtime, and
receipt behavior.

Apply only a reviewed plan with the supported durable request identity. If the
observer disconnects, reattach to the existing operation; never replay an
effectful deployment because the caller timed out.

Verify target-produced receipts, observed release/configuration identity,
health freshness, edge status, public site, API, health endpoint,
authentication path, and LPBS baseline journeys.

Explicitly reproduce and document:

```text
Bridge enrollment
  -> Bridge node ONLINE
  -> scenario-to-cloud target resolution
  -> scenario-to-cloud plan
  -> scenario-to-cloud apply
  -> deployment through the selected transport
```

Inspect the resolved target after every deployment create/update. If an explicit
Bridge binding is reset to SSH, reproduce it, capture evidence, and repair the
owning implementation. Do not work around it with manual database changes.

## Required lifecycle matrix

Run and retain evidence for at least:

1. Pristine SSH qualification.
2. Pristine Bridge enrollment and deployment.
3. Repeated onboarding and deployment idempotence.
4. VPS reboot and agent reconnect.
5. Bridge restart and agent reconnect.
6. Interrupted operation and durable reattachment.
7. Temporary network loss and recovery.
8. Clean pinned-revision update and rollback.
9. Deployment retirement, artifact/log cleanup, node revocation, and re-enrollment.
10. A negative test proving Bridge-bound operations do not silently fall back to SSH.

For each cycle measure service count, deployment count, artifact count, disk,
memory, logs, credentials, receipts, and residual privileged state.

## Issue and debt classification

Classify every issue as one or more of VPS/environment, provider/network,
security policy, Bridge, onboarding, scenario-to-cloud, deployment-manager,
tunnel-manager, credential authority, CLI ergonomics, documentation drift,
test/evidence gap, duplicated responsibility, obsolete code, missing
consolidation, architectural defect, or capacity limitation.

For each issue record the title, severity, target/revision, reproduction,
expected/actual behavior, logs/receipts, owning layer, existing paths searched,
why they were insufficient, proposed permanent fix, rejected alternatives,
regression test, documentation impact, and production-blocking status.

Use existing problems/bug-report workflows and do not create duplicate issue
stores. A defect outside the active scope should be reported to its owner and
not silently repaired unless the current authority covers it.

## Repair and validation rules

Make maintainable owner-local changes only. Consolidate duplicate state,
transport, onboarding, artifact, cleanup, and receipt paths where evidence shows
they overlap. Remove obsolete paths when safe. Preserve compatibility only when
there is a documented consumer.

Use focused regression tests and the relevant Test Genie phases by default.
Run broader certification only when the impact or acceptance policy requires it.
Use the server-owned wait protocol exactly once for non-terminal runs.

After each material repair:

1. Run the focused regression.
2. Re-run the smallest affected qualification cycle.
3. Update canonical docs and CLI help where needed.
4. Record the evidence and remaining uncertainty.

## Success criteria

The work is successful only when:

1. SSH mode works without VPS-to-Bridge connectivity.
2. Bridge-agent mode works through a secure, documented outbound VPS-to-Bridge endpoint.
3. The endpoint reaches the API/channel service rather than the UI or human login.
4. Bridge enrollment, online state, heartbeat, dispatch, artifacts, and reconnect work.
5. Target transport and enrollment identity are explicit and inspectable.
6. Explicit Bridge bindings are not silently overwritten or downgraded.
7. Revoked/offline Bridge targets fail closed with typed errors.
8. LPBS deploys, updates, rolls back, and recovers through the intended transport.
9. Secrets are delivered without leakage or duplicate stores.
10. The optimized deployment meets the measured disk/memory/CPU/performance
    budgets for the 25 GB / approximately 1 GB envelope with documented headroom.
11. Build, update, rollback, and recovery peaks also fit the envelope.
12. Repeated operations do not create unbounded services, artifacts, logs, or generations.
13. Cleanup and re-enrollment leave no privileged residue.
14. The hardened VPS retains a tested recovery path.
15. The public test edge works with the approved test domain.
16. Optimization changes are reusable in future deployments and are not VPS-only hacks.
17. All remaining limitations and any mitigation/resize decision are explicit,
    measured, classified, and owned.
18. Evidence is sufficient for a production-readiness decision.

## Required final deliverables

Produce sanitized, linked artifacts for:

- baseline and final VPS inventory;
- transport decision and SSH-vs-Bridge qualification matrix;
- machine-agent endpoint and authentication design;
- secure onboarding runbook;
- secure deployment runbook;
- scenario-to-cloud target-resolution report;
- capacity, performance, retention, and peak-usage budgets;
- baseline-versus-optimized benchmark report;
- optimization decision log showing interventions attempted, measurements,
  reusable code changes, rejected shortcuts, and any remaining mitigation;
- lifecycle/recovery/rollback matrix;
- cleanup and re-enrollment report;
- security posture report;
- issue/debt ledger;
- deduplication/consolidation report;
- refactoring and changed-code summary;
- regression/Test Genie summary;
- documentation/CLI ergonomics summary;
- evidence index containing operation IDs, plan/revision digests, receipts, and health results;
- final readiness decision.

The final decision must distinguish `production-ready`, `usable with documented
limitations`, `test-only`, `blocked by missing capability`, and `unsafe/not
ready`. State separately whether SSH qualification and Bridge-agent
qualification passed.

## Checkpoint and stop rules

Checkpoint the run-scoped evidence before budget exhaustion or any context loss.
The checkpoint must state changed, verified, remaining, and unverified items.

When the work crosses an owner boundary or cannot fit one bounded session, create
or update the appropriate governed plan/handoff before continuing. The handoff
must name the current phase, evidence location, exact next action, affected
owner, unresolved decision, and validation still owed. Do not hide unfinished
work in a generic “follow-up” list.

`Blocked` means a decision, credential, or approval that is genuinely missing.
Diagnosable friction is not blocked. Do not claim completion from partial setup,
one successful deployment, or a green transport health check alone.

Do not report “ready for full real deployment” unless every applicable verdict
above has passing evidence and a separate production authorization exists. If the
campaign ends with a sound implementation but incomplete external qualification,
report `usable with documented limitations` or `test-only`, with the exact gate
that remains.
