# Native Signal / Studio integration checkpoint

## Phase-six decoder / priority font-readiness checkpoint — 2026-09-15

Bounded W3 repair for finding 236d4d2c and the parent's live-browser font report.
The scenario-work-ladder scoped route and writing-standards evidence separation
govern this record. No parent harness, package/Vite/proto, assets, backend or
canonical documentation changes were made.

### Priority font repair

PublicLanding no longer awaits global document.fonts.ready or fails on every
global loadingerror. It collects the locally declared PresentationSans,
PresentationArchivo and Presentation Mono families actually selected by computed
styles inside .presentation-page. It loads their selected style/weight/text through
FontFaceSet.load, requires returned loaded faces, and watches only used families
for late failure. Missing/failed/unloaded required fonts fail closed; unrecognized
or absent presentation typography with the Font Loading API also fails closed.
The 15-second image/font deadline, listener cleanup and immediate global readiness
invalidation remain. Tests without a Font Loading API retain the existing jsdom
path. There is no external-font stub or blanket failure suppression in production.

PublicLanding's 29 tests include unrelated Space Grotesk and unused local-face
failures, a permanently pending global fonts.ready, all three used local families
failing, rejected/missing/unloaded faces, late failure and timeout cleanup. The
parent owns the same-origin capture-header repair and real-site browser rerun;
this checkpoint does not claim that live rerun.

### Decoder retirement and consumer migration

LandingConfigResponse now has one schema/type authority and exactly presentation
(required generated message), pricing, downloads and fallback. The generated
descriptor retains fields pricing=3, downloads=4, fallback=7, presentation=10.
The old response branch, normalizers for header/intro offers, fabricated variant/
sections/header defaults and duplicate response interface are removed. Both
network and bootstrap paths fail closed when presentation is missing. Bootstrap
uses the generated strict JSON decoder. Full page/display/fixtures/assets/actions/
diagnostics survive by identity; owner arrays are not sorted or reinterpreted.
Fallback comes from the retained response field, not invented metadata.

Public context exposes the diagnostics' resolved slug; private variant metadata
is not fabricated. Admin runtime labels fall back to that real slug when a private
name is absent. Admin snapshot/VariantSection/header/section-anchor types and SEO
inputs remain because recovery still uses them. Standalone getPlans and downloads
normalizers are unchanged. Tests compare generated public pricing against the
same getPlans result and preserve topups, amounts/currency/intros, web launch,
storefronts, exact download row/artifact identity and entitlement fields.

The admin coming-soon consumer now calls the existing private branding owner,
instead of reading retired public config.branding. Unconfirmed reads/writes disable
the toggle and display explicit reload feedback. It has no public provider refresh
or automatic writes. IMPORTANT EXISTING OWNER GAP: shared/api/branding.ts still
calls message.toJson(), masked by the old branding_pb ambient declaration in
vite-env.d.ts. Installed generated messages are plain v2 messages. Those files
were not changed in this slice. The parent needs to authorize/coordinate that
generated-codec correction before claiming the live coming-soon workflow works.
The hook regressions exercise typed owner behavior via its existing service seam,
not a live admin API. No demo/public-branding fallback was added.

Source search found no production consumers of public-landing/services/
navigation.service.ts. That helper and its test file were retired. They remain
recoverable from HEAD (both git cat-file checks passed). The shared section-anchor
helper stays for private recovery. The root BAS preservation gate passed before
and after: 1071 fields, 9 recovered payloads, 5 assets, 12 immutable source captures,
no sourceCaptureLiveDrift. Existing missing BAS installer/screenshot/thumbnail
evidence and historical Hero/Features component drift remain explicitly reported
by that authority; this slice does not qualify them.

### Exact source inventory (relative to ui/src)

```text
shared/api/landing.ts
shared/api/landing.test.ts
shared/api/types.ts
shared/api/schemas/landing.schema.ts
app/providers/LandingVariantContext.ts
app/providers/LandingVariantProvider.tsx
app/providers/LandingVariantProvider.test.tsx
app/providers/landingPresentationBootstrap.ts
surfaces/admin-portal/components/RuntimeSignalStrip.tsx
surfaces/admin-portal/components/RuntimeSignalStrip.test.tsx
surfaces/admin-portal/hooks/useComingSoonToggle.ts
surfaces/admin-portal/hooks/useComingSoonToggle.test.tsx
surfaces/admin-portal/routes/AdminAnalytics.tsx
surfaces/public-landing/routes/PublicLanding.tsx
surfaces/public-landing/routes/PublicLanding.test.tsx
surfaces/public-landing/presentation/publicTestFixtures.ts
surfaces/public-landing/presentation/DownloadPage.tsx (required-DTO access only)
surfaces/public-landing/presentation/DownloadPage.test.tsx
surfaces/public-landing/presentation/webLaunch.test.tsx
surfaces/public-landing/presentation/README.md
surfaces/public-landing/services/navigation.service.ts (deleted)
surfaces/public-landing/services/navigation.service.test.ts (deleted)
```

### Reproduction and observed focused output

From scenarios/landing-page-business-suite/ui:

```sh
DOM_PRINT_LIMIT=200 ./node_modules/.bin/vitest run src/shared/api/landing.test.ts src/shared/api/downloads.test.ts src/shared/api/variants.test.ts src/app/providers/LandingVariantProvider.test.tsx src/app/providers/useLandingVariant.test.tsx src/app/routes/publicRoutes.test.tsx src/shared/hooks/useMetrics.test.tsx src/shared/ui/SEOHead.test.tsx src/shared/lib/headerConfig.test.ts src/shared/lib/sections.test.ts src/surfaces/public-landing/presentation src/surfaces/public-landing/routes/PublicLanding.test.tsx src/surfaces/public-landing/routes/CheckoutPage.test.tsx src/surfaces/admin-portal/hooks/useComingSoonToggle.test.tsx src/surfaces/admin-portal/components/RuntimeSignalStrip.test.tsx src/surfaces/admin-portal/services/variant.service.test.ts src/surfaces/admin-portal/routes/Customization.test.tsx src/surfaces/admin-portal/routes/AdminAnalytics.test.tsx src/surfaces/admin-portal/routes/VariantEditor.test.tsx src/surfaces/admin-portal/presentation/PresentationAdminPage.test.tsx
./node_modules/.bin/tsc --noEmit
./node_modules/.bin/eslint src/shared/api/landing.ts src/shared/api/landing.test.ts src/shared/api/types.ts src/shared/api/schemas/landing.schema.ts src/app/providers/LandingVariantContext.ts src/app/providers/LandingVariantProvider.tsx src/app/providers/LandingVariantProvider.test.tsx src/app/providers/landingPresentationBootstrap.ts src/surfaces/admin-portal/hooks/useComingSoonToggle.ts src/surfaces/admin-portal/hooks/useComingSoonToggle.test.tsx src/surfaces/admin-portal/components/RuntimeSignalStrip.tsx src/surfaces/admin-portal/components/RuntimeSignalStrip.test.tsx src/surfaces/admin-portal/routes/AdminAnalytics.tsx src/surfaces/public-landing/routes/PublicLanding.tsx src/surfaces/public-landing/routes/PublicLanding.test.tsx src/surfaces/public-landing/presentation/publicTestFixtures.ts src/surfaces/public-landing/presentation/DownloadPage.tsx src/surfaces/public-landing/presentation/DownloadPage.test.tsx src/surfaces/public-landing/presentation/webLaunch.test.tsx
git diff --check
```

From repository root, before and after retirement:

```sh
python3 scripts/tests/test_bas_preservation.py
```

```text
Test Files  28 passed (28)
Tests       494 passed (494)
Start at    06:34:25
Duration    10.00s
Whole-UI typecheck: exit 0
Scoped ESLint: exit 0
Diff check: exit 0
Preservation gate before/after: passed / passed
```

Expected negative-case validation logs and the missing-provider ErrorBoundary
diagnostic appear in the passing run. No full Test Genie certification, dependency
operation, production write, team activation or live browser result is claimed.

## Phase-four terminal — external playback and mounted local preview, 2026-09-15

This terminal supersedes the incomplete 05:49 checkpoint below. Generated artifact
5a0c741e172e42163c9dcd35650cb849044b02d0d95825d9b04595ce18aeb16f is installed and
used directly. No generated file, ambient shim or descriptor bypass was added.
This bounded W3 slice uses the scenario-work-ladder scope and writing-standards
evidence separation; it does not certify the full scenario or start phase six.

The generated playback field now decodes into the finite product-demo renderer.
ProductDemo.test.tsx imports all 29 cases directly from the backend-owned
api/internal/presentation/testdata/playback-url-cases.json. Accepted URLs produce
the exact same privacy-parameter canonical URLs as Go. Raw authority validation
rejects explicit ports, including :443 and a bare colon before URL normalization.
Positive Vimeo IDs have 1–20 digits with no leading zero. Released poster closure,
click-only external loading, keyboard focus, configured caption/error copy,
15-second frame deadline and revision/source teardown are covered. Signal/Studio
JSON fixtures remain unchanged. No native video or new media owner was added.

PresentationEditorRoute now defaults to a 300ms debounced, authenticated, read-only
local document preview. The exported previewPresentationDocument(client, input,
signal) adapter sends generated Preview.document field 5, never revision, through
the existing session/no-store Connect transport with a 30-second deadline. Inputs
are the complete generated document plus explicit variantSlug, route and locale.
Each changed identity immediately hides the old preview and aborts its request.
Invalid syntax, owner validation, mismatched private diagnostics and auth loss
fail closed. Ephemeral response identity must be a matching requested/resolved
64-character lowercase SHA256 revision, with exact variant/route and locale.
The UI does not invent this identity or persist the document.

