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

One document contains `schema_version`, `bundle`, `apps`, `pages`, `assets`,
and localized `strings`. Bundle fields include `key`, `name`, `app_order`,
`max_app_slides`, `page_id`, `empty_page_id`, `default_locale` and `locales`.
Apps are keyed by canonical app key, with `slug`, `name`, `enabled`, `visibility`
(`public`/`private`), `publication` (`draft`/`published`), `page_id`, `tagline`,
`description`, `capabilities`, and optional `preservation_ref`.
Page fields include `id`, `locale`, `title`, `description`, `theme`, `navigation`,
`blocks` and `footer`. Each block has `id`, `kind`, `version`, `variant`, and
typed `content`; arrays are authoritative order. Version 1 renderers are finite.

The initial vocabulary covers product-hero, bundle-hero, capability-strip,
product-story, product-demo, app-spotlights, artifact-explorer, voice-story,
device-story, capability-roadmap, pricing, closing-action, faq and footer.
Content contracts include every visible/accessibility label, fixture content,
media reference and action; generic renderers contain no Aquila or BAS copy.
Use validated theme tokens and finite visual variants, not arbitrary CSS/JSX.

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

Diagnostics include requested/resolved route, variant and revision, locale,
bundle/app key, eligible app keys, fallback reason, preview state, block digest,
commerce snapshot and asset release refs. Hash normalized content, not timestamps
or route-only metadata. Root single-app and matching detail content hashes agree.

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
8. Legacy section snapshots have an explicit versioned adapter and migration
   receipt. Preserve BAS material before removing generic defaults. Do not make
   legacy first-app inference an alternate permanent resolver.

## Actions, assets and rendering

Configuration selects typed action references and labels for open, download,
purchase, request-access, unavailable, anchor and app-detail behavior. Pricing,
platform support, entitlements and installer URLs come from current owners; a
marketing edit cannot invent a charge or installer. An unavailable choice has a
visible reason. OS detection suggests, never hides, other supported platforms.

Released assets retain release ID/hash/dimensions/MIME/surface/provenance,
responsive alternatives, crop/focal policy and measured overlay regions. Serve
cached bytes same-origin with integrity checks. Fail or use an explicit configured
fallback for stale/missing/illegible assets. No private Backdrop URL reaches a
visitor. Artwork in bounded panels does not prove copy-over-art legibility;
measure the real surface/crop. Use local fonts, explicit media dimensions, eager
hero loading and lazy below-fold loading. Keep optional art off the critical
content path. Motion is optional and respects reduced motion.

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
