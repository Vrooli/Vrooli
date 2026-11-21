# Gateway Capability Manifest Maturation Plan

Status: complete. Last updated 2026-11-20.

This document is the durable program for the AI Gateway's capability manifest:
the closed vocabularies and owner surfaces that describe profiles, roles,
providers, and measures. It stands alone so a future agent can continue without
reconstructing the history from commits.

## Stance

Profiles, roles, and measures are **closed policy vocabularies**, not a growing
runtime manifest. The gateway does not discover capability at request time and
does not let an operator register a new profile or measure by editing a data
file alone. Adding a member is a deliberate code-and-policy change with a gate.

The rules that follow:

1. **One owner per fact.** A profile is declared once as data and projected into
   the UI and docs. A measure is named once in the handler and projected into the
   CLI and proto. A provider is declared once in `DefaultProviderSpecs` and
   projected into the deployed catalog.
2. **Generated artifacts are commits.** The profile constant and the capability
   reference are checked in and verified by `-check`; they are not built at
   runtime from whichever files happen to be present.
3. **Drift is a test failure.** A proto enum value, catalog row, CLI enum, or
   generated table that disagrees with the others fails a focused test rather
   than producing a stale operator surface.

A catalog row is inventory, never route availability. Candidate eligibility and
profile ordering remain the routing policy's job.

## Baseline Surfaces

| Surface | Owner | Location | Projection / consumer | Current gate |
|---|---|---|---|---|
| Profile wire enum | Shared proto | `packages/proto/schemas/ai-gateway/v1/shared/gateway.proto` | Go `sharedv1.Profile`, TypeScript `Profile` | Generated proto verification |
| Profile operator catalog | Gateway data | `config/profile-catalog.json` | UI catalog, capability reference | `gatewaycatalog` loader validation |
| Profile UI constant | Generated | `ui/src/consts/profileCatalog.generated.ts` | `RoutePreviewPage`, `api/gateway.ts` | `TestGeneratedProfilesTSMatchesCommitted` |
| Runtime accepted profiles | Gateway code | `api/internal/gateway/service.go` `AcceptedProfiles()` | Gateway validation, `GatewayService` capabilities | `TestAcceptedProfilesMatchProtoEnum` |
| Profile CLI enum | CLI manifest | `cli/manifest.json`, routing preview `profile` flag `values` | CLI parser and help | `TestCLIRoutingPreviewProfilesMatch` |
| Role catalog | Gateway data | `config/inference-role-catalog.json` | Routing candidate walk | Reference coverage test |
| Provider registry | Gateway data | `api/internal/providers/specs.seed.json` (embedded) | Execution, deployed catalog | `go test ./internal/providers/...` |
| Deployed provider catalog | Generated | `api/internal/catalog/providers.generated.json` | Packaged inference | `make verify-provider-catalog` |
| Capability reference | Generated | `docs/reference/capability-catalog.md` | Operators and agents | `TestGeneratedReferenceMatchesCommitted` |
| Shared capability publication | Gateway catalog projection | `api/internal/capabilitycatalog/`, `api/handlers/catalog/` | Cross-scenario Connect discovery over `common.v1.AIGatewayCapabilityCatalog` | `make endpoints`, catalog projection tests, proto generation verification |
| Measure names | Gateway code | `api/handlers/measures/measures.go` constants | CLI measures, proto RPCs | `TestRouteEventMeasuresMatchCLIAndProto` |
| Measure wire contract | Proto | `packages/proto/schemas/ai-gateway/v1/measures/measures.proto` | Connect RPC, CLI bindings | Manifest ↔ proto test |
| Measure operator commands | CLI manifest | `cli/manifest.json` measures group | `ai-gateway measures ...` | Manifest ↔ proto test |

Regeneration commands:

```bash
cd scenarios/ai-gateway
make gateway-catalog           # profile TS + capability reference
make verify-gateway-catalog    # fail if either is stale
make provider-catalog          # deployed provider projection
make verify-provider-catalog   # fail if the provider projection is stale
make check                     # lint, tests, catalog drift, coverage, temporal model
```

## Debt Register

