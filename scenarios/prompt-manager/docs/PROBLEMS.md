# Prompt Manager problems

## Work ladder

- Rung: W3
- Evidence: the campaign contract and requirements registry exist, while the current UX review still records implementation and acceptance gaps in `review/critique-07.md`; focused implementation repairs are the active route.
- Blocker: none for the current scoped repairs. The paused campaign still has higher-level W2 acceptance gaps for RCL behavior/calibration/independent review, page-template adoption, broader World state/mobile evidence, and qualified Agent Info health semantics.
- Measured: 2026-09-16
- Gate evidence: Prompt Manager HUD regression 21/21; Team Files regression 6/6; TypeScript clean; managed Prompt Manager rebuild completed and the existing API/UI remained healthy at 200/200; governed RCL CopyIconButton and CollectionPage adoption records exist; RCL catalog check/typecheck/boundary tests pass with zero broken version imports.
- New critique: the latest managed lifecycle start was still reporting a dependency-stage wait while the already-running API/UI remained healthy. Do not call owner readiness complete until that operation closes cleanly.
- New critique: real page-template adoption is now implemented, but calibrated RCL behavior evidence, attributable independent review, broader World live/mobile evidence, and qualified Agent Info health semantics remain open.
- Harness cleanup: the former InfoTab act warning and axe/jsdom canvas warning are resolved; the complete focused set is 69/69 with TypeScript clean.

## Fresh World density correction

- The latest BAS audit found the prior dense World claim too optimistic: 34 agents produced collisions on desktop and excessive visual weight on mobile.
- The repair adds deterministic collision separation, bounded map/rail sizing, dense-mode label suppression, smaller mobile markers, and separated overlays.
- Fresh receipts: desktop `46a6d964-c95c-4c5d-8ded-ab4503e1632b`; mobile `875a1c45-5bc0-42b3-9a84-0b7691d9f7c5`; HUD 21/21.
- This remains W3 implementation evidence only. State-matrix/journey acceptance, independent review, and qualified Agent Info health semantics remain open.
- The earlier dependency-stage wait is resolved: managed `make start` completed healthy with zero failed dependencies at 10:16:09 EDT. The optional experiment receipt-signing key remains unavailable by design.
- Agent Info state truth was tightened after a re-audit: fresh complete membership coverage is `Observed`, run activity stays `Loading` until its owner request resolves, and Empty/Unavailable/Partial/Attention/Stale are distinct. Focused InfoTab validation is 8/8 with TypeScript clean; qualified backend health semantics remain open.
