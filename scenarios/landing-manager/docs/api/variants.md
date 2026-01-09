---
title: "Variant Endpoints"
description: "A/B testing variant management APIs"
category: "reference"
order: 3
audience: ["developers"]
---

# Variant Endpoints

Endpoints for managing A/B testing variants.

## Public Endpoints

### GET /variants/select

Selects a variant based on configured weights.

**Authentication:** None

**Response:**
```json
{
  "id": 1,
  "slug": "control",
  "name": "Control (Original)",
  "description": "Original landing page design",
  "weight": 50,
  "status": "active",
  "axes": {
    "persona": "ops_leader",
    "jtbd": "launch_bundle",
    "conversionStyle": "demo_led"
  }
}
```

### GET /public/variants/{slug}

Returns a specific variant by slug (public access, active variants only).

**Authentication:** None

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `slug` | string | The variant slug |

**Response:** Same as `/variants/select`

**Errors:**
- `404 Not Found` - Variant not found or not active

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
- `400 Bad Request` - Missing required fields or invalid axes
- `409 Conflict` - Slug already exists

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

**Warning:** This permanently removes the variant and its sections. Historical analytics events are preserved but the variant data is gone.

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

- [API Overview](README.md)
- [Sections](sections.md) - Managing section content
- [Metrics](metrics.md) - Analytics per variant
