# Experience quality across design, implementation and release

Status: owner-directed target for a disabled delivery effort; implementation is unverified.
Owner direction: the two owner messages retained in [the commission](../internal/experience-quality-delivery/COMMISSION.md).
Delivery home: [Experience Quality Delivery](../internal/experience-quality-delivery/README.md).

## Purpose and stakes

Agents and people can understand, capture, assess, improve and qualify an application's complete experience through one coherent capability. Web Console is the first proving product: paid users must receive a responsive, consistent application, including its secondary surfaces and adverse states. The capability must also serve other scenarios and both new-product design and existing-product improvement.

The owner wants local agents to gain these capabilities while preserving Dot's effective cloud design and research workflows. Drive offers availability when the development machine is down; ChatGPT offers a separate subscription allowance and convenient image/design tools. Local execution offers close access to applications, code, automation and diverse agents. These are complementary advantages. No accepted design requires all work to move to one machine or requires every agent to have a Google connector.

This document owns the cross-scenario destination and integration decisions for this effort. Existing PRDs, requirements, experience contracts, selected product designs and domain APIs retain their authorities. Amend those canonical records through their owners before implementing a changed obligation. This document does not certify their present implementation or select new Web Console art direction.

## The experience we are aiming for

An operator can request “capture Web Console's current experience,” “find interaction and polish issues,” “compare this implementation with the selected design,” or “qualify this candidate for release.” Discovery selects the appropriate scope and existing owner operations. The agent gets a bounded result with a readable summary, a navigable atlas, original evidence, findings and coverage limits. It does not reconstruct a custom browser harness, translate private disk paths between scenarios, or assemble readiness signal files by hand.

The same collection supports several questions. A designer browses original-resolution screens and relevant interaction clips. An engineer retrieves exact reproduction and expected behavior. A release reviewer sees whether required coverage and findings have current evidence for the candidate. A cloud agent can continue research, new design, critique, specification and review of retained evidence while the machine is unavailable. Actual local execution resumes only when a suitable environment is available.

Small fixes remain small. A capture of one dialog need not create a dossier or broad audit. A substantial redesign warrants richer design and decision work. A release requires its declared experience coverage even when a smaller development pass was sufficient earlier.

## Authority and continuity across environments

| Subject | Authoritative home | Other representations |
| --- | --- | --- |
| Reusable general design/dossier judgment | Its existing personal/shared capability owner and adopted revision | Explicit source snapshots for receiving agents; no automatic local fork |
| Vrooli operating procedures, bindings, contracts and release policy | Repository/domain owner, with governed skills/programs and typed services | Drive's Vrooli adapter links to exact relevant sources and capability discovery |
| Product intent, selected design and owner decisions | The product's established project/canonical records | Source imports with version, meaning and adoption status preserved |
| Actual running experience | Attributable runtime captures and observations | Atlas, comparisons and reports derived from those records |
| Release standing and approvals | Deployment Manager and its declared evidence owners | Reviewer-facing projections; no independent release ledger |

Choose authority by subject, not by assuming that a repository or Drive always wins. A local correction does not silently replace a selected Drive design; an old Drive example does not override a current native API. Contradictions get a recorded disposition before dependent implementation.

The Vrooli adapter in Drive should carry concise discovery, source pointers, adoption/freshness rules and fallbacks. Avoid a second hand-maintained copy of operational procedures. Provide access in this order, subject to the actual task's tools and authority:

1. Current local discovery/owner operations and executor-readable source.
2. An authorized GitHub source at a verified published commit or release, for reading methods/specifications when local execution is unavailable.
3. A verified retained package or cached adopted revision, with its date and known differences explicit.
4. Cloud-native design/research and review of existing evidence, with local runtime qualification pending.

GitHub is a source fallback, not a running application. An uncommitted local file has no demonstrated GitHub counterpart. Record repository, path, ref, accessible route and content identity. Private repository access is conditional; do not publish private project material to obtain availability. A pointer is useful only if the receiving agent can actually follow it. Neither unavailable local service nor missing connector proves the capability does not exist.

