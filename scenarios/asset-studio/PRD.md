# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Own the production path from a declarative asset specification to a rendered, identity-consistent media artifact with full provenance — as a permanent capability. Characters, scenes, and products become validated, versioned records instead of hand-copied JSON; a render records which identity versions, which backend, which parameters, and what it cost; and an asset cannot be released while a frame fails identity conformance. It is the difference between generating an image and being able to generate the *same* character again in six weeks.
- **Primary users/verticals**: The marketing-crew producer agent commissioning media for an active campaign; the operator, who confirms identity conformance and authorises spend; `content-desk`, which references released assets by identifier; and any future scenario that needs brand-consistent generated media.
- **Deployment surfaces**: CLI (the agent-facing compose, render, and query verbs), API (Connect-RPC), and UI (an identity library, a spec composer, a render queue, and an asset review surface).
- **Value promise**: Makes visual consistency mechanical rather than a matter of prompt luck — the same character, scene, and product render the same way across frames, across artifacts, and across months — while keeping generation spend visible and every rendered artifact traceable to the exact inputs that produced it.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Identity registry | The system shall persist characters, scenes, and products as records carrying a frozen identity block, descriptive traits, and reference image links, and shall validate every record against its kind's declared schema
- [ ] OT-P0-002 | Identity immutability and versioning | The system shall refuse to mutate an identity block that any accepted asset already references, and shall require a new version instead
- [ ] OT-P0-003 | Idempotent canon import | The system shall import character, scene, and product definitions from the marketing rich-media catalogue without duplicating any item on re-run
- [ ] OT-P0-004 | Asset spec composition | The system shall compose an asset spec by binding identity records to a prompt template and shall resolve it into the model-facing payload that a render will submit
- [ ] OT-P0-005 | Spec validation | The system shall refuse to render a spec whose template contract has unfilled required fields, naming each missing field rather than submitting a partial prompt
- [ ] OT-P0-006 | Render job lifecycle | The system shall persist each render as a job with an explicit status and shall reject any transition not declared in the lifecycle contract
- [ ] OT-P0-007 | Render provenance | The system shall record, for every produced artifact, the spec version, each bound identity version, the backend, the model, the seed, and the resolved parameters
- [ ] OT-P0-008 | Generation cost accounting | The system shall record an estimated and an actual cost for every render job and shall report spend by spec, identity, and campaign reference
- [ ] OT-P0-009 | Asset library | The system shall store produced artifacts behind a blob seam with their metadata, derived variants, dimensions, and alt text, and shall serve them by stable identifier
- [ ] OT-P0-010 | Identity conformance gate | The system shall refuse to release an asset while any of its frames carries an unresolved conformance verdict against the identity it claims to depict
- [ ] OT-P0-011 | Operator-confirmed conformance | The system shall accept a conformance verdict only from a human operator identity and shall record which actor judged which frame, against which reference, and when
- [ ] OT-P0-012 | Disclosure metadata at birth | The system shall mark every generatively produced asset as AI-generated and carry its disclosure requirement on the asset record, so that a platform label never depends on a later step remembering
- [ ] OT-P0-013 | Credential-claim guard | The system shall hold an explicit, required-empty credential-claims field on every persona-depicting record and shall refuse release when it is non-empty
- [ ] OT-P0-014 | Asset reference surface | The system shall expose released assets by stable identifier with their metadata and disclosure state, so that a consuming scenario references an asset without copying its bytes
- [ ] OT-P0-015 | Operator workbench | The system shall provide a UI to browse the identity library, compose and validate a spec, watch the render queue, judge conformance, and release an asset, and shall state the specific cause whenever release is unavailable
- [ ] OT-P0-016 | Identity conditioning reference | The system shall let an identity version carry references to the conditioning artifacts that reproduce it — a trained adapter, a reference image set, or a look — and shall record those references in the provenance of every artifact rendered from it
- [ ] OT-P0-017 | Candidate set and selection | The system shall allow one render job to produce several candidate artifacts, shall require an operator to select which candidate proceeds, and shall attribute the job's full cost across the set rather than only to the selected one
- [ ] OT-P0-018 | Identity authoring and revision | The system shall let an operator create and revise an identity record and its conditioning references directly in the workbench, shall validate it against its kind's schema on write, and shall record whether an operator or an agent produced each revision