Saved-revision inspection remains an explicit separate mode and accepts only the
loaded revision or a retained published revision. Dirty source is retained during
that read. Draft Save, explicit confirmed Publish/Rollback and generation guards
remain separate operations. No automatic write occurs. The mounted preview uses
RevisionPreview and the same PresentationPage, never the public provider or
transaction actions. Preview-scoped container rules adapt content to its column;
the editor stacks below 1200px and uses two columns above it. Embedded previews
retain the focusable skip destination without nesting another main landmark.
Both explicit capture-boundary booleans remain on every presentation root.

The separately authorized CheckoutPage terms fix uses commerce-owner bundle.name
when nonempty and generic selected-plan copy otherwise. It changes no payment
call or checkout behavior. Tests use Orion Workspace and an unnamed bundle.

### Complete source inventory for this phase

Paths below are relative to ui/src. Historical earlier-slice edits in the shared
worktree are not newly claimed by this phase. The Go corpus is consumed, not edited.

```text
shared/api/productPresentation.ts
surfaces/admin-portal/components/AdminLayout.test.tsx
surfaces/admin-portal/routes/SectionEditor.test.tsx
surfaces/admin-portal/presentation/PresentationEditorRoute.tsx
surfaces/admin-portal/presentation/PresentationEditorRoute.test.tsx
surfaces/admin-portal/presentation/PresentationAdminPage.test.tsx
surfaces/admin-portal/presentation/PresentationLocalPreview.test.tsx
surfaces/admin-portal/presentation/RevisionPreview.tsx
surfaces/admin-portal/presentation/usePresentationEditor.ts
surfaces/admin-portal/presentation/useDocumentPreview.ts
surfaces/admin-portal/presentation/useDocumentPreview.test.tsx
surfaces/admin-portal/presentation/productPresentation.test.ts
surfaces/admin-portal/presentation/testFixtures.ts
surfaces/admin-portal/presentation/presentationEditor.css
surfaces/admin-portal/presentation/editorReview.tsx
surfaces/admin-portal/presentation/README.md
surfaces/public-landing/presentation/ProductDemo.tsx
surfaces/public-landing/presentation/ProductDemo.test.tsx
surfaces/public-landing/presentation/videoPlayback.ts
surfaces/public-landing/presentation/videoTestFixtures.ts
surfaces/public-landing/presentation/types.ts
surfaces/public-landing/presentation/contract.ts
surfaces/public-landing/presentation/registry.tsx
surfaces/public-landing/presentation/resourceClosure.ts
surfaces/public-landing/presentation/presentation.css
surfaces/public-landing/presentation/PresentationPage.tsx
surfaces/public-landing/presentation/PresentationPage.test.tsx
surfaces/public-landing/presentation/decode.ts
surfaces/public-landing/presentation/decode.test.tsx
surfaces/public-landing/presentation/commerce.test.tsx
surfaces/public-landing/presentation/productMarks.test.tsx
surfaces/public-landing/presentation/preview.tsx
surfaces/public-landing/presentation/visual-check.mjs
surfaces/public-landing/presentation/README.md
surfaces/public-landing/routes/CheckoutPage.tsx
surfaces/public-landing/routes/CheckoutPage.test.tsx
```

### Exact focused validation commands and output

Run from scenarios/landing-page-business-suite/ui:

```sh
DOM_PRINT_LIMIT=200 ./node_modules/.bin/vitest run src/surfaces/public-landing/presentation src/surfaces/public-landing/routes/PublicLanding.test.tsx src/surfaces/public-landing/routes/CheckoutPage.test.tsx src/surfaces/admin-portal/presentation src/surfaces/admin-portal/components/AdminLayout.test.tsx src/surfaces/admin-portal/routes/SectionEditor.test.tsx
./node_modules/.bin/tsc --noEmit
./node_modules/.bin/eslint src/shared/api/productPresentation.ts src/surfaces/admin-portal/presentation src/surfaces/admin-portal/components/AdminLayout.test.tsx src/surfaces/admin-portal/routes/SectionEditor.test.tsx src/surfaces/public-landing/presentation src/surfaces/public-landing/routes/CheckoutPage.tsx src/surfaces/public-landing/routes/CheckoutPage.test.tsx
git diff --check
PRESENTATION_EDITOR_ONLY=1 PRESENTATION_PLAYWRIGHT_MODULE=/home/matthalloran8/Vrooli/node_modules/.pnpm/playwright@1.62.1/node_modules/playwright node src/surfaces/public-landing/presentation/visual-check.mjs
PRESENTATION_VIDEO_ONLY=1 PRESENTATION_PLAYWRIGHT_MODULE=/home/matthalloran8/Vrooli/node_modules/.pnpm/playwright@1.62.1/node_modules/playwright node src/surfaces/public-landing/presentation/visual-check.mjs
```

```text
Test Files  19 passed (19)
Tests       389 passed (389)
Start at    06:19:02
Duration    7.47s
ProductDemo 86; decode 37; mounted local preview 6; preview hook 22;
generated client 5; CheckoutPage 22. Remaining focused suites also passed.
Whole-UI typecheck: exit 0. Scoped ESLint: exit 0. Diff check: exit 0.
Browser editor: 4 viewports (320/390/768/1440), no horizontal overflow,
desktop columns/mobile stacking, invalid-edit hiding, zero preview axe violations.
Browser video: 16 provider/layout/viewport cases, zero axe violations,
no pre-click provider requests, keyboard activation, poster failure, visible caption.
```

Browser evidence directories: /tmp/lpbs-presentation-review-sZlAiG (editor) and
/tmp/lpbs-presentation-review-6LahLZ (video), each with report.json and screenshots.
Desktop/mobile editor and video screenshots were inspected. The isolated editor
uses synthetic generated API responses; its existing remote admin font stylesheet
is stubbed offline and explicitly reported (8 requests). The local presentation
fonts load from bundled files. Private editor font delivery is not qualified.

Limits: no actual provider playback/subtitle, live authenticated preview, real
Save/Publish/Rollback, production release or full Test Genie/branch-threshold claim.
Provider requests are blocked after activation in the browser harness; iframe load
means embedded, not playing. Synthetic poster/art is not a BAS playback proof.
No parent-owned backend, server, package/Vite, capture scripts, canonical docs,
dependency or team state was changed. Existing public/admin mounting is retained.
Requested model: native gpt-6-astra; this handoff does not independently attest the
runtime model. Phase-six legacy decoder/protocol retirement remains parent-owned
and is not started here.

Backend coordination still needed beyond the 29 shared cases: the final read of
validation.go found that youtubePlaybackID also accepts watch?v= on
www.youtube-nocookie.com, although the agreed watch-host list excludes that host.
CanonicalPlaybackURL checks parsed.Fragment rather than a raw #, so a bare trailing
# also passes Go. The UI rejects both, with two additional regressions. Beauvoir
owns the parser/corpus corrections; UI policy was not loosened to match them.

## External video / unsaved preview checkpoint — 2026-09-15, 05:49 local

Scoped W3 implementation against the updated PRODUCT-PRESENTATION target. This is
an intermediate checkpoint, not a complete generated-wire integration or release
qualification. No native video, media owner, dependency, backend, public-capture
script, package/Vite configuration, production or team mutation was made.

The existing product-demo registry now has a recorded branch with typed external
playback props and released same-origin PNG poster closure. Only exact configured
YouTube/Vimeo URL forms produce canonical embeds; provider, layout and all visible
and accessibility copy are finite/required. Fixtures and recorded sources are
mutually exclusive. Poster rendering never requests a provider thumbnail or SDK.
Activation mounts the iframe with autoplay disabled, strict-origin referrers and
restricted permissions. Caption remains visible; resource error/15-second frame
timeout shows configured unavailable copy. Revision/source changes unmount the
player. An iframe load is called embedded, never playing or playback-qualified.
Provider error pages cannot reliably be inferred from cross-origin iframe load.

PresentationPage now emits explicit data-presentation-preview and
data-presentation-fallback booleans from decoded diagnostics for all public/private
views. Four combinations have direct DOM regressions. The legacy AdminLayout
breadcrumb test now expects Presentation; typed-route breadcrumb coverage was
added. SectionEditor tests use the canonical provider helper. Four pure prop/SVG
test files have narrow provider-free-exception reasons, as does the new video
suite. No test threshold was changed.

useDocumentPreview is implemented and independently tested but NOT mounted yet.
It debounces 300ms, aborts superseded reads, hides stale results synchronously by
document/route/locale/variant identity, checks private diagnostics, reports owner
validation failures and retains source. It has no persistence operations. The
remaining editor work is the generated RPC adapter and desktop side-by-side/mobile
stacked mounting while preserving explicit saved-revision inspection and CAS writes.

### Schema coordination / remaining integration

Source proto now declares PresentationProductDemo.playback field 8 and
PresentationPlayback string fields provider=1, external_url=2, layout=3,
play_label=4, caption=5, unavailable_label=6. PreviewPresentationRequest uses
document=5 (NOT preview_document). Installed generated modules still lack both
fields at this checkpoint. Parent must finish generation/SDA refresh; UI will then
map generated playback directly and call the existing authenticated Preview RPC
with document instead of revision. No ambient declarations or descriptor bypass
were introduced. Parent/core must also register recorded in Go validVariants and
provide exact ephemeral diagnostic identity conventions and URL-form policy.

### Complete source inventory for this checkpoint

Relative to ui/src; other earlier slices are inventoried separately below.

