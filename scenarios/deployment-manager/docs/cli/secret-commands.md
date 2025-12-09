# Secret Commands

Commands for identifying, classifying, and managing secrets for deployment profiles.

Secrets behave differently per deployment tier:
- **Infrastructure secrets** never leave Tier 1
- **Per-install secrets** are generated fresh on each installation
- **User-prompted secrets** are collected during first-run setup

## secrets identify

Identify all secrets required for a profile's target tier(s).

```bash
deployment-manager secrets identify <profile-id>
```

**Arguments:**
- `<profile-id>` - Profile ID

**Example:**

```bash
deployment-manager secrets identify profile-123
```

**Output:**

```json
{
  "profile_id": "profile-123",
  "scenario": "picker-wheel",
  "tier": 2,
  "secrets": [
    {
      "id": "jwt_secret",
      "type": "required",
      "source": "per_install_generated",
      "description": "JWT signing key for authentication tokens",
      "format": "^[A-Za-z0-9]{32,}$",
      "generator": {
        "type": "random",
        "length": 32,
        "charset": "alnum"
      }
    },
    {
      "id": "session_secret",
      "type": "required",
      "source": "user_prompt",
      "description": "Secret for signing cookies and sessions",
      "format": "^[A-Za-z0-9]{32,}$",
      "prompt": {
        "label": "Session Secret",
        "description": "Enter a 32+ character random string for signing cookies."
      }
    },
    {
      "id": "api_key",
      "type": "optional",
      "source": "user_prompt",
      "description": "Optional external API key for enhanced features"
    },
    {
      "id": "postgres_password",
      "type": "infrastructure",
      "source": "infrastructure",
      "description": "Database password - WILL NOT BE BUNDLED",
      "note": "Not needed with sqlite swap applied"
    }
  ],
  "summary": {
    "total": 4,
    "required": 2,
    "optional": 1,
    "infrastructure": 1,
    "bundleable": 3
  }
}
```

**Secret Classifications:**

| Source | Meaning | Bundle Behavior |
|--------|---------|-----------------|
| `per_install_generated` | Auto-generated on first run | Manifest includes generator config |
| `user_prompt` | User provides during setup | Manifest includes prompt config |
| `remote_fetch` | Fetched from external vault | Reference stored, fetched at runtime |
| `infrastructure` | Server-only credentials | **Never included in bundle** |

---

## secrets template

Generate a secrets template for a profile.

```bash
deployment-manager secrets template <profile-id> [--format env|json]
```

**Arguments:**
- `<profile-id>` - Profile ID

**Flags:**
- `--format` - Output format: `env` (default) or `json`

**Example - ENV format:**

```bash
deployment-manager secrets template profile-123 --format env
```

**Output:**

```env
# Picker Wheel Desktop - Secrets Template
# Generated: 2025-01-15T12:00:00Z
# Profile: picker-wheel-desktop (profile-123)
# Tier: 2 (Desktop)

# ============================================
# PER-INSTALL GENERATED
# These secrets are auto-created on first run
# ============================================

# JWT_SECRET - JWT signing key for authentication tokens
# Type: per_install_generated
# Format: 32+ alphanumeric characters
# Status: Auto-generated (no action required)

# ============================================
# USER PROVIDED
# These secrets are prompted during first-run wizard
# ============================================

# SESSION_SECRET - Secret for signing cookies and sessions
# Type: user_prompt
# Format: 32+ alphanumeric characters
SESSION_SECRET=

# API_KEY - Optional external API key for enhanced features
# Type: user_prompt (optional)
# API_KEY=

# ============================================
# INFRASTRUCTURE (excluded from bundle)
# ============================================

# POSTGRES_PASSWORD - Database password
# Status: NOT INCLUDED - Using sqlite swap instead
```

**Example - JSON format:**

```bash
deployment-manager secrets template profile-123 --format json
```

**Output:**

