# Progress — Device Control

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/device-control/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
The table retains selected dated milestones; consult the archive for the full sequence.

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/device-control/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append concise milestones when work lands. Keep detailed execution receipts
with the owning plan.

## Progress Log
| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-18 | codex | Closed the locally actionable agent and documentation gaps: bounded multi-iteration planning now records typed world models and per-step evidence, supports planner goal completion and dry-run, and promotion creates a deterministic flow consumed by the existing export path. | Archived |
| 2026-08-18 | codex | Closed additional plan-level gaps: `packages/mdns-go` now exposes the typed `Browse(ctx, []string, Options)` contract, the required README, IPv4/IPv6 multicast joins, address-family queries, and cancellation-preserved observations. | Archived |
| 2026-08-18 | codex | Repaired the generic mDNS browser's Linux shared-port fallback, canonical DNS-SD queries, section-aware unknown-record skipping, and requested-service PTR filtering. | Archived |
| 2026-08-18 | codex | Completed targeted implementation validation for the reachable non-mobile device plan: API/CLI/mDNS suites, focused race checks, gofumpt, golangci-lint, UI 183 tests/type-check/lint/coverage, requirements/business-health validation, and experience-contract validation pass. | Archived |
| 2026-08-17 | codex | Repaired the UI test provider harness and route providers, expanded focused coverage, and declared the `agents_list` REST projection explicitly as an external JSON shape. | Archived |

| 2026-10-22 | codex | done | Read-only semantic assertions now use the existing bounded retry budget; input actions remain one-shot. Failed target assertions capture a redacted frame before restoration when the live screen is on and unlocked. Four focused regression checks pass, including locked-screen refusal. Live TV runs d89b55ef-4a94-4910-beb7-3b18a3ea1b0a and dab43faa-17e8-4918-b88f-722499bc7d50 retain failure-time frames and four attempts. Unit phase 20261022-201739-3c9ea4c2 remains failed on desktop socket/AT-SPI/X11 fixtures; reuse issue knw-1792684676395178317. WebView forward teardown remains unqualified. |
## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map


## Agent usage and improvement implementation — 2026-09-04

### Summary

Implemented scenario-owned usage and improvement skills with bounded programs,
explicit workflow selection, verified promotion, preserved repair contracts,
and outcome-linked learning. Speed targets remain pending a comparable operator
baseline. Test evidence establishes implementation behavior, not household-device
coverage or a measured agent speedup.

Device programs are `prepare-task`, `replay-flow`, `author-flow`, and
`setpoint-read`. The FlowService owns a durable SQLite revision library, exact
device/context selection, atomic expected-version saves, idempotent retry, and
retained validation receipts. Candidate promotion requires every step to pass,
an explicit terminal assertion, and deterministic targeting. Dry-runs and vision
resolution cannot qualify. `property-assert` validates an adapter-observed value
without requiring a screen. Repair preflight uses the exact device adapter.
The old prompt-manager local-pack skill was retired in favor of the scenario source.

Shared changes: Memory accepts optional first-action, tool-round-trip,
visual-reasoning and workflow-reuse observations. Missing values remain absent;
ambiguous or retry-only histories cannot earn first-action timing. Test attempts
are excluded from operator metrics. Program Runtime captures nested contract
envelopes without leaking child stdout and refreshes declared source for each
new kernel; refresh scans program directories rather than the entire source tree.

### Findings

| Capability | Primary path | Verification | Failure path |
|---|---|---|---|
| Orientation and selection | Usage skill and discovery program | Exact identity and context | Refuse ambiguity; inspect owner inventory |
| Workflow reuse | Exact saved revision | Owner terminal result and evidence ID | Preserve failed result; no speculative alternate |
| Candidate promotion | author-flow | Assertions, complete execution, durable identity | Failed/pending candidate remains unsaved |
| Repair | author-flow plus expected revision | Existing checks preserved | Refuse stale or weakened baseline |
| Improvement | Improve skill and learning sensor | Comparable windows and sample counts | Missing/unreliable evidence cannot meet a target |

### Divergence probe

| Instruction | Possible divergent reading | Resolution and evidence |
|---|---|---|
| Select a matching workflow | Execute the first fuzzy result | Explicit identity/version required; search-only probe performs no execution |
| Save a successful candidate | Treat pending, dry-run or AI completion as verified | Owner gate and negative promotion probes refuse these paths |
| Repair while preserving checks | Drop an assertion to make the repair pass | Owner regression and live BAS refusal preserve the acceptance contract |
| Reduce time to first action | Count retries or missing timestamps as faster samples | Memory tests exclude retry-only/ambiguous histories and preserve missing values |
| Run an improvement goal | Repeatedly file work without implementing | Improve skill explicitly routes implementation under existing task authority |

