---
date: 2026-05-04
scenario: landing-page-business-suite
interactions:
  - cold landing page navigation
  - landing page scroll through public sections
  - navigate to checkout
  - navigate to feedback
  - navigate to user login
  - navigate to admin login
  - return to landing page
traces:
  before: /tmp/landing-page-business-suite/perf/trace.section.json
  after: /tmp/landing-page-business-suite/perf/trace.after.json
  capture_script: /tmp/landing-page-business-suite/perf/capture.js
status: fixed
related_skill_run: scenario-performance-audit
---

# Perf audit: user-facing views

## Framing

The request was to measure performance of `landing-page-business-suite`, focused on the landing page and other user-facing views. I audited the local scenario through Chrome headless at `1440x900`, using the standard scenario lifecycle and the profile-mode React/Vite build. The scripted interaction covered cold landing-page load, landing-page scroll, `/checkout`, `/feedback`, `/auth/login`, `/admin/login`, and a return to `/`.

## Methodology

- Profile-mode build verified: the served bundle contained `onProfilerRender` and route names.
- Capture script: `/tmp/landing-page-business-suite/perf/capture.js`
- Primary trace: `/tmp/landing-page-business-suite/perf/trace.section.json`
- Web-vitals sidecar: `/tmp/landing-page-business-suite/perf/trace.section.web-vitals.json`
- Follow-up trace after implementation: `/tmp/landing-page-business-suite/perf/trace.after.json`
- Bundle measured from `scenarios/landing-page-business-suite/ui/dist`.

## Per-component aggregation

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---:|---:|---:|---:|
| App | 25 | 60.7 | 2428 | 23999 |
| PublicLanding | 11 | 37.2 | 3382 | 23799 |
| LandingSection:pricing | 2 | 10.0 | 5001 | 8600 |
| LandingHeader | 4 | 7.8 | 1950 | 5600 |
| LandingSection:hero | 8 | 6.3 | 787 | 2399 |
| AdminLogin | 2 | 5.2 | 2601 | 4901 |
| Checkout | 5 | 4.5 | 900 | 2200 |
| Feedback | 2 | 3.7 | 1851 | 3601 |
| LandingSection:features | 4 | 3.6 | 900 | 1400 |
| LandingSection:testimonials | 2 | 2.2 | 1100 | 1300 |
| UserLogin | 2 | 1.5 | 750 | 1100 |
| LandingSection:footer | 2 | 1.2 | 600 | 700 |
| LandingSection:cta | 2 | 0.9 | 450 | 600 |
| LandingSection:faq | 2 | 0.8 | 400 | 500 |

## After-change aggregation

| component | before count | after count | before total(ms) | after total(ms) | before avg(μs) | after avg(μs) | delta(ms) |
|---|---:|---:|---:|---:|---:|---:|---:|
| App | 25 | 28 | 60.7 | 57.7 | 2428 | 2061 | -3.0 |
| PublicLanding | 11 | 13 | 37.2 | 35.9 | 3382 | 2762 | -1.3 |
| LandingSection:pricing | 2 | 2 | 10.0 | 10.5 | 5001 | 5250 | +0.5 |
| LandingHeader | 4 | 4 | 7.8 | 6.5 | 1950 | 1625 | -1.3 |
| LandingSection:hero | 8 | 8 | 6.3 | 5.8 | 787 | 725 | -0.5 |
| Checkout | 5 | 5 | 4.5 | 3.6 | 900 | 720 | -0.9 |
| Feedback | 2 | 2 | 3.7 | 3.1 | 1851 | 1550 | -0.6 |
| AdminLogin | 2 | 2 | 5.2 | 2.5 | 2601 | 1250 | -2.7 |

## Long-task summary

| metric | before | after | delta |
|---|---:|---:|---:|
| count | 0 | 0 | 0 |
| total(ms) | 0.0 | 0.0 | 0.0 |
| max(ms) | 0.0 | 0.0 | 0.0 |

Paint and load signals from the local headless run were low before and after. Before: first paint 24 ms, FCP 36 ms, LCP 236 ms. After: first paint 16 ms, FCP 32 ms, LCP 248 ms. These are local-loopback numbers, so they should not be treated as production network timings.

## Findings

