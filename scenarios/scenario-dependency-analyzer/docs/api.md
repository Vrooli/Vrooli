# Scenario Dependency Analyzer - API Reference

## Overview

The Scenario Dependency Analyzer API provides programmatic access to dependency analysis, graph generation, deployment readiness assessment, and metadata gap detection for Vrooli scenarios.

**Base URL**: `http://localhost:{API_PORT}/api/v1`
**Default Port**: Dynamically assigned (use `vrooli scenario port scenario-dependency-analyzer API_PORT`)
**Content-Type**: `application/json`

---

## Core Endpoints

### Health & Status

#### `GET /health`
Basic health check for the service.

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-01-22T10:00:00Z",
  "service": "scenario-dependency-analyzer",
  "readiness": true,
  "dependencies": {
    "database": {
      "connected": true,
      "latency_ms": 2.5
    }
  }
}
```

#### `GET /api/v1/health/analysis`
Comprehensive health check including analysis capabilities.

**Response**:
```json
{
  "status": "healthy",
  "capabilities": ["dependency_analysis", "graph_generation"],
  "scenarios_found": 45,
  "resources_available": 12,
  "database_status": "connected"
}
```

---

## Scenario Analysis

### `GET /api/v1/scenarios`
List all scenarios with metadata and last-scan status.

**Response**:
```json
[
  {
    "name": "ecosystem-manager",
    "display_name": "Ecosystem Manager",
    "version": "1.0.0",
    "last_scanned": "2025-01-22T09:30:00Z",
    "resource_count": 5,
    "scenario_count": 3
  }
]
```

### `GET /api/v1/scenarios/:scenario`
Get detailed information for a specific scenario including declared vs detected dependencies.

**Parameters**:
- `scenario` (path): Scenario name

**Response**:
```json
{
  "name": "ecosystem-manager",
  "service_config": {...},
  "dependencies": {
    "resources": [...],
    "scenarios": [...]
  },
  "drift": {
    "resources": {
      "missing": ["ollama"],
      "extra": []
    },
    "scenarios": {
      "missing": [],
      "extra": ["old-dependency"]
    }
  },
  "last_analyzed": "2025-01-22T09:30:00Z"
}
```

### `GET /api/v1/analyze/:scenario`
Perform full dependency analysis for a scenario.

**Parameters**:
- `scenario` (path): Scenario name or `all` for system-wide analysis
- `include_transitive` (query): Include transitive dependencies (default: `false`)

**Response**:
```json
{
  "scenario": "ecosystem-manager",
  "resources": [
    {
      "id": "uuid",
      "dependency_name": "postgres",
      "dependency_type": "resource",
      "required": true,
      "purpose": "Store ecosystem metadata",
      "access_method": "declared"
    }
  ],
  "detected_resources": [...],
  "scenarios": [...],
  "resource_diff": {
    "missing": [...],
    "extra": [...]
  },
  "scenario_diff": {
    "missing": [...],
    "extra": [...]
  },
  "deployment_report": {...}
}
```

---

## Dependency Scanning

### `POST /api/v1/scenarios/:scenario/scan`
Scan a scenario and optionally apply detected dependencies to `service.json`.

**Parameters**:
- `scenario` (path): Scenario name

**Request Body**:
```json
{
  "apply": false,
  "apply_resources": false,
  "apply_scenarios": false
}
```

**Response**:
```json
{
  "analysis": {...},
  "applied": false,
  "apply_summary": {
    "resources_added": [],
    "scenarios_added": [],
    "changes_made": 0
  }
}
```

---

## Deployment Intelligence

### `GET /api/v1/scenarios/:scenario/deployment`
Get full deployment readiness report including recursive DAG, tier fitness, and metadata gaps.

**Parameters**:
- `scenario` (path): Scenario name

**Response**:
```json
{
  "scenario": "ecosystem-manager",
  "report_version": 1,
  "generated_at": "2025-01-22T10:00:00Z",
  "dependencies": [
    {
      "name": "postgres",
      "type": "resource",
      "resource_type": "database",
      "tier_support": {
        "desktop": {
          "supported": true,
          "fitness_score": 0.95
        },
        "server": {
          "supported": true,
          "fitness_score": 1.0
        }
      },
      "children": []
    }
  ],
  "aggregates": {
    "desktop": {
      "fitness_score": 0.85,
      "dependency_count": 8,
      "blocking_dependencies": [],
      "estimated_requirements": {
        "ram_mb": 2048,
        "disk_mb": 500,
        "cpu_cores": 2
      }
    }
  },
  "bundle_manifest": {
    "scenario": "ecosystem-manager",
    "files": [...],
    "dependencies": [...]
  },
  "metadata_gaps": {
    "total_gaps": 5,
    "scenarios_missing_all": 2,
    "gaps_by_scenario": {
      "api-tools": {
        "scenario_name": "api-tools",
        "has_deployment_block": false,
        "suggested_actions": [
          "Add deployment block to .vrooli/service.json"
        ]
      }
    },
    "missing_tiers": ["mobile", "saas"],
    "recommendations": [
      "2 scenario(s) missing deployment blocks entirely - run scan --apply to initialize",
      "Add tier definitions for: [mobile, saas]"
    ]
  }
}
```

### `GET /api/v1/scenarios/:scenario/dag/export`
Export the recursive dependency DAG for a scenario.

**Parameters**:
- `scenario` (path): Scenario name
- `recursive` (query): Include full recursive tree (default: `true`)
- `format` (query): Output format (only `json` currently supported)

**Response**:
```json
{
  "scenario": "ecosystem-manager",
  "recursive": true,
  "generated_at": "2025-01-22T10:00:00Z",
  "dag": [
    {
      "name": "postgres",
      "type": "resource",
      "children": []
    },
    {
      "name": "api-tools",
      "type": "scenario",
      "children": [
        {
          "name": "redis",
          "type": "resource",
          "children": []
        }
      ]
    }
  ],
  "metadata_gaps": {...}
}
```

---

## Graph Visualization

### `GET /api/v1/graph/:type`
Generate dependency graph for visualization.

**Parameters**:
- `type` (path): Graph type - `resource`, `scenario`, or `combined`

**Response**:
```json
{
  "nodes": [
    {
      "id": "postgres",
      "label": "postgres",
      "type": "resource"
    },
    {
      "id": "ecosystem-manager",
      "label": "Ecosystem Manager",
      "type": "scenario"
    }
  ],
  "edges": [
    {
      "source": "ecosystem-manager",
      "target": "postgres",
      "label": "requires"
    }
  ],
  "metadata": {
    "total_nodes": 45,
    "total_edges": 120,
    "complexity_score": 2.67
  }
}
```

### `GET /api/v1/graph/:type/cycles`
Detect circular dependencies in the dependency graph.

**Parameters**:
- `type` (path): Graph type - `resource`, `scenario`, or `combined`

**Response**:
```json
{
  "has_cycles": true,
  "severity": "high",
  "message": "3 circular dependencies detected",
  "cycles": [
    {
      "description": "scenario-a → scenario-b → scenario-a",
      "cycle_type": "scenario",
      "length": 2,
      "required": true
    }
  ],
  "affected_dependencies": ["scenario-a", "scenario-b"],
  "recommendations": [
    "Break cycle by introducing abstraction layer",
    "Consider merging tightly coupled scenarios"
  ]
}
```

---

## Impact Analysis

### `GET /api/v1/dependencies/:name/impact`
Analyze the impact of removing a dependency.

**Parameters**:
- `name` (path): Dependency name (resource or scenario)

**Response**:
```json
{
  "dependency": "postgres",
  "severity": "critical",
  "impact_summary": "Removing postgres would break 23 scenarios",
  "total_affected": 23,
  "direct_dependents": [
    {
      "scenario_name": "ecosystem-manager",
      "required": true,
      "purpose": "Store ecosystem metadata"
    }
  ],
  "indirect_dependents": [...],
  "recommendations": [
    "Consider alternative: sqlite for lightweight deployments",
    "Requires major refactoring - not recommended"
  ]
}
```

---

## Optimization

### `POST /api/v1/optimize`
Get optimization recommendations for scenarios.

**Request Body**:
```json
{
  "scenario": "ecosystem-manager",
  "type": "resource",
  "apply": false
}
```

**Response**:
```json
{
  "results": {
    "ecosystem-manager": {
      "recommendations": [
        {
          "id": "opt-001",
          "recommendation_type": "resource_swap",
          "title": "Replace Ollama with OpenRouter for lightweight deployment",
          "description": "Ollama requires 4GB RAM; OpenRouter is API-based",
          "confidence_score": 0.85,
          "priority": "medium"
        }
      ],
      "summary": {
        "recommendation_count": 5,
        "high_priority": 2
      }
    }
  },
  "generated_at": "2025-01-22T10:00:00Z"
}
```

---

## Proposed Scenario Analysis

### `POST /api/v1/analyze/proposed`
Analyze dependencies for a proposed scenario description.

**Request Body**:
```json
{
  "name": "ai-chatbot",
  "description": "AI-powered chat with database storage",
  "requirements": ["nlp", "database", "api"],
  "similar_scenarios": ["ecosystem-manager"]
}
```

**Response**:
```json
{
  "recommended_resources": ["postgres", "ollama", "redis"],
  "recommended_scenarios": ["data-tools", "api-manager"],
  "similar_patterns": [
    {
      "scenario": "ecosystem-manager",
      "similarity_score": 0.78
    }
  ],
  "confidence_scores": {
    "resource": 0.85,
    "scenario": 0.72
  }
}
```

---

## Error Responses

All endpoints follow standard HTTP status codes:

- `200 OK`: Success
- `400 Bad Request`: Invalid parameters
- `404 Not Found`: Scenario not found
- `500 Internal Server Error`: Server error

**Error Response Format**:
```json
{
  "error": "Scenario 'invalid-name' not found"
}
```

---

## Rate Limiting

Currently no rate limiting is enforced. For production deployments, consider adding rate limiting middleware.

---

## Authentication

Currently no authentication is required. The service is designed for local/trusted network use. For production:
- Consider adding API key authentication
- Use HTTPS
- Implement IP whitelisting

---

## Pagination

Endpoints that return lists (e.g., `/scenarios`) currently return all results. Future versions will support pagination via `?page=N&limit=M` query parameters.

---

## Versioning

The API uses URL-based versioning (`/api/v1/`). Breaking changes will increment the version number.

---

## Examples

### Analyze and Apply Dependencies

```bash
# 1. Scan for dependencies
curl -X POST http://localhost:20400/api/v1/scenarios/my-scenario/scan \
  -H "Content-Type: application/json" \
  -d '{"apply": false}'

