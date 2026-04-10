# Morning Vision Walk

## Purpose
AI companion for morning walks that helps brainstorm ideas, understand Vrooli's current state, and collaboratively plan the day ahead. Designed for voice interaction during walks to transform unstructured thoughts into actionable insights.

## How It Helps Vrooli
- **Recursive Intelligence**: Generates insights that become training data for future conversations
- **Cross-Scenario Integration**: Feeds tasks to task-planner, ideas to idea-generator, and thoughts to stream-of-consciousness-analyzer
- **Meta-Planning**: Helps identify what scenarios and resources need improvement
- **Context Building**: Creates embeddings of conversations that other scenarios can search

## Dependencies
- **Shared Workflows**: ollama, chain-of-thought-orchestrator, embedding-generator, universal-rag-pipeline
- **Resources**: n8n, Ollama (llama3.2), PostgreSQL, Qdrant
- **Integrations**: stream-of-consciousness-analyzer, task-planner, mind-maps

## Key Features
- Voice-first interaction for hands-free brainstorming
- Automatic task extraction from conversations
- Context-aware responses using past conversation embeddings
- Daily planning with priority-based task organization
- Insight generation with breakthrough detection

## UX Style
Clean, calming interface with nature-inspired colors. Optimized for mobile use during walks. Voice input/output as primary interaction mode with visual feedback for key insights.

## CLI Usage
```bash
# Start a conversation
morning-vision-walk start  # alias: vision-walk start

# Review today's insights
morning-vision-walk insights --today

# Export tasks to task-planner
morning-vision-walk export-tasks

# Search past conversations
morning-vision-walk search "scenario improvements"
```

## API Endpoints
- `POST /conversation` - Handle conversation turn
- `GET /insights` - Retrieve generated insights
- `GET /tasks` - Get extracted tasks
- `POST /context` - Gather Vrooli context

## Validation
- Conversation endpoint responds with AI-generated response
- Tasks are properly formatted and stored in PostgreSQL
- Embeddings are generated and searchable in Qdrant
- UI shows conversation history and extracted insights
