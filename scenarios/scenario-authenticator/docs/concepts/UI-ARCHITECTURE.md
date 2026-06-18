# UI Architecture

> **Status: documentation-first orientation.** The UI is the template
> scaffold today (only `health` and the fenced `notes` example exist).
> The three audiences and feature folders below are the **target** UI
> design from [`../../PRD.md`](../../PRD.md); none are built yet.

## Purpose Of This Document

Describe the canonical layout of the `ui/` source tree for scenarios generated
from the `react-vite` template, the **slot taxonomy** that lets external
tools (notably `react-component-library`'s adoption resolver) place components
without asking the user for a path, and the **three UI audiences** this
authenticator serves from one deployment.

## Three Audiences, One Deployment

The UI serves three distinct audiences from a single binary/deployment.
They share the design-token plumbing, i18n, accessibility primitives, and
the same-origin API client, but are different feature surfaces with
different density and trust requirements (PRD §UX & Branding):

| Audience | What it does | Primary domains | Tier | Surface character |
|---|---|---|---|---|
| **Admin console** | Manage realms, users, roles/scopes, sessions, and audit. | realms, identity, authorization, sessions, audit | P0 baseline → P1 polished (OT-P1-007) | Dense, data-forward; **destructive-action confirmation gates** (realm deletion, session revoke). |
| **End-user self-service** | Edit profile, enroll/manage MFA, review and revoke active sessions, manage connected accounts, change password. | identity, mfa, sessions, federation | P1 (OT-P1-008) | Minimal, task-focused. |
| **Hosted login / consent screens** | Sign-in, registration, MFA challenge, OAuth consent — the screens RPs redirect to or embed same-origin. | identity, mfa, federation | P0 (basic) → P1 (per-realm branding) | Clean, trustworthy, fast; **per-realm branding** (logo, colors) at P1. |

Each audience is built as feature folders under `ui/src/features/<name>/`
(see [Source Layout](#source-layout)) — e.g. an admin realm-management
feature, a self-service MFA-enrollment feature, and a hosted sign-in
feature — each with a typed Connect client wrapper under
`ui/src/api/<domain>.ts`. They compose the product domains in
[`DOMAINS.md`](DOMAINS.md); the audiences are UI surfaces, not domains.

## Same-Origin Only

The UI talks **only to its own scenario's API, same-origin** — it never
makes a cross-origin browser call to anything, including (when this UI is
embedded by an RP) the authenticator from another origin. This is the
same rule RPs follow (see [`INTEGRATIONS.md`](INTEGRATIONS.md)): a
browser reaches the authenticator only through an API on its own origin.
Hosted login/consent screens are served by this scenario's own API, so
they are same-origin by construction.

## Design Tokens, i18n, Accessibility

- **Design tokens** flow through the `theme-token` slot
  (`ui/src/theme/` — `ThemeProvider` + `tokens.css`), delivering light
  and dark themes from the vrooli-default operational-console kit.
  Per-realm branding overrides (logo, colors) on hosted login/consent
  screens are a P1 deliverable layered over these tokens.
- **i18n** uses the `i18n-strings` slot (`ui/src/i18n/locales/`, one JSON
  per locale); user-facing copy is keyed, not inline. Error messages
  relay validation reasons faithfully **without leaking account
  existence** where that would aid enumeration.
- **Accessibility: WCAG AA across all surfaces.** Full keyboard
  navigation for every auth and management control — including MFA
  enrollment, session revocation, and OAuth consent. Template
  primitives are preserved: `role`, `aria-*`, and `data-testid`
  selectors. Interactive security prompts (MFA challenge, revoke
  confirmation, "new sign-in from an unrecognized device") must meet
  focus-management and announcement requirements so screen-reader users
  get the same timely security information as sighted users.

## Source Layout

```
ui/src/
├── api/            # api-client slot — Connect-RPC wrappers
├── app/            # app-bootstrap — Providers composition and route table
├── components/     # shared-component slot — cross-cutting components
│   └── ui/         # ui-primitive slot — headless primitives (kebab-case files)
├── consts/         # consts slot — strings + selectors registries
├── features/       # feature slot — per-feature folders (one subfolder per feature)
│   └── <feature>/  # feature-component slot — components inside a feature
├── hooks/          # hook slot — reusable React hooks
├── i18n/           # i18n bootstrap
│   └── locales/    # i18n-strings slot — one JSON per locale
├── layout/         # layout-shell + layout-nav slots — AppShell, Sidebar, TopBar, BottomNav
├── lib/            # lib-util slot — framework-agnostic utilities
├── pages/          # page slot — routed pages mounted under <Outlet />
├── test-utils/     # test-util slot — render helpers, factories, a11y
└── theme/          # theme-token slot — ThemeProvider + tokens.css
```

## Slots Are A Contract

Every directory above maps to a named slot in `ui/manifest.json`. The manifest
declares the directory **and** a default path pattern (e.g.
`{dir}/{ComponentName}.tsx`), so external tools can compute the canonical
filesystem path for a new file given just the component's name and slot.

A component library that publishes `"slot": "layout-nav"` and ships
`SidebarShell` knows — without any per-scenario configuration — that the file
should land at `ui/src/layout/SidebarShell.tsx`. Override the slot's `dir` in
a scenario-level overlay if you've reorganized; the resolver will pick up the
new path automatically.

## Adoption Resolver Flow

1. Library declares the component's slot (e.g. `"slot": "layout-nav"`).
2. Resolver looks up the slot in this scenario's UI manifest (this file's JSON
   sibling).
3. Resolver substitutes path-pattern tokens (`{dir}`, `{ComponentName}`,
   `{kebab-name}`, `{camelName}`, `{feature}`, `{locale}`) and returns the path.
4. Scenarios with no manifest fall through to a heuristic (scan for the slot's
   expected dir name) and then a final fallback
   (`ui/src/components/<ComponentName>.tsx`). Both flag warnings on the
   resulting adoption record.

## Extending The Manifest

- **Add a slot.** Add an entry to `ui/manifest.json`. Keep its `dir` inside
  `ui/src/` and pick a pattern that matches your file-naming convention. The
  schema (`scenario-ui-manifest/v1`) does not enum-restrict slot names — open
  set on purpose.
- **Override a slot in a single scenario.** Drop a partial manifest at
  `.vrooli/ui-manifest.json` in the scenario root; the resolver will read it
  as an overlay over the template manifest. (Overlay support tracked in
  scenarios/react-component-library's PRD.)
- **Add a `postApply` action** (auto barrel-export, route-register,
  i18n-merge). Reserved for a future schema bump (`scenario-ui-manifest/v2`).
  Document the intent in the consuming scenario's PRD until then.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape, IdP role, same-origin rule
- [`DOMAINS.md`](DOMAINS.md) — the product domains these audiences compose
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — no-cross-origin rule and RP contract
- [`../../PRD.md`](../../PRD.md) — UX & Branding, OT-P1-007/008
- Schema: `.vrooli/schemas/scenario-ui-manifest.schema.json` (`$id:
  scenario-ui-manifest/v1`)
- Manifest: `ui/manifest.json`
- Slot reference: [`ui-manifest.md`](../reference/ui-manifest.md)
- Adoption resolver: `scenarios/react-component-library/api/internal/adoptions/pathresolver.go`
