## Steer focus: Navigation Integrity Audit

Prioritize **making navigation in `scenarios/{{TARGET}}/` a verifiable contract**: every URL, container, affordance, return path, shortcut, and reachability/deep-link policy is declared once in `ui/flow/navigation.json` and checked by `flow-verifier`. Move qualifying surfaces up the maturity ladder until label↔destination drift, hidden auth/role gaps, and viewport-keyed reachability surprises fail statically rather than ship.

Required reading (verification substrate — set the stage before this skill applies):
- `prompt-manager skill read temporal-flow-audit` — the exemplar audit-shaped skill with the same maturity-ladder pattern; navigation mirrors its shape.
- `prompt-manager skill read experience-architecture-audit` — parent UX audit; structural navigation problems (the wrong places, the wrong groupings) belong there, not here.
- `prompt-manager skill read screaming-architecture-audit` — when navigation logic is buried outside the domain that owns the destination.
- `prompt-manager skill read vrooli-ui-interop` §4.6 — owns gamepad / spatial-nav. This skill declares `reachable_via` in spec; vrooli-ui-interop owns the runtime mechanics.
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.

Required reading (programmatic enforcement substrate):
- `flow-verifier flows --help` and `flow-verifier verify --help` — the scenario that loads, validates, reconciles, verifies, and renders navigation specs. This is the primary destination as this skill becomes more programmatic over time.

