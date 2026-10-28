# Scenario Notification Settings — State, Scoping Fix, Extraction Plan

Date: 2026-10-28. Scope: the per-scenario "Notifications" settings surface that
Web Console started, the bug that made it show every scenario's categories, and
the path to a reusable React Component Library (RCL) asset that any scenario can
drop into its own settings page.

Line numbers are as of this change and will drift.

## 1. Current state

### Data plane (Notification Hub)

| Piece | Where | Notes |
|---|---|---|
| REST catalog read | `notification-hub/api/handlers/preferences/module.go:166` (`getCatalog`) | `GET /api/v1/notifications/catalog` → `{sources[], preferences[]}` |
| REST preference read | `module.go:185` (`getPreferences`) | `GET /api/v1/notifications/preferences` |
| REST preference write | `module.go:195` (`putPreference`) | `PUT /api/v1/notifications/preferences`; optional `scope_key` = per-instance override |
| Source filter (new) | `module.go:125` (`sourceFilter`, `filterSources`, `filterPreferences`) | `?source_scenario=a[,b]`, repeatable |
| Catalog storage | `api/internal/preferences/preferences.go:317` (`ListCatalog`) | Observed + declared sources; hides undeclared sources that aren't installed scenarios |
| Read identity | `module.go:62` (`configuredRecipient`) via `main.go:257` `service.ResolveRecipient` | Reads use the installation's configured recipient, no auth required |
| Write identity | `module.go:58` → `internal/identity/identity.go:63` (`Subject`) | Writes require a verified owner (CF Access > Bearer > cookie) |
| Contract doc | `docs/concepts/NOTIFICATIONS.md` §Preference Model | Describes Connect endpoints; the shipped surface is REST (doc now notes this) |

There is no proto for preferences: the module header (`module.go:1-11`)
records a deliberate REST choice. `packages/proto/schemas/notification-hub/v1/`
has no catalog/preference messages, so **no proto change was needed**.

### Web Console consumer

| Piece | Where | Scenario-specific? |
|---|---|---|
| Same-origin REST facade | `web-console/api/notifications_proxy.go:40` mounted at `api/routes.go:152` | Generic. Forwards path, query, Bearer, device id, CF Access assertion; 503 on hub down |
| Fetch client | `web-console/ui/src/api/notifications.ts:71` (`authedFetch`), `:105` (`fetchNotificationCatalog`), `:137` (`updateNotificationPreference`) | Auth is Web Console's (`lib/auth#getAccessToken`, `lib/deviceIdentity`) |
| Source slug | `notifications.ts:93` `WEB_CONSOLE_SOURCE` | Yes |
| Per-session mute | `notifications.ts:147-183`, `hooks/useSessionNotificationMute.ts` | Web Console feature, but the `scope_key` pattern is generic |
| Settings section | `ui/src/components/settings/NotificationsSection.tsx:161` | See §3 |
| Mounted in | `ui/src/components/SettingsModal.tsx:80` | — |

### Hub's own settings (second, divergent implementation)

`notification-hub/ui/src/api/preferences.ts` and `ui/src/pages/SettingsPage.tsx`
reimplement the same types, urgency ladder, and preference indexing
(`SettingsPage.tsx:102`) with cookie auth (`credentials: "same-origin"`). It
correctly shows **all** sources — it is the cross-scenario view.

## 2. Bug: Web Console showed every scenario's categories

**Root cause (two layers).**

1. The hub catalog API had no way to ask for one source: `getCatalog` always
   returned `ListCatalog()` in full. The facade forwards the query string
   (`notifications_proxy.go:56`), so the gap was purely at the hub.
2. Web Console's client called the catalog with no scoping and
   `NotificationsSection` rendered `sources.map(...)` over whatever came back.
   The surface was built as a copy of the hub's cross-scenario view rather than
   a per-scenario view.

**Fix.**

