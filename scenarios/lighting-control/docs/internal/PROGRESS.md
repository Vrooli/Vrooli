# Progress — Lighting Control

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

### Addressable zone configuration — 2026-11-11

Scoped W3 gap: addressable controllers reported zero LEDs and existing SDK identity included mutable length, so detection could not imply useful static/Off output. Added explicit API/CLI/UI count configuration, separate durable desired counts, finite advertised range validation, stable resizable identities with unambiguous legacy ID preservation, and resize/readback before normal rendering. Early focused SDK/database/auth/UI tests and type-check pass. Final affected suites, independent adversarial review, and actual hardware qualification remain open. All changes remain in the isolated native-lighting delivery checkout; no live device or lifecycle effects were performed.

Independent maturation A identified and repaired legacy fixed-zone ID collision, incomplete whole-controller Off scope, and loss of partial resize facts. Independent B identified and repaired public supported-effects loss on SQLite roundtrip and count-free global Off scope. Global authority now binds the full fresh fingerprint→expected-count map on the mutation connection. Configuration also rejects impossible aggregate lengths before persistence. Partial resize completion is reported separately from unconfirmed rendering; persisted Off survives all tested failures.

Current retained fixture evidence: full API race suite passed after integrated repairs (`zone-api-final2.log`); final count-guard control/auth race and public ConfigureZone journey passed (`zone-final-focused.log`). All CLI packages and typed public binary-protobuf configuration request passed (`zone-cli-tests3.log`); generated endpoint/primitive owners regenerated outputs; governed proto verifier passed. UI full189 tests passed, API/localization/control focused18 passed, type-check and scoped ESLint passed. Test Genie run `20261111-134402-be015ba9` completed: unit and structure passed; dependencies failed on an SDA validation RPC deadline; docs failed on inherited mockup/link findings plus an introduced absolute checkout path now removed. The original waiter ended with an unexpected EOF; canonical terminal show and unfiltered artifacts preserve the verdict. Logs and exact changed-file/dependency provenance are retained under isolated `.vrooli/work/zone-configuration`.

Both initial reviews were finding-bearing and do not count as clean maturation passes. Fresh final-slice rereads requested from A/B; their final dispositions, dependency-provider recovery, remaining inherited documentation findings and actual hardware/headless proofs remain open. Actual root/device/permission/lifecycle operations were not performed by this outcome. SDA isolated-root install targeting was reported as `knw-1794404401031424288`; admission denied it and no install/manifest write occurred. UI modules were reused read-only from the exact unchanged main lock, with isolated proto override.


Final sequential source maturation is clean: independent fresh A and B found no material issues on the same scoped source/generated hashes. Reports and independent wire/storage/race/UI evidence remain with the isolated delivery review owner. Portable mockup provenance and explicit documentation registration now pass Test Genie docs run `20261111-145606-39dbd8b5`; the unchanged canonical cross-scenario backup runbook was added only as sparse validation support with a SHA receipt. The isolated dependency phase remains failed on the shared SDA deadline; read-only owner probes localize the known intermittent Code Facts import/admission route (existing report `knw-1794370895608027014`). Native MAIN SDA core success is separate evidence and does not replace an isolated dependency receipt. No service restart, index/cache mutation, dependency installation or physical qualification was performed.

### Offline saved inventory correction (E2)

Failed SDK discovery now returns the existing saved controller roster only for explicit `include_unavailable`; it preserves the actual backend error, last-seen age, enrollment and durable Off without synchronizing discovery or granting current capability authority. The UI retains the stricter unavailable observation, labels saved capabilities unverified, disables configure/apply/enroll while unavailable and keeps durable Off available. Actual authenticated API, typed CLI and UI fixtures cover offline inspection, failed storage, Off priority and reconnect.

Current isolated full API race (20 tested packages), all CLI (6 packages), full UI (192 tests/32 files), static checks and independent consumer aftercheck pass. Scoped Test Genie `20261111-161603-77afa5cc` passes unit18s/docs3s; decoded native provider payloads confirm all three isolated workspace roots and the isolated docs manifest. Unit reports23 warnings and docs104 advisory findings; PASS does not mean zero debt. Stored run evidence is degraded and does not assert `source_stable=true`; direct source-bound fixtures remain distinct. Historical pre-E2 clean A/B reviews are not current whole-program clean credit. See `.vrooli/work/zone-configuration/e2-offline/OUTCOME.json` for exact commands, hashes and applicability caveats. Actual native/hardware/headless proof and the routed dependency gate remain open.

### Scoped manual Off consumer correction (G2)

Fresh G found that a legitimate controller-scoped override disabled Dashboard and Controller global Off actions. The consumer now shares explicit all-enrolled/controller predicates for projection, actions and status. Dashboard Inspector, Controller status and Presets identify partial/unknown scope; widening Off never clears protection first. Existing global Presets wording and approved page composition remain intact. Normal UI/API/CLI proof, scoped public SQLite restart/roster-expansion/widening proof and independent aftercheck receipts are retained in `.vrooli/work/zone-configuration/g2-manual-off/`. G remains finding-bearing; historical pre-G2 clean credit is not promoted to final source. Actual SDK/OS/hardware/headless proof remains separate and parent-owned.