Read first when present:
- `scenarios/{{TARGET}}/ui/flow/navigation.json` — the navigation spec, if one exists.
- `scenarios/{{TARGET}}/ui/src/routes.generated.ts` — generated from the spec; nothing else should declare route paths.
- `scenarios/{{TARGET}}/docs/internal/EXPERIENCE-AUDIT.md` — Navigation section: prior findings and maturity status.
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` — overlay/disclosure boundaries that should match container declarations.
- `scenarios/{{TARGET}}/docs/internal/INVARIANTS.md` — auth/role/context invariants the spec encodes.
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — accepted navigation debt and known unreachable/over-reachable surfaces.

---

### 1. Scope Boundaries

This skill owns the **verifiable contract** layer of navigation. Structural problems (wrong groupings, missing feature surfaces, broken information architecture) belong to `experience-architecture-audit`; gamepad/spatial mechanics belong to `vrooli-ui-interop`.

**In scope:**
- declaring/maintaining `ui/flow/navigation.json` (routes, containers, affordances, presentations, overlays, return paths, shortcuts, reachability invariants, deep-link policy)
- enforcing label↔destination truthfulness via Vitest+RTL tests that consume the spec
- enforcing code↔spec coherence via `flow-verifier flows reconcile`
- enforcing reachability invariants and deep-link policy via `flow-verifier verify run`
- routing findings to existing docs (`EXPERIENCE-AUDIT.md`, `SEAMS.md`, `INVARIANTS.md`, `PROBLEMS.md`) by lens
- proposing flow-verifier schema/checker enhancements when a recurring manual finding could become declarative

**Out of scope (hand off):**
- the structural question of "are these the right places, grouped the right way?" → `experience-architecture-audit`
- gamepad reachability, focus rings, spatial-nav focus groups → `vrooli-ui-interop` §4.6 (declare `reachable_via: ["gamepad"]` in spec; mechanics live there)
- temporal/lifecycle behavior inside a page (loading, retries, polling) → `temporal-flow-audit`
- folder/file shape under `ui/src/` → `screaming-architecture-audit`
- comment discipline, function shape inside route components → `cognitive-load-reduction`
- new product features, behavior changes, performance refactors
- creating standalone `*_AUDIT.md` or dedicated `NAVIGATION.md` files — findings route to existing docs

If `experience-architecture-audit` has not yet been run on this scenario, note that as a prerequisite gap rather than fixing it inside this skill.

---

### 2. Navigation Integrity Maturity Model

Assess each surface (each declared `flowId` in `ui/flow/`) independently. A scenario may be Level 4 in the main app navigation and Level 1 in a recently-added feature surface.

| Level | Name | What exists | Where it's verified |
|---|---|---|---|
| 0 | Unmodeled | Routes live in `App.tsx` as string literals; labels/links/destinations only knowable by reading code; reachability and deep-link behavior are implicit. | — |
| 1 | Inventory | The Navigation section of `EXPERIENCE-AUDIT.md` lists every URL surface, container, and known reachability/deep-link gap, with `path:` links. | `grep` `EXPERIENCE-AUDIT.md` for entries dated this pass. |
| 2 | Declared spec | A schema-valid `ui/flow/navigation.json` declares every route, container, affordance (with per-container presentations), overlay, return path, shortcut, reachability invariant, and deep-link policy. | `flow-verifier flows validate --kind navigation` passes. |
| 3 | Code↔spec coherence | `routes.generated.ts` is generated from the spec and is the only place route paths are declared in `ui/src/`. Every `<Route path=>`, `<Link to=>`, and `useNavigate(...)` resolves to a registered route id. No orphans either direction. | `flow-verifier flows reconcile --flow <id>` returns zero discrepancies; `grep -rE "(to=\"/|navigate\\(\"/)" ui/src/` returns only `ROUTES.*` / `ROUTE_PATTERNS.*` consumers. |
| 4 | Behavioral conformance | Reachability invariants and deep-link policy pass on the static graph. Vitest+RTL asserts every spec affordance renders with declared label and resolves to the declared destination. At least one BAS flow walks the spec end-to-end (click each affordance, assert URL change, back-path preserves history, deep-link + refresh recovers state, overlays trap focus). | `flow-verifier verify run --flow <id>` passes; `pnpm test` covers each affordance + overlay; BAS flow green. |
| 5 | Programmatic drift gate | The Flow Studio descriptor for this surface is part of the scenario's Studio inventory. CI runs `flows validate` + `reconcile` + `verify run` and fails on drift. New routes/affordances cannot be added without spec changes (lint or pre-commit rejects raw path strings). | `make test` (or CI equivalent) includes the three flow-verifier checks; raw-path-string lint rule active in `eslint.config.js`. |

Do not treat the level as a score to inflate. Use it to identify the next concrete move: declare what's missing in the spec, then close the next reconcile diff, then add the next invariant.

---

### 3. Finding-Routing Table (No New Docs)

Every navigation finding belongs to one of the existing audit docs. Never create a standalone navigation report or a new file for these findings.

| Finding | Routes to | Lens that ultimately owns it |
|---|---|---|
| URL surface or feature page missing from the inventory | `EXPERIENCE-AUDIT.md` Navigation section | Navigation (this skill) |
| Label↔destination mismatch (`<Link to="/old">New Label</Link>`) | Fix in code; spec or code wrong — pick the truthful one. No doc entry unless deferred. | Navigation (this skill) |
| Affordance renders but spec declares it hidden under current context | Fix in code (or spec); record in `PROBLEMS.md` if the gap is accepted-for-now | Navigation (this skill) |
| Reachability invariant fails (`must_reach` over budget, or `must_not_reach` reachable) | Fix in code/spec; record counter-example in `PROBLEMS.md` if accepted-for-now | Navigation (this skill) |
| Deep-link policy gap (auth-required route renders without redirect) | Fix `RouteGate`/equivalent; ensure spec entry exists | Navigation (this skill) |
| Container's disclosure model (drawer, popover) lacks focus trap or `esc` dismiss | `SEAMS.md` (overlay scope boundary) + fix in code | Navigation (this skill) |
| Back/close behavior doesn't match declared `return_paths` rule | Fix in code or update spec to match honest behavior | Navigation (this skill) |
| Routes live in different domain than the feature that owns them | `ARCHITECTURE.md` | `screaming-architecture-audit` (hand off) |
| Information architecture itself is wrong (right routes, wrong groupings) | `EXPERIENCE-AUDIT.md` structural section | `experience-architecture-audit` (hand off) |
| Gamepad/spatial-nav focus-group or focus-ring issue | `EXPERIENCE-AUDIT.md` Navigation section, cross-reference `vrooli-ui-interop` §4.6 | `vrooli-ui-interop` (hand off) |
| Auth/role context model itself is wrong (not just spec wording) | `INVARIANTS.md` | Domain owner / `invariant-discovery-and-enforcement` |
| Recurring manual finding that could be a declarative schema field or checker rule | Open a flow-verifier backlog entry | Flow-verifier (capability promotion) |

Rule: if the finding is structural or about runtime mechanics owned by another skill, **stop here and hand off**. Do not paper over an experience-architecture or gamepad-mechanics problem with a navigation spec patch.

---

### 4. Canonical Spec Shape

When a scenario has more than a handful of routes, declare them in a spec rather than implicit handler code.

```text
scenarios/{{TARGET}}/
  ui/flow/
    navigation.json                # single source of truth
  ui/src/
    routes.generated.ts            # emitted by `flow-verifier flows codegen`
    App.tsx                        # consumes ROUTE_PATTERNS.*; no raw path strings
    components/
      RouteGate.tsx                # enforces deep_link_policy from spec
      Navigation.tsx               # renders affordances filtered by context
