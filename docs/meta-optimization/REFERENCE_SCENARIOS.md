# Reference Scenarios Registry

The `toolchain-validator` member runs the development toolchain against a **gold-star reference scenario** each heartbeat. This doc is the registry of which scenario currently holds that role.

**Posture:** State. This is not doctrine or memory — it's a pointer that gets updated when the operator changes which scenario is the reference.

**Revisit markers:** Review when the current reference scenario is itself restructured significantly, or when a new scenario overtakes it in quality.

---

## Current reference

| Role | Scenario | Notes |
|------|----------|-------|
| **Gold-star (primary)** | `reference-react-vite` | Operator-nominated 2026-04-24 (vision walk; `dec-1776981723540926630` accepted). Purpose-built scenario covering API+CLI+UI+tests+CI. First scan on 2026-04-24 produced 72 standards violations (41 High) — see `shared/TOOLCHAIN_SCAN.md`. Reference is currently dirty; cleanup is the implicit toolchain-validator agenda until it scores clean. |
| **Secondary references** | *(none yet)* | Scenarios used for specific tool categories (e.g., a scenario that best exercises `test-genie`, one that best exercises `scenario-auditor`). Populated as tooling matures. |

---

## Nomination rules

A scenario can be proposed as the gold-star reference if:

1. It's in `state: active` and being deployed (not a prototype).
2. It scores clean on every current toolchain tool (`scenario-auditor`, `test-genie`, `tidiness-manager`, and eventually `development-toolchain-validator`).
3. It exercises a reasonably broad cross-section of scenario patterns (API + CLI + UI, tests, CI wiring, resource integration).
4. Its structure is considered stable for at least 60 days.

Nomination is an operator action — file a note here, not a decision, when proposing one.

---

## Demotion rules

A scenario should be demoted from reference status if:

1. It accumulates persistent violations it can't fix because the violations are actually about tooling rules, not the scenario.
2. It's scheduled for deprecation or significant restructuring.
3. A better candidate has emerged and the operator chooses to promote it.

Demotion is also an operator action.

---

## Reference-scenario rot

When the gold-star reference starts producing violations that weren't there before, one of three things is true:

- The tools regressed (scored clean yesterday, dirty today) → file `toolchain-violation` against the tool
- The reference rotted (drifted from what the tools now expect) → file `toolchain-violation` against the reference
- The tools gained new rules and the reference hasn't caught up → file `toolchain-violation` proposing a reference update

`toolchain-validator`'s job is to distinguish these three cases. The operator resolves.

---

## History

A log of reference-scenario changes. Format: date, previous reference, new reference, reason.

- 2026-04-24 — *(unset)* → `reference-react-vite` — Operator nomination at vision walk. First scan produced 72 standards violations (41 High, 10 Medium, 20 Low, 1 Info), 0 security vulns, opaque `test-genie` 500. Reference is dirty against the tools that gate every other scenario; cleanup is the operator's call.

---

## Open questions

- Should there be more than one gold-star, rotating? (Keeps any single scenario from becoming test-overfit.)
- What's the threshold for "exercising a broad cross-section"? (Probably a checklist; not yet written.)
- Who notices when the current reference starts drifting — is it toolchain-validator's job, or the owning scenario's team?
