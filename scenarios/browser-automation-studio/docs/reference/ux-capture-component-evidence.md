# UX capture component test evidence

The existing `POST /ux-capture/scenario` owner accepts an optional
`evidence_profile: "componenttests"` and `region_selector` (CSS, at most 256
characters). The command surface remains the scenario capture command; this does
not add a route or change the default capture profile.

After the normal redaction prepass succeeds, this opt-in profile can add:

| Artifact kind | Bundle path | Limit | Content |
|---|---|---:|---|
| `region-pixels` | `pixels/region.png` | 5 MiB | PNG of one uniquely matched selector |
| `region-style-sample` | `style/region-sample.json` | 8 KiB | Allowlisted computed visual properties for the same uniquely matched selector |
| `accessibility-snapshot` | `accessibility/snapshot.json` | 40 KiB | BAS accessibility snapshot, clipped to 32 KiB |
| `console-summary` | `console/summary.json` | 1 MiB | Error and warning counts only; message text is omitted |
| `performance-metrics` | `performance/metrics.json` | 1 MiB | Navigation timings and bounded resource count |

Each persisted artifact has a declared kind, relative path, media type, byte
length, and SHA-256 digest. Full-viewport `pixels` remain the L0 evidence;
region pixels are supplementary. The bundle keeps all seven L0–L6 lens records
with explicit status and a reason for partial or absent lenses. RCL consumers
can read their story result from the redacted frozen page and retrieve declared
artifacts through `GET /ux-capture/bundles/{bundleID}/artifacts/{path}`. L0
becomes partial when a browser resource failed or was blocked before pixels were
captured; only counts by resource type are retained. A failed or blocked
stylesheet also makes L3 partial because computed styles may be incomplete.
The style sample contains only `color`, `background_color`, `border_color`,
`border_style`, `border_width`, `visibility`, `display`, and `opacity`, each
clipped to 128 characters. It contains no page text or element attributes.

Evidence is omitted when the privacy prepass does not clear the page. A missing
or non-unique region selector also omits the crop; it never changes the main
region evidence; it never changes the main viewport screenshot. Style sampling
runs only after successful redaction and requires exactly one region match.
Console messages are not persisted because they can carry application data.
