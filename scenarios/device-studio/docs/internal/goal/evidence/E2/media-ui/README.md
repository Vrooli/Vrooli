# E2 P3 media UI evidence

Status: UI implementation, focused verification, and leased desktop/mobile browser journey completed. Parent review remains open; this is not E2 epoch or product acceptance.

## Current implementation evidence

- The Project detail view imports PNG/JPEG, preserves a failed file for retry, lists saved records with pagination, explicitly selects an item, retrieves its typed metadata, and loads its original bytes through the project-owned content endpoint. The endpoint verifies its stored digest before returning bytes. UI metadata shows the filename, decoded dimensions, MIME type, and file size; the SHA-256 is in collapsed source details.
- The stage identifies the media as an original source image and says it is not a rendered shot. Projects keeps its honest no-shot state. The project page puts the source stage beside the import/list inspector on desktop and places the stage first on mobile.
- Focused checks passed: `pnpm type-check`; `pnpm exec vitest run src/features/projects/ProjectDetailPage.test.tsx src/features/projects/SourceMediaPanel.test.tsx src/app/routes.test.tsx src/api/client.test.ts` (4 files, 17 tests).
- Follow-up scoped suite fixes align `project.editName` in all three locales and mount the RCL create dialog only after the New project action has opened it once. The shell accessibility test no longer encounters a closed-overlay delayed state update.
- Checks passed: `pnpm run test:coverage` (29 files, 143 tests, 94.49% statement/line coverage), focused locale/shell/workspace tests (3 files, 9 tests), `pnpm type-check`, and `pnpm build`.
- The healthy UI at `http://127.0.0.1:21427` served exact built bytes after the final build. SHA-256: entry JS `a85a97b9f3c727ef18769fcd9aad52d8884c77ed0f048137641c4ed237a1ea14`, CSS `a016c2b918b4ceda90c2eddb440cf820024b8f5ef50c2fa01086fbf6c94ce47e`, lazy Project app chunk `98bca298e291c3ec4b8b42a18eeed530a1537fd4eace6f55a07e8dd944e28aee`; each response matched `ui/dist`.
- A larger synthetic PNG used only as browser evidence input is retained at `fixtures/ui-import-source.png` (1200×800, 10,015 bytes, SHA-256 `024b6838ee4653dd6ca9ae4c872c673e398e0974f57714b3e974ffee7e315cd3`). It is an abstract intake fixture, not product imagery and not a rendered device shot.
- The workspace worker’s separate API/CLI lease receipt is `../media/routed-live-journey.json`. It proves the API import/list/get/content path under a cleaned Test Genie lease, but it is not evidence of the UI journey.
- The final scoped Test Genie unit run passed: `20261020-032718-24366ec8` (1 passed, 0 failed; L2 Ready→L3). UI `pnpm run test:coverage` passed 29 files and 143 tests; `pnpm type-check` and `pnpm build` passed afterward.

## Leased browser journey

`browser-import-reopen-flow.json` uses BAS capture’s supported `--routed-test-mode` flag. Both saved execution receipts show `browser_profile.extra_headers: {"X-Vrooli-Test-Mode":"1"}`; this header was applied only to target browser traffic. The desktop flow created one uniquely named project, imported the retained fixture, returned to Projects, reopened it, reselected the saved item, and passed filename and dimension assertions. The mobile follow-up reopened the same project and media ID and passed its project-name and filename assertions.

- Desktop capture: 1440×1000 CSS viewport, 2880×2000 PNG raster. [View screenshot](captures/desktop/01858ea1-03e4-48db-96c9-044684c31864/screenshots/step-22-57d4879d-b6b7-43e4-9993-0c39cf0673b9.png). BAS execution `01858ea1-03e4-48db-96c9-044684c31864`: 22/22 steps, 2/2 assertions passed. DOM, accessibility tree, timeline, result and execution receipt are beside the image.
- Mobile capture: 430×932 CSS viewport, 860×1864 PNG raster. [View screenshot](captures/mobile/7ea261cf-3c06-4149-8c5d-00a9461388ec/screenshots/step-12-44c2cba4-0331-42f0-a354-bdfeeace31cc.png). BAS execution `7ea261cf-3c06-4149-8c5d-00a9461388ec`: 12/12 steps, 2/2 assertions passed, reopening the same project and selecting the same saved media.
- Routed lease, ClearTestPool counters, file-root write count, zero primary request/write counters, cleanup status, post-cleanup absence of the lease project from primary storage, and API health are recorded in [`browser-routed-lease.json`](browser-routed-lease.json). It records `testPoolRequests=39`, `testRootWrites=1`, and both primary counters as zero. The receipt states that the omitted scalar counters decode to default zero.
- Project ID: `451bec76-0b04-44d5-bb41-47a8a463cd7b`. Source-media ID: `d5c32717-a979-4e96-aa3b-658be9a7b210`. The captured DOM shows the fixture SHA-256 matching the locally hashed fixture; filename and dimensions were the explicit BAS assertions. Duplicate reimport was not exercised.

No renderer output, shot, qualified device, native-platform validation, source-rights conclusion, or overall product acceptance is claimed.

## P3 runtime growth accounting

The accounting compares current source with the preserved P3 prewrite snapshot in `prewrite-baseline/manifest.json` (2026-10-20 03:03:17 UTC), plus the follow-up snapshot in `followup-prewrite/manifest.json` (2026-10-20 03:26:14 UTC) for the later unit-fix edits. Across the directly snapshotted implementation and locale files, the total is **250 added / 13 deleted lines** (net +237 lines) and **+16,629 bytes**. This includes `ProjectDetailPage.tsx` (+13/-8, +144 B), new `SourceMediaPanel.tsx` (+119, +9,917 B), `api/projects.ts` (+20/-1, +1,066 B), `app/routes.tsx` (+1, +56 B), `styles.css` (−1, −288 B), and EN/JA/AR locale files (+31 each; +1,526/+1,986/+2,094 B), plus the later `ProjectWorkspace.tsx` dialog lifecycle fix (+4/−3, +128 B). The initial P3 changes in those files are +246/−10 lines and +16,501 bytes; the later workspace fix is +4/−3 lines and +128 bytes. This remains below the 850-line budget.

The Arabic locale's later namespace correction is byte-neutral (+1/−1 JSON lines) and included in the Arabic file's net comparison. The accounting excludes tests, evidence, authoring docs, selectors, generated string catalogs, and all files without a direct P3 prewrite snapshot. These are scoped snapshot comparisons, not whole-tree diffs. The original baseline snapshots remain unchanged.

The mobile screenshot is scrolled into the interaction area and clips the page's top and bottom. It demonstrates the reopened source-media interaction at that viewport, not a full first-fold composition check.

BAS console-log artifacts contained no console errors. The network-summary artifact says no activity was captured there; it is not evidence of zero requests. Use each run's `timeline.json` as the source for recorded network events.
