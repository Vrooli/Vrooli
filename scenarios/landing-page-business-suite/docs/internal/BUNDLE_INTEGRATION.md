# Business Suite Bundle Integration Status

## Last Updated

2026-09-14

## Integration Status

| Area | Status | Notes |
|---|---|---|
| Catalog registration | ✅ | LPBS exposes admin CRUD for `download_apps`, platform assets, and storefront links. |
| Public visibility | ✅ | App-level `metadata.enabled` is independent from stored release metadata; disabled records remain editable but are not advertised. |
| CMS metadata | ✅ | Admin can edit web URL, catalog status, feature gates, agent-plugin classification, copy, installers, and store links while preserving arbitrary metadata. |
| Desktop delivery | ✅ | Managed/direct assets, platform release notes, checksums, and entitlement flags remain supported. |
| App stores | ✅ | Apple App Store and Google Play storefront URLs are persisted and rendered as external links. |
| Subscription/credits | ✅ | Existing LPBS entitlement and credit APIs remain the source of truth; catalog feature-gate labels are descriptive and do not duplicate billing. |
| Scenario to Plugin | ✅ | The staged catalog contains a disabled Scenario to Plugin entry with `agent_plugin` metadata for future agent-facing publication. |

## Catalog Seed

The fallback Business Suite catalog includes the Offer Desk candidates and enabling
ramps: Web Console, Git Control Tower, Agent Manager, Swarm Manager, Prompt
Manager, App Library, Scenario to Plugin, Vrooli Onboarding, Vrooli Bridge,
Notification Hub, Device Control, Switchboard, Treasury, System Monitor, Portal,
and Vrooli Memory. Web Console is the only seeded public entry; the remaining
records retain polished metadata while remaining disabled until their release
gates clear.

## Verification Evidence

- `pnpm test -- --run src/surfaces/admin-portal/services/downloads.service.test.ts src/surfaces/admin-portal/hooks/useDownloadsForm.test.tsx src/surfaces/admin-portal/routes/DownloadSettings.test.tsx src/surfaces/public-landing/sections/DownloadSection.test.tsx`: 104 tests passed.
- `pnpm test -- --run`: 179 files and 2,201 tests passed.
- `pnpm run test:coverage`: 2,198 tests passed; the repository baseline still exits non-zero because global branch coverage is 84.17% against an 85% threshold.
- `pnpm build`: production UI build passed.
- `GOWORK=off go test ./... -count=1 -timeout 10m`: API suite passed.
- Live Connect `LandingConfigService/GetLandingConfig` returned 20 catalog records; the lifecycle-managed browser DOM reached `ready`, showed `bundle-app-web-console`, omitted disabled entries, and exposed the Web Console CTA.
- Final visual artifacts: desktop/mobile full-page screenshots, a dedicated Aquila catalog screenshot, hero/catalog video frames, and a 12-second valid MP4 under `.vrooli/artifacts/lpbs-20260914-aquila-final3/`.

## Remaining Qualification

The current lifecycle-managed instance is healthy. The admin route is protected and
its CMS CRUD, subscription-gated download authorization, and homepage variant
behavior are covered by focused tests and existing API contracts; no live admin
mutation was made during this verification. External store URLs and release
artifacts are intentionally absent for staged applications until their release
gates clear.

## Visual Capture Root Cause

The original MP4s were invalid: Chrome was launched with `--headless` while
`ffmpeg` captured the Xvfb display, so the display contained no browser window.
A fresh profile could also present Chrome's first-run dialog. The reusable
`scripts/capture-public-evidence.sh` now uses a visible app window, suppresses
first-run UI, records both the hero and catalog states, and fails closed when
the output is empty, too short, sustained-black, or has a near-zero first-frame
luma average.
