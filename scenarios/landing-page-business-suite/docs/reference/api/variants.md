---
title: "Variant Endpoints"
description: "A/B testing variant management APIs"
category: "reference"
order: 3
audience: ["developers"]
---

# Variant Endpoints

Endpoints for managing A/B testing variants.

## Public experiment identity

Generated `VariantService.SelectVariant` and `VariantService.GetPublicVariant`
remain public. They return a `variant` containing only `slug`, normalized
`status`, and `weight`. Names, descriptions, header configuration, axes and
section snapshots are administrator-only. Use
[LandingConfigService.GetLandingConfig](landing.md) to read published marketing
content; experiment selection is not a publication grant.

The old REST `GET /api/v1/variants/select` is no longer a public selector.
`GET /api/v1/public/variants/{slug}` and its section route are retired (404).
The CLI `variants-select` and `public-variant` operations use generated Connect
clients. A missing or inactive public variant returns NotFound.

Administrator endpoint examples below describe the legacy variant configuration
and recovery surface, not typed presentation publication. For current registered
transport paths, consult `.vrooli/endpoints.json`.

---

## Admin Endpoints

### GET /variants

Lists all variants with optional status filter.

**Authentication:** Admin session required

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | string | Filter by status: `active`, `archived`, `deleted` |

**Response:**
```json
{
  "variants": [
    {
      "id": 1,
      "slug": "control",
      "name": "Control (Original)",
      "weight": 50,
      "status": "active",
      "axes": {
        "persona": "ops_leader",
        "jtbd": "launch_bundle",
        "conversionStyle": "self_serve"
      },
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### GET /variants/{slug}

Returns a specific variant by slug (all statuses).

**Authentication:** Admin session required

**Response:** Single variant object

**Errors:**

| Status | Code | Description |
|--------|------|-------------|
| 401 | `AUTH_REQUIRED` | Admin session not provided |
| 404 | `VARIANT_NOT_FOUND` | Variant with slug doesn't exist |

---

### POST /variants

Creates a new variant with sections copied from Control.

**Authentication:** Admin session required

**Request:**
```json
{
  "slug": "variant-b",
  "name": "Variant B",
  "description": "New experimental variant",
  "weight": 25,
  "axes": {
    "persona": "automation_freelancer",
    "jtbd": "scale_services",
    "conversionStyle": "self_serve"
  }
}
```

**Response:** `201 Created` with the created variant

**Errors:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_FAILED` | Missing required fields (`slug`, `name`) |
| 400 | `INVALID_FORMAT` | Slug contains invalid characters (use lowercase, hyphens) |
| 400 | `INVALID_AXIS` | Axis value not in `variant_space.json` |
| 401 | `AUTH_REQUIRED` | Admin session not provided |
| 409 | `DUPLICATE_SLUG` | Slug already exists |

```json
// Example validation error
{
  "error": "Validation failed",
  "code": "VALIDATION_FAILED",
  "details": {
    "fields": {
      "slug": "Slug is required",
      "axes.persona": "Invalid persona: 'invalid' not in variant space"
    }
  }
}
```

**Notes:**
- New variants automatically copy all sections from the Control variant
- Weight is a relative value (doesn't need to sum to 100)

---

### PATCH /variants/{slug}

Updates an existing variant.

**Authentication:** Admin session required

**Request:** (all fields optional)
```json
{
  "name": "Updated Name",
  "description": "Updated description",
  "weight": 30,
  "axes": {
    "persona": "product_marketer"
  },
  "header_config": {
    "show_runtime_pill": true,
    "show_nav_anchors": true
  }
}
```

**Response:** Updated variant object

**Errors:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_FAILED` | Invalid field values |
| 401 | `AUTH_REQUIRED` | Admin session not provided |
| 404 | `VARIANT_NOT_FOUND` | Variant with slug doesn't exist |

---

### POST /variants/{slug}/archive

Archives a variant (remains queryable for analytics but excluded from random selection).

**Authentication:** Admin session required

**Response:**
```json
{
  "message": "Variant archived successfully",
  "slug": "variant-b"
}
```

**Errors:**

| Status | Code | Description |
|--------|------|-------------|
| 401 | `AUTH_REQUIRED` | Admin session not provided |
| 404 | `VARIANT_NOT_FOUND` | Variant with slug doesn't exist |
| 409 | `CANNOT_ARCHIVE` | Cannot archive the Control variant |

**Notes:**
- Archived variants still appear in historical analytics
- Cannot be selected by new visitors
- Can be restored by updating status back to `active`

---

### DELETE /variants/{slug}

Permanently deletes a variant.

**Authentication:** Admin session required

**Response:**
```json
{
  "message": "Variant deleted successfully",
  "slug": "variant-b"
}
```

**Errors:**

| Status | Code | Description |
|--------|------|-------------|
| 401 | `AUTH_REQUIRED` | Admin session not provided |
| 404 | `VARIANT_NOT_FOUND` | Variant with slug doesn't exist |
| 409 | `CANNOT_DELETE` | Cannot delete the Control variant |

**Warning:** This permanently removes the variant and its sections. Historical analytics events are preserved but the variant data is gone.

---

### POST /admin/variants/sync

Re-imports variant snapshot files from disk into Postgres.

**Authentication:** Admin session required

**Response:**
```json
{
  "status": "ok"
}
```

**Notes:**
- Uses the same `VARIANT_SNAPSHOT_MODE` and `VARIANT_SNAPSHOT_PRUNE` settings as startup.
- Intended for applying file edits without restarting the service.

---

## Weight Normalization

Weights are relative, not absolute. They're normalized at selection time:

```
Weights: Control=50, VariantA=30, VariantB=20
Total: 100

Control gets: 50/100 = 50% of traffic
VariantA gets: 30/100 = 30% of traffic
VariantB gets: 20/100 = 20% of traffic
```

If you set weights to 1, 1, 1, each gets 33.3% of traffic.

---

## See Also

- [API Overview](OVERVIEW.md)
- [Sections](sections.md) - Managing section content
- [Metrics](metrics.md) - Analytics per variant
