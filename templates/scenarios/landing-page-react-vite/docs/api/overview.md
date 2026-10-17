---
title: "API Reference"
description: "Complete REST API documentation for landing pages"
category: "reference"
order: 1
audience: ["developers"]
---

# API Reference

This document provides an overview of the REST API exposed by this landing page. All endpoints use JSON for request/response bodies unless otherwise noted.

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
| 409 | Conflict - Duplicate resource |
| 422 | Unprocessable Entity - Validation failed |
| 429 | Too Many Requests - Rate limited |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

---

## Error Catalog

All errors follow the standard format:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": { }  // Optional additional context
}
```

### Authentication Errors

| Code | HTTP | Message | Resolution |
|------|------|---------|------------|
| `AUTH_REQUIRED` | 401 | Authentication required | Include session cookie or login |
| `SESSION_EXPIRED` | 401 | Session has expired | Re-authenticate via `/admin/login` |
| `INVALID_CREDENTIALS` | 401 | Invalid email or password | Check credentials |
| `ACCESS_DENIED` | 403 | Access denied | Admin privileges required |

**Example:**
```json
{
  "error": "Session has expired",
  "code": "SESSION_EXPIRED"
}
```

### Validation Errors

| Code | HTTP | Message | Resolution |
|------|------|---------|------------|
| `VALIDATION_FAILED` | 400 | Validation failed | Check `details` for field errors |
| `MISSING_FIELD` | 400 | Required field missing: {field} | Include required field |
| `INVALID_FORMAT` | 400 | Invalid format for {field} | Check field format |
| `INVALID_JSON` | 400 | Invalid JSON body | Fix JSON syntax |

**Example:**
```json
{
  "error": "Validation failed",
  "code": "VALIDATION_FAILED",
  "details": {
    "fields": {
      "email": "Invalid email format",
      "weight": "Must be between 0 and 100"
    }
  }
}
```

### Resource Errors

| Code | HTTP | Message | Resolution |
|------|------|---------|------------|
| `NOT_FOUND` | 404 | {resource} not found | Check ID exists |
| `VARIANT_NOT_FOUND` | 404 | Variant not found | Check variant slug/ID |
| `SECTION_NOT_FOUND` | 404 | Section not found | Check section ID |
| `DUPLICATE_SLUG` | 409 | Slug already exists | Choose unique slug |
| `CANNOT_DELETE` | 409 | Cannot delete {resource} | Check for dependencies |

**Example:**
```json
{
  "error": "Variant not found",
  "code": "VARIANT_NOT_FOUND",
  "details": {
    "slug": "holiday-special"
  }
}
```

### Payment Errors

| Code | HTTP | Message | Resolution |
|------|------|---------|------------|
| `STRIPE_NOT_CONFIGURED` | 503 | Stripe not configured | Admin must configure Stripe keys |
| `CHECKOUT_FAILED` | 400 | Failed to create checkout session | Check Stripe configuration |
| `WEBHOOK_INVALID` | 400 | Invalid webhook signature | Check webhook secret |
| `SUBSCRIPTION_REQUIRED` | 403 | Active subscription required | User must subscribe |
| `PLAN_NOT_FOUND` | 404 | Pricing plan not found | Check price ID |

**Example:**
```json
{
  "error": "Active subscription required",
  "code": "SUBSCRIPTION_REQUIRED",
  "details": {
    "required_tier": "pro",
    "current_status": "none"
  }
}
```

### Metrics Errors

| Code | HTTP | Message | Resolution |
|------|------|---------|------------|
| `INVALID_EVENT_TYPE` | 400 | Invalid event type | Use allowed event types |
| `MISSING_VARIANT_ID` | 400 | variant_id required for event | Include variant_id |
| `EVENT_DATA_REQUIRED` | 400 | Event data required | Include event_data object |

**Example:**
```json
{
  "error": "Invalid event type",
  "code": "INVALID_EVENT_TYPE",
  "details": {
    "provided": "invalid_type",
    "allowed": ["page_view", "click", "scroll_depth", "form_submit", "conversion"]
  }
}
```

### Server Errors

| Code | HTTP | Message | Resolution |
|------|------|---------|------------|
| `INTERNAL_ERROR` | 500 | Internal server error | Check server logs |
| `DATABASE_ERROR` | 500 | Database error | Check database connection |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable | Retry with backoff |

**Example:**
```json
{
  "error": "Database error",
  "code": "DATABASE_ERROR"
}
```

### Error Handling Best Practices

**Client-side handling:**
```javascript
try {
  const response = await fetch('/api/v1/sections/42', {
    method: 'PATCH',
    body: JSON.stringify(data)
  });

  if (!response.ok) {
    const error = await response.json();

    switch (error.code) {
      case 'SESSION_EXPIRED':
        // Redirect to login
        window.location.href = '/admin/login';
        break;
      case 'VALIDATION_FAILED':
        // Show field errors
        displayFieldErrors(error.details.fields);
        break;
      case 'NOT_FOUND':
        // Resource deleted, refresh list
        refreshSections();
        break;
      default:
        // Generic error handling
        showToast(error.error);
    }
  }
} catch (e) {
  showToast('Network error. Please retry.');
}
```

**Retry strategy for 503:**
```javascript
async function fetchWithRetry(url, options, retries = 3) {
  for (let i = 0; i < retries; i++) {
    const response = await fetch(url, options);
    if (response.status !== 503) return response;
    await sleep(1000 * Math.pow(2, i)); // Exponential backoff
  }
  throw new Error('Service unavailable after retries');
}
```

---

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
