---
title: "API Reference"
description: "Complete REST API documentation for landing pages"
category: "reference"
order: 1
audience: ["developers"]
---

# API Reference

This document provides an overview of the REST API exposed by generated landing pages. All endpoints use JSON for request/response bodies unless otherwise noted.

## Base URL

```
http://localhost:${API_PORT}/api/v1
```

The actual port is allocated by Vrooli's lifecycle system.

## API Sections

The API is organized into the following domains:

| Section | Description | Authentication |
|---------|-------------|----------------|
| [Landing](landing.md) | Public landing page config, plans, branding | None |
| [Variants](variants.md) | A/B testing variant management | Admin |
| [Sections](sections.md) | Content section CRUD | Admin |
| [Metrics](metrics.md) | Analytics and event tracking | Mixed |
| [Payments](payments.md) | Stripe integration and billing | Mixed |
| [Admin](admin.md) | Admin portal management | Admin |

## Authentication

### Public Endpoints

No authentication required. Examples:
- `GET /landing-config`
- `GET /plans`
- `POST /metrics/track`

### User-Specific Endpoints

Require user identity via header:
```
X-User-Email: user@example.com
```

Examples:
- `GET /me/subscription`
- `GET /me/credits`
- `GET /entitlements`

### Admin Endpoints

Require session authentication via cookie (set by `/admin/login`).

Examples:
- `GET /variants` (all variants)
- `PATCH /sections/{id}`
- `PUT /admin/branding`

## Common Patterns

### Response Format

Success responses return the requested data directly:

```json
{
  "id": 1,
  "name": "Example",
  ...
}
```

### Error Format

All errors return this structure:

```json
{
  "error": "Description of the error"
}
```

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request - Invalid input |
| 401 | Unauthorized - Not authenticated |
| 403 | Forbidden - Not authorized |
| 404 | Not Found |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

## Health Check

### GET /health

Returns API health status and dependency checks.

**Authentication:** None

**Response:**
```json
{
  "status": "healthy",
  "service": "My Landing API",
  "version": "1.0.0",
  "readiness": true,
  "timestamp": "2024-01-15T10:30:00Z",
  "dependencies": {
    "database": "connected"
  }
}
```

## Quick Reference

### Most Common Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/health` | Health check |
| GET | `/landing-config` | Get page configuration |
| GET | `/plans` | Get pricing plans |
| POST | `/metrics/track` | Track analytics event |
| POST | `/checkout/create` | Create Stripe checkout |
| GET | `/subscription/verify` | Verify subscription |

### Admin Endpoints Summary

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/admin/login` | Authenticate |
| GET | `/variants` | List all variants |
| POST | `/variants` | Create variant |
| PATCH | `/sections/{id}` | Update section |
| GET | `/metrics/summary` | Analytics overview |
| PUT | `/admin/branding` | Update branding |

---

## Next Steps

- [Landing Endpoints](landing.md) - Public page configuration
- [Variants Endpoints](variants.md) - A/B testing management
- [Payments Endpoints](payments.md) - Stripe integration
