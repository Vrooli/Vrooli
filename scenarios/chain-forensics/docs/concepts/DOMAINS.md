# Domains — Chain Forensics

The scenario is a local-first evidence workstation and reusable trace engine. Its
domain vocabulary keeps chain observations, protocol-level correlation, analyst
hypotheses, and attribution claims separate so a later interpretation cannot
rewrite what a public source actually showed.

## Domain inventory

| Domain | Primary Archetype | Responsibility | Owns | First implementation | Source Paths |
|---|---|---|---|---|---|
| Case analysis | service | Reconstruct a bounded question into a source-linked graph, grade edges, record gaps, and prepare a private evidence packet. | Case metadata, flow graphs, analyst notes, packet lifecycle | Public 1HQ3 forfeiture case, with DOJ allegations kept separate from Bitcoin observations | `packages/chain-forensics/examples/cases/us-v-1hq3/`; future scenario case repository |
| Source acquisition | service | Retrieve normalized observations from free, keyless public chain interfaces; preserve capture time, URL, response hash, and completeness. | Provider adapters, cache entries, source terms, rate-limit/backoff state | mempool.space BTC address and transaction reads | `packages/chain-forensics/mempool_client.go`; future scenario source adapters |
| Protocol correlation | service | Decode token, launch, bridge, and service events and join matching occurrences across chains. | Versioned decoders, protocol identities, cross-chain correlation candidates | Deferred until the BTC source/event contract is stable | Future package protocol decoders |
| Labels and clusters | service | Store sourced public labels and evaluate explainable deposit-cluster rules without turning them into identity claims. | Label provenance, rule versions, candidate membership, counter-evidence | Deferred; requires independently sourced exchange-deposit labels | Future private scenario label store |
| Health | reporting | Report service readiness and dependency reachability. | No product data | Generated operational domain | Scenario lifecycle health endpoints |

## First vertical slice: published BTC forfeiture case

The first domain is **Case analysis**. It owns the case view and packet, while
**Source acquisition** supplies observations through an injectable interface.
The public case source records the U.S. complaint's narrative as a sourced legal
assertion. BTC transactions from mempool.space independently establish the
corresponding transaction inputs, outputs, block times, and satoshi amounts.
These are different evidence kinds and must remain distinct.

The initial flow is: identify the public case by ID → load the local case seed →
read or replay captured BTC source observations → normalize transaction outputs
into chain-specific occurrences → build typed edges → grade direct on-chain
edges separately from legal narrative links → render a local packet with source
URLs, hashes, gaps, and counter-explanations. The slice does not identify the
owner of an address or infer a person's intent.

| Surface | First slice obligation |
|---|---|
| API | Return typed case facts, events, edges, provenance, grades, and gaps; never merge legal assertion with chain observation. |
| CLI | `chain-forensics cases inspect <id> --json` must produce the same normalized result for scripts. |
| UI | Case review must show source facts and interpretations separately, with loading, empty, partial, and source-error states. |
| Go package | Deterministic chain-event validation, graph construction, grade rules, and packet serialization stay independent of HTTP, SQLite, and RPC clients. |

## Later domains and ownership boundaries

Cross-chain flow resolution composes observations from the same chain before
attempting bridge joins. A bridge edge is **confirmed** only when source and
destination protocol evidence match on bridge/message identity and relevant
asset/amount/recipient fields; source-only events remain pending. Exchange
cluster membership is a versioned hypothesis supported by a public deposit
address label and observable transaction evidence. Membership never means
common legal or natural-person control.

Treasury reconciliation is a consumer of the Go package. It must compare the
chain view against wallet/fork expectations without reading Treasury's private
database or accepting its labels as chain evidence. CI2 is the intended external
consumer; its owner path is not present today, so CI2 import evidence remains
unmet until the actual evaluator imports and tests this package.

## Non-domains

- Case collection is not a public tip service: packet storage is local by
  default, and there is no submission, publishing, contact, or attention-view
  route.
- Public labels do not become person records. The product has no natural-person
  identity resolution domain.
- Price and profit analytics are not part of flow quantity. Native units and
  decimal atomic quantities remain separate from valuation.
- Scenario API transport, SQLite infrastructure, and frontend shared widgets
  are implementation infrastructure, not product domains.

