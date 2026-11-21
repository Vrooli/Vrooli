# QEMU proving-ground qualification (phase 20)

The QEMU proving ground must exercise all 266 required matrix case/lane pairs
on resettable fresh hosts, including systemd, privilege, headless credential
initialisation, reboot and failure recovery. `qemu` is also one lane in that
matrix; its lane manifest contains 54 rows. Containers are not a substitute
(no systemd, no host-bound credential wraps, no privilege broker), and
operator machines are never failure-injection fixtures.

Everything in this guide is owner-managed. Nothing here installs a package or
downloads an image on its own.

## What the lane consists of

| Artifact | Path | Role |
|---|---|---|
| Lane manifest | `certification/lanes/qemu.json` | The 54 required `(case, qemu lane)` cells with fixture, fault, architectures, budgets and operator prerequisites. `api/certification/lanes_test.go` keeps it equal to the matrix. |
| Image manifest | `certification/lanes/qemu-images.json` | Approved base images with path, sha256, os and arch. Empty until EXT-06 is closed. |
| Readiness report | `GET /api/v1/instances/readiness` (`api/instance.Readiness`) | Typed host observation: tools, KVM, images, architectures, one next action. |
| Qualification program | `.vrooli/program-runtime/cloud-qemu-qualification.{json,py}` | QEMU-only ground binding; admits every matrix case/lane pair explicitly applicable to QEMU, not only the qemu lane. Returns standing, cell identities, evidence refs and next action. |
| Receipts | `certification/evidence/<CASE>.<LANE>.qemu.<content-sha256>.json` | One immutable target-bound receipt per case, lane and ground. Historical receipts remain unbound and do not satisfy current readiness. |
| Evidence | `~/.vrooli/plan-artifacts/scenario-to-cloud-professional-vps-delivery-certification/evidence/P20-qemu-qualification.md` | Phase standing and acceptance table. |

## Operator steps

### 1. Review and install the three QEMU host tools

`qemu`, `qemu-arm`, and `cloud-localds` are declared in
`scenarios/scenario-to-cloud/.vrooli/service.json` under `hostTools`. Vrooli host
requirements are the only install boundary; do not run `apt`, `sudo`, or any
other package manager by hand.

```text
vrooli host install qemu --dry-run --json
vrooli host install qemu-arm --dry-run --json
vrooli host install cloud-localds --dry-run --json
```

The reviewed Linux/apt package plans are respectively `qemu-system-x86`,
`qemu-system-arm`, and `cloud-image-utils`. Apply only the named tools, after
confirming each preview:

```text
vrooli host install qemu --sudo-mode ask
vrooli host install qemu-arm --sudo-mode ask
vrooli host install cloud-localds --sudo-mode ask
```

Each command may require operator authentication. This avoids applying unrelated
root host tools or safeguards from a global setup run. The readiness report
requires four binaries owned by these three tools: `qemu` supplies
`qemu-system-x86_64` and `qemu-img`, `qemu-arm` supplies
`qemu-system-aarch64`, and `cloud-localds` supplies `cloud-localds`. Accelerated
amd64 guests also need `/dev/kvm` to be openable by the service user (the `kvm`
group); without it the lane still runs as a real VM under TCG emulation.

### 2. Acquire an approved base image and record its digest

Acquire the pinned amd64 and arm64 Ubuntu 24.04 cloud images in the CL2 A2 packet from Canonical’s released-image channel. Verify the signed `SHA256SUMS` using Canonical’s Ubuntu cloud-image keyring, verify each image, store both read-only, and record their paths and hashes:

```json
{
  "schema_version": 1,
  "images": [
    {"path": "/var/lib/vm/ubuntu-24.04-server-cloudimg-amd64.img", "sha256": "<verified 64 hex>", "os": "ubuntu-24.04", "arch": "amd64", "source_url": "<distribution URL>", "recorded_at": "<UTC timestamp>"},
    {"path": "/var/lib/vm/ubuntu-24.04-server-cloudimg-arm64.img", "sha256": "<verified 64 hex>", "os": "ubuntu-24.04", "arch": "arm64", "source_url": "<distribution URL>", "recorded_at": "<UTC timestamp>"}
  ]
}
```