Cloud design must work without a continuously reachable local machine. Local runtime collection must work without Drive after the selected required materials have been received. Reconcile sources and changes on reconnection without destroying drafts, owner edits or historical evidence. General Drive ingestion/search is owned by the separate external-knowledge-delivery effort and is not a prerequisite for this effort's source handoff.

## Experience model and domain boundaries

Use existing contracts and preserve open-world semantics: an experience claim is a promised outcome, not an exhaustive census of every UI object. A capture inventory states what was requested/discovered and exercised. Link these concepts by stable identity without forcing every observed tooltip into a machine claim.

| Concern | Existing owner to extend/reconcile | Boundary |
| --- | --- | --- |
| Product experience intent, states, claims, bindings, readiness profiles, reconciliations and attestations | Experience Manager; scenario `experience/` | Own declared experience and evidence relationships; establish the missing collection/read model here if investigation confirms the fit |
| Browser execution, state entry actions, screenshots, accessibility/computed structure, raw recordings, durable artifacts | Browser Automation Studio | One browser/capture engine; verify output, state reached and execution identity |
| BAS workflow definitions and their coverage/governance | Workflow Health | Use existing workflow substrate for reproducible transitions and assertions |
| Standards, geometry, shell/chrome, accessibility and runtime health | UI Health | Reuse its verdicts and observation evidence; avoid reimplementing its checks in Experience Manager |
| Reusable components, examples, adoption and lifecycle presentation primitives | React Component Library | Change assets through draft/release/adoption operations; scenario-local work remains local until reuse warrants graduation |
| Validation intents, scoped suites, durable waits and receipts | Test Genie | Own execution and validation result, preserving provider truth separately from gate classification |
| Release applicability, exact candidate evidence, required criteria, waiver and promotion | Deployment Manager | Consume owner evidence; decide readiness without becoming a capture engine |
| Brand/asset truth and selected identity restrictions | Existing brand/content owners | Keep identity compliance separate from sufficient visual/interaction quality |
| Cross-provider source ingestion, Google connections and private search | Existing external knowledge/domain owners | Consume eventual capability; do not duplicate OAuth, ingestion or source search here |
| Task selection and visual/interaction judgment | Existing skills, amended where appropriate | Skills guide; governed programs compose repeated work; domain services own durable state |

These owner allocations are the starting target, not permission to preserve every current seam. An epoch may simplify or replace conflicting ownership when evidence supports a better boundary. Record the decision, preserved behavior, replaced owners, removed paths, consumer migration and validation. A second browser engine, parallel state registry, new audit scenario, or universal workflow framework needs explicit evidence that the existing owner cannot economically serve the need.

## Required outcomes

The acceptance scenarios and evidence requirements are in [VALIDATION.md](../internal/experience-quality-delivery/VALIDATION.md). These are destination obligations; status belongs in the delivery gap register and native product evidence.

