# Business Bundle

> Offer Desk is authoritative for this offer's lifecycle, variants, members,
> and trigger records. This document retains bundle positioning, dependency
> rationale, and expansion judgment rather than a live catalog snapshot.

**SKU ID:** `business`
**Target audience:** Solo entrepreneurs, independent developers, and small teams running lean software/business operations.
**Positioning:** A bundle that lets one person run a whole business — software development, project management, writing, finance, triage — as a single integrated ecosystem.

## Value proposition

Most indie operators juggle a dozen disconnected tools. The business bundle replaces them with a coherent set of Vrooli scenarios that share local resources, share agent context, and compound into capability the user owns. A subscriber gets the bundle's apps plus (on paid tiers) integrated API access so they never configure OpenAI, Anthropic, ElevenLabs, etc., separately.

## Headliner layer

Scenarios that are compelling on their own AND deployable with today's capabilities. These are the acquisition hooks — a prospect buys the bundle for one of these, then discovers the rest.

| Scenario | Why it's a headliner | Deployability today |
|---|---|---|
| `web-console` | Standalone appeal: lets a solo operator manage their own infrastructure without learning cloud-provider UIs. Compelling pitch on its own. | Deployable |
| `git-control-tower` | Standalone appeal: a powerful Git workflow surface. Can operate without its optional dependencies (agent-manager, etc.). Strong acquisition draw for developers. | Deployable |

Additional headliners are not pre-nominated. Depth-layer scenarios promote into the headliner set only when their prereqs ship and they cross the standalone-appeal + deployable-today bar. Catalog-strategist proposes each promotion via a `catalog-promotion` decision when it detects a crossing; the operator decides.

## Depth layer

Scenarios that either (a) sharply amplify a headliner once they ship, or (b) are themselves future headliners whose prereqs aren't built yet.

### Amplifiers — make existing headliners dramatically better

| Scenario | Amplifies | Effect |
|---|---|---|
| `agent-manager` | `git-control-tower` | Unlocks multi-agent workflows within GCT — turns GCT from a Git surface into an agent-orchestrated development platform. Major headliner-value boost. |
| `workspace-sandbox` | `git-control-tower`, `agent-manager` | Safe isolated execution environments for agents; prereq for trusted autonomy. |

### Future headliners — currently blocked by dependencies

| Scenario | Standalone appeal | Blocked by |
|---|---|---|
| `swarm-manager` | Very high — a project-management surface for backlogs, initiatives, and autonomous work. Vrooli uses it internally; external users would find it valuable. | `agent-manager` → `workspace-sandbox` |
| `prompt-manager` | Medium-high — skills, agents, teams, and the heartbeat system. More interesting to power users than the average subscriber. | Matures alongside agent-manager |

## Dependency DAG

```
                workspace-sandbox
                      │
                      ▼
                 agent-manager ──┐
                      │          │ (amplifies)
                      │          ▼
                      │     git-control-tower  ◄── headliner (ships today)
                      │
                      ▼
                swarm-manager         web-console  ◄── headliner (ships today)
                (future headliner)
```

The DAG is directional, not temporal. "Blocked by" means the upstream scenario must be shipped + deployable for the downstream one to become a candidate headliner. When `agent-manager` ships and stabilizes:

1. `git-control-tower` becomes dramatically more compelling → the business bundle's headliner appeal increases without adding a new scenario.
2. `swarm-manager`'s deployment-readiness gate partially clears → `swarm-manager` moves one step closer to headliner-eligible.

This is why catalog-strategist's daily loop is not vague strategic thinking but a concrete graph traversal: *did any upstream dependency ship? If yes, which downstream scenarios changed status?*

## Ordering rationale

The business bundle ships first among base bundles because:

1. **Audience is reachable.** Developers and solopreneurs are identifiable, addressable, and habitually pay for tools.
2. **Capability alignment is strongest today.** Vrooli's core strengths — code generation, agent orchestration, local-first automation — map directly to the bundle's scenarios.
3. **Revenue from this bundle funds everything else.** Lifestyle bundle needs capabilities not yet built; waiting for business-bundle cash flow is the pragmatic path.

Within the bundle, `web-console` and `git-control-tower` ship as initial headliners because they meet both criteria (standalone appeal + deployable today). Depth-layer scenarios promote into the headliner set as they become ready.

## Financial-planning overlap

Financial-planning capability is expected to belong to both the business and lifestyle bundles once it exists. Offer Desk's `belongs_to` graph is many-to-many by design.

## Expansion hooks (for agent-driven in-bundle discovery)

Each depth-layer scenario, once shipped, becomes a candidate in-bundle expansion target. The recommended expansion surface is **the agents themselves** — when a user is doing work in one scenario, agents suggest a relevant other scenario from the bundle. This is the primary retention mechanism for this bundle.

Do not default to email drips or in-app pop-ups for expansion in this bundle. Agents have better context.

## Live posture

Offer Desk exposes current headliner, depth-layer, membership, and lifecycle
records at read time. Money Ledger exposes subscriber and revenue posture when
those observations exist; this document intentionally carries no current
counts.

## Candidate add-ons parented to this bundle

- [property-services](../addons/property-services.md)

## Consumer products tied to this bundle

Consumer products for the business bundle are narrow. Business-bundle users come for the tool to work, not for merch. The legitimate sub-cases where consumer products fit here are:

- **Generated-asset output** — scenarios that produce printable artifacts as part of their normal workflow (business cards, branded stationery, printed marketing campaigns) can offer fulfillment via print-on-demand. The printed output is a natural completion of the user's task, not an advertising surface.
- **Paid deep-dive content** — scenario-specific workflow guides, advanced-usage books, paid courses on bundle capabilities. Complement subscription rather than replace it.

Affiliate links have narrow fit in this bundle — developer-adjacent goods (books, hardware accessories) only. Most business-bundle monetization remains subscription + services.

See [consumer-products](../../revenue-lines/consumer-products.md) and [affiliate-commerce](../../revenue-lines/affiliate-commerce.md) for the architectural and UX rules all such offerings must obey. Same gating applies: no activation until inventory-aware state and recommendation-blindness post-processor exist.