The provider never downloads or mutates a source image: `vps instance create` makes an
owned copy-on-write `disk.qcow2`, and the digest is verified before the disk is
created and again after destroy (P20-A05). `VROOLI_QEMU_IMAGE_MANIFEST`
overrides the manifest path for the API process.

### 3. Read the readiness report

```text
curl -s "$API/api/v1/instances/readiness" | jq .
curl -s "$API/api/v1/instances/readiness?verify=1" | jq .images   # hashes every recorded image
```

| Field | Meaning |
|---|---|
| `state` | `ready` or `unavailable`. |
| `tools[]` | `name`, `present`, `path`, `declared_owner` (the `hostTools` entry that owns it). |
| `kvm_available`, `kvm_detail` | Whether `/dev/kvm` is openable, and why not. |
| `architectures` | `amd64` and `arm64`, each `kvm`, `tcg` or `unavailable`. |
| `images[]` | Recorded images with `present` and `digest_state` (`verified`, `mismatch`, `unverified`, `missing`). |
| `limitations[]` | Every reason the lane is not ready, naming the external-input row (EXT-05, EXT-06). |
| `next_action` | Exactly one operator action: `host_setup`, `record_image`, `repair_image` or `run_qualification`. |

An `unavailable` state is a report, not an error: the endpoint answers 200 and
the qualification program refuses to proceed on it.

### 4. Run the qualification program

Create the owned guest first (REST pre-step, see `docs/local-qemu.md`):
`vps instance create`, `wait-for-ssh`, `snapshot clean`, then bind a fixture
deployment to it whose target `machine_id` is the guest identity
(`local-qemu:<id>`). The program never creates or destroys the guest.

`deployment apply` is a destructive binding with `requires_confirmation`, so
the session needs the grant. `library run` creates a session without grants
and is the right way to exercise admission; for a real run, create the session
yourself and prepend the inputs to the source:

```text
program-runtime sessions create --name qemu-lane --grants effect:destructive --wall-budget 180s --json
printf 'inputs = %s\n' "{\"request_id\":\"cloud-certification-fixture-v1-amd64-qemu-001\",\"release_ref\":\"$RELEASE_REF\",\"target_ref\":\"local-qemu:lane-a\",\"environment_ref\":\"$ENVIRONMENT_REF\",\"workload_ref\":\"stateless-web\",\"authorization_ref\":\"$AUTHORIZATION_REF\",\"cases\":[\"RUN-01\",\"RUN-03\",\"DATA-05\",\"OPS-07\"],\"faults\":{\"allowed\":[\"worker_restart\",\"target_reboot\"]}}" | cat - .vrooli/program-runtime/cloud-qemu-qualification.py | program-runtime programs submit --session-id "$SESSION_ID" --provenance operator --source-file - --async --wait-timeout 180s --json
```

Admission refuses, before any read or mutation:

| Refusal | Class | Cause |
|---|---|---|
| absent grant | `no_authorization` | `authorization_ref` empty, or a non-zero `limits.spend_ceiling_minor_units` |
| incompatible target | `incompatible_target` | `target_ref` not `local-qemu:<id>`; `owned_target_count` above 1; no deployment bound to the selector or id (404); the deployment's target `machine_id` or environment differs from the request |
| stale artifact | `stale_artifact` | `release_ref` not an immutable `sha256:<64 hex>`; `matrix_revision` not `cloud-launch-v2-two-grounds`; current receipts must bind `target_ground=qemu` or `target_ground=real_vps` and a generation; closure not `derived`; the compiled plan's release digest differs from `release_ref` |
| unknown fault | `unknown_fault_capability` | a `faults.allowed` entry outside the lane manifest's `fault_capabilities` |
| case outside the lane | `invalid_input` | a case/lane pair absent from the matrix applicability map, or a target ground outside that pair |

