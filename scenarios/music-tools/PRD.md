# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

Purpose: Music Tools is the local-first music capability primitive in Vrooli's `*-tools` family — the audio counterpart to `image-tools`. It owns three operation families over music and sound: **composition** (text/lyrics to song, instrumental, sound effects), **transformation** (stem separation, cover, section repaint, reference mastering), and **analysis** (embeddings, structure and beat segmentation, key and tempo, loudness, auto-tagging, audio-to-MIDI, lyric transcription). It holds no opinion about taste, libraries, or playback; those belong to consuming scenarios. Primary users are consuming scenarios (Music Library first, then Asset Studio, Backdrop Studio, Bedtime Story Generator, Content Desk) and operators driving music operations headlessly. Deployment surfaces are the Go API, the CLI, and a UI that enhances but never gates.

**Composition produces a set of candidate takes, not a track.** This is the scenario's defining contract and it is evidence-driven, not a preference: a 2026-09-18 spike established that no single generation was reliably usable while a batch of ten contained several that were. A commercial stock track is itself selected from a large pool, so a single generation compared against a curated winner is not a fair comparison. The scenario therefore generates surplus, retains every take with its own provenance, and lets the caller select. It does not rank, score, or learn preferences — that is taste, and taste stays with consumers.

Value promise: every music operation runs locally at zero marginal cost, on hardware the operator already owns, with the model's licence and commercial-use lane recorded against every output. Hosted competitors now carry a per-generation royalty; this scenario carries none, and has decided against a hosted fallback rung rather than merely lacking one (`docs/internal/DECISIONS.md`, 2026-09-18).

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Headless capability surface | Every composition, transformation, and analysis operation completes from the CLI with no UI and no ComfyUI dependency, returning report-shaped output.
- [ ] OT-P0-002 | Governed model registry | Every model is declared with hardware gates, disk cost, licence, and commercial-use lane; installs are checksum-verified and refuse to start when free disk is below the declared floor.
- [ ] OT-P0-003 | Arbitrated GPU execution | Every GPU operation claims through the capacity broker, degrades to a declared lower rung under contention, and releases on completion or failure — never evicting a co-resident tenant implicitly.
- [x] OT-P0-004 | Uniform decomposition | Any track, owned or generated, yields the same structured description — embeddings, structure and beats, key and tempo, loudness, tags — through a single contract.
- [ ] OT-P0-005 | Durable async jobs | Long operations are server-owned jobs that survive client disconnect, expose progress, and end in success or an explicit recoverable failure.
- [ ] OT-P0-006 | Bounded derived storage | Stems, frame-level embeddings, and generated audio live under a declared budget sized for discarded takes, with on-demand production and LRU eviction that prefers unselected takes.
- [ ] OT-P0-007 | Style library | Data-defined styles compile to a caption plus parameters, so a house sound is captured once and reused as the unit batches and inventories are configured with.
- [ ] OT-P0-008 | Batch composition and selection | A composition request returns the requested number of independently seeded takes, each separately addressable and retained, with selection recorded as a fact rather than interpreted as preference.
- [ ] OT-P0-009 | Honest provenance on every artifact | Every take records model, licence lane, applied capacity rung, seed, and both the caption as authored and the caption as the model received it.

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | Iterative transformation | Cover, section repaint, stem separation, and vocal-to-accompaniment are first-class operations, so a near-miss is edited rather than regenerated.
- [x] OT-P1-002 | Commercial lane isolation | A build configured for the permissive lane excludes every non-commercial model and still produces a working composition and analysis stack.
- [x] OT-P1-003 | Deterministic delivery ops | Loudness measurement, platform-target normalisation, and reference mastering run without a GPU and without a model download.
- [ ] OT-P1-005 | Maintained inventory | For a named style the scenario keeps a target number of ready takes and replenishes them as they are consumed. Takes are drawn through an explicit reservation lifecycle so no two consumers ship the same track. The replenish trigger is pluggable; replenish-on-consume is the v1 and requires no platform scheduler.

### 🟢 P2 – Future / expansion

- [x] OT-P2-001 | Quality composition tier | A larger generation model is selectable on hardware that can hold it, without changing the operation contract.
- [x] OT-P2-002 | Personal adapter training | Adapter training over a confirmed-favourites corpus, gated on a card with the headroom to run it.

## 🧱 Tech Direction Snapshot

Preferred: Go owns operation orchestration, the model registry, job lifecycle, and capacity claims; protobuf defines transport contracts; Python hosts model code behind process boundaries. Transport follows the split `image-tools` proved: discovery and metadata are Connect-RPC, and any edge carrying audio bytes is a REST multipart endpoint whose parameters stay proto-typed. Composition is a job submit, never a synchronous call — a ten-take batch is minutes of GPU time and cannot ride a request.