| ID | Required behavior |
| --- | --- |
| EQ-01 | When the operator states a design/capture/audit/release intent, agents MUST discover a proportionate supported route and obtain bounded useful results without depending on an OpenAI-only connector. |
| EQ-02 | When a supported execution environment is unavailable, agents MUST preserve cloud design/research/review capabilities and expose the exact pending runtime proof and recovery route. |
| EQ-03 | When methods or selected materials cross environments, the receiving agent MUST resolve source identity, adopted revision, precedence, usable bytes and necessary original assets. |
| EQ-04 | When existing experience is assessed, the system MUST reconcile requested scope with source/navigation/runtime discovery, include material secondary surfaces and adverse states, and expose unknown inventory. |
| EQ-05 | When a declared state is captured, execution MUST establish its preconditions, perform supported entry actions, assert the actual visible state and permit safe exit/reset. A state label or navigation success is insufficient. |
| EQ-06 | When a collection is requested, the system MUST retain explicit requested-versus-observed coverage across applicable device, theme, role, state, extent and output dimensions. |
| EQ-07 | When transient behavior is assessed, the system MUST retain faithful temporal evidence and observable expected behavior for feedback, races, interruptions, scrolling, resizing, focus and gestures. |
| EQ-08 | When evidence is presented, the atlas MUST derive from existing owner records, provide readable state/journey navigation, full-size originals and video timestamps, and distinguish observed baseline, proposed design and acceptance. |
| EQ-09 | When quality findings are reported, each material finding MUST name expected/actual behavior, reproduction, affected conditions, consequence, evidence, disposition and repair verification. |
| EQ-10 | When capture/analysis is repeated, durable operation identity MUST support bounded observation, resumability and selective valid reuse without replaying effects after an uncertain response. |
| EQ-11 | When runtime/policy evidence is incomplete, missing, skipped, stale, unverifiable or synthetic, its original evidence state MUST remain visible and MUST NOT become a required release pass. |
| EQ-12 | When a customer-facing UI candidate is prepared for deployment, Deployment Manager MUST require attributable experience coverage, behavioral evidence and appropriate visual review under one explicit policy. |
| EQ-13 | When a predecessor or build/source/material changes, reuse MUST respect applicable identities and affected conditions; required changed-behavior and regression checks MUST run. |
| EQ-14 | When Web Console is used as the pilot, its reported responsiveness, composer, message-position, overlay/gesture and visual-consistency problems MUST receive demonstrated dispositions and repairs within accepted scope. |
| EQ-15 | When the capability is generalized, a materially different scenario MUST exercise the same owner path and records without copied scenario-specific orchestration. |
| EQ-16 | When an owner path is replaced, obsolete mechanisms and callers MUST be retired or have a bounded explicit migration; unrelated functionality and unique validation protection MUST be preserved. |
| EQ-17 | When automation claims device, accessibility, visual or performance qualification, the claim MUST match the actual tested device/runtime/output and declared measurable or reviewed contract. |
| EQ-18 | When the effort completes, normal agent and operator entry points, maintenance ownership, recovery, learning capture and required deployment gates MUST use the delivered path. |

## Capture and audit evidence

Reconcile route/source manifests, visible navigation, component/overlay sources, selectors, existing experience documents, workflows and actual runtime observations. Make omissions explicit. Web Console is a one-route workspace with many overlays and modes: route count is a poor coverage denominator.

Each executable state recipe links to the owning workflow/fixture, preconditions, authorized effects, data/role, entry and assertion, readiness, exit/reset, output kinds and semantic experience bindings. Use a verified presentation instance or routed test storage for mutating cases. Preserve user sessions and data. Fixtures exercise declared behavior; they do not establish live backend, account or physical-device success.

Keep a bounded matrix, not an uncontrolled Cartesian product. The initial product-review expectation is desktop/light, desktop/dark, mobile/light and mobile/dark, with documented applicability and representative equivalence. Add other dimensions according to actual risk: keyboard open, rotation, coarse pointer, reduced motion, locale/RTL, zoom, roles and large/empty/error data. A pairwise structural matrix does not automatically fulfill four-variant product-review scope. Do not hide a unique state behind a representative route. Sample repeated/infinite content with an explicit extent and reason.

Maintain two evidence modes:

- Settled evidence: visible readiness outcome, screenshot, computed/accessibility structure and geometry, long-page/nested-container extent, normal and difficult states.
- Temporal evidence: normal-motion recording, actual pointer/tap/keyboard actions and timestamps, transition observations and useful timing/geometry/scroll/focus measurements. Reduced-motion checks remain a separate accessibility condition.

Do not infer responsiveness from endpoint latency alone. Measure action acknowledgement through pending, completion/error and recovery. Immediate visible acknowledgement and duplicate-action protection are required where appropriate; a pending indicator must survive handoffs between surfaces. Retain drafts through failures and make cancellation/uncertain completion honest. Choose numerical performance or stability budgets from baseline evidence and the product contract; no invented universal beauty score or unsupported precision.

Review gesture direction/threshold, scroll-versus-dismiss arbitration, interactive child controls, nested overlays, Escape/back/backdrop, focus restoration, keyboard/safe-area and reduced motion. For message views, preserve user intent during streaming, prepend, virtualization, row resize, navigation, reopen, and keyboard/viewport changes. For composers, exercise content near wrapping/growth thresholds, paste/delete, long tokens, voice insertion, different toolbar configurations and keyboard transitions.

