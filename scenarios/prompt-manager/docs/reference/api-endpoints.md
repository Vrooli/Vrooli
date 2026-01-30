# API Reference

Complete documentation for the prompt-manager REST API.

**Base URL:** `http://localhost:{PORT}/api/v1`

All endpoints return JSON. Error responses follow the format:
```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```

---

## Health

### GET /health

Check API server health.

**Response:**
```json
{
  "status": "healthy",
  "version": "2.0.0",
  "checks": {
    "database": "healthy"
  }
}
```

---

## Skills

[CODE: api/skills/handlers.go]

### GET /api/v1/skills

List all skills with optional filtering.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `folder` | string | Filter by folder: `core`, `local`, `drafts` |
| `tag` | string | Filter by tag |
| `mode` | string | Filter by mode |

**Response:**
```json
[
  {
    "id": "debugging",
    "name": "Debugging",
    "description": "Systematic debugging approach",
    "content": "## Steer focus: Debugging...",
    "modes": ["agent"],
    "tags": ["debugging"],
    "folder": "core",
    "draft": false,
    "usageCount": 42,
    "effectivenessRating": 4,
    "createdAt": "2024-01-15T10:00:00Z",
    "updatedAt": "2024-01-20T14:30:00Z"
  }
]
```

### GET /api/v1/skills/{id}

Get a single skill by ID.

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Skill ID (filename without extension) |

**Response:** Same as list item above.

**Errors:**
- `404` - Skill not found

### POST /api/v1/skills

Create a new skill.

**Request Body:**
```json
{
  "name": "My Skill",
  "description": "What this skill does",
  "content": "## Steer focus: My Skill\n\nContent here...",
  "folder": "local",
  "tags": ["debugging", "testing"],
  "modes": ["agent"],
  "draft": false
}
```

**Required Fields:** `name`, `content`, `folder`

**Response:** Created skill object with generated `id`.

### PUT /api/v1/skills/{id}

Update an existing skill.

**Request Body:** (all fields optional)
```json
{
  "name": "Updated Name",
  "description": "Updated description",
  "content": "Updated content...",
  "tags": ["new-tag"],
  "folder": "local",
  "draft": true
}
```

**Response:** Updated skill object.

**Notes:**
- Creates a new version in version history
- Can move skill to different folder by specifying `folder` (version history moves with the skill)

### DELETE /api/v1/skills/{id}

Delete a skill.

**Response:** `204 No Content`

**Notes:**
- Cannot delete skills in `core` folder
- Removes all version history

---

## Version History