**No hosted inference rung.** Composition fails explicitly rather than falling back to a cloud provider. This is a recorded decision with revisit triggers, not an omission: hosted music generation is where per-generation royalties and contested training-data provenance actually live, which is the exposure this scenario exists to avoid. The provider-chain *seam* is adopted so a rung could be added without rework; the rung is not built, and the scenario declares no meters. Two mechanical blockers back this up — the AI Gateway's `RequestKind` enum has no audio member, and no hosted music provider is wired in `resources/`.

Model execution is split across three isolated runtimes because the dependency stacks are mutually incompatible — a native Python environment for the embedding stack inside this scenario, plus two managed-service resources (`ace-step`, `music-mir`) that own their own environments and lifecycles. **No Docker**: resources acquire lockfile-pinned wheels and checksum-verified weights natively, so the desktop delivery tier does not require a container runtime. Storage flows through the shared api-core BlobStore seam. Non-goals: playback, library management, taste modelling, speech (owned by `audio-tools`), and any private GPU-management implementation that bypasses the capacity broker.

## 🤝 Dependencies & Launch Plan

Required resources: `ace-step` (composition and transformation), `music-mir` (structure, beats, separation), `qdrant` (embedding index for consumers), and optionally `ollama` and `openrouter` for caption assistance. Scenario consumers: Music Library first; then Asset Studio, Backdrop Studio, Bedtime Story Generator, Content Desk. Launch sequencing **revised 2026-09-18**: establish the registry and capacity contract, then ship **composition end to end** (style → batch → takes → selection), then analysis, then transformation. The original order put analysis first; the spike inverted the evidence. Composition is the tier that has been measured and proven on the reference host, it is the only tier with a waiting consumer today, and it is the tier whose contract the spike showed we had wrong. Analysis remains higher-value in the long run but has no measured throughput and no consumer that exists yet.

Two host conditions sit outside this scenario. Both were previously recorded as preconditions to resolve before composition code is written; **neither is, and treating them that way would block work that can proceed now** (corrected 2026-09-18 against `image-tools`, which already ships through both).

**The capacity broker does not actuate.** It runs `enforce = advisory` with `preempt_enabled = false`, so a claim that cannot be satisfied returns a queue verdict that never resolves, because nothing reclaims. This is a condition to design for, not a blocker: `image-tools` treats the broker as advisory by construction — a nil broker means no arbitration, any broker error leaves GPU selection untouched, and the broker may degrade a job to CPU but never blocks it (`api/internal/ai/capacity.go`). Every GPU target here (`OT-P0-003`, `MUS-P0-006`) is satisfiable against an advisory broker provided fit is computed against *free* VRAM rather than total, which is the check whose absence caused repeated out-of-memory failures during the spike.

**Nothing in the platform can schedule idle-time work.** There is no cron, timer, or recurring-work facility in the control plane, and no definition of host idleness to act on. This bounds *when* inventory is replenished, not *whether* it can be: a replenish-on-consume trigger needs no scheduler at all, and is the v1 for `OT-P1-005`. Idle-time replenishment is an optimisation over that, and its value is contention avoidance rather than latency — the spike established that competing for VRAM with resident tenants is the real failure mode, not the wait.

Operational risks: a 16 GB card shared with resident tenants means no two heavyweight models co-reside; derived audio outgrows disk faster than any other artifact class, and generating surplus takes multiplies that by roughly the batch size; and the strongest analysis models are non-commercial, so the lane split must hold from the first commit.

## 🎨 UX & Branding

Look and feel: an operations console, not a DAW — dense, keyboard-reachable, honest about queue depth and GPU contention. Because composition returns a set, the composition surface is an **audition-and-select** view: takes from one batch compared against each other, played without leaving the list, kept or discarded, with each take's provenance reachable. Selecting is an explicit act the operator performs, never something the interface does on their behalf by ordering or pre-picking. Accessibility: every operation state must be legible without relying on colour or audio alone; waveform and spectrogram surfaces need text equivalents; long-running work must announce progress to assistive technology. Voice: precise and unglamorous — name the model, the licence lane, the elapsed time, and the reason a job degraded or waited.

## 📎 Appendix

The three-runtime split, residency policy, and licence lanes are in `docs/concepts/ARCHITECTURE.md`. The model stack itself — identifiers, disk cost, VRAM tiers, per-model licences, and the confidence label on every figure — is in `docs/reference/model-registry.md`. The durable choices behind them, and the alternatives rejected, are in `docs/internal/DECISIONS.md`.

Measurement status as of 2026-09-18: **the composition tier has been measured on the reference host and nothing else has.** ACE-Step 1.5 turbo produces 45 s of audio in 17.9 s resident, 55.2 s with the DiT offloaded to CPU — a 3.1× cost that is the first concrete instance of an `OT-P0-003` degradation rung. Analysis, embedding, structure-and-beats, and separation remain entirely unmeasured, and they are the larger share of eventual GPU time. See `docs/internal/PERFORMANCE.md`; do not read one measured tier as a profiled scenario.

The composition recipe is implemented by this scenario's governed styles, compose, jobs, and takes surfaces. The `music-generation` core-pack skill is a pointer to those surfaces and retains only brief-writing judgment.
