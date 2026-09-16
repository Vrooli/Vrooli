# Configurable product presentation

Status: implementation target authorized by the operator on 2026-09-15. This
document defines expected behavior, not release certification. The preserved
effort is `/home/matthalloran8/.vrooli/plan-artifacts/efforts/aquila-configurable-product-presentation`;
its `requirements.json` and `findings/implementation-blueprint.md` retain the
full acceptance boundary. This document is the repository implementation seam.

## Identity and commercial boundary

Keep `web-console` as Aquila's canonical delivery/entitlement app key and use
`aquila` as its public slug. Do not mint a duplicate deliverable. The production
default has only Aquila public. Backdrop Studio is an asset provider and an
explicit private draft app until an operator publishes its commercial profile.
Browser Automation Studio is disabled/private and retains exact recovered copy,
assets, installer metadata, and source digests beside improved presentation copy.
Use the existing Vrooli Business Suite identity as editable bundle configuration;
this implementation does not authorize a commercial rebrand or bundle expansion.

Use the selected Signal and Studio designs from `sources/mockups/v2/` in the
effort. They establish typography, proportions, product views, artwork, responsive
behavior and feature depth. Do not substitute the old download-card shell.
Local speech does not mean all AI processing is local. Remote computers,
Android control and iPhone control remain coming soon with no active control CTA.

## Ownership and interfaces

The existing experimentation ConfigStore owns versioned presentation documents,
drafts and immutable published revisions. A pure `internal/presentation` domain
owns types, validation, normalization and deterministic resolution; it does not
read files or import commerce/delivery. Landing aggregation joins the resolved
page to existing delivery/commerce owners. No second app catalog database is
introduced. Public and admin transport use generated Connect contracts.

The canonical initial document is the declarative JSON file
`api/internal/presentationseed/recommended-signal-studio.json`. It is embedded
in the API binary so packaged installations do not depend on repository paths.
Startup seeds only a missing draft for each existing variant, through the same
ConfigStore compare-and-swap writer. Existing drafts and publications are never
overwritten. Seeding does not publish an app, qualify a capability, or release
artwork; those transitions require the normal publication verifier.
The composition root binds a new draft to the commerce owner's configured bundle
key (the packaged owner currently uses `business_suite`), not the design template's
illustrative `business-suite` key. This is an explicit bootstrap input, not an
alias or catalog inference. Existing revisions and historical BAS facts are not
rewritten when owner configuration changes.

One document contains `schema_version`, `bundle`, `apps`, `pages`, `assets`,
and localized `strings`. Bundle fields include `key`, `name`, `app_order`,
`max_app_slides`, `page_id`, `empty_page_id`, `default_locale` and `locales`.
Apps are keyed by canonical app key, with `slug`, `name`, `enabled`, `visibility`
(`public`/`private`), `publication` (`draft`/`published`), `page_id`, `tagline`,
`description`, `capabilities`, and optional `preservation_ref`.
Page fields include `id`, `locale`, `title`, `description`, `theme`, `navigation`,
`blocks`, `footer` and typed `display`. Each block has `id`, `kind`, `version`, `variant`, and
typed `content`; arrays are authoritative order. Version 1 renderers are finite.

`Page.display` owns locale-specific shell labels, finite brand marks, asset
alt/sizes labels, fixture display labels, block decoration/heading-break data,
and app exhibit references. The shell and each app exhibit may also carry an
optional real brand image: the shell accepts `brand_logo`, `brand_logo_alt` and
`footer_brand_logo`; an app exhibit accepts `logo` and `logo_alt`. The finite
`mark` stays required, so an image that fails to load still has a deterministic
fallback. Logo references are validated as relative same-origin asset paths
with no traversal or query/fragment, never third-party or remote URLs. When a
logo is present the renderer shows the brand image and otherwise the mark.
These values share the immutable page revision and
content digest. No production sidecar or JSX defaults supply missing copy.
Resolved display tables contain only selected blocks, referenced public app
exhibits and their fixture/asset closure. Reference validation includes display
slots, not merely block content. Native device/bundle hero fixtures are explicit
alternatives to released visual references, never fabricated asset releases.

The initial vocabulary covers product-hero, bundle-hero, capability-strip,
product-story, product-demo, app-spotlights, artifact-explorer, voice-story,
device-story, capability-roadmap, pricing, closing-action, faq and footer.
Content contracts include every visible/accessibility label, fixture content,
media reference and action; generic renderers contain no Aquila or BAS copy.
Use validated theme tokens and finite visual variants, not arbitrary CSS/JSX.

