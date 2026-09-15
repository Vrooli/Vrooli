# Testing — Vrooli Bridge

How to write tests against this scenario's shape. Read this *before*
your first non-trivial test — the patterns below are load-bearing for
the gates documented in [`SEAMS.md`](SEAMS.md), in `eslint.config.js`,
and in `.github/workflows/test.yml`.

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

Existing local entry points (choose the domain test for the behavior you change):

- [api/handlers/health/handler_test.go](../../api/handlers/health/handler_test.go)
- [ui/src/App.test.tsx](../../ui/src/App.test.tsx)
- [ui/src/api/health.test.ts](../../ui/src/api/health.test.ts)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## Multi-node soak

The acceptance soak is owned by [`scripts/soak.sh`](../../scripts/soak.sh). It
dispatches typed jobs to the Linux and macOS node IDs, restarts the control
plane only through `vrooli scenario restart`, injects randomly selected agent
kills and bounded network partitions, and continuously queries the durable
SQLite run table. The invariant is fail-closed: no queued or running run may
remain past `timeout_seconds + grace`.

Run it only after the operator has supplied both real node IDs, SSH hosts,
agent-kill commands, and partition enter/restore hooks. The default window is
24 hours; `--dry-run` validates the shape without touching a host. The script
writes the window, fault count, late-run count, fault log, and terminal-state
distribution to `docs/internal/SOAK-REPORT.md`. A dry run or a partial window
is evidence that the harness is configured, not evidence that the acceptance
invariant passed.

The shape is mature on purpose: every pattern below was already needed
in workspace-sandbox and got there by accumulating bugs. Starting here
means inheriting those lessons without repeating them.

## TL;DR — the canonical examples

These files are the source of truth. When in doubt, copy their shape:

- **API**: `api/handlers/health/handler_test.go` — table-driven, real
  middleware via `httpx.NewLiveServer`, fake pinger from `mocks/`,
  typed-proto decode via
  `assertx.MustUnmarshalProto[healthv1.Response]` (the wire shape lives
  in `packages/proto/schemas/vrooli-bridge/v1/health/health.proto`;
  assert on typed proto fields, not `map[string]any` chains). For
  endpoints whose wire shape isn't in proto yet, `MustDecodeJSON[T]`
  is the fallback — but adding the proto first is the right move.
- **UI composition**: `ui/src/App.test.tsx` — smoke-only composition
  test. App composes shell + features; feature behaviour belongs beside
  the feature.
- **UI health API**: `ui/src/api/health.test.ts` — typed health
  responses and failure handling at the API boundary. The current
  `HealthCard.tsx` feature is covered through the dashboard and shell
  composition tests listed above.
- **UI a11y**: `ui/src/components/AppShell.a11y.test.tsx` and
  `ui/src/features/health/HealthCard.a11y.test.tsx` — shell and feature
  accessibility are tested at their ownership boundary.
- **CLI**: `cli/app_test.go` — smoke gate (NewApp, --version, --help).
  When domain commands arrive, extend with `clitest.NewAPIServer` +
  `clitest.CaptureStdout` from `cli/internal/testutil/`.

If your test doesn't look like one of those three, ask why before
shipping.

## Qualification evidence harness (Phase 2)

The certification plan uses one bounded evidence vocabulary. A row is evidence
about a named target and candidate; it is not a readiness declaration by
itself. The statuses are `passed`, `failed`, `stale`, `unavailable`, and
`pending`. Historical rows without a current receipt stay `unknown` in the
dossier and are never upgraded from a phase label.

### D01–D17 traceability ledger

This is the current owner map. `planned` means the artifact or proof is not yet
delivered; it is intentionally not a passing claim.

| Deliverable | Observable obligation | Owner | Evidence producer | Status |
|---|---|---|---|---|
| D01 | PRD, requirements, support matrix, and prior-obligation ledger agree | Bridge | requirements validation, Test Genie business/docs, Plan Manager | in progress |
| D02 | One target projection exposes operation-specific readiness | api-core / Bridge | typed API tests and target receipts | verified in Phase 3–4 scoped validation |
| D03 | Pairing, grants, revocation, and privilege tiers are atomic and explicit | Bridge / control plane | security and integration receipts | verified in Phase 6 scoped validation |
| D04 | Onboarding is target-addressed, resumable, and schema-rendered | Onboarding / Bridge | onboarding operation receipts | verified in Phase 8 focused and scoped API evidence; Plan Manager producer validation retained as stale due concurrent source identity mutation |
| D05 | Enabled, important, and core policy choices produce the declared behavior | Onboarding / autoheal | policy tests and native receipts | planned |
| D06 | Complete and selective choices produce one deterministic closure and verified shipment | Onboarding / dependency analyzer / delivery | closure, digest, and activation receipts | planned |
| D07 | Linux, macOS, and Windows native lifecycle is proven on declared targets | Bridge / control plane | native host receipts | pending native targets |
| D08 | Durable operations recover with truthful uncertainty | Bridge | restart, cancel, and reconciliation receipts | planned |
| D09 | Credential delivery is scoped and audit readback is tamper-evident | credential owners / Bridge | credential and audit receipts | planned |
| D10 | VPS lifecycle preserves transport accountability and parity | scenario-to-cloud | owner operation receipts | planned |
| D11 | Attached mobile adapters and physical-device journeys are proven | platform ramps / Bridge | device host and device receipts | pending devices |
| D12 | UI, CLI, and API drive the same semantic journeys | Bridge / Web Console / Onboarding | Test Genie and BAS receipts | planned |
| D13 | Release receipts are applicable and promotion refuses mismatches | delivery-ramp-go / Deployment Manager | signed receipt and promotion decision | planned |
| D14 | Removal, retention, restore, diagnostics, and budgets are measured | owning domains | recovery and measure receipts | planned |
| D15 | M01–M10 fault, soak, and independent security evidence exists | Test Genie / platform owners | native matrix and independent review | pending prerequisites |
| D16 | Replaced paths are retired with consumer and behavior evidence | all touched owners | retirement ledger and regression diff | planned |
| D17 | Final dossier is complete and Plan Manager accepts the handoff | Plan Manager / Deployment Manager | certification index and final receipt | planned |

