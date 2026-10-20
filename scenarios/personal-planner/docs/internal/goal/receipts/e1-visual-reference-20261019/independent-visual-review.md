# Independent E1 visual and seed-receiving review

Reviewed 2026-10-19 against the original `ACCEPTANCE.md` visual-reference clause and current `QUEUE.md`. This is an independent evidence review, not an owner acceptance decision. No browser run, test, build, lifecycle operation, seed operation, or native wake was initiated.

## Retained run and isolated seed cleanup

Workflow Health run `e73c0ca8-e1b5-44ba-8d6c-0ee57ecec4b9` is recorded as `SUCCEEDED`, with the two selected visual cases each passing and no skips. The retained case timelines show the screenshot and assertion actions succeeded. This run targets `personal-planner@shadow`, build `sha256:701de714370b0018b4c56dafafc0db303327e66fd97e03688bf8b1092f819049`; the run receipt reports an installed lease, 78 test-pool requests, zero primary requests/writes and zero test-root writes. Those are the receipt's run-level values, not per-capture measurements.

I visually inspected all four current JPEGs and all six local PNG source assets, and loaded each file to verify its actual dimensions and SHA-256. Each source PNG hash matches the run receipt and the canonical manifest: desktop Day plate `b7bfff11…cb64b15` (1672×941), Day landscape `4bfe6e1b…a72e19a8` (1672×940), desktop Night plate `81eef441…2df994b` (1672×941), Night landscape plate `3c0738ed…9e3113` (1672×941), mobile Day plate `3306dc0e…b610764` (941×1672), and mobile Night plate `61263a26…1168da` (941×1672). The exact full hashes are in `receipt.json` and `visual-reference-manifest.json`.

The screenshots show the requested Day and Night Plan Month routes. The desktop captures show the star-chart desk and lake scene integrated behind the Planner navigation and calendar. The narrow captures use the portrait scene and compact Day/Week/More navigation; the calendar is responsive and readable in the visible area. I saw no obvious wrong-theme or wrong-scene substitution. The lease data is empty: the month shows empty-state “Clear” cells, with no event cards. So these images do not assess seeded event presentation or establish matching UI/CLI/API event IDs.

The screenshots show only the first three or so desktop calendar rows and the first four date cells in the narrow view, followed by the fixed navigation. Output dimensions exactly equal 2× the configured CSS viewport (1280×800 → 2560×1600; 390×844 → 780×1688), despite the recorded full-page request/extent. This is consistent with the calendar scrolling inside its own container while the document itself fits the viewport; the artifacts do not show the entire scrollable month. This limits whole-month visual inspection, but I do not add a requirement to capture every month row beyond the existing acceptance language.

## Screenshot quality and DPR

The raw screenshot outcome notes in both retained timelines mark all screenshot actions `screenshot_degraded=true` and record JPEG media. `scenarios/browser-automation-studio/playwright-driver/src/telemetry/screenshot.ts` sets this flag on the path where an initial PNG exceeds the effective screenshot-size limit and a compressed JPEG fallback is returned; that fallback preserves the requested `full_page` extent metadata. `config.ts` sets the default maximum to 512,000 bytes but allows `SCREENSHOT_MAX_SIZE` to override it. The run did not retain the effective runtime limit or original PNG byte sizes, so I cannot supply the threshold that triggered fallback. The JPEG output sizes are 213,579; 178,570; 127,417; and 110,399 bytes.

Neither timeline contains a `devicePixelRatio`, `device_scale_factor`, or `dpr` observation. The images are exactly twice the configured CSS viewport dimensions, which proves a 2× output-pixel ratio only; it does not independently prove the browser's actual DPR. The dimensions and missing browser field are distinct evidence.

## Seed state and cleanup

The separate Planner seed run is `46cdbec3-73f2-463f-bf0e-c41f186a680e`. Its private retained `apply.stdout` reports `status=seeded`, and `cleanup.stdout` reports `status=cleaned`; both identify the same scenario/run. The private receipt also records exact-token match and successful fail-closed cleanup. The four private stdout/stderr files are mode `0600`; both stderr files are empty. I read only whitelisted status/scenario/run identifiers and did not emit, copy, or disclose the token. The maintained CLI does not expose numeric resource/counter totals, so none are claimed.

## F1 disposition and owner feedback