> **Two of these targets are schema shape rather than behaviour.** `OT-P0-016`
> records conditioning-artifact references but does not render from them
> (`OT-P1-012` does), and `OT-P0-017` establishes job-to-artifact cardinality but
> not a rich selection surface. Both are P0 because they cannot be retrofitted
> into the provenance of artifacts already released — see D-017 and D-018. A plan
> should budget them as columns on existing tables, not as features.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Multi-frame identity binding | The system should hold a spec containing several frames that share bound identity versions, so that a slideshow or a shot sequence is one artifact rather than a set of unrelated renders
- [ ] OT-P1-002 | Video render backend | The system should submit video specs through ai-gateway to a video-capable model and should track their longer-running jobs on the same lifecycle as image renders
- [ ] OT-P1-003 | Screen-capture spec kind | The system should support a capture spec whose source is a scripted browser or desktop session executed through browser-automation-studio, producing an artifact on the same asset model as a generated one
- [ ] OT-P1-004 | Templated compositing | The system should assemble an artifact from ordered segments with declared intro, outro, caption, and lower-third slots, without becoming a general timeline editor
- [ ] OT-P1-005 | Automated conformance scoring | The system should score a rendered frame against its identity's reference images and should present that score to the operator as a recommendation, never as an automatic pass
- [ ] OT-P1-006 | Spend budget and confirmation | The system should carry a spend budget per spec and per campaign reference and should require explicit confirmation before a job that would exceed it
- [ ] OT-P1-007 | Character sheet generation | The system should produce a composite multi-angle reference sheet for a character and should attach it to that identity as its conformance reference
- [ ] OT-P1-008 | Federated retrieval registration | The system should register the identity library and asset metadata as a search-hub provider so they are reachable from federated query
- [ ] OT-P1-009 | Look recipe reuse | The system should reference reusable style recipes held by image-tools rather than re-encoding style parameters per spec
- [ ] OT-P1-010 | Regeneration from provenance | The system should re-render any released artifact from its recorded provenance alone, and should report when a bound identity version or backend is no longer available
- [ ] OT-P1-011 | Agent invocation from the workbench | The system should let the operator commission spec composition or conformance triage through agent-manager from the workbench and should record the provenance of what the agent produced
- [ ] OT-P1-012 | Conditioning-artifact rendering | The system should pass an identity version's conditioning artifacts to the backend through image-tools at render time, so identity retention comes from the artifact rather than from prose description alone
- [ ] OT-P1-013 | Regional refinement | The system should apply a masked regional edit to a produced artifact through image-tools, should record the parent artifact and the operation in the derived artifact's provenance, and shall return the result to conformance rather than inheriting the parent's verdict

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Persona voice | The system may attach persona voiceover produced through audio-tools, bound to the same identity record as the visual depiction
- [ ] OT-P2-002 | Variant generation for testing | The system may derive labelled variants of one spec for outcome comparison once publish telemetry exists to compare them against
- [ ] OT-P2-003 | Drift monitoring across generations | The system may detect an identity drifting across successive renders even when each individual frame passes conformance
- [ ] OT-P2-004 | Desktop packaging | The system may ship as a Tier 2 desktop application through scenario-to-desktop, which is the external delivery target rather than hosted deployment

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go API with Connect-RPC proto contracts, Go CLI over cli-core primitives, React + TypeScript + Vite UI — the standard react-vite scenario shape. The render job is modelled as a Level 5 formal flow through flow-verifier, because a long-running external job with retries, partial failure, and cost is exactly the shape that decays into untested branches otherwise.
- Data + storage expectations: SQLite in-process for identities, specs, render jobs, provenance, conformance verdicts, and asset metadata. **Unlike `content-desk`, this scenario stores bytes** — produced artifacts live behind the template's BlobStore seam with the filesystem implementation, and only metadata sits in proto payloads.
- Integration strategy: ai-gateway for every inference call, image and video alike, so model policy and routing stay where they are owned; image-tools for deterministic operations, look recipes, and image analysis; browser-automation-studio as the executor for capture specs at P1; search-hub for federated retrieval; vrooli-events receipts arrive automatically through api-core. Third-party packages are governed exclusively through Scenario Dependency Analyzer.
- Non-goals / guardrails: This scenario is not a timeline video editor — compositing is templated slots, and a request that needs frame-level editing is out of scope by design. It does not draft copy, does not publish, does not own account identity or credentials, does not decide persona strategy or the AI-UGC stance, and does not drive a browser itself. It never calls a model vendor directly.

