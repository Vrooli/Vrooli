# Source Record: Professional Integrations and Monetization UX

Date: 2026-09-03
Workspace: `/home/matthalloran8/Vrooli`

This file preserves the material discussion and investigation context that led
to the Plan Manager plan `professional-integrations-monetization-ux`. It is a
source artifact for the plan, not an independent implementation specification.

## Conversation decisions

### Account UX precedent

- Web Console account settings was implemented first as a shared-component
  precedent for future monetized scenarios.
- The user wanted a professional, mature, intuitive SaaS surface.
- The account UX uses shared React Component Library assets where possible.
- The Web Console profile entry belongs in the existing toolbar.
- A persistent subscription banner does not belong in the top bar.
- Account-related information belongs in account settings unless an urgent
  account issue requires the existing global banner.
- A profile icon or profile-add icon may show an optional notification marker
  for sign-in, low credits, or an account issue.

### Button and control-library decision

- React Component Library `Button` should default to the pill shape.
- Web Console must not compensate for a wrong global library default with an
  application adapter.
- Square controls are explicit exceptions for mobile toolbars and welded or
  repeated compact controls.
- `InputGroup` is the preferred pattern for a credential field with an attached
  action, as used by the New Session custom command launcher.

### Integrations UX discussion

The current Web Console and Git Control Tower pages were reviewed. Both pages
primarily display capability and dependency health. They do not yet provide a
complete user-managed integration experience.

The agreed mental model is:

1. Connected accounts: GitHub, OpenRouter, Slack, BYOK providers, and other
   authenticated provider connections.
2. Runtime dependencies: local resources and Vrooli scenarios such as Ollama,
   Audio Tools, and Browser Automation.
3. Commercial context: plan, credits, entitlements, relevant billing issues,
   and carefully contextualized offers.

The primary user reasons for visiting Integrations are to connect a provider,
manage a key or OAuth session, test a connection, fix an expired authorization,
review permissions, see scenario usage, add a second account, disconnect, or
understand why a dependency is unavailable.

The current capability registry remains useful for runtime dependency health,
but it should not be the main visual model for provider connections. Technical
reason codes, raw operator commands, tiny diagnostic copy, and planned future
cards should move behind details or into a runtime-dependencies surface.

### Monetization discussion

Landing Page Business Suite already owns subscription status, credits, plans,
entitlements, bundles, and pricing. The user proposed a periodic status request
that can return time-insensitive notices, scenario recommendations, install
links, offers, and placement/visibility conditions.

The design direction is accepted with safeguards:

- Entitlements, billing status, and credit balances are authoritative account
  data.
- Offers, recommendations, and notices may be cached and eventually
  consistent.
- Eligible offers should be filtered by the commercial owner, not recomputed
  independently by each scenario.
- A stale offer must never grant access.
- Integrations pages may show a relevant recommendation, such as installing a
  missing dependency or adding AI credits, but should not become generic ad
  surfaces.
- A shared account/commercial-context client should provide caching,
  deduplication, identity propagation, offline behavior, and consistent errors.

## Investigated current implementation

### Web Console

- `scenarios/web-console/ui/src/components/settings/IntegrationsSection.tsx`
  composes `SettingsList` with `IntegrationsPanel`.
- `scenarios/web-console/ui/src/components/IntegrationsPanel.tsx` polls
  `useCapabilities`, groups scenario and resource capabilities, displays status,
  features, messages, reason codes, operator commands, and limited scenario
  start/restart actions.
- `scenarios/web-console/ui/src/hooks/useCapabilities.ts` uses React Query with
  30-second polling and pauses when the settings surface is closed.
- `scenarios/web-console/ui/src/api/capabilities.ts` decodes capability status,
  feature status, provider status, reason metadata, and lifecycle actions.
- `scenarios/web-console/api/internal/capabilities/registry.go` is the source of
  truth for Web Console's scenario/resource dependency registry.
