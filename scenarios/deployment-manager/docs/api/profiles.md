# Profile Endpoints

Endpoints for managing deployment profiles.

## GET /profiles

List all deployment profiles.

**Request:**

```bash
curl "http://localhost:${API_PORT}/api/v1/profiles"
```

**Response:**

```json
{
  "profiles": [
    {
      "id": "profile-1704067200",
      "name": "picker-wheel-desktop",
      "scenario": "picker-wheel",
      "tiers": [2],
      "version": 3,
      "created_at": "2025-01-15T12:00:00Z",
      "updated_at": "2025-01-15T14:30:00Z"
    }
  ],
  "total": 1
}
```

---

## POST /profiles

Create a new deployment profile.

**Request:**

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/profiles" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "picker-wheel-desktop",
    "scenario": "picker-wheel",
    "tiers": [2]
  }'
```

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Profile name |
| `scenario` | string | Yes | Target scenario |
| `tiers` | array[int] | Yes | Target tier(s): 1-5 |
| `swaps` | object | No | Initial dependency swaps |
| `settings` | object | No | Initial settings |

**Response:**

```json
{
  "id": "profile-1704067200",
  "name": "picker-wheel-desktop",
  "scenario": "picker-wheel",
  "tiers": [2],
  "swaps": {},
  "secrets": {},
  "settings": {},
  "version": 1,
  "created_at": "2025-01-15T12:00:00Z",
  "updated_at": "2025-01-15T12:00:00Z",
  "created_by": "system",
  "updated_by": "system"
}
```

---

## GET /profiles/{id}

Retrieve a profile by ID.

**Request:**

```bash
curl "http://localhost:${API_PORT}/api/v1/profiles/profile-1704067200"
```

**Response:**

```json
{
  "id": "profile-1704067200",
  "name": "picker-wheel-desktop",
  "scenario": "picker-wheel",
  "tiers": [2],
  "swaps": {
    "postgres": "sqlite"
  },
  "secrets": {
    "jwt_secret": { "class": "per_install_generated" },
    "session_secret": { "class": "user_prompt" }
  },
  "settings": {
    "env": {
      "LOG_LEVEL": "info",
      "DEBUG": "false"
    }
  },
  "version": 5,
  "created_at": "2025-01-15T12:00:00Z",
  "updated_at": "2025-01-15T14:30:00Z",
  "created_by": "system",
  "updated_by": "system"
}
```

---

## PUT /profiles/{id}

Update a profile.

**Request:**

```bash
curl -X PUT "http://localhost:${API_PORT}/api/v1/profiles/profile-1704067200" \
  -H "Content-Type: application/json" \
  -d '{
    "tiers": [2, 3],
    "swaps": {
      "postgres": "sqlite",
      "redis": "in-process"
    },
    "settings": {
      "env": {
        "LOG_LEVEL": "debug"
      }
    }
  }'
```

**Request Body:** Partial update - only include fields to change.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Profile name |
| `tiers` | array[int] | Target tiers |
| `swaps` | object | Dependency swaps (replaces existing) |
| `secrets` | object | Secret configuration |
| `settings` | object | Settings (merged with existing) |

**Response:** Updated profile object.

**Notes:**
- Each update increments the `version` field
- Previous versions are preserved for rollback

---

## DELETE /profiles/{id}

Delete a profile.

**Request:**

```bash
curl -X DELETE "http://localhost:${API_PORT}/api/v1/profiles/profile-1704067200"
```

**Response:**

```json
{
  "status": "deleted",
  "id": "profile-1704067200"
}
```

---

## GET /profiles/{id}/versions

Get version history for a profile.

**Request:**

```bash
curl "http://localhost:${API_PORT}/api/v1/profiles/profile-1704067200/versions"
```

**Response:**

```json
{
  "profile_id": "profile-1704067200",
  "current_version": 5,
  "versions": [
    {
      "version": 1,
      "updated_at": "2025-01-15T12:00:00Z",
      "updated_by": "system",
      "changes": "Created profile"
    },
    {
      "version": 2,
      "updated_at": "2025-01-15T12:30:00Z",
      "updated_by": "system",
      "changes": "Added postgres->sqlite swap"
    },
    {
      "version": 3,
      "updated_at": "2025-01-15T13:00:00Z",
      "updated_by": "system",
      "changes": "Updated settings.env"
    }
  ]
}
```

---

## GET /profiles/{id}/validate

Validate a profile for deployment readiness.

**Request:**

```bash
curl "http://localhost:${API_PORT}/api/v1/profiles/profile-1704067200/validate"
```

**Response:**

```json
{
  "profile_id": "profile-1704067200",
  "valid": true,
  "checks": [
    { "name": "fitness_threshold", "status": "pass", "details": "Score 70 >= 50" },
    { "name": "secrets_complete", "status": "pass", "details": "All required secrets configured" },
    { "name": "licensing", "status": "pass", "details": "All dependencies OSS-compatible" },
    { "name": "resource_limits", "status": "pass", "details": "RAM 256MB within limit" },
    { "name": "platform_binaries", "status": "pass", "details": "Binaries for all platforms" },
    { "name": "dependency_swaps", "status": "pass", "details": "All blockers resolved" }
  ],
  "ready_for_deployment": true
}
```

---

## GET /profiles/{id}/cost-estimate

Estimate deployment costs (for tier 4/5).

**Request:**

```bash
curl "http://localhost:${API_PORT}/api/v1/profiles/profile-456/cost-estimate"
```

**Response:**

```json
{
  "profile_id": "profile-456",
  "tier": 4,
  "estimates": {
    "monthly": {
      "compute": 45.00,
      "storage": 10.00,
      "bandwidth": 15.00,
      "total": 70.00,
      "currency": "USD"
    }
  },
  "breakdown": {
    "compute": {
      "instance_type": "t3.medium",
      "vcpu": 2,
      "memory_gb": 4
    }
  }
}
```

---

## Profile Data Model

```json
{
  "id": "profile-{timestamp}",
  "name": "human-readable-name",
  "scenario": "scenario-name",
  "tiers": [2, 3],
  "swaps": {
    "original-dep": "replacement-dep"
  },
  "secrets": {
    "secret-id": {
      "class": "per_install_generated|user_prompt|infrastructure",
      "value": "optional-value"
    }
  },
  "settings": {
    "env": { "KEY": "value" },
    "custom": "any-value"
  },
  "version": 1,
  "created_at": "ISO8601",
  "updated_at": "ISO8601",
  "created_by": "system|user-id",
  "updated_by": "system|user-id"
}
```

---

## Related

- [CLI Profile Commands](../cli/profile-commands.md) - CLI interface
- [Swaps Endpoints](swaps.md) - Swap analysis before applying
- [Bundles Endpoints](bundles.md) - Generate manifests from profiles