The original F1 clause calls for current Plan/Month desktop and supported-narrow Day/Night captures, exact route/state/source/build/runtime/viewport/DPR/fixture/capture identity, side-by-side review, and dispositions for material differences. The retained set supplies the four requested views, identity and viewport details, the empty leased state, capture receipts, and a visual comparison against all six applicable source plates. I found no obvious scene/theme mismatch; the visible calendar text and controls are readable.

The precise F1 evidence gap is the **actual browser DPR**. The CSS viewport and 2× output-pixel dimensions are recorded, but neither case timeline records `window.devicePixelRatio` or an effective browser `device_scale_factor`. F1 explicitly asks for viewport and DPR provenance, so that field remains unverified until an existing capture-time browser-profile record establishes it or the owner records the actual value through the existing visual workflow. If no such record exists, the established reviewer should classify that single item as unverified/open under the existing F1 rule. No other new screenshot-quality gate is justified by this review.

The JPEG `degraded` flag is explained by the retained source's oversized-PNG-to-JPEG fallback and does not by itself disqualify a capture under the original criteria; I inspected the resulting images directly. The empty month is a documented truthful-data state, not a requirement to reproduce seeded UI/CLI/API IDs in this visual run. The viewport-sized screenshots show the visible portion of the inner-scrolling calendar; F1 asks for calendar readability and responsive composition, not an image of every scroll row. The October 3 artifacts remain historically not found, but F1's current-runtime capture requirement is supplied by the October 19 run; the old artifacts are not a blocker for this comparison.

**Prepared owner feedback (no native action taken):** “F1 Plan/Month has four current desktop/narrow Day/Night captures, exact source/build/routes/CSS viewports and empty-lease state, and a side-by-side comparison with the six applicable source PNGs. I found no obvious scene/theme mismatch. The only explicit F1 provenance field missing from these retained artifacts is actual browser DPR: 2× output pixels per CSS pixel is not an observed DPR. Please use any already-retained effective browser-context record to add that value; otherwise leave DPR unverified for the existing independent visual disposition. The JPEG degraded flag is an oversized-PNG compression fallback, not a separate clean-PNG requirement. Empty data and the inner-scrolling month are recorded limits; the original criteria do not require seeded-event parity or every month row in one screenshot. This is a Plan/Month visual disposition only and does not accept E1/Q1 or replace the original independent review.”

This review identifies the first remaining F1 item; it does not self-accept F1 or E1. The existing independent judgment-profile review remains the single original return condition for the complete E1 packet, and should return one exact verdict and any unmet original criteria rather than requesting another review round absent new evidence.

## Exact evidence identities

- `ACCEPTANCE.md`: SHA-256 `298366865fbcfcb1c6b3189e08afe72a83730be1ecfa558abb996f0a61d42862`.
- `QUEUE.md`: SHA-256 `0304fd3bc82b5a47f413915b4efbf5a65dd913a6cbff070fb3ef5750fdbc43d0`.
- `receipt.json`: SHA-256 `8572868d17d56263a81978a416278cfecbed65630be12982c16a130639347bde`.
- Desktop case `latest.json` / `timeline.json`: SHA-256 `96c409e2e31eccedf4758814437f2c6e9c821e339cce6c0d2fa2bfd50a8e32ba` / `61e40a98c88ae01f47b1613e1910afba01cc10efa6449978f5a26b559cb625ac`.
- Narrow case `latest.json` / `timeline.json`: SHA-256 `5c1d6fb184b69db95517db3034189acf19aae94e8e8c52fc6dcc8435fc90669a` / `80ed334e015b61665797dfc5f7b801c6b41033d648707dae59c0ea560f89426c`.
- Screenshot fallback source `scenarios/browser-automation-studio/playwright-driver/src/telemetry/screenshot.ts`: SHA-256 `9d438b44610267dbc0f84a83f4bad941193e42974bd3ce791843a4e32ecbc50b`.
- Screenshot limit/default/override source `scenarios/browser-automation-studio/playwright-driver/src/config.ts`: SHA-256 `397b2808f9551560c41a529955a550edc5f150b04570aa3cf07f847673f46b89`.
- Source-art manifest `scenarios/personal-planner/docs/reference/mockups/visual-reference-manifest.json`: SHA-256 `e7978155ffe7dee58c1d5650534e143cbf2e8fba1bb20eede16627fddb2ba06c`.
- Protected seed outputs live under the parent run's `private/planner-e1-seed/`; raw token material is intentionally excluded from this review.
