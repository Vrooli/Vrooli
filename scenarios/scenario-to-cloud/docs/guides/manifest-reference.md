# Manifest Reference

> [CODE: api/domain/manifest.go] — CloudManifest type definition
> [CODE: api/manifest/validator.go] — Validation and normalization logic

Complete reference for the deployment manifest configuration.

Canonical source:
```bash
scenario-to-cloud manifest schema
```

## Schema Overview

```json
{
  "version": "...",
  "scenario": { ... },
  "target": { ... },
  "edge": { ... },
  "ports": { ... },
  "dependencies": { ... },
  "bundle": { ... }
}
```

## Scenario Section

Identifies which scenario to deploy.

```json
{
  "scenario": {
    "id": "my-scenario"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Scenario directory name |

## Target Section

Defines where to deploy.

```json
{
  "target": {
    "type": "vps",
    "vps": {
      "host": "192.168.1.100",
      "user": "root",
      "port": 22
    }
  }
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | Yes | - | Deployment target type (`vps`) |
| `vps.host` | string | Yes | - | Hostname or IP address |
| `vps.user` | string | No | `root` | SSH username |
| `vps.port` | number | No | `22` | SSH port |

There is no key field. Target access is a Bridge enrollment or the credential binding `vrooli/scenario-to-cloud:ssh-key` (see [VPS Setup](vps-setup.md) "Target Access"); a manifest that carries `key_path` is refused by the schema.

## Edge Section

Configures public access and HTTPS.

```json
{
  "edge": {
    "domain": "app.example.com",
    "dns_policy": "required",
    "caddy": {
      "enabled": true,
      "email": "admin@example.com"
    }
  }
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `domain` | string | Yes | - | Public domain name |
| `dns_policy` | string | No | `required` | DNS enforcement mode: `required`, `warn`, or `skip` |
| `caddy.enabled` | boolean | No | `true` | Enable Caddy reverse proxy |
| `caddy.email` | string | No | - | Email for Let's Encrypt |

## Ports Section

Override default port mappings.

```json
{
  "ports": {
    "ui": 3000,
    "api": 8080,
    "ws": 8081
  }
}
```

Ports are optional. If not specified, the scenario's default ports are used.

## Dependencies Section

Declare required resources and scenarios.

```json
{
  "dependencies": {
    "resources": ["postgres", "redis"],
    "scenarios": []
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `resources` | string[] | Resource IDs to start |
| `scenarios` | string[] | Dependent scenario IDs |

## Secrets Section

`secrets.bundle_secrets[]` declares the credentials a deployment needs. Two
fields decide what happens when one is unsatisfied, and they are not
alternatives:

| Field | Effect when nothing satisfies the secret |
|-------|------------------------------------------|
| `required: true` | **Blocks.** The plan compiles as `needs_input` with an operator handoff, and no action runs. |
| `capability: "<name>"` | **Warns.** The plan still applies and carries an advisory naming the capability. |
| neither | **Silent.** Nothing mentions it. |

```json
{
  "secrets": {
    "bundle_secrets": [
      {
        "id": "sendgrid-api-key",
        "class": "user_prompt",
        "required": false,
        "capability": "customer sign-in email",
        "description": "Restricted SendGrid API key for transactional sign-in email delivery.",
        "target": { "type": "env", "name": "SENDGRID_API_KEY" },
        "descriptor": { "logical_id": "vrooli/landing-page-business-suite", "field": "sendgrid-api-key" }
      }
    ]
  }
}
```

Declare `capability` for any secret that is optional for the deployment but
load-bearing for something a customer pays for: mail delivery, payments,
storage. Without it, "optional" also means "unmentioned" — the condition that
let a deployment ship with customer sign-in email unconfigured and review as
clean.

Write the capability in operator words ("customer sign-in email", not
"SENDGRID_API_KEY"): it is printed to a person deciding whether to approve the
deploy. `capability` on a `required` secret is ignored, because a blocker is
already reported.

Advisories are excluded from the plan's semantic digest, so a warning
appearing or clearing between review and apply never invalidates the review.

## Identity topology

The cloud manifest selects what is deployed; it does not invent a second
authentication schema. The target scenario's declared authentication metadata
and the dependency snapshot together determine the identity topology.

Use these rules:

| Deployment intent | Manifest/dependency behavior |
|---|---|
| Public hosted product | Reference the approved hosted or external identity provider. Do not add a private authenticator instance per request or per customer without an explicit tenant design. |
| Self-hosted LPBS | Include `scenario-authenticator` in the analyzer-derived scenario dependency snapshot and bundle it in the same mini-Vrooli installation. Treat its realm and signing keys as installation data. |
| Single-user private app | Keep the product in `personal_local` unless the operator explicitly enables multi-user or remote identity. Do not require an LPBS website login merely because the artifact was downloaded. |
| Multi-user installation | Declare `local_multi_user` or the appropriate remote/shared-provider mode in the scenario/bundle authentication metadata and fail closed if its provider is unavailable. |

The cloud deployer must preserve the selected mode, provider, audience,
offline behavior, and dependency provenance in generated deployment metadata.
Secrets, passwords, private keys, refresh tokens, and website sessions never
belong in this manifest. Bootstrap and account linking use explicit,
short-lived server/browser flows; matching email addresses are not a linking
protocol.

## Bundle Section

Bundle composition and runtime safety defaults.

```json
{
  "bundle": {
    "include_packages": true,
    "include_autoheal": true,
    "scenarios": ["agent-inbox", "vrooli-autoheal"],
    "resources": ["postgres"]
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `include_packages` | boolean | `true` | Include npm/go dependencies |
| `include_autoheal` | boolean | `true` | Include `vrooli-autoheal` in bundle |
| `scenarios` | string[] | from dependencies | Explicit scenarios bundled |
| `resources` | string[] | from dependencies | Explicit resources bundled |

## Complete Example

```json
{
  "version": "1.0.0",
  "scenario": {
    "id": "agent-inbox"
  },
  "target": {
    "type": "vps",
    "vps": {
      "host": "vps.example.com",
      "user": "deploy",
      "port": 22
    }
  },
  "edge": {
    "domain": "inbox.example.com",
    "caddy": {
      "enabled": true,
      "email": "ops@example.com"
    }
  },
  "ports": {
    "ui": 3000,
    "api": 8080
  },
  "dependencies": {
    "resources": ["postgres", "ollama"],
    "scenarios": []
  },
  "bundle": {
    "include_packages": true,
    "include_autoheal": true
  }
}
```

## Closure binding

When the API refreshes a manifest (`ForceBundleBuild`), `dependencies.scenarios`,
`dependencies.resources`, `bundle.scenarios` and `bundle.resources` are the
projection of the deployment closure and `dependencies.closure_digest` records
the closure digest the snapshot came from. See
[Deployment Closure](../reference/closure.md).
