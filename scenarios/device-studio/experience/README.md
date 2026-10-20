# Device Studio experience contract

**Design only. No product UI, completed journey, visual approval or runtime accessibility evidence is claimed.** These contracts specify what the future UI must communicate and let an implementation reviewer see the intended states, scope and recovery paths. [DESIGN.md](../DESIGN.md) owns the visual concept, shared-shell ownership, responsive composition and interaction rules. [PRD.md](../PRD.md) and [requirements/](../requirements/index.json) own committed behavior and acceptance obligations.

## Route truth and contract inventory

[index.json](index.json) contains **16 draft pages and eight draft journeys**. Three pages document actual generated template routes: `/` is placeholder Home plus health, `/notes` is the removable example, and `/settings` provides template preferences. Their existence does not implement Device Studio. Thirteen entries propose product routes for projects/project, library/asset, import, creation, review, shot, jobs/job, result, account/providers and project privacy/budgets. All entries use schema-supported `draft`; `x-route-status` distinguishes implemented template from planned product routes.

Every future page names communication priorities, the 18 shared intended UX states plus task-specific states, linked operational targets, and aspirational claims. `x-design` specifies planned layout, controls and critical behavior; `x-requirements` links actual requirement IDs where supplied. Schema-supported extension metadata is **design documentation**, not grounded elements or checker results. Future controls have no invented `elements`, `regions`, `bindings`, testids or state-setup hooks.

The Notes example retains its actual `notes-surface` testid binding. Its inherited pin was corrected from `experience-surface@1.0.0` to **1.0.3**, because the actual UI imports `ExperienceSurface/1` and the current package export resolves to 1.0.3, whose canonical contract exists. No library or UI release bytes were changed. The original failing receipt is retained below.

## Intended journeys

| Journey | Scope |
|---|---|
| [First photographic shot](journeys/first-photographic-shot.json) | Local project, privacy/limits, source fitting, explicit render and direct output review |
| [Controlled device motion](journeys/controlled-device-motion.json) | Real device/joint motion, compatibility refusal, independent source clock and readable hold |
| [Agent creation and release](journeys/agent-creation-and-release.json) | Disclosed bounded creation, retained comparisons, separate operator release |
| [Offline recovery](journeys/offline-recovery.json) | Local drafts, interruption, reconciliation, partial and final results without duplicate work |
| [Account and funded work](journeys/account-and-funded-work.json) | Optional scoped connection, independent data consent and cap/funding gates, accurate settlement |
| [Mobile calibration](journeys/mobile-calibration.json) | Full precision controls on narrow viewports with numeric/keyboard paths |
| [Scoped library release](journeys/scoped-library-release.json) | Safe import, rights/provenance, exact artifact/version and independent qualification |
| [Portable replay and handoff](journeys/portable-replay-and-handoff.json) | Secret-free bundle, missing-media relink, compatible replay and direct launch-video/API/CLI artifacts |

These are planned sequences. No browser workflow has executed them. All **38 claims are custom/aspirational**, with no machine-tier claims, manual attestations or fabricated captures. The 18 shared state declarations are intended behavior, not evidence that every template surface already supports them.

## Acceptance and evidence frontier

The semantic design covers the full committed P0 scope: eight representative device groups, photographic composition, controlled phone/lid motion, guided and advanced editing, reference-based AI/agent creation, reusable asset review, durable jobs, independent privacy/funding/authority gates, desktop delivery, public integrations and output/worker qualification. A useful first photographic slice does not complete that scope.

AI and agent actions must be named before initiation with provider/funding, exact data egress, nested scope, enforceable aggregate limits and stop behavior. Deterministic editing, stored asset reuse and ordinary rendering never launch hidden AI. Account connection and credentials do not imply project media consent. Cost-bearing work without a safe enforceable upper bound is refused. Local projects remain usable without cloud sign-in.

Before product UI work, complete the governed concept and paired desktop/mobile mockup checkpoints. Actual mockups and human ratification are absent here. Implement real routes and controls, declare independently meaningful async regions with their genuine component pins, add accessible names and real selector bindings, and provide deterministic state setup. Then capture actual mobile/desktop/theme/keyboard/reduced-motion/RTL/error paths and execute journeys. Promote only the claims that have real checker or manual evidence. No fake selectors or notes-only bindings should be added to make the contract green.

The current visual frontier is **concept authored, before mockup ratification and implementation**. It is distinct from experience-manager's parser/reference capability levels. `spec_contract=L3` means these draft documents parse and link cleanly. Empty active-page or manual-claim sets can yield clean structure/manual sub-assessments vacuously; those labels do not prove any built product, visual quality, accessibility or freshness of nonexistent evidence. The home placeholder and orientation gate remain unchanged.

## Validation retained

- [Canonical validation](validation/spec-validation.json): `experience-manager spec validate device-studio --json` reports `PASSED`, zero findings; parser/reference local level L3.
- [Cross-document agent review](validation/cross-document-review.json): retained corrections followed by two fresh clean semantic passes on pinned document bytes. These are same-author document reviews, not independent human or runtime acceptance.
- [Recipe byte replay](validation/recipe-replay.json): authoring script reproduced all25 page/journey/index documents byte-for-byte; this is document reproducibility only.
- [Semantic authoring audit](validation/authoring-audit.json): 86 checks passed for file/index parity, all-draft status, real OT/REQ links, exact journey page/state links, complete shared intended-state vocabulary, absence of fake future bindings/attestations, actual Notes selector/version and all 14 committed P0 target groups. Includes design/contract hashes.
- [Template-owner bug](validation/template-pin-bug.json): report-bug published `knw-1792459024617298162`; the canonical template source remains outside this work boundary.
- [Inherited pin finding](validation/spec-validation-inherited-pin.json): first run failed on the generated Notes 1.0.0 canonical reference; fixed by aligning the spec with the actual 1.0.3 export.

Reproduce structural validation from the repository root:

```sh
experience-manager spec validate device-studio --json
python3 scenarios/device-studio/experience/audit-contracts.py
```

[author-contracts.py](author-contracts.py) is the deterministic design authoring recipe for these initial page/journey documents; it preserves the real template Notes binding and corrected pin. The JSON documents are the delivered contracts. Keep recipe and documents aligned if reauthoring. [audit-contracts.py](audit-contracts.py) audits contract/source linkage only and never starts a UI, worker or provider. No broad test suite, renderer, account login, install or lifecycle operation is required to validate this documentation change. The CLI performed its normal stale-fingerprint self-rebuild before the first request; that is not product implementation.

Runtime execution, visual and temporal acceptance, accessibility, responsive evidence, actual worker and credit integration, supported platform packaging and external-media/budget enforcement remain unimplemented obligations in the requirements and delivery plan.
