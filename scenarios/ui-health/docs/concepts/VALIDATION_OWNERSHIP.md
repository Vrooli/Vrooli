# Validation Ownership

Validation findings are intentionally routed by concern. A provider may link to
another provider's evidence, but it must not duplicate its verdict.

| Concern | Authority | Evidence | Primary severity / presentation |
| --- | --- | --- | --- |
| Declared identity, `theme-color`, manifest values, and iOS status metadata | Brand Manager | `index.html`, web manifest, token contract | Required declaration drift; direct remediation |
| Library ownership of application chrome | ui-health | Declared archetype, parsed production JSX, file-scoped ejection reason | Required source error; never a provenance advisory |
| Rendered chrome/safe-area alignment and iframe-safe geometry | ui-health | Runtime screenshot, DOM, layout, viewport profile | Required runtime finding, grouped by viewport |
| Root height, scroll ownership, and cross-boundary scrolling | ui-health | Source plus runtime layout | Cross-boundary `scrollIntoView`/`window.scrollTo` advisory; scoped container `scrollTo` allowed |
| Stateful user intent such as restoration, selection scrolling, and tab transitions | Experience Manager | Explicit BAS interaction trace plus capture | Required claim failure with its state precondition |
| Governed component provenance and adoption freshness | react-component-library | Adoption inventory | Grouped governance advisory; never placed ahead of correctness blockers |
| Validation truth versus whether a run blocks policy | Test Genie | Provider result, evidence state, configured policy | Show provider truth first, then independent gate classification |

Autofix is limited to syntax-safe, template-recognized edits. Runtime layout,
scroll behavior, and chrome decisions remain detection-only because a mechanical
rewrite cannot establish intended ownership or interaction behavior.

## Safe-area evidence scope

Mobile dimensions alone do not exercise a notch or home indicator. Runtime
captures retain raw CSS env() insets and the host's effective bottom inset.
A host viewport ending above the physical screen bottom can correctly resolve
that effective inset to zero. A zero-inset mobile capture is a valid browser
render, with an explicit `visual_safe_area_not_exercised` information finding;
it provides no notch coverage.

Top chrome and bottom navigation can have different colors. The runtime capture
records the opaque semantic bottom owner (open library drawer/sheet, otherwise
navigation) as `chrome.safeAreaColors.bottom`; legacy single-color captures still
use their declared fallback. Pixel analysis scales CSS insets to screenshot
pixels. Chromium inset emulation proves document geometry and paint, not native
Safari/Home Screen chrome outside the document screenshot.


### Installed-client observations

`VisualHealthService.RecordBrowserObservation` retains bounded client-reported
geometry for 24 hours, capped at 256 records. UI Health owns its domain schema
and repository. GCT mounts an intake-only same-origin facade; it forces the
scenario identity. Its mobile client reports on load/resize and overlay changes,
with a per-tab random identity and the loaded entry-chunk filename. Reports
contain rectangles, computed edge colors and inset/viewport readings, without
page text. The receiver always marks them `browser_geometry`.

Read them with `ui-health visual observations --scenario git-control-tower --json`.
A report can reveal a stale build or a shortened frame directly on an installed
phone. It cannot show the OS home indicator or establish a native pixel color.
The GCT report probe measures 100dvh and 100lvh independently of innerHeight.

`chrome.requireNativeScreen=true` makes `visual_native_screen_unverified` an
error when capture metadata and screenshot artifacts do not establish a
`physical_screen` capture. Ordinary browser analysis reports this limitation as
INFO. BAS runtime artifacts explicitly declare `capture.kind=browser_viewport`.
A synthetic standalone flag, nonzero CDP insets, or browser geometry never
satisfies installed native-screen qualification. Physical-screen producers must
retain their device transport provenance with the screenshot; a metadata label
alone is not independent evidence.


Receiver-computed browser observations also report unused corroborated drawable
canvas, controls that fail client hit testing, toolbar height above its declared
limit, and overlay/body computed-color disagreement. Keyboard-shrunken frames
and modal interception are handled separately. These findings diagnose browser
geometry/styles; they do not claim access to OS pixels. Parser lookup falls
back to UI Health's running workspace when a target fixture is beneath an
unrelated Git root lacking the required TypeScript toolchain.
