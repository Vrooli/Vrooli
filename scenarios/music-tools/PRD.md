# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

Purpose: Music Tools is the local-first music capability primitive in Vrooli's `*-tools` family — the audio counterpart to `image-tools`. It owns three operation families over music and sound: **composition** (text/lyrics to song, instrumental, sound effects), **transformation** (stem separation, cover, section repaint, reference mastering), and **analysis** (embeddings, structure and beat segmentation, key and tempo, loudness, auto-tagging, audio-to-MIDI, lyric transcription). It holds no opinion about taste, libraries, or playback; those belong to consuming scenarios. Primary users are consuming scenarios (Music Library first, then Asset Studio, Backdrop Studio, Bedtime Story Generator, Content Desk) and operators driving music operations headlessly. Deployment surfaces are the Go API, the CLI, and a UI that enhances but never gates.

Value promise: every music operation runs locally at zero marginal cost, on hardware the operator already owns, with the model's licence and commercial-use lane recorded against every output. Hosted competitors now carry a per-generation royalty; this scenario carries none.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Headless capability surface | Every composition, transformation, and analysis operation completes from the CLI with no UI and no ComfyUI dependency, returning report-shaped output.
- [ ] OT-P0-002 | Governed model registry | Every model is declared with hardware gates, disk cost, licence, and commercial-use lane; installs are checksum-verified and refuse to start when free disk is below the declared floor.
- [ ] OT-P0-003 | Arbitrated GPU execution | Every GPU operation claims through the capacity broker, degrades to a declared lower rung under contention, and releases on completion or failure — never evicting a co-resident tenant implicitly.
- [ ] OT-P0-004 | Uniform decomposition | Any track, owned or generated, yields the same structured description — embeddings, structure and beats, key and tempo, loudness, tags — through a single contract.
- [ ] OT-P0-005 | Durable async jobs | Long operations are server-owned jobs that survive client disconnect, expose progress, and end in success or an explicit recoverable failure.
- [ ] OT-P0-006 | Bounded derived storage | Stems, frame-level embeddings, and generated audio live under a declared budget with on-demand production and LRU eviction; the library-wide materialisation path is closed by construction.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Iterative transformation | Cover, section repaint, stem separation, and vocal-to-accompaniment are first-class operations, so a near-miss is edited rather than regenerated.
- [ ] OT-P1-002 | Commercial lane isolation | A build configured for the permissive lane excludes every non-commercial model and still produces a working composition and analysis stack.
- [ ] OT-P1-003 | Deterministic delivery ops | Loudness measurement, platform-target normalisation, and reference mastering run without a GPU and without a model download.
- [ ] OT-P1-004 | Style library | Data-defined styles compile to a caption plus parameters, so a house sound is captured once and reused.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Quality composition tier | A larger generation model is selectable on hardware that can hold it, without changing the operation contract.
- [ ] OT-P2-002 | Personal adapter training | Adapter training over a confirmed-favourites corpus, gated on a card with the headroom to run it.

## 🧱 Tech Direction Snapshot

Preferred: Go owns operation orchestration, the model registry, job lifecycle, and capacity claims; protobuf defines transport contracts; Python hosts model code behind process boundaries. Model execution is split across three isolated runtimes because the dependency stacks are mutually incompatible — a native Python environment for the embedding stack inside this scenario, plus two managed-service resources (`ace-step`, `music-mir`) that own their own environments and lifecycles. **No Docker**: resources acquire lockfile-pinned wheels and checksum-verified weights natively, so the desktop delivery tier does not require a container runtime. Storage flows through the shared api-core BlobStore seam. Non-goals: playback, library management, taste modelling, speech (owned by `audio-tools`), and any private GPU-management implementation that bypasses the capacity broker.

## 🤝 Dependencies & Launch Plan

Required resources: `ace-step` (composition and transformation), `music-mir` (structure, beats, separation), `qdrant` (embedding index for consumers), and optionally `ollama` and `openrouter` for caption assistance. Scenario consumers: Music Library first; then Asset Studio, Backdrop Studio, Bedtime Story Generator, Content Desk. Launch sequencing: establish the registry and capacity contract, ship analysis end to end, then composition, then transformation, then the style library. Operational risks: a 16 GB card shared with resident tenants means no two heavyweight models co-reside; derived audio outgrows disk faster than any other artifact class; and the strongest analysis models are non-commercial, so the lane split must hold from the first commit.

## 🎨 UX & Branding

Look and feel: an operations console, not a DAW — dense, keyboard-reachable, honest about queue depth and GPU contention. Accessibility: every operation state must be legible without relying on colour or audio alone; waveform and spectrogram surfaces need text equivalents; long-running work must announce progress to assistive technology. Voice: precise and unglamorous — name the model, the licence lane, the elapsed time, and the reason a job degraded or waited.

## 📎 Appendix

Model stack, VRAM tiers, the three-runtime split, and licence lanes are recorded in `docs/concepts/ARCHITECTURE.md`; the durable choices behind them are in `docs/internal/DECISIONS.md`.
