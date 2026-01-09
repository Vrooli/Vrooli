# Canonical PRD Template

> **Version**: 2.0.0
> **Last Updated**: 2025-11-xx
> **Status**: Canonical Specification
> **Source of Truth**: PRD Control Tower (API/UI/CLI)

## Why This Exists

Product Requirements Documents in Vrooli are **static operational charters**. They capture the “what” of a scenario or resource, freeze immediately after the initial generation, and act as the anchor for requirements registries and operational targets. Implementation changes flow through `requirements/`, docs/PROGRESS.md, and automated requirement syncing – never by rewriting the PRD narrative itself.

Key principles:
- **Minimal + Machine-Readable** – < 100 lines when possible, simple heading hierarchy, deterministic parsing
- **Static After Publish** – PRDs never get manually edited after generation (checkboxes may flip automatically)
- **Operational Focus** – Describe the capability, outcomes, dependencies, and UX tone; leave technical minutiae elsewhere
- **Traceable Targets** – Operational targets link downstream to requirements by `prd_ref` references

## Required Sections (with Emojis!)

Every PRD MUST follow the exact layout below. Additional sections belong in appendices or supporting docs.

### 1. `# Product Requirements Document (PRD)`
Start the file with an H1 that includes “Product Requirements Document (PRD)”. Optionally add metadata blockquotes the generator supplies (template version, canonical reference, etc.).

### 2. `## 🎯 Overview`
Explain the product at a glance.
- **Purpose** – What permanent capability does this scenario/resource add?
- **Primary users / verticals**
- **Deployment surfaces** – CLI, API, UI, automations, etc.
Use short paragraphs or bullets; keep implementation hints out.

### 3. `## 🎯 Operational Targets`
This section holds every measurable outcome and is the backbone for requirements syncing.

Under it, define three subsections (even if some lists are empty):
- `### 🔴 P0 – Must ship for viability`
- `### 🟠 P1 – Should have post-launch`
- `### 🟢 P2 – Future / expansion`

Each operational target is a **single checklist line** using pipe-delimited shorthand:
```
- [ ] OT-P0-001 | Cross-platform capture flow | Capture, organize, and playback workflows without desktop installs
- [x] OT-P1-002 | Marketing-ready share links | Shareable, branded playback links surfaced in UI
```
Guidelines:
- IDs are stable strings (use generator-provided numbering)
- Keep title + description to one line to preserve readability
- Do **not** embed dependencies, requirement IDs, or implementation notes – those go in requirements/ and docs/PROGRESS.md
- Checkboxes may auto-flip when requirement sync detects completion

### 4. `## 🧱 Tech Direction Snapshot`
High-level architectural intent without implementation detail. Example bullets:
- Preferred UI/API stacks
- Data storage expectations
- Integration strategy (shared workflows vs. direct APIs)
- Non-goals

### 5. `## 🤝 Dependencies & Launch Plan`
Call out required local resources, scenario dependencies, risks, and launch sequencing. Stick to textual bullets (no YAML needed).

### 6. `## 🎨 UX & Branding`
Describe the desired look, feel, tone, accessibility targets, and any brand guardrails. Think “experience promise” rather than screen inventories. Example prompts:
- Visual palette, typography tone, or motion language
- Accessibility commitments (WCAG level, color contrast expectations)
- Voice/personality when presenting content

### Optional Appendix
If more context is vital (references, inspiration links, etc.), append `## 📎 Appendix` after the required sections. The parser ignores the appendix, so format freely.

## Operational Target ↔ Requirements Flow
1. Generator writes the PRD with the structure above.
2. Generator seeds `requirements/index.json` with entries whose `prd_ref` look like `Operational Targets > P0 > OT-P0-001`.
3. Tests tag `[REQ:ID]`; requirement sync updates statuses, and PRD Control Tower surfaces completion state by reading checkboxes.

## Editing Rules
- PRD is read-only once generated. If business intent changes, regenerate a new PRD rather than editing the existing one.
- Auto-updaters may toggle the `[ ]` → `[x]` checkboxes to reflect validated operational targets.
- Narrative changes belong in README.md, docs/PROGRESS.md, docs/PROBLEMS.md, or requirements files.

## Reference Implementations
- Template emitted by `scripts/scenarios/templates/react-vite/PRD.md`
- Enforced by PRD Control Tower validators (Go + UI) and scenario-auditor `prd-structure` rule
- Referenced by `scenarios/ecosystem-manager/prompts/templates/scenario-generator.md`
