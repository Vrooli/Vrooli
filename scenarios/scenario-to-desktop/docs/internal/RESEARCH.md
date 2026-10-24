# Research Notes

## Desktop release readiness review — 2026-09-07

Open the interactive visual assessment (preserved HTML).
It includes the capability map, proposed A/B ownership behavior, 15 journey
families, eight completion stages, source links and an embedded downloadable
observation snapshot. Scope: the operator's requirement #4, independent of the
first scenario selected for deployment. This was an investigation, not repair,
publication or release certification.

**Assessment:** substantial Linux execution/evidence infrastructure exists;
the full requested desktop release contract is not ready. There is no defensible
percentage-complete estimate. The contract gap is W0: signing/notarization is
still P1, Windows/macOS remain compile-only in OT-P0-003, and generic P0
obligations do not fully express cross-app lifecycle and remote desktop evidence.
No lower-rung gates were run under this gap. Existing results were inspected.

### Observed state

| Boundary | Evidence | Interpretation |
|---|---|---|
| Fault profiles | Live `GET /api/v1/validation/profiles`: 25 profiles, only Normal executable; agrees with `packages/delivery-ramp-go/validationmatrix/profile_contract.go` | 24 profile adapters remain implementation work; not a total completion percentage |
| Local target | `GET /api/v1/validation/targets`: linux/amd64 available with declared baseline capabilities | Native Linux/Xvfb path; no automatic physical-device claim |
| Bridge | Same inventory reports node-reach transport unauthenticated; `bridge_client.go` returns degraded for passed jobs lacking desktop evidence | Current integration unavailable and desktop transport incomplete; device connectivity is unknown |
| Native platforms | `api/livedesktop/platform_other.go` provides an unavailable backend | Windows/macOS need native desktop control/capture adapters, not just packages |
| Latest unit record | `20260907-214452-54b12663`, completed 2026-09-07 21:47 UTC, failed | Reports missing UI role discovery plus API tests, UI coverage and UI type-check command failures; root causes not established here |
| Broader record | `20260906-021110-123b1988`, comprehensive, failed | Failed contracts, dependencies, unit, storage, tidiness, security, programs (advisory), proto; API, quality and channel-conformance unavailable. Outcomes are historical and do not diagnose vulnerabilities |
| Native journey | Hello Desktop capture `b38422ce-8930-47d9-be7f-8afed84c434f`, 2026-09-06 05:25 UTC, v2/pass/eight actions | Producer JSON inspected; native behavior not re-executed or video re-audited |
| Combined evidence | `run-63a9781eaacc421f49fabd83` and `run-ca34229709796ab4ef200722`, Aug 9, each 2/2 required cells passed | Producer evidence exists; both retained `release_report.reported_at` values are zero, so these records do not demonstrate governance acknowledgment |
| Signing | Prerequisites command reports Linux GPG installed; maintained problems report missing release trust anchor | Tool availability is not signing identity, notarization, artifact trust or release approval; production authority not exercised |

The latest unit log was read through Test Genie's opaque artifact catalog:
`artifact_8d159139c3be6bdee3063df5c934f31b`; phase result:
`artifact_2c0c29151f45297795fb4ae297dc19cf`. The embedded snapshot retains
the inspected target/profile responses, recent run summaries, journey and
bounded matrix metadata. Many concurrent workspace changes limit exact-source
attribution. Search discovery was partially degraded; owner APIs and source
remained available.

### Architectural recommendation

Reuse `delivery-ramp-go` for matrix semantics, BAS for semantic workflows,
Bridge for authenticated transport, desktop adapters for native execution,
the credential authority for secret policy, and Deployment Manager for approval.
Keep host lifecycle and recovery in the control plane.

For A bundles B / standalone B joins, extend the existing broker/control-plane
substrate into stable user-scoped service ownership. A and B acquire use leases;
opening B attaches its UI instead of transferring process ownership. Compatibility
must include protocol, schema/data domain, configuration and trust scope. Reuse a
compatible instance; explicitly isolate or refuse incompatible ones. An absolute
one-version rule risks unsafe data sharing. Test launch orders, concurrency,
owner failure, upgrades, Tier 1 coexistence and uninstall with active clients.

