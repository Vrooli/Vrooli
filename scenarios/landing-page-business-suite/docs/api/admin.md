---
title: "Admin Endpoints"
description: "Admin portal management APIs"
category: "reference"
order: 7
audience: ["developers"]
---

# Admin Endpoints

Endpoints for admin portal authentication, branding, and asset management.

## Authentication

### POST /admin/login

Authenticates an admin user.

**Authentication:** None

**Request:**
```json
{
  "email": "admin@localhost",
  "password": "admin123"
}
```

**Response:**
```json
{
  "success": true,
  "email": "admin@localhost"
}
```

Sets session cookie for subsequent requests.

**Errors:**
- `401 Unauthorized` - Invalid credentials

---

### POST /admin/logout

Logs out the current admin session.

**Authentication:** Admin session required

**Response:**
```json
{
  "success": true,
  "message": "Logged out"
}
```

---

### GET /admin/session

Checks if current session is valid.

**Authentication:** None (returns auth status)

**Response:**
```json
{
  "authenticated": true,
  "email": "admin@localhost"
}
```

Or when not authenticated:
```json
{
  "authenticated": false,
  "email": null
}
```

---

## Branding

### GET /admin/branding

Returns full branding configuration.

**Authentication:** Admin session required

**Response:**
```json
{
  "site_name": "My Landing",
  "tagline": "Your tagline here",
  "logo_url": "/uploads/logo.png",
  "favicon_url": "/uploads/favicon.ico",
  "default_title": "My Landing - Home",
  "default_description": "Build amazing products with our platform",
  "theme_primary_color": "#F97316",
  "canonical_base_url": "https://example.com",
  "robots_txt": "User-agent: *\nAllow: /"
}
```

---

### PUT /admin/branding

Updates branding settings.

**Authentication:** Admin session required

**Request:**
```json
{
  "site_name": "My Landing",
  "tagline": "Your tagline",
  "logo_url": "/uploads/logo.png",
  "favicon_url": "/uploads/favicon.ico",
  "default_title": "My Landing - Home",
  "default_description": "...",
  "theme_primary_color": "#F97316",
  "canonical_base_url": "https://example.com",
  "robots_txt": "User-agent: *\nAllow: /"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Branding updated"
}
```

---

### POST /admin/branding/clear-field

Clears a specific branding field (resets to default).

**Authentication:** Admin session required

**Request:**
```json
{
  "field": "logo_url"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Field cleared"
}
```

---

## Variant SEO

### PUT /admin/variants/{slug}/seo

Updates SEO configuration for a variant.

**Authentication:** Admin session required

**Request:**
```json
{
  "title": "Custom Page Title",
  "description": "Custom meta description",
  "og_image_url": "/uploads/og-image.png",
  "canonical_url": "https://example.com/variant"
}
```

**Response:**
```json
{
  "success": true,
  "message": "SEO updated"
}
```

---

## Asset Management

### GET /admin/assets

Lists uploaded assets.

**Authentication:** Admin session required

**Response:**
```json
{
  "assets": [
    {
      "id": 1,
      "filename": "logo.png",
      "url": "/uploads/logo.png",
      "category": "logo",
      "alt_text": "Company logo",
      "size_bytes": 24567,
      "mime_type": "image/png",
      "uploaded_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

---

### POST /admin/assets/upload

Uploads a new asset file.

**Authentication:** Admin session required

**Content-Type:** `multipart/form-data`

**Form Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `file` | file | The file to upload |
| `category` | string | Optional: `logo`, `favicon`, `og_image`, `general` |
| `alt_text` | string | Optional: alt text for accessibility |

**Response:**
```json
{
  "id": 2,
  "url": "/uploads/new-image.png",
  "filename": "new-image.png"
}
```

**Example:**
```bash
curl -X POST http://localhost:3000/api/v1/admin/assets/upload \
  -H "Cookie: session=xxx" \
  -F "file=@logo.png" \
  -F "category=logo" \
  -F "alt_text=Company Logo"
```

---

### GET /admin/assets/{id}

Returns asset metadata.

**Authentication:** Admin session required

**Response:** Single asset object

---

### DELETE /admin/assets/{id}

Deletes an asset.

**Authentication:** Admin session required

**Response:**
```json
{
  "success": true,
  "message": "Asset deleted"
}
```

---

## Download App Management

### GET /admin/download-apps

Lists all download apps with their assets.

**Authentication:** Admin session required

**Response:**
```json
{
  "apps": [
    {
      "app_key": "vrooli-pro",
      "name": "Vrooli Pro",
      "description": "The complete automation suite",
      "enabled": true,
      "platforms": {
        "windows": {
          "download_url": "https://...",
          "version": "1.2.3"
        },
        "mac": {
          "download_url": "https://...",
          "version": "1.2.3"
        }
      },
      "store_links": {
        "app_store": "https://apps.apple.com/...",
        "google_play": "https://play.google.com/..."
      },
      "release_notes": "Bug fixes and improvements"
    }
  ]
}
```

---

### POST /admin/download-apps

Creates a new download app.

**Authentication:** Admin session required

**Request:**
```json
{
  "app_key": "new-app",
  "name": "New App",
  "description": "App description",
  "platforms": { ... },
  "store_links": { ... }
}
```

---

### PUT /admin/download-apps/{app_key}

Updates a download app.

**Authentication:** Admin session required

**Request:** Same as POST

---

### DELETE /admin/download-apps/{app_key}

Deletes a download app and its installer assets.

**Authentication:** Admin session required

**Response:**
```json
{
  "success": true
}
```

---

## System

### POST /admin/reset-demo-data

Resets all data to demo defaults.

**Authentication:** Admin session required

**Environment:** Requires `ENABLE_ADMIN_RESET=true`

**Response:**
```json
{
  "success": true,
  "message": "Demo data reset complete"
}
```

**Warning:** This deletes all variants, sections, metrics, and subscriptions.

---

## See Also

- [API Overview](README.md)
- [Admin Guide](../ADMIN_GUIDE.md) - Using the admin portal
- [Payments](payments.md) - Stripe settings
