# Security — Web Search

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

> **Scaffold status (2026-06-09):** The posture below is the *intended*
> design from `PRD.md` / requirements. Nothing is implemented yet — these
> are the mandatory mitigations the implementation must honor, not
> validated controls.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## The Trust Boundary (read first)

web-search ingests **unvetted external content**. Live web results come
from arbitrary internet sources via the local SearXNG resource, and
`findings` are claims *distilled* from that content. The whole posture
of this scenario is built around treating that content as untrusted:

- **Findings are a peer corpus, not curated truth.** They are surfaced
  *with* provenance (citations + retrieval date) and uncertainty signals,
  never presented as established fact. See
  [`../concepts/DATA.md`](../concepts/DATA.md).
- **Findings are NOT written into knowledge-observatory.** KO owns
  curated documentation; web-search owns its own findings store
  exclusively. No doc/markdown is ever created in KO. This separation is
  a deliberate trust boundary — unvetted web-derived claims must never
  leak into the curated docs corpus. (PRD Non-goals.)

## Data Sensitivity

| Data | Sensitivity | Owner | Notes |
|---|---|---|---|
| Findings (cited claims) | low–medium (unvetted external) | findings | Distilled from public web content; carry source URLs + retrieval date. Surfaced with provenance and (on conflict) a dispute warning. Never treated as curated fact. |
| Briefs (research run output) | low–medium (unvetted external) | findings (produced by research) | Container for one run; same trust class as the findings it holds. |
| Query strings → SearXNG | low, privacy-relevant | livesearch | Sent to the **local** SearXNG resource only. SearXNG is privacy-respecting and does no external user tracking. Must not embed user PII. |
| Live-web result cache | low (ephemeral) | livesearch | TTL-evicted snippets from SearXNG; no durable corpus, no full-page storage. |
| Search-hub control token | secret (in-memory) | federation | Gates mutating provider verbs (reindex / config / query-overrides). |

**No user PII is stored in findings or briefs.** Findings are about the
world, not about users. Query strings should be free of personal data
before they reach SearXNG.

## Auth And Authorization

The scenario inherits no auth provider from the template. The
security-relevant access control is the **search-hub control-token
handshake**: mutating verbs on the registered providers
(reindex, config changes, query-overrides) must present the control
token; read/query verbs do not. Authorization belongs at the API/service
layer — the UI and CLI must not enforce it locally.

Add bearer/`API_TOKEN` auth only when a protected deployment surface
requires it (see [`configuration.md`](../reference/configuration.md));
the local-dev default is unauthenticated like the rest of the fleet.

## Secrets

| Secret | Source | Required? | Notes |
|---|---|---|---|
| search-hub control token | exchanged at registration / lifecycle env | yes (for mutations) | Gates reindex/config/query-override verbs. Never logged. |
| `API_TOKEN` | lifecycle env (optional) | no (dev) | CLI ↔ API bearer; enforce only in protected deployments. |
| Resource endpoints (SearXNG/Qdrant/Ollama/reranker URLs) | lifecycle env | yes (at runtime) | Local resource URLs, not credentials, but treated as config not hard-coded. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Unvetted external content treated as fact | Agents/operators act on false or biased web claims. | Every finding carries citations + retrieval date; disputed findings surfaced **with a "sources conflict" warning** and both sources, **never silently resolved**; findings are a peer corpus, never written into KO's curated docs. | intended (P0/P1) |
| Contradiction silently resolved | A real disagreement is hidden, eroding honesty. | Status state machine forbids silent resolution; `active → disputed` carries a `dispute_note`; resolution is an explicit, audited action via the dispute review queue (human command / targeted re-research / new evidence). | intended (P1, OT-P1-005/006) |
| External-engine rate-limit / ban | SearXNG or upstream engines block us; live search dies. | Live web off the default federated path (SCOPE_EXTERNAL, gated); TTL cache dampens repeats; token-bucket budget governor caps external QPS per window and returns a graceful "rate-limited, try later" with **no external call**. | intended (P0, OT-P0-007) |
| User PII leaking to external engines | Privacy exposure via query strings. | SearXNG is local and privacy-respecting (no external user tracking); no user PII stored in findings/briefs; queries should be PII-free before dispatch. | intended (P0) |
| Unauthorized mutation of the providers | Corpus poisoning or routing tampering via reindex/config/override. | search-hub control-token gates all mutating verbs; read paths are open. | intended (P0, federation) |
| Store growth / rot | Stale or unused findings degrade recall over time. | Supersede-not-delete with audit trail; age-based confidence decay; per-query reconcile; P2 usage-telemetry curation + GC. | intended (P1/P2) |
| Findings hard-deleted / lost | Loss of auditability/recoverability. | Findings are **never** hard-deleted — supersede/archive only; archived rows retained with provenance; SQLite "migrate, never recreate" policy. | intended (P0/P1) |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Nothing implemented yet | n/a (scaffold) | All mitigations above are design intent; validate as each level (L0–L3) ships. |
| Budget governor + cache unproven | high | Required before live web can be auto-routed (P2 OT-P2-002 is explicitly gated behind a proven cache + governor). |
| Dispute review queue not built | medium | Required for OT-P1-005/007; until then disputed findings surface with a warning but have no resolution UI. |
| No multi-user auth model | conditional | Required before any protected or multi-tenant deployment. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership, retention, privacy notes
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — SearXNG / search-hub / browserless contracts and failure modes
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — finding status state machine and budget governor
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