### Completion recommendations

1. Align generic P0 release obligations and an explicit OS/architecture/installer
   support matrix with the operator's goal.
2. Diagnose applicable recorded engineering failures and restore trustworthy
   focused checks through Test Genie before release certification.
3. Implement and prove shared scenario/resource lifecycle with compatibility,
   scoped authority, atomic acquisition, lease renewal and fenced recovery.
4. Complete target-native and Bridge/emulator transport adapters plus executable,
   isolated fault injection for network, credentials, providers, updates and crashes.
5. Validate real credential backends, browser sign-in, shared entitlement adoption,
   scope, expiry, revocation, recovery and paid authority boundaries in installed apps.
6. Prove signed N→N+1 delivery through the intended staging feed on every supported
   platform, including interruption, wrong artifact/channel, tamper and migration
   failure. The current update-intent marker records outcome; it is not rollback.
7. Require exact-artifact evidence acknowledgment in Deployment Manager and
   chapter-to-video review. Finish typed matrix CLI bindings and durable pipeline
   wait; reuse existing obligations in PROGRESS.md.
8. Certify declared support on clean native hosts with performance budgets,
   accessibility/manual UX review, privacy canaries, failure repetition, soak,
   staged rollout and recovery rehearsal. Requalify changed release inputs.

Security review targets from source: the secret modal enables Node integration
and disables context isolation (`templates/vanilla/main.ts`); the main window's
new-window handler directly invokes `shell.openExternal`; journey redaction
checks four text patterns, while matrix recording evidence is assigned
`Redacted: true`. Audit sender identity, external URL policy, and pixel/log/HAR
privacy. No exploit or leaked credential was demonstrated. Shell recommendations
follow the [Electron security checklist](https://www.electronjs.org/docs/latest/tutorial/security).
Validate signing/update guarantees against the pinned dependency version and
[electron-builder's update documentation](https://www.electron.build/docs/features/auto-update/).

Finite testing cannot establish 100% certainty. Release acceptance should require
all declared cells on exact distributed bytes, native signing/update proof,
reviewed security findings, approved budgets, an acknowledged governance receipt
and rehearsed recovery. The visual contains the detailed journey matrix and
acceptance evidence for each work stage.

## Related Scenarios & Resources
- **deployment-manager** and **scenario-dependency-analyzer** will consume desktop telemetry and bundle manifests once offline builds mature.
- **app-monitor**/**Cloudflare tunnel** remain the primary exposure path for thin clients; **scenario-auditor** references deployment telemetry for health signals.
- Capabilities that matter for bundling: `postgres` (build metadata), `redis` (cache), `browser-automation-studio` (screenshot validation), and eventually local model runtimes for offline inference.

## Open Questions
- How will bundle manifests express dependency swaps (e.g., Ollama/Postgres replacements) so desktop runtime can honor them without shipping heavy services?
- Should telemetry storage move from JSONL to a lightweight embedded store when offline bundles land to enable richer diagnostics?
- What minimum `vrooli` shim is needed on desktop to keep CLI compatibility when no global install exists?

## External Inspiration
- Electron auto-update ecosystems (Squirrel/MSIX/PKG + differential updates) for release channels.
- Portable runtime supervisors and Deno deploy kits for ideas on embedding binaries without root privileges.
- Offline-first app packaging patterns from VS Code and Obsidian (data dir versioning, extension sandboxing) for future bundled mode. 

## Preserved visual sources

Historical HTML and editable canvas sources live beneath the protected
control-plane runtime home (`~/.vrooli` for the invoking user).
These are dated evidence or design supplements; this relocation does not
change plan status or establish release readiness.

- `plan-artifacts/docs-html-progress-20260908/scenarios/scenario-to-desktop/docs/internal/desktop-readiness-2026-09-07.html`

The originating installation must retain these artifacts with its durable backups.
Exact originals and SHA-256 hashes are in `backups/docs-html-progress-20260908/manifest.json`.
