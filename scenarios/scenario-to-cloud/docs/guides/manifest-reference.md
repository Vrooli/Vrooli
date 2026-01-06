# Manifest Reference

Complete reference for the deployment manifest configuration.

## Schema Overview

```json
{
  "scenario": { ... },
  "target": { ... },
  "edge": { ... },
  "ports": { ... },
  "dependencies": { ... },
  "options": { ... }
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

## Options Section

Deployment behavior options.

```json
{
  "options": {
    "include_packages": true,
    "autoheal": true,
    "force_rebuild": false
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `include_packages` | boolean | `true` | Include npm/go dependencies |
| `autoheal` | boolean | `true` | Enable automatic restart on failure |
| `force_rebuild` | boolean | `false` | Rebuild even if bundle exists |

## Complete Example

```json
{
  "scenario": {
    "id": "agent-inbox"
  },
  "target": {
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
  "options": {
    "include_packages": true,
    "autoheal": true
  }
}
```
