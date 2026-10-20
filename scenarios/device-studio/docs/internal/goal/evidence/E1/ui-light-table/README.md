# Light-table UI implementation evidence

## Baseline and intent

`baseline-before/manifest.json` records SHA-256 and byte size for every original `ui/src` file; `baseline-before/bytes/` retains those bytes. This is a task-local prewrite snapshot because the scenario UI had no tracked baseline at assignment start. No API, CLI, BAS workflow, or backend behavior was changed.

The implementation follows the approved Projects board: a compact “Your projects” heading and New action, a featured real project with an explicit no-preview state and Continue editing action, and saved rows when available. On phone, the actual project and empty preview state stack. Project creation opens a keyboard-accessible dialog. No mockup content is presented as live project data, and no library, jobs, or renderer behavior is implied. The revision-2 projects and shot boards informed composition and typography; the Nooch dossier influenced authored hierarchy and spacing, not its content or palette.

## Source and owner boundary

`ui/src/layout/AppShell.tsx` is currently a small scenario-owned top-navigation shell because the approved horizontal navigation is not available in RCL `AppShell/2` or `AppNavigation/1`. The manifest retains the declared `navigated-console` / `AppShell` adoption and now explains that this is the shared-library expectation with a narrow exception. `docs/reference/component-library-gaps.md` contains the `shell-ejection` record for exactly this entry; `ui-health validate scenario device-studio --static-only --json` returned no shell-ownership finding. The report remains degraded for other findings, including adoption-provider, template-slot alignment, design-token and raw-navigation advisories. Shared release files were untouched.

The RCL CLI has an `eject` handler but the command is absent from `register.go` bindings and `cli/manifest.json`; `react-component-library adoptions eject --help` returns `Unknown subcommand: adoptions eject`. This observed discrepancy was filed to Scenario QA as `knw-1792464870605212108`. No assumption is made about why the command is absent. The ui-health exception is sufficient for this scoped shell and does not claim that the ejection operation was invoked.

## Visual evidence

The `*-chrome.png` images in `captures/` are read-only Chrome captures from the running Device Studio UI at `127.0.0.1:21427`; `projects-desktop-chrome.png` and `projects-mobile-chrome.png` use an isolated browser with `prefers-color-scheme: light`, and the `*-dark-chrome.png` variants use dark preference. They remain the visual theme comparison. The browser choice only affected its temporary profile.

Current BAS screenshots, execution receipts, capture-response receipts, console logs, network notes, and DOM snapshots are retained in `captures/bas-desktop/` and `captures/bas-mobile/`. Both were captured by one-page `browser-automation-studio capture` calls with no interaction flow, no routed test mode, `networkidle` readiness, and no project writes. The execution Get receipts report `EXECUTION_STATUS_COMPLETED`, workflow `99f8ccb9-46ff-405c-ab7d-a41f85cf74e4` version 1, for `http://127.0.0.1:21427/`. Requested browser CSS viewports were 1440×1000 and 430×932 at BAS's default device scale factor; screenshot rasters are 2880×2000 and 860×1864. Readiness was `ready` with `explicit-networkidle`; both console and network artifacts state no logs/activity. Each DOM snapshot contains the real primary project “E1 validation workspace renamed”, revision 2, the no-preview message, and Continue editing link. These observations match the owner’s independent before/after read-only Get in `../primary-restart-readonly.json`.

Capture commands (each used its corresponding evidence directory as `--out`):

```bash
browser-automation-studio capture --url http://127.0.0.1:21427/ --capture screenshot,console-logs,network,dom --width 1440 --height 1000 --wait-for networkidle --label device-studio-light-table-desktop --json
browser-automation-studio capture --url http://127.0.0.1:21427/ --capture screenshot,console-logs,network,dom --width 430 --height 932 --wait-for networkidle --label device-studio-light-table-mobile --json
```

The BAS capture execution IDs are `924da4b3-73ba-40a4-8310-5004a673a271` (desktop) and `cc3d3018-6828-4820-b772-3f705555ddca` (mobile). Their exact capture responses are preserved as `capture-response.json`; `execution-receipt.json` was read from the BAS owner after capture. Screenshot SHA-256 values are `157562ecce34282e9dcb4d8c9af087509cf214d505f83eddb7f77927aef73e83` (desktop) and `18fd274ee6a8f904abe80d65640b7493f59320129cc193deedd7e710ea41d2d0` (mobile). The artifact references are `bas-capture://924da4b3-73ba-40a4-8310-5004a673a271/screenshot` and `bas-capture://cc3d3018-6828-4820-b772-3f705555ddca/screenshot`.

The Device Studio owner performed one supported restart and recorded a read-only project Get before and after; the project remained at revision 2 in `../primary-restart-readonly.json`. I did not restart the scenario. The desktop captures are 1440×1000; phone captures are 430×932.

The BAS CLI rebuilt successfully at current source fingerprint `a0f7e143794dc7df410d0fd44bed1384177d59472add4f21f5a191696d953033`; both capture runs completed without a compile error. These read-only one-page captures and the Chrome theme variants support visual review, not product acceptance. The parent owns acceptance.

The CSS requests a Charter-like display family with Georgia fallback and Ubuntu/Avenir/system sans fallback. Host fonts may be available, but no bundled desktop font proof was established. Screenshots show actual host fallback rendering; exact typography portability remains unverified.

## Verification

The UI suite previously passed 28 test files / 136 tests, and TypeScript type-check passed. The parent reports that the scoped E1 Test Genie unit gate passed earlier. After the final CSS selector correction, the normal package command `pnpm build` passed and wrote the usual `ui/dist`; the running server serves that output. Served CSS SHA-256 matched `ui/dist/assets/index-C9Y1F6sp.css`: `0fc02bc6e0e3e41d2024c4a73bc07134b0289bb8999db5802110dbe24bef3997`. This task did not rerun Test Genie.

This evidence records implementation and current-host visual review only. No product acceptance, desktop-browser portability, or live-source rights beyond the existing project state are claimed.
