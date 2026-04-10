# Personal Digital Twin

## Purpose
Creates personalized AI agents that learn from user data and provide tailored interactions. Each digital twin embodies unique personality traits, knowledge, and conversation styles based on ingested data.

## Core Value to Vrooli
This scenario enables:
- **Personalized AI Interactions**: Each user gets a unique AI assistant that learns their preferences
- **Knowledge Preservation**: Captures and preserves individual or organizational knowledge
- **Cross-Scenario Integration**: Other scenarios can leverage digital twins for personalized experiences

## How Other Scenarios Use This
- **morning-vision-walk**: Can use your personal twin for brainstorming sessions
- **study-buddy**: Adapts teaching style based on your twin's learning preferences  
- **life-coach**: Provides advice aligned with your twin's values and goals
- **product-manager-agent**: Understands your work style and project preferences

## Key Features
- **Data Ingestion**: Processes documents, conversations, and notes to build knowledge base
- **Personality Modeling**: Captures traits like communication style, interests, expertise
- **Semantic Memory**: Vector-based knowledge retrieval for contextual responses
- **Fine-Tuning Support**: Can adapt base models to match persona characteristics
- **Multi-Persona Management**: Support multiple twins per user (personal, professional, etc.)

## Architecture
```
UI (React) → API (Go) → Resources
                ↓
    ├── PostgreSQL (personas, conversations)
    ├── Qdrant (vector embeddings)
    ├── N8N Workflows (automation)
    └── Ollama (AI inference)
```

## Shared Resources Used
- **ollama.json**: AI text generation for persona responses
- **embedding-generator.json**: Creates vector embeddings for knowledge base
- **smart-semantic-search.json**: Retrieves relevant context from memory
- **structured-data-extractor.json**: Processes uploaded documents

## CLI Usage
```bash
# Create a new persona
personal-digital-twin create --name "Professional Me" --traits "analytical,detail-oriented"

# Ingest data
personal-digital-twin ingest --persona-id <id> --file resume.pdf
personal-digital-twin ingest --persona-id <id> --text "I prefer concise communication"

# Chat with persona
personal-digital-twin chat --persona-id <id> --message "What are my main skills?"

# Export persona for backup
personal-digital-twin export --persona-id <id> --output persona-backup.json
```

## API Endpoints
- `POST /api/personas` - Create new digital twin
- `POST /api/personas/{id}/ingest` - Add data to knowledge base
- `POST /api/chat` - Interact with digital twin
- `GET /api/personas/{id}/knowledge` - Retrieve knowledge items
- `POST /api/personas/{id}/train` - Fine-tune underlying model

## N8N Workflows
1. **data-ingestion.json**: Processes documents and builds knowledge base
2. **persona-chat.json**: Handles conversations with context retrieval
3. **model-training.json**: Manages fine-tuning jobs for personalization

## Development
```bash
# Convert and start the scenario
vrooli scenario run personal-digital-twin
vrooli scenario start personal-digital-twin

# Access services
# UI: http://localhost:3080
# API: http://localhost:8090
# Chat: http://localhost:8000
```

## Future Enhancements
- Voice cloning for audio interactions
- Behavioral pattern learning from interaction history
- Cross-persona knowledge sharing with permissions
- Integration with external data sources (email, calendar, social)
- Persona marketplace for sharing anonymized traits

## Security Considerations
- All persona data encrypted at rest
- User authentication required for access
- Data segregation between personas
- GDPR-compliant data export/deletion

## Dependencies
- Ollama (required): AI inference
- PostgreSQL (required): Structured data storage
- Qdrant (required): Vector search
- MinIO (optional): File storage for large documents
- Unstructured.io (optional): Advanced document processing