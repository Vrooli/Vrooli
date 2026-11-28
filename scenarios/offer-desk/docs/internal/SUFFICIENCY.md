# Monetization sufficiency dossier

This dossier records which hand-maintained surfaces are replaced by the pair
and supports the live monetization-team adoption. It distinguishes machine
state from surviving judgment so the team does not recreate a second catalog
or ledger.

| Hand-maintained surface | Replacement | Requirement | Evidence | Disposition |
|---|---|---|---|---|
| Offer/channel/revenue-line catalog | Offer Desk typed catalog graph | MIG-001, MIG-003 | `docs/internal/PROGRESS.md` (Phase 6 import and Phase 8 validation) | replaced for lifecycle state; narrative remains prose |
| Tier and bundle pricing matrix | Offer Desk variants and `sells_at` edges | GRAPH-002, MIG-001 | `docs/internal/PROGRESS.md` (Phase 6 import and Phase 8 validation) | replaced for declared structure |
| Scenario-to-SKU membership map | Offer Desk `belongs_to` edges and findings | INT-001 | `docs/internal/PROGRESS.md` (Phase 8 retirement and validation) | replaced with validation |
| Candidate trigger and promotion tracking | Offer Desk gates, evaluations, proposals | GATE-001, GATE-002, UI-002 | `docs/internal/PROGRESS.md` (Phase 8 validation) | replaced for machine-evaluable state |
| Cash, burn, and revenue input sheet | Money Ledger operator import and journal | CTR-006, POS-001 | `../../../money-ledger/docs/internal/PROGRESS.md` (Phase 7 financial cutover) | replaced for admitted monetary facts |
| Time allocation and hours | Money Ledger operator measures | POS-002, CTR-006 | `../../../money-ledger/docs/internal/PROGRESS.md` (Phase 7 financial cutover) | replaced as measures, never money |
| MRR field | Money Ledger derived-rate finding | CTR-006 | `../../../money-ledger/docs/internal/PROGRESS.md` (Phase 7 financial cutover) | not a posting; remains derived from future commerce events |
| Four canonical financial rules | Money Ledger declared goals | POS-002 | `../../../money-ledger/docs/internal/PROGRESS.md` (Phase 7 financial cutover) | replaced as explicit thresholds |
| Offer posture beside finances | Offer Desk board reading Money Ledger | INT-002, INT-005 | `docs/internal/PROGRESS.md` (Phase 8 validation) | replaced when ledger is available; named degradation otherwise |
| Obligation-space summary | Offer Desk `space --projection offers` | INT-004 | `docs/internal/PROGRESS.md` (Phase 8 validation) and `docs/spaces/offers.json` | replaced for the current authored denominator |
| Judgment prose, positioning, and philosophy | Source documents and live team adoption | MIG-003 | `docs/internal/PROGRESS.md` (Phase 8 retirement) | not replaced; intentionally remains human judgment |
| Cross-scenario journey authoring | Experience Manager capability request `experience-manager-cross-scenario-journeys` | experience contract boundary | `docs/internal/PROGRESS.md` (Phase 8 retirement) | not replaced; platform request remains a boundary |

## Phase 8 source disposition — approved

This is the file-by-file disposition required before retiring any source under
`docs/monetization/`. The table is the approved disposition. Every
file has exactly one disposition:

- `replaced`: the file is machine state only and may be removed after its
  records are queried and the live backup is verified.
- `judgment`: the file carries strategy, rationale, policy, taxonomy, or
  operator guidance that no scenario record replaces; it remains.
- `mixed`: the file contains both extracted state and judgment. The records
  are authoritative in the named instrument; the reasoning remains in the
  file, with a short authority note added after approval.

The Phase 6 catalog cutover wrote 36 records and reported 40 nonblocking
reference findings. The current Offer Desk graph contains 13 `belongs_to`
edges, 17 `sells_at` edges, 5 triggers, 7 proposals, 1,850 evaluation runs,
and 40 migration findings. The Phase 7 financial cutover read all 13 live
operator-input fields and wrote no postings or measures because every field is
absent or not applicable; Money Ledger holds seven declared goals and an
UNKNOWN position until observations arrive. Phase 8 then made Offer Desk the
live instrument and left Money Ledger as its covered financial source.

