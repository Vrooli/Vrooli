---
title: "Admin Endpoints"
description: "Admin portal management APIs (auth, branding, assets, remote profiles)"
category: "reference"
order: 7
audience: ["developers"]
---

# Admin Endpoints

Endpoints for admin portal authentication, branding, asset management, and remote profile management.

## Authentication

### POST /admin/login

Authenticates an admin user.

**Authentication:** None

**Request:**
```json
{
  "email": "admin@localhost",
  "password": "<replace-at-deploy>"
}
```

**Response:**
```json
{
  "authenticated": true,
  "email": "admin@localhost",
  "reset_enabled": false
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
  "email": "admin@localhost",
  "reset_enabled": false
}
```

Or when not authenticated:
```json
{
  "authenticated": false,
  "email": null,
  "reset_enabled": false
}
```

---

### GET /admin/profile

Returns the authenticated admin profile and whether defaults are still in use.

**Authentication:** Admin session required

**Response:**
```json
{
  "email": "admin@localhost",
  "is_default_email": true,
  "is_default_password": true
}
```

---

### PUT /admin/profile

Updates the admin email and/or password. `current_password` is required for all changes.

**Authentication:** Admin session required

**Request (update email):**
```json
{
  "current_password": "changeme123",
  "new_email": "owner@example.com"
}
```

**Request (update password):**
```json
{
  "current_password": "changeme123",
  "new_password": "StrongerPass123!"
}
```

**Response:**
```json
{
  "email": "owner@example.com",
  "is_default_email": false,
  "is_default_password": false
}
```

**Errors:**
- `400 Bad Request` - Missing fields or weak password
- `401 Unauthorized` - Current password invalid
- `409 Conflict` - Email already in use

---

## Remote Profiles

Remote profiles let the admin UI/CLI manage a deployed LPBS instance by storing an encrypted
`admin_session` cookie and proxying allowlisted admin requests. Remote sessions are encrypted
at rest using `LPBS_REMOTE_PROFILE_ENCRYPTION_KEY` (or `LPBS_API_KEY_ENCRYPTION_KEY` fallback).

### GET /admin/remote-profiles

Lists configured remote profiles.

**Authentication:** Admin session required

**Response:**
```json
{
  "profiles": [
    {
      "id": 1,
      "tag": "prod",
      "label": "Production",
      "api_base": "https://example.com/api/v1",
      "connector_id": "b88495f9f4c14e5c90183f5f6f8d0b2a",
      "remote_session_id": "af9d9f4b...",
      "status": "active",
      "has_session": true,
      "session_expires_at": "2026-02-04T18:15:00Z",
      "last_login_at": "2026-02-04T17:15:00Z",
      "last_used_at": "2026-02-04T17:45:00Z",
      "created_by": 1,
      "created_at": "2026-02-04T17:00:00Z",
      "updated_at": "2026-02-04T17:45:00Z"
    }
  ]
}
```

---

### POST /admin/remote-profiles

Creates a new remote profile.

**Authentication:** Admin session required

**Request:**
```json
{
  "tag": "prod",
  "label": "Production",
  "api_base": "https://example.com/api/v1"
}
```

**Notes:**
- `api_base` must end with `/api/v1` (the remote LPBS API base)
- HTTPS is required in production environments

**Response:** Remote profile object (see list response)

**Errors:**
- `409 Conflict` - Tag already exists
- `400 Bad Request` - Invalid tag or api_base

---

### PUT /admin/remote-profiles/{id}

Updates a remote profile.

**Authentication:** Admin session required

**Request (partial fields allowed):**
```json
{
  "label": "Production (VPS)",
  "api_base": "https://example.com/api/v1"
}
```

**Response:** Updated remote profile object

---

### DELETE /admin/remote-profiles/{id}

Deletes a remote profile.

**Authentication:** Admin session required

**Response:**
```json
{
  "success": true
}
```

---

### POST /admin/remote-profiles/{id}/login

Logs in to the remote LPBS instance and stores the remote session cookie.