- Hub: `getCatalog` and `getPreferences` accept an optional `source_scenario`
  filter (repeatable or comma-separated). Absent → unchanged (the hub's own
  Settings page). Present → only those sources and their preference rows; an
  unknown/undeclared source returns an empty catalog, never other scenarios.
  Filtering is in the handler (storage untouched — the catalog is small and
  `ListCatalog`'s installed-scenario filtering stays the single source of truth).
- Web Console: `fetchNotificationCatalog(sourceScenario = WEB_CONSOLE_SOURCE)`
  sends `?source_scenario=web-console` **and** filters the response
  client-side, so an un-restarted hub that ignores the parameter still cannot
  leak other scenarios. Query key is now `["notification-catalog", source]`
  (shared by the settings section and the per-session mute hook).
- `NOTIFICATIONS.md` documents the filter and that a scenario page MUST pass it.

**Tests added.**

- `notification-hub/api/handlers/preferences/module_test.go`:
  `TestCatalog_SourceFilterScopesToOneScenario`,
  `TestCatalog_SourceFilterAcceptsListsAndUnknownSources`,
  `TestPreferences_SourceFilter`.
- `web-console/api/notifications_proxy_test.go`:
  `TestNotificationsProxyForwardsCatalogSourceFilter`.
- `web-console/ui/src/api/notifications.test.ts`: scoped request + client-side
  drop of foreign sources. `NotificationsSection.test.tsx`: section requests
  `web-console` only.

**Restart needed.** notification-hub (API) for the server filter; web-console
(UI rebuild + API) for the client change. The client-side filter makes the
visible bug go away with only a web-console restart.

**Not changed (follow-up copy):** the section's description/empty-state strings
(`i18n/locales/*.json` → `settings.notificationsSection`) still say "which of
your apps can notify you". With per-scenario scoping it should read like
"Choose which Web Console notifications reach you…". Left alone because the
copy lives in three locales and a concurrent agent is editing notification copy.

## 3. Reusability assessment

What is already generic:

- The **wire shape** (`NotificationCatalog`, `NotificationSource`,
  `NotificationCategory`, `NotificationPreference`) — identical in both UIs.
- `categoriesFor` (synthesises the source-wide "All notifications" row for a
  source with no declared categories) and `effectivePreference` (stored row →
  declared defaults fallback, ignoring `scope_key` rows) —
  `NotificationsSection.tsx:30-67`. Pure functions; no Web Console coupling.
- Loading skeleton / empty / unavailable state composition, and the
  `SettingsList` / `Avatar` RCL primitives it already uses.
- The facade (`notifications_proxy.go`) has nothing Web Console-specific except
  its file location.

What is scenario-specific / blocks drop-in reuse:

| Coupling | Where | Resolution |
|---|---|---|
| Auth headers via Web Console's `getAccessToken` + `deviceIdentity` | `notifications.ts:71-90` | Inject a `fetch`-like transport into the client |
| Hub link via Web Console's `/api/v1/external-url` | `notifications.ts:128`, `NotificationsSection.tsx:178` | Make `hubHref` a prop; each scenario resolves it however it already resolves sibling links |
| i18n keys `strings.settings.notificationsSection.*` (react-i18next + generated strings) | `NotificationsSection.tsx` throughout | Asset takes a `labels` prop with English defaults |
| `wc-*` Tailwind tokens (13 uses) and Web Console's `SettingsToggle` primitive | `NotificationsSection.tsx`, `settings/primitives.tsx:12` | Use RCL `Switch`/`Select` + RCL tokens |
| Raw `<select>` for urgency | `NotificationsSection.tsx:289` | RCL `Select` |
| Multi-source grouping (`sources.map`) | `NotificationsSection.tsx:269` | Keep grouping (harmless with one source), but hide the group header when exactly one source — a scenario's own page needs no "Web Console" avatar header |
| Mutation disables **all** controls while any write is pending | `NotificationsSection.tsx:293,309` | Optimistic per-row update (RCL `useOptimisticAction`) |
| React Query dependency | section + hook | Fine for the data hook; must not leak into the RCL component |

Missing for "a few lines" adoption:

1. **A transport story for scenarios other than Web Console.** Reads need no
   auth; writes need a verified owner. Options: (a) each scenario mounts a
   copy of the facade (Go) and the browser sends its own Bearer — today's Web
   Console model; (b) the browser calls the hub directly cross-origin — needs
   CORS + a token the hub accepts, and breaks on tunnel subdomains without
   cookie sharing. (a) is the proven path; the facade should move to a shared
   Go package (e.g. `packages/api-core` `notificationsfacade.Handler(resolve)`)
   so a scenario mounts it in one line.
2. **A shared typed client + hook** (does not exist; both UIs hand-roll it).
3. **A presentational component** with no network behaviour (does not exist).

## 4. Extraction plan

RCL policy (see `react-component-library/docs/concepts/INTEGRATIONS.md`
"Provider-neutral integration presentation"): an RCL asset owns **no network
behaviour, credential authority, or mutation**; consumers pass data and
callbacks. So the extraction is split in two.

### 4a. RCL asset — `NotificationPreferencesPanel` (presentational)

```ts
interface NotificationPreferencesPanelProps {
  status: "loading" | "ready" | "unavailable" | "error";
  sources: NotificationSource[];          // usually exactly one
  preferences: NotificationPreference[];  // global rows; scope_key rows ignored
  onChange: (next: NotificationPreference) => void | Promise<void>;
  pendingKeys?: ReadonlySet<string>;      // `${source}\0${category}` rows in flight
  onRetry?: () => void;
  hubHref?: string;                        // "Open Notification Hub" / setup link
  showSourceHeader?: boolean;              // default: sources.length > 1
  urgencyLevels?: readonly string[];       // default low|normal|high|critical
  labels?: Partial<NotificationPreferencesLabels>; // English defaults
}
```

Ship the pure helpers (`categoriesFor`, `effectivePreference`, `prefKey`) and
the wire types as companions of the asset so the hook can import them.
Built on `SettingsList`, `Switch`, `Select`, `Avatar`, `useOptimisticAction`.
Stories: loading, unavailable, empty (with/without `hubHref`), one declared
source, one source with no categories ("All notifications" row), multi-source,
row pending, disabled urgency when off.

### 4b. Data hook + client — `useScenarioNotificationPreferences`

```ts
function useScenarioNotificationPreferences(opts: {
  sourceScenario: string;
  transport: (path: string, init?: RequestInit) => Promise<Response>; // adds auth
  enabled?: boolean;
}): {
  status; sources; preferences; pendingKeys;
  setPreference(next: NotificationPreference): Promise<void>;
  refetch(): void;
  // per-instance overrides (Web Console session mute generalised):
  setScopedEnabled(scopeKey: string, enabled: boolean): Promise<void>;
  scopedEnabled(scopeKey: string): boolean | undefined;
}
```

It always sends `?source_scenario=` and re-filters the response (same guard as
Web Console). It cannot live in RCL (network). Home: a small shared TS package
alongside `packages/api-base` (e.g. `packages/notification-client`), with
`@tanstack/react-query` as a peer. Adopting it is then:

```tsx
const prefs = useScenarioNotificationPreferences({ sourceScenario: "my-app", transport: authedFetch });
<NotificationPreferencesPanel {...prefs} onChange={prefs.setPreference} hubHref={hubUrl} />
```

plus one Go line mounting the shared facade, plus the scenario's existing
`.vrooli/notifications.json` declaration.

### 4c. Migration steps

1. **Web Console (proving ground).** Refactor `NotificationsSection` into the
   presentational panel + hook *inside* web-console, with the props above,
   RCL `Switch`/`Select`, no `wc-*` tokens, labels via prop. Keep its tests
   green; add the single-source header hide and per-row pending.
2. **Shared Go facade.** Move `notificationsProxy` into a shared package;
   Web Console mounts it from there (behaviour-identical, same tests).
3. **RCL harvest.** It is a *new* asset, so use
   `react-component-library components ingest web-console
   ui/src/components/settings/NotificationPreferencesPanel.tsx
   NotificationPreferencesPanel --companion-files <helpers/types>`
   (`draft-begin` only begins a draft from an existing release). Add stories,
   `components check`, then `draft-publish` (operator-driven; not done here).
4. **Shared client package.** Publish the hook/client via the governed package
   path (Scenario Dependency Analyzer for the new workspace dep).
5. **Web Console adopts the released asset** (`@vrooli/react-component-library/NotificationPreferencesPanel/1`).
6. **Notification Hub adopts it** for its cross-scenario page (`sources` =
   all, `showSourceHeader` true, cookie transport). This retires the divergent
   `SettingsPage` preference rendering and `ui/src/api/preferences.ts`.
7. **Other producers** (git-control-tower, agent-manager, …) add a
   "Notifications" settings section with the three lines above.

## 5. Open questions

1. **Read/write identity split.** Reads use the configured recipient
   (`module.go:62`, `service.ResolveRecipient`), writes use the verified
   subject (`module.go:196`). If those keys ever differ (multi-user, or a
   CF Access subject ≠ configured recipient), a toggle writes a row the next
   read never returns, so the switch "snaps back". Should reads also prefer
   the verified subject when present?
2. **Should a scenario be allowed to write only its own source?** Today any
   authenticated caller can write any `source_scenario` through any facade.
   A scenario facade could pin `source_scenario` server-side — defence in depth
   that matches the scoping intent.
3. **Facade vs direct-to-hub.** Is a per-scenario Go facade acceptable as the
   standard, or should the hub expose a CORS-enabled, token-authenticated
   surface so pure-UI scenarios need no API change?
4. **REST vs Connect.** `NOTIFICATIONS.md` still describes Connect endpoints.
   Promote the preferences surface to proto (one `ListNotificationCatalog`
   with `repeated string source_scenarios`) before other scenarios depend on
   it, or ratify REST in the contract?
5. **Observed-but-undeclared categories.** A scenario's page scoped to itself
   will show observed categories with raw keys as display names. Should the
   scoped view hide undeclared categories, or nudge the producer to declare?
6. **Channel override.** The model has `channel_override` but neither UI
   exposes it. In or out of the shared panel's v1?
7. **Docs manifest.** `docs/research/` is not a section in
   `docs/manifest.json`; register this file or move it under `internal/`.
