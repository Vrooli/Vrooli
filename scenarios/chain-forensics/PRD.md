# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario chain-forensics`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

Purpose: Chain Forensics is a permanent, self-hostable capability for reconstructing public digital-asset flows as source-linked facts, separating those facts from analyst interpretation and attribution. It lets investigators and Vrooli scenarios follow bounded transaction graphs across Bitcoin, Ethereum, Base, and Solana, with explicit chain-data gaps and confidence grades. Primary users are authorized fraud-response analysts, public-interest investigators, compliance teams working from public evidence, and Vrooli capability owners who need repeatable linkability analysis. Deployment surfaces are a direct case-review UI, a typed Connect/API contract, stable CLI, locally retained case and private packet storage, and a reusable Go library imported by scenario owners such as Crypto Infrastructure CI2. The full CF6 workstream is a 100–200 hour capability program: BTC, EVM, Solana SPL and Pump-style launch decoding; Across and Wormhole correlation; evidence-based exchange deposit clusters; three private showcase cases; and a real CI2 import. A generated scaffold or one public-case trace is only an initial milestone, not mission completion.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Source-linked public case reconstruction | Reconstruct the public 69,370 BTC forfeiture complaint as typed chain events and a trace graph; distinguish court allegations, on-chain observations, interpretations, and gaps; every fact has a source link and capture identity; attribution is graded and does not identify a natural person from an address.
- [ ] OT-P0-002 | Private case and packet boundary | Create and review local-only cases and redacted tip-ready evidence packets with source citations, counter-explanations, unresolved gaps, and explicit grades; no automatic submission, public publishing, contacting, or accusation is available.
- [ ] OT-P0-003 | Reusable typed graph contract | Define versioned chain-neutral case, occurrence, asset, source, flow, bridge-message, cluster, attribution, and packet types plus deterministic graph operations in an importable Go package; the API and CLI consume the same typed model.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Multi-chain public source coverage | Add bounded, rate-aware, keyless readers for Bitcoin, Ethereum, Base, and Solana, including SPL transfers and Pump-style launch instructions; cache immutable source responses and expose pruned history, pagination, RPC, and address-history limitations.
- [ ] OT-P1-002 | Bridge and cluster evidence | Correlate Across and Wormhole from source and destination chain events, then produce reproducible exchange deposit clusters only when each rule has public source support, a version, a counter-explanation, an evidence grade, and an unresolved-gap record.
- [ ] OT-P1-003 | Investigator workflow | Deliver a polished responsive case-review UI, documented Connect/API endpoints, stable CLI output, and useful loading, empty, error, partial-history, stale-provider, rate-limited, export, and privacy states; keep human labels separate from chain facts.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Private cross-chain rug showcase | Trace one real public rug from its launch through a bridge to a publicly evidenced exchange cluster in a private packet; if a candidate cannot meet the evidence floor, retain it as unmet and do not lower grades or publish an accusation.
- [ ] OT-P2-002 | Treasury fork reconciliation | Reconcile Treasury fork-wallet activity from chain data alone, including replay, removal, re-inclusion, and exact quantity; do not read Treasury databases, use signing keys, broadcast, or modify Treasury code.
- [ ] OT-P2-003 | Crypto Infrastructure CI2 adoption | Have the owner-owned CI2 evaluator import and test the same Go library against seeded linkability cases; claim adoption only after the real CI2 consumer path and test pass, and leave the target open if no owner-owned import path exists.

## 🧱 Tech Direction Snapshot