| ID | Gap | Symptom | Real Fix | Owner | Status |
|---|---|---|---|---|---|
| D1 | Role→resource binding is a string pair (`provider`, `resource_role`) with no compile-time or generated check that the resource actually exposes that role. | A role candidate can name `extract.structured` while the local resource only exposes `chat.default`; routing discovers this only when the resource policy command runs. | Gate every candidate against the resource's canonical `model-policy.json`; a committed projection is an optional optimization, not a second list. | Gateway + resource owners | **Closed 2026-11-20:** `TestRoleCandidatesNameKnownResourceRoles` and `gen-provider-catalog` validation read resource policies directly. This caught and fixed `browser.extract → ollama` naming `extract.structured`; it now uses `chat.default`. |
| D2 | Provider membership has more than one declaration path: the provider registry, wrapper-less supplier JSON declarations, and the generated deployed catalog. | Adding a provider can update one path and silently miss another; the packaged catalog can disagree with execution. | Make one declaration the owner, derive the rest, and keep `verify-provider-catalog` as the gate. | Gateway | **Closed 2026-11-20:** `specs.seed.json` owns registered providers; generator tests and validation reject supplier IDs that collide with any registered provider (including non-projected adapters), an existing resource declaration, another supplier file, or an already projected entry. Supplier rows require an explicit supported family, locality, endpoint, and valid policy JSON. |
| D3 | The profile vocabulary used to be copied by hand into the proto, `AcceptedProfiles`, the UI, and CLI help. Request kinds and role kinds also lacked a canonical lookup for their resource capability semantics. | A new enum or role kind could appear without a gateway capability mapping; routing could silently reject a kind or drift from resource policy vocabulary. | Profiles use the JSON catalog and generated projections. `api/internal/requestvocab` maps each selectable proto `RequestKind` to required input modalities, output kind, and provider capability aliases, owns role-kind semantics, and validates coverage. Routing and role-catalog validation consume these lookups. | Gateway | **Closed 2026-11-20:** proto enum coverage, fail-closed lookup, role-kind validation, modality/capability mappings, generated reference, and focused tests |
| D4 | Operator surfaces are still hand-authored in places: CLI flag help repeats role guidance, and prose docs describe role intent separately from the catalog. | Documentation and `--help` can lag the role catalog even when routing is correct. | Project closed enums and role summaries from the catalog into generated reference material; keep the CLI manifest as the command-binding owner but stop copying enum text by hand. | Gateway | **Closed 2026-11-20:** CLI profile/privacy enums are checked; UI labels are localized; generated references cover request modalities/capabilities, roles, providers, candidates, and privacy classes; the shared catalog is discoverable over Connect. |
| D5 | Measure identity is declared in Go handler declarations, the CLI manifest, and the proto service. | A renamed measure can break the measures-health probe or leave a CLI command bound to a missing RPC. | Compare all handler declarations, CLI measure commands, and proto RPCs; require every persisted stateful domain to be declared and measured or explicitly omitted. | Gateway | **Closed 2026-11-20:** reusable coverage test discovers persisted tables with `created_at` and requires a declared stateful domain, then checks handler/CLI/proto closure and omission reasons. `media_executions` plus inference recovery roots, remote consent grants, private provider plans, and private provider receipts are explicitly omitted with owner and privacy rationale. The measures-health probe reports Domain Coverage L3 with zero warnings. |
| D6 | Package boundaries are asymmetric: `gatewaycatalog` reads provider specs and the role catalog, while resource policy is read only through CLI seams. | A generator can accidentally become a second policy evaluator, or a test can depend on live resource state. | Keep catalog builders pure projections over committed inputs and compiled specs. Resource-policy validation belongs in a separate, explicitly gated adapter. | Gateway | **Closed 2026-11-20:** catalog builders project committed native declarations only; resource-policy validation stays in the provider-catalog generator's explicit gate. The Connect catalog has no runtime resource dependency. |
| D7 | Generated artifacts can go stale if a contributor edits data and forgets to regenerate. | The reference page or UI labels describe an older profile set. | `make verify-gateway-catalog` in the scenario check path plus the committed-output tests. | Gateway | Landed |
| D8 | Locale parity for new profile copy is manual. | Adding a profile row requires locale edits before the UI can render real copy. | Keep `pnpm strings:check` and the locale parity test as the gate; treat profile copy as a normal locale change. | Gateway UI | Landed as a gate |
| D9 | Long files and high-complexity functions obscured behavior boundaries in provider generation, conformance, inference, routing, request vocabulary, role catalogs, and CLI replay. | The recorded scan counted 98 functions above threshold 10 against a baseline of 28; five files exceeded the 500-line threshold. | Split files by responsibility and extract semantic helpers while preserving behavior; keep the recorded allowance unchanged. | Gateway | **Closed 2026-11-20:** Test Genie `tidiness` passed run `20261120-155022-e15d01f6`; local debt and complexity/coupling are L4, no long-file findings remain, and the production scan has no functions above 10. The recorded baseline was not raised. |

