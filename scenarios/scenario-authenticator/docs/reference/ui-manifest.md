# UI Manifest Reference — Scenario Authenticator

> **Current UI reference with planned extensions.** The shipped UI contains
> health, the shared shell, settings, and accessibility primitives. The
> three audiences, routes, feature folders, and declared widgets/selectors
> below are planned extensions; the old `notes` example is not shipped.
> Route paths and selectors are therefore not a claim that those future
> screens already exist. Each UI feature talks
> **same-origin** to its own API (`ui/server.js` proxies to the API
> process); it never makes a cross-origin call to the authenticator.

This document has two parts: the **planned UI surface** for this scenario
(the three audiences and how they map onto features), and the **stable
ui/manifest slot contract** inherited from the `react-vite` template that
governs where any UI building block lives.

## Planned UI surface — three audiences, one deployment

Per the PRD (UX & Branding) the scenario ships three distinct UI
audiences from one deployment. Each composes the API domains from
[`api-endpoints.md`](api-endpoints.md); the audiences are **UI surfaces**,
not domains (see [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)).

### 1. Admin console (P1) — `admin`

Dense, data-forward management for realm and system administrators, with
destructive-action confirmation gates.

| Route (planned) | Surface | Backing domain(s) | Tier |
|---|---|---|---|
| `/admin/realms` | Realm list + create/edit; delete is confirmation-gated | `realms` | P1 |
| `/admin/realms/:id` | Per-realm policy, branding, token TTLs, redirect URIs | `realms`, `authorization` | P1 |
| `/admin/users` | Realm-scoped user pool; role assignment | `identity`, `authorization` | P1 |
| `/admin/roles` | Role/scope definitions and assignment | `authorization` | P1 |
| `/admin/sessions` | All sessions in scope; revoke (confirmation-gated) | `sessions` | P1 |
| `/admin/audit` | Security-event query, filter, newest-first | `audit` | P1 |

### 2. End-user self-service (P1) — `account`

Minimal, task-focused surfaces for the signed-in user.

| Route (planned) | Surface | Backing domain(s) | Tier |
|---|---|---|---|
| `/account/profile` | View/edit profile; change password | `identity` | P1 |
| `/account/mfa` | TOTP enrollment + recovery codes; passkey register/remove | `mfa` | P1 |
| `/account/sessions` | Review active sessions; revoke / log out everywhere | `sessions` | P1 |
| `/account/connected` | Connected social accounts (link/unlink) | `federation` | P1 |
| `/account/apikeys` | Create/list/revoke personal API keys | `apikeys` | P1 |

### 3. Hosted login / consent (P0 screens → P1 branding) — `auth`

The screens adopting scenarios redirect to or embed **same-origin** for
sign-in, registration, MFA challenge, and OAuth consent. These must be
clean, trustworthy, and fast; per-realm branding (logo, colors) is
rendered at P1.

| Route (planned) | Surface | Backing domain(s) | Tier |
|---|---|---|---|
| `/auth/:realm/sign-in` | Email/password sign-in + social provider buttons | `identity`, `federation` | P0 (branding P1) |
| `/auth/:realm/register` | Account registration | `identity` | P0 |
| `/auth/:realm/mfa` | MFA challenge (TOTP / passkey / recovery code) | `mfa` | P1 |
| `/auth/:realm/consent` | OAuth consent screen | `federation` | P1 |
| `/auth/:realm/reset` | Password-reset request + completion | `identity` | P1 |