```

The spec owns:
- **contexts** — declared conditional dimensions (`auth`, `role`, `viewport`, feature flags). `requires` (routes) and `show_when` (containers, affordances, presentations) are predicates over contexts. Predicate DSL: `AND`/`OR`/`NOT`/`=`/`!=`/`IN`/`CONTAINS`.
- **routes** — URL targets with `requires`, `redirect_if_unmet`, `deep_link` recoverability, `parents` for back-path inference.
- **containers** — layout-bearing elements (`persistent`/`drawer`/`popover`) with their own visibility rules, disclosure model (`always_visible`/`click_to_open`/`hover_to_open`), and focus-trap/dismiss semantics.
- **affordances** — logical intents with one `to` destination and one or more **presentations** describing how they appear in each container. The same `nav_settings` affordance can present in `top_nav_bar`, `bottom_nav`, and `hamburger_menu` — the verifier knows they're the same intent and the disclosure cost is added to reachability automatically.
- **overlays** — modals/drawers/dialogs that aren't navigation targets (confirm-logout, delete-confirmation).
- **return_paths** — per-route back-rule (`history_back`/`canonical_parent`/`prompt`) with fallback for missing history.
- **shortcuts** — keyboard bindings, scope (global/local), `excluded_routes`, optional `show_when`.
- **reachability_invariants** — policy assertions: under a given context, certain routes must (or must not) be reachable from a start route, within a viewport-keyed click budget (`max_clicks: { "desktop": 1, "mobile": 2 }`).
- **deep_link_policy** — direct-entry resolution rules (auth-required → redirect to login with preserved target; admin-only → redirect non-admins; etc.).

The spec does **not** own: animation/transition/CSS, real auth machinery, gamepad runtime mechanics. Container declarations include `reachable_via: ["mouse", "keyboard", "gamepad"]` per presentation; gamepad behavior is implemented in `vrooli-ui-interop` substrate.

Canonical reference fixture: `scenarios/flow-verifier/api/internal/flows/schemas/examples/navigation-full.json` (embedded). It exercises every schema field — read it before authoring a new spec.

---

### 5. Reachability, Deep-Link, and Spec Conformance

For Level 4+ surfaces, prove navigation completeness with checks that fail on drift:

**Reachability invariants.** Every gated destination has both a positive and a negative invariant:
- positive: `must_reach: ["settings_index"]` from `home` given `auth=logged_in` within `max_clicks: { desktop: 1, mobile: 2 }`
- negative: `must_not_reach: ["admin_users"]` from `home` given `auth=logged_in AND role!=admin`
- coverage rule of thumb: every route with a non-trivial `requires` predicate has at least one negative invariant proving it's unreachable when the predicate fails.

**Deep-link policy.** Every route whose `requires` references a gated context has a matching `deep_link_policy` entry declaring the redirect target. Spec validation should fail without it.

**Code↔spec reconciliation.** Every `<Route path=>` in `App.tsx` matches a spec route. Every `<Link to=>` and `useNavigate(...)` argument resolves to a registered route id (consumed via `ROUTE_PATTERNS.*`). Every spec affordance renders in its declared container with its declared label and resolves to its declared destination.

**Behavioral conformance (component tier — Vitest+RTL).** For each affordance: render the host container under the declared context, assert the element exists with the declared label and `test_id`, click it, assert the route changed. For each overlay: render at host route, trigger, assert focus trap, dismiss via each declared `dismiss` mechanism.

**Behavioral conformance (E2E tier — BAS).** Walk the spec end-to-end. Click each affordance, assert URL change. Verify back-path preserves history. Deep-link + refresh recovers state for `deep_link: "recoverable"` routes. Modal overlays trap focus; nested overlays restore prior scope on close.

Coverage percentages are not proof of navigation completeness. A suite can render every component while never asserting that an admin link is hidden from a non-admin.

---

### 6. CLI Commands (programmatic substrate)

| Command | Purpose | When to run |
|---|---|---|
| `flow-verifier flows list --kind navigation --root <scenario>` | Inventory navigation specs in this scenario | Start of every audit |
| `flow-verifier flows validate --kind navigation --root <scenario>` | Schema + structural validation (no orphan refs, predicates parse) | After every spec edit |
| `flow-verifier flows codegen --flow <id> --root <scenario> --write` | Emit `routes.generated.ts` | After adding/renaming routes |
| `flow-verifier flows reconcile --flow <id> --root <scenario>` | Walk `ui/src/`; assert every `<Route path=>`/`<Link to=>`/`useNavigate` resolves; flag orphans either direction | After every code or spec edit touching routes |
| `flow-verifier verify run --flow <id> --root <scenario>` | Reachability invariants + deep-link policy on the static graph | After every spec edit or context-model change |
| `flow-verifier flows studio --flow <id> --root <scenario>` | Print Studio descriptor for visual review (routes, affordances, containers, context toggles, invariant pass/fail) | When eyeballing a graph change |
| `flow-verifier flows new <ui-dir> --flow-id <id> --kind navigation --root <scenario>` | Scaffold a minimal valid `navigation.json` for a new surface | When introducing a new feature surface |

If a recurring manual check has no CLI surface, **add it to flow-verifier rather than working around it in this skill** — that's how the skill becomes more programmatic over time.

---

### 7. Audit Checklist

Map every URL surface:
- routes, sub-routes, parameter-bearing pages
- containers and their disclosure models
- overlays (modals, drawers, dialogs) and their dismiss semantics
- shortcuts and their scopes
- gated destinations (auth, role, feature flag, viewport)

For each, identify:
- declared in spec? (or just in code)
- reconcile clean against `ui/src/`?
- reachability invariants present (positive + negative for gated routes)?
- deep-link policy covers it?
- component-tier test asserts label + destination?
- E2E flow walks it?
- current maturity level
- next maturity step

Then improve the highest-risk gaps first:
- introduce a spec where none exists; codegen `routes.generated.ts`; replace raw path strings
- close reconcile diffs (orphan routes, orphan links, label drift)
- add negative reachability invariants for every gated route
- add deep-link policy entries for every auth/role/flag-gated route
- ensure overlays have focus traps and matching dismiss declarations
- record accepted exceptions in `PROBLEMS.md` with rationale

Avoid large risky rewrites in one loop. If the correct redesign is too broad (e.g., the IA itself is wrong), hand off to `experience-architecture-audit` and document the candidate.

---

### 8. Documentation

Use `knowledge-observatory-tools` to read and update the **Navigation** section of `scenarios/{{TARGET}}/docs/internal/EXPERIENCE-AUDIT.md`.

This section is an index and memory layer, not the detailed spec source of truth. Detailed routes/containers/affordances/invariants belong in `ui/flow/navigation.json` and pass through `flow-verifier`.

Recommended shape for the EXPERIENCE-AUDIT.md Navigation section:

```markdown
## Navigation