Preserve original bytes and raw recordings. Annotated/cropped/edited/redacted derivatives retain their own identity and provenance. Evidence includes actual source/build, environment, collection/execution, state/role/data, requested and actual viewport/DPR/raster/theme/extent, artifact identity, time and qualification limits. Privacy, retention and materialization remain with their owners. No raw credential/private transcript content in generalized fixtures or public repository fallbacks.

## Release integration

The required UI experience criterion is owned by Deployment Manager policy and fed by existing producers. It should join UI Health findings, Experience Manager coverage/claims, Test Genie workflow/regression receipts and explicit visual review. Reuse the same collection and finding identities used during development and design; do not create a release-only duplicate audit.

Applicability is explicit: customer-facing UI releases are in scope, while a genuinely UI-less artifact has a reasoned not-applicable disposition. Internal tools may adopt different policy, but must not be mislabeled as customer-ready by default. First release needs a complete applicable baseline. Updates need changed-surface/behavior and retained critical-journey evidence; broad UI/component/theme changes require expanded coverage based on impact. A version label does not prove low impact.

Bind evidence and review to exact candidate/artifact, target class, source/recipe/profile versions and selected design where applicable. Test Genie's skipped-on-infrastructure-absence convention remains truthful producer behavior; release policy must refuse required unknown/skipped coverage instead of treating a zero-error report as sufficient. Expiry and source changes invalidate only evidence whose applicability no longer holds; reuse cannot silently preserve an old approval.

Subjective visual judgment remains explicit reviewed evidence with scope and reviewer provenance. Deterministic regressions can gate directly. Heuristic/image-model findings remain advisory until the owning doctrine supports promotion; do not impersonate a human attestation with an agent verdict or smuggle a flaky beauty score into CI. Existing human approval/waiver and promotion routes remain the only routes. This effort can test approval/refusal using authorized fixtures without publishing or charging anyone.

## Simplification and adaptive delivery

Maintain the target here, dated facts in RESEARCH.md, contradictions/gaps in GAPS.md, decisions in DECISIONS.md and live work only in QUEUE.md. Each meaningful discovery updates the affected source and next slice; a queue entry is not the specification. Preserve settled constraints and alternatives when changing a design.

Before feature accumulation, inventory duplicate capture adapters, handwritten readiness/state inventories, artifact/path translation, atlas scripts, validation implementations, skill instructions and outdated entry paths. For each, choose retain, repair, replace/delete or explicit migration based on callers and protected behavior. Measure actual modules/paths and operator/agent effort before selecting budgets. Do not demand net-negative lines for a legitimate new capability, and do not call moving code or renaming owners simplification. Replacement epochs state removed owners/callers and compare measured before/after complexity alongside behavior.

The first enabled delivery outcome should establish one useful end-to-end Web Console evidence/review journey while reconciling the needed ownership and deleting superseded paths. Follow with substantial slices chosen from the largest remaining gaps. Validate a second product, failure/recovery cases, cloud/offline continuity and release refusal/acceptance before declaring the capability generally delivered. Generalization and debt reduction must serve these outcomes rather than postpone user benefit indefinitely.

## Bounds and unsettled choices

Preparation authorizes research, local source import, target and handoff documents, disabled native team configuration and independent handoff review. It does not authorize team activation, delivery runs, product implementation, live mutation, deployment/publication, new paid spend, account setup or credential acquisition.

Later implementation can extend the necessary declared owner boundary through recorded epochs and canonical contract amendments after activation. Respect existing BAS/RCL/desktop/external-knowledge teams; coordinate actual overlap rather than assuming either a whole-project lock or freedom to overwrite. Required shared schema/protobuf/generated changes remain part of the whole outcome when needed, through their governed owner commands. A worker's exclusion only constrains that worker.

Initial recommendations that need discovery rather than a new owner decision: use Experience Manager as the collection/read-model home if its current seams support it; use existing BAS workflows for action recipes; generate atlas views from durable evidence; choose a contrasting second scenario after inspecting current ownership. Consequential new art direction, external publication, live account/device access or a scope change remains explicit. Missing physical/device/player access leaves that claim unverified and owned; it does not stop independent engineering or turn simulated evidence into a pass.
