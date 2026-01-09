---
title: "Section Endpoints"
description: "Content section management APIs"
category: "reference"
order: 4
audience: ["developers"]
---

# Section Endpoints

Endpoints for managing landing page content sections.

## Section Types

| Type | Description |
|------|-------------|
| `hero` | Main headline, subheadline, primary CTA |
| `features` | Product feature grid |
| `pricing` | Pricing tiers and comparison |
| `testimonials` | Customer quotes |
| `faq` | Frequently asked questions |
| `cta` | Secondary call-to-action block |
| `video` | Embedded video content |
| `downloads` | Download rail for apps |
| `footer` | Links, copyright, social |

---

## Public Endpoints

### GET /public/variants/{variant_id}/sections

Returns enabled sections for public display.

**Authentication:** None

**Response:**
```json
{
  "sections": [
    {
      "id": 1,
      "variant_id": 1,
      "section_type": "hero",
      "content": {
        "headline": "Build Landing Pages Fast",
        "subheadline": "Production-ready in minutes",
        "cta_text": "Get Started",
        "cta_url": "/signup"
      },
      "order": 1,
      "enabled": true
    },
    {
      "id": 2,
      "variant_id": 1,
      "section_type": "features",
      "content": {
        "title": "Features",
        "items": [
          {
            "icon": "Zap",
            "title": "Fast",
            "description": "Generate in 60 seconds"
          }
        ]
      },
      "order": 2,
      "enabled": true
    }
  ]
}
```

---

## Admin Endpoints

### GET /variants/{variant_id}/sections

Returns all sections for a variant (including disabled).

**Authentication:** Admin session required

**Response:** Same as public endpoint, but includes `enabled: false` sections

---

### GET /sections/{id}

Returns a single section by ID.

**Authentication:** Admin session required

**Response:**
```json
{
  "id": 1,
  "variant_id": 1,
  "section_type": "hero",
  "content": {
    "headline": "Build Landing Pages Fast",
    "subheadline": "Production-ready in minutes",
    "cta_text": "Get Started",
    "cta_url": "/signup"
  },
  "order": 1,
  "enabled": true,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Errors:**
- `404 Not Found` - Section not found

---

### PATCH /sections/{id}

Updates section content. Changes reflect immediately (live preview).

**Authentication:** Admin session required

**Request:**
```json
{
  "content": {
    "headline": "Updated Headline",
    "subheadline": "Updated subheadline",
    "cta_text": "New CTA"
  }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Section updated successfully"
}
```

**Notes:**
- Only the `content` field is updated
- Changes take effect immediately
- The admin preview shows updates within 300ms

---

### POST /sections

Creates a new section.

**Authentication:** Admin session required

**Request:**
```json
{
  "variant_id": 1,
  "section_type": "features",
  "content": {
    "title": "Features",
    "items": [
      {
        "icon": "Zap",
        "title": "Fast",
        "description": "Generate in 60 seconds"
      }
    ]
  },
  "order": 2,
  "enabled": true
}
```

**Response:** `201 Created` with created section

**Errors:**
- `400 Bad Request` - Invalid section_type or missing required fields

---

### DELETE /sections/{id}

Deletes a section.

**Authentication:** Admin session required

**Response:**
```json
{
  "success": true,
  "message": "Section deleted successfully"
}
```

**Errors:**
- `404 Not Found` - Section not found

---

## Content Schemas

Each section type has a specific content schema. See `.vrooli/schemas/sections/*.json` in the template for complete definitions.

### Hero Schema

```json
{
  "headline": "string (required)",
  "subheadline": "string",
  "cta_text": "string",
  "cta_url": "string",
  "background_image_url": "string",
  "background_gradient": "string"
}
```

### Features Schema

```json
{
  "title": "string",
  "subtitle": "string",
  "items": [
    {
      "icon": "string (Lucide icon name)",
      "title": "string",
      "description": "string"
    }
  ]
}
```

### Pricing Schema

```json
{
  "title": "string",
  "subtitle": "string",
  "tiers": [
    {
      "name": "string",
      "price": "number",
      "interval": "month | year",
      "features": ["string"],
      "cta_text": "string",
      "highlighted": "boolean"
    }
  ]
}
```

---

## Ordering Sections

Section order is controlled by the `order` field. When reordering:

1. Fetch all sections for the variant
2. Update `order` values as needed
3. PATCH each section with new order

Example: Move section from position 3 to position 1:
```bash
# Section 3 → 1, Section 1 → 2, Section 2 → 3
curl -X PATCH .../sections/3 -d '{"content": {...}, "order": 1}'
```

---

## See Also

- [API Overview](README.md)
- [Variants](variants.md) - Managing variants that contain sections
- [Agent Guide](../../api/templates/AGENT.md) - Adding new section types
