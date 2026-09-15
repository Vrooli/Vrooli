# `branding` phase

The `branding` phase is a **thin delegating phase**: it calls the
`brand-manager` scenario's shared `ScenarioValidationService` and maps
scenario-scoped brand-identity findings into the shared
`FINDING_SOURCE_BRANDING` channel. test-genie does not inspect a scenario's
branding itself; those checks live in `brand-manager` — the single scenario that
both authors and validates branding — alongside its brand-authoring API/CLI/UI.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

At maximum maturity the scenario is **accessible and visibly branded on every surface it exposes**: a real display name that agrees across manifest, document, and metadata; a reusable color-and-typography token system with an intentional typeface; publishable marks (logo, favicon, home-screen icon) served by convention; light- and dark-mode contrast that meets WCAG AA with declared color-scheme and reduced-motion support; complete install metadata so it launches as a branded web app; and branded share, CLI, and API surfaces. The culminating capability, `accessibility_application`, reaches L3 "Brand applied" — accessible brand tokens are not merely declared but demonstrably applied through brand-manager markers or manifest metadata.

## The rungs and their gates

Each of the six capabilities carries its own monotone L0→L3 ladder (L0 is uniformly "target unresolvable / foundation absent"; each rung implies the one below). The table shows the climb from L1 to the top rung and the single next unlock.

| Capability | L1 (Foundation) | L2 (Ready) | L3 top rung — North Star | Next unlock from L1 |
|---|---|---|---|---|
| `identity_contract` | Display name declared | Identity cross-checked across surfaces | **Identity consistent** — declared and consistent everywhere | Consistent naming across manifest, document, and metadata |
| `visual_system` | Color-token system present | Typography declared | **Typeface intentional** — color + typography ready for consistent application | Typography choices that make the brand recognizable |
| `brand_assets` | Core marks (logo, favicon) present | Assets coherent (references resolve, no residue, SVGs safe) | **Assets publishable** across browser, install, and share surfaces | Valid referenced assets with template residue removed |
| `accessibility_application` | Light-mode contrast readable | Accessible theming (dark + color-scheme + reduced-motion) | **Brand applied** — accessible and visibly applied | Dark-mode contrast and browser color-scheme support |
| `install_surface` | Mobile chrome branded (theme color + standalone) | Install metadata complete (manifest, safe-area) | **Install surface clean** — consistently branded installable web app | Complete install manifest and iOS safe-area/status-bar handling |
| `share_and_product_surfaces` | Social metadata (OG + Twitter card) present | Share preview branded (approved image) | **Product surfaces branded** — share, CLI, and API all carry the brand | A branded social preview image |

## What each finding means

Each finding caps its capability at the named rung; only ERROR severities fail the phase (branding emits no BLOCKER), so nearly all branding findings are honest, non-failing hardening debt.

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `has-display-name` | identity_contract | L1 | ERROR | **Yes** |
| `name-consistency` | identity_contract | L3 | WARNING | No |
| `has-color-system` | visual_system | L1 | WARNING | No |
| `has-typography` | visual_system | L2 | INFO | No |
| `has-logo` / `has-favicon` | brand_assets | L1 | WARNING | No |
| `no-template-residue` / `referenced-assets-exist` / `svg-asset-safety` | brand_assets | L2 | WARNING | No |
| `wcag-aa-contrast` | accessibility_application | L1 | WARNING | No |
| `dark-mode-contrast` / `color-scheme-declared` / `reduced-motion-support` | accessibility_application | L2 | WARNING / INFO | No |
| `theme-color-present` / `standalone-capable` | install_surface | L1 | WARNING / INFO | No |
| `manifest-completeness` / `ios-statusbar-safe-area` | install_surface | L2 | WARNING | No |
| `open-graph` / `twitter-card` | share_and_product_surfaces | L1 | WARNING / INFO | No |

