# Graph Query API Documentation

## Overview

The Tech Tree Designer provides powerful graph query endpoints designed for agent/LLM consumption. These endpoints enable intelligent exploration of technology dependencies, hierarchical relationships, and strategic pathways.

## Agent-Friendly Query Endpoints

### 1. Neighborhood Query

**Endpoint**: `GET /api/v1/graph/neighborhood`

Returns all stages within N hops of a given stage using dependency relationships (prerequisite → dependent).

**Query Parameters**:
- `stage_id` (required): The origin stage UUID
- `depth` (optional): Number of hops to traverse (default: 2, max: 5)
- `direction` (optional): Traversal direction
  - `"prerequisites"` - Only follow prerequisite edges (what this needs)
  - `"dependents"` - Only follow dependent edges (what depends on this)
  - `"both"` - Bidirectional traversal (default)
- `include_hierarchy` (optional): Include hierarchical children (default: false)
- `include_scenarios` (optional): Include scenario mappings (default: false)
- `max_results` (optional): Limit results (default: 100)
- `tree_id` (optional): Specify tree (defaults to active official tree)

**Example Request**:
```bash
curl "http://localhost:8020/api/v1/graph/neighborhood?stage_id=abc-123&depth=2&direction=both&include_scenarios=true"
```

**Response**:
```json
{
  "tree": { ... },
  "neighborhood": {
    "origin": {
      "stage": { "id": "abc-123", "name": "Foundation ERP", ... },
      "sector": { "id": "...", "name": "Manufacturing", ... },
      "scenario_mappings": [
        { "scenario_name": "production-planner", "completion_status": "completed" }
      ]
    },
    "stages": [
      {
        "stage": { "id": "def-456", "name": "MES Integration", ... },
        "sector": { ... },
        "distance": 1
      },
      ...
    ],
    "edge_count": 15,
    "max_depth_reached": 2
  }
}
```

**Use Cases**:
- "Show me all technologies within 2 hops of scenario X"
- "What are the immediate prerequisites for this stage?"
- "What other capabilities depend on this foundation?"

---

### 2. Shortest Path Query

**Endpoint**: `GET /api/v1/graph/path`

Finds the shortest dependency path between two stages using BFS.

**Query Parameters**:
- `from` (required): Starting stage UUID
- `to` (required): Target stage UUID
- `max_depth` (optional): Maximum search depth (default: 10, max: 20)
- `tree_id` (optional): Specify tree

**Example Request**:
```bash
curl "http://localhost:8020/api/v1/graph/path?from=abc-123&to=xyz-789"
```

**Response**:
```json
{
  "tree": { ... },
  "path": {
    "found": true,
    "length": 3,
    "path": [
      { "stage": { "name": "Foundation ERP" }, "sector": { ... }, "distance": 0 },
      { "stage": { "name": "MES Integration" }, "sector": { ... }, "distance": 1 },
      { "stage": { "name": "Real-time Analytics" }, "sector": { ... }, "distance": 2 },
      { "stage": { "name": "Digital Twin" }, "sector": { ... }, "distance": 3 }
    ]
  }
}
```

**Use Cases**:
- "What's the dependency chain from basic productivity to AGI?"
- "Show the shortest path between two sectors"
- "What intermediate steps are needed to reach this capability?"

---

### 3. Ancestor Chain Query

**Endpoint**: `GET /api/v1/graph/ancestors`

Returns the hierarchical parent chain (following `parent_stage_id` relationships, not dependencies).

**Query Parameters**:
- `stage_id` (required): The child stage UUID
- `depth` (optional): How many levels to traverse (default: 10, max: 50)
- `include_children` (optional): Include siblings at each level (default: false)
- `tree_id` (optional): Specify tree

**Example Request**:
```bash
curl "http://localhost:8020/api/v1/graph/ancestors?stage_id=abc-123&depth=5"
```

**Response**:
```json
{
  "tree": { ... },
  "ancestors": {
    "origin": { "stage": { "name": "ML Model Training" }, ... },
    "ancestors": [
      { "stage": { "name": "Data Analytics Platform" }, "distance": 1 },
      { "stage": { "name": "Core Data Management" }, "distance": 2 },
      { "stage": { "name": "Software Foundation" }, "distance": 3 }
    ],
    "depth_reached": 3
  }
}
```

**Use Cases**:
- "What is the hierarchical context of this stage?"
- "Show me the parent → grandparent → great-grandparent chain"
- "What broader capability does this belong to?"

---

### 4. Graph View Export

**Endpoint**: `GET /api/v1/graph/export/view`

Exports graph data in text or JSON format optimized for LLM/agent consumption.

**Query Parameters**:
- `format` (optional): `"text"` (human-readable) or `"json"` (structured, default: text)
- `stage_id` (optional): If provided, exports neighborhood; otherwise exports full tree
- `depth` (optional): If using stage_id, depth of neighborhood (default: 2)
- `tree_id` (optional): Specify tree

**Example Requests**:

**Full Tree as Text**:
```bash
curl "http://localhost:8020/api/v1/graph/export/view?format=text"
```

**Neighborhood as JSON**:
```bash
curl "http://localhost:8020/api/v1/graph/export/view?format=json&stage_id=abc-123&depth=2"
```

