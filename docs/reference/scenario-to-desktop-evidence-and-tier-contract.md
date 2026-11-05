# Scenario-to-desktop evidence and tier contract

This is the canonical contract for desktop journey evidence and communication
claims. A recording or a successful window launch is not communication proof.
Every supported claim requires a machine assertion and reviewer-visible
evidence. Compile/package checks are reported separately from native runtime
evidence.

The provider-neutral contract owner is the shared Go module
`packages/delivery-ramp-go`: it owns the journey/evidence schema, disposition
rules, target inventory, validation matrix, transports, and reference-only
verdict semantics. `scenario-to-desktop` remains the desktop ramp and owns the
Electron, X11, xdotool, ffmpeg, capture-store, and HTTP/CLI/UI adapter code.

The Tier 1 and Tier 2 names in this document are **technical deployment tiers**.
They are not the commercial monetization tiers described in
`docs/monetization/strategy/TIERS.md`.

## Packaging strategy prerequisites

The shared packaging strategy contract may declare `capabilities.host_tools`
for strategy-specific prerequisites. Scenario-wide lifecycle manifests must
not mark those tools as required: doing so blocks every strategy before one is
selected. The desktop build preflight checks only the selected strategy's
requirements for the requested platform. Electron therefore remains buildable
without WebKitGTK, while the Wails in-process strategy checks its Linux
WebKitGTK and WebKitWebDriver prerequisites when selected.

## Terminology

| Term | Meaning |
|---|---|
| Tier 1 | A local Vrooli control plane and its locally managed resources. |
| Tier 2 | A generated desktop application and its user-scoped runtime. |
| Bundled private | The desktop supervisor starts a verified resource artifact selected by the immutable bundle plan. |
| External-server / thin-client | The desktop shell calls a configured Tier 1 scenario API; it does not directly own or expose the server's resources. |
| Shared resource | A consented broker lease to an already-running Tier 1 resource. The broker authorizes use; the desktop supervisor does not control its lifecycle. |
| Tier 2 peer | Another desktop runtime addressed through the authenticated `vrooli-bridge` relay. The relay route is implemented for bounded, scoped scenario calls; the full peer capability remains evidence-gated. |

## Capability matrix

### Capability selection

The producer MUST select a journey in this order:

1. Use an explicit `S2D_JOURNEY_CAPABILITY` selection when it names a
   registered fixture.
2. Use the deployment-mode baseline for the target.
3. Use the scenario-specific registered fixture when the scenario declares
   one.
4. Use `desktop.launch.baseline`.

The selected capability, deployment mode, and selection reason MUST be written
to both the journey sidecar and the evidence manifest. An unsupported or
unavailable selected capability MUST remain visible; the producer MUST NOT
silently fall back after execution starts.

| Capability | Provider / route | Machine assertions | Visual evidence | Status |
|---|---|---|---|---|
| Baseline launch and Hello Desktop interaction | Desktop app → local fixture API | usable window, semantic greeting, input, geometry, clean shutdown | versioned chaptered journey + useful MP4 | Certified by phase-16 pipeline `f65aeaf2-51f0-386f-a8e8-cf270ac35d7f` (`smoke-hello-desktop-1789551829909`) |
| Bundled private dependency | verified bundle manifest → desktop supervisor → private service | artifact digest, startup ordering, native readiness, operation response, shutdown | dependency readiness, operation result, provider chapter | Certified by phase-16 Web Console pipeline `e8cb60e0-a6ec-70fc-39bf-16b09904cc00` (`smoke-web-console-1789552894526`) |
| Tier 2 → Tier 1 thin-client | desktop scenario API → configured Tier 1 API | route class, response identity, no direct resource endpoint | route and result chapters | Supported as a deployment mode; pending phase-16 certification run ID |
| Tier 2 → Tier 1 shared resource | consented broker → scoped lease → resource operation | provider identity, lease expiry, readiness, broker has no lifecycle mutation | provider/lease/result chapters | Provider selection, fallback, and redacted observation implemented; pending live-broker certification run ID |
| Private fallback | shared provider unavailable → verified private artifact | fallback decision, private digest, readiness, no expired credential reuse | fallback chapter and failure reason | Implemented and tested; pending phase-16 certification run ID |
| Tier 2 → Tier 2 peer | `[node/]scenario[@variant]` → `vrooli-bridge` relay → node-local scenario transport | authenticated versioned protocol, scoped authority, cancellation, loss isolation | both-side request/response evidence | Evidence-gated; pending a certifying run ID after the ten-item peer bar below is met. |