```json
{
  "profile_id": "profile-123",
  "generated_at": "2025-01-15T12:00:00Z",
  "secrets": {
    "per_install_generated": [
      {
        "id": "jwt_secret",
        "description": "JWT signing key for authentication tokens",
        "generator": {"type": "random", "length": 32, "charset": "alnum"}
      }
    ],
    "user_prompt": [
      {
        "id": "session_secret",
        "description": "Secret for signing cookies and sessions",
        "required": true,
        "format": "^[A-Za-z0-9]{32,}$"
      },
      {
        "id": "api_key",
        "description": "Optional external API key for enhanced features",
        "required": false
      }
    ],
    "excluded": [
      {
        "id": "postgres_password",
        "reason": "infrastructure",
        "note": "Not needed with sqlite swap applied"
      }
    ]
  }
}
```

---

## secrets validate

Validate a profile's secrets configuration.

```bash
deployment-manager secrets validate <profile-id>
```

**Arguments:**
- `<profile-id>` - Profile ID

**Example:**

```bash
deployment-manager secrets validate profile-123
```

**Output (success):**

```json
{
  "profile_id": "profile-123",
  "valid": true,
  "checks": [
    {
      "secret": "jwt_secret",
      "status": "pass",
      "details": "Generator configured correctly"
    },
    {
      "secret": "session_secret",
      "status": "pass",
      "details": "Prompt configuration valid"
    },
    {
      "secret": "postgres_password",
      "status": "skip",
      "details": "Infrastructure secret excluded (sqlite swap applied)"
    }
  ],
  "summary": {
    "passed": 2,
    "failed": 0,
    "skipped": 1
  }
}
```

**Output (failure):**

```json
{
  "profile_id": "profile-123",
  "valid": false,
  "checks": [
    {
      "secret": "jwt_secret",
      "status": "fail",
      "details": "Missing generator configuration"
    },
    {
      "secret": "database_url",
      "status": "fail",
      "details": "Infrastructure secret cannot be bundled; apply swap or remove from bundle"
    }
  ],
  "summary": {
    "passed": 1,
    "failed": 2,
    "skipped": 0
  },
  "blockers": [
    "Fix jwt_secret: Add generator configuration",
    "Fix database_url: Apply postgres->sqlite swap or mark as external"
  ]
}
```

**Validation Checks:**
- Required secrets have proper source configuration
- Generators have valid type/length/charset
- Prompts have label and description
- No infrastructure secrets in bundle
- Format regexes are valid
- All referenced secrets exist in scenario

---

## Secret Classes Reference

### per_install_generated

Secrets automatically generated on first app launch.

```json
{
  "id": "jwt_secret",
  "class": "per_install_generated",
  "generator": {
    "type": "random",      // random, uuid, timestamp
    "length": 32,          // for random type
    "charset": "alnum"     // alnum, alpha, numeric, hex, base64
  },
  "target": {
    "type": "env",         // env or file
    "name": "JWT_SECRET"   // env var name or file path
  }
}
```

### user_prompt

Secrets collected from user during first-run wizard.

```json
{
  "id": "session_secret",
  "class": "user_prompt",
  "prompt": {
    "label": "Session Secret",
    "description": "Enter a 32+ character random string",
    "placeholder": "e.g., abc123xyz..."
  },
  "format": "^[A-Za-z0-9]{32,}$",  // validation regex
  "target": {
    "type": "env",
    "name": "SESSION_SECRET"
  }
}
```

### remote_fetch

Secrets fetched from external vault at runtime.

```json
{
  "id": "stripe_key",
  "class": "remote_fetch",
  "vault": {
    "type": "hashicorp",
    "path": "secret/data/stripe",
    "key": "api_key"
  },
  "target": {
    "type": "env",
    "name": "STRIPE_API_KEY"
  }
}
```

### infrastructure

Server-only credentials that never leave Tier 1.

```json
{
  "id": "postgres_password",
  "class": "infrastructure",
  "note": "Required for Tier 1 only; use sqlite swap for desktop"
}
```

---

## Related

- [Profile Commands](profile-commands.md) - Profiles hold secrets configuration
- [Secrets Management Guide](../guides/secrets-management.md) - Detailed strategies
- [Desktop Workflow](../workflows/desktop-deployment.md) - How secrets flow in desktop deployment
