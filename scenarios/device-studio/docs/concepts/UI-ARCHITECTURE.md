# UI architecture — Device Studio

## Shared contract

Use Template Manager's shared UI architecture and React Component Library shell/primitives. Keep generated Connect clients, i18n, accessibility, selectors and feature-folder seams. Product rules remain in the API.

## Scenario details

Target features follow projects, library/import, shot stage/inspector/timeline, creation/review, jobs/results and account/project settings. The browser stage evaluates canonical pose/source time and reports preview readiness; it is not an authoritative final renderer. Debounce interactive drafts, cancel stale preview requests, and use optimistic revision checks on save. Job views subscribe to durable API state, not browser-owned process assumptions.

Use shared forms, async regions, badges, settings, errors and list/detail layouts. Build scenario-specific3D stage/calibration/timeline controls where no suitable shared asset exists; record generic gaps through the library owner. AccountSettingsSurface is a researched placeholder, while smaller MonetizationAccount/AuthClient pieces exist. Do not edit a released library directory.

Draft experience controls have no fabricated testids or machine attestations. During each real UI slice bind accessible roles/names and actual selectors, reconcile routes, then promote claims with executed evidence. Never mark a future page active simply because its JSON validates.
