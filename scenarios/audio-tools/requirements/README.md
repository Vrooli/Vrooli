# Requirements

Requirement modules live here, one folder per group of operational
targets. Every requirement links back to a PRD operational target via
`prd_ref` and carries at least one validation entry pointing at its
proof.

- Statuses are earned, not asserted: auto-sync updates them from
  `[REQ:ID]`-tagged test results on comprehensive suite runs.
- Replace scaffolded manual validation stubs with test-typed entries
  (a `ref` to the test file plus the `[REQ:ID]` tag) as behavior lands.
- Validate with `business-health validate scenario <scenario>`; inspect
  traceability with `business-health matrix show <scenario>`.

## Expensive validation

Long-running audio qualification is intentionally marked `out_of_band` so the
ordinary scenario suite stays bounded and reproducible. The owning CLI is the
only supported runner:

```bash
audio-tools validation run-expensive
audio-tools validation check-freshness
```

The intended command duration is at least 60 minutes. Its historical artifact
and freshness check do not prove that each STT/TTS branch ran for that duration:
the September investigation found early-EOF and branch-duration gaps. Before
using it for qualification, repair and prove minimum input/sample counts,
per-branch elapsed duration, child failure propagation, and final-tail assertions
under [TESTING.md](../docs/internal/TESTING.md). Synthetic PCM can validate
transport behavior, not recognition quality or native microphones. A recent
timestamp alone cannot repair an invalid measurement method.

## Portable voice contract revision

The 2026-09-09 Business Health regeneration preserves the six original OT IDs
and all ten original ATD IDs and their test references. The PRD now has 15
targets; the registry has 24 requirements. New portable-routing, streaming,
quality, commercial, privacy, device and evidence obligations start `planned`.
Their planned manual entries describe validation work, not completed manual
attestations or a substitute for automated tests. Add real tagged tests at the
owning boundary; use manual logs only for actual manual qualification.

The four previous `complete` statuses (ATD-P0-004, ATD-P0-006, ATD-P1-001,
ATD-P1-002) were returned to `in_progress` after the owner validator reported
no earned sync snapshot. Historical test observations remain intact. No sync
snapshot was manufactured and no comprehensive certificate is claimed.

Target IDs and requirement IDs are independent: for example ATD-P0-004 still
means replay-safe protocol, while the newly added OT-P0-004 means explicit
portable routing. Follow `prd_ref`, not matching suffixes.

Product scope is now explicit; pending numeric, billing and native-cohort
decisions are centralized in the [decision sheet](../docs/internal/TESTING.md#pilot-decision-sheet-and-safe-first-slice).
The v2 setpoint program lists all 15 targets but has no owner-backed acceptance
joins. The all-target visibility regression prevents silent omission; listing
an obligation does not prove it.