No divergence remains on these probed instructions. This is a bounded probe,
not a claim that every possible user task has been tested.

### Contract integrity and prose retirement map

Routine leaves use owning CLIs; programs use registered typed bindings. Structured
Attempt JSON and typed workflow JSON are necessary inputs, not output-parsing
workarounds. No raw ADB, browser-driver endpoint, database access, or shell-based
device control is part of the usage path. Ordinary usage invokes registered
programs; temporary sessions are used only by program validation.

| Repeated agent work retired | Durable replacement | Verification |
|---|---|---|
| Reconstructing orientation joins | Bounded discovery/prepare program | Ambiguous-target probes and live read |
| Hand-copying a successful draft into storage | Owner promotion plus author-flow | Persistence/replay regression and live browser test |
| Overwriting a broken saved flow | Revision-preserving owner update | Stale/weakened repair refusal |
| Informal claims that reuse is faster | Outcome-linked Memory cohorts | Optional-field and test-exclusion checks |
| Parsing nested program prints | Runtime returns a bounded envelope Handle | Kernel regression tests |

### Validation

- All 24 declared fixtures across both scenarios passed in fresh test-provenance
  sessions, with a preflight explain call for each fixture.
- All 11 program behavior probes passed. The owning Go tests invoke these probes.
- Device API `go test ./...`, device CLI control tests, BAS workflow handler and
  service tests, Memory learning tests, Runtime contract-index tests, and all 30
  kernel Handle tests passed.
- Live BAS: create/execute/save, exact replay, revision-2 repair, stale-revision
  refusal, weakened-assertion refusal, and original-revision replay all passed.
- Live device orientation read succeeded. A deterministic property adapter proved
  execution, assertion failure, durable restart/read, and saved revision replay.
- Memory accepted the new effort fields, returned the same entry on exact retry,
  and excluded the test attempt from operator measurements.
- Both scenario Test Genie program and skill-set phases passed. Broader unit
  runs have provider/UI failures; device business checks also report older
  unearned completion claims. These runs are not represented as green.

Machine-readable program IDs and live evidence are retained in
`.vrooli/program-runtime/tests/validation-evidence.json`. Behavioral probes are
in `.vrooli/program-runtime/tests/test_workflows.py`.

### Recommendations

A (recommended): use the registered usage skill for authorized real tasks and
capture comparable operator attempts; then use the improve skill to move the
first measured bottleneck through its owning implementation workflow.
B: continue fixture validation for an unavailable device or environment, retaining
that evidence class separately. This can improve correctness but cannot earn a
physical-device claim or operator speed baseline.

### Notes

No physical device was actuated for this implementation validation. Exact
screen/transport/app coverage and earned latency targets require authorized
representative device tasks. The self-improvement skills can guide sustained
implementation; they do not turn missing evidence into a completion claim.

## Historical handoff leads retained during condensation

These clauses are preserved from dated entries; later work may have superseded
them. They do not assert that an old failure or gap still exists.

| 2026-09-18 | Codex | Documentation-first remote desktop baseline | Documented Device Control desktop ownership, companion boundary, WebRTC/session data model, security invariants, and validation matrix. |

- **2026-08-18 — partial**: Full API/race/lint, mDNS live, Cast live, UI, requirements, and independent Avahi checks pass; owner-present PIN/event/UI walkthroughs and the provider-owned Test Genie/baseline gate remain deferred.

- **2026-08-17 — partial**: The bounded API fallback remains healthy and device-control API/UI/build checks pass; the remaining failures are the shared `@scenario/self` workflow resolver and repeated UI-health chrome captures.

- **2026-08-17 — partial**: The only remaining integrated failure from that run is the shared BAS `@scenario/self` workflow resolver.

- **2026-08-17 — partial**: Cleared the remaining local Go quality floor: four pre-existing gofumpt findings were normalized, and the full device-control API suite, `go vet ./...`, and `golangci-lint run --timeout=5m` now pass.

- **2026-08-17 — partial**: Retained run `20260817-044920-e9b29908` reflects pre-run status-bar/docs state and the shared workflow-provider failure, so fresh integrated evidence remains pending.

- **2026-08-16 — partial**: Final managed physical screenshot flow `98fdca8f-13ce-4117-94f5-b95928e4edc2` passed after deployment and retained screenshot evidence `4f3ff631-e5a3-40bf-86bc-84daa1aa1caf` at 720x1600 with only the declared `status_bar_identifiers` rule; the remaining narrow black band is the intentional status-bar privacy mask, not a geometry bar.