Unsupported and unavailable are terminal evidence states, not degraded passes.
An unavailable environment must include the missing capability and next action.

## Evidence contract

The producer persists a `journey-evidence.v2` sidecar. Each chapter contains a
stable ID, purpose, action, bounded readiness policy, settle policy,
monotonic and wall-clock timestamps, expected and observed values, assertion
status, capture references, and optional recording-relative offsets. Readiness
events and settle events are part of the same timeline.

The manifest stores capture references, checksums, media-derived metadata, the
timeline version/chapter IDs, ordering verification, and redaction status. It
never sends video bytes or credentials to governance. Every planned chapter
MUST occur in the sidecar, manifest, CLI projection, and UI. A chapter that did
not execute MUST have disposition `not_run`; the producer MUST retain its ID,
purpose, action, assertion, and reason. A missing video offset is shown as
missing; it is never invented.

## Runtime request routing

For a bundled application, the UI service MUST serve bundle assets and the
health path from the bundle. It MUST route every other request to the bundled
API service.

The routing rule MUST NOT name a scenario, enumerate a path-prefix allowlist,
or match a protocol-specific path shape. A navigation request MAY receive the
bundle's application shell when the request accepts HTML. An API request MUST
NOT receive the application shell.

## Child process environment

A bundled child MUST receive only the allowlisted environment variable names
and the variables declared by its bundle manifest. The runtime MUST remove all
other inherited names before it starts the child.

The default allowlist is:

| Name | Reason |
|---|---|
| `HOME` | Libraries need the user-scoped home for non-secret caches. |
| `USER` | Tools use the logical account name for ownership and diagnostics. |
| `LANG` | Child processes need the requested locale for stable text behavior. |
| `LC_ALL` | Operators can force a deterministic locale for evidence. |
| `TZ` | Time formatting must use the configured timezone. |
| `TMPDIR` | Tools need the runtime's temporary workspace. |
| `DISPLAY` | Native Linux children need the assigned display. |
| `XAUTHORITY` | Native Linux children need the assigned display credentials. |
| `PATH` | Executables need the platform tools and bundled binaries. |
| `SSL_CERT_FILE` | TLS clients need the platform certificate file when configured. |
| `SSL_CERT_DIR` | TLS clients need the platform certificate directory when configured. |

## Isolation observation

Each bundled child MUST report the paths and session adoption it resolved at
readiness. The run MUST retain an isolation observation containing `state_root`,
`socket_path`, `database_path`, `adopted_session_count`, and its `source`.

`screen_content_source` MUST be derived from the isolation observation. A run
with no observation MUST use `unknown`. If a child resolves a path under the
operator home or the live session root, the run MUST fail with
`live_state_resolved` and MUST name the resolved path.

## Journey assertions

An input chapter MUST fail with `surface_unchanged` when its before and after
captures are identical and no application-level assertion passed.

A chapter MUST fail when its captured frames contain an application surface
error. The observed value MUST carry the error text. A journey MUST NOT report
`pass` while a surface error is visible.

Field redaction reports only inspection of journey and manifest fields. The
manifest MUST also report `screen_content_source` for recorded pixels, with one
of `isolated_instance`, `bundled_runtime`, `operator_declared_safe`, or
`unknown`. A passed manifest MUST NOT use `unknown`. Screen-content inspection
MUST NOT be inferred from field-redaction status.

Each required gate MUST use its own signal:

| Gate | Independent predicate |
|---|---|
| `protocol_readiness` | The smoke protocol emits its success marker and the producer records the matching readiness event. |
| `visual_launch` | The target application window is usable, the demo trace contains `main_window_shown` and `app_ready`, and the recording has useful frames. |
| `semantic_journey` | The selected journey reaches `pass` and every required chapter passes its assertion. |
| `capture_integrity` | `ffprobe` and sampled-frame inspection prove a decodable, non-empty recording with useful frames. |
| `artifact_persistence` | The journey and manifest are persisted, checksummed, and cross-referenced by non-empty capture IDs. |

No gate MAY copy another gate's disposition or use a constant success value.

### Bounded desktop-cell claim

A passing desktop BAS cell proves only that the selected provider-owned journey
executed against the packaged application, that the required machine and
application assertions passed, and that the resulting journey and capture
artifacts are bound to the same run, target, artifact, and cell identities.
It does not prove an untested deployment tier, a live peer transport, or a
release claim for a capability whose matrix status does not include a
certifying run ID. Phase-16 certification is the authority for replacing the
pending status text above with a run ID.

