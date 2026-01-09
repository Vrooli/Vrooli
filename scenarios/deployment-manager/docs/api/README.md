# API Reference

The deployment-manager exposes a REST API for all deployment orchestration features.

## Base URL

```
http://localhost:{API_PORT}/api/v1
```

Get the port from the lifecycle system:

```bash
API_PORT=$(vrooli scenario port deployment-manager API_PORT)
```

## Authentication

Most endpoints require authentication via Bearer token:

```bash
curl -H "Authorization: Bearer ${API_TOKEN}" \
  http://localhost:${API_PORT}/api/v1/profiles
```

The token is set via the `API_TOKEN` environment variable when the scenario starts.

## Endpoints Overview

| Category | Endpoint | Method | Description |
|----------|----------|--------|-------------|
| **Health** | `/health` | GET | API health check |
| **Dependencies** | `/dependencies/analyze/{scenario}` | GET | Analyze scenario dependencies |
| **Fitness** | `/fitness/score` | POST | Calculate fitness scores |
| **Profiles** | `/profiles` | GET, POST | List/create profiles |
| **Profiles** | `/profiles/{id}` | GET, PUT, DELETE | Profile CRUD |
| **Profiles** | `/profiles/{id}/versions` | GET | Version history |
| **Profiles** | `/profiles/{id}/validate` | GET | Validate profile |
| **Profiles** | `/profiles/{id}/cost-estimate` | GET | Cost estimation |
| **Swaps** | `/swaps/analyze/{from}/{to}` | GET | Analyze swap impact |
| **Swaps** | `/swaps/cascade/{from}/{to}` | GET | Cascade analysis |
| **Secrets** | `/profiles/{id}/secrets` | GET | List required secrets |
| **Secrets** | `/profiles/{id}/secrets/template` | GET | Generate template |
| **Secrets** | `/profiles/{id}/secrets/validate` | POST | Validate secrets |
| **Bundles** | `/bundles/validate` | POST | Validate manifest |
| **Bundles** | `/bundles/merge-secrets` | POST | Merge secrets into manifest |
| **Bundles** | `/bundles/assemble` | POST | Assemble full manifest |
| **Bundles** | `/bundles/export` | POST | Export with checksum |
| **Deployments** | `/deploy/{profile_id}` | POST | Start deployment |
| **Deployments** | `/deployments/{id}` | GET | Deployment status |
| **Telemetry** | `/telemetry` | GET | List telemetry summaries |
| **Telemetry** | `/telemetry/upload` | POST | Upload telemetry |

## Endpoint Documentation

| Category | Documentation |
|----------|---------------|
| Bundles | [bundles.md](bundles.md) |
| Profiles | [profiles.md](profiles.md) |
| Fitness | [fitness.md](fitness.md) |
| Swaps | [swaps.md](swaps.md) |
| Deployments | [deployments.md](deployments.md) |
| Telemetry | [telemetry.md](telemetry.md) |

## Common Response Formats

### Success

```json
{
  "status": "success",
  "data": { ... }
}
```

### Error

```json
{
  "error": "error_code",
  "details": "Human-readable description"
}
```

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| `200` | Success |
| `201` | Created |
| `400` | Bad request (invalid input) |
| `401` | Unauthorized (missing/invalid token) |
| `404` | Resource not found |
| `502` | Bad gateway (upstream service unavailable) |
| `500` | Internal server error |

## Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes* | `Bearer {token}` |
| `Content-Type` | For POST/PUT | `application/json` |

*Health endpoints allow anonymous access.

## Example: Complete Desktop Deployment Flow

```bash
API_PORT=$(vrooli scenario port deployment-manager API_PORT)
BASE="http://localhost:${API_PORT}/api/v1"

# 1. Analyze dependencies
curl "$BASE/dependencies/analyze/picker-wheel"

# 2. Check fitness
curl -X POST "$BASE/fitness/score" \
  -H "Content-Type: application/json" \
  -d '{"scenario": "picker-wheel", "tiers": [2]}'

# 3. Create profile
PROFILE=$(curl -X POST "$BASE/profiles" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "picker-wheel-desktop",
    "scenario": "picker-wheel",
    "tiers": [2]
  }' | jq -r '.id')

# 4. Assemble bundle manifest
curl -X POST "$BASE/bundles/assemble" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario": "picker-wheel",
    "tier": "tier-2-desktop",
    "include_secrets": true
  }'

# 5. Export bundle with checksum
curl -X POST "$BASE/bundles/export" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario": "picker-wheel",
    "tier": "tier-2-desktop"
  }' > bundle.json
```

## Source Code

Routes are defined in:
- `scenarios/deployment-manager/api/server/routes.go`

Handlers are organized by domain:
- `api/bundles/handler.go`
- `api/profiles/handler.go`
- `api/fitness/handler.go`
- `api/swaps/handler.go`
- `api/deployments/handler.go`
- `api/secrets/handler.go`
- `api/telemetry/handler.go`
- `api/dependencies/handler.go`

## Related

- [Deployment Guide](../DEPLOYMENT-GUIDE.md) - Complete deployment walkthrough
- [CLI Reference](../cli/README.md) - Command-line interface documentation
- [Desktop Workflow](../workflows/desktop-deployment.md) - Full desktop deployment guide
- [Roadmap](../ROADMAP.md) - Implementation status and planned work