**Authentication:** Admin session required

**Request:**
```json
{
  "email": "admin@localhost",
  "password": "your-remote-admin-password"
}
```

**Response:** Updated remote profile object

**Errors:**
- `401 Unauthorized` - Remote credentials invalid

---

### POST /admin/remote-profiles/{id}/logout

Clears the stored remote session cookie.

**Authentication:** Admin session required

**Response:** Updated remote profile object

---

### POST /admin/remote-profiles/{id}/test

Validates the stored remote session.

**Authentication:** Admin session required

**Response:** Updated remote profile object

**Errors:**
- `401 Unauthorized` - Remote session expired

---

### GET /admin/remote-profiles/{id}/session-links

Returns "both sides" session visibility for one profile:
- local stored-session status (`has_session`, local status, expiry)
- remote-side sessions discovered on the connected LPBS deployment for this profile's connector id

**Authentication:** Admin session required

**Response:**
```json
{
  "profile_id": 1,
  "profile_tag": "prod",
  "connector_id": "b88495f9f4c14e5c90183f5f6f8d0b2a",
  "local_has_session": true,
  "local_status": "active",
  "local_session_expires_at": "2026-02-04T18:15:00Z",
  "remote_session_id": "af9d9f4b...",
  "remote_sessions": [
    {
      "session_id": "af9d9f4b...",
      "admin_email": "admin@localhost",
      "connector_id": "b88495f9f4c14e5c90183f5f6f8d0b2a",
      "profile_tag": "prod",
      "origin": "lpbs-local",
      "created_at": "2026-02-04T17:15:00Z",
      "last_activity": "2026-02-04T17:45:00Z",
      "expires_at": "2026-02-11T17:15:00Z"
    }
  ]
}
```

---

### POST /admin/remote-profiles/{id}/remote-revoke

Revokes all remote-side sessions linked to this profile connector, then clears the local stored session.

**Authentication:** Admin session required

**Response:** Same shape as `GET /admin/remote-profiles/{id}/session-links`

---

### GET /admin/remote-profile-sessions

Lists incoming connector sessions on the current LPBS instance (this is what the deployed LPBS uses to show and revoke remote-profile logins).

**Authentication:** Admin session required

**Query params:**
- `connector_id` (optional): filter to one connector

**Response:**
```json
{
  "sessions": [
    {
      "session_id": "af9d9f4b...",
      "admin_email": "admin@localhost",
      "connector_id": "b88495f9f4c14e5c90183f5f6f8d0b2a",
      "profile_tag": "prod",
      "origin": "lpbs-local",
      "created_at": "2026-02-04T17:15:00Z",
      "last_activity": "2026-02-04T17:45:00Z",
      "expires_at": "2026-02-11T17:15:00Z",
      "ip_address": "203.0.113.10",
      "user_agent": "LPBS-RemoteProfile/1 connector_id=..."
    }
  ]
}
```

---

### DELETE /admin/remote-profile-sessions/{session_id}

Revokes one incoming connector session on the current LPBS instance.

**Authentication:** Admin session required

**Response:**
```json
{
  "success": true
}
```

---

### POST /admin/remote-profiles/{id}/proxy

Proxies an allowlisted remote admin request using the stored remote session cookie.

**Authentication:** Admin session required

**Request:**
```json
{
  "method": "POST",
  "path": "/admin/download-artifacts/commit",
  "query": { "bundle_key": "landing-page" },
  "headers": { "Content-Type": "application/json" },
  "body": {
    "artifact_id": 42,
    "app_key": "landing-suite",
    "platform": "windows"
  }
}
```

**Allowlisted paths (prefix match):**
- `/admin/download-storage`
- `/admin/download-artifacts`
- `/admin/download-assets`
- `/admin/download-apps`

**Response:** Pass-through status + body from remote LPBS

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
- [Admin Guide](../../guides/ADMIN_GUIDE.md) - Using the admin portal
- [Payments](payments.md) - Stripe settings