The pre-existing proto drift around `BrowserSessionBinding`,
`billing_operation_id`, and `NO_ELIGIBLE_ROUTE` was **resolved 2026-11-20** by the
main agent: the messages/fields/enum value were added to
`packages/proto/schemas/ai-gateway/v1/inference/inference.proto`, the ai-gateway
proto artifact was regenerated, and the full API suite passes.

## Phases

### Phase 0 — Inventory truth

Establish that the profile vocabulary and role/provider inventory are observable
and reproducible.

Gate:

```bash
cd scenarios/ai-gateway/api
GOWORK=off go test ./internal/gatewaycatalog/... -count=1
GOWORK=off go run ./cmd/gen-gateway-catalog -check
```

Exit criteria: the catalog covers every selectable proto profile exactly once;
the generated UI constant and reference match the committed files.

Status: landed 2026-11-20.

### Phase 1 — Profile vocabulary closure

Close the profile loop across proto, runtime, CLI, and UI.

Gate:

```bash
cd scenarios/ai-gateway/api
GOWORK=off go test ./internal/gateway/... -count=1
```

Exit criteria: `AcceptedProfiles()` normalized values equal the proto enum minus
`PROFILE_UNSPECIFIED`; each accepted profile has one catalog row; the routing
preview CLI flag declares exactly that set; the UI renders catalog labels through
i18n rather than hardcoded copy.

Status: landed 2026-11-20. The CLI enum is declared at
`groups[name="routing"].commands[name="preview"].flags[name="profile"].values`.
Other profile flags remain open-coded and are covered only when they declare a
closed set.

### Phase 2 — Role inventory truth

Make the role catalog legible as inventory, not just routing order.

Gate:

```bash
cd scenarios/ai-gateway/api
GOWORK=off go test ./internal/gatewaycatalog/... -count=1
```

Exit criteria: every role in `config/inference-role-catalog.json` appears in the
generated roles table and candidate matrix; every provider in
`DefaultProviderSpecs` appears in the providers table; qualification candidates
are explicitly excluded from dispatch counts.

Status: landed 2026-11-20. Inventory rendering and coverage gates are in
place, and the D1 binding check is now gated directly against resource policy
files (`TestRoleCandidatesNameKnownResourceRoles`).

### Phase 3 — Provider registry single owner

Collapse the provider declaration paths so execution and the deployed catalog
cannot disagree.

Gate:

```bash
cd scenarios/ai-gateway
make verify-provider-catalog
```

Exit criteria: a declaration source names each provider's id, family,
locality, text, and spend capability; the deployed catalog is a pure projection;
supplier declarations cannot duplicate an entry owned elsewhere.

Status: landed 2026-11-20. `specs.seed.json` owns registered providers; supplier
declarations are limited to wrapper-less endpoints and the generator rejects
collisions with registered providers, resources, other supplier files, and
already projected entries.

### Phase 4 — Measures closure

Extend the `route_events.*` drift gate to every stateful domain before adding a
new one.

Gate:

```bash
cd scenarios/ai-gateway/api
GOWORK=off go test ./internal/gateway/... -count=1
# plus the measures-health probe for the touched domain
```

Exit criteria: every handler measure constant has a matching CLI command and
proto RPC; no CLI measure binds a method the proto service does not define; no
proto measure RPC lacks an operator command; the measures-health domain coverage
probe passes.

Status: landed. The gate discovers persisted entities with `created_at`, checks
that each is declared stateful, compares handler declarations/CLI commands/proto
RPCs, and requires a reasoned omission when no measure exists. The
`measures-health` probe reports Domain Coverage L3 with zero warnings; media,
inference recovery, consent, and private provider records remain intentionally
outside route analytics.

### Phase 5 — Operator surface projection

Stop hand-copying profile and role vocabulary into operator surfaces.

Gate:

```bash
cd scenarios/ai-gateway
make verify-gateway-catalog
cd ui && pnpm strings:check && pnpm type-check
```

Exit criteria: the routing preview profile enum and the capability reference are
generated or checked; no operator surface declares a profile list that the gate
does not compare; role summaries in the reference come from data or code, not
prose alone.

Status: landed 2026-11-20. CLI command bindings remain manifest-owned; all
closed profile/privacy values are checked, and the generated reference has
coverage tests for profile, request-kind, role, provider, and candidate entries.

### Phase 6 — Shared capability publication

Publish the existing native inventory over a shared, read-only contract so
cross-scenario discovery does not need to import Gateway internals.

Gate:

