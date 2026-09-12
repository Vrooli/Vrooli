# Security — Plan Manager

This document records plan-manager's security and privacy posture. Update
it before adding auth, user data, external APIs, payment flows, secrets,
or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- How sensitive is the data plan-manager holds?
- What authentication and authorization apply?
- How are secrets handled?
- What is the threat model, and what gaps remain?

## Data Sensitivity

- plan-manager holds structured plan and phase records, plan→code
  references (file paths and possibly small code snippets), validation and
  staleness results, candidate (unvalidated) findings, and per-plan
  velocity.
- Sensitivity is low: the data describes implementation work, not
  credentials or personal data. Code snippets are intended to be small
  illustrative references, NOT secrets — secrets must never be stored in
  plans.
- Candidate findings are explicitly unvalidated and must be presented as
  such, never as confirmed bugs or facts.

## Auth And Authorization

- v1 has no external network auth surface. plan-manager trusts the local
  Vrooli runtime: callers are local agents/operators and in-ecosystem
  Connect-RPC consumers.
- There is no per-user authorization model in v1 because the runtime is
  single-tenant and local. Authn/authz are deferred until plan-manager is
  exposed beyond the local trust boundary.

## Secrets

- plan-manager stores no credentials and has no secret material of its own
  in v1.
- It must reject/avoid persisting secrets that might appear in code
  snippets or references; plans are not a secret store.
- No external API keys are required because all integrations are local and
  soft.

## Threat Model

- Primary note: plans reference code paths (and may embed small snippets).
  Anyone able to read the shared `~/.vrooli` SQLite store can therefore see
  which files matter and read those snippets. This is low sensitivity by
  design, but it does disclose code structure, so secrets must never be
  written into plans and snippet size should be kept minimal.
- Trust boundary: the local Vrooli runtime is trusted; there is no remote
  attacker surface in v1 because there is no external network auth surface.
- Misrepresentation risk: surfacing candidate findings as if confirmed
  would mislead agents/operators. The contract that candidate findings stay
  labeled unvalidated is a security/correctness control, not just UX.
- Out of scope: plan-manager does not read agent transcripts and does not
  spawn agents, so it is not a vector for transcript exfiltration or agent
  hijacking.

## Security Gaps

- No authentication/authorization layer yet — acceptable only while the
  runtime stays local and single-tenant; must be added before any remote
  exposure.
- No formal secret-scrubbing on ingested code snippets yet; until
  implemented, callers are responsible for not submitting secrets.
- This scenario is pre-implementation, so these controls are documented
  intentions rather than verified behavior.

## Cross-References

- [`PERFORMANCE.md`](PERFORMANCE.md) — performance constraints
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) — operational procedures
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system architecture