Preferred stack: Go for deterministic evidence, graph, attribution, and source-adapter library code; React/TypeScript for the case-review UI; Connect for typed scenario/API contracts; and a stable CLI for scripted case operations. Store cases and user labels locally with source provenance as immutable records. Keep observed chain facts, interpretations, public labels, and attribution assessments in distinct typed fields. Bitcoin: use documented keyless mempool.space REST reads; bounded address/transaction history, cache by immutable transaction/block identity, honor rate responses, and never broadcast. Ethereum and Base: use public JSON-RPC for known transaction, receipt, block, and bounded log reads; raw RPC does not provide address-wide native history, and Blockscout's current API migration requires an API key so it is excluded. Solana: use the official public RPC with finalized commitment, bounded signatures/transaction reads, 429-aware backoff, and explicit pruned-history gaps; it is rate-limited and not a production SLA. Pump-style launch decoding uses official program documentation/IDL plus chain facts, never paid trading APIs. Across evidence comes from V3FundsDeposited and FilledV3Relay on source/destination chains. Wormhole evidence follows on-chain lock/burn, VAA identity, and destination mint/release. Exchange names require an explicit public source; otherwise label only a service-like cluster. No paid APIs/indexers, credentials, or new secret material. Cache reads by chain/request/block identity, bound depth/pages/concurrency/bytes, expose source capture identity, and separate valuation from quantity.

## 🤝 Dependencies & Launch Plan

Required resources: begin with scenario-owned local storage and the existing Vrooli lifecycle, API, UI, proto, and Test Genie substrate; add no blockchain node or paid hosted indexer. Scenario dependencies: none are required for the first published BTC vertical slice; CI2 is a later owner-owned consumer, not a fake dependency. Data integrations are anonymous public reads against the source endpoints and official documentation named in the PRD appendix; use bounded concurrency, request caching, retry-after handling, source capture IDs, and transparent degraded behavior. Operational risks include provider throttling or reorg/pruned history, incomplete address history on raw EVM RPC, ambiguous bridge matching, unreliable service labels, and attribution overreach. Launch sequencing follows the 8-phase Plan Manager plan: qualify/generate the scenario, build the public BTC slice, add EVM/Base/Solana decoders, correlate two bridges, grade clusters, pursue a private rug case, prove Treasury fork reconciliation and actual CI2 import, then harden and checkpoint. Do not access private victims, contact people/exchanges/law enforcement, publish accusations or packets, submit tips, sign, broadcast, or inspect Treasury secrets/databases.

## 🎨 UX & Branding

Look and feel: a restrained investigative workspace that keeps source evidence legible beside the graph, with confidence and source-quality badges that cannot be confused with proof of personal identity. Accessibility: WCAG AA contrast, keyboard-complete graph and case review, visible focus, accessible tables and status announcements, responsive 390px and desktop layouts, and clear loading, empty, error, partial, stale, rate-limit, and privacy states. Voice: precise, neutral, and evidence-first; label court allegations as allegations and chain observations as observations. Make the privacy boundary visible before any packet export, default to redacted local output, and require explicit operator action for local export. Branding should communicate a professional public-evidence tool without implying law-enforcement authority or certainty of attribution.

## 📎 Appendix

Primary source and contract references: U.S. v. 69,370 BTC forfeiture complaint (2020-11-05): https://www.justice.gov/d9/press-releases/attachments/2020/11/05/complaint_crypto_seizure.pdf ; mempool.space REST API docs: https://mempool.space/docs/api/rest ; Solana public RPC documentation: https://solana.com/docs/rpc ; Solana cluster docs: https://solana.com/docs/references/clusters ; Across intent lifecycle and event correlation: https://docs.across.to/guides/concepts/intent-lifecycle ; Wormhole Token Bridge: https://wormhole.com/docs/products/token-bridge/overview/ ; Wormhole VAAs: https://wormhole.com/docs/protocol/infrastructure/vaas/ ; Pump program docs/IDL: https://github.com/pump-fun/pump-public-docs/blob/main/docs/PUMP_PROGRAM_README.md ; Pump bonding curve: https://pump.fun/docs/bonding-curve ; Blockscout API-key migration notice: https://www.blog.blockscout.com/switch-to-the-blockscout-pro-api/ . These sources establish public interfaces or published allegations, not identity ownership or guilt.
