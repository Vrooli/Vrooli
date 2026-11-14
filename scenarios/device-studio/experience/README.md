# Device Studio experience contract

**Contract status: draft. Scoped E1 Projects and D3 saved-shot UI implementation/evidence exist; no whole-product acceptance is claimed.** These contracts specify intended communication, scope and recovery paths. [DESIGN.md](../DESIGN.md) owns visual direction and interaction rules. [PRD.md](../PRD.md) and [requirements/](../requirements/index.json) own committed behavior and acceptance obligations.

## Route truth and contract inventory

[index.json](index.json) contains **16 draft pages and eight draft journeys**. The `/projects` list (also available at `/`) and `/projects/:projectId` workspace are partially implemented; scoped E1/D3 evidence is linked from their implementation annotations. `/settings` retains template preferences. The Notes template example was removed through governed detemplate. Library/asset, import, creation, review, standalone shot, jobs/result, accounts/providers and project privacy/budgets remain planned product routes. Entries stay schema-supported `draft`; `x-route-status` records current partial versus planned implementation without promoting acceptance.

Pages name communication priorities, the shared intended UX states, linked operational targets and claims. `x-design` specifies intended layout, controls and critical behavior; `x-requirements` links requirement IDs where supplied. Extension metadata is design documentation, not checker results. Implemented elements are grounded only where actual selectors/evidence exist; planned controls have no invented bindings or state-setup hooks.

The prior Notes selector and inherited-pin audit describe the pre-detemplate template state and are retained only as historical evidence. Notes is absent from the current router/UI; no library or UI release bytes were changed by the governed detemplate operation.

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

The concept and paired Projects/Shot mockup checkpoints have been completed and ratified. E1/D3 have real routes, controls and scoped routed evidence. The eight journeys below are still not end-to-end product acceptance; the other routes and lifecycle states need real implementation, grounded bindings, deterministic state setup and applicable desktop/mobile/theme/keyboard/reduced-motion/RTL/error evidence. Promote only claims supported by actual checker or manual evidence. Do not add fake selectors or retain obsolete Notes bindings to make the contract green.

The current visual frontier is **approved Light-table direction with scoped E1/D3 runtime implementation**. It is distinct from experience-manager's parser/reference capability levels. `spec_contract=L3` means these draft documents parse and link cleanly. A scoped capture or clean structural audit does not prove full accessibility, final rendering, library/jobs/accounts, or whole-product visual acceptance. The template home route and orientation gate remain separate from the implemented Projects list.

## Validation retained

- [Canonical validation](validation/spec-validation.json): `experience-manager spec validate device-studio --json` reports `PASSED`, zero findings; parser/reference local level L3.
- [Cross-document agent review](validation/cross-document-review.json): retained corrections followed by two fresh clean semantic passes on pinned document bytes. These are same-author document reviews, not independent human or runtime acceptance.
- [Recipe byte replay](validation/recipe-replay.json): authoring script reproduced all25 page/journey/index documents byte-for-byte; this is document reproducibility only.
- [Semantic authoring audit](validation/authoring-audit.json): historical pre-detemplate audit snapshot; its 86 checks covered file/index parity, draft status, links, intended states, absence of fake future bindings/attestations, then-current Notes selector/version and all 14 committed P0 target groups. It is not a current Notes/source audit. Includes the source fingerprints used for that run.
- [Template-owner bug](validation/template-pin-bug.json): report-bug published `knw-1792459024617298162`; the canonical template source remains outside this work boundary.
- [Inherited pin finding](validation/spec-validation-inherited-pin.json): first run failed on the generated Notes 1.0.0 canonical reference; fixed by aligning the spec with the actual 1.0.3 export.

Reproduce structural validation from the repository root:

```sh
experience-manager spec validate device-studio --json
python3 scenarios/device-studio/experience/audit-contracts.py
```

[author-contracts.py](author-contracts.py) is the deterministic design authoring recipe for these initial page/journey documents; it preserves the real template Notes binding and corrected pin. The JSON documents are the delivered contracts. Keep recipe and documents aligned if reauthoring. [audit-contracts.py](audit-contracts.py) audits contract/source linkage only and never starts a UI, worker or provider. No broad test suite, renderer, account login, install or lifecycle operation is required to validate this documentation change. The CLI performed its normal stale-fingerprint self-rebuild before the first request; that is not product implementation.

Runtime execution, visual and temporal acceptance, accessibility, responsive evidence, actual worker and credit integration, supported platform packaging and external-media/budget enforcement remain unimplemented obligations in the requirements and delivery plan.