**Text Response Example**:
```
# Tech Tree: Vrooli Strategic Roadmap
Tree Type: official
Version: 1.0.0

## Sector: Manufacturing
Category: manufacturing
Progress: 35.2%
Advanced manufacturing systems and digital twins

### Stages:
- **Foundation ERP** (foundation) - 85.0%
  Scenarios: production-planner (completed), inventory-optimizer (in_progress)
- **MES Integration** (operational) - 45.0%
  Scenarios: shop-floor-monitor (in_progress)
...

## Dependencies

- Foundation ERP → MES Integration (required, strength: 1.00)
- MES Integration → Real-time Analytics (required, strength: 0.85)
...
```

**JSON Response** (when `format=json`):
```json
{
  "tree": { ... },
  "sectors": [ ... ],
  "dependencies": [ ... ],
  "query_type": "full_tree"
}
```

**Use Cases**:
- "Export this view so I can paste it into ChatGPT"
- "Give me a text summary of the local neighborhood"
- "I need structured JSON for programmatic analysis"

---

## Performance Considerations

### Depth Limits
- **Neighborhood**: Max depth 5 to prevent exponential explosion
- **Shortest Path**: Max depth 20 to prevent infinite loops
- **Ancestors**: Max depth 50 (hierarchies can be deep)
- **Max Results**: 100 by default, configurable up to 200

### Optimization Tips
1. **Start with small depths**: Use depth=1 or depth=2 first
2. **Use direction filters**: `"prerequisites"` or `"dependents"` reduces search space by ~50%
3. **Limit scenarios**: Set `include_scenarios=false` if not needed
4. **Cache results**: Query results are deterministic for a given tree state

### Query Cost Estimates
| Query Type | Typical Cost | Notes |
|------------|--------------|-------|
| Neighborhood (depth=1) | ~10ms | Direct neighbors only |
| Neighborhood (depth=2) | ~50ms | Quadratic growth |
| Neighborhood (depth=3) | ~200ms | Can retrieve 50+ stages |
| Shortest Path | ~30-100ms | Depends on graph density |
| Ancestors | ~5-20ms | Linear with depth |
| Full Export | ~100-300ms | Loads entire tree |

---

## Common Agent Patterns

### Pattern 1: Local Context Discovery
```bash
# "What technologies are immediately related to scenario X?"
GET /graph/neighborhood?stage_id=<id>&depth=1&include_scenarios=true
```

### Pattern 2: Strategic Path Planning
```bash
# "What's the path from our current state to goal Y?"
GET /graph/path?from=<current_stage>&to=<goal_stage>
```

### Pattern 3: Bottleneck Analysis
```bash
# "Show me what depends on this critical stage"
GET /graph/neighborhood?stage_id=<id>&depth=2&direction=dependents
```

### Pattern 4: Capability Context
```bash
# "What broader technology does this belong to?"
GET /graph/ancestors?stage_id=<id>&depth=5
```

### Pattern 5: LLM Discussion Export
```bash
# "Export this area for discussion"
GET /graph/export/view?format=text&stage_id=<id>&depth=2
```

---

## Integration Examples

### Python Agent Example
```python
import requests

def get_local_context(stage_id: str, depth: int = 2):
    """Get neighborhood context for strategic planning."""
    response = requests.get(
        "http://localhost:8020/api/v1/graph/neighborhood",
        params={
            "stage_id": stage_id,
            "depth": depth,
            "include_scenarios": "true",
            "direction": "both"
        }
    )
    return response.json()

def find_strategic_path(from_stage: str, to_stage: str):
    """Find dependency path between capabilities."""
    response = requests.get(
        "http://localhost:8020/api/v1/graph/path",
        params={"from": from_stage, "to": to_stage}
    )
    data = response.json()

    if data["path"]["found"]:
        path = data["path"]["path"]
        print(f"Path length: {data['path']['length']}")
        for step in path:
            print(f"  {step['distance']}: {step['stage']['name']}")
    else:
        print("No path found")

def export_for_llm(stage_id: str = None):
    """Export graph view as text for LLM consumption."""
    params = {"format": "text"}
    if stage_id:
        params["stage_id"] = stage_id
        params["depth"] = 2

    response = requests.get(
        "http://localhost:8020/api/v1/graph/export/view",
        params=params
    )
    return response.text
```

### JavaScript/TypeScript Example
```typescript
async function analyzeStageContext(stageId: string) {
  // Get immediate neighborhood
  const neighborhood = await fetch(
    `/api/v1/graph/neighborhood?stage_id=${stageId}&depth=1&include_scenarios=true`
  ).then(r => r.json())

  // Get hierarchical context
  const ancestors = await fetch(
    `/api/v1/graph/ancestors?stage_id=${stageId}&depth=5`
  ).then(r => r.json())

  return {
    dependencies: neighborhood.neighborhood.stages,
    lineage: ancestors.ancestors.ancestors
  }
}
```

---

## Error Handling

### Common Errors

**404 Not Found**:
```json
{ "error": "stage not found: abc-123" }
```
- Stage UUID doesn't exist in the specified tree

**400 Bad Request**:
```json
{ "error": "stage_id query parameter required" }
```
- Missing required parameters

**500 Internal Server Error**:
```json
{ "error": "failed to load origin stage: ..." }
```
- Database connection issues or data corruption

### Best Practices
1. Always validate stage UUIDs before querying
2. Handle `found: false` in path queries gracefully
3. Set reasonable timeouts (queries should complete in <1s)
4. Cache results where appropriate
5. Use exponential backoff for retries

---

## Future Enhancements

Potential future additions:
- **Graph clustering**: Identify technology clusters automatically
- **Centrality metrics**: Find most critical nodes
- **Community detection**: Discover natural groupings
- **Timeline projection**: Estimate completion paths
- **Impact analysis**: Predict cascading effects of changes
- **Viewport export**: Export only visible graph region from UI