Error messages relay validation reasons faithfully ("password is too
short") **without** leaking account-existence information where that
would aid enumeration. Security prompts (MFA challenge, session-revoke
confirmation, "new sign-in from an unrecognized device") communicate
state plainly without alarmism.

## Feature-folder mapping (`ui/src/features/<name>/`)

Each audience surface is implemented as one or more feature folders under
`ui/src/features/`. A feature owns its components, hooks, same-origin API
client wrapper (`ui/src/api/<feature>.ts`, a generated Connect client),
and tests. Planned mapping:

| Feature folder | Audience | API client | Backing domain |
|---|---|---|---|
| `ui/src/features/health/` | (shipped scaffold) | `ui/src/api/health.ts` | `health` |
| `ui/src/features/sign-in/` | hosted login | `ui/src/api/identity.ts` | `identity`, `federation` |
| `ui/src/features/register/` | hosted login | `ui/src/api/identity.ts` | `identity` |
| `ui/src/features/mfa/` | self-service + hosted | `ui/src/api/mfa.ts` | `mfa` |
| `ui/src/features/sessions/` | self-service + admin | `ui/src/api/sessions.ts` | `sessions` |
| `ui/src/features/connected-accounts/` | self-service | `ui/src/api/federation.ts` | `federation` |
| `ui/src/features/apikeys/` | self-service + admin | `ui/src/api/apikeys.ts` | `apikeys` |
| `ui/src/features/profile/` | self-service | `ui/src/api/identity.ts` | `identity` |
| `ui/src/features/realms/` | admin | `ui/src/api/realms.ts` | `realms` |
| `ui/src/features/users/` | admin | `ui/src/api/identity.ts` | `identity` |
| `ui/src/features/roles/` | admin | `ui/src/api/authorization.ts` | `authorization` |
| `ui/src/features/audit/` | admin | `ui/src/api/audit.ts` | `audit` |

The `notes` feature folder ships as the worked example and is removed by
`vrooli scenario detemplate` once the first real feature is green.

## Declared widgets / selectors (planned)

To keep automated testing and accessibility stable, each surface declares
its key interactive elements with stable `data-testid` selectors (a
template accessibility primitive that is preserved). Planned anchors:

| Surface | Widget | `data-testid` (planned) |
|---|---|---|
| sign-in | Email field | `signin-email` |
| sign-in | Password field | `signin-password` |
| sign-in | Submit | `signin-submit` |
| sign-in | Social provider button | `signin-provider-<provider>` |
| register | Submit | `register-submit` |
| mfa | TOTP code input | `mfa-totp-code` |
| mfa | Recovery-code input | `mfa-recovery-code` |
| sessions | Per-session revoke button | `session-revoke-<id>` |
| sessions | Log-out-everywhere button | `sessions-revoke-all` |
| realms (admin) | Delete confirmation dialog | `realm-delete-confirm` |
| apikeys | Created-key reveal (shown once) | `apikey-secret-reveal` |
| consent | Approve / deny | `consent-approve` / `consent-deny` |

Destructive actions (`realm-delete-confirm`, `sessions-revoke-all`,
`session-revoke-*`, `apikey` revoke) route through a confirmation gate.

## Design tokens, i18n, and accessibility

- **Design tokens.** Light + dark themes come from the vrooli-default
  operational-console kit (clean, trustworthy, secure, unobtrusive).
  Per-realm branding (logo + color overrides) on the hosted login/consent
  screens is a **P1** deliverable so hosted products present their own
  identity; branding values come from the realm record
  (`branding` in [`api-endpoints.md`](api-endpoints.md#realms-p0-default-realm--p1-multi-realm)),
  applied via CSS custom properties so a realm theme is a token overlay,
  not a code fork. Theme CSS lives under the `theme-token` slot
  (`ui/src/theme/`).
- **i18n.** All user-facing strings live in `ui/src/i18n/locales/<locale>.json`
  (the `i18n-strings` slot). Security and error copy is faithful but
  enumeration-safe across locales. The PWA install surface (seeded
  `ui/public/site.webmanifest`, `apple-icon-180.png`, `favicon-196.png`,
  maskable icons) is kept valid; generic icons are replaced when product
  branding is confirmed.
- **Accessibility — WCAG AA across all surfaces.** Full keyboard
  navigation for every auth and management control, including MFA
  enrollment, session revocation, and OAuth consent. Template
  accessibility primitives are preserved: `role`, `aria-*` attributes,
  and `data-testid` selectors. Interactive security prompts meet
  focus-management and announcement requirements so screen-reader users
  receive the same timely security information as sighted users.

## Same-origin / no-cross-origin invariant

Every feature's API client talks to its **own** scenario API over the UI
origin; `ui/server.js` proxies `/api/*` and the Connect RPC namespace to
the API process. There are no cross-origin browser calls anywhere — the
hosted login UI talks to its own authenticator API same-origin, and
adopting scenarios that embed these screens do so same-origin to *their*
API, which forwards to the authenticator API-to-API. This mirrors the
backend invariant in [`api-endpoints.md`](api-endpoints.md#architecture-invariants-read-before-adding-an-endpoint).

---

## ui/manifest slot contract (inherited from `react-vite`)

Stable reference for the slots declared in this scenario's
[`ui/manifest.json`](../../ui/manifest.json), inherited from
`templates/scenarios/react-vite/ui/manifest.json`. The auth features
above resolve their filesystem paths through these slots. Update both the
manifest and this section when adding or renaming a slot.

### Contract

| Field | Value |
|---|---|
| `kind` | `scenario-ui` |
| `schema` | `scenario-ui-manifest/v1` |
| `template` | `react-vite` |

### Slots (v1)

| Slot | `dir` | Path pattern | Requires `feature`? |
|---|---|---|---|
| `ui-primitive` | `ui/src/components/ui` | `{dir}/{kebab-name}.tsx` | no |
| `shared-component` | `ui/src/components` | `{dir}/{ComponentName}.tsx` | no |
| `layout-shell` | `ui/src/layout` | `{dir}/{ComponentName}.tsx` | no |
| `layout-nav` | `ui/src/layout` | `{dir}/{ComponentName}.tsx` | no |
| `page` | `ui/src/pages` | `{dir}/{ComponentName}.tsx` | no |
| `feature` | `ui/src/features/{feature}` | `{dir}` (folder) | yes |
| `feature-component` | `ui/src/features/{feature}` | `{dir}/{ComponentName}.tsx` | yes |
| `hook` | `ui/src/hooks` | `{dir}/{camelName}.ts` | no |
| `api-client` | `ui/src/api` | `{dir}/{camelName}.ts` | no |
| `lib-util` | `ui/src/lib` | `{dir}/{camelName}.ts` | no |
| `consts` | `ui/src/consts` | `{dir}/{camelName}.ts` | no |
| `i18n-strings` | `ui/src/i18n/locales` | `{dir}/{locale}.json` | no |
| `theme-token` | `ui/src/theme` | `{dir}/{kebab-name}.css` | no |
| `test-util` | `ui/src/test-utils` | `{dir}/{camelName}.ts` | no |

`defaults.slot` is `shared-component` — components that publish no slot
resolve through this slot.

### Path-Pattern Tokens

| Token | Meaning | Example |
|---|---|---|
| `{dir}` | The slot's `dir` value. | `ui/src/components` |
| `{ComponentName}` | PascalCase. | `SignInForm`, `RealmTable` |
| `{componentName}` / `{camelName}` | camelCase. | `useSession`, `realmPolicy` |
| `{kebab-name}` | kebab-case. | `mfa-code-input`, `error-boundary` |
| `{feature}` | Feature folder; must be supplied when `requiresFeature: true`. | `sign-in`, `sessions`, `realms` |
| `{locale}` | Locale code. Only used by `i18n-strings`. | `en`, `ja`, `ar` |

### Resolution Order (Adoption Resolver)

1. **Explicit override** — caller supplied a path.
2. **Template manifest** — this file resolves the slot and substitutes tokens.
3. **Heuristic** — manifest missing or slot missing; scan `ui/src/` for a
   matching directory name. Warning attached.
4. **Fallback** — `ui/src/components/<ComponentName>.tsx`. Warning attached.

### Overlays

Scenarios may override individual slot `dir` values inside
`.vrooli/ui-manifest.json` in the scenario root. The overlay must not
introduce new slot names — those live on the template manifest. (Overlay
loader tracked in `scenarios/react-component-library/PRD.md`.)

## Cross-References

- [`api-endpoints.md`](api-endpoints.md) — the API domains these surfaces compose
- [`cli-commands.md`](cli-commands.md) — the parallel CLI surface
- [`configuration.md`](configuration.md) — per-realm branding + policy knobs
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — domains vs UI surfaces
- Concept: [`../concepts/UI-ARCHITECTURE.md`](../concepts/UI-ARCHITECTURE.md)
- Schema: `.vrooli/schemas/scenario-ui-manifest.schema.json`
- Resolver: `scenarios/react-component-library/api/internal/adoptions/pathresolver.go`
