# Calibration Review Worksheet

Phase 10 requires a human operator to review 50 imported entries. This file
records the reproducible selection method and the decision fields; it does not
substitute an automated judgement.

## Selection

Run this query against the scenario database. It selects eight entries from
each current facet and two additional `gotcha` entries, ordered by immutable
entry ID (50 entries total).

```sql
WITH latest AS (
  SELECT entry_id, facet_id,
    ROW_NUMBER() OVER (PARTITION BY entry_id ORDER BY assigned_at DESC, id DESC) AS rn
  FROM facet_assignments
), ranked AS (
  SELECT e.id, l.facet_id, e.source_path, e.body,
    ROW_NUMBER() OVER (PARTITION BY l.facet_id ORDER BY e.id) AS facet_rank
  FROM entries e JOIN latest l ON l.entry_id=e.id AND l.rn=1
  WHERE e.source_harness<>''
)
SELECT id, facet_id, source_path, body FROM ranked
WHERE facet_rank<=8 OR (facet_id='gotcha' AND facet_rank BETWEEN 9 AND 10)
ORDER BY facet_id, facet_rank;
```

## Per-entry decision

For each row, record all three fields in the Phase 10 plan log:

1. Entry ID and current facet.
2. `correct`, or the one replacement facet from the closed six-facet set.
3. Whether the operator would nominate it for a pin; nomination is not an
   automatic pin and must remain false unless explicitly approved.

Then record sample size, correct count, every correction, nominated-pin count,
and the operator identity/date. Only then may the taxonomy fit, pin budget, and
review interval be calibrated in `DECISIONS.md` and configuration.

## Completed decisions

### Batch 1 — 2026-07-28

The operator accepted all ten entries except these corrections:

| Entry ID | Previous facet | Operator facet | Rationale |
|---|---|---|---|
| `14e98ce3-496e-4ed5-8b08-89b277a61bab` | entity-record | gotcha | The template's pre-existing test failure is a recurring trap. |
| `17cdea3a-efe5-4abf-911a-9b2568593788` | entity-record | environment-fact | Sandbox-aware CLI behavior is a durable tooling/environment fact. |
| `0eff4fd2-5d88-4407-985c-ea1dd3be752f` | environment-fact | entity-record | The structure-health plan is a named project artifact. |

`14948487-eb38-4197-b977-1f4634b9de28` (host environment) remains
`environment-fact`; the operator noted that another category could be
defensible, but did not request a change. No pin nomination was made in this
batch.

### Batch 2 — 2026-07-29

The operator accepted the reviewer reads. The following entries require a
later append-only facet assignment through the operator surface:

| Entry ID | Previous facet | Operator facet | Rationale |
|---|---|---|---|
| `1be243e9-2cb1-4798-9690-243a077af755` | environment-fact | gotcha | OSC reply loop is a recurring failure trap. |
| `333fd36b-2e49-4ce2-8f49-374d139e84f8` | environment-fact | entity-record | Completed overlay-consolidation project context. |
| `3c1a684e-f9cf-4109-897e-185e91f3e325` | environment-fact | entity-record | Durable-data design and plan artifact. |
| `3cafe948-0db0-4ab7-98aa-c25f2314f3dc` | environment-fact | entity-record | Completed Test Genie project record. |
| `42587320-dc00-48f7-b203-043eaae19c80` | environment-fact | gotcha | Advisory-only UI-health enforcement is a trap. |
| `13e5b243-2534-451e-98ef-5b031f2ca7d7` | episode | thread | System-monitor work remains active. |
| `13eb00c9-fb1e-468c-a5d0-6b917256d70d` | episode | gotcha | Silent-reset triage protocol is reusable failure guidance. |

`3c876b64-b1e4-40a0-b8e6-37ef0a3b51b5` remains `environment-fact` and is
nominated for a pin because the host-audio limitation affects browser audio
testing across unrelated work. The four reviewed episodes remain `episode`.

### Batches 3–5 — delegated operator review, 2026-07-29

The operator explicitly delegated the remaining 30 decisions to the reviewer's
documented reads. These are operator-authorized decisions, not an independent
automated quality measurement. Entries absent from the table retain their
current facet.

| Entry ID | Previous facet | Approved facet | Basis |
|---|---|---|---|
| `260b01ed-a9f4-444a-8274-2a75513294b6` | episode | thread | Image-tools plan remains active. |
| `38f6ad42-c346-4116-8c32-84d138630d72` | episode | standing-rule | Search-context taxonomy guides future work. |
| `01e20b2c-06b4-4b68-9751-4741fa93edbf` | gotcha | entity-record | Completed skill-audit project record. |
| `02fc09c6-ce5c-43b5-9830-76945668e175` | gotcha | entity-record | Legacy `MEMORY.md` artifact record. |
| `03c75b4c-8341-4e08-b67d-0b5187b0f609` | gotcha | entity-record | Completed Tunnel Manager project. |
| `054c0cd4-5a55-44c4-ba85-e5664269788a` | gotcha | standing-rule | CLI manifest source-of-truth guidance. |
| `063c7eff-6ef8-4bb1-86cc-cf9e895e84c1` | gotcha | entity-record | Completed Plan Manager project record. |
| `0abb78be-68eb-4f1e-9810-e06bd90b6137` | gotcha | entity-record | App-issue-tracker replacement record. |
| `0ac2f6f8-a903-4901-82a0-215da8f75fb2` | gotcha | entity-record | Web Console performance project. |
| `0cdb9e37-2de9-494f-8e27-d828202c4a32` | gotcha | thread | Validation-substrate plan remains unstarted. |
| `0d558a88-b74d-4e0b-8233-e9005352b8f3` | gotcha | entity-record | Completed Swarm Manager UI project. |
| `06f8ec5e-2480-4058-9b17-fd19d861678a` | standing-rule | entity-record | Business-health plan artifact. |
| `10fff96c-7e2f-4646-a62f-bf557ad4c157` | standing-rule | entity-record | Completed swarm-goals project. |
| `1c26c81a-42dc-4f2d-9314-5f7125d9cf1a` | standing-rule | entity-record | Stats-repair project record. |
| `2043beb3-71b6-47db-9ecf-b7acb3eb4e1d` | standing-rule | entity-record | Plan Manager defect-map record. |
| `2acfc7a8-4ffc-4168-bde1-02b67b2b95f5` | standing-rule | thread | Connect migration remains in progress. |
| `2c7e6002-8d26-497b-a4dc-ed5f74dd8e3e` | standing-rule | entity-record | Completed typed-doc validation project. |
| `00748fbc-ca6b-427a-9321-24159ee574a0` | thread | entity-record | Observability surface description. |
| `01ad0a48-52ab-4c25-a2d4-9ca05d672995` | thread | entity-record | Completed records-adoption project. |
| `04de7f78-486c-4c1a-be3a-6b242c49dd25` | thread | standing-rule | Safe-restart guidance. |
| `0a3fba52-7e96-4719-8e7a-5a29d9372e29` | thread | episode | Completed Grok runner refactor. |
| `0c43cfeb-2d43-4070-bf15-675d8c24a20c` | thread | gotcha | Audio auto-stop race is a recurring failure. |
| `0e37124a-0ab1-47a8-affa-ddf43368ae61` | thread | episode | Completed macOS compatibility phase. |

The remaining delegated entries retain their assigned facets: GCT backlog audit
and interactive-runner delivery are episodes; prompt-manager plumbing is a
gotcha; orchestration-summary and standards-gate guidance are standing rules;
messages-view and wizard ergonomics remain threads.