```bash
cd packages/proto && make verify-committed-gen
cd scenarios/ai-gateway/api
GOWORK=off go test ./internal/capabilitycatalog ./handlers/catalog ./internal/modules -count=1
cd ../.. && make endpoints
```

Exit criteria: the Connect RPC returns profiles, request kinds with modality
semantics, inference and code roles, role candidates, and provider family facts
projected from their native declarations. The shared contract explicitly marks
the catalog as inventory only: it confers no consumer authority, route
availability, or dispatch order.

Status: landed 2026-11-20. `CatalogService.GetCapabilityCatalog` publishes
`common.v1.AIGatewayCapabilityCatalog`; candidate inventory is sorted so the
role catalog's dispatch order does not leak as an implied consumer sequence.
The RPC is intentionally omitted from CLI bindings because its supported use
is cross-scenario Connect discovery.

## Validation Evidence and Scope

The focused implementation gates pass: API and CLI Go suites, all 98 UI tests,
UI string parity and type checking, `make check`, proto drift verification,
gateway/provider catalog checks, coverage floors, temporal models, measures
domain coverage, performance, storage, business, and tidiness.

The provider inventory runtime smoke also passes after fixing JSON-aware
redaction in `ExecRunner`: route preview returns valid inventory without
corrupting credential identifiers. The gateway restarted healthy with the
updated build identity. The capability catalog RPC was verified through its
generated contract and handler tests; a live HTTP call was blocked by the
repository's PreToolUse hard-stop policy.

The comprehensive Test Genie run `20261120-155254-6911edfc` reported 23 passed,
6 failed, and 1 skipped. After that run, the storage receipt declaration and
queued-row close were fixed; the targeted storage phase passed as
`20261120-160622-2f0d0814`. The remaining failed phases are outside this
capability-manifest program: workflow execution needs an authenticated BAS
account; experience-contract received `assessment.phase="experience"` instead
of `"experience-contract"` from Experience Manager; monetization-conformance
requires product/app registration decisions; Stripe test outcomes lack retained
provider receipts; model-eval lacks Search Hub denominator snapshots and an
available Agent Manager owner executor. The model-eval run did not attempt
inference. No paid inference was used for validation.

## Metrics

Track these at each phase review:

| Metric | Target | How measured |
|---|---|---|
| Generated artifacts stale | 0 | `make verify-gateway-catalog`, `make verify-provider-catalog` |
| Profile vocabulary copies outside declared owners | 0 | `TestCLIRoutingPreviewProfilesMatch` plus grep review |
| Selectable proto profiles missing a catalog row | 0 | `TestProfileCatalogCoversProtoEnum`, `TestAcceptedProfilesHaveCatalogRows` |
| Roles missing from the reference | 0 | `TestReferenceTablesCoverEveryRoleAndProvider` |
| `route_events.*` measures missing a CLI command or RPC | 0 | `TestRouteEventMeasuresMatchCLIAndProto` |
| Locale files out of parity after a profile change | 0 | `pnpm strings:check`, locale parity test |
| Time to add a profile | A catalog row, locale entries, and regeneration | Manual, by change shape |

## Anti-Goals

- **No dynamic capability registry.** A runtime service that lets an operator add
  a profile, role, or measure without a code change makes every drift gate
  decorative.
- **Do not move provider or model catalogs into the gateway.** Resources own
  concrete models, credentials, and context windows.
- **Do not generate resource policy.** The gateway projects resource-owned
  declarations; it does not become their editor.
- **No profile scoring or adaptive ordering in this program.** Profiles stay
  deterministic and previewable. Measures are analytics, never hidden scoring.
- **Do not make docs generation depend on a live scenario or resource CLI.** The
  generator must run from a clean checkout.
- **Do not treat catalog presence as route availability.** A role can be listed
  while every candidate is unconfigured or policy-blocked.
- **Do not add a profile without a routing-policy test.** A catalog row alone is
  not behavior.
- **Do not weaken a gate to make a stale artifact pass.** Regenerate the artifact
  or fix the owner.

## Cross-References

- [`../reference/capability-catalog.md`](../reference/capability-catalog.md) — generated profile, role, provider, and candidate inventory.
- [`../reference/roles-profiles-policies.md`](../reference/roles-profiles-policies.md) — role/profile/policy ownership and sampling contract.
- [`DECISIONS.md`](DECISIONS.md) — durable decisions, including closed profile/measure vocabularies.
- [`PROBLEMS.md`](PROBLEMS.md) — structural gaps D1–D8 with their current status.
- [`PROGRESS.md`](PROGRESS.md) — dated implementation history.
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system boundaries and owned domains.
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — domain ownership map.