- surfaces/public-landing/presentation/ProductDemo.tsx (new)
- surfaces/public-landing/presentation/ProductDemo.test.tsx (new)
- surfaces/public-landing/presentation/videoPlayback.ts (new)
- surfaces/public-landing/presentation/videoTestFixtures.ts (new, tests/dev only)
- surfaces/public-landing/presentation/types.ts
- surfaces/public-landing/presentation/contract.ts
- surfaces/public-landing/presentation/registry.tsx
- surfaces/public-landing/presentation/resourceClosure.ts
- surfaces/public-landing/presentation/presentation.css
- surfaces/public-landing/presentation/PresentationPage.tsx
- surfaces/public-landing/presentation/PresentationPage.test.tsx
- surfaces/public-landing/presentation/decode.ts (fallback diagnostics only so far)
- surfaces/public-landing/presentation/decode.test.tsx
- surfaces/public-landing/presentation/commerce.test.tsx
- surfaces/public-landing/presentation/productMarks.test.tsx
- surfaces/public-landing/presentation/preview.tsx (isolated video review entry)
- surfaces/public-landing/presentation/visual-check.mjs (isolated bounded video checks)
- surfaces/public-landing/presentation/README.md
- surfaces/admin-portal/components/AdminLayout.test.tsx
- surfaces/admin-portal/routes/SectionEditor.test.tsx
- surfaces/admin-portal/presentation/useDocumentPreview.ts (new, not mounted)
- surfaces/admin-portal/presentation/useDocumentPreview.test.tsx (new)

Signal/Studio JSON fixtures are unchanged. The synthetic video fixture's existing
art poster and sample URLs are explicitly test inputs, not a released BAS video.

### Evidence and honest limits

Focused combined command (ui/):

```text
./node_modules/.bin/vitest run src/surfaces/public-landing/presentation src/surfaces/public-landing/routes/PublicLanding.test.tsx src/surfaces/admin-portal/presentation src/surfaces/admin-portal/components/AdminLayout.test.tsx src/surfaces/admin-portal/routes/SectionEditor.test.tsx
Test Files  1 failed | 16 passed (17)
Tests       1 failed | 321 passed (322)
Start at    05:48:04
Duration    6.51s
```

The sole failure is the intentional Go/UI finite variant comparison: UI now
implements recorded; core registration is still pending. The test was not disabled
or weakened. ProductDemo: 55 passed; useDocumentPreview: 22 passed; PublicLanding:
23 passed; PresentationPage: 21 passed; AdminLayout: 12 passed. Whole UI typecheck,
scoped ESLint and git diff --check passed. This does not claim a rerun of the
parent-owned complete Test Genie/unit-policy suite or its global branch threshold.

Bounded isolated Chromium command:

```text
PRESENTATION_VIDEO_ONLY=1 PRESENTATION_PLAYWRIGHT_MODULE=/home/matthalloran8/Vrooli/node_modules/.pnpm/playwright@1.62.1/node_modules/playwright node src/surfaces/public-landing/presentation/visual-check.mjs
PRESENTATION_EVIDENCE=/tmp/lpbs-presentation-review-MnybAZ
16 provider/layout/viewport cases passed; no pre-click provider requests;
keyboard activation/focus, caption persistence, failed-poster isolation passed;
16 axe audits: zero violations; zero page errors.
```

Read report.json and view video-youtube-split-1440.png and
video-youtube-stacked-320.png in that directory (both visually inspected). Provider
requests are blocked after click by the harness: NO playback/subtitle availability
claim is made. Parent owns real provider playback and publication evidence.

## Font-failure capture readiness — 2026-09-15, 05:26 local

Scoped W3 negative-case repair after the completed admin migration. FontFaceSet.ready
fulfillment is no longer treated as proof of successful font loading: errored faces
make the page unavailable before capture readiness or weighted exposure. A mounted
loadingerror listener also clears both global readiness indicators and the rendered
page after a late failure, stays active after initial readiness, and is removed on
unmount. Later loadingdone cannot restore a failed page. The existing 15-second
bound, image-failure handling and test DOMs without the Font Loading API remain.

Complete source inventory for this follow-up (relative to ui/src):

- surfaces/public-landing/routes/PublicLanding.tsx — face-status check and lifecycle listener.
- surfaces/public-landing/routes/PublicLanding.test.tsx — fulfilled-ready/failed-face,
  late failure/listener cleanup, no-API regressions and realistic timeout fixture.
- surfaces/public-landing/presentation/README.md — this evidence record.

Before repair the two new failure regressions failed (21 passed, 2 failed), with
captureReady incorrectly remaining true. Final focused output:

```text
./node_modules/.bin/vitest run src/surfaces/public-landing/routes/PublicLanding.test.tsx src/surfaces/public-landing/presentation/usePresentationExposure.test.tsx
✓ usePresentationExposure.test.tsx (13 tests) 24ms
✓ PublicLanding.test.tsx (23 tests) 266ms
Test Files  2 passed (2)
Tests       36 passed (36)
Start at    05:25:58
Duration    1.07s
```

Whole UI tsc --noEmit, ESLint on both changed TSX files, and git diff --check passed.
This is focused implementation evidence, not new browser/font release qualification;
parent owns the previously reported real-font/axe evidence. No design, assets,
retirement, backend, Node/server, dependencies or team state changed. PM remains disabled.

## Phase 6 admin migration / gated retirement — 2026-09-15, 05:20 local

Scoped W3 migration complete. The existing protected legacy section route now
redirects to /admin/presentation/:variantSlug, retaining explicit variant selection
for numeric/key/new bookmarks. Customization's primary edit action opens the typed
document; a separate Metadata action preserves existing variant forms, weights,
axes, header configuration and raw metadata workflows. VariantEditor content links,
legacy attention/focus links, and admin/dashboard content-resume links open the
same typed document. No first-section lookup, public random configuration fetch,
or public preview window is used to select a private document.

Preview actions lead to the saved-revision controls and existing RevisionPreview
(native PresentationPage). Saving remains draft-only; publication and rollback
retain expected_generation, validation, explicit confirmation and dirty guards.
Private variant resume uses existing landing_admin_experience storage, never the
public visitor identity. Authenticated legacy-bookmark integration tests cover
variant selection, publication/rollback controls, private preview, no automatic
mutation, and absence of public config/metrics calls. Generic code gains no product
narrative or default product.

PlanPreview is now a private owner-facts view, reusing formatOwnerPrice and existing
billing labels. It shows configured prices plus unsaved owner form labels/features,
has no transaction handlers, filters demo/hidden plans, and has an honest empty
state. It no longer imports PricingSection or injects a three-card marketing layout.
Existing pricing form/service workflows remain unchanged and covered by 85 tests.
This does not republish private owner metadata as public marketing content.

### Retirement / recovery evidence

Authoritative gate: python3 scripts/tests/test_bas_preservation.py.
Recovery doctrine: repo-root docs/internal/BAS-PRESERVATION.md.
Source-byte manifest: .vrooli/presentation-preservation/source-captures.json
(relative to the scenario). All captures remain untouched. Every preserved field,
payload and asset keeps its inventory recovery reference; no source capture was
deleted. Recovery for the removed clean tracked tests/admin helpers is git commit
54b62f1f82c728e5f96059b227dcb540b897efd5 plus the original paths below.

All three retirement batches ran the canonical gate immediately before and after
deletion. Each of those six checks passed with the same preserved-profile digest:
11411b7923ceb2564935ea09710fc0c9c3655471efcaf89a41899b237d827c6e.
Each verified 1,071 fields, 9 payloads, 5 asset references and 12 source captures.
A final post-retirement gate also passed. The canonical receipt reports expected
Hero/Features live-source absence as componentSourceDrift; the recovery bytes still
verify. The gate is not a second renderer, public publication or asset qualification.

Retired files were checked clean in git before deletion. Their desired replacement
behavior was tested before retirement; obsolete implementation-specific tests were
replaced by typed editor/privacy/navigation/owner-facts regressions, with existing
native registry/resource/exposure/download tests retained.

Remaining preserved limitations: BAS installer bytes, BAS-specific screenshots and
the referenced video thumbnail remain unlocated. BAS remains private/draft/disabled,
and the PM team remains enabled:false. No backend, API composition, Node/server,
dependencies, deployment, descendants or team state was changed in this slice.
Legacy variant metadata/header/raw-snapshot APIs and private pricing demo-form
facilities were deliberately not retired; parent/API ownership remains unchanged.

### Complete source inventory

Updated (relative to ui/src):

- app/routes/adminRoutes.tsx
- surfaces/admin-portal/routes/SectionEditor.tsx
- surfaces/admin-portal/routes/SectionEditor.test.tsx
- surfaces/admin-portal/components/plans/PlanPreview.tsx
- surfaces/admin-portal/components/plans/PlanPreview.test.tsx
- surfaces/admin-portal/hooks/useCustomizationPage.ts
- surfaces/admin-portal/hooks/useCustomizationPage.test.tsx
- surfaces/admin-portal/hooks/useAdminHome.ts
- surfaces/admin-portal/hooks/useAdminHome.test.tsx
- surfaces/admin-portal/config/navigation.utils.ts
- surfaces/admin-portal/config/navigation.utils.test.ts
- surfaces/admin-portal/config/adminPages.ts
- surfaces/admin-portal/config/adminPages.test.ts
- surfaces/admin-portal/presentation/PresentationEditorRoute.tsx
- surfaces/admin-portal/presentation/PresentationAdminPage.tsx
- surfaces/admin-portal/presentation/PresentationAdminPage.test.tsx
- surfaces/admin-portal/routes/Customization.tsx
- surfaces/admin-portal/routes/Customization.test.tsx
- surfaces/admin-portal/routes/VariantEditor.tsx
- surfaces/admin-portal/routes/VariantEditor.test.tsx
- surfaces/admin-portal/routes/LandingDashboard.tsx
- surfaces/admin-portal/routes/LandingDashboard.test.tsx
- surfaces/admin-portal/routes/BrandingSettings.test.tsx
- surfaces/admin-portal/services/README.md
- surfaces/public-landing/presentation/README.md

Retired (relative to ui/src):

