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

## Contract matrix

| Domain | Authoritative owner | Consumer surface | Preserved behavior | Current limitation |
|---|---|---|---|---|
| Runtime capability health | Capability registry in each owning scenario/control-plane adapter | Web Console and Git Control Tower runtime-dependency sections | Existing availability states, feature declarations, reason codes, and declared scenario start/restart actions remain intact | Runtime health does not identify an authenticated provider account |
| Credential connection metadata | Integration Hub contract in `common.v1.integrations`; current OpenRouter projection is served by the Hub | Connected-account cards | Secret values remain authority-only; signed-out local runtime behavior remains available | Broader provider-driver lifecycle and scenario binding are deferred; the pilot supports metadata-only lifecycle actions |
| Repository credentials | Git Control Tower credential API and repository scope | Git Control Tower connected-account cards and dedicated credential tabs | Repository/remote boundaries, HTTPS and SSH behavior, and existing credential hooks remain intact | Full cross-scenario binding metadata awaits Integration Hub |
| Commercial account facts | Landing Page Business Suite AccountService | Optional consumer context | Subscription, credit, and entitlement facts remain commercial-owner data and never authorize operations | Context is presentation-only and requires authenticated identity |
| Commercial notices and offers | Landing Page Business Suite server-side eligibility | Optional Integrations recommendation section | Placement, capability relevance, stable IDs, expiration, dismissal, freshness, and owned CTAs are preserved | Generic advertising and client-side entitlement decisions are prohibited |
| Shared presentation | React Component Library `IntegrationCard@0.1.4`, `StatusBadge`, `Button`, and `InputGroup` | Web Console and Git Control Tower | Provider-neutral states and token-bound controls are shared while application shells remain independent | Provider auth protocol UI and lifecycle storage do not belong in RCL |

This matrix is the change-boundary record: a new provider or scenario must add
an owner and consumer row before introducing a new connection field, action,
promotion rule, or secret-handling path.
