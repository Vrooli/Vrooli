# Generated Golden Registry

The `toolchain-validator` member of the `meta-optimization` team runs the development toolchain against a **generated golden** each heartbeat. This file is the registry of which durable golden identity currently holds that role.

Edits go through `meta-optimization` decisions; agents may propose nominations or demotions but do not edit the registry directly.

---

## Generated-Golden Contract

- `golden_slug` is the durable identity used by manifests, validation records, reports, and history.
- The template (`template_id` + pinned version) is the source of truth for the golden substrate.
- Validation runs materialize a fresh physical path outside `scenarios/` by default, pass that path to agent-manager and tool runners, then clean it up unless explicitly retained for debugging.
- Diff and manifest paths normalize back to the golden's logical root. Operators should not pin manifests to temporary materialization paths.
- Historical committed references may appear in older records, but no current operational workflow should require maintaining `scenarios/reference-react-vite`.

Deletion criterion for committed references: remove a committed `scenarios/reference-*` tree only after code, docs, tests, manifests, and tool runners no longer consume that path as an active source tree.

---

## Current Golden

Each row pairs a generated golden identity with the template it is materialized from. The generated golden's quality is bounded above by the template's; see [Template-Golden Coupling](#template-golden-coupling) below.

| Role | Golden slug | Template | First registered | Last audit | Notes |
|------|----------|----------|-----------|------------|-------|
| **Gold-star (primary)** | `reference-react-vite` | `path:templates/scenarios/react-vite/` | 2026-04-24 | 2026-05-04 — **stale** (template fitness audit `template-fitness-audit/react-vite-template/2026-05-04`; not re-run since) | Operator-nominated 2026-04-24 (vision walk; `dec-1776981723540926630` accepted). The slug remains durable for manifests and records; current validation materializes from the template instead of using a committed scenario tree. Historical scans from 2026-04-24 produced 72 standards violations (41 High) against the then-committed reference — see `shared/TOOLCHAIN_SCAN.md`. |
| **Secondary references** | *(none yet)* | — | — | — | Scenarios used for specific tool categories (e.g., a scenario that best exercises `test-genie`, one that best exercises `scenario-auditor`). Populated as tooling matures. |

---

## Nomination rules

A scenario (paired with the template it is generated from) can be nominated as the gold-star generated golden if:

1. It's in `state: active` and being deployed (not a prototype).
2. It scores clean on every current toolchain tool (`scenario-auditor`, `test-genie`, `tidiness-manager`, and eventually `development-toolchain-validator`).
3. It exercises a reasonably broad cross-section of scenario patterns (API + CLI + UI, tests, CI wiring, resource integration).
4. Its structure is considered stable for at least 60 days.

Nomination is an operator action — propose during a vision walk and file a `meta-self-improvement` decision.

---

## Demotion rules

A scenario should be demoted from gold-star golden status if:

1. It accumulates persistent violations it can't fix because the violations are actually about tooling rules, not the scenario.
2. It's scheduled for deprecation or significant restructuring.
3. A better candidate has emerged and the operator chooses to promote it.

Demotion is also an operator action.

---

## Generated-Golden Rot

When the gold-star generated golden starts producing violations that weren't there before, one of three things is true:

- The tools regressed (scored clean yesterday, dirty today) → file `toolchain-violation` against the tool
- The template rotted or its generated output drifted from what the tools now expect → file `toolchain-violation` against the template/golden pair
- The tools gained new rules and the generated golden hasn't caught up → file `toolchain-violation` proposing a template update

A fourth case worth distinguishing: the **template** rotted (drifted away from what's fit to be copied), and the reference inherits the rot at every regeneration → file `toolchain-violation` of subtype `reference-stale-from-template`. See the next section.

`toolchain-validator`'s job is to distinguish these four cases. The operator resolves.

---

## Template-Golden Coupling

Each gold-star generated golden is materialized from a template. The golden's quality is bounded above by the template's. When the registered template version is older than the template's last meaningful contract version, manifests for that golden may need refreshing; `toolchain-validator` flags this as a `toolchain-violation` of subtype `reference-stale-from-template`.

The longer-cadence template audit uses the [`reference-pattern-fitness`](REFERENCE_PATTERN_FITNESS.md) lens, owned by `toolchain-validator`. It is **not** in `path:docs/scenario-qa/methods/audit/` — that registry is scoped to the `quality-auditor` member of `scenario-qa`, who audits regular scenarios where multiplier framing would mislead. Findings from the template audit feed back into the template; subsequent generated-golden materializations pick up the improvements automatically.

The `Last audit` column links to the most recent typed audit record under `template-fitness-audit/<slug>/<YYYY-MM-DD>` — typically a fitness audit of the template, not of the reference itself. When the reference itself is the audit target (rare; usually only when probing reference-specific drift the template doesn't carry), the column note clarifies.

---

## History

A log of generated-golden registry changes. Format: date, previous golden, new golden, reason.

- 2026-04-24 — *(unset)* → `reference-react-vite` — Operator nomination at vision walk. First scan produced 72 standards violations (41 High, 10 Medium, 20 Low, 1 Info), 0 security vulns, opaque `test-genie` 500. Reference is dirty against the tools that gate every other scenario; cleanup is the operator's call.
- 2026-05-04 — registry table extended from 3 columns (Role / Scenario / Notes) to 6 (Role / Scenario / Template / Generated / Last audit / Notes); added Template-Reference Coupling section. Establishes the template→reference pair as one entity and links each row to its `reference-pattern-fitness` audit. Filed under `meta-self-improvement` decision (see [`REFERENCE_PATTERN_FITNESS.md`](REFERENCE_PATTERN_FITNESS.md)).
- 2026-07-02 — active model changed from committed reference scenario trees to generated golden substrates. `reference-react-vite` remains the durable golden slug for historical records and manifests.
