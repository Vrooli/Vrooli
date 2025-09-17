# CrewAI Quick Start Examples

## Getting Started

These examples demonstrate how to use CrewAI for common tasks.

### 1. Create a Research Crew

```bash
# Load the research crew configuration
curl -X POST http://localhost:8084/inject \
  -H "Content-Type: application/json" \
  -d '{"path": "/home/matthalloran8/Vrooli/resources/crewai/examples/research_crew.json"}'

# Execute a research task
curl -X POST http://localhost:8084/execute \
  -H "Content-Type: application/json" \
  -d '{
    "crew_name": "research_crew",
    "input": {
      "topic": "AI agents and their applications",
      "depth": "comprehensive"
    }
  }'
```

### 2. Create Individual Agents

```bash
# Create a data analyst agent
curl -X POST http://localhost:8084/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "data_analyst",
    "role": "Data Analyst",
    "goal": "Analyze data and find insights",
    "backstory": "Expert in statistics and data analysis",
    "tools": ["database_query", "llm_query"]
  }'

# Create a writer agent
curl -X POST http://localhost:8084/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "content_writer",
    "role": "Content Writer",
    "goal": "Create engaging content",
    "backstory": "Professional writer with SEO expertise",
    "tools": ["web_search", "llm_query", "file_reader"]
  }'
```

### 3. Create a Custom Crew

```bash
# Create crew with multiple agents
curl -X POST http://localhost:8084/crews \
  -H "Content-Type: application/json" \
  -d '{
    "name": "content_team",
    "description": "Team for content creation",
    "agents": ["data_analyst", "content_writer"],
    "process": "sequential"
  }'
```

### 4. Execute Tasks

```bash
# Run a simple task
curl -X POST http://localhost:8084/execute \
  -H "Content-Type: application/json" \
  -d '{
    "crew_name": "content_team",
    "input": {
      "task": "Write a blog post about machine learning trends",
      "length": "1000 words",
      "style": "technical but accessible"
    }
  }'

# Check task status
curl -X GET http://localhost:8084/tasks/{task_id}
```

### 5. Using Tools

Available tools for agents:
- `web_search` - Search the web for information
- `file_reader` - Read local files
- `api_caller` - Call external APIs
- `database_query` - Query databases
- `llm_query` - Query Ollama LLM
- `memory_store` - Store information in Qdrant
- `memory_retrieve` - Retrieve information from Qdrant

### 6. Monitor Performance

```bash
# View metrics
curl -X GET http://localhost:8084/metrics | jq .

# Check active tasks
curl -X GET http://localhost:8084/tasks | jq .
```

## CLI Usage

```bash
# Start the service
vrooli resource crewai develop

# Check status
vrooli resource crewai status

# Run tests
vrooli resource crewai test all

# View logs
vrooli resource crewai logs
```

## Tips

1. **Use Sequential Process**: For tasks that depend on each other
2. **Use Hierarchical Process**: When you need a manager agent
3. **Assign Appropriate Tools**: Give agents only the tools they need
4. **Leverage Memory**: Use Qdrant for persistent agent memory
5. **Monitor Metrics**: Track success rates and execution times

## Troubleshooting

If crews aren't executing:
1. Check service health: `curl http://localhost:8084/health`
2. Verify agents exist: `curl http://localhost:8084/agents`
3. Check logs: `vrooli resource crewai logs`
4. Ensure dependencies (Ollama, Qdrant) are running