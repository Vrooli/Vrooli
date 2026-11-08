# API Overview

A SaaS landing page and operations suite with A/B testing, analytics, Stripe payments, subscriptions, credits, and an admin portal for content management.

## What's Included

- **React + Vite Frontend**: Modern UI with public landing and admin portal
- **Go HTTP API**: Backend service with PostgreSQL
- **A/B Testing**: Whole-page variant testing with weight-based traffic allocation
- **Stripe Integration**: Subscriptions, one-time payments, credits, customer portal
- **Analytics Dashboard**: Page views, clicks, conversions, per-variant metrics
- **Admin Portal**: Content management without code changes

## Quick Start

```bash
# Start the scenario
make start

# View logs
make logs

# Stop
make stop
```

**Access Points:**
| Surface | URL |
|---------|-----|
| Public Landing | `http://localhost:${UI_PORT}/` |
| Admin Portal | `http://localhost:${UI_PORT}/admin` |
| API Health | `http://localhost:${API_PORT}/health` |

Vrooli allocates `API_PORT` and `UI_PORT` during lifecycle startup. Configure
administrator credentials through the scenario's supported secret/configuration
surface; do not rely on development defaults in a deployed environment.

## Documentation

| Getting Started | Reference | Internal |
|-----------------|-----------|----------|
| [Quick Start](../../QUICKSTART.md) | [API Reference](OVERVIEW.md) | [Architecture](../../concepts/ARCHITECTURE.md) |
| [Admin Guide](../../guides/ADMIN_GUIDE.md) | [Configuration](../../guides/CONFIGURATION_GUIDE.md) | [Seams](../../internal/SEAMS.md) |
| [Content Guide](../../guides/CONTENT_GUIDE.md) | [Security](../SECURITY.md) | [Progress](../../internal/PROGRESS.md) |
| [Deployment](../../guides/DEPLOYMENT.md) | [Stripe Webhooks](../STRIPE_WEBHOOKS.md) | [Known Issues](../../internal/PROBLEMS.md) |

**Full documentation:** See [the documentation manifest](../../manifest.json) for the complete navigation structure.

## Hosted AI accounting recovery

Send and retain an `Idempotency-Key` before submitting funded AI work. If the
entire response is lost, call `GET /api/v1/ai/receipt` with the **original issued
AI bearer key** and the **original `Idempotency-Key` header**. LPBS derives the
operation internally and checks its durable key/account binding. A raw Gateway
operation ID, website token, or replacement access key cannot substitute for
this pair. Expired/revoked keys and removed account membership fail authentication.

| HTTP | Body | Meaning |
| --- | --- | --- |
| 200 | `{"status":"settled","charged_credits":200}` | Exact wallet charge is durable; this says nothing about response delivery or content recovery. |
| 200 | `{"status":"released","charged_credits":0}` | Definitive refusal or released hold; this operation is terminal. |
| 202 | `{"status":"pending"}` | Charge/outcome remains unresolved. Credits are deliberately omitted. |
| 404 | `{"status":"unknown"}` | No matching operation is visible to this exact key/account. Wrong-key/account reads have the same response. |
| 400 / 401 / 503 | Generic error | Missing caller idempotency key, failed authentication, or unavailable owner storage. |

Receipt reads are uncached. An owned pending hold may reconcile against the
Gateway's durable private receipt; reading cannot dispatch another provider
request. The public response excludes content, provider/model/routing details,
cost, tokens and private operation IDs. Unknown and pending do not authorize a
new dispatch. Retrying the original funded request with the same caller key
continues to return a conflict without another charge or dispatch; use the
receipt read to recover its accounting result. Calls that omitted a caller key
cannot recover through this contract after total response loss. Key rotation
and response-content replay are outside this contract.

The AI Gateway `HostedDoorClient` currently generates no stable caller key and
does not propagate receipt metadata through its generic inference RPC. This
LPBS route provides recovery to callers that retain their own key; a generic
Gateway client identity contract remains follow-up work.

## Configuration

Key configuration files:

| File | Purpose |
|------|---------|
| `.vrooli/service.json` | Lifecycle, ports, dependencies |
| `config/branding.json` | Site branding (name, logo, colors) |
| `.vrooli/styling.json` | Design tokens and CSS theming |
| `config/variant_space.json` | A/B testing axes (persona, JTBD, style) |
| `config/variants/*.json` | Variant content snapshots |
| `.vrooli/fallback/fallback.json` | Offline-safe landing payload |

## Tech Stack

**Frontend:** React 18, TypeScript, Vite, TailwindCSS, shadcn/ui, Lucide icons

**Backend:** Go, `net/http`, PostgreSQL

## Requirements

See [PRD.md](../../../PRD.md) for the full product requirements document with operational targets.