The full rule inventory (all 29 codes with per-surface applicability and auto-fix status) is the table in [Rules](#rules) below.

## The canonical fix

- **Identity (`has-display-name`, `name-consistency`)** → set a real, non-placeholder `service.displayName`, then align `<title>`, app-name, apple-title, and manifest `name` to it. Name consistency is auto-fixable (rewrite surfaces); the display name itself is manual product judgment.
- **Visual system (`has-color-system`, `has-typography`, `custom-font-loaded`)** → declare the canonical color tokens in `ui/src/design-tokens.css` (a baseline set is auto-created when none exists), then add heading/body font tokens. Font selection and self-hosting are design choices left manual.
- **Brand assets (`has-logo`, `has-favicon`, `no-template-residue`, `asset-validity`, `referenced-assets-exist`, `svg-asset-safety`, `apple-touch-icon`, `public-asset-convention`)** → ship a real logo and favicon, remove Vite scaffold residue (auto), repair dangling references, and relocate public assets under `/public/` (auto). Image transforms and SVG sanitizing stay manual.
- **Accessibility (`wcag-aa-contrast`, `dark-mode-contrast`, `color-scheme-declared`, `reduced-motion-support`, `brand-markers-applied`)** → re-balance palettes to pass AA in both schemes (manual design judgment); declare `color-scheme` and a `prefers-reduced-motion` block (auto); apply the brand via the apply domain.
- **Install surface (`theme-color-*`, `standalone-capable`, `ios-statusbar-safe-area`, `manifest-completeness`)** → inject theme-color, standalone metas, safe-area handling, and fill scalar manifest fields (all auto); icons need a logo and stay guidance-only.
- **Share / product (`open-graph`, `twitter-card`, `social-preview-image`, `cli-branding`, `api-branding`)** → derive OG/Twitter metadata from identity (auto); author a branded 1200×630 preview image and user-facing CLI/API copy (manual).

## How to verify

```bash
# See the current rung, gaps, and next move for every branding capability:
brand-manager validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases branding
test-genie runs findings --scenario <scenario>
```

The `branding` line in the scorecard shows the current rung, the single highest-unlock next move, and a runnable doc-search topic that resolves back to the sections above.

## What it runs

```text
scenario-validation/v1.ScenarioValidationService.ValidateScenario
```

Test Genie reads the shared `status`, `assessment.local`, and
`assessment.findings` fields. Each assessment finding is mapped to an
`ArchitectureFinding{Source: FINDING_SOURCE_BRANDING}`, so it carries a
deterministic stable ID, normalized severity, and the per-source effort default.
The phase summary carries brand-manager's `current_level`, `next_level`,
`clean`, and `unknown_count` convergence signals; phase pass/fail still comes
only from `status`.

The phase is **optional**: when `brand-manager` is not running or has no
branding to assess for the target, the phase skips rather than fails.

## Rules

brand-manager owns the rule semantics and severity tier. Rules are
**surface-conditional**: UI rules skip silently when the target has no `ui/`, the
CLI rule skips without `cli/`, so CLI/API-only scenarios collect no false
positives. The branding maturity ladder (L0–L6) evaluates, per scenario:

| Rule | Surface | What it checks | Default severity | Auto-fix |
|---|---|---|---|---|
| `has-display-name` | any | `service.json` declares a non-placeholder display name | error | no |
| `has-color-system` | ui | canonical `ui/src/design-tokens.css` defines the core color tokens | warning | yes (create-only baseline when none) |
| `has-typography` | ui | heading + body font tokens are defined | info | no |
| `has-logo` | ui | a logo asset is present | warning | no (image generation is non-deterministic) |
| `has-favicon` | ui | a favicon is present / referenced | warning | yes (derive from logo; else guidance-only) |
| `wcag-aa-contrast` | ui | light-scheme color pairings (incl. primary-foreground-on-primary, accent/state) meet WCAG 2.1 AA | warning | no |
| `brand-markers-applied` | ui | applied CSS `/* brand-manager:* */` markers / manifest `_brand` keys are present | info | no (brand-projection — needs an assigned brand) |
| `dark-mode-contrast` | ui | the same pairings pass under the dark-scheme overrides | warning | no |
| `color-scheme-declared` | ui | a shipped dark scheme declares `color-scheme` | info | yes (inject meta) |
| `name-consistency` | ui | `<title>`/app-name/apple-title/manifest name agree with `service.displayName` | warning | yes (rewrite surfaces) |
| `theme-color-consistency` | ui | `<meta theme-color>` agrees with manifest `theme_color` | warning | yes (align manifest) |
| `no-template-residue` | ui | no Vite scaffold favicon/title or lorem-ipsum copy | warning | yes (Vite favicon/title; lorem is manual) |
| `asset-validity` | ui | referenced icons non-empty, apple-touch-icon opaque, maskable icon declared | warning | no (image transform) |
| `theme-color-present` | ui | a `<meta theme-color>` exists (+ dark media variant when dark ships) | warning | yes (inject; dark variant guidance-only) |
| `standalone-capable` | ui | both `mobile-web-app-capable` + `apple-mobile-web-app-capable` are set | info | yes (inject metas) |
| `ios-statusbar-safe-area` | ui | translucent status bar pairs with `viewport-fit=cover` + `env(safe-area-inset-*)` | warning | yes (inject cover + safe-area token) |
| `manifest-completeness` | ui | manifest declares name/icons(192/512+maskable)/colors/display/start_url/id | warning | yes (scalars; icons need a logo) |
| `open-graph` | ui | `og:type/title/description/site_name` present | warning | yes (derive from identity) |
| `twitter-card` | ui | `twitter:card/title/description` present | info | yes (derive from identity) |
| `cli-branding` | cli | the CLI manifest surfaces the display name | info | no (human-facing copy) |
| `api-branding` | any | `service.description` (the API/OpenAPI title) is real, not template residue | info | no (human-authored copy) |

**Deferred (intentional coverage gaps, not silent):** design-system depth
(spacing/radius token completeness, color-format consistency), pixel/device-accurate
runtime rendering of the mobile status bar (that is ui-health / a device harness,
not static-declaration validation), and brand artwork generation (non-deterministic).

## Severity contract

This phase only normalizes the emitted severity string:

| brand-manager severity | normalized | feeds the `branding` dimension as a gap? |
|---|---|---|
| `SEVERITY_ERROR` | ERROR | **yes** |
| `SEVERITY_WARNING` | WARNING | advisory |
| `SEVERITY_INFO` | INFO | advisory |

## Auto-fix

Deterministic rules expose `PreviewFix` (dry-run) and `ApplyFix` (writes) over
the same `ScenarioValidationService` contract. Non-deterministic gaps (e.g. logo
image generation, palette re-balancing) report a finding with guidance and no
fix candidate.

A finding's `autofix_available` flag is computed from whether the registered
fixer can actually produce a candidate for that scenario right now — the flag and
the fixer derive from one rule registry, so the contract can never advertise a
fix it cannot perform. Most fixers are **self-contained** (they derive the
correct value from the scenario's own `service.json` + existing assets), so
test-genie's brandless `ApplyFix` can remediate PWA/manifest/social/consistency
branding for any scenario without first assigning a brand.
