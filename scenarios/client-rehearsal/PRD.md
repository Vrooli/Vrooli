# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario client-rehearsal`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

Purpose: Client Workflow Rehearsal is a permanent Vrooli capability for importing client workflow exports, replacing every imported credential with isolated synthetic stand-ins, rehearsing the workflow against reusable simulated services, injecting controlled faults, and producing an evidence-backed repair and client handoff. Primary users / verticals: freelance automation engineers and operators who need to validate, repair, and explain client automations before delivery. Deployment surfaces: operator UI, stable API, and CLI; engine runtimes and companion simulators are managed behind typed adapters. Value promise: move from fragile one-off mocks and manual demos to repeatable, safe, inspectable rehearsals that work across unlike workflows without handwritten per-workflow stubs.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Safe synthetic rehearsal | Import representative workflow exports, replace every client credential and sensitive value with scoped synthetic stand-ins, execute through an engine adapter against simulated services with no real-client data or credential use, and retain an auditable run record.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Generic coverage and repair | Rehearse thirty unlike public community workflows spanning webhook, schedule, polling, sub-workflow, error-workflow, large-payload, and community-node cases; repair ten seeded defects and retain evidence and a handoff for each.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Reusable engine and simulator ecosystem | Extend the same engagement, credential-substitution, simulator, fault, scoring, and handoff contracts across additional execution engines and client-shaped workflows without changing domain semantics.

## 🧱 Tech Direction Snapshot

Preferred stacks / frameworks: Go API and CLI, React + TypeScript operator UI, proto-backed Connect contracts, and an engine-adapter boundary. Data + storage expectations: SQLite for scenario-owned engagement, workflow revision, credential binding, simulator profile, run, fault, repair, and handoff records; workflow sources and traces retain provenance and stay synthetic. Integration strategy: managed engine/resource and Workspace Sandbox companion-service owners; simulator behavior derived from OpenAPI descriptions or recorded HTTP exchanges; stable API and CLI for reuse. Non-goals / guardrails: never retain real client secrets or messages; never contact real integrations during rehearsal; no hand-written per-workflow service stubs; no hosted n8n offering or exposing n8n configuration to clients under the Community license; do not claim unverified network isolation.

## 🤝 Dependencies & Launch Plan

Required resources: no external runtime is assumed for the scenario scaffold; managed n8n and engagement-scoped Workspace Sandbox companion services are separate owner-managed dependencies that remain undeclared here until their contracts and isolation are verified. Scenario dependencies: none for the first synthetic vertical slice. Operational risks: workflow formats and community nodes vary; credential replacement must fail closed; simulated APIs must preserve source provenance; engine and sandbox network isolation are currently unverified and no containers may be started or pulled until owner admission and isolation proof exist. Launch sequencing: scenario-specific charter and requirements; typed engagement/workflow/simulation contracts; one synthetic workflow end-to-end with operator review; generic simulator derivation and diverse corpus; seeded repair/handoff cases and client-shaped Gmail/Sheets/CRM showcase; later engine expansion only after owner contracts and license review.

## 🎨 UX & Branding

Look and feel: professional operations console with clear workflow identity, run timeline, fault and repair evidence, and handoff preview; calm neutral surfaces with high-contrast status and no decorative dashboard metrics. Accessibility: keyboard-operable workflows, semantic headings and controls, visible focus, reduced-motion support, WCAG AA contrast, and explicit text alongside status color. Voice and messaging: precise, calm, evidence-led; distinguish simulated behavior, injected faults, and unverified assumptions. Branding hooks: scenario identity and Vrooli design tokens; no client marks or imported assets by default.

## 📎 Appendix

Approved source: Plan Manager plan freelance-f7-client-workflow-rehearsal (cb9276c9-de00-4507-8cd4-6e2841024e9e), execution a2913c5d-bfa5-48a4-8544-7637835d141e. Full program target includes thirty unlike workflows, ten seeded repairs, all named trigger families, and a client-shaped Gmail/Sheets/CRM workflow. Research mission: /home/matthalloran8/.vrooli/research/topics/freelance-workflow-readiness/MISSION.md.