### Controlled target inventory and missing inputs

Inventory is captured from the current host and owner CLIs. Absence means
unknown or pending; it does not authorize a substitute target.

| Target/cell | Observed capability | Required authority or input | Current disposition |
|---|---|---|---|
| `target-local-linux-amd64` / M01 | Current host: Linux 7.0.0-30-generic, x86_64; SSH and Bridge are available | Explicit disposable-target grant and restoration procedure | available for controlled qualification, not yet run |
| `target-linux-vps` / M02 | No owner-supplied VPS identity in this inventory | VPS identity, SSH authority, restricted-ingress policy, restore point | pending operator input |
| `target-macos-arm64` / M03 | No macOS host or SSH identity observed | Real Apple Silicon host, SSH authority, GUI/headless prerequisites | pending operator input |
| `target-windows-amd64` / M04 | No Windows host or service authority observed | Real x64 host, administrative service-install authority, restore point | pending operator input |
| `target-android-physical` / M05 | `adb` is installed; no attached device identity recorded | Physical device, USB/debug authorization, host adapter grant | pending device and grant |
| `target-ios-physical` / M06 | `xcrun` is not installed | Physical device, macOS host, trust/developer prerequisites, signing authority | pending host, device, and signing |
| `target-android-emulator` / M07 | No emulator identity recorded | Disposable emulator definition and reset hook | pending owner inventory |
| `target-ios-simulator` / M08 | No simulator identity recorded | macOS host, simulator runtime, reset hook | pending host and runtime |
| `target-second-desktop` / M09 | No second fixture identity recorded | Fixture-alpha/beta ownership and shared dependency contract | pending fixture owner |
| `target-full-install` / M10 | No clean target receipt recorded | Clean target, complete-selection digest, restore procedure | pending controlled target |

The current host facts were collected with `uname -srm`, `go env GOOS GOARCH`,
and command discovery for `ssh`, `adb`, `xcrun`, and PowerShell. Exact grants,
credentials, signing material, and production identities do not belong in this
document; record only their opaque authority reference in a receipt.

### Evidence index and negative controls

The private plan artifact directory owns the execution index at
`/home/matthalloran8/.vrooli/plan-artifacts/vrooli-bridge-deployment-foundation-certification/evidence/`.
Each cell receipt follows this bounded shape:

```json
{
  "schema_version": 1,
  "cell_id": "M01",
  "target_id": "target-local-linux-amd64",
  "candidate_digest": "sha256:<64 lowercase hex>",
  "journey": "first-touch-subset-reboot-revoke-remove",
  "status": "pending",
  "evidence_refs": [],
  "producer": "test-genie|git-control-tower|owner",
  "captured_at": "<RFC3339>",
  "limitations": ["No native receipt has been captured yet."]
}
```

The index must preserve failed, stale, unavailable, uncertain, and not-
applicable rows separately from passed rows. Required negative controls are:
different candidate digest, replayed generation, revoked target during an
operation, missing receipt artifact, and an unapproved destructive target.
Every negative control must leave the controlled target at its recorded restore
state before another case begins.

### Generic shared-dependency fixtures

Fixture-alpha and fixture-beta are intentionally generic scenario identities
owned by their respective fixture owners. Both declare the same dependency
`fixture-shared-runtime-v1`; alpha additionally declares `fixture-alpha-data`,
while beta declares `fixture-beta-data`. The harness must show that a complete
or selective closure contains the shared dependency once, and that removing
alpha does not remove it while beta still owns it. No fixture is treated as a
real target until its owner supplies an identity, reset hook, and restore
receipt.

### Governed qualification command

The recurring composition is `vrooli-bridge.qualification`. It validates all
target authority before any binding call, then delegates validation admission
to Test Genie. Test Genie owns the durable operation and idempotency key; the
program returns only a bounded receipt identity and one resume/wait command.
`plan_only=true` is safe for contract inspection and refusal tests.

```bash
program-runtime library run vrooli-bridge.qualification --input 'request_key=bridge-m01-attempt-1,candidate_digest=sha256:<64 lowercase hex>,target_ids=["target-local-linux-amd64"],authorized_target_ids=["target-local-linux-amd64"],authorization_ref=grant-ref,phases=["business","storage","contracts"]' --json
```

Repeat the exact request key and candidate inputs to attach to the same owner
operation. Reusing that key with another candidate or target set must return a
typed owner conflict; it must not create a second operation. On a non-terminal
receipt, run the returned Test Genie wait exactly once. To resume after a lost
client, use the returned `test-genie validation get <receipt-id> --json` command.
The program never performs native target mutation; later phases add the
target-specific owner bindings and postconditions.

### Restoration contract

Before any destructive fixture case, record the target identity, active
generation, selected closure digest, and a restore-point reference. After a
pass or failure, verify the generation and shared-dependency ownership from the
target owner. If restoration is not verified, the next case remains `pending`
and the target is not reused. Production targets are refused unless a durable
explicit authority reference names both the target and the permitted mutation.
