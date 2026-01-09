---
title: "Architecture"
description: "System design, components, and deployment topology"
category: "concepts"
order: 4
audience: ["developers"]
---

# Architecture

This document describes the system architecture of landing pages generated from this template.

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
├── app/
│   ├── App.tsx              # Root component, routing
│   └── providers/           # Context providers
│       ├── VariantProvider  # A/B test variant context
│       └── AuthProvider     # Admin authentication
├── surfaces/
│   ├── public-landing/      # Visitor-facing pages
│   │   ├── routes/          # Page components
│   │   ├── sections/        # Hero, Features, Pricing, etc.
│   │   └── components/      # Shared landing components
│   └── admin-portal/        # Admin interface
│       ├── routes/          # Admin pages
│       ├── components/      # Admin UI components
│       └── controllers/     # Thin orchestration layer
└── shared/
    ├── api/                 # API client functions
    │   ├── landing.ts       # Public endpoints
    │   ├── variants.ts      # Variant management
    │   ├── metrics.ts       # Analytics tracking
    │   └── payments.ts      # Stripe integration
    ├── hooks/               # Custom React hooks
    ├── lib/                 # Utilities
    │   ├── fallbackLandingConfig.ts  # Offline fallback
    │   └── stylingConfig.ts          # Design tokens
    └── components/          # Shared UI components
```

### Backend (Go + Gin)

```
api/
├── main.go                  # Entry point, router setup
├── *_handlers.go            # HTTP handlers (entry layer)
│   ├── landing_handlers     # Public landing config
│   ├── variant_handlers     # A/B testing management
│   ├── metrics_handlers     # Analytics events
│   ├── account_handlers     # User subscriptions
│   └── admin_handlers       # Admin portal APIs
├── *_service.go             # Business logic (domain layer)
│   ├── landing_service      # Config assembly
│   ├── variant_service      # Variant selection
│   ├── content_service      # Section CRUD
│   ├── metrics_service      # Event processing
│   └── download_service     # Entitlement gating
├── auth.go                  # Session middleware
├── logging.go               # Structured logging
└── initialization/
    └── postgres/
        ├── schema.sql       # Database DDL
        └── seed.sql         # Initial data
```

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

### Database Schema

```
+------------------+       +------------------+       +------------------+
|    variants      |       |    sections      |       |     events       |
+------------------+       +------------------+       +------------------+
| id (PK)          |<------| id (PK)          |       | id (PK)          |
| slug (unique)    |       | variant_id (FK)  |       | variant_id (FK)  |
| name             |       | section_type     |       | event_type       |
| status           |       | content (JSONB)  |       | event_data (JSON)|
| weight           |       | order            |       | session_id       |
| axes (JSONB)     |       | enabled          |       | visitor_id       |
| created_at       |       | created_at       |       | created_at       |
| updated_at       |       | updated_at       |       +------------------+
+------------------+       +------------------+

+------------------+       +------------------+       +------------------+
|  subscriptions   |       |     credits      |       |   admin_users    |
+------------------+       +------------------+       +------------------+
| id (PK)          |       | id (PK)          |       | id (PK)          |
| user_email       |       | user_email       |       | email (unique)   |
| stripe_sub_id    |       | balance          |       | password_hash    |
| status           |       | bonus            |       | created_at       |
| plan_tier        |       | updated_at       |       | updated_at       |
| current_period_* |       +------------------+       +------------------+
| created_at       |
+------------------+

+------------------+
|    branding      |
+------------------+
| id (PK)          |
| site_name        |
| tagline          |
| logo_url         |
| favicon_url      |
| primary_color    |
| canonical_url    |
| stripe_keys      |
+------------------+
```

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
API Request → Database → JSON assembly → Cache headers → Response
                |
                v
            Fallback (if API unavailable)
            .vrooli/variants/fallback.json
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
     │              │              │ SELECT       │
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
- [Seams & Testability](SEAMS.md) - Code organization
- [Configuration Guide](CONFIGURATION_GUIDE.md) - All config files
- [Deployment Guide](DEPLOYMENT.md) - Production setup
