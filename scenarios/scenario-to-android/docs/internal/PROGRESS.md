# Progress — Scenario to Android

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/scenario-to-android/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
Copied July 7 template history was removed from this scenario’s log.
The table retains selected dated milestones. Other milestones and their limitations
remain in the archive; consult it before resuming historical work.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative.
The scenario history below begins with its 2026-08-10 regeneration.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-17 | codex | partial | Closed current Android dependency and test-harness drift through Scenario Dependency Analyzer: pinned React Query and Testing Library to the api-base contract, aligned `eslint-plugin-react-hooks` with the approved range, and made delivery-surface fixture swaps unmount the exact provider-owned render. Current API tests, UI tests 150/150, production build, requirements, and dependency governance pass; fresh Phase 9 admission remains externally saturated. |
| 2026-08-17 | codex | partial | Fresh server-owned run `20260817-035220-b6bec178` completed 20/21 phases: every Android implementation phase, including experience, passed; only the shared workflow provider failed because the durable provider returned no shared validation response. The physical Galaxy A03s matrix evidence and targeted API/CLI/UI/build checks remain green; this is not claimed as an integrated PASS until workflow-health recovers. |
| 2026-08-17 | codex | partial | Hardened the Android/iOS delivery experience surfaces against real AX reconciliation: provider-wrapped component tests, semantic table/list/status/form bindings, distinct Android unverified versus Play-ready channel fixtures, explicit lease-loss promotability, 44px mobile actions, and safe-area-aware bottom navigation. Android server-owned run `20260817-011717-14d1aeaf` reached 19/21; remaining dependency runtime state was external and the final post-fix run is being validated under concurrent Test Genie maintenance. |
| 2026-08-17 | codex | partial | Android experience fixtures now distinguish missing SDK toolchain, missing `/dev/kvm`, unpaired USB debugging, and expired wireless pairing with named remedies. |
| 2026-08-17 | codex | partial | Revalidated Android API/UI/build/requirements and the shared delivery spine after the mobile provider repairs. The physical Galaxy A03s evidence remains retained and honest; a fresh Phase 9 Android suite was attempted but Test Genie rejected admission with caller-queued capacity saturated, so no new integrated Android verdict is claimed. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

## Governed native-project builds — 2026-10-22

Added `project_kind=native` to the existing build/sign/artifact pipeline. Native source is copied into a unique private build directory, bounded and fingerprinted, with required files and manifest metadata checked; symlinks and conflicting package metadata are refused. Portal's ordinary TV preview uses this path, without Capacitor generation or in-place source mutation. Debug-only cleartext qualification accepts one private IP; release builds cannot request it. The synchronous build handler now extends its response deadline to match the existing bounded builder rather than dropping the HTTP response at 30 seconds.

Focused native preparation/isolation/policy tests passed. Test Genie `20261022-161830-e6d61d24` unit and dependencies passed before the later response-deadline/source-consistency refinements. Real APK/AAB builds succeeded through the owner; v0.1.2 APK was installed through Device Control. The first v0.1.1 build produced artifacts but lost its HTTP response; its retained failure and source/artifact provenance were later corroborated by an owner build returning identical artifact bytes. Native UI qualification belongs to Portal and remains incomplete.

### Qualification checkpoint — 2026-10-22

The v0.1.3 ordinary APK displays native pages. An asserted connection-screen trial proves host UFW drops the TV's connection to the LAN Portal gateway; both native HTTP and WebView timeout. The existing HTTPS route is reachable by the TV and responds with HTTP 302, while a host-side probe received 403; these are different observations and neither is a catalog success. Inputs enumeration shows HDMI 1/2/3 and AV; switching was not attempted. Explicit sleep/wake verbs were accepted, but state remained on, so physical sleep is unverified. The preview was returned to stock Home and held leases released.

Focused native preparation and build handler packages passed again after source-consistency/response-deadline refinements. v0.1.3 was built and installed through owners; its application manifest has ordinary launcher categories and no HOME category.
