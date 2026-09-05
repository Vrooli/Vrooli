# Portability truth chain III baseline — 2026-08-26

This is the immutable before-state for the portability truth-chain III work. The
repository already contained unrelated in-flight changes when the plan started;
the measurements below describe that state, not a clean historical commit.

## Twelve plan measures

| Measure | Before value | Command / evidence |
|---|---:|---|
| Capabilities | unavailable | `vrooli capability ledger --json`; the service was unreachable because `scenarios/web-console/.vrooli/service.json` declares unknown capability `agent-hooks`. The plan's authored expectation is 45. |
| Situation distribution | unavailable | Same ledger command; unavailable for the same service error. The plan records the audit baseline as 38 `no_work_required` and 1 `real_peer_nobody_wired`. |
| Fleet blocked rows | unavailable | `vrooli capability fleet --json`; same infrastructure-manager reachability failure. The plan records 19 rows across 9 scenarios. |
| Conformance targets/findings | unavailable | `vrooli capability conformance --json`; command was attempted while the service was unavailable. The plan records 598 targets and 0 findings. |
| Policy cells | 126 `no_work_required`, 8 `no_equivalent_ever`, 0 work | `jq '[.platform_policies|..|strings]|group_by(.)|map({v:.[0],n:length})' .vrooli/capability-vocabulary.json` and the plan's audit. |
| Circular safeguard cells | 23 files | `rg -l 'is not applicable on this host OS' internal/safeguards --glob 'safeguard.json'`; current source still contains the authored boilerplate. |
| Conformance test-compile failures | 4 modules | Plan audit's Linux-vs-darwin/windows `go vet` sweep: root, `agent-manager`, `scenario-to-desktop`, and `vrooli-emulator`. |
| Unguarded shell-out sites | 19 | Plan audit inventory; Phase 9 will regenerate the authoritative inventory under its evidence record. |
| Unguarded `/proc`/`/sys` files | 53 | Plan audit inventory; Phase 10 will regenerate the authoritative inventory under its evidence record. |
| Scenarios blocked/degraded by OS | 9 scenarios / 19 rows | Plan audit inventory; live readout unavailable until infrastructure-manager is healthy. |
| Resources using `compose-service` | 2 | `jq -r '.driver' resources/*/resource.json | grep -c compose-service` (kokoro and speaker-verification). |
| Resources projected by ledger | absent | Current `Grid` has an in-memory resource claim type, but the proto/readout projection does not expose it. |

## Baseline service observations

`vrooli scenario start infrastructure-manager` was attempted through the
control-plane lifecycle. It failed during the UI build because the existing
`@vrooli/react-component-library` projection does not export
`@vrooli/react-component-library/useLocale/1.0.1`. This is outside the plan's
acceptance boundary. The live ledger and fleet commands consequently returned a
degraded reachability error identifying the unrelated `web-console/agent-hooks`
manifest defect.

The Plan Manager/Git Control Tower collection was re-anchored under the same
collection name. Its terminal generation remained partial because Test Genie
reported admission saturation for queued members, no phase result for
`algorithm-library` and `api-library`, and an unknown `integration` phase in
`vrooli-emulator`'s testing configuration. Those findings are recorded in the
execution log; this document is the durable measurement baseline used while the
collection is repaired.

## Resource driver census

At baseline, the two plan-targeted container-dependent resources were:

```text
kokoro	compose-service
speaker-verification	compose-service
```

All other resource-driver changes are out of scope for the two migration phases.
