---
title: "Architecture"
description: "System design, components, and deployment topology"
category: "concepts"
order: 4
audience: ["developers"]
---

# Architecture

This document describes the architecture of the Landing Page Business Suite: a
subscription, credit, customer, download, and admin-operations service.

**Key Implementation Files:**
- [CODE: api/main.go] - API server entry point and service initialization
- [CODE: api/routes.go] - Per-domain HTTP route registration
- [CODE: ui/src/App.tsx] - React application entry point and routing
- [CODE: ui/src/app/routes/publicRoutes.tsx] - Public-surface route table
- [CODE: ui/src/app/routes/adminRoutes.tsx] - Admin-portal route table
- [CODE: ui/src/app/routes/userAuthRoutes.tsx] - User-auth surface route table
- [CODE: api/landing_config_service.go] - Landing page configuration service
- [CODE: api/ai_gateway_service.go] - AI gateway request routing + credit accounting
- [CODE: api/user_auth_service.go] - End-user magic-link / JWT auth
- [CODE: api/internal/administration/remote_profile_service.go] - Remote profile storage + proxy service
- [CODE: api/plan_store.go] - File-based plan catalog (pricing source of truth)
- [CODE: cli/main.go] - Operator CLI surface
- [CODE: api/internal/schema/schema.go] - Ordered registry for authoritative, domain-owned database DDL

## Table of Contents

