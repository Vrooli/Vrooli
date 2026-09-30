# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

Purpose: Music Library is a personal music player whose recommendation engine is **legible and steerable** rather than secret. It decomposes every track you play into the attributes that make it what it is, learns what you actually like from those attributes, shows you its own model of your taste, lets you edit it directly, and generates new music toward it at zero marginal cost. Real and generated tracks live in one library and are ranked on identical footing. Primary user: a single listener on their own hardware and their own files. Deployment surfaces: UI first (this is a listening product), plus API and CLI for automation and evidence.

Value promise: commercial services optimise a hidden model for aggregate engagement, will not tell you what it believes about you, and cannot follow you quickly into a new interest. Every attribute here is computed locally, every recommendation explains itself, exploration is a control you own, and the candidate pool grows without limit because generation costs nothing. This is the first scenario in the **Lifestyle bundle**.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Trustworthy playback over your own files | The library points at folders the listener manages, never moves or mutates them, and delivers reliable playback including formats the browser cannot decode natively.
- [ ] OT-P0-002 | Uniform decomposition of the library | Every track is decomposed through Music Tools into embeddings, structure, tempo, key, loudness, and tags, with progress and resumability across a library-scale first pass.
- [ ] OT-P0-003 | A taste profile you can read and edit | The learned preference model is a visible surface: what it believes, how confident it is, and which attributes drive it — with every attribute pinnable, boostable, or bannable.
- [ ] OT-P0-004 | Truthful explanation for every recommendation | Each recommendation reports the actual contributions that produced it, distinguishing predicted preference from uncertainty-driven exploration; explanations are derived from the ranking computation, never generated after the fact.
- [ ] OT-P0-005 | Preference elicitation that respects effort | Pairwise comparison with informative pair selection, so few judgements move the model a lot; sparse comparisons propagate to a dense preference function that can score never-heard and just-generated tracks.
- [ ] OT-P0-006 | Generated tracks are always disclosed | Any generated track is identifiable as generated wherever it appears, carrying its model, licence lane, prompt, seed, and lineage.
- [ ] OT-P0-007 | Recommendation blindness | The component producing recommendations cannot observe what is sold or monetised; any offer insertion is strictly post-processing over rankings already made.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Exploration as an owned control | A dial governs the exploitation/exploration balance, with its current effect visible in what is queued.
- [ ] OT-P1-002 | Context-aware taste | Distinct listening contexts carry distinct profiles, so one taste does not average away another.
- [ ] OT-P1-003 | Generation toward the profile | A background queue produces candidates conditioned on the current profile, which re-enter the same index and compete on equal footing.
- [ ] OT-P1-004 | Degeneracy resistance | Calibration against the listener's own distribution plus continuous exploration and unbounded candidate growth keep the system from collapsing onto one sound.
- [ ] OT-P1-005 | Behavioural signal with reliability gating | Completion, replay, and skip inform the model; section-level attribution of skips is treated as a hypothesis requiring repetition across plays and correction for position bias before it carries weight.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Mobile and offline listening | Packaged mobile delivery with offline sync of a bounded working set.
- [ ] OT-P2-002 | Personal adapter | Fine-tuning generation on a confirmed-favourites corpus, gated on hardware with the headroom to train.

## 🧱 Tech Direction Snapshot

Preferred: Go owns the library index, preference model, ranking, and playback session state; protobuf defines transport; React owns presentation. All audio understanding is delegated to `music-tools` — this scenario computes no embeddings itself. Vector search runs on `qdrant`; relational state on SQLite by default. The preference model is a Bayesian formulation over frozen embeddings that returns both an expected preference and an uncertainty at every point, because uncertainty is what drives informative pair selection, honest exploration, and truthful explanation. **The library is pointed at, never imported**: source files are read-only and never moved, while generated audio and derived artifacts live in managed storage under a budget. Non-goals: audio model execution, stem or embedding computation, music distribution or publishing, and any path that lets monetisation state reach the ranking computation.

## 🤝 Dependencies & Launch Plan

Scenario dependencies: `music-tools` (required — all decomposition and generation), `landing-page-business-suite` (optional, for bundle entitlement). Resources: `qdrant` for the embedding index; optionally `postgres` in place of SQLite. Launch sequencing: point at a library and play reliably; run decomposition to completion and make the library searchable by attribute; ship the readable taste profile and elicitation; add ranking with truthful explanation; then the exploration dial; then the generation loop. Operational risks: generated music must survive repeat listening, which is an open empirical question and is deliberately tested before the generation loop is built; first-pass decomposition of a large library is a multi-hour job that must be resumable; and single-user feedback is inherently sparse, which is why elicitation efficiency is a P0 rather than a refinement.

## 🎨 UX & Branding

Look and feel: a listening application first — calm, image-forward, and comfortable at length — with the analytical surfaces available on demand rather than imposed on playback. The explanation surface is the product's signature and must feel like insight, not telemetry. Accessibility: playback and queue control must be fully keyboard-operable; every state that is conveyed by waveform, colour, or audio needs a text equivalent; the generated-content disclosure must be present in the accessibility tree, never colour or icon alone. Voice: candid and specific — it says what it believes about your taste and how sure it is, and admits when it is guessing.

## 📎 Appendix

The taste-engine design, its rejected alternatives, and the degeneracy literature behind the exploration requirements are recorded in `docs/concepts/ARCHITECTURE.md` and `docs/internal/DECISIONS.md`.
