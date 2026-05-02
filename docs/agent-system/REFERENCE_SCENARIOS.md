# Reference Scenarios Registry

The `toolchain-validator` member of the `meta-optimization` team runs the development toolchain against a **gold-star reference scenario** each heartbeat. This file is the registry of which scenario currently holds that role.

Edits go through `meta-optimization` decisions; agents may propose nominations or demotions but do not edit the registry directly.

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

Nomination is an operator action — propose during a vision walk and file a `meta-self-improvement` decision.

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