## 🤝 Dependencies & Launch Plan

- Required resources: SQLite in-process, and filesystem storage for produced artifacts. No external storage resource at P0.
- Scenario dependencies: ai-gateway (all image and video inference, required); image-tools (deterministic operations, look recipes, analysis; required at P0 for derived variants); browser-automation-studio (capture execution, P1); content-desk (consumer — it references released assets, and this scenario does not depend on it); search-hub (federated retrieval, P1); audio-tools (persona voice, P2); vrooli-events (run correlation, automatic).
- Operational risks: Identity conformance is the central promise and it is unvalidated — nobody has yet rendered the same character twice through this pipeline, so reference-image comparison may prove too weak to catch the drift that matters or too strict to pass anything, and at P0 the operator may have no reference material stronger than prose (D-020); generation spend is unbounded in a way editorial work is not, and a mis-specified multi-frame video spec can cost real money before any human sees a frame, which is why cost accounting is P0 rather than an operational nicety; **the marketing rich-media catalogue is empty** — it holds `_template.json` skeletons and zero authored characters, scenes, or products — so import has nothing to import and the first identity must be authored rather than migrated (D-021); video model interfaces are moving quickly and a payload that is compatible today may not be in six months, which is the reason every call routes through ai-gateway rather than a vendor SDK; and no persona exists in canon and no persona account exists on any platform, so the artifacts this scenario produces have nowhere to be published until both a persona and a `channel-strategy-update` decision exist.
- Launch sequencing: One image post type activated end to end first — a single-identity still image, composed from a **product** identity authored directly in the workbench, rendered through ai-gateway, conformance-confirmed, released, and referenced by a `content-desk` draft. A product subject is chosen over a persona deliberately (D-021): the gate mechanics are identical, and a product depends on no canon decision, no AI-UGC disclosure protocol, and no persona account, all of which sit outside this scenario and are blocked while the marketing team is paused. That narrow slice proves the whole spine and is deliberately chosen over building the general pipeline, because a render pipeline with no released artifact cannot validate its own conformance model. Then identity registry hardening. Then canon import, which is a migration path rather than the day-one path and is therefore sequenced after authoring. Then provenance and cost. Then character conformance, once a persona exists in canon. Then multi-frame binding. Then video. Then capture and compositing. Persona accounts, the disclosure protocol, and the paired post-type skills that make image and video types active are outside this scenario and are the real gate on its output being publishable.

## 🎨 UX & Branding

- Look & feel: Vrooli Operational Console per root DESIGN.md — calm, dense, technical, slate neutrals with blue for primary commands and cyan for technical emphasis; light, dark, and system modes are first-class. The signature surface is a **two-pane identity-first workbench**: a library of characters, scenes, and products on the left, and on the right the spec composer with a live resolved-prompt preview. The second surface is a render queue with cost and state per job. The third is a conformance review that puts the rendered frame beside its reference sheet, because that comparison is the judgement the operator is actually making.
- Accessibility: WCAG AA contrast in both themes, visible focus states, 44px touch targets, no status conveyed by colour alone, reduced motion respected, and full keyboard reachability for library browse, spec composition, queue inspection, conformance judgement, and release. Every asset carries alt text as a required field rather than an optional one. On mobile the panes become stepwise panels with preserved context rather than a shrunken workbench.
- Voice & messaging: Precise and operational. A disabled release control always states its reason inline — an unresolved conformance verdict, a non-empty credential-claims field, a missing disclosure flag — because a dead control with no explanation is the failure mode this surface exists to prevent. Cost is always shown before a job is submitted, never only after. A generated frame is always labelled as generated, in the tool as well as on the platform.
- Branding hooks: Inherits the vrooli-default design kit; replace the generic PWA icons when product branding exists. Keep the seeded web app manifest, service worker, maskable icons, relative install asset URLs, and safe-area tokens valid.
