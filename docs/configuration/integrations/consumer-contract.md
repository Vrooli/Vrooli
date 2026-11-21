# Integration consumer contract

Consumer surfaces keep three domains separate:

1. Connected accounts are authenticated connection metadata. They may show a
   provider identity, scopes, bindings, verification time, health, and the
   lifecycle actions supported by the owner. They never contain keys, tokens,
   or credential-authority values.
2. Runtime dependencies are local resources or scenarios that provide a
   capability. Their health and lifecycle actions come from the capability
   contract and are not account health.
3. Commercial context is optional presentation guidance from Landing Page
   Business Suite. The server evaluates eligibility and returns stable content
   IDs, placement, freshness, expiration, dismissal, and owned CTA
   destinations. A cached item cannot grant an entitlement or authorize an
   operation.

The Integrations placement is allow-listed and capability-scoped. Generic
advertising is not a valid consumer response. A commercial-context failure
must leave account and runtime sections usable; signed-out consumers render
local runtime health and omit private account/commercial data.

The provider-neutral protobuf view contract lives in
`common.v1.integrations`. The commercial-context contract lives in
`landing_page_business_suite.v1`. Integration Hub owns connection lifecycle and
credential-authority interaction; consumer scenarios only render safe metadata
and invoke typed owner operations.

## Integration records and dependency views

An integration record names an owned capability contract and the scenarios
that consume it. A consumer's `uses` relationship describes why it calls an
integration; a runtime dependency declares what must be present for the
scenario to run. These are related views, not interchangeable facts: the
scenario or resource owner remains authoritative for its capability and
lifecycle, and the deployment manifest remains authoritative for required
dependencies. A UI may group by capability or dependency, but it must link both
views to their owner and must not infer that an installed dependency is an
authenticated account or an eligible commercial route.

Dependency entries declare `uses: [{id, name, description, group}]`. The ID names
the consumer capability, while `dependencyKind` and `dependencySlug` name its
provider. Both views project the same registry snapshot; labels and grouping
must not be redeclared in each app. Local installation observations carry
`location` (`machine` or `bundled`), `version`, and `selected`; absence means
unknown. Health-message text and remote model versions are never installation
evidence.

The integration record schema and generated inventory are owned by their
contract source. This page defines how consumer surfaces present the records;
it does not introduce a second registry or duplicate dependency version facts.

## Contract matrix

| Domain | Authoritative owner | Consumer surface | Preserved behavior | Current limitation |
|---|---|---|---|---|
| Runtime capability health | Capability registry in each owning scenario/control-plane adapter | Web Console and Git Control Tower runtime-dependency sections | Existing availability states, feature declarations, reason codes, declared start/restart actions and owner-supplied troubleshooting commands remain intact | Runtime health does not identify an authenticated provider account |
| Credential connection metadata | Integration Hub contract in `common.v1.integrations`; current OpenRouter projection is served by the Hub | Connected-account cards | Secret values remain authority-only; signed-out local runtime behavior remains available | Broader provider-driver lifecycle and scenario binding are deferred; the pilot supports metadata-only lifecycle actions |
| Repository credentials | Git Control Tower credential API and repository scope | Git Control Tower connected-account cards and dedicated credential tabs | Repository/remote boundaries, HTTPS and SSH behavior, and existing credential hooks remain intact | Full cross-scenario binding metadata awaits Integration Hub |
| Commercial account facts | Landing Page Business Suite AccountService | Optional consumer context | Subscription, credit, and entitlement facts remain commercial-owner data and never authorize operations | Context is presentation-only and requires authenticated identity |
| Commercial notices and offers | Landing Page Business Suite server-side eligibility | Optional Integrations recommendation section | Placement, capability relevance, stable IDs, expiration, dismissal, freshness, and owned CTAs are preserved | Generic advertising and client-side entitlement decisions are prohibited |
| Shared presentation | React Component Library `IntegrationsOverview@1.1.1`, `SettingsList`, and `Button` | Web Console and Git Control Tower | Capability/dependency projections, observed installation facts, provider-neutral states and token-bound controls are shared while application shells remain independent | Provider auth protocol UI and lifecycle storage do not belong in RCL |

This matrix is the change-boundary record: a new provider or scenario must add
an owner and consumer row before introducing a new connection field, action,
promotion rule, or secret-handling path.
