# Device Studio: decisions resolved and implementation readiness

2026-10-19. Follow-up to v7 after the operator selected 1A–7A and added cross-platform desktop architecture, normal Vrooli monetization/account/credits integration, and explicit AI/agent disclosure. [decisions.json](decisions.json) records those answers without modifying v7. The living [device domain](../../v2/device-domain.md) now points to the accepted scope. Its pre-update content is retained in device-domain-input.md.

## Conclusion

The product scope and owner boundaries are sufficiently defined to author the implementation contract and start scenario delivery. No further product-preference question is required by this research. The unresolved work is concrete engineering and visual qualification: GPU renderer provisioning, professional 3D motion/scene finish, mechanical laptop adaptation, scoped desktop credit-funded operations, broad library assets, provider receipts and platform packaging. These belong in explicit early implementation gates; another broad research round would not establish the missing runtime/art evidence.

No scenario scaffold, formal implementation plan, production fix, dependency installation, account/login action, paid provider invocation or release was performed. Two source-level out-of-scope integration defects were filed through the report-bug skill, with bounded evidence and no claim of a live billing/auth reproduction.

## What the additional research established

- **Commercial model:** the existing strategy preserves local/open-source/BYOK use. Optional Vrooli-funded operations go through trusted shared owners. Official desktop delivery entitlement is separate from access to local editing/rendering after installation. Offer Desk owns offer/declared-price relationships; LPBS owns business accounts, subscriptions, wallets/reservations and delivery; Money Ledger records financial actuals. Device Studio needs no new wallet, account database or price ladder.
- **Account integration:** local operator identity, linked commercial account, signed entitlement lease and per-operation spending authority are distinct. The scoped desktop linking path is useful but its lease is not already the consumer access token expected by the current funded inference path. This requires an explicit trusted adapter. Web Console's browser sign-in source also disagrees with the current LPBS PKCE/code contract. Shared account UI pieces exist; a complete catalogued AccountSettingsSurface is currently a null placeholder, so its presence is not reusable functionality.
- **Funding coverage:** AI Gateway's metered text path does not certify the separate image/media path as credit-funded. Preserve the actual local/BYOK/platform funding source, authoritative quote/rate version and known-versus-unknown usage. A reservation estimate is not a hard whole-job ceiling; nested calls/agents/retries must be covered by Device Studio's aggregate admission policy and the upstream service's enforceable caps.
- **Desktop portability:** reuse platform adapters and typed capability availability, per-user storage and declared bundle artifacts. Target a private shell/API/SQLite/UI/composition bundle plus qualified media tools and an optional verified native-renderer pack. Provider/scenario services are explicit optional integrations unless a chosen feature requires them. Do not require the development checkout or assume the entire fleet is installed. Each OS still needs actual installer/runtime tests.
- **GPU diagnosis:** an owned read-only Blender probe successfully initializes the CUDA driver and sees the RTX 4070 Ti SUPER. Cycles logs that neither precompiled kernels nor a CUDA compiler is found; the installed build also lacks OptiX support. The immediate fault is the current renderer's kernel/compiler availability, not evidence that the GPU is absent. Provision/qualify the correct tool artifact at the control-plane owner before rendering benchmarks.
- **Library breadth:** twelve refreshed public creator APIs still advertise downloadable CC BY models for additional tablet/fold/watch/display candidates. These remain acquisition leads, not qualified assets. Variant/generation ambiguity, preview complexity and exact mount/support geometry need inspection. Existing manufacturer/reference research supports authoring accurate or explicitly generic alternatives when suitable free assets are absent.
- **AI disclosure:** action labels and preflight summaries identify AI calls, agent launches, execution/funding source, aggregate budget and outgoing data. A disclosed bounded autonomous job can iterate without repeating prompts; its activity shows every child. Scope expansion needs a revised explicit choice. Ordinary editing and deterministic rendering cannot silently start AI/agents.

## Implementation and validation direction

[implementation-approach.md](implementation-approach.md) describes the complete target, modules, delivery sequence and evidence matrix. Start with common device/scene/presentation/motion/shot contracts, GPU and paid-account adapter proofs, and early art-directed 3D motion. Deliver the photographic saved-shot workflow as the first useful slice, then complete physical motion/advanced editing, reviewed broad assets and autonomous creation, desktop/commerce integration, and real launch-video consumer validation.

The completion boundary includes the accepted eight groups, phone spin/laptop lid motion, guided plus advanced controls, 1080p SDR 24/30 fps landscape/portrait outputs, reproducible project packages and AI creation with review. Current-host runtime evidence comes first; cross-platform architecture/build paths are initial implementation obligations. Platform releases require their own actual packaged-app/signing evidence. No successful import, schema fixture or rendering job substitutes for human visual acceptance.

Validation combines domain incompatibility tests, measurement/reference comparisons, phone and desktop screen-chrome diagnostics, source clocks/color/masks, continuous motion/collision/contact/temporal review, durable job recovery and cancellation, privacy/budget enforcement, account/credit replay and revocation tests, and fresh-user desktop install/offline/update checks. Use scoped Test Genie phases during development and release-wide certification for the release claim. Measure performance only on the qualified worker; do not extrapolate the earlier constrained CPU runs.

## Evidence map

- [Answered decisions](decisions.json)
- [Monetization and owner implementation](monetization.md)
- [Identity, credits and shared account UI](identity-credits.md)
- [Desktop and renderer portability](desktop-rendering.md)
- [AI/agent interaction contract and acceptance examples](interaction-contract.md)
- [Representative catalog follow-up](catalog-follow-up.md), [creator API observations](catalog-refresh.json)
- [Implementation and validation approach](implementation-approach.md)
- [Identity source-contract probe](identity-contract-probe.json), [usage SQL replay probe](monetization-replay-probe.json), [GPU probe](desktop-rendering-evidence/gpu-probe.json)
- Final reference/source validation, manifest and journal receipt are recorded alongside this report.

The v7 image-provider and Asset Studio gaps remain outstanding at their owners; this round does not erase them. Pricing, bundle promotion, signing credentials and live paid test caps remain governed owner/operator configuration at the relevant execution point. None requires another speculative product-preference questionnaire now.
