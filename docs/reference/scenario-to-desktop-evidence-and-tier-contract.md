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

| Capability | Provider / route | Machine assertions | Visual evidence | Status |
|---|---|---|---|---|
| Baseline launch and Hello Desktop interaction | Desktop app → local fixture API | usable window, semantic greeting, input, geometry, clean shutdown | versioned chaptered journey + useful MP4 | Implemented on native Linux when Xvfb/openbox/xdotool/ffmpeg are available |
| Bundled private dependency | verified bundle manifest → desktop supervisor → private service | artifact digest, startup ordering, native readiness, operation response, shutdown | dependency readiness, operation result, provider chapter | Contract and supervisor implemented; native fixture journey is required before a release claim |
| Tier 2 → Tier 1 thin-client | desktop scenario API → configured Tier 1 API | route class, response identity, no direct resource endpoint | route and result chapters | Supported as a deployment mode; communication journey remains environment-dependent |
| Tier 2 → Tier 1 shared resource | consented broker → scoped lease → resource operation | provider identity, lease expiry, readiness, broker has no lifecycle mutation | provider/lease/result chapters | Provider selection, fallback, and redacted observation implemented; requires a live broker fixture for release evidence |
| Private fallback | shared provider unavailable → verified private artifact | fallback decision, private digest, readiness, no expired credential reuse | fallback chapter and failure reason | Implemented and tested |
| Tier 2 → Tier 2 peer | `[node/]scenario[@variant]` → `vrooli-bridge` relay → node-local scenario transport | authenticated versioned protocol, scoped authority, cancellation, loss isolation | both-side request/response evidence | Relay route implemented for bounded scenario calls; the ten-item peer bar below remains required before a full peer capability is claimable. |

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
never sends video bytes or credentials to governance. A visual pass requires a
persisted journey, useful decoded MP4, ordered chapters, and verified
redaction. A missing video offset is shown as missing; it is never invented.

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
explicit target disposition. Remote desktop streaming remains outside the
bridge dispatch seam until its separately owned transport contract exists.

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
`/home/matthalloran8/.vrooli/plan-artifacts/cross-ramp-delivery-spine/oracle`.
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
