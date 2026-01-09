# Architecture

System design and components of Scenario-to-Cloud.

## Overview

Scenario-to-Cloud is a deployment orchestrator that transfers Vrooli scenarios from a local development environment to a production VPS.

```
┌─────────────────────────────────────────────────────────────┐
│                    Local Machine                             │
│  ┌──────────────────┐     ┌──────────────────┐              │
│  │   Scenario-to-   │────▶│   Bundle         │              │
│  │   Cloud UI       │     │   Creator        │              │
│  └────────┬─────────┘     └────────┬─────────┘              │
│           │                        │                         │
│           ▼                        ▼                         │
│  ┌──────────────────┐     ┌──────────────────┐              │
│  │   Go API         │────▶│   Local Vrooli   │              │
│  │   Server         │     │   Scenarios      │              │
│  └────────┬─────────┘     └──────────────────┘              │
└───────────┼─────────────────────────────────────────────────┘
            │ SSH/SCP
            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Remote VPS                                │
│  ┌──────────────────┐     ┌──────────────────┐              │
│  │   Mini-Vrooli    │────▶│   Running        │              │
│  │   Installation   │     │   Scenario       │              │
│  └──────────────────┘     └────────┬─────────┘              │
│                                    │                         │
│  ┌──────────────────┐              ▼                         │
│  │   Caddy          │◀────────────────────────               │
│  │   (HTTPS)        │     Public Traffic                     │
│  └──────────────────┘                                        │
└─────────────────────────────────────────────────────────────┘
```

## Components

### UI (React + TypeScript)

Single-page application providing:
- Deployment wizard with step-by-step guidance
- Deployment management (list, inspect, stop, delete)
- Real-time status updates via polling
- Documentation browser

### API (Go)

RESTful API server handling:
- Manifest validation and normalization
- Bundle creation and management
- VPS operations via SSH
- Deployment persistence (PostgreSQL)
- Documentation serving

### Bundle System

Creates minimal, self-contained packages:
- Core Vrooli scripts
- Scenario files
- Resource configurations
- No unnecessary files

### SSH/SCP Integration

Secure file transfer and remote execution:
- Key-based authentication
- Idempotent operations
- Error recovery

## Data Flow

### Deployment Flow

1. **Manifest Creation**: User configures deployment via UI
2. **Validation**: API validates manifest against schema
3. **Planning**: API generates execution plan
4. **Bundle**: Creates tarball of minimal Vrooli + scenario
5. **Transfer**: SCP sends bundle to VPS
6. **Setup**: SSH runs setup scripts on VPS
7. **Deploy**: SSH starts scenario services
8. **Verify**: Health checks confirm deployment

### Inspection Flow

1. **Request**: UI triggers inspect via API
2. **SSH**: API connects to VPS
3. **Check**: Runs `vrooli scenario status`
4. **Logs**: Retrieves recent log output
5. **Response**: Returns status to UI

## Database Schema

```sql
CREATE TABLE deployments (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    scenario_id TEXT NOT NULL,
    status TEXT NOT NULL,
    manifest JSONB NOT NULL,
    bundle_path TEXT,
    bundle_sha256 TEXT,
    setup_result JSONB,
    deploy_result JSONB,
    last_inspect_result JSONB,
    error_message TEXT,
    error_step TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    last_deployed_at TIMESTAMPTZ,
    last_inspected_at TIMESTAMPTZ
);
```

## Security Considerations

### SSH Keys

- Uses local SSH key for VPS access
- Supports key path configuration
- No password authentication

### Input Validation

- All manifests validated before use
- Path traversal protection in docs API
- JSON size limits on requests

### Network

- HTTPS enforced via Caddy
- Automatic certificate management
- No sensitive data in logs
