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
| Capture | App-only or already-composed. Already-composed uses plain presentation and does not receive additional chrome. |
| Phone | Clock/carrier text, battery percentage, cellular/Wi-Fi/home visibility, status visibility, light/dark appearance, and explicit optional top/bottom RGB overrides. Profile owns status/home insets and icon layout. Hiding icons does not silently reclaim a physical unsafe region. |
| Default fills | Named fixed edge-mode policy: sample a bounded top/bottom source strip, quantize colors, choose the modal bin with deterministic tie-breaking, then choose its representative color. Pin policy/version and resolved colors. Do not average text into the background. Transparent sources require an explicit matte before sampling. |
| Browser desktop | Title/address, toolbar/window-control visibility, light/dark chrome, versioned normalized window rectangle, desktop colors, and optional menu/dock visibility. Use authored neutral wallpaper geometry and generic icons with retained provenance. |
| App fit | Aspect-preserving contain for E2, with an explicit matte and recorded source-to-viewport transform. Cover remains render-blocked until its crop and mapping are implemented and tested. |
| Text | Bundled font bytes and rasterizer version/digest; declared supported glyphs. Unsupported glyphs produce an actionable issue rather than silent substitution or host-font fallback. This does not restrict languages inside the captured app image. |
| Derived geometry | Return display size, app viewport rectangle, fitted app rectangle, scale/translation, and any letterbox regions. Caller-supplied viewport dimensions are not render authority. |

Use one pure presentation owner for layout and rasterization, consumed by compiler validation and the private worker. Extend the typed snapshot and private manifest together; do not precompose an untracked screen blob outside the pinned job closure. The raw screen source remains digest-bound input. The worker receipt adds presentation profile, resolved fills, font/rasterizer identities and app mapping; the composition profile version changes when pixel semantics change. A queued job pins its old executable and complete presentation input, so later UI changes cannot mutate it.

Qualify both profiles with actual source captures: dark and light phone scenes, two configurable phone status variants, browser-on-desktop and plain capture. Check that no app pixels enter reserved bars, no simulated bars are duplicated, aspect is preserved, the physical pill remains protected, and changed title/clock/color modifies only the expected presentation regions before projection. Retain display rasters and photographic outputs for direct inspection. At least one exact replay must produce the same output bytes. Video edge sampling/smoothing and source-continuous screen entry remain E2V work.

## Timing and media contract

Record source presentation timestamps, rational output frame rate, trim, playback rate, loop/hold and out-of-range policy. Decode variable-frame-rate media by timestamp; never assume frame index equals time or that 24 fps input maps one-for-one to 30 fps output. Evaluate source frame, simulated clock, device/joint pose and camera at the same canonical output time. Seek completion is verified before browser preview capture; native jobs pin decoded inputs or a qualified deterministic decoder.

Screen media is visual-only by default; strip source audio from device-shot export unless an explicit qualified passthrough policy is selected. Final film narration/music/mixing belongs to launch-video. A passthrough option must preserve declared trim/rate and A/V sync and must never unexpectedly include microphone audio. Preserve original source bytes regardless of derived preview encoding.

## Photographic versus physical mapping

Four-corner planar warping maps a flat panel in a pinned photographic plate. Supersample perspective mapping and resolve premultiplied alpha to reduce seams. Device/camera/occlusion outside the app surface stays fixed before lossy encoding. Crop-only movement cannot reveal parallax or close a lid. A generated plate's hardware is approximate; it is not a render of the exact imported GLB.

3D maps media to measured surface UVs and occlusion geometry. Curved/flexible screens use their actual mesh/deformation. Review z-fighting under the whole scene bounds and grazing angles; a successful standalone GLB view is insufficient. Native and portable screen adapters may differ but must carry separate qualification receipts.

## Color and readability

Declare source transfer/primaries/range, display texture interpretation, view transform/exposure and output SDR encoding. Preserve authentic UI content; physically lit appearance may alter observed colors. Offer a separately qualified color-preserving presentation route where needed, never promise emission alone preserves pixel values. Reject unsupported HDR conversion or make an explicit derived conversion with receipt. Test bright/dark UI, saturated colors, fine text, subpixel edges and motion; quality is judged at actual output footprint.

1080p landscape means 1920×1080; portrait means 1080×1920 at 24 or 30 fps SDR. High-resolution stills require sufficient source/plate detail. Scaling a small plate to 4K is not native 4K fidelity. Define required readable regions and hold duration per shot brief; not every background label must remain legible during transition.

## Cross-References

[rendering](rendering.md), [acceptance](../internal/ACCEPTANCE.md), [data](../concepts/DATA.md), [requirements protocols](../../requirements/verification-protocols.md).
