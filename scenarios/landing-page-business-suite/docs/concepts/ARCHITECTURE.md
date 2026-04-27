---
title: "Architecture"
description: "System design, components, and deployment topology"
category: "concepts"
order: 4
audience: ["developers"]
---

# Architecture

This document describes the system architecture of landing pages generated from this template.

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
- [CODE: api/remote_profiles_service.go] - Remote profile storage + proxy service
- [CODE: api/plan_store.go] - File-based plan catalog (pricing source of truth)
- [CODE: cli/main.go] - Operator CLI surface
- [CODE: initialization/postgres/schema.sql] - Authoritative database DDL

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
|                          | HTTP/JSON                                              |
|                          v                                                        |
|              +-----------------------+                                            |
|              |      GO API           |                                            |
|              |      (Gin)            |                                            |
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
| **UI ↔ API** | REST/JSON over HTTP. All business logic server-side. |
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

The scenario ships **three runtime surfaces** that share the same database and config:

| Surface | Path | Purpose |
|---------|------|---------|
| **HTTP API** | `api/` | Public landing endpoints, admin portal APIs, billing/Stripe webhooks, AI gateway |
| **Operator CLI** | `cli/` | Out-of-band admin operations (remote-profile management, service auth, scripted setup) |
| **AI Gateway** | `api/ai_gateway_*.go` | First-class subdomain inside the API: routes credit-accounted LLM traffic on behalf of authenticated end users |

```
api/
├── main.go                  # Server struct, service composition, schema bootstrap
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

initialization/
├── postgres/
│   ├── schema.sql           # Database DDL (authoritative location)
│   └── seed.sql             # Initial data
└── configuration/           # Variant + branding seed payloads
```

#### Domain map

The HTTP API exposes ~8 logical domains, registered from `api/routes.go`:

| Domain | Routes | Currently grouped via |
|--------|--------|------------------------|
| `landing` | `/api/v1/landing-config`, `/api/v1/plans`, `/api/v1/variant-space`, `/api/v1/customize` | `registerLandingRoutes` |
| `billing` | `/api/v1/billing/*`, `/api/v1/checkout/*`, `/api/v1/webhooks/stripe`, `/api/v1/subscription/*`, `/api/v1/admin/coupons*`, credits + commerce-admin endpoints | `registerBillingRoutes`, `registerCommerceAdminRoutes`, `registerCreditsRoutes` |
| `downloads` | `/api/v1/downloads`, `/api/v1/admin/download-*`, content + branding + variant + update endpoints | `registerCommerceAdminRoutes`, `registerContentRoutes`, `registerVariantRoutes`, `registerUpdateRoutes` |
| `ai` | `/api/v1/ai/*` | `registerAIRoutes` |
| `metrics` | `/api/v1/metrics/*`, `/api/v1/waitlist`, `/api/feedback`, `/api/v1/admin/feedback*`, `/api/v1/admin/waitlist*` | `registerMetricsRoutes`, `registerFeedbackRoutes`, `registerWaitlistRoutes` |
| `admin` | `/api/v1/admin/login`, `/api/v1/admin/profile`, `/api/v1/admin/users*`, `/api/v1/admin/docs/*`, `/api/v1/admin/settings/stripe*` | `registerAdminCoreRoutes`, `registerAdminUserRoutes`, `registerDocsRoutes` |
| `remote-profile` | `/api/v1/admin/remote-profiles*`, `/api/v1/admin/remote-profile-sessions*` | `registerRemoteProfileRoutes` |
| `user-auth` | `/api/v1/auth/*`, `/api/v1/me/*`, `/api/v1/entitlements` | `registerAuthRoutes`, `registerAccountRoutes` |
| `health` (cross-cutting) | `/health`, `/api/v1/health`, `/api/v1/deploy-readiness` | `registerHealthRoutes`, `registerDeployReadinessRoute` |

> **Transitional layout.** All HTTP-API source today lives in a single `package main` under `api/`. Per-domain `register*Routes` functions provide *file-level* grouping; physical `api/domain/<name>/` subpackages have **not yet** been extracted. The follow-up backlog item `qa-deep-landing-page-business-suite-api-domain-subpackages-20260424` tracks the move into per-domain Go subpackages and the design of any shared package needed to break import cycles. Until then, "domain boundaries" in the API are enforced by file naming + route-registration grouping rather than by package boundaries.

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
     │ GET /landing-config         │              │
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

## See Also

- [Core Concepts](CONCEPTS.md) - A/B testing, data flow
- [Seams & Testability](../internal/SEAMS.md) - Code organization
- [Configuration Guide](../guides/CONFIGURATION_GUIDE.md) - All config files
- [Deployment Guide](../guides/DEPLOYMENT.md) - Production setup
