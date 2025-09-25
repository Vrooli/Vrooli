# Idea Generator Platform

## Overview
The Idea Generator is an interactive AI-powered platform for brainstorming, refining, and managing innovative ideas within campaign contexts. It combines document intelligence, semantic search, and real-time AI chat refinement to create a comprehensive ideation workspace.

## Features

### 🎯 Core Capabilities
- **Campaign-Based Organization**: Organize ideas within color-coded campaign tabs
- **AI-Powered Idea Generation**: Interactive dice-roll interface for instant idea generation
- **Document Intelligence**: Upload and process documents (PDFs, DOCX, etc.) for AI context
- **Semantic Search**: Natural language search across all ideas and documents
- **Real-Time Chat Refinement**: Work with specialized AI agents to refine ideas
- **Multi-Agent System**: Six specialized agents (Revise, Research, Critique, Expand, Synthesize, Validate)

### 💡 Key Innovations
- **Context-Aware Generation**: Ideas are generated using campaign context and uploaded documents
- **Vector Embeddings**: All content is semantically searchable via Qdrant
- **Chat History Persistence**: All refinement conversations are saved and searchable
- **Idea Evolution Tracking**: Complete version history of idea development

## Architecture

### Resources Used
- **PostgreSQL**: Primary database for campaigns, ideas, and chat history
- **MinIO**: Document storage and file management
- **Qdrant**: Vector database for semantic search
- **Ollama**: Local LLM for idea generation and refinement
- **n8n**: Workflow orchestration for all backend processes
- **Windmill**: Interactive UI platform
- **Redis**: Real-time chat sessions and caching
- **Unstructured-IO**: Document processing and text extraction

### Workflows
1. **idea-generation-workflow**: Context-aware idea generation with embeddings
2. **document-processing-pipeline**: Upload → extraction → embeddings → storage
3. **semantic-search-workflow**: Hybrid vector + keyword search
4. **chat-refinement-workflow**: Real-time AI chat for idea development
5. **campaign-sync-workflow**: Campaign CRUD operations

## User Interface

### Main Components
- **Campaign Tabs**: Colored chips for switching between campaigns
- **Ideas Sidebar**: Searchable list of all ideas with filters
- **Center Workspace**: 
  - Campaign context display
  - Document upload zone
  - Animated dice button for idea generation
  - Idea input/edit area
- **Chat Panel**: Real-time refinement with AI agents

### Interactive Features
- Drag-and-drop document upload
- Real-time semantic search
- WebSocket-based chat
- Animated dice with sound effects
- Auto-save and version tracking

## Getting Started

### Prerequisites
All required resources are automatically configured through the Vrooli platform using the modern lifecycle system.

### Deployment
The scenario follows the modern Vrooli lifecycle pattern and can be launched through the orchestrator:

```bash
# Using the orchestrator (recommended)
vrooli scenario launch idea-generator

# Or manually using the service configuration
cd /path/to/idea-generator
vrooli service setup .
vrooli service develop .
```

### Access Points
- **UI**: Check logs for actual port (typically 35000-39999 range)
- **API**: Check logs for actual port (typically 15000-19999 range) 
- **API Health**: `curl http://localhost:<API_PORT>/health`
- **Workflows**: http://localhost:5678 (n8n if running)
- **Windmill**: http://localhost:5681 (if running)
- **CLI**: `idea-generator --help`

### Current Status (2025-09-24)
- ✅ API Server: Running and healthy
- ✅ UI Server: Running and serving web interface
- ✅ CLI: Installed and functional
- ⚠️  Resources: Some resources may need to be started separately
- ℹ️  Ports are dynamically allocated - check `make logs` for actual ports

### CLI Usage
The scenario includes a lightweight CLI for managing the platform:

```bash
# Check service status
idea-generator status

# List campaigns and ideas
idea-generator campaigns
idea-generator ideas

# Open web interfaces
idea-generator dashboard
idea-generator workflows-ui

# Check service health
idea-generator health
```

## Usage Guide

### Creating a Campaign
1. Click "+ New Campaign" in the tabs bar
2. Enter campaign name, description, and select a color
3. Add context information and objectives
4. Upload relevant documents for AI context

### Generating Ideas
1. Select a campaign tab
2. Click the dice button to generate an idea
3. Edit the generated idea or type your own
4. Click "Save Idea" to store or "Start Refining" to chat

### Refining Ideas
1. Click any idea in the sidebar to open chat
2. Select an agent type or use "Auto" mode
3. Chat with AI agents to develop the idea
4. Click "Store Idea" when refinement is complete

### Searching Content
1. Use the search bar in the sidebar
2. Search works across ideas and documents
3. Filter by status, tags, or campaign
4. Results show similarity scores

## Testing
```bash
# Run integration tests
./test.sh

# Test API endpoints
./test/test-api-endpoint.sh

# Test CLI functionality
cd cli && bats idea-generator.bats
```

## Value Proposition
This platform demonstrates a complete SaaS application ($10K-50K value) that showcases:
- Advanced AI integration with local models
- Document intelligence and processing
- Real-time collaborative features
- Enterprise-grade search capabilities
- Scalable multi-tenant architecture

## Technical Highlights
- **Recursive Learning**: Ideas and refinements improve future generations
- **Emergent Capabilities**: Agents discover novel combinations of resources
- **Self-Improvement**: Each interaction enhances the system's intelligence
- **Production-Ready**: Complete with authentication, persistence, and monitoring

## Future Enhancements
- Team collaboration features
- API integrations for external data sources
- Advanced analytics and insights
- Export to various formats (PDF, presentations)
- Mobile responsive design
- Multi-language support

## Support
For issues or questions, please refer to the Vrooli documentation or contact the development team.