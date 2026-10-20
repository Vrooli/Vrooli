# Saved-shot UI evidence

These captures use the approved `Daylight study` Test Genie lease and existing shot/media fixtures. They are interaction and layout evidence for review, not product acceptance or render qualification.

The final read-only captures came from BAS executions `18e27df2-b84c-4daa-a472-6070ebffa880` (desktop, 1440×1000 CSS viewport) and `7dba4dca-46b0-4d65-a2ce-9b8d48c5dad4` (mobile, 430×932 CSS viewport). Both used the routed `X-Vrooli-Test-Mode: 1` browser header and opened the real `Daylight, held still` shot after its CLI revision advanced to 2. The screen shows the retained v6 scene plate with the calibrated quad overlay and the raw v4 `source-frame84.png` as screen source. BAS screenshots are device-pixel scaled (2×). The `final-*` captures include first viewport, full page, and expanded calibration on both viewport sizes.

The same-context conflict attempt is BAS execution `b29c435e-b6cb-488d-b0f8-684911f38966`. Its editor loaded `Unfinished study` revision 1, retained that same page while the owner CLI saved revision 2, then submitted through the UI. The UI returned `Invalid input: invalid shot input`, so it did not demonstrate the expected stale-revision conflict and no UI reapply/save was attempted. The failure screenshot is `captures/desktop/stale-save-invalid-input.png`. Workspace compared the backend behavior separately: a CLI stale save against the same blank revision returned the typed `expected 1, current 2` conflict. The likely UI cause is recorded by the API owner: the serializer included an empty `PlateQuad` with an unspecified coordinate convention for an otherwise incomplete snapshot. This requires a UI fix and a fresh browser proof; the current browser attempt is not evidence of a successful UI conflict flow.

The final captures above predate the bounded UI refinement currently staged in source and `ui/dist`. They expose the reviewed layout issue: selected-shot stage placement followed the project header, shot-management controls, shot list, and pinned revision controls, which pushed the stage down on mobile. The staged source now auto-selects the first saved shot, makes shot/project management subordinate disclosures, and places a compact serif shot title and history control immediately before the scene stage. It also omits an empty protobuf `PlateQuad` when no calibration point is present. Focused UI verification passes, but those visual and incomplete-save fixes still need a new leased browser proof and independent parent review.

Workspace's lease-owner receipt, counters, cleanup, and primary-project exclusion are in [routed-lease-receipt.json](routed-lease-receipt.json). The scene plate remains a scene plate with calibration overlay, not a rendered composite. D3 visual acceptance remains pending.

## Files

- `conflict-same-context.json` is the attempted same-page conflict flow. It is retained with its failed outcome; it should not be treated as a passing scenario.
- `read-main-shot-revision-1.json` is the desktop read-only inspection flow reused for final desktop screenshots. Its filename predates the current revision; the final captures show revision 2.
- `read-main-shot-mobile.json` is the mobile read-only capture flow.
- `desktop-session-parameters.json` supplies the project route, session-scoped test header, and timeout.
- `captures/desktop/` and `captures/mobile/` contain the exact screenshots inspected for this handoff. Their hashes are in `capture-manifest.json`.
