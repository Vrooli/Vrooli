# Go-To-Market — Infrastructure Manager

**Status: not-applicable, deliberately.** There is no external market for
this scenario. It is the `infra-health` team's instrument, and its
"launch" is internal adoption by that team. This document records the
internal adoption motion instead, because that is the real analogue and it
does have failure modes worth naming.

## Purpose Of This Document

Use this document to answer who this is for, how they come to rely on it, and
what would make adoption fail. External positioning is not applicable; see
[`MONETIZATION.md`](MONETIZATION.md) for why.

## Audience And Positioning

| Audience | What they need from it |
|---|---|
| `runtime-health-scanner` | The daily reader. Needs one ranked answer to "which signal is worth investigating this heartbeat?", replacing a triage ladder walked by hand. |
| `infra-contrarian` | Needs the setpoint's integrity and the trust distribution, to challenge findings whose evidence is instrument fault or whose deadband states the current reading. |
| `platform-code-auditor` | Peripheral. Its lane is judgment-shaped and stays outside the board (see `DOMAINS.md` § Deferred Domains). |
| The operator | Reads the board at the morning vision walk, where infra-health findings drain at Phase 5.7. |
| Peer instruments | `meta-optimization-manager` supervises capability-owner reachability and is the natural watcher of this instrument's own condition. |

**Positioning, internally:** this is *the one address*, not another tool.
Invariant 1 of the instrument contract is that a member starts at the board,
not at the tools. If a member still has to know which of autoheal, capacity,
storage-manager or test-genie holds the answer, adoption has not happened
regardless of whether the scenario is running.

## Channels

Internal only.

| Channel | Role |
|---|---|
| `infra-health` member documents | The primary channel. Members' `RESPONSIBILITIES.md` and `HEARTBEAT.md` cite the board instead of naming sensors and deadbands. |
| `team.json::instrument` | The declaration that makes adoption machine-readable — `status: live`, naming this scenario and the scenarios it covers. Read by `prompt-manager graph instruments`. |
| `cli-health` command federation | How agents outside the team discover the surface. |
| Morning vision walk | Where findings reach the operator. |

## Launch Motion

Adoption is not a launch event; it is a substitution. The sequence matters:

1. **Ship the setpoint read and the live join** (`targets` + `readings`), so the board can answer something real.
2. **Point one member at it** — `runtime-health-scanner` first, since its heartbeat is the highest-frequency reader.
3. **Delete the prose it replaced.** Roughly 25 of the 67 lines in `runtime-health-scanner/RESPONSIBILITIES.md` — the Sensor-First, Sensor-Integrity, Capacity Supervision, Validation-Cost Supervision and Capability Supervision sections — are sensor routing and duplicated deadbands that collapse to a board citation. The incident-surface and remediation-workflow sections stay; they are judgment.
4. **Flip `team.json::instrument` to `live`** and name `coversScenarios`.
5. **Confirm the ratchet turned** — the team's orientation cost should fall in the same cycle its scenario coverage rose.

Step 3 is the one that is usually skipped and the only one that produces the
benefit. Canon is blunt about it: adding an instrument on top of existing
wiring gives you both, and both is worse than either.

## Messaging

Plain and unhedged, matching the scenario's own voice rules:

- **"One address, not another tool."** The value is subtraction, not addition.
- **"No number without its qualifier."** Every ratio carries denominator-confidence; every reading carries a trust verdict.
- **"It ranks; you decide."** The board holds no authority, and saying so plainly is what keeps members willing to trust it.

What *not* to claim: that the board makes the platform more reliable. It makes
the platform's reliability **observable**. Reliability improves when findings
get acted on, which is the team's work and the operator's decisions.

## Validation Experiments

Adoption is measured, not asserted:

| Question | Measurement | Source |
|---|---|---|
| Did the team actually consolidate onto one address? | `domainAddresses` for `infra-health` trends toward 1 plus declared fallbacks | `prompt-manager graph instruments` |
| Did the team get simpler? | Orientation cost falls in a cycle where scenario coverage rose | `prompt-manager graph orientation-cost` |
| Is the board being read? | Read counts against the board's surface | `cli-health` / receipt stream |
| Do findings get acted on, and do the fixes work? | Actuation efficacy — findings whose sensor returned in band | this scenario, `OT-P1-003` |

The last one is the honest test. A board that produces findings nobody closes
is a board that has been adopted in form and not in substance.

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — why there is no external market
- [`../../PRD.md`](../../PRD.md) — the capability and its users
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — what the board actually serves
- `docs/agent-system/TARGET_MODEL.md` § "Why invariant 6 is the one that compounds" — why deleting the replaced prose is the whole payoff