- **2026-08-15 — partial**: The remaining architecture warnings are pre-existing manifest maturity debt, not binding errors.

- **2026-08-13 — partial**: Follow-up live validation found the Galaxy A03s `screenrecord` path can produce a valid H.264 container whose display body is uniformly black even while ADB screenshot/state probes report an awake launcher.

## Physical TV observation and control repairs — 2026-10-22

Observe-only flows no longer register rotation restoration; keyguard and rotation restoration preserve only the state actually changed, including absent raw Android settings. Android launch resolves a TV launch activity and reports accepted versus observed foreground explicitly. Reconnect refreshes capabilities; exact known-device reads preserve discovery timestamps. Added fixed read-only hardware/WebView/storage/Home inspection and CLI frame capture through the existing flow owner. The device detail page now exposes actual captured frames beside directional controls, holds its own lease, serializes input/capture, reports failures, and refreshes connection state.

Physical evidence: two explicit-Home/SmartTube/frame/Back/Home cycles showed SmartTube on screen and matching foreground state; Home focus movement was visible. An ambient-screen launch remained accepted without visible foreground, retained as a failure. Observe-only capture receipts show no restoration. The Portal ordinary preview was installed and displayed; stock Home remained selected and was recovered. A service restart overlapped one later preview capture, invalidated that observation, and was followed by explicit Home recovery and lease release; lifecycle changes and device trials must be serialized.

Focused adapter/control/handler/CLI regressions, UI tests, type-check and lint passed. Test Genie `20261022-155620-5a312a15` programs passed; unit failed on shared desktop Unix-socket path length, global UI coverage, and a subsequently refreshed primitive artifact. Shared socket defect filed as `knw-1792684676395178317`. No comprehensive readiness claim.

### Qualification checkpoint — 2026-10-22

The v0.1.3 ordinary APK displays native pages. An asserted connection-screen trial proves host UFW drops the TV's connection to the LAN Portal gateway; both native HTTP and WebView timeout. The existing HTTPS route is reachable by the TV and responds with HTTP 302, while a host-side probe received 403; these are different observations and neither is a catalog success. Inputs enumeration shows HDMI 1/2/3 and AV; switching was not attempted. Explicit sleep/wake verbs were accepted, but state remained on, so physical sleep is unverified. The preview was returned to stock Home and held leases released.

Fixed governed flow contracts: scalar aggregate primary fields now preserve validation runnable and run identity instead of ambiguously selecting repeated lists; author/replay v2 emit valid JSON. Eighteen program probes pass. Saved flow `e60f5da8-3ad6-4486-83e0-e23630f7876e` revision 1, context `portal-tv:ordinary-preview:v0.1.3`, passed authoring (`1272f570-8710-48cf-8f12-34bd3c095805`) and exact replay (`b7308fc6-baf4-477b-89dd-21cda0e9eb7e`). Device-page route changes now isolate leases; 12 focused tests, type-check and lint pass. Test Genie `20261022-165528-e75f9a05` dependencies and programs pass after SDA module reconciliation. The prior unit run remains failed on shared desktop socket fixtures and global UI coverage; 125 UI cases passed within that run.

### Final evidence and handoff — 2026-10-22

A native Home shortcut opened SmartTube, with matching captured frame and foreground state. Two Back presses remained within SmartTube, so return-to-preview behavior is unverified; Home recovered stock Google TV Home and the lease was released. The later HTTPS browser-opening attempt ended on native Home, not a verified sign-in page. No authenticated Vrooli render is claimed. Device Control's built UI now places frame capture and directional controls above detailed inventory; the deployed BAS capture confirms both are visible together.

The existing Drive refined-plan document's Experiments tab now contains the implementation findings and evidence link; its original tab was preserved. Curated ZIP includes protocols, compact receipts, native source, APK/AAB and checksums, excluding credentials, room photo and personalized recommendation images. ZIP SHA-256: `b67039df24256bd667f1262b398cab8b16fd5295ef2b37e0dba0ebaf1f23b620`. Full private evidence remains in the local experiment directory. A Swarm follow-up capture was refused by the exact `swarm.bookkeeping` / `swarm-manager:capture/intake` authorization gate; no grant was widened. The remaining findings are retained here and in Drive.

Work-record append was attempted through `vrooli-memory journal note --kind work-record` with all required fields, but the owner returned `unavailable: unexpected EOF` without an entry ID. Read-side reconciliation did not establish whether the write persisted; no blind retry was made. Repository and Drive records are the confirmed handoff.

## Reverse-network qualification — 2026-10-22