- `scenarios/web-console/docs/concepts/ARCHITECTURE.md` describes the capability
  registry and the current two-group Integrations panel.
- `scenarios/web-console/docs/business/MONETIZATION.md` and
  `scenarios/web-console/docs/internal/BUNDLE_INTEGRATION.md` define the current
  LPBS boundary, BYOK behavior, ai-gateway enforcement, entitlement lease, and
  credential-authority rules.

### Git Control Tower

- `scenarios/git-control-tower/ui/src/components/SettingsTabIntegrations.tsx`
  renders a read-only capability list with status icons, dependency kind,
  descriptions, messages, and feature tags.
- `scenarios/git-control-tower/ui/src/lib/api-settings.ts` defines the current
  capability and credential models.
- `scenarios/git-control-tower/ui/src/lib/hooks-settings.ts` polls capabilities
  and separately manages HTTPS credentials, SSH keys, credential testing, and
  remote URLs.
- Git Control Tower therefore has credential functionality, but the user's
  provider connection model is fragmented across capability, credential, SSH,
  and repository settings concepts.

### React Component Library assets

Existing assets that can compose a mature surface include:

- `PageHeader`, `SettingsList`, `Card`, `CardGrid`, `CollectionPage`,
  `ResourceCollection`, `ResourceDetail`, `DescriptionList`, `StatusBadge`,
  `HealthIndicator`, `Alert`, `Banner`, `EmptyState`, `AsyncBoundary`,
  `Button`, `ButtonGroup`, `IconButton`, `NotificationBadge`, `Avatar`,
  `InputGroup`, `SearchInput`, `FilterBar`, `DataToolbar`, `AuditTrail`,
  `Dialog`, `DrawerShell`, `ResponsivePanel`, and `ResponsiveDialog`.
- `ResourceCollection` already provides token-bound headers, search, filters,
  status, retry, refresh, row actions, selection, pagination, and responsive
  behavior.
- `ResourceDetail` already provides metadata, history, freshness, actions, and
  loading/error/empty/permission/offline states.
- `StatusBadge`, `HealthIndicator`, and `NotificationBadge` already encode the
  status and attention vocabulary needed for provider connections.

### Future integration model

- `docs/configuration/integrations/connectors.md` defines a connector as the
  reusable provider/auth recipe and a connection as one authenticated instance.
- `docs/configuration/integrations/connections.md` defines connection creation,
  listing, probing, refreshing, binding, unbinding, revocation, scratch state,
  and scenario bindings.
- `integration-hub` is explicitly deferred in `docs/configuration/architecture.md`.
  It will own connector definitions, connection instances, auth flows,
  credential-authority identities, binding, refresh, revoke, and unbound state.
- The connector design supports API keys, OAuth web, OAuth device, external
  sign-in commands, and app passwords.

### Landing Page Business Suite model

- `packages/proto/schemas/landing-page-business-suite/v1/account.proto` exposes
  subscription, credit, and entitlement views.
- `packages/proto/schemas/landing-page-business-suite/v1/config.proto` exposes
  public landing configuration, intro offers, pricing, and downloads.
- `scenarios/landing-page-business-suite/PRD.md` calls for subscription-aware
  pricing APIs, credits and entitlements, and entitlement-gated app downloads.
- The future commercial context contract must preserve the distinction between
  authoritative entitlements and cacheable merchandising content.

## Design principles captured for execution

1. Build one provider-neutral shared pattern before generalizing it across many
   scenarios.
2. Separate user-managed connections from runtime dependency health.
3. Keep commercial policy in Landing Page Business Suite and enforcement in the
   trusted service that owns the billable operation.
4. Use Connect/proto-first contracts for new cross-scenario APIs.
5. Keep secrets write-only and return metadata-only views.
6. Do not turn a settings page into an indiscriminate advertising surface.
7. Use RCL governed draft/adoption workflows for shared assets.
8. Treat visual validation, responsive behavior, accessibility, i18n, and
   scenario-level health as acceptance requirements.

