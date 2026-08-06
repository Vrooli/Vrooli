# Decisions — Document Manager

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-05 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-05 | **Collapse three scenarios into one.** `document-manager` (repo documentation quality), `secure-document-processing` (encryption/compliance shell), and `data-structurer` (schema extraction) are all retired; this scenario reclaims the `document-manager` name. | ~14,400 lines of Go across three scenarios that all predated AI Gateway, carried none of the current template surface, overlapped each other more than they served the actual need, and had no dependents outside prose mentions in two READMEs. | One coherent charter instead of three partial ones. The old trees must be removed, not left to confuse discovery. | If a retired capability turns out to have had a real consumer. |
| 2026-08-05 | **This scenario exposes no search or index endpoint.** | The retired `document-manager` built `POST /api/index` and `POST /api/search` over its own Qdrant collection. Rebuilding that shape would create a second semantic corpus competing with the ledger and break the unified cross-scope recall that `vrooli-memory` decision D9 makes the product. | This scenario owns bytes, derivations, versions, anchors, sensitivity and custody. Meaning and recall belong to `vrooli-memory`; federation to `search-hub`. Resisting a "just add search" request is a standing obligation. | Never, without a corresponding change to the ledger's recall architecture. |
| 2026-08-05 | **Sensitivity resolves before tier selection, not after.** | A confidential document must never be *eligible* for the remote tier, rather than being blocked from it once selected. | The residency guarantee is structural instead of advisory, so there is no fallback path that can get it wrong. Constrains the pipeline ordering permanently. | Never. Reversing this invalidates the product's central claim. |
| 2026-08-05 | **Re-derivation mints a new version; it never overwrites.** | Parsers and models improve, and a citation minted last year must still resolve. | `derivation_versions` is append-only, storage grows with each re-derivation, and anchor resolution must be version-aware. Accepted cost. | If storage growth becomes untenable — but prune parse outputs (`regenerable: true`) before considering version collapse. |
| 2026-08-05 | **The custody journal is append-only and outlives the document.** | Chain-of-custody evidence that can be edited or deleted is not evidence. | No repository method issues UPDATE or DELETE on `custody_records`. Document deletion writes a tombstone. Retention horizon for the journal itself is still undecided. | When a legal retention limit requires journal expiry — which is a policy decision, not an engineering one. |
| 2026-08-05 | **Security is repositioned from encryption to provable processing locality.** | `secure-document-processing` treated AES-256 and a compliance-framework badge wall as the product. Encryption at rest is table stakes; the defensible claim is a per-document record of where each step executed. | Vault and MinIO are deferred rather than adopted. Marketing must not claim certification we do not hold. The claim depends on AI Gateway route evidence, which creates an upstream dependency on a gap that is not yet closed. | If a buyer requires managed key custody beyond OS-level disk encryption. |
| 2026-08-05 | **Nothing is gated; the paid tier is metered only.** | Canon forbids gating what a self-hoster could run with their own keys, and BYOK must stay valid. | Free tiers must be genuinely good, because they compete with Paperless-ngx rather than with a crippled trial. Revenue depends entirely on real marginal-cost operations. | If a gated feature is ever proposed, treat it as a signal the framing drifted rather than as a pricing question. |
| 2026-08-05 | Keep the template's unused `fileRootPath` seam with a `//nolint:unused` directive rather than deleting it. | `react-vite` v1.6.5 ships this documented-as-mandatory routed-storage seam unreferenced, so the template's own `golangci-lint --fix` post-hook fails on every fresh generation. | Generation completes cleanly here; the seam is preserved for the artifact store, which is its first real caller. The underlying template defect is upstream and unfixed. | Remove the directive when the corpus artifact store calls it. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| 2026-08-05 | `document-manager` is an AI-powered documentation-quality SaaS for tracking repository documentation, with a $25K–50K revenue target. | This scenario's charter: local-first document ingestion and understanding. | The old scenario managed *documentation about repos*, not documents. Its improvement-queue shape survives as the low-confidence review queue; nothing else carried forward. |
| 2026-08-05 | `secure-document-processing` is an enterprise encryption and compliance pipeline ("SecureVault Pro"). | Custody and sensitivity domains in this scenario. | The security *thesis* was right — security is why someone pays — but the implementation was an upload form with encryption-themed styling and unimplemented compliance frameworks. Reframed from "we encrypt it" to "we can prove where it ran." |
| 2026-08-05 | `data-structurer` is a standalone scenario converting unstructured data to schema-defined structured data. | The `enrichment` domain's schema-first extraction verb (`DOC-P1-004`). | The charter was sound; the implementation called Ollama directly and created Postgres tables per schema. Absorbed rather than discarded. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
