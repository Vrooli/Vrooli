# Decisions — Content Desk

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
| 2026-07-28 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-07-28 | **Replace `campaign-content-studio` rather than migrate it.** | The fleet's in-place-modernization precedent (`landing-page-business-suite`) was chosen to protect ~29.5k lines of real product. The prior scenario held ~3.2k lines of generic campaign CRUD plus RAG generation, no consumers outside its own skill, and a domain model that changes completely — `campaigns/documents/generate` becomes `campaigns/artifacts/claims/posttypes/review/ledger`. That is a rewrite, not a migration. | Fresh generation on a new slug. The old tree was moved out of the repo rather than deleted, and its skill plus five cross-referencing skills still need resolution. | If a capability in the retired scenario is discovered to have been load-bearing. Nothing found so far suggests it was. |
| 2026-07-28 | **The scenario owns the ledger and the gates; it does not generate copy.** | Generation prompts live in the paired `x-<type>` skills, where the doc-plus-skill discipline puts executable procedure. Absorbing them would freeze prompt iteration behind a deploy and collapse the separation between strategic reasoning and procedure. | The desk consumes drafts; it never authors them. Skills stay mutable and independently reviewable. | If skill-authored drafts prove impossible to constrain to the required field set, which would argue for a stricter authoring surface — not for absorbing the prompts. |
| 2026-07-28 | **`claims` answers "is it true"; `review` answers "is it allowed".** | Some assertions are unfalsifiable by design — persona traits and lifestyle implication are permitted marketing embellishment under existing canon. Routing them into a truth gate would make it fail constantly on statements no evidence can settle, and a gate that fails constantly gets disabled. | Two domains with two different mechanisms. Policy rules — credential claims, real-person likeness, fabricated testimonials, missing disclosure — score as review failure modes, never as unverified claims. | Never expected. Merging them is the most likely way this scenario decays. |
| 2026-07-28 | **Claims are a shared library cited by many drafts, not per-draft annotations.** | The same fact is asserted across many posts. Hanging claims off a single draft would mean verifying the same thing repeatedly and would make it impossible to ask which published posts depend on a fact that has since changed. | A many-to-many citation join. Verify once, cite many, re-check on a schedule. This is what enables the contamination report (`CONTENTD-P1-002`). | If claim reuse turns out to be rare in practice, the join is still harmless — but the contamination capability would lose most of its value. |
| 2026-07-28 | **Evidence has two strengths, and re-runnable checks are required where they are cheap.** | A citation proves someone looked; it does not prove the claim is true, and it rots silently because the link still resolves after the fact behind it changed. A stored command with an expected result can be re-run before publish and after. | Quantitative, existence, and status claims require a check. Capability claims accept a named test. Novelty claims require a dated prior-art record and expire. | If check authoring proves burdensome enough that authors avoid the claim kinds that require them, which would show up as claim-kind skew. |
| 2026-07-28 | **Approval is operator-only and is never automated.** | It is the last human check before a public assertion. Every other gate in the scenario is mechanical precisely so that this one stays cheap enough to be real. | The API refuses approval from a non-operator actor. The workbench is designed so approving takes seconds, because operator attention — not agent capacity — becomes the binding constraint once agents are freer. | Never for the approval itself. The *speed* of approval is a UI problem and is expected to need iteration. |
| 2026-07-28 | **Campaign artifact slots are a hard cap, not a target.** | Campaign sprawl is already a named marketing risk, and the existing canon carries "outstanding artifact slots" as decorative checkboxes an operator ticks by hand. As an enforced constraint the same field bounds both sprawl and operator review load. | A draft beyond budget is refused rather than queued. The budget is the only mechanism bounding review load. | If operators routinely hit the cap on work that all genuinely belongs, which would mean the budget is set below real need. |
| 2026-07-28 | **Import idempotency is by content-addressed key, never by offset or watermark.** | Every import source is a file a human or an agent rewrites and reorders. Positional keys break on the first reflow and re-import an entire file as new; watermarks desynchronise on any out-of-order edit. | Re-import is a no-op for unchanged items — that is the diff, with no cursor or state file. **Accepted consequence:** an item edited in place hashes differently and imports as a new record rather than updating the old one, which is correct under an append-oriented ledger. | If edit churn produces visible duplicate pairs that reviewers cannot reconcile. |
| 2026-07-28 | **No credential, in any table, in any form.** | Marketing canon already routes account handles and credentials away from editorial surfaces, and the scheduler owns identity with vault references. | Account eligibility is a single question answered elsewhere: *is this account eligible for this lane*, plus a reason. A schema change introducing a token column is a defect, not a feature. | Never. |
| 2026-07-28 | **The team restructure this scenario enables is explicitly out of scope.** | The scenario makes a smaller marketing roster possible by externalising pipeline state into gates. Rewriting the operating model against a scenario that does not yet run would repeat the error of designing a workbench against an empty publish log. | No member, topic, or operating-model change ships with this scenario. The restructure is a separate, decision-gated change that should follow observed behaviour. | Once the P0 loop has produced published posts and the real handoffs are visible. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