- **Public landing render is the only meaningful React hotspot.** Evidence: `PublicLanding` committed 11 times for 37.2 ms total, with one 23.8 ms commit. Other public/auth routes were small: checkout 4.5 ms total, feedback 3.7 ms, user login 1.5 ms, admin login 5.2 ms.
- **The pricing section is the heaviest landing subsection.** Evidence: `LandingSection:pricing` committed twice for 10.0 ms total, 5.0 ms average, and 8.6 ms max. The relevant work is concentrated in `ui/src/surfaces/public-landing/sections/PricingSection.tsx`: `renderTier` is recreated in render at lines 199-294, plan filtering/sorting/tier building runs in render at lines 296-360, fallback tier array construction runs at lines 379-440, and `paddedTiers` depends on `fallbackTiers`, which is rebuilt every render, at lines 472-479.
- **The landing header has non-trivial commit cost.** Evidence: `LandingHeader` committed four times for 7.8 ms total, 5.6 ms max. `PublicLanding.tsx` computes section ordering, downloads, header config, nav links, CTAs, and runtime metadata at lines 148-200, then renders a sticky/header subtree at lines 279-286 and 357 onward. Some of this work is expected after config load, but it will repeat when route-level state changes.
- **The initial public bundle is too large for a landing page.** Evidence: profile build produced a single JS file of 1,483,097 bytes uncompressed and 413 KB gzip, plus 77.9 KB CSS. The built bundle contains Monaco loader/editor code even though Monaco is only used by the admin variant editor. The static import chain starts with `ui/src/app/routes/adminRoutes.tsx` importing every admin route at lines 5-29; `VariantEditor.tsx` imports `@monaco-editor/react` at line 3. Because `App.tsx` imports the complete route tables eagerly, public visitors download admin/editor code on the landing page.
- **Landing scroll triggered repeated failed metric calls.** Evidence: the capture logged repeated `400 Bad Request` console errors during landing scroll and return to landing. Those errors did not create long tasks in this local trace, but they are noisy and can distort real-user monitoring. The likely call path is section CTA/scroll tracking through `useMetrics`; the payload or route context should be checked against the metrics API contract.

## Implemented outcome

- Public route, auth route, and admin route components are now loaded through `React.lazy` under the existing route table modules, with `Suspense` at the app route shell.
- Monaco remains statically imported by `VariantEditor`, but `VariantEditor` is now route-lazy, so Monaco loader/editor code moved out of the public landing entry and into the editor chunk.
- The normal production public entry chunk dropped from 1,285,103 bytes uncompressed / 346.47 KB gzip to 475,152 bytes uncompressed / 142.23 KB gzip.
- The profile-mode public landing path no longer includes the old single 1.48 MB JS bundle; public landing and pricing are separate route chunks.
- `PricingSection` now memoizes plan filtering/sorting/tier derivation and renders tier cards through a memoized `PricingTierCard`.
- Metrics tracking now sends `variant_slug` to match the Go API contract, deduplicates page-view and scroll-depth tracking across section-level hook users, and emits one page view plus at most four scroll-depth events per active variant/page. Browser validation after the fix produced five `/metrics/track` responses, all HTTP 201, with no console errors.

## Recommendations + outcome

| # | Recommendation | Status | Notes |
|---|---|---|---|
| 1 | Code-split routes with `React.lazy` and `Suspense`, especially admin routes and `VariantEditor`. | fixed | Public entry is now 475,152 bytes instead of 1,285,103 bytes in the normal build. |
| 2 | Split Monaco into an editor-only dynamic import. | fixed | Monaco text is absent from the public entry and present only in the `VariantEditor` chunk. |
| 3 | Memoize pricing derivations in `PricingSection.tsx`. | fixed | Filtering, sorting, tier derivation, fallback normalization, and tier selection are memoized/stable. |
| 4 | Extract and memoize pricing tier cards. | fixed | `PricingTierCard` is wrapped in `React.memo`; stable callbacks keep card props predictable. |
| 5 | Investigate metric `400` responses during scroll/page activity. | fixed | Root cause was `variant_id` vs API-required `variant_slug`, plus duplicate section hook trackers. Browser validation now shows five 201 responses and no console errors. |

## New dependencies

- None.
