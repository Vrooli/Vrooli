# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario device-studio`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Device Studio adds a permanent, local-first capability for creating reusable product-presentation shots from authentic screenshots/recordings, qualified device and scene assets, motion and reference-based AI creation. It selects pinned photographic composition for eligible fixed views and actual 3D rendering for physical movement, keeping real app media separate from generated scenery and simulated system chrome.
- **Primary users / verticals**: Product builders, designers, marketers and their agents producing launch videos, app demonstrations and reusable device/scene libraries.
- **Deployment surfaces**: Browser workbench, typed Connect API, discoverable CLI and downloadable private desktop architecture for Windows/macOS/Linux. The current host is the first runtime qualification target; every advertised packaged platform needs its own native release evidence.
- **Value promise**: Save, reopen, edit and render an inspectable shot without repeated ad hoc agent scripts; retain source, rights, physical/craft review, execution and funding evidence.
- **Commitment and current state**: All P0 targets define the accepted first release, not just its early photographic slice. Eight representative groups, phone rotation/laptop lid motion, advanced controls and AI creation remain committed. Implementation and scoped current-host evidence are tracked in README.md and docs/internal/goal/QUEUE.md. Each target earns completion through its requirement protocols; scaffold examples and unrelated historical experiments do not count as completion. P1/P2 are explicit later roadmap.
- **First-release representative groups**: Slab phone, laptop, tablet, foldable/dual-screen, round watch, monitor/TV, smart display/digital frame and embedded vehicle host; supported variants, mounts, views and motions are explicit rather than inferred from the family.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Immutable domain and shot contracts | The system shall compile versioned device, scene, presentation, motion and source revisions into inspectable immutable shots.
- [ ] OT-P0-002 | Qualified representative library | The system shall provide reviewed assets and explicitly supported poses for all eight accepted device groups.
- [ ] OT-P0-003 | Photographic device composition | When a fixed-view photographic shot is selected, the system shall map authentic screen media into its pinned plate and reject unsupported physical motion.
- [ ] OT-P0-004 | Controlled physical device motion | The system shall render genuine phone rotation, laptop opening and closure, and continuous camera entry through qualified physical geometry.
- [ ] OT-P0-005 | Guided and advanced shot editing | The UI shall provide a guided shot builder and accessible advanced camera, lighting, placement, calibration and keyframe controls.
- [ ] OT-P0-006 | Authentic screen presentation | The compositor shall preserve source identity and timing within explicit viewport, mask, chrome, crop and audio policies.
- [ ] OT-P0-007 | Reference-based AI creation and refinement | When requested, the system shall autonomously search, import, construct, compare and refine bounded candidates from references before review.
- [ ] OT-P0-008 | Rights and scoped visual qualification | The system shall separate rights, physical correctness, visual acceptance and generic publication and bind each verdict to immutable artifacts and use scope.
- [ ] OT-P0-009 | Durable jobs and reproducible recovery | The system shall persist owned attempts, checkpoints, receipts and dependency-valid caches and recover or cancel without losing truthful state.
- [ ] OT-P0-010 | Disclosure, privacy and total budgets | Before AI or agent work, the system shall disclose its scope and enforce aggregate budgets, media-egress policy and bounded untrusted execution.
- [ ] OT-P0-011 | Local use and normal Vrooli commerce | The system shall preserve personal-local and BYOK use and integrate optional scoped-account Vrooli-funded operations and paid delivery through their owners.
- [ ] OT-P0-012 | Cross-platform desktop architecture | The system shall support private desktop packaging and platform adapters without requiring the development checkout or full Vrooli fleet.
- [ ] OT-P0-013 | Public operations and launch-video integration | The system shall expose discoverable API and CLI operations that let launch-video save, reopen and render shots and consume standalone artifacts.
- [ ] OT-P0-014 | Qualified output and renderer profiles | The system shall qualify 1080p SDR video at 24 and 30 fps in landscape and portrait and high-resolution stills on explicitly supported workers.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Variant comparison and bounded batches | When several saved variants are selected, the workspace should compare their final looks and render a bounded batch with separate receipts.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Advanced video output profiles | Where additional profiles are introduced, the exporter may expose 4K, higher frame rates or HDR only after separate qualification.
- [ ] OT-P2-002 | Optical and virtual presentation targets | Where optical adapters are introduced, the compiler may support projected or per-eye targets through separately qualified receiver/viewer contracts.

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**: Governed Go/React/Connect scenario structure with domain-owned business logic and UI/CLI adapters; a dedicated browser stage evaluates the same compiled pose and source clocks as final renderers with visibly approximate shading.
- **Data + storage expectations**: Engine-independent per-domain repositories for projects, devices, scenes, presentations, motions, shots, jobs, creations and reviews; embedded SQLite for private mutable state and shared api-core storage/blob primitives for immutable source/derived bytes. Definitions and shot recipes are backend-neutral. Native Blender packages preserve editing/render detail; portable GLBs have separate target qualification and may embed only a representative static screen.
- **Rendering direction**: Deterministic photographic compositor for fixed-view plate replay; owner-managed Blender/native workers for genuine parallax, phone rotation, rigged lid movement and exact geometry. Pinned FFmpeg and FFprobe inspect/decode/encode the declared SDR output profiles. GPU use requires backend/kernel/device/capacity and actual-render qualification; CPU fallback is explicit, never a silent quality or funding change.
- **Integration strategy**: Reuse BAS capture, Image Tools/AI Gateway operations and provider receipts, Asset Studio generic publication/byte identity, Backdrop ambient assets, canonical execution/containment and tool provisioning owners, RCL shared UI and LPBS identity/commerce adapters. LPBS remains authoritative for funded reserve/settlement; metered text routing does not certify funded media. Offer Desk owns catalog/price declarations and Money Ledger owns financial actuals.
- **Non-goals / guardrails**: No competing generic asset, account, wallet, provider-router, host-installer or full-film authority; no general mesh/material graph editor. Native mesh work is import/export or bounded AI-assisted construction. No automatic paid fallback, purchase-required asset sourcing or fabricated native-device/backend evidence. Photographic hardware is approximate when generated and cannot inherit measured 3D fidelity. 4K/HDR and optical targets are later adapters, not first-release claims.

