# Screen media and simulated presentation

## Scope

Foundation contract, 2026-10-19. Product behavior below is specified for implementation; the generated reference scaffold does not implement it. Research demonstrates individual techniques, not a released scenario.

Preserve real app media as a separate immutable source. OS/browser decoration and generated scenery are simulated layers with independent provenance.

## Display composition

Compose the physical display mask, hardware occlusions, simulated system regions and actual app viewport explicitly. The phone status background extends around the Dynamic Island; only the pill/camera is physical occlusion. Do not fill the whole top strip black. Status/home background defaults to the corresponding app-edge color with an explicit sampling/override policy. Avoid temporal flicker from changing sample colors; declare fixed sample, smoothed timeline or manual fill. Clock, Wi-Fi, signal, battery, contrast theme and home indicator are customizable per profile. These are visual approximations unless exact OS metrics are established.

Do not add a second status bar when the recording already includes one. A media import declares app-only versus full-screen capture; profile selection previews the intended crop/insets before saving. Fit supports contain, cover and explicit crop with aspect preservation. Any cropping is visible and recorded; do not stretch app pixels to fill device glass.

Laptop/monitor options include app-only, browser-with-desktop and other declared host profiles. Address/title/tab state, browser controls, menu bar, dock/taskbar, wallpaper and window placement remain separate. Use permitted fonts/icons/wallpapers; no real credentials, sensitive URLs or notifications are fabricated into the scene. Browser chrome is optional and may differ by chosen OS/profile.

## First still presentation implementation

Capture area and simulated presentation are separate typed choices. An app-only image may use plain, phone-system or browser-desktop presentation. An already-composed physical-screen image uses plain presentation; adding simulated bars to that capture is an explicit incompatibility. Until a renderer implements a presentation kind, a saved draft may retain it but render admission must explain the unsupported choice.

The still presentation renderer will produce one deterministic display raster before photographic projection. It must return the actual app viewport and fit transform alongside its profile/version, so screen-entry and later video work can use the same mapping. Keep source pixels separate from simulated regions. For the first phone profile, expose status/home visibility, clock, connectivity and battery state, appearance and explicit top/bottom fill overrides. The default fill samples fixed source-edge colors with a named deterministic policy; it never treats the whole status strip as hardware. The plate's hardware mask protects the pill/camera independently. Profile geometry is expressed in display-raster pixels or versioned normalized fractions, never mistaken for measured physical device dimensions.

Browser-desktop presentation places the app inside an explicit window viewport and renders the surrounding toolbar, desktop and optional menu/dock regions separately. Bind window placement, title/address, colors and visibility to the revision. Fonts/icons must be bundled with recorded provenance and consistent rasterization; do not depend on the host's font search paths. Standard-library image operations and a governed image/font dependency are preferred over introducing a browser or Python runtime solely for these overlays. This is the next renderer obligation after the plain saved-shot path, not a claim that current chrome fields are executable.

### E2 implementation boundary

The first executable profiles are a generic tall-phone system presentation and a generic browser on a desktop. They are visual simulations, not certified replicas of a named OS. Profile geometry scales from a declared reference raster; the target display raster comes from the scene. A profile has an immutable identifier/version and minimum usable dimensions. Reject layouts with an empty app viewport, overlapping required regions or out-of-bounds controls. Hardware occlusion continues to belong to the scene/device mask, rather than a second painted black pill in the presentation raster.

| Configuration | First executable contract |
|---|---|
| Capture | App-only or already-composed. Plain app-only uses a full-target uniform contain fit and matte without simulated chrome. Already-composed uses the identity path and receives no additional chrome. |
| Phone | Clock/carrier text, battery percentage, cellular/Wi-Fi/home visibility, status visibility, light/dark appearance, and explicit optional top/bottom RGB overrides. Profile owns status/home insets and icon layout. Hiding icons does not silently reclaim a physical unsafe region. |
| Default fills | Named fixed edge-mode policy: sample a bounded top/bottom source strip, quantize colors, choose the modal bin with deterministic tie-breaking, then choose its representative color. Pin policy/version and resolved colors. Do not average text into the background. Transparent sources require an explicit matte before sampling. |
| Browser desktop | Title/address, toolbar/window-control visibility, light/dark chrome, versioned normalized window rectangle, desktop colors, and optional menu/dock visibility. Use authored neutral wallpaper geometry and generic icons with retained provenance. |
| App fit | Aspect-preserving contain for E2, with an explicit matte and recorded source-to-viewport transform. Cover remains render-blocked until its crop and mapping are implemented and tested. |
| Text | Bundled font bytes and rasterizer version/digest; declared supported glyphs. Unsupported glyphs produce an actionable issue rather than silent substitution or host-font fallback. This does not restrict languages inside the captured app image. |
| Derived geometry | Return display size, app viewport rectangle, fitted app rectangle, scale/translation, and any letterbox regions. Caller-supplied viewport dimensions are not render authority. |

