# `branding` phase

The `branding` phase is a **thin delegating phase**: it calls the
`brand-manager` scenario's shared `ScenarioValidationService` and maps
scenario-scoped brand-identity findings into the shared
`FINDING_SOURCE_BRANDING` channel. test-genie does not inspect a scenario's
branding itself; those checks live in `brand-manager` — the single scenario that
both authors and validates branding — alongside its brand-authoring API/CLI/UI.

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
| `has-color-system` | ui | design tokens define the core color tokens | warning | yes (create-only baseline when none) |
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