The TV loaded Portal's actual catalogue and rendered Nooch's Your day screen using explicit loopback ADB reverse routes over its authorized wireless transport. Live run `b7d2101e-d15a-40e9-a118-987f341a4162` remains failed because the final exact Nooch accessibility assertion did not match; captured page rendering is confirmed separately. Earlier LAN UFW rejection remains valid for that path and does not imply general TV-to-host inability.

Device Control now owns bounded `reverse-tcp` setup/cleanup and exact-destination `reverse-tcp-remove` recovery. A typed local ADB server request is necessary because the CLI rejects explicit localhost addresses. Android 11 normalizes listed endpoint names; an early exact-name cleanup receipt was insufficient, corrected and reconciled by recovery run `9cc0de21-153e-409e-98ed-815e67457dd5` before final use. Corrected final cleanup verifies both routes removed. Stock Home/default component and empty closing sessions are confirmed; the preview's previous HTTPS origin was restored. No Assistant, firewall, VPN or microphone change.

Debug preview v0.1.4 supports only scoped debug localhost HTTP; native build artifact SHA-256 `a2262a69ab1771a94a71f9debf438d176cb93e2c50214f0f6fce3fc141cd0b7f`. Focused protocol/adapter/control/recovery/declaration and native build preparation tests pass. Test Genie `20261022-175436-c0f61748` programs pass; unit still fails on shared desktop fixtures and UI coverage, with its reverse declaration fixture subsequently repaired. No full-suite readiness claim.

Next work: reuse existing WebView/CDP observation to qualify semantics, choose a durable authenticated route, then test an explicit TV microphone path into existing Portal/AudioTools owners. Current APK has no recording permission or Assistant service. Hona's mesh repository does not establish the TV voice/role mechanism. Full private protocol, failures, primary sources and frames are retained under `vrooli-experiments/tv-capabilities-20261022T144942Z/reverse-network-20261022`. Work-record write again returned EOF without identity; repository/Drive records are the confirmed handoff.

Drive experiment addendum and 267,049-byte evidence ZIP verified in the existing folder. ZIP SHA-256 `6afd4dcad688f2be56332b2affd3764964d39b342415904c0ed76df4e91ac49e`; bundle link is retained in the private `drive-handoff.json`. Scenario to Android artifact checksum is for APK+NUL+AAB, independently verified; the APK alone is 87,292 bytes with SHA-256 `42947b4fc599d432fd0b8b10b8db9efbb08c5002d6fc23004815c7796308d12d`. The ZIP SHA256SUMS uses the correct individual APK digest.

## Home-first and remaining TV research — 2026-10-22

The operator requires direct home access with a saved secure-domain fallback for travel. Portal's existing TV README and decisions now specify separate routes, paired host verification, lightweight bounded home probing, visible remote sign-in and no automatic write replay. Reuse `api-base`'s existing `createScenarioProxyHost` in Portal's UI gateway; App Monitor is a source-backed precedent, not a running/qualified TV dependency. Gateway-aware application URL resolution and local HTTPS remain implementation work. No automatic fallback, DNS, firewall or VPN change was installed.

Remote observation `4d740007-2c70-4bbd-abde-12abdd00b610` passed its operations and captured the real Cloudflare Access email-code sign-in page. No email/code was submitted; authenticated Portal access remains unqualified. Fresh local navigation trials failed at selection and then catalogue readiness; their exact causes are not fully established and earlier Nooch rendering is not a fresh replay PASS. Both owned routes were restored, the previous HTTPS setting restored, preview closed and stock Home/empty sessions verified.

Device Control now exposes fixed read-only voice facts through InspectDevice, with nullable/unknown fields and verified-system-user scope. Live facts: microphone feature false; selected Katniss voice service; OEM policy input names without an audio-delivery claim; Assistant role query unknown. No audio was recorded. AudioTools reports healthy general API, available Whisper batch serviceability and unavailable pinned Kyutai streaming. WebView forward teardown remains unqualified; source-audit bug `knw-1792696363648007859` filed without physical CDP attachment.

Focused voice/inspection/reverse-flow checks and governed proto generation/verification pass. Test Genie `20261022-192128-ae97fc18` unit fails on desktop socket/AT-SPI/X11 fixtures and existing coverage floors (all 125 UI cases pass). The research, failures, sources and 151,781-byte evidence bundle are verified in the existing Drive report/voice documents. ZIP SHA-256 `cdf0a9685920e256fbad99d457bbcbb3e653d2b5219d3f14ebc0b0a141775b89`. Private protocol/receipts: `vrooli-experiments/tv-capabilities-20261022T144942Z/local-first-20261022`. Work-record `0f7955e9-9c2a-4804-8771-e425d21502e5` persisted; attribution verification is invalid, so it is not a trusted caller-proof claim. No comparable learning baseline is claimed.