`product-demo` also preserves the existing configurable video-demo obligation
(OT-P1-009). Its `recorded` variant selects `renderer_ref: video` and a typed
`playback` object: provider (`youtube` or `vimeo`), external URL, layout (`stacked`
or `split`), localized load-player label, caption and unavailable-player label.
Require a released `poster_ref` and player `alt_text`. Reject fixture_ref,
media_ref, unsupported providers/layouts and conflicting fields in this mode.
Fixture modes retain their required fixture_ref and do not accept ignored
playback/media fields. No arbitrary iframe HTML or configurable player options.
Normalize only exact HTTPS provider host/path/ID forms; reject credentials,
nondefault ports, unsafe characters and unrecognized query parameters. Use
canonical privacy-enhanced YouTube/Vimeo embeds only after keyboard or pointer
activation. Before activation, including poster failure, no iframe, provider
thumbnail, SDK, preconnect or provider/media request is allowed. Do not autoplay.
Keep the caption visible, support focus and failure states, tear down the player
on revision/source changes, and collapse both layouts on mobile. The public and
private renderer share these behaviors; private URLs must not become referrers.
A loaded iframe is not proof that the provider played the video or supplied
captions. Playback and subtitle availability need separate evidence. Native
video/WebVTT delivery is not represented by the PNG-only Backdrop asset owner;
do not add a new media-storage owner to close this legacy external-video gap.

Artifacts are typed plan/image/html-preview/video/audio/code/pdf examples with
configured filenames, captions, alt text, dimensions and safe references. Do not
execute HTML from marketing configuration. Voice examples contain transcripts,
summaries, labels, qualification text and waveform data. Capabilities have stable
IDs, status, localized benefit text, constraints and private owner evidence refs.
Publishing an available claim requires owner evidence; editors cannot promote a
roadmap item solely by changing its status label. Public responses exclude private
evidence, source paths, disabled narratives and credentials.

The request carries variant assignment, route, locale and optional preview
revision. The resolved response carries `schema_version`, `mode`, `scope`,
`app_key`, `page`, `selected_app_keys`, and `diagnostics`. Scope is bundle or app;
mode is empty, single-app, bundle or app-detail. App spotlights expand once into
ordered app-owned content with matching detail routes. Hero items reference
unique eligible app keys and supported exhibit kinds; multiple artwork panels in
one app group still represent one app. Hero composition capacity is separate from
the page-wide spotlight cap; editorial supports up to three groups, grid handles
larger sets. Zero hero items invents no app.

`selected_app_keys` is the capped spotlight selection. Public app summaries also
include any eligible apps referenced by the independently configured hero, so a
zero spotlight cap does not erase hero exhibits. Root draft preview simulates
the same public membership rules. Only an explicit authenticated app-detail
preview may inspect a disabled/private/draft profile; preview permission alone
does not turn those profiles into bundle members.

Diagnostics include requested/resolved route, variant and revision, locale,
bundle/app key, eligible app keys, fallback reason, preview state, block digest,
commerce snapshot and asset release refs. Hash normalized content, not timestamps
or route-only metadata. Root single-app and matching detail content hashes agree.

Public `GetLandingConfig` is read-only. The server-rendered document and browser
share the existing anonymous `metrics_visitor_id` identity so hydration and later
navigation do not re-randomize assignment. Only a displayed, eligible weighted
assignment may call `RecordPresentationExposure`; its variant, revision, route,
locale, content digest and weight fingerprint are rechecked before the existing
metrics owner deduplicates the write. Explicit variant URLs and admin previews
do not contribute exposures. These diagnostic values are not authorization
tokens. Metrics failure does not erase a successfully loaded presentation.

Legacy snapshots are administrator-only recovery data. The old public variant
and public-section REST routes are retired. Public experiment Connect methods
return only slug, normalized status and weight; header copy, descriptions, axes
and section payloads never bypass typed presentation publication. Authenticated
snapshot reads and exports retain their recovery content.

Request-scoped test routing must reach every downstream read and write. Forward
only the canonical development routing marker from the document server; never
forward private cookies or authorization to public configuration readers. An
absent or expired test lease must fail closed instead of falling back to primary
storage. Owner HTTP clients propagate that marker through the shared transport.

## Resolution and publication invariants

1. An app participates only if it is a member, enabled, published and public.
   Installer outages, process state and database order do not affect membership.
2. Zero eligible apps uses the configured empty page. One uses the app's entire
   page. Multiple use the bundle page and at most `max_app_slides` spotlights.
3. `/apps/:slug` uses the same app resolver as single-app root. Unknown, private,
   disabled or unpublished public routes return 404 without narrative leakage.
4. Reject duplicate block IDs, slugs, app-order keys, unknown references, unsafe
   actions, unsupported versions/kinds, invalid caps and duplicate spotlight
   collections. A zero cap is valid; a cap above membership yields all eligible
   apps. No frontend sorting overrides array order.
5. Pin one immutable presentation revision for a response. Authenticated preview
   can resolve an explicit draft, is visibly labeled, noindex/no-store, and never
   enters public analytics, sitemap or assignment caches.
