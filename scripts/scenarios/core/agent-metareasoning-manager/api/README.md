# Agent Metareasoning Manager API

## Overview

The Agent Metareasoning Manager API provides programmatic access to decision support tools, reasoning workflows, and prompt management capabilities. It integrates with n8n and Windmill to orchestrate complex reasoning patterns that enhance agent decision-making.

## Architecture

The API is built with TypeScript, Express, and follows a clean layered architecture:

```
src/
├── controllers/    # Request handlers
├── services/       # Business logic
├── repositories/   # Data access layer
├── schemas/        # Zod validation schemas
├── middleware/     # Express middleware
├── routes/         # Route definitions
└── config/         # Configuration & DI container
```

## Getting Started

### Prerequisites

- Node.js >= 18.0.0
- PostgreSQL database
- Redis instance
- n8n server (optional, for workflow execution)
- Windmill server (optional, for app execution)

### Installation

```bash
cd api
npm install
npm run build
```

### Configuration

The API uses environment variables for configuration:

```bash
# API Server
METAREASONING_API_PORT=8093
METAREASONING_API_HOST=localhost

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=metareasoning
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your-password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# n8n Integration
N8N_BASE_URL=http://localhost:5678
N8N_WEBHOOK_BASE=http://localhost:5678/webhook

# Windmill Integration
WINDMILL_BASE_URL=http://localhost:8000
WINDMILL_WORKSPACE=demo
```

### Running the Server

```bash
# Production
npm start

# Development (with auto-reload)
npm run dev

# Development with file watching
npm run dev:watch
```

## API Endpoints

### Health Check

**GET** `/api/health`

Check the health status of the API and its dependencies.

```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "version": "1.0.0",
  "services": {
    "database": "connected",
    "redis": "connected"
  }
}
```

### Prompt Management

**GET** `/api/prompts`

List all available reasoning prompts.

Query Parameters:
- `category` (optional): Filter by category
- `limit` (optional): Number of results (default: 20)
- `offset` (optional): Pagination offset

**GET** `/api/prompts/:id`

Get a specific prompt by ID.

**POST** `/api/prompts`

Create a new reasoning prompt.

Request Body:
```json
{
  "name": "Decision Framework",
  "category": "decision",
  "description": "Framework for structured decision-making",
  "content": "Given the scenario: {scenario}, analyze...",
  "parameters": ["scenario", "options", "criteria"]
}
```

**PUT** `/api/prompts/:id`

Update an existing prompt.

**DELETE** `/api/prompts/:id`

Delete a prompt.

**POST** `/api/prompts/:id/test`

Test a prompt with sample input.

Request Body:
```json
{
  "input": "Should we migrate to microservices?"
}
```

### Workflow Management

**GET** `/api/workflows`

List all workflows.

Query Parameters:
- `type` (optional): Filter by type (`n8n` or `windmill`)
- `limit` (optional): Number of results
- `offset` (optional): Pagination offset

**GET** `/api/workflows/:id`

Get workflow details.

**POST** `/api/workflows/:id/execute`

Execute a workflow.

Request Body:
```json
{
  "input": {
    "topic": "Remote work policy",
    "depth": "comprehensive"
  },
  "async": false,
  "timeout": 30000
}
```

**GET** `/api/workflows/:id/executions/:executionId`

Get workflow execution status.

### Analysis Endpoints

**POST** `/api/analysis/decision`

Perform decision analysis.

Request Body:
```json
{
  "scenario": "Should we migrate to microservices?",
  "options": ["Microservices", "Monolith", "Hybrid"],
  "criteria": ["Cost", "Scalability", "Complexity"],
  "include_recommendation": true
}
```

Response:
```json
{
  "status": "success",
  "data": {
    "recommendation": "Based on the analysis, Microservices is recommended",
    "confidence": 0.85,
    "reasoning": [
      "Better scalability for growth",
      "Team has necessary expertise",
      "Cost justified by benefits"
    ],
    "options_analysis": [...],
    "next_steps": [...]
  }
}
```

**POST** `/api/analysis/pros-cons`

Generate pros and cons analysis.

Request Body:
```json
{
  "topic": "Remote work policy",
  "depth": "detailed",
  "context": "Post-pandemic workplace"
}
```

**POST** `/api/analysis/swot`

Perform SWOT analysis.

Request Body:
```json
{
  "target": "New product launch",
  "context": "Q1 2024 market conditions",
  "include_recommendations": true
}
```

**POST** `/api/analysis/risks`

Assess risks.

Request Body:
```json
{
  "proposal": "Cloud migration project",
  "risk_categories": ["Technical", "Financial", "Operational"],
  "include_mitigation": true
}
```

### Template Management

**GET** `/api/templates`

List reasoning templates.

**GET** `/api/templates/:id`

Get template details.

**POST** `/api/templates`

Create a new template.

**POST** `/api/templates/:id/apply`

Apply a template with specific context.

## Authentication

The API supports token-based authentication for production use:

1. Include an API token in the `Authorization` header:
```
Authorization: Bearer your-api-token
```

2. API tokens can be managed through the database or CLI.

## Rate Limiting

Default rate limits:
- 100 requests per minute per IP
- Configurable via environment variables

## Error Handling

The API returns consistent error responses:

```json
{
  "error": "Resource not found",
  "message": "Workflow with ID xyz not found",
  "requestId": "req-123",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

HTTP Status Codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `404` - Not Found
- `429` - Too Many Requests
- `500` - Internal Server Error
- `503` - Service Unavailable

## Testing

```bash
# Run all tests
npm test

# Type checking
npm run type-check

# Linting
npm run lint
npm run lint:fix
```

## Development

### Project Structure

- **Controllers**: Handle HTTP requests and responses
- **Services**: Contain business logic and orchestration
- **Repositories**: Abstract database operations
- **Schemas**: Zod schemas for request/response validation
- **Middleware**: Cross-cutting concerns (auth, logging, errors)
- **Config**: Application configuration and DI setup

### Adding a New Endpoint

1. Define schemas in `src/schemas/`
2. Create service method in `src/services/`
3. Add controller method in `src/controllers/`
4. Register route in `src/routes/`
5. Add tests in `*.test.ts` files

### Database Migrations

The database schema is defined in `initialization/storage/postgres/schema.sql`. Apply migrations:

```bash
psql -U postgres -d metareasoning < initialization/storage/postgres/schema.sql
```

## Integration with n8n

The API integrates with n8n workflows through webhook endpoints:

1. Workflows are registered in the database
2. API triggers workflows via HTTP webhooks
3. Results are processed and returned

Example n8n webhook URL:
```
http://localhost:5678/webhook/pros-cons-analyzer
```

## Integration with Windmill

For Windmill scripts and apps:

1. Scripts are registered with workspace and path
2. API triggers jobs via Windmill API
3. Polls for completion and returns results

## CLI Usage

Use the CLI to interact with the API:

```bash
# Check health
metareasoning health

# List prompts
metareasoning prompt list

# Run analysis
metareasoning analyze decision "Should we adopt AI?"

# Execute workflow
metareasoning workflow run pros-cons-analyzer
```

## Production Deployment

1. Build the TypeScript code:
```bash
npm run build
```

2. Set production environment variables
3. Run with process manager:
```bash
NODE_ENV=production node dist/server.js
```

4. Use reverse proxy (nginx) for SSL termination
5. Configure monitoring and logging

## Monitoring

The API exposes metrics:
- Request count and latency
- Database connection pool stats
- Redis connection status
- Workflow execution metrics

Access metrics at `/api/metrics` (when enabled).

## License

GPL-3.0 - Part of the Vrooli project