Use one pure presentation owner for layout and rasterization, consumed by compiler validation and the private worker. Extend the typed snapshot and private manifest together; do not precompose an untracked screen blob outside the pinned job closure. The raw screen source remains digest-bound input. The worker receipt adds presentation profile, resolved fills, font/rasterizer identities and app mapping; the composition profile version changes when pixel semantics change. A queued job pins its old executable and complete presentation input, so later UI changes cannot mutate it.

The integration extends the existing typed chrome messages with appearance,
optional RGB fill/matte values, phone status and battery-text visibility, and
browser normalized window placement, colors and menu/dock visibility. Presence
must distinguish an omitted color from explicit black, and omitted optional
visibility from explicit false. Canonical defaults are versioned and persisted;
browser window controls require a visible toolbar rather than being ignored.
The first matte control applies to simulated presentation. Preserve existing
plain/already-composed revisions under their old semantic version. New
plain/app-only physical revisions use the shared contain/matte profile and a
separate immutable semantic version; reject unsupported plain-only values
instead of silently discarding them.

Compiler eligibility must validate profile geometry and supported text through
the presentation owner. If source-dependent validation needs decoded pixels,
open the bounded, digest-verified raw source through the existing media owner;
do not infer opacity from PNG format or manufacture a fake opaque sample.
Transparent simulated sources require a persisted matte. Missing controls may
remain an incomplete draft, while unsupported or unsafe supplied values produce
field-addressed issues. The worker repeats the input and profile checks before
rendering. Its receipt carries the actual resolved fills and mapping.

Submission freezes a versioned, canonical compiled closure with all source
bytes/bindings, geometry, presentation configuration and semantic/profile/font
identities, plus the exact worker bytes. Dispatch creates only attempt identity
fields from the job ledger and stages that closure; it must not call the current
shot compiler. Compose the display raster inside the private worker, then
project it into the scene. Old saved plain revisions retain their stored
semantics and pixel result; an explicit save may create a new version. Unknown
versions fail with an actionable incompatibility. Do not silently upgrade a
queued job or imply that historical jobs without a complete closure have the
new replay guarantee.

An idempotent retry identifies the original accepted request by project, shot,
revision and output profile. It returns the existing job and its first pinned
closure without resolving current media or executable bytes, including after
an application upgrade. Reusing the key for a different request is a conflict.
Concurrent first submissions still serialize through the existing job ledger;
a lookup alone does not replace its atomic uniqueness check.

The integrated editor renders an explicitly saved revision. Unsaved changes
cannot silently enter the job, and the UI must state which revision is shown.
Keep the calibration overlay distinct from a succeeded photographic PNG. Use
the existing public job contract for status, cancellation, reopening and verified
download, without a second preview-render engine or invented percent progress.

Qualify both profiles with actual source captures: dark and light phone scenes, two configurable phone status variants, browser-on-desktop and plain capture. Check that no app pixels enter reserved bars, no simulated bars are duplicated, aspect is preserved, the physical pill remains protected, and changed title/clock/color modifies only the expected presentation regions before projection. Retain display rasters and photographic outputs for direct inspection. At least one exact replay must produce the same output bytes. Video edge sampling/smoothing and source-continuous screen entry remain E2V work.

### Phone readability correction from live E2 evidence

The first live dark-source phone render exposed two defects in the generic v2 defaults: LIGHT ink was nearly invisible against sampled dark fills, and the default carrier label occupied the physical pill region. Preserve those outputs as negative evidence. New phone saves will use `device-studio-presentation-v3` and phone profile version3; browser remains v2 and the photographic homography/compositor algorithm is unchanged.

For new v3 phone revisions, persist AUTO appearance by default and an empty carrier. AUTO chooses between the declared near-black and near-white palettes using deterministic contrast against resolved top and bottom fills independently. Explicit LIGHT/DARK remain user overrides. Keep app-edge fills unchanged; do not recolor the system background to compensate for unreadable ink. Pin the contrast policy and resolved status/home inks in the v3 layout/receipt.

All v3 appearances use profile-declared safe status regions. If supplied, carrier text must fit a separate bounded region clear of the central hardware exclusion area and required clock/connectivity indicators. Validate actual bundled-font metrics; an unfit label produces an actionable issue instead of clipped text beneath the pill. These are simulation-profile coordinates, not a claim of measured manufacturer OS metrics. Qualify them against the actual retained scene hardware mask.