- surfaces/public-landing/sections/HeroSection.tsx
- surfaces/public-landing/sections/FeaturesSection.tsx
- surfaces/public-landing/sections/PricingSection.tsx
- surfaces/public-landing/sections/DownloadSection.tsx
- surfaces/public-landing/sections/FAQSection.tsx
- surfaces/public-landing/sections/CTASection.tsx
- surfaces/public-landing/sections/FooterSection.tsx
- surfaces/public-landing/sections/TestimonialsSection.tsx
- surfaces/public-landing/sections/VideoSection.tsx
- surfaces/public-landing/sections/HeroSection.test.tsx
- surfaces/public-landing/sections/FeaturesSection.test.tsx
- surfaces/public-landing/sections/PricingSection.test.tsx
- surfaces/public-landing/sections/DownloadSection.test.tsx
- surfaces/public-landing/sections/CTASection.test.tsx
- surfaces/public-landing/sections/VideoSection.test.tsx
- surfaces/public-landing/sections/publicLandingSections.test.tsx
- shared/lib/fallbackLandingConfig.ts
- shared/lib/fallbackLandingConfig.test.ts
- surfaces/admin-portal/hooks/useSectionForm.ts
- surfaces/admin-portal/hooks/useSectionForm.test.tsx
- surfaces/admin-portal/controllers/sectionEditorController.ts
- surfaces/admin-portal/controllers/sectionEditorController.test.ts
- surfaces/admin-portal/components/SectionEditorComponents.tsx
- surfaces/admin-portal/components/SectionEditorComponents.test.tsx

### Focused validation

Whole UI tsc --noEmit passed after retirement. ESLint passed for changed source/test
files, and git diff --check passed. Final focused command (ui working directory):

```sh
./node_modules/.bin/vitest run src/surfaces/admin-portal/presentation src/surfaces/admin-portal/routes/SectionEditor.test.tsx src/surfaces/admin-portal/routes/Customization.test.tsx src/surfaces/admin-portal/routes/VariantEditor.test.tsx src/surfaces/admin-portal/routes/LandingDashboard.test.tsx src/surfaces/admin-portal/routes/BrandingSettings.test.tsx src/surfaces/admin-portal/components/plans/PlanPreview.test.tsx src/surfaces/admin-portal/hooks/useCustomizationPage.test.tsx src/surfaces/admin-portal/hooks/useAdminHome.test.tsx src/surfaces/admin-portal/hooks/useBillingForm.test.tsx src/surfaces/admin-portal/services/pricing.service.test.ts src/surfaces/admin-portal/config/adminPages.test.ts src/surfaces/admin-portal/config/navigation.utils.test.ts src/surfaces/public-landing/presentation src/surfaces/public-landing/routes/PublicLanding.test.tsx
```

Final output (navigation test emitted before the remaining output chunk):

```text
✓ src/surfaces/admin-portal/config/navigation.utils.test.ts (4 tests) 3ms
✓ src/surfaces/admin-portal/presentation/PresentationAdminPage.test.tsx (9 tests) 750ms
     ✓ mounts the existing layout and explicit variant selection without public assignment or preview metrics  384ms
 ✓ src/surfaces/admin-portal/hooks/useCustomizationPage.test.tsx (34 tests) 1807ms
 ✓ src/surfaces/admin-portal/hooks/useBillingForm.test.tsx (31 tests) 1730ms
 ✓ src/surfaces/admin-portal/hooks/useAdminHome.test.tsx (30 tests) 1624ms
 ✓ src/surfaces/admin-portal/routes/BrandingSettings.test.tsx (12 tests) 812ms
 ✓ src/surfaces/public-landing/presentation/DownloadPage.test.tsx (28 tests) 347ms
 ✓ src/surfaces/admin-portal/presentation/PresentationEditorRoute.test.tsx (18 tests) 650ms
 ✓ src/surfaces/public-landing/presentation/PresentationPage.test.tsx (17 tests) 312ms
 ✓ src/surfaces/public-landing/routes/PublicLanding.test.tsx (20 tests) 220ms
 ✓ src/surfaces/public-landing/presentation/decode.test.tsx (30 tests) 224ms
 ✓ src/surfaces/admin-portal/routes/Customization.test.tsx (3 tests) 143ms
 ✓ src/surfaces/admin-portal/routes/LandingDashboard.test.tsx (4 tests) 122ms
 ✓ src/surfaces/public-landing/presentation/webLaunch.test.tsx (35 tests) 128ms
 ✓ src/surfaces/admin-portal/routes/VariantEditor.test.tsx (5 tests) 118ms
 ✓ src/surfaces/public-landing/presentation/commerce.test.tsx (11 tests) 106ms
 ✓ src/surfaces/admin-portal/presentation/RevisionPreview.test.tsx (4 tests) 68ms
 ✓ src/surfaces/admin-portal/components/plans/PlanPreview.test.tsx (3 tests) 56ms
 ✓ src/surfaces/admin-portal/routes/SectionEditor.test.tsx (4 tests) 28ms
 ✓ src/surfaces/public-landing/presentation/usePresentationExposure.test.tsx (13 tests) 19ms
 ✓ src/surfaces/public-landing/presentation/productMarks.test.tsx (11 tests) 19ms
 ✓ src/surfaces/admin-portal/presentation/productPresentation.test.ts (4 tests) 16ms
 ✓ src/surfaces/admin-portal/services/pricing.service.test.ts (54 tests) 8ms
 ✓ src/surfaces/public-landing/presentation/links.test.ts (22 tests) 3ms
 ✓ src/surfaces/admin-portal/config/adminPages.test.ts (3 tests) 3ms

 Test Files  25 passed (25)
      Tests  409 passed (409)
   Start at  05:20:39
   Duration  11.28s (transform 773ms, setup 793ms, import 5.13s, tests 9.32s, environment 5.27s)
```

During final validation, a breadcrumb helper name collided with an existing local
routeLabel variable. Regression/typecheck caught it; renaming the decoder fixed the
collision before the passing run above. No release/browser-capture claim is made
for this admin-only slice.

## Canonical download types unblocked — 2026-09-15, 05:01 local

Narrow parent-granted overlap removed only the stale download_pb and
shared/downloads_pb ambient module blocks from vite-env.d.ts; unrelated declarations
remain unchanged. No manual assetId declaration or type assertion replaces them.
The generated metadata fields are JsonObject (google.protobuf.Struct). Admin app
and asset metadata now pass through the existing protobuf fromJsonString/toJson
helpers with StructSchema, preserving nested objects, arrays, false, numbers and
null. Response metadata already has the generated JsonObject type and needs no
invented wrapper. Canonical schema round-trip regression added.

This supersedes the selector checkpoint's typecheck blocker. Whole-UI tsc --noEmit
passes. Focused tests: 135 passed across six files, start 05:01:02, duration 2.99s:
DownloadPage 28; DownloadSection 16; shared/api/downloads 16; useDownloadsForm 33;
public downloads.service 32; admin downloads-workflow 10. Existing negative tests
emit expected validation stderr. Scoped lint and diff checks pass after replacing
one unnecessary asymmetric test matcher with a literal expected object.

Complete source inventory for this followup: ui/src/vite-env.d.ts,
ui/src/shared/api/downloads.ts, ui/src/shared/api/downloads.test.ts, and this README.
The removed ambient declarations are recoverable from git/prior checkpoint history;
they are superseded by the installed generated module, not removed functionality.
No legacy renderer retirement, SectionEditor migration or PlanPreview changes yet:
phase 6 waits for Wegener's exact source-snapshot gate and Beauvoir's caller/test
handoff. No API/main/Node changes, dependency work, descendants or team activation.

## Exact download selector — 2026-09-15, 04:53 local

Scoped W3 followup: the new second-Linux-row/invalid-ID regressions reproduced the
old platform-only limitation (13 failed, 15 passed), then passed after implementation.
This supersedes the historical same-platform selector limitation below.

Compatible API signature:
`requestDownload(appKey, platform, legacyUser?, selector?: { assetId: number })`.
The legacy user remains ignored; omitted selector preserves existing wire input.
Explicit selectors must be positive safe integers and serialize through the installed
generated optional bigint assetId field. The ID is delivery Asset.ID, never ArtifactID.
The chooser allows multiple rows for one platform, sends the selected unique ID,
and refuses missing/invalid/duplicate row IDs with explicit neutral UI reasons.
Returned app/bundle/platform/version/ID/checksum checks and session/entitlement
gates remain enforced. A stale/not-found response requires reload; no catalog URL
or alternate platform row is substituted. Tests verify second Linux ID 13 is sent
instead of ArtifactID 900, wrong/stale rows expose no link, and legacy callers work.

Complete source inventory for this small followup (relative to ui/src):

- shared/api/downloads.ts and downloads.test.ts
- surfaces/public-landing/presentation/DownloadPage.tsx and DownloadPage.test.tsx
- surfaces/public-landing/presentation/systemUi.ts and README.md

Focused command/output:

```text
./node_modules/.bin/vitest run src/shared/api/downloads.test.ts src/surfaces/public-landing/presentation/DownloadPage.test.tsx src/surfaces/public-landing/sections/DownloadSection.test.tsx src/surfaces/public-landing/services/downloads.service.test.ts
DownloadPage.test.tsx         28 passed
downloads.test.ts            15 passed
DownloadSection.test.tsx     16 passed
downloads.service.test.ts    32 passed
Test Files 4 passed (4); Tests 91 passed (91)
Start 04:53:33; Duration 1.85s
```

Expected stderr comes from existing malformed admin-payload negative tests.
ESLint for all five changed source/test files and git diff --check passed.
Whole-UI tsc remains blocked by the parent-owned stale ambient download_pb module
in ui/src/vite-env.d.ts:27–56: its AuthorizeDownloadRequest at line 31 omits assetId,
shadowing the installed generated field. No shim, cast or alternate protocol was
added; parent notified to remove the stale declaration. Backend/host/dependency
files were not edited. No live authorization or production qualification claimed.

## Browser identity / rendered exposure / readiness — 2026-09-15, 04:45 local

This checkpoint supersedes earlier notes deferring the exposure seam. Installed
generated descriptors supply RecordPresentationExposure and diagnostics fields;
no dependency installation, protocol shim or backend/server edit was made here.