[CODE: api/skills/handlers.go#GetVersions]

### GET /api/v1/skills/{id}/versions

Get version history for a skill.

**Response:**
```json
{
  "skillId": "my-skill",
  "current": 3,
  "versions": [
    {
      "version": 1,
      "name": "My Skill",
      "content": "Original content...",
      "updatedAt": "2024-01-15T10:00:00Z"
    },
    {
      "version": 2,
      "name": "My Skill v2",
      "content": "Updated content...",
      "updatedAt": "2024-01-18T12:00:00Z"
    },
    {
      "version": 3,
      "name": "My Skill v3",
      "content": "Current content...",
      "updatedAt": "2024-01-20T14:30:00Z"
    }
  ]
}
```

### POST /api/v1/skills/{id}/revert/{version}

Revert a skill to a specific version.

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Skill ID |
| `version` | int | Version number to revert to |

**Response:**
```json
{
  "skillId": "my-skill",
  "revertedTo": 2,
  "newVersion": 4,
  "restoredAt": "2024-01-21T09:00:00Z"
}
```

**Notes:**
- Reverts content and name from specified version
- Creates a new version (does not delete history)

---

## Usage Tracking

### POST /api/v1/skills/{id}/use

Record skill usage (call when user copies/uses a skill).

**Response:**
```json
{
  "skillId": "debugging",
  "usageCount": 43,
  "lastUsed": "2024-01-21T09:00:00Z"
}
```

### PUT /api/v1/skills/{id}/rating

Set effectiveness rating for a skill.

**Request Body:**
```json
{
  "rating": 4,
  "notes": "Very helpful for complex bugs"
}
```

**Rating:** Integer 1-5

**Response:** `204 No Content`

---

## Search

[CODE: api/search/handlers.go]

### GET /api/v1/search/skills

Full-text search across skills.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `q` | string | Search query (searches name, description, content) |
| `tag` | string | Filter by tag |
| `folder` | string | Filter by folder |

**Response:**
```json
{
  "results": [
    {
      "id": "debugging",
      "name": "Debugging",
      "description": "Systematic debugging approach",
      "folder": "core",
      "tags": ["debugging"],
      "modes": ["agent"],
      "score": 0.95,
      "highlight": "...systematic **debugging** approach for..."
    }
  ],
  "total": 1,
  "query": "debugging"
}
```

**Notes:**
- Results ranked by relevance score
- Highlight shows matching context

---

## AI Search

### POST /api/v1/search/ai

Semantic search powered by embeddings, with optional combined output formatting.

**Request Body:**
```json
{
  "query": "react coherence",
  "limit": 5,
  "output": "results",
  "format": "xml",
  "renderLimit": 3
}
```

**Output Options:** `results`, `combined`, `both` (default: `results`)
**Format Options:** `xml`, `markdown`, `json` (applies when output includes `combined`)

**Response:**
```json
{
  "results": [
    {
      "id": "react-coherence",
      "name": "React Coherence",
      "description": "Architectural patterns for React",
      "folder": "core",
      "tags": ["react"],
      "modes": ["frontend"],
      "score": 0.92,
      "scorePercent": 92
    }
  ],
  "combined": "<skills>...</skills>",
  "skillCount": 3,
  "totalTokens": 2500,
  "format": "xml",
  "total": 1,
  "query": "react coherence",
  "method": "ai",
  "output": "both"
}
```

### GET /api/v1/search/ai/status

Returns AI search availability status.

---

## Sync

### GET /api/v1/skills/sync

Get all skills with hash for change detection.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `tag` | string | Filter by tag |

**Response:**
```json
{
  "skills": [...],
  "lastUpdated": "2024-01-21T09:00:00Z",
  "hash": "abc123def456"
}
```

**Use Case:** Clients can cache skills and compare hash to detect changes.

---

## Read

### POST /api/v1/skills/read

Read multiple skills by identifier, with optional combined output formatting.

**Request Body:**
```json
{
  "identifiers": ["debugging", "testing", "refactor"],
  "resolve": "auto",
  "allowMissing": true,
  "output": "auto",
  "format": "xml"
}
```

**Output Options:** `skills`, `combined`, `both`, `auto` (default: `auto` → combined for multiple identifiers, skills for single)
**Format Options:** `xml`, `markdown`, `json` (applies when output includes `combined`)

**Response:**
```json
{
  "skills": [
    {
      "id": "debugging",
      "name": "Debugging",
      "description": "Systematic debugging approach",
      "content": "...",
      "modes": ["agent"],
      "tags": ["debugging"],
      "folder": "core"
    }
  ],
  "combined": "<skills>...</skills>",
  "skillCount": 3,
  "totalTokens": 2500,
  "format": "xml",
  "resolve": "auto",
  "output": "both",
  "missing": [],
  "ambiguous": []
}
```

**Use Case:** Read skills for piping or generate combined output for LLM context.

---

## Tags

[CODE: api/tags/handlers.go]

### GET /api/v1/tags

List all tags.

**Response:**
```json
[
  {
    "id": "abc123",
    "name": "debugging",
    "color": "#FF5733",
    "description": "Skills for debugging issues"
  }
]
```

### POST /api/v1/tags

Create a new tag.

**Request Body:**
```json
{
  "name": "my-tag",
  "color": "#3B82F6",
  "description": "Tag description"
}
```

**Required Fields:** `name`

**Response:** Created tag object.

---

## Testing (Ollama)

[CODE: api/testing/handlers.go]

### POST /api/v1/skills/{id}/test

Test a skill with Ollama LLM.

**Prerequisites:** Ollama running with model loaded.

**Request Body:**
```json
{
  "model": "llama3.2",
  "variables": {
    "TARGET": "src/auth/login.ts"
  },
  "maxTokens": 1000,
  "temperature": 0.7
}
```

**Response:**
```json
{
  "testId": "test-123",
  "model": "llama3.2",
  "response": "Based on the debugging skill...",
  "responseTime": 2500.5,
  "tokenCount": 450,
  "testedAt": "2024-01-21T09:00:00Z"
}
```

**Notes:**
- Variables replace `{{VAR}}` placeholders in skill content
- Test results saved to database for history

### GET /api/v1/skills/{id}/test-history

Get test history for a skill.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 10 | Max results to return |

**Response:**
```json
[
  {
    "id": "test-123",
    "skillId": "debugging",
    "model": "llama3.2",
    "inputVariables": "{\"TARGET\": \"src/auth\"}",
    "response": "...",
    "responseTime": 2500.5,
    "tokenCount": 450,
    "rating": 4,
    "notes": "Worked well",
    "testedAt": "2024-01-21T09:00:00Z"
  }
]
```

---

## Members

[CODE: api/members/handlers.go]

Members represent team entities visualized in the 3D world.

### GET /api/v1/members

List all members.

**Response:**
```json
[
  {
    "id": "member-1",
    "name": "Alice",
    "bodyColor": "#3B82F6",
    "headColor": "#F59E0B",
    "accentColor": "#10B981",
    "skills": ["debugging", "testing"],
    "createdAt": "2024-01-15T10:00:00Z",
    "updatedAt": "2024-01-20T14:30:00Z"
  }
]
```

### GET /api/v1/members/{id}

Get a single member.

### POST /api/v1/members

Create a new member.

**Request Body:**
```json
{
  "name": "Bob",
  "bodyColor": "#3B82F6",
  "headColor": "#F59E0B",
  "accentColor": "#10B981",
  "skills": []
}
```

**Required Fields:** `name`, `bodyColor`, `headColor`, `accentColor`

### PUT /api/v1/members/{id}

Update a member.

### DELETE /api/v1/members/{id}

Delete a member.

---

## Open Graph Metadata

[CODE: api/ogmeta/handlers.go]

### GET /api/v1/og-metadata

Fetch Open Graph metadata from a URL (for link previews).

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `url` | string | URL to fetch metadata from |

**Response:**
```json
{
  "url": "https://example.com/article",
  "title": "Article Title",
  "description": "Article description...",
  "image": "https://example.com/image.jpg",
  "siteName": "Example Site",
  "type": "article",
  "favicon": "https://example.com/favicon.ico"
}
```

**Notes:**
- Results cached for 15 minutes
- Timeout after 10 seconds