### Smoke target isolation

The smoke target MUST be an isolated instance. A thin-client run MUST start or
discover a named isolated Tier 1 instance and MUST target its runtime-reported
URL. A bundled run MUST target only ports reported by its private runtime. The
producer MUST refuse a live Tier 1 port and report
`live_instance_target_refused` when the target matches it.

If an isolated instance cannot be started or identified, the run MUST fail with
`isolated_instance_unavailable`. A probe that identifies another application
MUST report `probe_app_mismatch`; a probe that cannot identify the application
MUST report `probe_app_unidentified`. The producer MUST tear down the isolated
instance after the run and retain its identity in the evidence.

### Voided evidence

When a capture is unsafe, superseded, or otherwise invalid, the producer MUST
mark it void with a reason and `superseded_by` when applicable. It MUST NOT
delete the capture. Voided evidence is excluded from release-gate selection but
remains available for audit.

## Validation ownership and matrix surfaces

Desktop validation is a composition of provider-owned evidence and
delivery-ramp-go-owned release-gate semantics, with scenario-to-desktop as the
desktop adapter. The workflow provider discovers and runs
semantic scenario workflows; it publishes a generic execution reference and
redacted, checksummed artifacts. Scenario-to-desktop binds that reference to a
desktop cell, adds target/runtime/machine evidence, and computes the
fail-closed release gate. Test Genie is only the generic validation-run
orchestrator and must not discover, configure, or probe BAS or any other
provider.

The scenario-to-desktop API exposes the durable matrix endpoints backed by the
shared spine at:

- `GET /api/v1/validation/targets` for the locally probed target inventory.
- `POST /api/v1/validation/matrices` and `POST .../{run_id}/start` to create
  and dispatch an immutable selection.
- `GET .../{run_id}`, `GET .../{run_id}/wait`, and `POST .../{run_id}/abort`
  for inspect, reattach, and cancellation.
- `POST .../{run_id}/rerun` for selector-scoped new runs; prior evidence is
  never overwritten.
- `GET .../{run_id}/compare/{prior_run_id}` for stable cell-identity
  comparison across artifact or target runs.

Bridge targets may dispatch an allowlisted durable job and preserve node,
job, run, and artifact identity. Dispatch alone does not imply desktop
execution, renderer access, or video evidence; those capabilities remain an
explicit target disposition.

### Remote desktop validation transport

Remote desktop validation is batch validation of a built artifact on a target
node with captured evidence. It does not provide live remote desktop
streaming, interactive renderer access, or a VNC-like session over Bridge.
Remote streaming remains out of scope for this contract.

A remote cell MUST perform these steps:

1. Place the named artifact on the Bridge target.
2. Verify the artifact digest on the target node.
3. Dispatch the typed desktop-artifact validation verb.
4. Upload one bounded evidence bundle from the target-owned run.
5. Retrieve the bundle through the Bridge run-artifact service.
6. Expand the bundle into typed evidence on the caller side.

The expanded bundle MUST provide `KIND_DESKTOP_RUNTIME`, `KIND_TARGET`,
`KIND_MACHINE_ASSERTION`, and `KIND_BAS_WORKFLOW` evidence. The target command
currently encodes these JSON members in a bounded gzip/tar archive named
`evidence-bundle.tar.gz`; the archive is an implementation detail of the
Bridge output binding, not a streaming protocol. Every evidence item MUST
contain non-empty `evidence_id`, `uri`, and `sha256` values and MUST set
`redacted` to true. The caller computes `sha256` from retrieved bytes; it MUST
NOT trust a digest reported by the target.

On macOS, the target validator MUST apply only artifact-scoped launch
preparation before opening the placed application: remove the artifact's
`com.apple.quarantine` attribute when present and apply an ad-hoc signature
with `codesign --force --deep --sign -`. A read-only disk image MUST be copied
to a private temporary staging location before those operations. The target
MUST NOT change system Gatekeeper policy (including `spctl --master-disable`),
and the preparation result belongs in the target-owned runtime trace.

Cross-OS ownership is deliberately split:

| Question | Owner |
| --- | --- |
| Does scenario source pass on each target OS at a revision? | Bridge `GateService` via deployment-manager `crossosgate` |
| Does a built desktop artifact work on a platform? | `validationmatrix` release gate |
| Is the local Linux desktop artifact good? | `validationmatrix` local cell |