## 🤝 Dependencies & Launch Plan

- **Required local resources**: Private project/job storage and a packaged deterministic composition/media-tool path; no shared database or full scenario fleet is required for personal-local core use. A qualified native renderer pack is required only for advertised local 3D operations. Optional service absence produces a truthful unavailable state.
- **Scenario dependencies**: BAS is an optional capture source because importing existing media is supported. Image Tools/AI Gateway, Agent Manager/execution owners, canonical identity/credential authority, LPBS, Asset Studio and Backdrop are composed at their owning boundaries for selected capture, generation, agent creation, commercial service or shared publication operations. These are target adapters, not claims that missing upstream contracts are already usable.
- **Operational risks**: Native GPU packaging/kernel availability; photographic-quality 3D lighting/materials; mixed lid/base source geometry; exact display topology/clock mapping; actual free-asset dependency rights; source resolution/readability; scoped desktop lease-to-paid-operation gap; unknown media costs and hard aggregate budget enforcement; untrusted scripts/imports; process isolation; platform signing and native install/update evidence.
- **Launch sequencing**: Establish immutable contracts and qualification gates, prove owner-managed worker and scoped commercial operations, and art-direct short phone/lid motion early. Deliver imported/pinned photographic saved-shot replay as the first useful slice. Complete native motion/advanced editing, all eight reviewed groups and autonomous reference-based creation, then qualify local/BYOK/funded integration and per-platform desktop delivery. Exercise actual launch-video consumption through the public contract. A first slice does not close the full P0 scope.
- **Validation boundary**: Requirement and operational-target statuses follow earned, requirement-linked validation through the owning evidence sync. Pending or partial protocols remain open. Follow docs/TESTING.md and Test Genie; structural contract validation is not runtime, craft, paid-flow or release certification. No configured signing identity, paid-call credential or live spending cap is presumed.

## 🎨 UX & Branding

- **Look and feel**: A restrained professional shot workbench with a persistent large preview, clear source/device/scene/presentation/motion steps and progressive disclosure for advanced controls. Adapt the canonical design kit and RCL shell; do not redesign shared chrome or force users into a general-purpose 3D editor.
- **Accessibility**: Preserve canonical WCAG/accessibility floors, visible labels and keyboard/numeric alternatives for camera, placement, calibration and timelines. Status is communicated in text as well as color. Responsive review works at narrow widths without demanding precise mobile 3D dragging.
- **Voice and messaging**: Plain actions and specific incompatibility reasons; distinguish measured facts, estimates, unknowns, generated appearance, physical qualification and human craft acceptance. Show loading, empty, partial, stale, error, cancelled and recovery states without pretending missing provider/account data is zero.
- **AI and agent disclosure**: Identify AI calls and agent launches visibly before initiation, including funding source, total nested budget and outgoing data. Authorized bounded iterations do not require repetitive prompts; scope/funding/egress expansion requires a revised explicit choice. Deterministic editing/rendering never starts hidden AI or agents.
- **Local and account behavior**: Local project work does not require website sign-in. Optional scoped account linking enables qualified Vrooli-funded services; official paid desktop acquisition is a separate entitlement. BYOK keys use the credential owner. Project media stays local by default; external generation/critique/agent vision needs independent per-project opt-in. Direct PNG/MP4/model delivery remains usable without HTML-gallery navigation.
- **Branding hooks**: Use Device Studio as the working product name. Keep seeded PWA/icon/manifest and shared theme/i18n contracts valid; final brand assets go through brand-manager rather than being declared complete by this charter.

## 📎 Appendix

The accepted product decisions, research snapshots and preserved original archive are indexed in [docs/research/README.md](docs/research/README.md), including the v8 decisions and broad device-domain sources. Durable scenario contracts and consumer flows are maintained in docs/. requirements/verification-protocols.md records per-requirement procedures and executed scope; docs/internal/ACCEPTANCE.md owns cross-cutting final craft/release criteria. P0 delivery_scope remains committed. The requirement registry, manual-attestation ledger and owner-computed checkboxes distinguish earned evidence from pending verification. Native/portable outputs and source fixtures establish only their exact retained input, profile and review scope.