The shared helper exports getVisitorId, isValidVisitorId, visitorIdFromCookie and
VISITOR_ID_KEY from ui/src/shared/lib/visitorIdentity.ts. The exact ID grammar is
`^[A-Za-z0-9_-]{1,128}$`, without trimming/decoding IDs; duplicate named cookies are
rejected. Precedence: valid bootstrap visitor, single valid cookie, safe legacy
localStorage ID, stable memory/new random ID. Cookie writes use Path=/,
SameSite=Lax, Max-Age=31536000, and Secure on HTTPS. Matching cookies are not
rewritten. This is anonymous correlation, never authentication.

Bootstrap accepts visitorId, keeps the rendered server document without an initial
second selection, and permits explicit refresh/navigation using that same visitor.
useMetrics uses the provider identity. Protected routes/previews do not initialize
public IDs. GetLandingConfig remains a read. The generated exposure caller is
exported from shared/api/landing.ts with a ten-second transport timeout.
usePresentationExposure runs only in mounted PublicLanding, after fonts/eager images
are ready and document.visibilityState is visible. It sends exact revision, route,
locale, digest, weight fingerprint and WEIGHTED_VISITOR source once per mounted proof.
The owner remains authoritative for revalidation and cross-mount deduplication.
Explicit variants, private previews, missing proof and download workflows do not
write exposures. Transport failure cannot hide a marketing page; no client retry
or fabricated recorded count. No live exposure-write claim is made by these tests.

Readiness now has a 15-second deadline covering fonts and image decode/load, removes
load/error listeners and timers on completion/unmount, and fails closed without
fallback art. Late lazy-image failure immediately clears global captureReady and
sets experienceState unavailable. Configured links reject URL credentials, encoded
controls/backslashes/separators and dot traversal before browser normalization.
Explicit locale/variant scope follows only known internal root/detail/download
links, including the chooser's back link; weighted pages gain no variant parameter,
and launch/checkout URLs remain opaque. Existing asset basing is unchanged: DOM
tests verify /proxy/api/v1/... in both src and responsive srcset exactly once. This
is component evidence, not a live proxy-network or release-qualification claim.

Complete source inventory for this checkpoint (paths relative to ui/src):

- shared/lib/visitorIdentity.ts and visitorIdentity.test.ts (new)
- shared/hooks/useMetricsHook.ts and useMetrics.test.tsx
- shared/api/landing.ts and landing.test.ts (generated exposure caller/test only;
  concurrent pricing normalization retained)
- app/providers/LandingVariantContext.ts, LandingVariantProvider.tsx,
  LandingVariantProvider.test.tsx and landingPresentationBootstrap.ts
- surfaces/public-landing/presentation/usePresentationExposure.ts and
  usePresentationExposure.test.tsx (new)
- surfaces/public-landing/presentation/links.ts and links.test.ts (new test)
- surfaces/public-landing/presentation/publicIntegration.ts, DownloadPage.tsx,
  decode.test.tsx and this README.md
- surfaces/public-landing/routes/PublicLanding.tsx and PublicLanding.test.tsx

Focused command (ui working directory):

```sh
./node_modules/.bin/vitest run src/surfaces/public-landing/presentation src/surfaces/public-landing/routes/PublicLanding.test.tsx src/surfaces/admin-portal/presentation/RevisionPreview.test.tsx src/app/providers/LandingVariantProvider.test.tsx src/shared/lib/visitorIdentity.test.ts src/shared/hooks/useMetrics.test.tsx src/shared/api/landing.test.ts
```

Final focused output:

```text
useMetrics.test.tsx                    9 passed
LandingVariantProvider.test.tsx      14 passed
PresentationPage.test.tsx            17 passed
DownloadPage.test.tsx                17 passed
decode.test.tsx                      30 passed
webLaunch.test.tsx                   35 passed
PublicLanding.test.tsx               20 passed
commerce.test.tsx                    11 passed
RevisionPreview.test.tsx              4 passed
usePresentationExposure.test.tsx     13 passed
landing.test.ts                      21 passed
productMarks.test.tsx                11 passed
visitorIdentity.test.ts              20 passed
links.test.ts                        22 passed
Test Files 14 passed (14); Tests 244 passed (244)
Start 04:45:42; Duration 4.58s
```

Expected stderr: the existing missing-pricing negative test logs its validation
rejection. Whole-UI tsc --noEmit, scoped ESLint and git diff --check passed. An
expanded lint invocation including landing.test.ts also identified two existing
unnecessary optional chains in its pricing/download-normalization assertions
(lines 141/143); those parent-owned assertions were preserved and excluded from
the final scoped lint command. Existing PM team remains enabled:false.

Remaining: owner/server integration and release verification are parent-owned.
Same-platform release selection still requires authorized download asset identity
in the RPC (see delivery handoff below); UI honestly disables ambiguous options.
No descendants, broad suite, dependency changes, deployment or team activation.

## Shared head marks / sanitized owner pricing — 2026-09-15, 04:31 local

Parent server can directly import the dependency-free ES module:
`ui/src/surfaces/public-landing/presentation/productMarks.js`.
Exports: frozen productMarkPaths, frozen productMarkDrawing (viewBox, fill, stroke
width/caps/joins), and getProductMarkPath(unknown). The latter uses own-property
lookup and returns undefined for invalid/prototype keys; never select a default
product. ProductMark consumes this same grammar. productMarks.d.ts supplies local
TypeScript declarations, not a generated-protocol/ambient shim. A direct Node ESM
import verified all four paths, drawing geometry and unknown-key rejection.

For the parent's server helper under ui/server/:

```js
import { getProductMarkPath, productMarkDrawing } from '../src/surfaces/public-landing/presentation/productMarks.js';
```

Colors remain configured theme authority; the native letter-a mark uses accent,
other marks inherit primary ink. No manifest/favicon/apple head mutation was
performed here; parent owns runtime head rendering and packaging the shared module.
No new PWA subsystem, font/art generation, dependency or second path catalog.

Pricing now projects only stripe_price_id, amount_cents, currency, billing_interval
and intro_enabled into its pure renderer. It neither retains nor displays owner
plan names, arbitrary bundle/plan metadata, subtitles or feature marketing.
Card headings use explicitly configured purchase-action labels for that plan_ref;
without labels, only the source amount/billing is rendered. All narrative must
come from presentation configuration. A regression injects private owner names and
features and verifies neither enters the renderer join or DOM.

### Delivery projection handoff for Beauvoir

Needed catalog identity: asset id, app_key, bundle_key, platform, release_version,
checksum and requires_entitlement. Final authorized response must bind the same
identity and contain the strict-safe download URL; DownloadPage now also rejects
returned bundle mismatch. Catalog artifact URLs, update API keys and arbitrary
metadata are not needed. Safe web_url remains necessary for configured app-bound
web launch. Prefer exact uint64 handling when the selector seam lands; the existing
numeric adapter must not claim arbitrary-ID precision.

If multiple installers share platform/version, retain an explicit safe public
variant/filename/architecture label so choices can be distinguished. Do not ask
the UI to infer variant identity from private artifact URLs. The current protocol
still accepts only app/platform; same-platform alternatives remain disabled pending
the asset selector. Direct Beauvoir messaging was not resolved: no native message
tool was exposed, search returned unrelated portal/switchboard commands, and the
required discovery fallback returned no result. Parent was asked to relay these
fields or provide a direct worker address; no message to another recipient was sent.

Complete changed source inventory for this follow-up: productMarks.js,
productMarks.d.ts, productMarks.test.tsx, primitives.tsx, commerce.ts,
PricingCards.tsx, commerce.test.tsx, DownloadPage.tsx, DownloadPage.test.tsx,
README.md. All within the owned presentation folder. Shared concurrent changes
were preserved; provider/exposure integration remains OPEN pending parent schema.

Final focused command is the preceding web-launch command plus scoped lint over
the presentation directory. Direct Node import also passed.

```text
PresentationPage.test.tsx       17 tests 319ms
LandingVariantProvider.test.tsx 14 tests 370ms
decode.test.tsx                 29 tests 223ms
DownloadPage.test.tsx           17 tests 274ms
PublicLanding.test.tsx          15 tests 161ms
webLaunch.test.tsx              35 tests 142ms
commerce.test.tsx               11 tests 112ms
RevisionPreview.test.tsx         4 tests  77ms
productMarks.test.tsx           11 tests  18ms
Test Files 9 passed (9)
Tests     153 passed (153)
Start     04:31:40
Duration  3.53s
Whole-UI typecheck, scoped lint, git diff --check: exit 0
```

No server-head/live launch, installer-transfer, payment or exposure certification
is claimed. Existing PM team remains disabled.

## Web-only owner launch / ongoing provider integration — 2026-09-15, 04:26 local

The preserved delivery baseline in `.vrooli/fallback/fallback.json` identifies
`web-console` / `business_suite` with metadata.web_url `/app/web-console` and an
empty platform list. This was inspected as owner-intent evidence, NOT imported
as a runtime fallback or claimed to be a live database read. Existing public
ProtoDownloads and UI normalization preserve the typed metadata; a new generated
response test proves that path. The parent concurrently added the Opens owner
join; this UI does not manufacture that observation.

Public configured app-bound open actions now require a ready owner observation,
exactly one matching-bundle delivery row, an eligible app, and matching validated
metadata.web_url. Marketing detail routes, owner/row URL disagreements, absent,
disabled or ambiguous rows, private apps and unsafe URLs stay unavailable per
action. The ordinary root/detail renderer and the get-started/download chooser
both receive config.downloads through resolvePublicPresentation's optional fourth
argument. No slug-to-launch guess, alternate app catalog, installer inference,
download-to-open action conversion or label invention is present.

DownloadChooser accepts configured launchActions and resolvedActions; it displays
the web launch even with zero installers and separately says no installers are
currently available. The web-only workflow does not invoke download session checks
or show an irrelevant installer sign-in gate. Its owner-backed launch retains the
destination's existing authentication/entitlement authority. A provider-free
preview without owner joins remains visibly unavailable and effect-free.