6. Publish validates the complete document and atomically advances the active
   revision with optimistic concurrency. Rollback selects a retained validated
   revision, without changing commerce or installer state. Mutation conflicts
   produce explicit errors; reload cannot expose partial writes.
7. A last-known-good fallback matches bundle, locale and app scope. If none
   exists, return an explicit unavailable response. Preserve requested/resolved
   variant and reason. A control capture cannot claim a fallback is control.
   Retained publications are rollback candidates, not automatic fallback grants:
   their membership, claims or product scope may have been withdrawn. This
   implementation does not configure an automatic LKG selector. A missing or
   corrupt active publication therefore fails closed without selecting an older
   revision or changing the active pointer. Recovery is an explicit, reverified
   administrator rollback. Default-locale fallback within the same immutable
   document remains supported and is reported as `locale_unavailable`.
8. Legacy section snapshots have an explicit versioned adapter and migration
   receipt. Preserve BAS material before removing generic defaults. Do not make
   legacy first-app inference an alternate permanent resolver.

### Explicit legacy import (adapter version 1)

The authenticated document editor performs this conversion locally, then uses the
existing read-only preview and generation-guarded SaveDraft operations. Import
does not need a new publication endpoint. It requires an explicit existing target
app that is disabled, private and draft, plus an exact configured page locale.
It never selects an app by array position or changes its eligibility.

Retain the complete original UTF-8 snapshot, its SHA-256, adapter version, target
identity and field-level dispositions in the document's administrative string
tables. Use bounded base64 chunks for original bytes and receipt JSON; these
tables are not part of the public projection. Existing profile copy and blocks
remain intact. Append conservatively mapped recovery blocks in numeric section
order with source-array order breaking ties. Reimporting the same source into
the same target is idempotent. Preserve disabled and unknown sections without
rendering them. Refuse malformed envelopes, ambiguous section identities,
unsupported adapter versions and oversized input before changing local state.

Receipt source and target paths use RFC 6901 JSON Pointers. Targets resolve
against the generated document's canonical proto-name JSON, including the typed
content oneof. Only emitted content may receive a mapped disposition. Reimport
verifies original bytes, ordered chunk identities and the deterministic receipt;
damaged recovery storage is an error, not a successful idempotent import.

Map supported narrative text, feature items and FAQs into the finite typed
vocabulary. Preserve original prices, testimonials, links, assets and executable
markup as recovery data rather than activating them. Imported action labels may
appear only as unavailable actions with the target page's configured reason.
Every normalization or omitted rendered field has a disposition; exact source
bytes remain recoverable. An import receipt is not product-claim qualification,
asset release, publication approval or evidence of a live migration.

### Private editing continuity

Preserve OT-P0-012/013 during typed-editor migration: desktop editing and private
preview are side by side, stacking on mobile. A valid local document edit schedules
a read-only preview after a 300ms debounce, without saving, publishing or public
metrics. The server validates and resolves the supplied document through the same
pure resolver; it does not create a persisted revision or change generation.
The authenticated Preview RPC accepts exactly one of a retained revision or an
ephemeral document. Ephemeral diagnostics carry a content-derived preview identity
and explicit preview/noindex/no-store markers, never a public revision claim.
Cancel superseded requests and discard stale responses by document/route/locale
identity. Syntax or validation errors hide the obsolete preview and show errors.
Retained-revision inspection remains explicit; publish and rollback still require
saved revisions, generation checks and user confirmation. Preview failure does not
erase local edits or silently save them. Keep all preview reads behind existing
administrator authorization and request-scoped test storage checks.

## Actions, assets and rendering

Configuration selects typed action references and labels for open, download,
purchase, request-access, unavailable, anchor and app-detail behavior. Pricing,
platform support, entitlements and installer URLs come from current owners; a
marketing edit cannot invent a charge or installer. An unavailable choice has a
visible reason. OS detection suggests, never hides, other supported platforms.
An app's existing delivery-owned web URL can enable an open action even when no
installer exists. `/apps/:slug/download` is a noindex workflow over that app's
detail revision, with explicit platform selection and existing authorization.
It is not another product page or an alias to the detail CTA itself.

The typed public aggregate projects visible commerce facts, not arbitrary owner
metadata, private API keys, local source paths, hidden plans or legacy product
descriptions. Presentation copy supplies the narrative; current commerce and
delivery owners supply prices, currency, billing, release identity and access.

Released assets retain release ID/hash/dimensions/MIME/surface/provenance,
responsive alternatives, crop/focal policy and measured overlay regions. Serve
cached bytes same-origin with integrity checks. Fail or use an explicit configured
fallback for stale/missing/illegible assets. No private Backdrop URL reaches a
visitor. Artwork in bounded panels does not prove copy-over-art legibility;
measure the real surface/crop. Use local fonts, explicit media dimensions, eager
hero loading and lazy below-fold loading. Keep optional art off the critical
content path. Motion is optional and respects reduced motion.

