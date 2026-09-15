# Problems — Scenario to iOS

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Append entries as they appear. The entries below were recorded during
scenario authoring on 2026-08-10, before any code existed — they are
constraints discovered while designing, not defects introduced by it.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-08-17 — Shared lifecycle cannot reload the current iOS API binary

**Symptom:** The source-level iOS route contract is corrected and the API
binary builds with `/api/v1/ios/*`, but the running service still serves the
previous `/ios/*` binary. `vrooli scenario restart` and `stop` fail before
replacing it because the control plane cannot read
`resources/sherpa-onnx/resource.json`.

**Root cause:** A concurrent shared resource migration removed or relocated a
resource manifest required by the lifecycle port allocator. This is outside
the iOS scenario and is not an Apple-toolchain limitation.

**Workaround:** Use the route regression test, generated endpoint manifest,
and targeted API/CLI tests as source evidence. Reload the scenario through the
normal lifecycle after the shared manifest is restored; do not run the API
binary directly.

**Real fix:** Restore the authoritative `sherpa-onnx` resource manifest or
make the control plane tolerate its intentional removal, then restart
`scenario-to-ios` and rerun the public iOS CLI probes.

**Owner:** Control-plane/resource-migration workstream.

**Refs:** `resources/sherpa-onnx/resource.json`,
`api/handlers/*/handler.go`, `api/handlers/*/endpoints.go`, and
`vrooli scenario restart scenario-to-ios`.

### 2026-08-16 — Test Genie admission and control-plane build are degraded

**Symptom:** The server-owned iOS suite run `20260816-192247-0f265c85` remained
queued because preview admission was saturated. The normal `vrooli scenario
test` path also could not rebuild while concurrent work left `internal/hostreqkit`
and test-genie module metadata inconsistent.

**Root cause:** Shared test/control-plane changes are being developed by a
parallel agent during this delivery-ramp implementation.

**Workaround:** Validate changed behavior with package, API, CLI, UI, and
targeted build tests; retain the durable run identity and rerun the
server-owned suite once admission and module freshness recover.

**Real fix:** Restore control-plane compilation and let the queued run reach a
terminal verdict before claiming suite-level acceptance.

**Owner:** test-genie/control-plane workstream.

**Refs:** `20260816-192247-0f265c85`, `internal/hostreqkit/manifest.go`.

### 2026-08-16 — Shared validation host has a stale Redis container after resource migration

**Symptom:** Fresh server-owned Android and iOS runs fail before phase
execution when their dependency startup reaches Redis. The control plane
refuses the Redis data directory because it is owned by UID 999, while the
invoking user is UID 1000.

**Root cause:** A legacy Docker container (`vrooli-redis-resource`) still
bind-mounts the per-user Redis cache while the working resource manifest is
being migrated to the managed-service driver. The managed-service storage
ownership guard correctly fails closed rather than adopting foreign-owned
state.

**Workaround:** Linux can still validate the iOS API, CLI, UI, and the
unsupported/unavailable boundary. Fresh server-owned iOS validation must wait
for the control-plane migration/remediation.

**Real fix:** The resource/control-plane workstream must retire the stale
container through an auditable Vrooli lifecycle path and converge or reset the
regenerable Redis cache under the invoking user's ownership. Do not add
scenario-local repair logic or manually change ownership.

**Owner:** resource/control-plane workstream.

**Refs:** fresh runs `20260816-212537-b5a43d34` and
`20260816-212539-6ee9d837`; scenario-qa `knw-1786915919029734855`;
`resources/redis/resource.json`.

### 2026-08-17 — Phase 9 admission is saturated after the iOS PASS

**Symptom:** Fresh Phase 9 attempts for device-control and hello-mobile remain
`queued` with no progress since creation (`20260817-090402-bb45848c` and
`20260817-094636-3bbc63e0`); Android and desktop retries are rejected with
`resource_exhausted: test-genie admission is saturated`.

**Root cause:** The shared Test Genie caller-queued worker capacity is saturated
while its scenario is being changed by another agent. This is independent of
the iOS implementation: the fresh iOS run `20260817-083715-3875a9d7` passed
all 21 phases before the saturation appeared.

**Workaround:** Keep the iOS Linux frontier validated by the fresh PASS and
targeted tests; do not infer suite PASS for queued or rejected Phase 9 runs.

**Real fix:** Restore Test Genie admission/worker progress, then rerun the
Phase 9 mobile-stack and desktop suites and synchronize their findings.

**Owner:** Test Genie maintainers.

**Refs:** runs `20260817-083715-3875a9d7`, `20260817-090402-bb45848c`,
`20260817-094636-3bbc63e0`; plan `mobile-delivery-ramps-implement-scenario-to-ios-complete`.

### 2026-08-10 — The only available macOS node has a dated release capability