Phases and bindings:

| Phase | Bindings | Effect |
|---|---|---|
| validate | none | admission only |
| collect | `deployment/get` or `deployment/resolve`, `deployment/plan`, `validate("scenario-to-cloud")` | resolves the deployment, checks it is bound to `target_ref`, compiles the plan, reads the latest recorded Test Genie run as evidence |
| decide | none | refuses a stale closure or release; names fault cells that still lack an arming binding |
| act | `deployment/apply` (destructive, `request_key = request_id`, confirmed), `operation/wait` (one bounded wait), `operation/target_receipt` (fresh target read for each successful target-owned step), `deployment/health`, `edge/status` | one admitted operation; a non-terminal operation is reported with its id, never as a failure |
| classify | none | `journey_verified` only when the operation succeeded, a target-owner receipt matches the operation/plan/step/target identity, and a current healthy observation names `release_ref` |

Standings: `refused`, `operation_pending`, `journey_failed`, `journey_unverified`,
`journey_verified`. The program never writes a `passed` receipt: on
`journey_verified` every requested cell is `pending_assertions` with the
operation id, and the next action is to collect per-case assertions. A missing
or mismatched fresh target receipt produces `journey_unverified`. Fault arming
(`api/faultinject`) still has no governed binding, so fault cells are named as
pending in `signals.fault_injection`.

**Current standing (2026-09-09).** `programs submit --explain` reports no
diagnostics. All eight fixtures behave as declared in test provenance, and a
read-only probe naming the production deployment id with a `local-qemu:` target
is refused as `incompatible_target` in `collect` (the production deployment has
no target `machine_id`), so the program cannot be pointed at a live host. No
apply has run: the host has no QEMU tools (EXT-06), no image, and therefore no
owned guest.

### 5. Read the receipts

Each cell's target-ground receipt uses the semantic prefix
`<CASE>.<LANE>.qemu`, where `LANE` is the selected matrix lane and the ground
is exactly `qemu` for this program. The `qemu` lane itself uses
`<CASE>.qemu.qemu`.
`certification.WriteReceipt` appends a content SHA-256 suffix to produce an
immutable filename. Its schema is v2 and binds the target ground, machine,
generation, architecture and OS. The shape is the registry's receipt schema
(`api/certification/evidence.go`): `case_id`, matrix-matching
`requirement_refs`, `verdict`, `lane: "qemu"`, `candidate`, `target`,
non-empty `operation_refs`, `validation_receipt_ref`,
`artifact_refs`, `assertions`, `observed_at`, `limitations`. A v2 receipt must
also carry at least one `operation:<id>` reference, a `test-genie:<receipt>`
validation reference, and requirement references that exactly match the matrix
row.

Readiness precedence per cell is failed > passed > unavailable > stale >
missing. An `unavailable` receipt keeps the cell open by name; it can never
satisfy certification. This program verifies one fixture deploy/health journey
plus the operation's fresh target-owner receipt and returns
`pending_assertions`; it does not write certification receipts. A separate
per-case owner assertion and target evidence are required before a v2 verdict
can be appended. The `real_vps` ground has a separate Bridge-fenced
qualification program and cannot reuse QEMU evidence. A failed receipt is
never masked by a later pass and must be withdrawn by its owner with a reason.

The arm64 lane is reported `unavailable` until `qemu-system-aarch64` (TCG on
this amd64 host) or an arm64 host exists (EXT-05). Certification stays
incomplete for arm64 until then (P20-A06); no substitute closes it.

## Budgets

From `certification/budgets.json`: lane timeout 7200 s, one owned target per
qualification, at most eight fault injections per run, spend ceiling 0, fresh-host
restore within 900 s, planned unavailability at most 60 s, zero unexplained
residual owned resources, owned ephemeral targets removed within one hour of the
terminal state. Cleanup policy is
`retain_failed_evidence_then_remove_owned_ephemeral_resources`.