Capture checkpoint labels and JSON ordering are configurable, including custom
sections such as voice and artifacts. Capture both desktop and mobile checkpoints
and record each viewport journey. Correlate desktop and mobile screenshots against
their respective decoded video frames in order; successful DOM assertions alone
do not establish recording integrity. Derive the desktop contact sheet from
distributed recording frames, not from a tile of independent screenshots.
The recorded document's typed bootstrap must match its requested route, variant,
revision and page identity. Its block IDs/kinds must match mounted blocks in exact
order. Integration captures additionally pin the page digest and payload hash to
the real API/SSR preflight. Each checkpoint requires a strictly later decoded
frame, including valid static detail navigation. Navigation stays on the configured
origin; test-mode headers never reach external font or media origins.

Capture has a validated outer deadline (180 seconds by default, at most five
minutes), in addition to bounded navigation and media operations. Expiry closes
the owned browser contexts, aborts media commands, and retains a failure receipt.
Only fonts actually required by the public page determine readiness; an unrelated
admin-font error cannot hide public content. Required font failures still reject
capture. Run the existing UI `test:evidence` script for the 25 deterministic and
real headed/headless capture checks. This uses installed Playwright, Chrome,
FFmpeg/FFprobe and Xvfb; it installs nothing. The opt-in
`TestPresentationBrowserEvidenceIntegration` separately composes the actual
ConfigStore/API/SSR/Vite UI over isolated leased storage. Its receipts disclose
fixture-only qualification and membership, never a live release or cutover.

Both selected designs use one finite React renderer registry, not an iframe of
the mockup. App-scoped content feeds root and detail unchanged. The admin editor
can edit all copy and block props, reorder blocks/apps, validate, preview, publish
and rollback. Structured forms and an inspectable document view share a schema.
Errors identify the field and remediation; invalid documents never become public.
SEO metadata and canonical policy follow resolved route/mode, with usable
non-JavaScript metadata on the deployed head/prerender path.

## Acceptance procedures

All procedures are pending until evidence is linked in the requirement registry.

- Given zero, one and multiple eligible apps, when root resolves, then mode,
  identity, hero, complete block sequence and actions match the rules above.
- Given reordered blocks/apps and caps zero, one and above membership, when the
  response renders, then order and spotlight count match configuration exactly.
- Given the same app revision/locale/experiment, when root and detail resolve,
  then normalized content digests, actions, capabilities and ordering match.
- Given disabled BAS, when public data/routes/hero/catalog are requested, then
  private narrative is absent and the route is 404; authorized preview remains
  complete and the preservation regression verifies every recovered field/asset.
- Given an editor changes copy, an artifact example, block order or hero app
  order, when preview and publish succeed, then the public page reflects them
  without component edits; rollback restores the prior content and conflicts fail.
- Given published assets, when desktop/mobile pages render, then resolved bytes,
  dimensions and release/hash/legibility metadata agree. Missing assets reject
  publication or use the exact configured fallback with explicit diagnostics.
- Given 320/390/768/1440 CSS-pixel viewports, keyboard, zoom and reduced motion,
  when root/detail/bundle pages and dialogs are used, then there is no horizontal
  overflow, clipped action, unreachable detail link or essential moving content;
  focus, modal return, artifact selection and labels work. Check contrast with an
  automated accessibility audit and a screen-reader spot check. Primary mobile
  narrative is at least 16px; mobile action targets are approximately 44px.
- Given live pricing/delivery records or outages, when actions resolve, then
  marketing copy cannot override current prices, entitlement or platform support.
- Given actual desktop/mobile journeys, when evidence is captured, then hero,
  artifact/product region, bundle catalog, closing state and app-detail navigation
  are shown in the recorded page with route/variant/revision/browser/viewport/DPR.
- Given RGB black, encoded limited-range black, white, late black, a frozen
  changing journey, wrong app, fallback mismatch, missing hero media, login
  overlay, decoder failure or viewport mismatch, when evidence is validated,
  then it fails with a precise machine-readable reason and retains failed bytes.
  A legitimate dark page and a deliberately static journey pass. Inspect decoded
  distributed frames plus continuous blank intervals and checkpoints; produce a
  viewable contact sheet and receipt. Reproduce the original capture failure
  separately from merely proving the old YAVG threshold was wrong.

Use focused package regressions while editing and the relevant Test Genie phases
under `docs/TESTING.md`. Lab performance checks are diagnostic; do not claim field
LCP/CLS/INP percentiles from a local screenshot. Final review independently checks
visual fidelity, content provenance, accessibility and evidence integrity.
