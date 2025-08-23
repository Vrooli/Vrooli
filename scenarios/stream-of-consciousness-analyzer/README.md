# Stream of Consciousness Analyzer

Transform unstructured thoughts, voice notes, and brain dumps into organized, searchable knowledge.

## Overview

The Stream of Consciousness Analyzer helps users capture fleeting thoughts and unstructured ideas, then automatically organizes them into structured notes with AI-powered insights. Perfect for:

- **Daily Journaling**: Capture morning thoughts and reflections
- **Meeting Notes**: Quick brain dumps after meetings
- **Creative Brainstorming**: Free-flow idea capture
- **Task Management**: Extract action items from rambling thoughts
- **Knowledge Management**: Build a searchable personal knowledge base

## Features

### Core Capabilities
- **Multi-modal Input**: Text, voice, or file upload
- **Campaign Organization**: Group thoughts by context (work, personal, projects)
- **AI-Powered Analysis**: Automatic extraction of topics, action items, and insights
- **Semantic Search**: Find related thoughts across time
- **Pattern Detection**: Identify recurring themes and ideas
- **Real-time Processing**: Instant organization as you type/speak

### Technical Features
- PostgreSQL for structured data storage
- Qdrant for semantic search capabilities
- N8n workflows for processing pipeline
- Responsive UI with calming, mindful aesthetic

## Architecture

### Components
- **API Server** (`api/`): Go-based backend for data management
- **UI Server** (`ui/`): Express.js server with interactive frontend
- **CLI Tool** (`cli/`): Command-line interface for batch processing
- **N8n Workflows** (`initialization/n8n/`):
  - `process-stream.json`: Main processing pipeline
  - `organize-thoughts.json`: Structure extraction
  - `extract-insights.json`: Pattern and insight detection
  - `campaign-context-builder.json`: Campaign management

### Data Flow
1. User inputs stream of consciousness text/voice
2. N8n workflow processes with Ollama for structure extraction
3. Data stored in PostgreSQL with metadata
4. Embeddings generated and stored in Qdrant
5. UI displays organized notes with insights

## Usage

### Web Interface
Access the mindful UI at `http://localhost:8091` to:
- Input thoughts via text or voice
- Switch between campaigns
- View AI-extracted insights
- Search through organized notes
- Track patterns over time

### CLI Usage
```bash
# Process a stream of text
soc-analyzer process "Just had a great idea about..."

# Process with specific campaign
soc-analyzer process --campaign work "Meeting notes: discussed roadmap..."

# Search notes
soc-analyzer search "onboarding improvements"

# Extract insights from recent entries
soc-analyzer insights --days 7
```

### API Endpoints
- `POST /api/process-stream` - Process raw text into organized notes
- `POST /api/organize` - Organize multiple thought fragments
- `POST /api/insights` - Extract insights from text
- `GET /api/campaigns` - List available campaigns
- `GET /api/search` - Semantic search through notes

## Configuration

### Environment Variables
```bash
PORT=8080              # API server port
UI_PORT=8091          # UI server port
N8N_URL=http://localhost:5678
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
QDRANT_URL=http://localhost:6333
OLLAMA_URL=http://localhost:11434
```

### Campaign Setup
Campaigns help organize thoughts by context. Default campaigns:
- `general` - Uncategorized thoughts
- `daily` - Daily reflections and journaling
- `work` - Professional ideas and tasks
- `personal` - Personal growth and goals

## Development

### Prerequisites
- Go 1.21+
- Node.js 18+
- PostgreSQL
- Qdrant
- N8n
- Ollama with llama3.2 model

### Setup
```bash
# Install dependencies
cd ui && npm install

# Initialize database
psql -f initialization/postgres/schema.sql

# Start services
vrooli scenario start stream-of-consciousness-analyzer
```

### Testing
```bash
# Run scenario tests
vrooli test scenario stream-of-consciousness-analyzer

# Test specific endpoints
curl -X POST http://localhost:8080/api/process-stream \
  -H "Content-Type: application/json" \
  -d '{"text": "Test stream", "campaign": "general"}'
```

## Use Cases

### Personal Knowledge Management
Build a searchable second brain by consistently capturing thoughts and letting AI organize them into interconnected knowledge.

### Meeting Documentation
Quickly dump meeting thoughts and automatically extract action items, decisions, and follow-ups.

### Creative Writing
Capture story ideas, character thoughts, and plot points in free-flow, then see them organized by theme.

### Mental Health Journaling
Track mood patterns, recurring thoughts, and personal insights over time with AI-assisted pattern recognition.

## Future Enhancements
- Voice transcription with Whisper
- Mobile app for on-the-go capture
- Integration with other note-taking tools
- Collaborative campaigns for team brainstorming
- Advanced visualization of thought networks