`unavailable` means the target could not be reached or lacks a required
capability. `failed` means validation ran and did not pass. These dispositions
MUST remain distinct and MUST NOT collapse into one another.

Verdicts use these labels consistently: `pass`, `failed`, `degraded`,
`unavailable`, `unsupported`, and `not_run`.

## Profile and target-capability contract

Environment profiles are typed enum values, not free-form labels. The
scenario-to-desktop API publishes their provider-owned contract at
`GET /api/v1/validation/profiles`, including the required target capabilities.
The matrix keeps a selected cell applicable when a capability is missing, then
records `unsupported` with the exact missing capability so the release gate
cannot silently omit the cell.

The profile inventory covers normal, offline, slow/high-latency/packet-loss
network, reconnect, provider failure/unavailable, missing/expired/unavailable/
wrong-scope credential, update discovery/download/verification/interruption/
rollback/restart/failure, and bundled-private/Tier 1/shared-provider/fallback/
Tier 2-peer communication. Target capability checks include network control,
credential control, provider control, updater/update-feed, crash recovery,
communication peer, native surfaces, process metrics, and clean shutdown.

The local Linux target currently advertises only capabilities it can prove at
inventory time. Profiles requiring an unimplemented adapter therefore remain
visible and produce an evidence-backed `unsupported` disposition; they are
never converted into a passing normal run.

## Ownership and secret boundary

The dependency analyzer and bundle manifest own resource selection. The
supervisor owns only private bundled service lifecycle. The broker owns shared
authorization and leases. Evidence may expose provider tier, safe route class,
service identity, readiness, artifact digest, fallback decision, and lease
expiry; it must not expose endpoints, bearer tokens, generated operator
configuration, or secret values.

## Peer status

The intended route is the node-axis resolver over `vrooli-bridge`: a caller
addresses `[node/]scenario[@variant]`, Bridge admits the typed command against
the node's scopes, and the node-agent executes it through the node's own local
transport. The control plane never dials a scenario port on the peer.

The bounded relay path is implemented and exercised by the Bridge suite and a
live minimouse status call. That is not, by itself, a full Tier 2 peer claim.
Before peer capability evidence can be marked pass, the route must meet this
ten-item bar with both-side evidence:

1. discovery;
2. identity;
3. authentication;
4. capability negotiation;
5. scoped authority;
6. retry and timeout;
7. cancellation;
8. replay protection;
9. failure isolation; and
10. shutdown.

Until that bar is met, the capability disposition remains evidence-gated rather
than a release-ready peer claim. `tier2-desktop-peer` is a precedence rank in
the shared-provider selector, not a transport implementation.

## Shared-spine extraction evidence

The reference-ramp extraction was validated against the durable oracle in
`~/.vrooli/plan-artifacts/cross-ramp-delivery-spine/oracle`.
Pipeline `50e8c5fb-882e-d5cf-61eb-86b900b8eb4b` completed the bundled Linux
build and native smoke test for `hello-desktop` with smoke test
`smoke-hello-desktop-1786376651446` and recording capture
`109c91f4-20b9-4c46-a557-dd4ac423ae1b`.

The rerun journey capture
`9c08207c-b610-4319-8709-798dddb5f39a` matched the oracle journey
`0c3d1061-5996-4ea9-aba6-73ece0a39961` after normalizing run timestamps,
capture IDs, and monotonic/video offsets: schema `2`, evidence version
`journey-evidence.v2`, capability `hello-desktop`, profile `normal-review`,
pass disposition, the same eight ordered actions, eight passed steps, and
the same 33-event type sequence. The rerun manifest preserved all five
required gates, the target/runner contract, ordered chapter IDs, verified
redaction, and all four artifact kinds. Dynamic capture sizes, checksums,
durations, and references are intentionally run-specific.

Rerun artifact checksums are recorded here for auditability:

- journey JSON: `ebcd740828afa4e59b28c4a6e813bd622407a62f0d32916e896dfec12adbc7ad`
- evidence manifest: `6e4de2ae5ef30d74b7301b6dd70378977d862755e12437e15ffd473b02ad6bcc`
- MP4 recording: `167665a260fabac768fd06a1b15977eebd63e7163e9ab7d0198a8fc3af6d42b3`

The recording decoded as H.264 at 1920x1080 with 266 frames and 17.733333
seconds. This proves the shared journey and manifest contracts survived a
fresh native reference-ramp execution; it does not make a provider-specific
visual recording byte-for-byte deterministic.