Stored v2 revisions retain their original missing-value defaults, geometry, serialized config/layout and output bytes. Do not reinterpret an absent legacy appearance as AUTO. New Create/Save may choose v3; read/compile/dispatch uses the stored semantic version. The private config must carry enough version identity for the worker to select the same phone layout. Newly added receipt fields stay absent in legacy v2 serialization. Browser AUTO is unsupported and must be rejected rather than silently treated as DARK.

## Timing and media contract

Record source presentation timestamps, rational output frame rate, trim, playback rate, loop/hold and out-of-range policy. Decode variable-frame-rate media by timestamp; never assume frame index equals time or that 24 fps input maps one-for-one to 30 fps output. Evaluate source frame, simulated clock, device/joint pose and camera at the same canonical output time. Seek completion is verified before browser preview capture; native jobs pin decoded inputs or a qualified deterministic decoder.

Screen media is visual-only by default; strip source audio from device-shot export unless an explicit qualified passthrough policy is selected. Final film narration/music/mixing belongs to launch-video. A passthrough option must preserve declared trim/rate and A/V sync and must never unexpectedly include microphone audio. Preserve original source bytes regardless of derived preview encoding.

The first video profile supports silent output only. An absent/unspecified audio policy means silent without rewriting historical revision bytes; explicit SILENT is supported. A saved PASSTHROUGH request remains visible and produces an actionable `unsupported_audio_policy` render blocker before job creation. Unknown enum values are invalid input. The editor explains that output is silent and reports source audio as detected, absent, or unreported.

New accepted inspection receipts use receipt version 2 with nested version-1 audio facts (original container stream index and codec). An empty list means observed zero audio streams. Legacy version-1 receipts remain byte-identical and valid with audio facts absent, meaning unreported. Rejected inspections do not fabricate audio facts. Manifest/protocol, source clock, and tool-profile identities stay at their current versions because their formats and timing rules do not change. The source digest and immutable receipt bind observed facts; no parallel audio ledger is needed.

## Photographic versus physical mapping

Four-corner planar warping maps a flat panel in a pinned photographic plate. Supersample perspective mapping and resolve premultiplied alpha to reduce seams. Device/camera/occlusion outside the app surface stays fixed before lossy encoding. Crop-only movement cannot reveal parallax or close a lid. A generated plate's hardware is approximate; it is not a render of the exact imported GLB.

3D maps media to measured surface UVs and occlusion geometry. Curved/flexible screens use their actual mesh/deformation. Review z-fighting under the whole scene bounds and grazing angles; a successful standalone GLB view is insufficient. Native and portable screen adapters may differ but must carry separate qualification receipts.

## Color and readability

Declare source transfer/primaries/range, display texture interpretation, view transform/exposure and output SDR encoding. Preserve authentic UI content; physically lit appearance may alter observed colors. Offer a separately qualified color-preserving presentation route where needed, never promise emission alone preserves pixel values. Reject unsupported HDR conversion or make an explicit derived conversion with receipt. Test bright/dark UI, saturated colors, fine text, subpixel edges and motion; quality is judged at actual output footprint.

1080p landscape means 1920×1080; portrait means 1080×1920 at 24 or 30 fps SDR. High-resolution stills require sufficient source/plate detail. Scaling a small plate to 4K is not native 4K fidelity. Define required readable regions and hold duration per shot brief; not every background label must remain legible during transition.

## Cross-References

[rendering](rendering.md), [acceptance](../internal/ACCEPTANCE.md), [data](../concepts/DATA.md), [requirements protocols](../../requirements/verification-protocols.md).


## Plain physical app-only fitting — live integration repair

Public family review exposed that physical Plain currently requires a precomposed source at the exact display raster. That prevents placing ordinary app captures on arbitrary screen families without adding phone/browser chrome. Plain now accepts explicit APP_ONLY and composes it with the existing uniform contain/matte configuration into the chosen device raster. This reuses `presentations.Config{Target: ...}`: full-target viewport, centered uniform fit and letterbox, with no added chrome and no pixel stretching. Physical Plain APP_ONLY is pinned as presentation v2; historical Plain ALREADY_COMPOSED stays v1 and retains exact-raster validation and direct-pixel identity. Do not infer phone system areas on watches, TVs, frames or vehicle displays. Still and already supported recording paths use the same rule; unsupported joint/recording combinations remain blocked. Invalid supported-presentation configuration must produce a useful invalid-argument result, not generic500. Repair admission150runtime/150tests, no new schema or parallel fitter. Current family harness failures and upside-down outputs remain retained; actual qualified outputs await corrected source review.