**Symptom:** `minimouse` is `darwin/amd64` — Intel. It can validate iOS indefinitely, but its ability to produce **submittable** builds expires.

**Root cause:** macOS 26 Tahoe was the last Intel-supporting release, and Xcode 27 dropped Intel entirely. Apple's current App Store Connect floor — Xcode 26 with the iOS 26 SDK, mandatory since 28 April 2026 — is still satisfiable on this node. When Apple raises the floor again, projected around April 2027 from their annual cadence, it will not be.

**Workaround:** Structural, and already reflected in the design: the build host is modelled as a **bridge-node role** rather than a named machine, and validation capability is tracked separately from release capability. A host that can validate but not produce submittable builds reports that as a distinct state.

**Real fix:** Register an Apple Silicon node before the floor moves. Because the host is a role, that is a node registration rather than a rewrite.

**Owner:** repository owner (hardware acquisition).

**Refs:** `S2I-P0-004`, `docs/concepts/ARCHITECTURE.md`.

### 2026-08-10 — Xcode state on `minimouse` is unknown and not exposed by any CLI

**Symptom:** Every P0 target depends on a macOS node with Xcode 26 or later, correct simulator runtimes, and a logged-in GUI session. None of that has been verified on the live node, and no `vrooli-bridge` subcommand reports it.

**Root cause:** Node capability reporting covers OS, architecture, and scopes but not installed developer toolchains. The bridge agent *does* install on macOS as a LaunchAgent bootstrapped into the `gui/<uid>` domain and explicitly documents that SSH-only sessions have no GUI bootstrap — so the prerequisite is understood in code, but its live state is unobserved.

**Workaround:** The `targets` domain probes the node directly rather than trusting a bridge capability field, and reports each failure with its own distinct reason.

**Real fix:** Verify on the live node, then decide whether toolchain probing belongs in this ramp permanently or should be promoted into bridge node capabilities.

**Owner:** unassigned.

**Refs:** `S2I-P0-003`, `scenarios/vrooli-bridge/agent/internal/service/service_install.go`.

### 2026-08-10 — No Apple Developer Program enrollment exists

**Symptom:** Signing, physical-device provisioning, TestFlight, and the App Store are all unsatisfiable. `ios-xcuitest` is also blocked, because WebDriverAgent cannot be signed without an enrollment.

**Root cause:** Not yet done. Enrollment is $99/yr and requires an Apple ID, 2FA, and identity verification; organization enrollment additionally needs a D-U-N-S number. None of it can be automated.

**Workaround:** Two real paths remain open. Simulator conformance needs no enrollment at all. For physical hardware, the `ios-mirror` strategy needs no developer account — which is exactly why it is worth having, despite being pixel-grade.

**Real fix:** Owner enrolls. The readiness ladder reports every dependent rung with enrollment as the next action until then.

**Owner:** repository owner.

**Refs:** `S2I-P0-009`, `S2I-P1-001`, `S2I-P1-002`.

### 2026-08-10 — Mirror-derived evidence must never gate a release, and that rule needs enforcement

**Symptom:** The `ios-mirror` strategy is the first usable physical-iPhone path, but it reads pixels through OCR rather than semantics. It cannot distinguish a correctly-rendered screen from a convincing image of one, and it has no stable element identity to assert against.

**Root cause:** A structural property of the approach — iPhone Mirroring plus synthetic HID events plus OCR — not a defect to be fixed.

**Workaround:** The `releases` domain records promotability per contributing cell, and mirror-derived evidence is marked non-promotable.

**Real fix:** None; this is permanent. The requirement is that the non-promotion rule is *enforced in the gate*, not merely documented — a forbidden-transition case in the release flow contract.

**Owner:** this scenario, at implementation time.

**Refs:** `S2I-P0-010`, `S2I-P1-002`, `docs/concepts/FLOWS.md`.

### 2026-08-17 — Physical iOS capture remains hardware-gated

**Symptom:** iOS simulator and physical-device capture cannot produce runtime
evidence on this Linux host, while the shared producer redaction policy is now
implemented and verified by `device-control`.

**Root cause:** No qualifying macOS bridge, iOS simulator, WebDriverAgent
runtime, or Apple signing identity is registered. Whole-screen mirror evidence
also remains advisory and non-promotable by design.

**Workaround:** Keep simulator/bridge paths terminally unavailable or
unsupported with named next actions, and use the shared producer policy before
any future capture reference is emitted.

**Real fix:** Register the macOS bridge and Apple runtime, then exercise the
existing adapters without changing the matrix, evidence, readiness, CLI, or UI
layers.

**Owner:** repository owner / bridge operator.

**Refs:** `docs/reference/apple-hardware-activation.md`,
`scenarios/device-control/docs/concepts/REDACTION.md`, `S2I-P0-010`.

### 2026-08-17 — Apple-hosted `hello-mobile` conformance remains unavailable