### Surfaces

| Flow ID | Surface | Spec | Maturity | Reconcile | Verify | Last audited |
|---|---|---|---|---|---|---|

### Unmodeled Surfaces

| Surface | Why it should be modeled | Current risk | Recommended next step |
|---|---|---|---|

### Accepted Exceptions

| Finding | Why accepted | Cleanup trigger |
|---|---|---|

### Audit Notes

- [Date] [Agent/author]: [Short note with evidence and links.]
```

When updating:
- verify existing claims against `ui/flow/navigation.json` and `flow-verifier` output before extending them
- link to source files with `path:` references
- record unmodeled surfaces rather than leaving discoveries in chat
- keep long route/affordance tables out of this doc once the spec exists — the spec is the source of truth
- create `path:docs/internal/` if needed

---

### 9. Output Expectations

By the end of this loop, the scenario should:
- have a clearer inventory of URL surfaces in `EXPERIENCE-AUDIT.md`
- move at least one surface up the maturity ladder
- have fewer hidden label↔destination drifts and unreachable/over-reachable gaps
- keep navigation rules in `ui/flow/navigation.json` rather than scattered across `App.tsx`, link components, and gate components
- have executable validation (`validate` + `reconcile` + `verify run`) for important navigation surfaces
- leave future agents with a lower-drift `EXPERIENCE-AUDIT.md` Navigation section

Avoid superficial edits that rename or reshuffle navigation code without improving spec coverage or moving up the maturity ladder.
