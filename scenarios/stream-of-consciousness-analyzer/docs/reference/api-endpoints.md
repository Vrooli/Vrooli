# API Reference

All endpoints are prefixed with `/api/v1`. The API returns JSON responses.

## Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Infrastructure health check (database critical) |
| GET | `/api/v1/health` | Client-facing health check |

## Schemes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/schemes` | List all schemes |
| POST | `/api/v1/schemes` | Create a scheme |
| GET | `/api/v1/schemes/{id}` | Get scheme by ID |
| PUT | `/api/v1/schemes/{id}` | Update scheme |
| DELETE | `/api/v1/schemes/{id}` | Delete scheme (cascades to information) |

**Create/Update body:** `{ "name": "string" }`

## Information (Canvas Items)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/schemes/{schemeId}/information` | List information in scheme |
| POST | `/api/v1/schemes/{schemeId}/information` | Create information item |
| PUT | `/api/v1/schemes/{schemeId}/information/{infoId}` | Update information item |
| DELETE | `/api/v1/schemes/{schemeId}/information/{infoId}` | Delete information item |

**Create body:** `{ "type": "text", "content": "...", "canvas_x": 0, "canvas_y": 0 }`

**Update body:** (partial) `{ "content": "...", "canvas_x": 100 }`

## Thoughts

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/thoughts` | List all thoughts (optionally filter by `?scheme_id=`) |
| POST | `/api/v1/thoughts` | Create a thought |
| GET | `/api/v1/thoughts/{id}` | Get thought by ID |
| PUT | `/api/v1/thoughts/{id}` | Update thought |
| DELETE | `/api/v1/thoughts/{id}` | Delete thought (cascades edges) |

**Create body:** `{ "scheme_id": "uuid|null", "title": "...", "body": "...", "canvas_x": 0, "canvas_y": 0 }`

## Thought Edges

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/thoughts/{id}/edges` | List edges from a thought |
| POST | `/api/v1/thoughts/{id}/edges` | Create edge from thought to target |
| DELETE | `/api/v1/thoughts/{id}/edges/{edgeId}` | Delete an edge |

**Create body:** `{ "target_id": "uuid", "label": "causes" }`

## Export

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/schemes/{id}/export` | Export scheme as structured graph (thoughts + edges + information) |

## Suggestions

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/providers` | List available LLM providers and their status |
| POST | `/api/v1/schemes/{id}/suggestions` | Generate ghost node suggestions for a scheme |

## Error Responses

All errors return: `{ "error": "description" }`

Common status codes:
- `400` — Invalid request body
- `404` — Resource not found
- `500` — Internal server error
