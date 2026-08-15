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
      "port": 22,
      "key_path": "~/.ssh/id_rsa"
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
| `vps.key_path` | string | No | `~/.ssh/id_rsa` | Path to SSH private key |

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
