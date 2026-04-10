# API Reference

All endpoints are served under the API port allocated by Vrooli. Routes are registered in [CODE: api/handlers/brands.go#RegisterRoutes].

## Health

### `GET /health`

Returns service health status including database connectivity.

**Implementation**: [CODE: api/main.go] (health handler via `api-core/health`)

**Response** (200):
```json
{
  "status": "healthy",
  "service": "brand-manager",
  "version": "1.0.0",
  "readiness": true,
  "timestamp": "2026-03-26T12:00:00Z",
  "dependencies": {
    "sqlite": "healthy"
  }
}
```

---

## Brands

### `POST /api/v1/brands`

Create a new brand. [REQ: BM-REQ-CRUD-CREATE]

**Implementation**: [CODE: api/handlers/brands.go#CreateBrand]

**Request Body**:
```json
{
  "name": "My Brand",
  "description": "Optional description",
  "identity": { "display_name": "My Brand", "tagline": "..." },
  "colors": { "primary": "#3B82F6", "background": "#0F172A" },
  "typography": { "heading_font": "Inter", "body_font": "Inter" },
  "voice": { "tone": "professional", "style": "concise" }
}
```

**Required fields**: `name`

**Response** (201): Created brand with `id`, `version: 1`, timestamps.

### `GET /api/v1/brands`

List all brands with optional filtering. [REQ: BM-REQ-CRUD-READ]

**Implementation**: [CODE: api/handlers/brands.go#ListBrands]

**Query Parameters**:
| Param | Type | Description |
|-------|------|-------------|
| `name` | string | Filter by name (contains match) |
| `limit` | int | Max results to return |
| `offset` | int | Skip N results for pagination |

**Response** (200): Array of brand objects.

### `GET /api/v1/brands/{id}`

Get a single brand by ID. [REQ: BM-REQ-CRUD-READ]

**Implementation**: [CODE: api/handlers/brands.go#GetBrand]

**Response** (200): Brand object. (404 if not found.)

### `PUT /api/v1/brands/{id}`

Update a brand. Only non-zero fields are merged. [REQ: BM-REQ-CRUD-UPDATE]

**Implementation**: [CODE: api/handlers/brands.go#UpdateBrand]

**Request Body**: Same structure as create. Only provided fields are updated.

**Response** (200): Updated brand with incremented `version`. A new version snapshot is created automatically.

### `DELETE /api/v1/brands/{id}`

Delete a brand and all associated versions and assignments (CASCADE).

**Implementation**: [CODE: api/handlers/brands.go#DeleteBrand]

**Response** (204): No content. (404 if not found.)

---

## Versions

### `GET /api/v1/brands/{id}/versions`

List all version snapshots for a brand, ordered newest first. [REQ: BM-REQ-CRUD-VERSION]

**Implementation**: [CODE: api/handlers/brands.go#ListVersions]

**Response** (200):
```json
[
  {
    "id": "uuid",
    "brand_id": "uuid",
    "version": 2,
    "snapshot": "{...full brand JSON...}",
    "created_at": "2026-03-26T12:00:00Z"
  }
]
```

---

## Assignments

### `POST /api/v1/assignments`

Assign a brand to a scenario. One brand per scenario (upserts). [REQ: BM-REQ-ASSIGN-LINK]

**Implementation**: [CODE: api/handlers/brands.go#CreateAssignment]

**Request Body**:
```json
{
  "brand_id": "uuid",
  "scenario_name": "my-scenario",
  "elements": ["colors", "typography"]
}
```

**Required fields**: `brand_id`, `scenario_name`

**Response** (201): Assignment with captured `brand_version`.

### `DELETE /api/v1/assignments/{id}`

Remove a brand assignment.

**Implementation**: [CODE: api/handlers/brands.go#DeleteAssignment]

**Response** (204): No content. (404 if not found.)

---

## Scenario Status

### `GET /api/v1/scenarios/{name}/status`

Check branding status for a scenario. [REQ: BM-REQ-API-STATUS]

**Implementation**: [CODE: api/handlers/brands.go#GetScenarioStatus]

**Response** (200) — branded:
```json
{
  "scenario": "my-scenario",
  "has_brand": true,
  "brand_id": "uuid",
  "brand_version": 2,
  "elements": ["colors", "typography"],
  "applied_at": "2026-03-26T12:00:00Z"
}
```

**Response** (200) — unbranded:
```json
{
  "scenario": "my-scenario",
  "has_brand": false,
  "brand_id": null,
  "brand_version": null
}
```

---

## WCAG Contrast Validation

### `POST /api/v1/contrast/check`

Check WCAG AA contrast ratio for a single foreground/background color pair. [REQ: BM-REQ-WCAG-CALC] [REQ: BM-REQ-WCAG-VALIDATE]

**Implementation**: [CODE: api/handlers/contrast.go#CheckContrast]

**Request Body**:
```json
{
  "foreground": "#1A202C",
  "background": "#FFFFFF"
}
```

**Response** (200):
```json
{
  "foreground": "#1A202C",
  "background": "#FFFFFF",
  "ratio": 16.32,
  "aa_normal": true,
  "aa_large": true
}
```

### `POST /api/v1/contrast/brand`

Validate all standard WCAG AA pairings for a brand's color palette. [REQ: BM-REQ-WCAG-VALIDATE] [REQ: BM-REQ-WCAG-REJECT]

**Implementation**: [CODE: api/handlers/contrast.go#CheckBrandContrast]

**Request Body**:
```json
{
  "primary": "#1a365d",
  "secondary": "#2d3748",
  "accent": "#8B0000",
  "background": "#FFFFFF",
  "surface": "#F7FAFC",
  "text": "#1A202C"
}
```

**Pairings checked**: text-on-background, text-on-surface, primary-on-background, primary-on-surface, accent-on-background.

**Response** (200):
```json
{
  "pairs": [
    { "foreground": "#1A202C", "background": "#FFFFFF", "ratio": 16.32, "aa_normal": true, "aa_large": true },
    { "foreground": "#1a365d", "background": "#FFFFFF", "ratio": 12.14, "aa_normal": true, "aa_large": true }
  ],
  "pass_all": true
}
```