safeOwnerHref accepts validated HTTPS or app-absolute paths, rejecting protocol-
relative URLs, raw/encoded backslashes and path separators, dot traversal,
credentials, controls/whitespace, unsafe schemes and fragment-only launches.
Opaque query tokens are preserved. Only app-relative destinations gain the proxy
basename; HTTPS destinations do not. The same strict helper now validates returned
authorized installer URLs; the permissive legacy sanitizeArtifactUrl helper is no
longer used by DownloadPage. Owner-side validation should match these rejection
cases, including absolute dot traversal and marketing-detail destinations.

### Confirmed server/UI workflow contract; exposure is still open

Browser route `/apps/:slug/download` requests/bootstrap-matches `/apps/:slug` with
the same raw variant/locale. Parent server can bootstrap that detail document on a
workflow refresh. A focused provider test verifies no second config selection.
Client head remains noindex and uses the underlying detail title/description and
detail canonical, exclusively from configured canonicalBaseUrl or the server's
presentation-canonical-base marker. No authority means no canonical. The workflow
does not introduce an alternate metadata owner. canonicalPresentationHref now
shares this calculation between PublicLanding and DownloadPage.

The provider's initial no-refresh behavior is an INTERIM safeguard, not completed
visitor/assignment/exposure integration. Parent/core are preparing a typed seam;
UI provider/bootstrap ownership remains ongoing pending that generated schema.
Do not treat this checkpoint, page views or initial pinning as exposure proof.
Release-selector authorization and safe sign-in return remain separate pending
owner contracts from the prior download slice.

Complete changed source inventory: presentation/publicIntegration.ts,
DownloadChooser.tsx, DownloadPage.tsx, systemUi.ts, webLaunch.test.tsx,
DownloadPage.test.tsx, README.md; public-landing/routes/PublicLanding.tsx;
app/providers/LandingVariantProvider.test.tsx and LandingVariantProvider.tsx
(interim-seam comment only). Shared API/auth/services, server, backend, schemas,
assets and dependencies were not edited. No deployment/team activation.

```sh
./node_modules/.bin/vitest run src/surfaces/public-landing/presentation src/surfaces/public-landing/routes/PublicLanding.test.tsx src/surfaces/admin-portal/presentation/RevisionPreview.test.tsx src/app/providers/LandingVariantProvider.test.tsx
./node_modules/.bin/tsc --noEmit
./node_modules/.bin/eslint src/surfaces/public-landing/presentation src/surfaces/public-landing/routes/PublicLanding.tsx src/app/providers/LandingVariantProvider.test.tsx
```

```text
PresentationPage.test.tsx       17 tests 421ms
LandingVariantProvider.test.tsx 14 tests 383ms
decode.test.tsx                 29 tests 262ms
DownloadPage.test.tsx           16 tests 333ms
PublicLanding.test.tsx          15 tests 178ms
webLaunch.test.tsx              35 tests 171ms
commerce.test.tsx               10 tests 164ms
RevisionPreview.test.tsx         4 tests  71ms
Test Files 8 passed (8)
Tests     140 passed (140)
Start     04:26:18
Duration  3.98s
Whole-UI typecheck, scoped lint, git diff --check: exit 0
```

This is generated-contract/component proof, not live launch, installer transfer,
authentication or experiment certification. Prior asset-placement browser evidence
below remains scoped to that run; no new screenshot claim is made for web launch.

## Asset placement checkpoint — 2026-09-15, 04:17 local

Parent review located a bounded W3 loss: ResolvedAsset retained crop/focal metadata,
but MediaAsset dropped it and CSS supplied placement. MediaAsset now retains a
finite `center | contain | cover` policy and copied, finite unit-square focal
coordinates. Unknown/empty policies and out-of-range/nonfinite focal coordinates
fail at resource projection, including generated-response decode. Values are not
clamped or replaced with defaults.

AssetImage is the only raw img renderer in this folder. It now applies inline
objectFit (`contain` for contain; `cover` for cover and legacy center) and
objectPosition (`100*x% 100*y%`) from released metadata. The configured focal point
is authoritative even for legacy center. This covers hero prints, backdrop main
images/thumbnails, artifact image/HTML/video frames, generic Visual and closing
art. No frame dimensions, closing copy panel, CSS design, fonts or artwork changed.
Parent's core currently accepts any nonempty policy; align its publication surface
policy to this finite mapping. This UI check does not certify release placement.

Width srcsets require matching reference surface, asset surface, MIME, crop policy,
focal coordinates and aspect ratio, plus configured sizes and unique widths.
Distinct crop/art-direction alternatives remain referenced resources but are NOT
flattened into width candidates. Their selection needs explicit art-direction
policy; the renderer does not infer breakpoints or force a different composition.

Complete changed source inventory (this folder): resources.ts (finite MediaAsset),
resolvedResources.ts (validation/projection/alternative filtering), primitives.tsx
(DOM placement), decode.test.tsx (18 new placement/alternative regressions plus
positive srcset DOM assertions), visual-check.mjs (computed-style verification),
README.md (handoff). Shared concurrent edits/commits were preserved.

```sh
./node_modules/.bin/vitest run src/surfaces/public-landing/presentation/decode.test.tsx src/surfaces/public-landing/presentation/PresentationPage.test.tsx src/surfaces/admin-portal/presentation/RevisionPreview.test.tsx
./node_modules/.bin/tsc --noEmit
./node_modules/.bin/eslint src/surfaces/public-landing/presentation/{resources.ts,resolvedResources.ts,primitives.tsx,decode.test.tsx,visual-check.mjs}
```

```text
decode.test.tsx             29 tests 229ms
PresentationPage.test.tsx   17 tests 332ms
RevisionPreview.test.tsx     4 tests  65ms
Test Files 3 passed (3)
Tests     50 passed (50)
Start     04:17:42
Duration  1.44s
Whole-UI typecheck, scoped lint, git diff --check: exit 0
```

Browser command remains the visual-check.mjs command below. Final report:
`/tmp/lpbs-presentation-review-6qFbvz/report.json`; 12 viewport captures, 12 axe
contexts with zero violations, no overflow/missing images/runtime errors.
All eight Signal/Studio viewport checks confirm computed placement equals the
configured inline placement. Signal mobile and Studio desktop closing captures
were inspected: opaque copy panels remain intact. Isolated build: 109 modules,
726ms. No live asset release/publication qualification claimed; PM team disabled.

## Current download/commerce checkpoint — 2026-09-15, 04:09 local

Bounded W3 implementation under the next operator assignment. Scenario-work-ladder
kept this at the established UI/service boundary; writing-standards distinguishes
implemented behavior from pending live/owner evidence. Requested native model is
gpt-6-astra; runtime attestation is not independently verified.

### Route and data contract for parent

- `/apps/:slug/download` mounts DownloadPage. It requests the canonical
  `/apps/:slug` document with the requested locale/variant and uses its app_key
  to select the scoped config.downloads entry. No first-app or first-platform
  selection. The workflow is noindex, carries its document route/revision, and
  does not mint a new presentation mode or canonical URL.
- Parent should resolve each configured download action to this app-specific
  workflow href. Existing unique action observations drive every CTA location;
  a focused test covers repeated configured CTAs and proxy basenames. UI does
  not rewrite a self-detail destination into an invented ready action.
- DownloadPage reuses UserAuthProvider.refreshSession and the existing
  shared/api/downloads.requestDownload(app, platform). Authorization derives
  identity from the session. No email-as-identity, auth fork, catalog-URL bypass,
  cached credential or direct storage-client path is added. The owner decision
  determines entitlement; sign-in alone does not claim subscription access.
- Current AuthorizeDownloadRequest contains only app/platform. A selected unique
  platform is authorized, then app/platform/release/id/checksum are compared
  before exposing the returned safe link. Ambiguous same-platform options remain
  visible but disabled. Full release-specific selection is NOT complete.
- Recommended owner extension: optional immutable `asset_id` (uint64) and
  `expected_release_version` (string) on the same authorization RPC. Owner must
  bind selection to app/platform, enforce session entitlement and reject changed
  records. Preserve exact IDs through the generated client; do not hide uint64
  precision loss in the current numeric adapter. Parent owns protocol/backend
  changes and SDA refresh. No shim or speculative protocol write was made here.
- Current sign-in route `/auth/login` lacks an ordinary safe return-to workflow
  parameter. UI reuses that route and tells visitors to return/recheck session;
  automatic return remains an owner/auth integration gap, not an invented OAuth
  redirect. An owner-validated return-to seam is recommended separately.
- Parent server must serve the download workflow route (noindex) and understand
  that its document identity is `/apps/:slug`, not a new published page. Existing
  exact bootstrap matching is unchanged. Exposure/cross-route revision accounting
  remains deferred to the parent's typed seam; no false download/exposure counts.

`PresentationPage({presentation, resolvedActions?, resolvedPricing?})` stays
provider-free. PublicLanding supplies resolvePricing(presentation, config.pricing).
Pricing `plan_refs` match owner stripe_price_id, preserve configured order, and
show source amount_cents/100 with Intl currency/locale formatting and explicit
month/year/one-time billing. No annual-to-monthly discount, synthetic free plan,
variable-price guess or checkout URL inference. Missing, duplicate, hidden,
wrong-bundle and invalid prices are unavailable. Intro-enabled records show the
standard source amount with neutral qualification; checkout owns actual intro
terms. Purchase actions remain configured and owner-resolved.

DownloadChooser is a controlled provider-free view for previews/dev; DownloadPage
owns public session and authorization effects. Selection changes/unmount/session
identity changes suppress stale authorization links. A ready URL is not a claim
that bytes transferred. Native keyboard selection, release facts/notes, explicit
guest/access-denied/outage states and responsive cards use centralized neutral
system strings; product title/description come only from canonical page config.

### Complete source inventory for this slice

All paths relative to `ui/src/`:

| Files | Responsibility |
| --- | --- |
| surfaces/public-landing/presentation/DownloadPage.tsx, DownloadPage.test.tsx | Public app workflow, existing session authorization, identity/release checks and 16 regressions |
| surfaces/public-landing/presentation/DownloadChooser.tsx | Controlled native platform/release chooser; no effects |
| surfaces/public-landing/presentation/commerce.ts, PricingCards.tsx, commerce.test.tsx | Owner plan join/formatting, pure cards and 10 regressions including repeated CTA destinations |
| surfaces/public-landing/presentation/commerceFixtures.ts | Synthetic review/test owner records; no production import |
| surfaces/public-landing/presentation/PresentationPage.tsx, registry.tsx | Optional owner-price prop through finite compact pricing renderer |
| surfaces/public-landing/presentation/systemUi.ts, presentation.css | Neutral system vocabulary, responsive accessible chooser/cards |
| surfaces/public-landing/presentation/preview.tsx, visual-check.mjs | Isolated commerce review and four additional viewport/axe captures |
| surfaces/public-landing/routes/PublicLanding.tsx | Supplies scoped live owner pricing to pure renderer |
| app/providers/LandingVariantProvider.tsx, LandingVariantProvider.test.tsx | Canonical detail-document mapping for workflow route |
| app/routes/publicRoutes.tsx, publicRoutes.test.tsx | Explicit app download route |
| surfaces/public-landing/presentation/README.md | Evidence and exact remaining owner contracts |

Read/reused existing DownloadSection, download service/helpers, generated download
contract, session auth/invocation and pricing owner joins. No legacy mixed-product
marketing was copied. No shared service, API, auth, admin, backend/proto, checkout,
server, dependencies or asset content edits. PM team observed disabled.

### Focused validation output

From `ui/`:

```sh
./node_modules/.bin/vitest run src/surfaces/public-landing/presentation src/surfaces/public-landing/routes/PublicLanding.test.tsx src/app/providers/LandingVariantProvider.test.tsx src/app/routes/publicRoutes.test.tsx src/surfaces/admin-portal/presentation/RevisionPreview.test.tsx src/shared/api/downloads.test.ts src/surfaces/public-landing/services/downloads.service.test.ts
./node_modules/.bin/tsc --noEmit
./node_modules/.bin/eslint src/surfaces/public-landing/presentation src/surfaces/public-landing/routes/PublicLanding.tsx src/app/providers/LandingVariantProvider.tsx src/app/providers/LandingVariantProvider.test.tsx src/app/routes/publicRoutes.tsx src/app/routes/publicRoutes.test.tsx
PRESENTATION_PLAYWRIGHT_MODULE=/home/matthalloran8/Vrooli/node_modules/.pnpm/playwright@1.62.1/node_modules/playwright node src/surfaces/public-landing/presentation/visual-check.mjs
```

```text
PresentationPage.test.tsx       17 tests 316ms
LandingVariantProvider.test.tsx 13 tests 364ms
DownloadPage.test.tsx           16 tests 255ms
PublicLanding.test.tsx          15 tests 146ms
commerce.test.tsx               10 tests 103ms
RevisionPreview.test.tsx         4 tests  68ms
decode.test.tsx                 11 tests  60ms
publicRoutes.test.tsx            4 tests  53ms
downloads.test.ts                7 tests  14ms
downloads.service.test.ts       32 tests   5ms
Test Files 10 passed (10)
Tests     129 passed (129)
Start     04:09:41
Duration  3.42s
Whole-UI tsc --noEmit: exit 0
Scoped ESLint: exit 0, no findings
git diff --check: exit 0
```

Existing download transport malformed-payload tests log expected validation stderr.
An earlier new pricing fixture incorrectly used unsupported variant `cards`; it
was corrected to canonical `compact` before this passing run. No registry widening.
Final browser report `/tmp/lpbs-presentation-review-rZ7oWz/report.json`: 12 viewport
captures, 12 axe contexts with zero violations, no overflow/missing images/runtime
errors. Commerce mobile/desktop screenshots were inspected; mobile select clipping
was corrected and recaptured. Full filename remains visible in release facts.
Isolated build 109 modules / 714ms. Evidence is synthetic component/browser proof,
not real installer bytes, live entitlement, payment, publication or exposure proof.

## Current mounting/bootstrap checkpoint — 2026-09-15, 03:52 local

Public `/` and `/apps/:slug` now mount the canonical native renderer. The old
PublicLanding first-app/legacy marketing resolver is removed. Unknown/private
details stay unavailable/not-found at their URL; absent typed presentation does
not load review fixtures. Protected routes do not request public config or emit
public page metrics. Authenticated admin mounting is documented in the admin README.

Bootstrap contract: application/json script `lpbs-presentation-bootstrap` contains
`{request:{route,locale,variant},config,canonicalBaseUrl}`. `config` is raw generated
protobuf JSON, parsed strictly through exported `parseLandingConfigJson` in
`shared/api/landing.ts`; generated responses use exported `decodeLandingConfig`.
The provider only consumes an exact request identity with public diagnostics.
It retains the initial resolved variant/revision without an automatic or manual
config refresh. The public transport does not offer a revision-pin/exposure seam.
There is no initial assignment RPC; ordinary page-view metrics are NOT evidence
of a legitimate owner assignment/exposure. Exposure accounting and preserving
assignment/revision across later route requests remain explicitly deferred to the
parent. Navigating to a different request currently uses the ordinary public RPC.
Do not certify experiment exposure or cross-route revision continuity from this UI.

Canonical URLs use configured `canonicalBaseUrl` or the server-owned
`presentation-canonical-base` meta marker plus diagnostics.resolved_route, never
window Host guesses. Unset authority produces no canonical. Page title/description
are authoritative; no arbitrary asset becomes an OG image. Root diagnostics expose
revision/mode/digest/route/locale and capture readiness after critical images/fonts.
Proxy basenames apply to internal links/assets and preview navigation; Vite emits
relative hashed font URLs. Source-unqualified assets fail closed.

`resolvePublicPresentation` in `publicIntegration.ts` validates public identity and
maps unique ready owner action observations without deriving transaction targets.
Unsafe/unavailable actions remain disabled. A download href aimed at an app-detail
route is rejected as a workflow self-loop. Building the actual platform selection
workflow is outside this checkpoint. Private preview maps local navigation only.

### Complete source inventory added/changed by mounting slice

Paths below are relative to `ui/src/`; prior renderer/editor inventories follow.

| Files | Responsibility |
| --- | --- |
| app/providers/LandingVariantProvider.tsx, LandingVariantContext.ts, landingPresentationBootstrap.ts, LandingVariantProvider.test.tsx | Route-scoped requests, race isolation, exact bootstrap pin, public context |
| app/routes/publicRoutes.tsx, publicRoutes.test.tsx, adminRoutes.tsx | Root/detail/not-found and authenticated admin routes |
| shared/api/landing.ts, landing.test.ts, types.ts, schemas/landing.schema.ts | Generated presentation plumbing, strict bootstrap codec; existing pricing retained |
| surfaces/public-landing/routes/PublicLanding.tsx, PublicLanding.test.tsx | Canonical rendering, configured SEO, diagnostics and public page metrics |
| surfaces/public-landing/presentation/publicIntegration.ts, PublicPresentationState.tsx, systemUi.ts, publicTestFixtures.ts | Public boundary, neutral failure UI, synthetic test data only |
| surfaces/public-landing/presentation/registry.tsx, exhibits.tsx, presentation.css, README.md | Capture landmarks, proxy-safe fonts, evidence |
| surfaces/admin-portal/presentation/PresentationAdminPage.tsx, PresentationAdminPage.test.tsx | Existing authenticated layout, explicit variant selection, dirty navigation |
| surfaces/admin-portal/presentation/PresentationEditorRoute.tsx, RevisionPreview.tsx, RevisionPreview.test.tsx | Proxy-safe private native preview, no transactions |
| surfaces/admin-portal/presentation/README.md, DISPLAY-FIELD-HANDOFF.md | Current integration handoff |
| surfaces/admin-portal/components/AdminLayout.tsx, config/navigation.ts | Guarded logout and presentation navigation entry |

No App, server, backend, protobuf, checkout/Stripe, vite-env, dependencies or asset
content changes in this mounting slice. No deployment or team activation; PM team
observed disabled. Shared worktree edits were preserved.

### Final focused output

Run from `ui/`:

```sh
./node_modules/.bin/vitest run src/app/providers/LandingVariantProvider.test.tsx src/app/routes/publicRoutes.test.tsx src/surfaces/public-landing/routes/PublicLanding.test.tsx src/surfaces/public-landing/presentation src/surfaces/admin-portal/presentation src/shared/api/landing.test.ts src/surfaces/admin-portal/components/AdminLayout.test.tsx src/surfaces/admin-portal/config/navigation.utils.test.ts
./node_modules/.bin/tsc --noEmit
```

```text
PresentationAdminPage.test.tsx    5 tests 546ms
PresentationEditorRoute.test.tsx 18 tests 697ms
PresentationPage.test.tsx        17 tests 331ms
LandingVariantProvider.test.tsx  12 tests 363ms
PublicLanding.test.tsx           15 tests 154ms
AdminLayout.test.tsx             11 tests 137ms
decode.test.tsx                  11 tests  59ms
RevisionPreview.test.tsx          4 tests  77ms
landing.test.ts                  20 tests  18ms
publicRoutes.test.tsx             3 tests  53ms
productPresentation.test.ts       4 tests  17ms
navigation.utils.test.ts          4 tests   4ms
Test Files 12 passed (12)
Tests     124 passed (124)
Start     03:52:51
Duration  4.78s
Whole-UI tsc --noEmit: exit 0
Scoped ESLint: exit 0
git diff --check: exit 0
```

