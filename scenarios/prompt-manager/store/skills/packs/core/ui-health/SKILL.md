## Steer focus: UI Health

Prioritize **making the scenario's UI render correctly and embed safely across every Vrooli deployment context** in `scenarios/{{TARGET}}/ui/`. ui-health is the single authority for UI validation — manifest contract, static interop, project-UI standards (i18n / design tokens / accessibility / favicon-PWA / strict-config), bundle freshness, and live runtime render — and it ships deterministic auto-fixers for the safe mechanical subset. This skill routes you to that authority, then helps you reason about and remediate what it reports.

This skill replaces the retired `react-stability`, `react-coherence`, and `vrooli-ui-interop` skills. Their remediation guidance lives below as lenses; their rubrics now live in the provider (`ui-health`'s rule engine + `.vrooli/maturity.json`), not in prose.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.

Read first when present:
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — durable mental model and UI surface map.
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — unresolved UI drift and deferred fixes.

Hands off to (stays separate):
- `prompt-manager skill read ux` — stylistic / visual-judgment work (layout taste, copy, micro-interactions). ui-health validates *correctness and standards*; `ux` owns *taste*.

---

### 0. Programmatic Validation — run this first

Do not hand-audit UI conformance. Take a **photograph** with the provider; its findings (each carries a code, severity, location, and remediation) are the single source of truth. **Do not re-derive or duplicate the L0–L5 ladder here — it lives in `scenarios/ui-health/.vrooli/maturity.json`.**

```bash
# Full static report (manifest + interop + project-UI standards + freshness + AI surface).
# No browser, no auto-start — the fast "quick check" mode every caller can run.
ui-health validate scenario {{TARGET}} --static-only --json

# Full report INCLUDING the live runtime/render group (drives the UI through
# browser-automation-studio over Connect). Auto-starts the target when needed.
# When BAS or the UI is unavailable the runtime checks report skipped, never failed.
ui-health validate scenario {{TARGET}} --json

# The phase-pipeline view (what test-genie runs as the `ui-health` phase, execution-mode):
test-genie execute {{TARGET}} --json
```

Then **let the fixers do the mechanical work** before you touch anything by hand:

```bash
# Preview every safe deterministic fix (dry-run; writes nothing):
ui-health fix run {{TARGET}} --json

# Apply a specific fixer:
ui-health fix run {{TARGET}} --rule standard_tsconfig_strict --apply
ui-health fix run {{TARGET}} --rule standard_i18n_locale_parity --apply

# Or sweep every provider's deterministic fixes through test-genie:
test-genie fix {{TARGET}} --deterministic --apply
```

A finding's `fix_class` tells you the posture: `auto` (a fixer exists — apply it), `detection_only` / absent (report-only — fix by hand using the lenses below). Never hand-edit something a fixer owns; never expect a fixer for a judgment call (hex→token, plural-category synthesis, authoring a test).

The manual lenses below are how you reason about and fix each finding the provider surfaces; §0 is how you discover the work and prove it resolved (re-run until clean).

---

### 1. Scope Boundaries

**In scope:**
- UI manifest contract (template/overlay/slot alignment) and its drift.
- Deployment-context interop: iframe-bridge dep/init/guard/appId, vite relative base, proxy-aware router basename, single API-base resolution, no hardcoded localhost, shortcut relay, iframe-safe scroll/viewport, spatial navigation.
- Project-UI standards: i18n locale parity, design-token usage (no raw hex), accessibility test harness, favicon/webmanifest/viewport, tsconfig strict + react-stability ESLint.
- Crash-prevention stability: hook discipline, defensive data access, error-boundary placement, runtime validation at boundaries.
- Coherence: scope-driven state architecture, the sharing decision tree, styling/theming organization.
- Live runtime render correctness (handshake, console errors, asset/proxy load) via the provider's runtime group.

**Out of scope:**
- Stylistic / visual-design judgment → `ux`.
- Proto-first API contracts between services → `interoperability-steer`.
- Dead links / orphan routes UX correctness → `navigation-integrity-audit`.
- Shared-package internals (api-base, iframe-bridge themselves) → `platform-scope`; this skill covers *consuming* them.
- Inventing new UI features under the banner of conformance cleanup.

---

### 2. The Core Model

#### 2.1 The seven check groups (what the provider reports)

| Group | What it asserts | Posture |
|---|---|---|
| Static manifest | template/overlay/slot contract valid and aligned on disk | mostly auto-fixable (missing slot dirs) |
| Static interop | deployment-context correctness (slots [A]–[I], scroll/viewport, spatial nav) | safe subset auto-fixable; rest report-only |
| Project-UI standards | i18n parity, no-raw-hex, a11y harness, favicon/PWA, strict-config, stability lint | strict-flip + i18n key-scaffold auto; rest report-only |
| Freshness | the built bundle matches source (content-hash, not mtime) | report-only (rebuild) |
| Runtime/render | live render through BAS: handshake, console, assets | report-only; **skipped, not failed, when BAS/UI down** |
| AI surface search | embeddings/provenance of the UI surface | informational |
| Auto-fix | the deterministic remediation universe | drives `fix run` |

#### 2.2 The three deployment contexts (the *why* behind interop)

Every Vrooli scenario UI must behave identically in three contexts without conditional branches in component code:

1. **localhost** — direct dev server.
2. **Cloudflare tunnel** — public origin, root-mounted.
3. **app-monitor proxy / iframe** — served at `/apps/<name>/proxy/`, embedded in the host frame.

The interop layer (the slots) absorbs all context differences so components just call `buildUrl("/endpoint")` and `navigate("/page")` and never check whether they're proxied. **Rule: no `if (isProxied)` branches in component code** — if you find one, the fix is to push the difference down into the interop slot, not to special-case the component.

---

### 3. Finding → Lens Decision Model

Walk the table by the finding's code family. Two agents reading the same report must land in the same lens.

| Finding code family | Lens | Fix posture |
|---|---|---|
| manifest (`slot_*`, `overlay_*`, `contract_*`) | §4.1 Manifest | auto (slot dirs) / else align by hand |
| `interop_*` (bridge/base/router/api-base/localhost/server) | §4.2 Interop | safe subset auto; else slot pattern |
| `interop_banned_scroll`, `interop_h_screen` | §4.3 Iframe-safe scroll/viewport | h-screen auto; scrollIntoView report-only |
| `interop_spatial_nav_init`, `interop_focus_visible_styles` | §4.4 Spatial nav & focus | manual |
| `standard_i18n_locale_parity` | §4.5 i18n | non-plural keys auto; plurals/orphans manual |
| `standard_no_raw_hex`, theming | §4.6 Tokens & theming | report-only (hex→token is judgment) |
| `standard_a11y_harness` | §4.7 Accessibility | manual (author the harness) |
| `standard_pwa_manifest` | §4.8 Favicon/PWA | manual (insert standard tags) |
| `standard_tsconfig_strict`, `standard_eslint_stability` | §4.9 Strict config | strict-flip auto; eslint manual |
| `runtime_*` (handshake/console/render) | §4.10 Runtime render | report-only — debug the live UI |

---

### 4. Remediation Lenses

These fold the durable guidance from the retired React skills. Each rule's own remediation/GoodExample/BadExample is in the provider output — these lenses carry the cross-cutting wisdom a single rule can't.

#### 4.1 Manifest
A missing slot directory is auto-created by the fixer. An overlay/contract mismatch means the scenario's `ui/manifest.json` disagrees with the template it claims — align the overlay, don't fork the template. "Predates template layout" collapses to one summary finding; treat it as a re-scaffold decision, not per-slot whack-a-mole.

#### 4.2 Interop — the convergence pattern (slots)
The healthy UI converges on a fixed set of slots, each absorbing one deployment-context difference:
- **[A] `ui/package.json`** — depends on `@vrooli/api-base` + `@vrooli/iframe-bridge`.
- **[B] `ui/vite.config.ts`** — `base: './'` (relative assets load under any mount path). Carry an `INTEROP-CRITICAL` comment.
- **[C] `ui/server.js`** (if present) — use `startScenarioServer()`/`createScenarioServer()` from `@vrooli/api-base/server`; route API through `proxyToApi`. Never hand-roll Express.
- **[D] `ui/src/main.tsx`** — `initIframeBridgeChild({ appId, captureLogs, captureNetwork })` **guarded** by `window.parent !== window`. Keep capture on.
- **[E] `ui/src/App.tsx`** — `BrowserRouter` basename from a proxy-aware `getRouterBasename()` (returns `""` on localhost, `/apps/<name>/proxy` proxied).
- **[F] `ui/src/api/client.ts`** — a single `resolveApiBase()` (≤2 production files); never rebuild its output with `window.location.origin`.
- **[G] shortcuts** — relay via `emitShortcutIntent()`; no app-level `keydown` scattered outside hooks/dismissible components.

When a finding flags one of these, the fix is to adopt the slot pattern, not to special-case the symptom. The safe-subset fixers handle `base: './'`-comment insertion (protective comments) and `h-screen`→`h-full`; bridge-guard wrapping and api-base rewrites are report-only because they touch behavior.

#### 4.3 Iframe-safe scroll & viewport
Inside the host iframe, several DOM APIs traverse the boundary and shift the **host** document. `scrollIntoView()` is the common offender — it scrolls every scrollable ancestor including ones outside the iframe; under the host's `overflow:hidden` shell the content shifts up with no recovery. Use `container.scrollTo(...)` scoped to your own scroll container. For sizing, viewport units (`h-screen`/`w-screen`/`100vh`/`100vw`) resolve against the outer window — use `h-full`/`w-full`/`100%` with a height chain to the root. (`h-screen` is auto-fixable; `scrollIntoView` is report-only — the safe rewrite depends on which container you mean.)

#### 4.4 Spatial navigation & focus
Call `initSpatialNav()` in `main.tsx` (or declare the `// spatial-nav: disabled` opt-out). Overlays (Dialog/Drawer/FloatingPanel) must trap focus scope (`pushScope`/`popScope` or `<SpatialGroup mode="modal">`). Interactive elements need a visible focus indicator (`focus-visible:` classes, a global `:focus-visible` policy, or `[data-spatial-focus]`) so keyboard/gamepad users can navigate inside the frame.

#### 4.5 i18n parity
Every non-`en` catalog under `ui/src/i18n/locales` must carry the same **base** key set as `en.json`. The check is plural-aware: i18next CLDR plural variants (`items_one`/`items_other`/…) collapse to a base key, so locales with different plural categories are *not* drift. The fixer scaffolds missing **non-plural** keys (English value as a placeholder). Missing plural concepts and orphan keys are left to you — synthesizing the right CLDR plural set and deleting translations are judgment calls. Keep `SUPPORTED_LOCALES`/`LOCALE_CODES` and `LOCALE_CONFIG` in sync (TypeScript enforces it) and persist the choice to `vrooli.locale`.

#### 4.6 Design tokens & theming
Colors live in the token layer (`theme/tokens.css`, the generated `design-tokens.css`), consumed via `var(--surface-*)`/`var(--text-*)` or Tailwind token classes — never raw hex in components (that freezes color across dark-mode and design-kit swaps). The no-raw-hex finding is report-only: replace each literal with the right semantic token (a mapping is a judgment call). `ThemeProvider` toggles `data-theme` on `<html>`; dark mode is a token re-bind, not per-component overrides.

#### 4.7 Accessibility
Ship the harness: an axe-based dep (`axe-core`/`jest-axe`/`vitest-axe`) **and** at least one `*.a11y.test.tsx` asserting no violations on the app shell (template ref: `ui/src/test-utils/a11y.ts`). The static check verifies the harness exists; the runtime group runs the live axe sweep. Authoring tests is manual — no fixer.

#### 4.8 Favicon / PWA
`ui/index.html` needs a responsive viewport meta, a `rel="manifest"` link, and an icon link; the webmanifest needs `"display":"standalone"` (template ref: `ui/index.html` + `public/site.webmanifest`). Insert the standard tags by hand.

#### 4.9 Strict config & stability lint
`ui/tsconfig.json` must set `"strict": true` (pair with `noUncheckedIndexedAccess`) — the fixer flips an explicit `false`; an absent flag is added by hand. The ESLint flat config must set `react-hooks/rules-of-hooks`, `import/no-cycle`, `@typescript-eslint/no-explicit-any`, and `@typescript-eslint/no-non-null-assertion` to `"error"` (warn is ignored; only error blocks the build). These are the crash-prevention net: strict catches null/any/implicit-this; the lint rules catch conditional hooks (state corruption), circular imports (undefined-at-load), `any` escapes, and `!` papering over real nulls.

#### 4.10 Runtime render (stability at runtime)
`runtime_*` findings come from driving the live UI through BAS. A failed handshake means the bridge never initialized (check slot [D] guard + appId). Console errors / page errors point at crash sources — apply defensive data access (optional chaining, `??` defaults, type guards before property access; never `!`) and strategic error boundaries so one feature's crash doesn't blank the app. A render-broken screenshot with a clean handshake usually means an asset/proxy base problem (slot [B]/[F]). These are report-only — debug the running UI; the provider gives you the screenshot + console/network evidence.

---

### **5. Output Expectations**

Re-run §0 until the report is clean (or only report-only/known findings remain, recorded). Then:
- Record durable UI model changes in `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` (surfaces, slots adopted, theme/i18n decisions).
- Record unresolved drift and deferred report-only findings in `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md`.
- **Do not create a standalone `UI_HEALTH_AUDIT.md`** — a one-off report freezes a single session's view and rots. Use `knowledge-observatory-tools` to fold findings into the durable docs.

The goal is a UI that renders identically on localhost, tunnel, and proxy/iframe; passes the project-UI standards; and self-heals the mechanical subset via the fixers — verified by a clean `ui-health validate scenario {{TARGET}}`, not by prose assertion.