1. [System Overview](#system-overview)
2. [Component Architecture](#component-architecture)
3. [Data Architecture](#data-architecture)
4. [Request Flow](#request-flow)
5. [Deployment Topology](#deployment-topology)
6. [Technology Decisions](#technology-decisions)

---

## System Overview

```
+-----------------------------------------------------------------------------------+
|                              LANDING PAGE SYSTEM                                   |
+-----------------------------------------------------------------------------------+
|                                                                                   |
|    VISITORS                           OPERATORS                                   |
|        |                                  |                                       |
|        v                                  v                                       |
|  +-------------+                  +---------------+                               |
|  | Public      |                  | Admin Portal  |                               |
|  | Landing     |                  | /admin/*      |                               |
|  | /           |                  +-------+-------+                               |
|  +------+------+                          |                                       |
|         |                                 |                                       |
|         +----------------+----------------+                                       |
|                          |                                                        |
|                          v                                                        |
|              +-----------------------+                                            |
|              |     REACT + VITE      |                                            |
|              |     (UI Layer)        |                                            |
|              +-----------+-----------+                                            |
|                          |                                                        |
|                          | Generated Connect RPC + documented HTTP exceptions     |
|                          v                                                        |
|              +-----------------------+                                            |
|              |      GO API           |                                            |
|              |   (net/http + mux)    |                                            |
|              +-----------+-----------+                                            |
|                          |                                                        |
|            +-------------+-------------+                                          |
|            |             |             |                                          |
|            v             v             v                                          |
|     +----------+   +-----------+   +--------+                                     |
|     | Postgres |   |  Stripe   |   | File   |                                     |
|     | (Data)   |   |  (Billing)|   | System |                                     |
|     +----------+   +-----------+   +--------+                                     |
|                                                                                   |
+-----------------------------------------------------------------------------------+
```

### Key Boundaries

| Boundary | Purpose |
|----------|---------|
| **UI ↔ API** | Generated Connect RPC for proto-owned operations; REST only for external-shape, upload, webhook, and probe edges. All business logic remains server-side. |
| **API ↔ Database** | Direct PostgreSQL via `database/sql`. No ORM. |
| **API ↔ Stripe** | HTTPS to Stripe APIs. Webhook verification. |
| **Public ↔ Admin** | Route-based separation. Session auth for admin. |

---

## Component Architecture

### Frontend (React + Vite)

```
ui/src/
├── App.tsx                  # Thin composer: providers + per-surface route tables
├── main.tsx                 # Vite/React bootstrap
├── app/
│   ├── providers/           # Context providers
│   │   ├── LandingVariantProvider  # A/B test variant context
│   │   ├── AdminAuthProvider       # Admin session/auth
│   │   └── UserAuthProvider        # End-user (magic-link) auth
│   └── routes/              # Surface-aligned route tables consumed by App.tsx
│       ├── publicRoutes.tsx        # /, /checkout, /feedback (+ coming-soon guard)
│       ├── adminRoutes.tsx         # /admin/* (ProtectedRoute)
│       └── userAuthRoutes.tsx      # /admin/login, /auth/login, /auth/verify
├── surfaces/
│   ├── public-landing/      # Visitor-facing pages
│   │   ├── routes/          # Page components
│   │   ├── sections/        # Hero, Features, Pricing, etc.
│   │   └── components/      # Shared landing components
│   ├── admin-portal/        # Admin interface
│   │   ├── routes/          # Admin pages
│   │   ├── components/      # Admin UI components (incl. ProtectedRoute)
│   │   └── controllers/     # Thin orchestration layer
│   └── user-auth/           # End-user magic-link login + verify surface
│       ├── routes/          # UserLogin, VerifyMagicLink
│       └── index.ts         # Surface barrel
└── shared/
    ├── api/                 # API client functions
    │   ├── landing.ts       # Public endpoints
    │   ├── variants.ts      # Variant management
    │   ├── metrics.ts       # Analytics tracking
    │   └── payments.ts      # Stripe integration
    ├── hooks/               # Custom React hooks
    ├── lib/                 # Utilities
    ├── consts/              # Shared constants
    ├── test-utils/          # Vitest helpers
    └── ui/                  # Shared UI primitives (ErrorBoundary, Toast, …)
```

### Backend (Go API + CLI)

The scenario ships **num[sot]:three runtime surfaces** that share the same database and config:

| Surface | Path | Purpose |
|---------|------|---------|
| **HTTP API** | `api/` | Public landing endpoints, admin portal APIs, billing/Stripe webhooks, AI gateway |
| **Operator CLI** | `cli/` | Out-of-band admin operations (remote-profile management, service auth, scripted setup) |
| **AI Gateway** | `api/ai_gateway_*.go` | First-class subdomain inside the API: routes credit-accounted LLM traffic on behalf of authenticated end users |

```
api/
├── main.go                  # Server composition and lifecycle wiring only
├── routes.go                # Per-domain register*Routes() calls — single composer
├── auth.go                  # Admin session middleware (requireAdmin, requireAdminOrService)
├── user_auth_*.go           # End-user magic-link + JWT auth (handlers, middleware, service)
├── account_*.go             # Subscription / credits / entitlements for end users
├── ai_gateway_*.go          # AI Gateway: model listing, chat, stream, usage
├── billing_*.go             # Stripe checkout + portal + webhook
├── stripe_service.go        # Stripe client + plan/coupon import
├── plan_*.go                # File-backed plan/pricing catalog (PlanStore)
├── download_*.go            # Download hosting, entitlement gating, S3 storage
├── content_handlers.go      # Branding, assets, SEO
├── variant_*.go             # A/B testing variant config
├── metrics_*.go             # Analytics event ingestion + summary
├── feedback_*.go            # In-product feedback intake
├── waitlist_*.go            # Coming-soon waitlist capture
├── remote_profile*.go       # Remote-profile storage, proxy, session linking, revoke
├── apikeys_service.go       # API-key vault for upstream LLM providers
├── limits_service.go        # Cost-based credit limits per tier / app
├── usage_service.go         # Credit reservations, usage reporting
└── *_test.go                # Per-domain tests (currently colocated in `package main`)

cli/
├── main.go                  # Operator CLI entrypoint
├── app.go                   # Command wiring
├── domains/                 # Per-domain CLI commands (mirrors API domains)
└── internal/                # Shared CLI helpers (auth, config)

api/internal/
├── administration/          # Admin accounts, sessions, remote profiles + schema
├── commerce/                # Plans, Stripe, subscriptions, credits, usage + schema
├── delivery/                # Entitled download catalog/storage + schema
├── experimentation/         # Variant-selection policy, independent of config decoding
├── intelligence/            # Upstream AI-provider error classification
├── metrics/                 # Analytics event persistence + schema
├── content/                 # Assets, feedback, waitlist + schema
├── schema/                  # Thin ordered schema registry (no business rules)
└── testutil/                # Test-only shared helpers; forbidden in production imports
```

#### Domain map

The HTTP API exposes ~8 logical domains, registered from `api/routes.go`:

| Domain | Routes | Currently grouped via |
|--------|--------|------------------------|
| `landing` | `LandingConfigService.GetLandingConfig`, `/api/v1/plans`, `VariantSpaceService`, `/api/v1/customize` | `registerLandingRoutes` |
| `billing` | `LandingPagePaymentsService`, `BundleAdminService`, `CouponAdminService`, Stripe webhook, Stripe import, credits + remaining commerce-admin endpoints | `registerBillingRoutes`, `registerCommerceAdminRoutes`, `registerCreditsRoutes` |
| `downloads` | `/api/v1/downloads`, `/api/v1/admin/download-*`, content + branding + variant + update endpoints | `registerCommerceAdminRoutes`, `registerContentRoutes`, `registerVariantRoutes`, `registerUpdateRoutes` |
| `ai` | `/api/v1/ai/*` | `registerAIRoutes` |
| `metrics` | `/api/v1/metrics/*`, `/api/v1/waitlist`, `/api/v1/feedback`, `/api/v1/admin/feedback*`, `/api/v1/admin/waitlist*` | `registerMetricsRoutes`, `registerFeedbackRoutes`, `registerWaitlistRoutes` |
| `admin` | `/api/v1/admin/login`, `/api/v1/admin/profile`, `/api/v1/admin/users*`, `/api/v1/admin/docs/*`, `StripeSettingsService` Connect procedures | `registerAdminCoreRoutes`, `registerAdminUserRoutes`, `registerDocsRoutes` |
| `remote-profile` | `/api/v1/admin/remote-profiles*`, `/api/v1/admin/remote-profile-sessions*` | `registerRemoteProfileRoutes` |
| `user-auth` | `/api/v1/auth/*`, `/api/v1/me/*`, `/api/v1/entitlements` | `registerAuthRoutes`, `registerAccountRoutes` |
| `health` (cross-cutting) | `/health`, `/api/v1/health`, `/api/v1/deploy-readiness` | `registerHealthRoutes`, `registerDeployReadinessRoute` |

> **Migration state.** Runtime schema ownership has moved into domain packages.
> Proto-owned public pricing, payments, Stripe settings, landing configuration,
> branding, SEO, Variant Space, measures, and the admin bundle and coupon
> paths are now generated Connect procedures consumed by typed UI or CLI
> clients. Remaining REST paths are migration work unless they are a webhook,
> upload, third-party-shaped callback, or operational probe. Services and
> handlers remain partially colocated in `package main`; each move must preserve
> behavior and land with its tests. `main.go` is restricted to configuration,
> database wiring, schema registration, service composition, route registration,
> and process start.

### Target capability ownership

| Capability | Primary archetype | API owner | UI/CLI owner |
|---|---|---|---|
| Admin and remote profiles | CRUD + integration | `administration` | admin portal / `cli/domains/remoteprofiles` |
| Plans, subscriptions, Stripe, coupons | Temporal workflow + integration | `commerce` | billing portal / `cli/domains/billing` |
| Credits, usage, reservations, provider keys | Policy + temporal workflow | `commerce` | admin usage views / `cli/domains/credits` |
| Downloads and release hosting | Blob + entitlement policy | `delivery` | downloads views / `cli/domains/downloads` |
| Landing configuration and variants | Configuration | `experimentation` policy plus file-backed config substrate | public landing / `cli/domains/variants` |
| Metrics, feedback, waitlist | Reporting + CRUD | `metrics` and `content` | analytics/admin portal |
| End-user authentication | Temporal security workflow | `administration` and `commerce` | user-auth surface / `cli/domains/auth` |
| AI gateway | Integration + credit orchestration | `intelligence` with commerce credit policy | API surface / `cli/domains/ai` |

### Responsibility Layers

```
+------------------------------------------------------------------+
|                        PRESENTATION LAYER                         |
|  Handlers: Parse requests, validate input, serialize responses    |
|  Location: api/*_handlers.go                                      |
+------------------------------------------------------------------+
                               |
                               v
+------------------------------------------------------------------+
|                          DOMAIN LAYER                             |
|  Services: Business rules, validation, orchestration              |
|  Location: api/*_service.go                                       |
+------------------------------------------------------------------+
                               |
                               v
+------------------------------------------------------------------+
|                      INFRASTRUCTURE LAYER                         |
|  Database: SQL queries, Stripe API calls                          |
|  Location: Within services (no separate repository layer)         |
+------------------------------------------------------------------+
```

---

## Data Architecture

### File-based Configuration (Source of Truth)

- `.vrooli/plans.json` — bundle + pricing catalog (PlanStore). Writes are atomic; Stripe imports batch updates via `PlanService`.
- Plan updates and Stripe imports enforce bundle↔Stripe product alignment to prevent cross-product contamination, plus tier invariants (free plans are $0; credits/donations are one-time).
- `config/variants/*.json` — landing variants + sections (ConfigStore)
- `config/branding.json` — public branding for the landing UI
- `.vrooli/fallback/fallback.json` — baked fallback variant payload

### Database Schema (Runtime State)

The database stores runtime state (sessions, subscriptions, metrics) rather than configuration:

- `admin_users`, `admin_sessions` — admin auth/session state (includes connector-attributed remote-profile sessions via user-agent metadata)
- `metrics_events` — analytics events
- `checkout_sessions` — Stripe checkout tracking
- `subscriptions`, `subscription_schedules` — subscription lifecycle cache
- `subscription_tier_limits` — per-tier usage limits
- Download tables (apps/artifacts/storage) for gated assets

> **Note:** Legacy `bundle_products`/`bundle_prices` tables may still exist for migrations, but pricing is now sourced from `.vrooli/plans.json`.

### Data Flow Patterns

**Variant Selection:**
```
URL param → localStorage → API weight-based → Store in localStorage
     |            |              |
     v            v              v
   Force      Sticky         Random
  variant    returning      selection
```

**Content Loading:**
```
API Request → ConfigStore (JSON) → JSON assembly → Cache headers → Response
                  |
                  v
              Fallback (if config unavailable)
              .vrooli/fallback/fallback.json
```

**Event Tracking:**
```
User action → Frontend SDK → POST /metrics/track → Database INSERT
                   |
                   +→ Include variant_id for A/B attribution
```

---

## Request Flow

### Public Landing Page Load

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌──────────┐
│ Browser │    │  Vite   │    │ Go API  │    │ Postgres │
└────┬────┘    └────┬────┘    └────┬────┘    └────┬─────┘
     │              │              │              │
     │ GET /        │              │              │
     │─────────────>│              │              │
     │              │              │              │
     │  index.html  │              │              │
     │<─────────────│              │              │
     │              │              │              │
     │ POST LandingConfigService.GetLandingConfig │
     │────────────────────────────>│              │
     │              │              │              │
     │              │              │ Read config  │
     │              │              │─────────────>│
     │              │              │              │
     │              │              │ variant,     │
     │              │              │ sections     │
     │              │              │<─────────────│
     │              │              │              │
     │  { variant, sections, ... } │              │
     │<────────────────────────────│              │
     │              │              │              │
     │ Render sections             │              │
     │              │              │              │
```

### Admin Section Update

```
┌─────────┐    ┌─────────┐    ┌──────────┐
│ Admin   │    │ Go API  │    │ Postgres │
└────┬────┘    └────┬────┘    └────┬─────┘
     │              │              │
     │ PATCH /sections/42          │
     │ + session cookie            │
     │─────────────>│              │
     │              │              │
     │              │ Verify       │
     │              │ session      │
     │              │              │
     │              │ UPDATE       │
     │              │ sections     │
     │              │─────────────>│
     │              │              │
     │              │     OK       │
     │              │<─────────────│
     │              │              │
     │ { updated section }         │
     │<─────────────│              │
     │              │              │
```

### Stripe Checkout Flow

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ Browser │    │ Go API  │    │ Stripe  │    │Postgres │
└────┬────┘    └────┬────┘    └────┬────┘    └────┬────┘
     │              │              │              │
     │ POST /checkout/create       │              │
     │─────────────>│              │              │
     │              │              │              │
     │              │ Create       │              │
     │              │ session      │              │
     │              │─────────────>│              │
     │              │              │              │
     │              │ session_url  │              │
     │              │<─────────────│              │
     │              │              │              │
     │ { url }      │              │              │
     │<─────────────│              │              │
     │              │              │              │
     │ Redirect to Stripe          │              │
     │────────────────────────────>│              │
     │              │              │              │
     │   ... payment ...           │              │
     │              │              │              │
     │              │ Webhook      │              │
     │              │<─────────────│              │
     │              │              │              │
     │              │ INSERT subscription         │
     │              │────────────────────────────>│
     │              │              │              │
```

---

## Deployment Topology

### Local Development

```
┌─────────────────────────────────────────────────────┐
│                    DEVELOPER MACHINE                 │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌─────────────┐    ┌─────────────┐                │
│  │ Vite Dev    │    │ Go API      │                │
│  │ :3000       │───>│ :8080       │                │
│  │ (HMR)       │    │             │                │
│  └─────────────┘    └──────┬──────┘                │
│                            │                        │
│                            v                        │
│                    ┌─────────────┐                 │
│                    │ PostgreSQL  │                 │
│                    │ :5432       │                 │
│                    └─────────────┘                 │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### Production (Single Server)

```
┌─────────────────────────────────────────────────────┐
│                    PRODUCTION SERVER                 │
├─────────────────────────────────────────────────────┤
│                                                     │
│         ┌─────────────────────────┐                │
│         │  nginx / Cloudflare     │                │
│         │  (SSL termination)      │                │
│         └───────────┬─────────────┘                │
│                     │                               │
│     ┌───────────────┴───────────────┐              │
│     │                               │              │
│     v                               v              │
│  ┌─────────────┐           ┌─────────────┐        │
│  │ Static      │           │ Go API      │        │
│  │ Assets      │           │ :3000       │        │
│  │ (built UI)  │           │             │        │
│  └─────────────┘           └──────┬──────┘        │
│                                   │                │
│                                   v                │
│                           ┌─────────────┐         │
│                           │ PostgreSQL  │         │
│                           │ :5432       │         │
│                           └─────────────┘         │
│                                                    │
└─────────────────────────────────────────────────────┘
```

### Production (Vrooli Managed)

```
┌────────────────────────────────────────────────────────────────┐
│                         VROOLI SERVER                           │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  ┌──────────────────┐                                          │
│  │ Cloudflare       │                                          │
│  │ Tunnel           │<─── Internet traffic                     │
│  └────────┬─────────┘                                          │
│           │                                                    │
│           v                                                    │
│  ┌──────────────────┐    ┌──────────────────┐                 │
│  │ app-monitor      │───>│ Your Landing     │                 │
│  │ (routing)        │    │ Page Scenario    │                 │
│  └──────────────────┘    └────────┬─────────┘                 │
│                                   │                            │
│                                   v                            │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │                    SHARED RESOURCES                       │ │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐         │ │
│  │  │ PostgreSQL │  │ Redis      │  │ Ollama     │         │ │
│  │  │            │  │ (optional) │  │ (optional) │         │ │
│  │  └────────────┘  └────────────┘  └────────────┘         │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

---

## Technology Decisions

### Why React + Vite?

| Consideration | Decision |
|---------------|----------|
| **Build speed** | Vite's esbuild is 10-100x faster than webpack |
| **HMR** | Sub-second updates during development |
| **Ecosystem** | Largest component ecosystem |
| **Type safety** | TypeScript-first with excellent tooling |

### Why Go + Gin?

| Consideration | Decision |
|---------------|----------|
| **Performance** | High throughput for metrics ingestion |
| **Deployment** | Single binary, easy containerization |
| **Type safety** | Compile-time checks reduce payment bugs |
| **Concurrency** | Goroutines for parallel Stripe/DB calls |

### Why PostgreSQL?

| Consideration | Decision |
|---------------|----------|
| **JSONB** | Flexible section content without migrations |
| **Reliability** | Battle-tested for financial data |
| **Shared resource** | Vrooli's default database |
| **Features** | CTEs, window functions for analytics |

### Why No ORM?

| Consideration | Decision |
|---------------|----------|
| **Performance** | Direct SQL avoids N+1 queries |
| **Transparency** | Queries are explicit and auditable |
| **Flexibility** | PostgreSQL-specific features easily used |
| **Trade-off** | More boilerplate, but fewer surprises |

### Why localStorage for Variants?

| Consideration | Decision |
|---------------|----------|
| **Simplicity** | No server-side session management |
| **Privacy** | Works without authentication |
| **Persistence** | Survives page refreshes |
| **Trade-off** | Doesn't sync across devices |

---

## Security Architecture

```
                              ┌─────────────────┐
                              │   Public Web    │
                              └────────┬────────┘
                                       │
                    ┌──────────────────┴──────────────────┐
                    │                                     │
                    v                                     v
           ┌───────────────┐                    ┌───────────────┐
           │ Public Routes │                    │ Admin Routes  │
           │ No auth       │                    │ Session auth  │
           └───────────────┘                    └───────┬───────┘
                    │                                   │
                    │                           ┌───────┴───────┐
                    │                           │ Auth          │
                    │                           │ Middleware    │
                    │                           │ - Cookie      │
                    │                           │ - bcrypt      │
                    │                           └───────┬───────┘
                    │                                   │
                    └──────────────────┬────────────────┘
                                       │
                                       v
                              ┌─────────────────┐
                              │ Stripe Webhook  │
                              │ Signature       │
                              │ Verification    │
                              └─────────────────┘
```

### Security Measures

| Layer | Measure |
|-------|---------|
| **Authentication** | bcrypt password hashing, session cookies |
| **Authorization** | Admin-only routes behind middleware |
| **Webhooks** | Stripe signature verification |
| **Inputs** | JSON schema validation, SQL parameterization |
| **Secrets** | Environment variables, never in code |
| **CORS** | Configured for production domains |

---

## Scaling Considerations

### Bottleneck Analysis

| Component | Bottleneck | Mitigation |
|-----------|------------|------------|
| **API** | CPU (Go handles well) | Horizontal scaling |
| **Database** | Connection limits | Connection pooling |
| **Metrics** | Write volume | Batch inserts, Redis buffer |
| **Static assets** | Bandwidth | CDN (Cloudflare) |

### Future Scaling Path

1. **Redis caching** - Session and subscription verification
2. **Read replicas** - Analytics queries off primary
3. **Message queue** - Async metric processing
4. **CDN** - Static asset distribution

---

## Zone Map

The API is in an incremental migration: legacy root files remain the composition
edge, while new work follows this zone map. Domain packages must remain
transport-free; `handlers/` translates HTTP and receives generic response
behavior from the root.

| Directory | Zone | May Import | Enforcement |
|---|---|---|---|
| `api/handlers/metrics/` | transport edge | `internal/metrics`, standard library | `internal/testutil/no_prod_import_test.go` and handler tests |
| `api/internal/administration/` | domain and persistence | standard library, database driver as needed | production-import test |
| `api/internal/clock/` | cross-cutting substrate | standard library | seam-registry test |
| `api/internal/content/` | domain and persistence | standard library, database driver as needed | production-import test |
| `api/internal/commerce/` | domain and persistence | standard library, database driver as needed | production-import test |
| `api/internal/delivery/` | domain and persistence | standard library, database driver as needed | production-import test |
| `api/internal/experimentation/` | variant-selection policy | standard library | focused decision tests |
| `api/internal/intelligence/` | provider error semantics | standard library | focused error tests |
| `api/internal/metrics/` | domain and persistence | `internal/clock`, standard library, database driver as needed | production-import and seam-registry tests |
| `api/internal/schema/` | schema registry substrate | domain schema packages and standard library | schema registry tests |
| `api/internal/testutil/` | test-only substrate | standard library | production imports are forbidden |
| `api/templates/` | embedded API templates | standard library | compile/test coverage |

### Boundary Maturity

| Zone | Level | Evidence | Remaining Drift |
|---|---|---|---|
| API transport/domain | 2 | Metrics handlers live under `handlers/metrics`; its domain logic is transport-free. | Most legacy handlers and services remain in the root `main` package. |
| API substrate | 2 | Schema registry, clock, and test utility packages have named responsibilities. | Generic HTTP response and server composition seams are still root-owned. |
| UI | 3 | Surface-aligned routes, providers, and typed clients are documented and tested. | Keep consolidating legacy shared utility imports as files move. |
| CLI | 2 | Per-domain CLI packages and a manifest exist. | Continue migrating legacy command wiring to typed command contracts. |

## See Also

- [Core Concepts](CONCEPTS.md) - A/B testing, data flow
- [Seams & Testability](../internal/SEAMS.md) - Code organization
- [Configuration Guide](../guides/CONFIGURATION_GUIDE.md) - All config files
- [Deployment Guide](../guides/DEPLOYMENT.md) - Production setup
