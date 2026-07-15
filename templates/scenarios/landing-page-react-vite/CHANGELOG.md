# Changelog

All notable changes to the `landing-page-react-vite` template are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this template aims to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - Unreleased

The Go API rewrite from gorilla/mux to **Proto + Connect-RPC** on the
`api-core` module architecture. This is a greenfield rebuild — no mux
compatibility layer, no parallel old/new handlers.

### Added

- Complete Connect-RPC wire contract under
  `packages/proto/schemas/landing-page-react-vite/v1/`: 13 new proto files
  (`account`, `variant`, `variant_space`, `content`, `metrics`, `branding`,
  `assets`, `docs`, `seo`, `download`, `bundles`, `admin`, `config`) alongside
  the existing `billing`/`pricing`/`settings`, exposing 15 Connect services.
  `billing.proto` gains the `GetBillingPortal` RPC.
- `api/` rebuilt on the api-core module pattern: `main.go`
  (Connect → `EnsureSchemas` → seed → `server.New`), per-domain
  `handlers/<domain>/` (Connect handler + module + endpoints) and
  `internal/<domain>/` (logic + migration-owned `schema.sql`), a modules
  registry, and a shared PostgreSQL testcontainer harness
  (`internal/testutil/pgtest`).
- Branding domain ported end-to-end (Connect handler, storage, schema, and
  real-database handler tests) as the reference vertical.
- Variant, content, metrics, seo, docs, and variant_space domains ported
  end-to-end on the reference pattern, each with real-database (or filesystem)
  handler tests: weighted A/B variant selection with axis validation against
  the file-backed variant space, header-config normalization and
  export/import snapshots; ordered content sections; idempotent analytics
  ingestion with funnel/summary rollups; per-variant SEO resolution over
  branding defaults plus raw `sitemap.xml`/`robots.txt`; the filesystem
  markdown docs browser; and verbatim variant-space serving. A shared
  `internal/variantspace` package loads and validates the axis catalog.
- Financial cluster ported end-to-end: `internal/{plan,paymentsettings,stripe,account}`
  and `handlers/{payments,bundles,account}`. The `LandingPagePaymentsService`
  (checkout, subscription verify/cancel, pricing, Stripe settings, billing
  portal) plus the raw `POST /api/v1/webhooks/stripe` receiver, `BundleAdminService`,
  and `AccountService`. Stripe is fully simulated in-process (hand-rolled
  HMAC-SHA256 webhook verification, subscription lifecycle, intro-pricing
  schedules, credit top-ups) with no `stripe-go` dependency. A shared
  `internal/jsonval` package bridges Postgres JSONB and the `common.v1.JsonValue`
  wire type used by the pricing metadata fields.
- Admin cluster ported end-to-end: `internal/{admin,adminreset,download}` and
  `handlers/{admin,reset,download}`. `AdminAuthService` (bcrypt login) now uses
  **stateless HMAC-signed session cookies** and a Connect interceptor to gate
  admin-only procedures — the `gorilla/sessions` cookie store is gone.
  `AdminResetService` performs env-gated TRUNCATE + reseed; `DownloadService`
  serves the entitlement-gated download catalog.
- Assets cluster ported end-to-end: `internal/assets` and `handlers/assets`.
  `AssetsService` (list/get/delete) plus two REST exceptions — the multipart
  `POST /api/v1/admin/assets/upload` endpoint and the static
  `GET /api/v1/uploads/{path}` file server. Image derivatives (logo/favicon/OG
  sizes) are generated with the standard-library image codecs only.
- Landing-config aggregator ported end-to-end: `internal/landingconfig` and
  `handlers/config`. `LandingConfigService.GetLandingConfig` composes the
  selected variant, its content sections, pricing, downloads, header
  presentation, and public branding into one payload, failing closed to a
  built-in renderable fallback when the database has no usable variant. `seed.go`
  seeds a functional default dataset (admin operator, branding, control/variant-a
  with hero sections and axes, bundle pricing, and the download catalog) reused
  by the admin demo-reset.
- All 15 Connect services are now wired end-to-end with real-database (or
  filesystem/in-memory) tests; the legacy gorilla/mux flat API is fully removed.

### Changed

- Storage schema moves from the monolithic in-Go `ensureSchema` block to
  per-domain migration-owned `schema.sql` applied via
  `database.EnsureSchemas`.
- `api/go.mod` drops `gorilla/sessions` (admin auth moves to a Connect
  interceptor over the session cookie) and adds `connectrpc.com/connect`.
  `gorilla/mux`/`gorilla/handlers` remain only as the api-core server
  scaffold (identical to the `react-vite` template).

### Removed

- The flat-file gorilla/mux API (`*_handlers.go`, `*_service.go`, the
  1,123-line `main.go`, `auth.go`) — replaced by the domain-module layout.

## [0.9.0] - 2026-07-14

Pre-rewrite contract-stabilization release. The template remains quarantined
pending the Go API rewrite (gorilla/mux → Proto + Connect-RPC) scheduled for
`1.0.0`; deep validation is expected to fail until that rewrite lands. This
release makes the template pass **shallow** validation so only the API rewrite
blocks de-quarantine.

### Fixed

- `api/go.mod` now declares the required `replace github.com/vrooli/cli-core =>
  {{PACKAGES_REL_FROM_API}}/cli-core` directive (api-core transitively imports
  `cli-core/cliutil`, so `GOWORK=off go mod tidy` failed without it) and targets
  `go 1.25.0` (was `1.24.0` with a `go1.24.10` toolchain pin).
- `ui/package.json` `file:` dependency paths for `@vrooli/api-base` and
  `@vrooli/iframe-bridge` were five `../` deep and resolved outside the repo;
  corrected to three `../` so they resolve from a generated scenario's `ui/`.
- `template.json` `docs.requirementsGuide` pointed at a nonexistent path; it now
  points at the current guide
  (`scenarios/test-genie/docs/phases/business/requirements-sync.md`).

### Added

- `template.json` `version` (seeded at `0.9.0`, pre-rewrite).
- `template.json` `orientation` block with a landing-specific adoption work
  order (scaffold health, requirements registry, dependency decisions, design
  language, brand/content, payments configuration, demo-content removal).
- `template.json` `exampleDomain` metadata declaring the illustrative SaaS demo
  content payload (`api/templates/saas-landing-page.json`) so `detemplate` and
  lifecycle checks have a marker to reason about.
- `template.json` `copyExcludes` for `CHANGELOG.md` so this template-owned
  changelog is not copied into generated scenarios.
- This `CHANGELOG.md`.

### Removed

- Committed `ui/node_modules` install tree and a stray `api/landing-manager.test`
  build artifact from the template source tree.

### Known limitations

- The Go API still uses `gorilla/mux` rather than Connect-RPC, and the README
  still describes a "Go (Gin) API"; both are addressed by the `1.0.0` rewrite.
  Deep validation will fail until then.

## [1.0.0] - Unreleased

- Planned: rewrite the Go API from `gorilla/mux` to Proto + Connect-RPC, wire the
  UI to the generated Connect clients, reach template test parity, pass deep
  validation, and de-quarantine the template. The README stack description is
  corrected as part of this rewrite.
