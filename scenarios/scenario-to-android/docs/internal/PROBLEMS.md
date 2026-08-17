# Problems — Scenario to Android

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

### 2026-08-17 — Physical Android gate still needs a capable target

**Symptom:** The Galaxy A03s produces redaction-verified, decodable physical
evidence, but its `offline_transition` chapter is unavailable because the
handset does not advertise `network-control`.

**Root cause:** The handset lacks a device-level network-control capability;
this is not a capture-redaction failure. Device Control now owns and verifies
the default redaction policy before publishing evidence references.

**Workaround:** The matrix-wide gate records the physical target for the
chapters it proves and allows an emulator or another capable target to satisfy
`offline_transition`. If no target has the capability, the gate remains closed.

**Real fix:** Run the unchanged matrix with a target that advertises
`network-control` and retain its satisfying target identity.

**Owner:** repository owner / device operator.

**Refs:** `api/internal/conformance`, `packages/delivery-ramp-go/validationmatrix`,
`scenarios/device-control/docs/concepts/REDACTION.md`.

### 2026-08-10 — No Google account state exists, and the verification deadline is close

**Symptom:** Every `distribution` channel except ADB internal, and every `readiness` rung, is currently unsatisfiable. No Play Console registration exists and no developer verification has been performed.

**Root cause:** Not yet done; both are owner actions requiring identity and payment, and neither can be automated.

**Workaround:** The readiness ladder reports each rung honestly with its next action. ADB internal distribution remains available and needs no account.

**Real fix:** Owner completes Play Console registration and developer verification. Note the dates: Play requires target API 36 for new apps and updates from **31 August 2026** (extension to 1 November 2026), and developer verification begins enforcement on certified devices in Brazil, Indonesia, Singapore, and Thailand on **30 September 2026**, going global through 2027 — and it gates sideloading, not only Play.

**Owner:** repository owner.

**Refs:** `S2A-P0-009`, `docs/concepts/INTEGRATIONS.md`.

### 2026-08-10 — Wireless ADB flakiness would surface as failed conformance chapters

**Symptom:** Anticipated, not yet observed. Wireless debugging pairing expires on device reboot and the link is less reliable than USB, so a transport fault presents as a failed journey chapter rather than as a transport error.

**Root cause:** The failure surfaces at the wrong layer — a dropped `adb` connection mid-journey is indistinguishable from an app that stopped responding.

**Workaround:** Support both transports but default release-relevant runs to USB, so release evidence cannot fail for network reasons.

**Real fix:** Classify transport loss as a distinct, non-product failure class in evidence, so a reviewer can tell a flaky link from a broken app.

**Owner:** unassigned.

**Refs:** `S2A-P1-001`, `docs/concepts/FLOWS.md`.

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

**Workaround:** None that is safe in the scenario. Targeted Android tests and
the retained physical-device evidence remain usable; fresh server-owned
scenario validation must wait for the control-plane migration/remediation.

**Real fix:** The resource/control-plane workstream must retire the stale
container through an auditable Vrooli lifecycle path and converge or reset the
regenerable Redis cache under the invoking user's ownership. Do not add
scenario-local repair logic or manually change ownership.

**Owner:** resource/control-plane workstream.

**Refs:** fresh runs `20260816-212537-b5a43d34` and
`20260816-212539-6ee9d837`; scenario-qa `knw-1786915919029734855`;
`resources/redis/resource.json`.

### 2026-08-17 — Workflow provider has no shared validation response

**Symptom:** Fresh run `20260817-035220-b6bec178` passed 20/21 phases, but the
workflow phase failed with `durable provider terminal run has no shared
validation response`.

**Root cause:** The shared Test Genie workflow provider is being changed by a
parallel workstream and returned a maturity-contract error after the scenario
phases had completed; the Android workflow implementation itself had no
reported findings.

**Workaround:** Treat the run as non-passing. Use the 20 passing phase results,
focused tests, and retained physical evidence without inferring an integrated
PASS.

**Real fix:** Restore workflow-health's shared validation response contract,
then rerun the server-owned Android suite.

**Owner:** Test Genie maintainers.

**Refs:** run `20260817-035220-b6bec178`; workflow artifacts; plan
`mobile-delivery-ramps-implement-scenario-to-ios-complete`.

### 2026-08-17 — Fresh Phase 9 suite cannot enter Test Genie

**Symptom:** `vrooli scenario test scenario-to-android` builds its 21-phase
plan but is rejected before run creation with
`resource_exhausted: test-genie admission is saturated (caller queued run
capacity)`.

**Root cause:** Shared Test Genie admission/worker capacity is saturated while
another agent changes the test-genie scenario; this is not an Android product
failure.

**Workaround:** Use the passing Android API/UI/build/requirements checks and
the retained physical Galaxy A03s matrix evidence. Do not claim a fresh
Phase 9 integrated PASS until a run reaches terminal state.

**Real fix:** Restore Test Genie caller admission and rerun the unchanged
Android suite, then reconcile its findings manifest.

**Owner:** Test Genie maintainers.

**Refs:** rejected attempt `2026-08-17 05:53:48`; prior run
`20260817-053156-f4d27515`; plan `mobile-delivery-ramps-implement-scenario-to-ios-complete`.

## Architecture Drift

### 2026-08-17 — Physical Android unlock authority is stale

**Symptom:** The Galaxy A03s target is ready and discoverable, but the public
`android run` path cannot complete its physical install/cold-start gate.

**Root cause:** The active device-control profile returns `wrong_credential`
when asked to unlock the handset. The credential is authority-held and is not
available to the scenario.

**Workaround:** Keep the emulator required-cell gate and retained physical
review recording as the current evidence. The matrix fails closed rather than
claiming physical coverage.

**Real fix:** An authorized operator must provision or rotate the device-control
credential, then rerun the unchanged `hello-mobile` matrix.

**Owner:** Device-control credential authority/operator.

**Refs:** `run-a0c09a943fb950543dcf51fb`, `run-f22e9273e238472801819a9b`, and
`device-control auth provision`.

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Whole scenario | The seven declared product domains are implemented under `api/internal/{builds,targets,journeys,readiness,distribution,releases}` and composed through the declared handlers. The old template-domain drift is resolved. | Architecture and source ownership are now aligned; remaining constraints are device capability and integrated Test Genie/provider evidence. | Keep new platform behavior in the declared domains and promote shared behavior into the delivery spine rather than recreating it in the ramp. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
