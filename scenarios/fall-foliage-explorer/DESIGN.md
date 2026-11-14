# Design Contract

## Intent

Fall Foliage Explorer should feel like a practical seasonal planning tool: warm, map-forward, information-dense, and easy to scan. The UI should help users compare regions, timing, reports, photos, and trips without turning the experience into a marketing page.

## Feedback & State

Use visible loading, empty, and error states for API-backed regions, reports, trips, and prediction requests. Preserve the autumn palette while maintaining WCAG-friendly contrast for status badges, buttons, and map-adjacent controls.

## Request Lifecycle

The UI resolves its API base through app-monitor proxy metadata before falling back to loopback. API-dependent controls should remain usable when read-only fallback data is available, and write failures should be surfaced clearly rather than hidden.

## Cross-References

- [CODE: ui/src/app.js] - UI state, API calls, bridge initialization, map, reports, trips, and exports.
- [CODE: ui/src/styles.css] - Responsive layout and autumn visual treatment.
- [DOC: docs/concepts/ARCHITECTURE.md] - Runtime surface ownership.