**Symptom:** The shared `hello-mobile` fixture and its Android proof exist, but
the iOS end-to-end conformance path cannot run on this Linux host.

**Root cause:** The iOS fixture requires a registered macOS bridge with an
iOS simulator, WebDriverAgent-capable runtime, and the Apple toolchain.

**Workaround:** Keep the Linux project-generation, target-probing, readiness,
and terminal-unavailable journey checks green. Android physical evidence remains
the available mobile fixture proof.

**Real fix:** Register a trusted macOS bridge, then run the iOS fixture through
the unchanged twelve-chapter journey and retain the producer-owned evidence.

**Owner:** repository owner / bridge operator.

**Refs:** `scenarios/hello-mobile/`, `S2I-P0-012`,
`docs/reference/apple-hardware-activation.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Whole scenario | The seven declared product domains are implemented under `api/internal/{builds,targets,journeys,readiness,distribution,releases}` with shared handler composition. The remaining gap is Apple-hosted runtime evidence, not an unowned source tree. | Linux-testable implementation is covered; native iOS runtime and signing remain unavailable until the bridge and Apple credentials exist. | Register the macOS bridge and rerun the existing adapters; do not move matrix, evidence, readiness, CLI, or UI ownership into the adapter ledger. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

### 2026-08-18 — No scenario can authenticate to vrooli-bridge

**Symptom:** With bridge discovery correctly wired, the iOS target inventory
reports a bridge row that is `unavailable` with reason
`bridge inventory unavailable: unauthenticated`. `POST
/vrooli.vrooli_bridge.v1.registry.NodeRegistryService/ListNodes` returns HTTP
401 `{"code":"unauthenticated"}` even though the bridge API is healthy and
`nodes doctor` reports the macOS node dispatchable.

**Root cause:** vrooli-bridge accepts three authorization schemes —
`BreakGlass`, `LocalSession` (enrollment-based, `OS1.<claims>.<sig>` signed by
an enrolled client keypair), and owner `Bearer`. All three are *owner* or
*enrolled-client* identities. There is no service identity a co-located
scenario can present, no scenario is enrolled, and `VROOLI_BRIDGE_API_TOKEN`
— the variable the shared spine reads — appears nowhere else in the
repository. Scenario-to-bridge authentication has never existed, so the bridge
transport has never been exercised end to end by any delivery ramp.

**Scope:** Not iOS-specific. `scenario-to-android` and `scenario-to-desktop`
construct the same client and hit the same 401. Android is unaffected in
practice only because its devices attach locally through `device-control`
rather than through bridge discovery.

**Workaround:** None that preserves the trust boundary. Do not embed an owner
token in a scenario environment: owner credentials are not service
credentials, and doing so would give every ramp full owner authority over the
fleet. The honest state is the one now reported — an `unavailable` bridge row
naming the cause.

**Real fix:** An identity decision, not a wiring fix. Either enroll each ramp
as a `LocalSession` client with narrowly scoped capabilities, or issue
scenario-scoped service credentials through `scenario-authenticator` and teach
bridge to validate them. Whichever is chosen must scope authority to the verbs
a ramp actually needs (`host inventory` to probe, plus its validation verbs)
rather than granting owner-equivalent access.

**Owner:** unassigned — needs an identity/delegation decision from the
repository owner.

**Refs:** `scenarios/vrooli-bridge/api/internal/auth/middleware.go`,
`scenarios/vrooli-bridge/api/internal/auth/auth.go` (`ValidateLocal`),
`packages/delivery-ramp-go/validationmatrix/bridge_client.go`, and
`knw-1787034912555784148`.

### 2026-08-18 — A bridge node cannot report its Apple toolchain until its control plane is updated

**Symptom:** A darwin node answers `host inventory --json` over dispatch, but
the payload contains no `xcodebuild`, `xcrun`, or `simctl` entries and no
`apple_toolchain` probe status.

**Root cause:** Toolchain detection was added to `internal/hostinventory` in
this change. A node runs whatever revision it was provisioned to; `minimouse`
is on `78c6be32`, which predates it.

**Workaround:** The ramp distinguishes this from a genuinely absent toolchain.
An unprobed node reports missing capability `host toolchain probe` with the
next action "update the node's control plane so it reports Apple toolchain
facts", rather than `xcodebuild` — so nobody installs Xcode on a machine that
already has it. `TestDiscoverDistinguishesUnprobedToolchainFromAbsentToolchain`
pins the distinction.

**Real fix:** Provision the node to a revision containing the toolchain probe
(`vrooli-bridge provision sync <node> --revision <rev>`). Note this requires
the change to exist in a git revision, which the repository owner's standing
no-git rule reserves to them.

**Owner:** repository owner (provisioning).

**Refs:** `internal/hostinventory/toolchain.go`,
`packages/delivery-ramp-go/validationmatrix/host_capabilities.go`.