Expected stderr: the existing missing-pricing regression logs PricingOverview
validation `_errors: ['Required']`. Scoped lint covered changed provider/routes,
both presentation directories, AdminLayout/navigation and API source; it excluded
existing landing.test.ts optional-chain lint findings at lines 134/136.
Browser rerun: `/tmp/lpbs-presentation-review-1Pkhe8/report.json`, Signal/Studio at
320/390/768/1440, no overflow/missing images/runtime errors, eight axe contexts
with zero violations, menu/Escape/conversation/artifact keyboard checks pass.
Isolated build: 103 modules, 1.49s. This is fixture-based browser evidence, not live
published-root, real authorized mutation, release qualification or exposure proof.

## Prior decoder checkpoint (historical scope/evidence)

Bounded W3 UI implementation, 2026-09-15. Scenario-work-ladder scoped the work
to implementation; writing-standards keeps observed evidence separate from
integration still owned by the parent. No scenario certification is claimed.

## Stable entry boundary

```tsx
import { decodeProductPresentation } from '../../../shared/api/productPresentation';
import { PresentationPage } from './presentation';

const presentation = decodeProductPresentation(config.presentation);
<PresentationPage presentation={presentation} resolvedActions={ownerJoins} />;
```

Call the decoder only after the parent handles an absent config.presentation.
It accepts the installed generated ResolvedProductPresentation, not arbitrary
JSON or an administrative Document. It uses explicit typed camelCase-to-snake_case
projections and exhaustive block/fixture oneof switches, without any/schema casts.
The generated strict JSON parser is exposed as parseProductPresentation for
standalone fixture review and tests. Unknown JSON or retained binary fields,
kind/oneof disagreements, unsupported variants, missing required messages/copy,
and broken resource references fail closed.

Page.display is mandatory and inline. There is no alternate display prop or
runtime sidecar. Protobuf omitted false/empty fields retain protobuf defaults;
required display messages and labels are still checked. No product-name fallback,
route heuristic, membership selection, implicit ordering, provider lookup, or
runtime mockup import is present.

Owner action observations in the wire response are intentionally not promoted
into transaction handlers. Parent supplies resolvedActions keyed by
actionKey(action): [kind, app_key, plan_ref, target]. Ready entries contain href
or onActivate; unavailable entries contain reason. Unjoined transactions remain
disabled with configured reasons. Authorized admin preview uses the same decoder
and page internally and never supplies transaction joins.

The installed and repository shared product_presentation_pb.ts hashes matched:
e84c668f29c8407c922b472ab588e6a3f4d2a2f46418d21117189d0419302627.
Parent reports protocol-source digest
54092391239066175c121427d8664ffcc98c5c3981abf68100ff9819b19e86d3.
No additional dependency refresh is currently needed; this slice performed none.
Future descriptor changes must refresh @vrooli/proto-types through SDA.

## Canonical resource and composition rules

- Canonical fixtures/assets and page-owned display are the sole native demo
  content, label, and URL sources. Resource indexes are internal.
- Hero composition is independent of selected_app_keys (the capped slide set).
  Each configured hero app must be eligible in diagnostics.eligible_app_keys and
  have a resolved spotlight profile. Zero slide keys can still render two hero
  groups. Up to three hero groups preserve configured order.
- Device display.fixture_ref and hero display.hero_fixture_refs are explicit
  alternatives to visual_ref, exactly one per exhibit. App display also chooses
  exactly one fixture or asset; it never selects membership.
- Display block/app/fixture/asset keys, anchors, native backdrop images, artifact
  frames, capabilities, and responsive alternatives must resolve. Workspace
  labels and asset label entries must be configured; empty alt is allowed for
  deliberately decorative art.
- Responsive width srcsets require configured sizes and closed alternatives
  sharing surface, MIME, aspect ratio, crop policy and focal point. No dimensions
  or sizes are guessed. Different compositions/aspect ratios stay on the base
  asset pending explicit art-direction policy. Candidate order is preserved;
  repeated widths are not offered as competing browser choices.
- contract.ts matches Go validVariants; a focused source-consistency test checks
  the finite sets without relying on map iteration order.
- Review fixture JSON uses generated protobuf content envelopes and inline display.
  Fixtures have illustrative eligibility/release provenance, not publication
  evidence. Studio does not enable or publish an additional product.

## Native behavior retained

Signal centered hero and Studio editorial composition; workspace sessions, review,
terminal/conversation; keyboard artifact tabs and structured plan/HTML/storyboards;
configured voice/summary/qualification; native phone/workflow; roadmap, story,
catalog, closing, footer, finite FAQ and pricing-action blocks. No iframe,
executable markup, fake microphone/player, or fake transactional workflow.
Artwork stays bounded away from narrative headings. These review checks do not
qualify production copy-over-art contrast or release availability.

## Complete source inventory

| Files | Responsibility |
| --- | --- |
| PresentationPage.tsx, index.ts | Pure entry and exports |
| decode.ts, decode.test.tsx | Generated response projection, oneofs/defaults, closure and Go variant tests |
| types.ts, contract.ts | Renderer unions/actions and finite assertions |
| resources.ts, resolvedResources.ts, resourceClosure.ts | Canonical PageDisplay, native indexes, closed references, responsive srcsets |
| registry.tsx | Finite block rendering |
| exhibits.tsx, ArtifactExplorer.tsx, primitives.tsx, links.ts | Native demos, accessible interaction, configured media, safe navigation |
| presentation.css | Scoped responsive Signal/Studio styling |
| fixtures/signal.json, fixtures/studio.json | Inline generated-shape review configuration |
| preview.html, preview.tsx | Isolated unmounted review entry |
| PresentationPage.test.tsx, tsconfig.json, visual-check.mjs | Focused component/type/browser proof |
| README.md | Integration and evidence handoff |
| ui/public/presentation/*.png | Previously copied survey-relief, pale-moon, tidal-halftone art |
| ui/public/presentation/*.ttf, *-OFL.txt | Previously copied Archivo/Plex fonts and licenses |

This integration slice additionally changes the owned admin presentation
directory and new shared/api/productPresentation.ts; the admin README lists those
files. No App/router/PublicLanding/existing landing.ts/vite-env/backend changes.

## Focused evidence

From ui/:

```sh
./node_modules/.bin/vitest run src/surfaces/public-landing/presentation src/surfaces/admin-portal/presentation
./node_modules/.bin/tsc --noEmit -p src/surfaces/public-landing/presentation/tsconfig.json
./node_modules/.bin/tsc --noEmit -p src/surfaces/admin-portal/presentation/tsconfig.json
./node_modules/.bin/eslint src/surfaces/public-landing/presentation src/surfaces/admin-portal/presentation src/shared/api/productPresentation.ts
PRESENTATION_PLAYWRIGHT_MODULE=/home/matthalloran8/Vrooli/node_modules/.pnpm/playwright@1.62.1/node_modules/playwright node src/surfaces/public-landing/presentation/visual-check.mjs
```

Final focused output (03:24:39 local, 2026-09-15):

```text
PresentationPage.test.tsx        17 tests 414ms
PresentationEditorRoute.test.tsx 18 tests 790ms
RevisionPreview.test.tsx          3 tests  65ms
decode.test.tsx                  11 tests  73ms
productPresentation.test.ts       4 tests  17ms
Test Files 5 passed (5)
Tests     53 passed (53)
Duration  2.21s
Both focused TypeScript projects: exit 0
Scoped ESLint: exit 0, no findings
```

Browser report: /tmp/lpbs-presentation-review-ZGpy2t/report.json.
Both designs at 320/390/768/1440: no overflow, missing images, or runtime errors.
Eight axe contexts: zero violations. Mobile menu/Escape, conversation toggle,
artifact End/Home pass. Studio desktop hero and Signal mobile HTML-artifact
screenshots inspected after this decoder migration. Standalone Vite build:
103 modules, 754ms. Fonts resolve from the isolated public-asset interceptor.

Historical renderer evidence remains /tmp/lpbs-presentation-review-8ZIc2z/report.json.
Earlier broad Test Genie unit run 20260915-064546-a951a9d2 failed after 114 seconds
with 4 errors/74 warnings (Go command/framework-policy and broader findings).
That is historical advisory evidence, not a new run or attributed regression.
Parent reports the former whole-UI pricing type gap repaired; this slice runs
only its focused projects and does not claim whole-scenario certification.

Remaining parent work: production/router mounting, public config and owner-action
integration, real authenticated server preview/publication validation, and released
asset qualification. No dependencies, descendants, team activation or deployment.
The PM team was observed disabled. The separate branding inventory/output remains
/home/matthalloran8/.vrooli/plan-artifacts/efforts/aquila-configurable-product-presentation/findings/web-console-ui-branding.md.

## Real brand logo images with finite-mark fallback — 2026-09-16

Shell and app displays now accept an optional, same-origin brand image next to
the required finite mark: `brand_logo`/`brand_logo_alt`/`footer_brand_logo` on
the shell and `logo`/`logo_alt` on an app exhibit. The Mark stays required, so
an image that never loads still has a deterministic SVG fallback.

`BrandLogo` in `primitives.tsx` renders the configured image when present and
otherwise delegates to `ProductMark`. It rejects any non-relative logo
(`https:`, scheme-relative, `javascript:`) at render time, and the Go display
validator rejects the same references plus traversal and query/fragment. The
LPBS platform uses the Vrooli logo (`/public/logo.webp`, copied from
vrooli-onboarding); Aquila's exhibit uses the Aquila eagle
(`/public/apps/aquila.png`).

Server-side vocabulary: `PresentationShellDisplay.brand_logo`,
`brand_logo_alt`, `footer_brand_logo` and `PresentationAppDisplay.logo`,
`logo_alt` (proto regenerated), mirrored in `display.go`, `resources.ts`,
`decode.ts`, `PresentationPage.tsx`, `registry.tsx` and `presentation.css`.
Seed and every committed presentation revision carry the new fields. Adding a
scenario to the bundle resolves its name and icon through
`scripts/sync-bundle-catalog.mjs`.
