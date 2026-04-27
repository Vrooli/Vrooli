---
title: "Assumptions"
description: "Operating assumptions the implementation relies on"
category: "internal"
order: 100
audience: ["developers"]
internal: true
---

# Assumptions

This document records the **operating assumptions** the landing-page-business-suite scenario relies on. If any of these change, expect cascading work across the API, UI, and operator CLI.

## Runtime environment

- **Postgres is always reachable.** The API connects to a Postgres database at startup using `POSTGRES_*` env vars (or `DATABASE_URL`); see `api/main.go:resolveDatabaseURL`. The scenario does not run in a degraded "no-database" mode.
- **Schema is owned by this scenario.** `ensureSchema` in `api/main.go` runs idempotent `CREATE TABLE IF NOT EXISTS` / `ALTER TABLE … ADD COLUMN IF NOT EXISTS` statements. The authoritative DDL is mirrored at `initialization/postgres/schema.sql`. We assume nothing else writes to the same database.
- **The lifecycle system seeds `POSTGRES_*` env vars and a writable filesystem.** Variant + branding JSON config under `config/` is treated as **tracked, source-of-truth** state, not runtime state.
- **Stripe keys, webhook secrets, and AI provider keys are optional at boot.** The API starts without them and reports them as missing through `/api/v1/health` and `/api/v1/deploy-readiness`. Features that require them surface clear error semantics rather than crashing.

## Identity model

- **Two distinct identity surfaces, deliberately separate**:
  - **Admin** — single seeded admin row (`admin_users.id = 1`) with bcrypted password + cookie session (`admin_sessions`). Admin auth is the *operator* identity; it does not unlock end-user features.
  - **End user** — magic-link → JWT model backed by `users` + `auth_tokens` + `user_sessions`. End-user identity is the *customer* identity; it gates `/api/v1/me/*`, `/api/v1/auth/*`, `/api/v1/ai/*`, billing portal, and downloads.
- These two surfaces never share a session or a cookie. A request authenticated as admin is **not** also authenticated as the user with the same email.

## Pricing & credits

- **`.vrooli/plans.json` is the source of truth for pricing.** Database tables `bundle_products` / `bundle_prices` are legacy and may still exist for in-flight data, but `PlanService` reads from the file-backed catalog.
- **Stripe is the source of truth for actual money movement.** Local subscription/credit state is a cache; webhooks are the only path that creates entitlements.
- **Credits are cost-based.** `subscription_tier_limits.cost_multiplier = 1_000_000` translates dollars to internal units. AI usage is reserved (`credit_reservations`) before a stream starts and finalized on completion to prevent TOCTOU.

## Data assumptions

- All tables use Postgres-native `gen_random_uuid()` where UUIDs are used. The `pgcrypto` extension is assumed available on the target Postgres.
- `metrics_events` is append-only at the API layer; admin endpoints aggregate but never mutate individual rows.
- Remote-profile sessions are encrypted-at-rest via `remote_profiles.encrypted_session`. The encryption key is not stored in the database.

## Out-of-band tooling

- The **operator CLI** (`cli/`) shares the same Postgres + config but talks to the API over HTTP using a service-bearer token, not by importing API code. There is no shared Go package between `api/` and `cli/`.
