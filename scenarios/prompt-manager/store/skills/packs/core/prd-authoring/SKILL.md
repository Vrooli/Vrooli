---
name: "prd-authoring"
description: "Judgment guide for authoring a scenario's business contract (PRD.md + requirements/) through the business-health wizard — the deterministic, no-AI interview that renders a canonical PRD and requirements skeleton conformant by construction. Covers when to author vs preserve, the interactive-TTY vs answers-file wizard flow (resumable, diff-preview first), taking capability-dedup hints seriously (answer-space cell #34), what the validator enforces (exact emoji section headings, OT line format `- [ ] OT-P0-001 | Title | description`, one target per tier, Purpose/Preferred content anchors), requirements linkage rules (every requirement carries a prd_ref to a real operational target plus at least one validation entry; statuses are earned by sync, never asserted), the finding-code remediation docs (docs/findings/<code>.md) and the deterministic business-health fix loop, and anti-gaming bans (no fabricated validation refs, no hand-flipped statuses, don't delete P0 requirements to silence findings)."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["prd","requirements","business-health","wizard","operational-targets","contract","authoring","capability-dedup","canonical-prd-template","validation-refs","anti-gaming"]
  topics: ["prd authoring","write a prd","business contract","requirements bootstrap","prd overhaul","operational targets"]
  icon: "file-text"
  status: "active"
  defaultScope: "architecture-scope"
  revision: 2
  createdAt: "2026-07-02T00:00:00Z"
  updatedAt: "2026-07-21T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "test-genie", "vrooli"]
    commands: ["prompt-manager", "test-genie fix", "test-genie run", "vrooli scenario"]
  origin:
    kind: "authored"
---
# PRD Authoring

## Purpose

Judgment guide for authoring a scenario's **business contract** — its `PRD.md` and `requirements/` registry — through the **business-health** scenario, the fleet-wide owner of the PRD/requirements contract. Use this skill when you are deciding *what a scenario promises and how you'd know*, and you want that intent to be conformant, linked, and honest from the start.

business-health never writes prose with AI. It renders structure and validates linkage; the product judgment is yours. This skill is about supplying good judgment to a deterministic engine — not about the mechanics of any one CLI command (those live in `scenarios/business-health/docs/reference/cli-commands.md`).

## When To Use

- **New scenario contract** — a scenario has code (or a plan) but no PRD, or a starter-template PRD that says nothing real.
- **PRD overhaul** — product scope has genuinely changed (new capability, retired capability, repositioned users) and the existing contract now lies.
- **Requirements bootstrap** — a PRD exists but `requirements/` is missing or a starter skeleton.

Do **not** use this to nudge statuses green or to reshape a PRD to match drifted code without deciding which side is wrong — that is the concern of `requirements-traceability-steer`, and its anti-gaming bans apply here too.

## Required Reading

- `path:scenarios/business-health/docs/reference/canonical-prd-template.md` — the exact PRD shape the validator enforces (sections, emoji headings, operational-target line format).
- `path:scenarios/prompt-manager/store/skills/packs/core/requirements-traceability-steer/SKILL.md` — once a PRD exists, this is how you keep the requirements registry a *true* claim (the two skills are the author-side and the steer-side of the same contract).

## The Wizard Flow

The wizard is a deterministic interview: it asks the questions the canonical template needs answered, then renders a PRD + requirements skeleton that validates clean **by construction**. There is no AI-generation path — the wizard was deliberately built without one so judgment stays with the calling agent.

```
Fresh contract (no baseline to preserve)?
  → Yes → drive the business-health wizard
  → No, a baseline exists → copy it in, then validate + fix (see scenario-generation)
```

Two ways to drive it:

- **Interactive TTY** — `business-health wizard start <scenario> --interactive`. Best when you are exploring the product's shape as you answer.
- **Answers file** — author a JSON answers file, then `business-health wizard start <scenario>`, `business-health wizard answer <scenario> --answers <file>`, `business-health wizard preview <scenario>`, `business-health wizard apply <scenario>`. Best when you already know the intent (e.g. synthesized from a plan, workshop rounds, or `enhance/`/`archive/` materials) and want a reviewable, re-runnable authoring step.

Properties to rely on:

- **Resumable** — sessions persist; you can answer in passes and come back. Nothing is written to the scenario tree until `apply`.
- **Diff-preview first** — `preview` shows exactly what `apply` will write. Read it before applying.
- **Capability-dedup hints** — when the wizard surfaces *"a similar capability already exists in scenario X"*, take it seriously (this is answer-space cell #34). Overlapping capabilities are how the fleet accretes duplicate scenarios. Resolve the overlap — reuse, extend, or consciously differentiate the existing capability — **before** applying, rather than minting a near-duplicate.

After `apply`, confirm with `vrooli scenario requirements validate <scenario> --json`.

## What The Validator Enforces

Author to these so the contract validates clean; the wizard produces them for you, but you own them once they exist:

- **Canonical sections with exact emoji headings** — the PRD must carry the sections from `canonical-prd-template.md` verbatim (`## 🎯 Overview`, `## 🎯 Operational Targets` with `### 🔴 P0` / `### 🟠 P1` / `### 🟢 P2`, `## 🧱 Tech Direction Snapshot`, `## 🤝 Dependencies & Launch Plan`, `## 🎨 UX & Branding`). The parser is deterministic; a reworded heading is a finding.
- **Operational-target line format** — each target is a single checklist line: `- [ ] OT-P0-001 | Title | one-line description`. IDs are stable strings; keep title + description to one line; do not embed requirement IDs or implementation notes.
- **At least one target per tier** — every priority subsection must exist (P0/P1/P2), and the contract expects real targets, not empty lists left as scaffold.
- **Required content anchors** — the Overview must actually answer **Purpose** (the permanent capability), users/surfaces; Tech Direction must state a **Preferred** stack/approach. Empty or placeholder anchors read as starter-template findings.

## Writing Standard (EARS one-liners + RFC 2119 tiers)

The validator enforces the *shape* of an operational-target line; you own the
*wording*. Two conventions make targets unambiguous:

- **EARS-shaped descriptions.** Write each target's one-line description as an
  outcome claim in EARS form where it fits on one line: "When ⟨trigger⟩, the
  ⟨system⟩ shall ⟨response⟩" (or the ubiquitous form "The ⟨system⟩ shall
  ⟨response⟩"). A target whose description names an observable response is
  directly coverable by a requirement; "support X" or "improve Y" is not.
- **RFC 2119 keywords match the tier.** The priority tiers carry obligation
  strength: P0 targets are **MUST/shall** claims, P1 are **SHOULD**, P2 are
  **MAY**. Use those keywords with their RFC 2119 meanings and do not use them
  loosely elsewhere in the PRD — a P2 written as "must" (or a P0 as "could")
  misstates the contract.

This is wording guidance inside the existing line format — it changes nothing
about IDs, headings, or the parser.

## Requirements Linkage Rules

The PRD is the *what*; `requirements/` is *how you'd know*. The link between them is the whole point:

- **Every requirement carries a `prd_ref` to a real operational target.** A `prd_ref` that matches no `OT-` line in PRD.md is a dangling claim (`business_orphaned_ref` / `intent.prd_ref_unmatched`). Every P0/P1 target should have at least one requirement covering it.
- **Every requirement carries at least one validation entry.** A requirement with an empty `validation[]` is an unmeasurable claim (`business_req_no_validation`). For genuinely manual requirements, use a `manual` validation type with logged evidence — not an empty list.
- **Statuses are earned by sync, never asserted.** A requirement's `complete` status comes from `[REQ:ID]`-tagged tests passing on a comprehensive test-genie run, not from you typing `"status": "complete"`. Hand-set statuses are exactly what `business_status_unearned` catches.

## Remediation: The Deterministic Fix Loop

Every finding code is addressable. When `validate` reports findings:

1. Read the finding's remediation doc: `scenarios/business-health/docs/findings/<code>.md` (e.g. `prd_template_sections.md`, `business_req_no_validation.md`, `intent.prd_ref_unmatched.md`).
2. For `fix_class: auto` findings, run the deterministic fixer: `business-health fix preview <scenario>` to see the diff, then `business-health fix apply <scenario>` to write it (scope with `--rules <code>,<code>`; also reachable via `test-genie fix <scenario> --deterministic`). The auto fixers handle template-section scaffolding, registry creation, status normalization, and `prd_ref` stubs for orphaned targets.
3. For findings that require judgment (a missing capability, a real coverage gap), the fixer will not silently invent content — you author the answer and re-drive the wizard or hand-link the requirement.

## Anti-Gaming Rules

Making the contract *green* is not the goal; making it *true* is. Never:

- **Fabricate validation entries** — a `validation[].ref` must point at a test/evidence that actually exists and actually exercises the requirement. A ref to a non-existent or unrelated test is a lie the sync will eventually expose.
- **Hand-flip statuses** — never set a requirement to `complete` (or flip a PRD `[ ]` → `[x]`) without a passing validation. Statuses are earned from evidence, full stop.
- **Delete P0 requirements to silence findings** — dropping a must-ship requirement because it has no coverage removes the claim, not the gap. If a P0 is genuinely unmet, leave it visible (`not_implemented` or an open finding) and record the gap in `docs/internal/PROBLEMS.md`.
- **Reword canonical headings or fabricate targets** to dodge the parser — the section shape is the shared contract every scenario is measured against.

## Anti-Patterns

- **Don't** hand-write PRD.md from nothing — the wizard is conformant by construction; hand-authoring re-invents the parser's expectations and drifts.
- **Don't** skip the dedup hint — a "similar capability in X" hint is the cheapest moment to avoid a duplicate scenario.
- **Don't** apply the wizard without reading the `preview` diff.
- **Don't** treat `validate` passing as done when the business phase still reports drift — validate is structural; linkage/evidence honesty is the rest of the job (`requirements-traceability-steer`).