| # | Source path | Disposition | Replacement or surviving judgment | Reason |
|---:|---|---|---|---|
| 1 | `README.md` | `judgment` | Surviving plan-of-record entrypoint and navigation | It explains how to read the canon and its editing authority; it is not state. |
| 2 | `catalogs/CATALOG.md` | `mixed` | Offer Desk catalog graph for lifecycle records; keep lifecycle rules and guardrails | The tables and current status are state, while promotion discipline and guardrails are durable judgment. |
| 3 | `catalogs/README.md` | `judgment` | Surviving catalog module guide | It is a navigation and interpretation guide, not a record source. |
| 4 | `catalogs/channels/README.md` | `judgment` | Surviving channel-axis rules and interpretation | It defines what a channel means and how it differs from marketing; the live channel rows move to Offer Desk. |
| 5 | `catalogs/channels/app-stores.md` | `mixed` | Offer Desk channel node; keep channel hypothesis, anti-patterns, and activation judgment | The channel posture and telemetry status are extracted state; the hypothesis and operating discipline remain judgment. |
| 6 | `catalogs/channels/community-content.md` | `mixed` | Offer Desk channel node; keep audience, safety, and cadence judgment | The lifecycle posture is machine state; the channel strategy and guardrails are not. |
| 7 | `catalogs/channels/in-product-expansion.md` | `mixed` | Offer Desk channel node and `feeds` relationship; keep expansion rationale | The active posture and feed relationship are records; the expansion logic is judgment. |
| 8 | `catalogs/channels/oss-discovery.md` | `mixed` | Offer Desk channel node; keep OSS discovery and trust rules | The channel lifecycle is state; the distribution strategy and anti-patterns are judgment. |
| 9 | `catalogs/channels/skill-registries.md` | `mixed` | Offer Desk channel node; keep publication, security, and recommendation-blindness rules | The channel record is replaced; the detailed policy remains human-authored judgment. |
| 10 | `catalogs/channels/web-seo.md` | `mixed` | Offer Desk channel node; keep acquisition and landing-page strategy | The channel posture is state; the strategy and constraints remain. |
| 11 | `catalogs/revenue-lines/README.md` | `mixed` | Offer Desk revenue-line nodes; keep lifecycle and services discipline | The registry is state; the architectural and activation rules are judgment. |
| 12 | `catalogs/revenue-lines/affiliate-commerce.md` | `mixed` | Offer Desk revenue-line node; keep recommendation-blindness and disclosure rules | The line record is extracted; the safety and UX policy remains. |
| 13 | `catalogs/revenue-lines/app-development.md` | `mixed` | Offer Desk revenue-line node; keep execution-risk and activation judgment | The lifecycle posture is state; the line's strategic rationale remains. |
| 14 | `catalogs/revenue-lines/consulting.md` | `mixed` | Offer Desk revenue-line node; keep last-resort and activation judgment | The revenue-line record is extracted; the strategic boundary remains. |
| 15 | `catalogs/revenue-lines/consumer-products.md` | `mixed` | Offer Desk revenue-line node; keep inventory, recommendation, and UX rules | The line state is replaced; the architectural constraints are durable judgment. |
| 16 | `catalogs/revenue-lines/flipping.md` | `mixed` | Offer Desk revenue-line node; keep candidate rationale and capability constraints | The candidate posture is state; the business-case reasoning remains. |
| 17 | `catalogs/revenue-lines/lead-generation.md` | `mixed` | Offer Desk revenue-line node; keep legal surface and candidate playbooks | The line record is extracted; legal and playbook judgment must remain visible. |
| 18 | `catalogs/revenue-lines/subscription.md` | `mixed` | Offer Desk revenue-line node; keep portfolio role and instrumentation judgment | The product line record is state; the portfolio rationale remains. |
| 19 | `catalogs/scenario-sku-map.json` | `replaced` | Offer Desk `belongs_to` graph: 13 live edges | It is a machine-only many-to-many membership map; Offer Desk now owns that state and its validation findings. |
| 20 | `catalogs/skus/addons/elder-care.md` | `mixed` | Offer Desk offer node; keep hypothesis, candidate rationale, and future questions | Status and revisit trigger are state; the hypothesis and decision criteria are judgment. |
| 21 | `catalogs/skus/addons/family-with-kids.md` | `mixed` | Offer Desk offer node; keep hypothesis, candidate rationale, and future questions | Status and revisit trigger are state; the hypothesis and decision criteria are judgment. |
| 22 | `catalogs/skus/addons/property-services.md` | `mixed` | Offer Desk offer node and service relationship; keep legal and productization judgment | Promotion state and trigger are records; the vertical and legal reasoning remains. |
| 23 | `catalogs/skus/base/business.md` | `mixed` | Offer Desk offer/variant graph; keep bundle positioning and dependency rationale | Active status, tier structure, and membership are state; bundle strategy remains. |
| 24 | `catalogs/skus/base/lifestyle.md` | `mixed` | Offer Desk offer/variant graph; keep market thesis and gating rationale | Candidate status and trigger are state; the long-term positioning and open questions remain. |
| 25 | `evidence/BENCHMARKS.md` | `judgment` | Surviving evidence protocol and explicitly empty pre-launch benchmark ledger | No benchmark facts exist to extract; the source discipline and expected comparison method remain. |
| 26 | `evidence/FINANCIAL_MODEL.md` | `mixed` | Money Ledger position/goals for live posture; keep formulas, assumptions, and honesty conventions | Current observations and thresholds belong in Money Ledger; the model's conceptual math remains judgment. |
| 27 | `evidence/README.md` | `judgment` | Surviving evidence module guide | It explains evidence organization and does not duplicate live facts. |
| 28 | `evidence/TELEMETRY_ROADMAP.md` | `judgment` | Surviving capability-gap and measurement roadmap | It is a prioritization document for future telemetry, not current telemetry state. |
| 29 | `governance/HOW_TO_GATHER_INPUTS.md` | `judgment` | Surviving field-by-field operator request | It tells the operator how to supply absent values; the values themselves live in the input adapter and Money Ledger. |
| 30 | `governance/adoption-validation.md` | `judgment` | Surviving validation runbook | It describes how to validate adoption and contains no replaceable business state. |
| 31 | `governance/changelog.md` | `judgment` | Surviving historical canon changelog | Historical decisions are not reconstructed by a scenario record. |
| 32 | `governance/editing.md` | `judgment` | Surviving authority and change-flow policy | It defines who may curate canon and is not a state surface. |
| 33 | `manifest.json` | `judgment` | Surviving machine-readable plan-of-record contract | It describes the document contract and must remain for PoR validation. |
| 34 | `operating/OPERATING_MODEL.md` | `judgment` | Surviving operating loops and ownership boundaries | It explains judgment and workflow; read-time instrument results replace only its live-state snapshots. |
| 35 | `operating/README.md` | `judgment` | Surviving operating module guide | It is navigation and validation guidance, not current state. |
| 36 | `strategy/FUNNEL.md` | `judgment` | Surviving funnel definitions and strategic rules | Funnel definitions and interpretation are not replaced by Offer Desk records. |
| 37 | `strategy/PRICING.md` | `mixed` | Offer Desk `sells_at` relationships (17 live edges; 8 added in Phase 6); keep pricing principles and unresolved decisions | The matrix's declared relationships are state, while price posture and decisions remain judgment; TBD cells are not fabricated into facts. |
| 38 | `strategy/README.md` | `judgment` | Surviving strategy module guide | It is navigation and extension policy, not state. |
| 39 | `strategy/STRATEGY.md` | `judgment` | Surviving monetization positioning and long-term strategy | It carries principles and rationale, not replaceable records. |
| 40 | `strategy/TIERS.md` | `mixed` | Offer Desk variants and tier relationships; keep delivery-mode rationale and guardrails | Tier lifecycle/status is state; the delivery strategy and pricing rationale remain. |
| 41 | `taxonomies/monetization-opportunity/README.md` | `judgment` | Surviving opportunity classification rules | Taxonomy definitions and evidence discipline remain human-curated policy. |
| 42 | `taxonomies/monetization-opportunity/taxonomy.json` | `judgment` | Surviving machine-readable opportunity taxonomy | It is a routing contract, not a current opportunity record. |
| 43 | `taxonomies/monetization-validation/README.md` | `judgment` | Surviving validation classification rules | Taxonomy definitions and materiality thresholds remain policy. |
| 44 | `taxonomies/monetization-validation/taxonomy.json` | `judgment` | Surviving machine-readable validation taxonomy | It is a routing contract, not a current validation record. |

**Approval status:** approved by the operator on 2026-08-16 through the
continuation instruction for this plan. The approved removal set is limited to
`catalogs/scenario-sku-map.json`; all other files either retain judgment or
received the approved mixed-content authority edit. The retired file is
recoverable at `~/.vrooli/retired/adopt-offer-desk-and-money-ledger-20260816/`.