# 2. Review the detected drift
curl http://localhost:20400/api/v1/scenarios/my-scenario | jq '.drift'

# 3. Apply the changes
curl -X POST http://localhost:20400/api/v1/scenarios/my-scenario/scan \
  -H "Content-Type: application/json" \
  -d '{"apply": true, "apply_resources": true}'
```

### Export Deployment Bundle

```bash
# Get full DAG with metadata gaps
curl "http://localhost:20400/api/v1/scenarios/my-scenario/dag/export?recursive=true" \
  | jq . > deployment-bundle.json

# Check for gaps
jq '.metadata_gaps.recommendations' deployment-bundle.json

# Validate against the desktop bundle schema
ajv validate -s ../../docs/deployment/bundle-schema.desktop.v0.1.json -d deployment-bundle.json
```

### Integration with deployment-manager

```bash
# deployment-manager can consume the DAG export
SCENARIO="my-app"
DAG=$(curl -s "http://localhost:20400/api/v1/scenarios/${SCENARIO}/dag/export?recursive=true")

# Pass to deployment manager
echo "$DAG" | deployment-manager bundle create --from-dag -
```

---

## Webhook Support (Future)

Future versions will support webhooks for:
- `scenario.analyzed` - When analysis completes
- `dependency.drift.detected` - When drift is found
- `metadata.gap.found` - When gaps are identified
