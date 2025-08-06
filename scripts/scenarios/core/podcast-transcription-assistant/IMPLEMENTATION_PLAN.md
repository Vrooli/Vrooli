# Podcast Transcription Assistant - Implementation Plan

## Overview

A comprehensive podcast transcription tool that allows users to:
- Upload audio files for transcription
- Browse and manage transcription history
- Perform semantic search across transcriptions
- Use AI analysis tools (summarize, insights, custom prompts)
- View transcriptions in detailed popup interface

## Architecture

### Resource Stack
- **Whisper** (port 8090): Core transcription engine
- **Ollama** (port 11434): AI analysis and embeddings
- **PostgreSQL** (port 5433): Transcription metadata and analysis results
- **MinIO** (port 9000): Audio file and transcription storage
- **Qdrant** (port 6333): Vector database for semantic search
- **Windmill** (port 5681): Professional UI application
- **n8n** (port 5678): Workflow automation between services

## Implementation Steps

### 1. Service Configuration (.vrooli/service.json)
**Status:** Pending
- Enable MinIO for file storage
- Enable Qdrant for semantic search
- Enable Windmill for UI
- Configure proper initialization phases
- Set up MinIO buckets: audio-files, transcriptions, exports
- Configure Qdrant collections: transcription-embeddings

### 2. Database Schema (initialization/storage/schema.sql)
**Status:** Pending

Core tables:
```sql
-- Transcriptions management
transcriptions (
    id, filename, file_path, transcription_text, 
    duration_seconds, file_size_bytes, whisper_model_used,
    created_at, updated_at, embedding_status
)

-- AI analysis results
ai_analyses (
    id, transcription_id, analysis_type, prompt_used, 
    result_text, created_at, processing_time_ms
)

-- User sessions and preferences
user_sessions (
    id, session_id, preferences, search_history,
    created_at, last_activity
)
```

### 3. n8n Workflows (initialization/automation/n8n/)
**Status:** Pending

**A. Transcription Pipeline** (`transcription-pipeline.json`)
- Webhook trigger for file uploads
- Store audio file in MinIO
- Send to Whisper for transcription
- Store transcription text in PostgreSQL
- Generate embeddings via Ollama
- Store embeddings in Qdrant
- Update UI with completion status

**B. AI Analysis Workflow** (`ai-analysis-workflow.json`)
- Webhook trigger with transcription ID + analysis type
- Fetch transcription text from PostgreSQL
- Send to Ollama with appropriate prompt:
  - Summary: "Provide a concise summary of this transcription"
  - Key Insights: "Extract key insights and main points"
  - Custom: Use user-provided prompt
- Store analysis result in ai_analyses table
- Return formatted result

**C. Semantic Search Workflow** (`semantic-search.json`)
- Webhook trigger with search query
- Generate query embedding via Ollama
- Search Qdrant for similar transcriptions
- Fetch transcription details from PostgreSQL
- Return ranked results with similarity scores

### 4. Windmill UI (initialization/automation/windmill/)
**Status:** Pending

**Main Application** (`transcription-manager-app.json`)

**Layout Components:**
- **Header**: Application title and upload button
- **Search Bar**: Text input with semantic search
- **Transcription Grid**: Cards showing transcription history
- **Popup Modal**: Detailed transcription viewer with AI tools
- **Settings Panel**: Whisper model selection, UI preferences

**Key Features:**
- File upload with progress indicator
- Transcription status tracking
- Search with live results
- AI analysis buttons in popup
- Export capabilities (TXT, PDF, JSON)

**Supporting Scripts:**
- `upload-handler.ts`: Process file uploads, trigger n8n workflow
- `search-handler.ts`: Handle search queries and results
- `ai-analysis.ts`: Trigger AI analysis workflows
- `transcription-viewer.ts`: Format and display transcription content

### 5. Configuration Files (initialization/configuration/)
**Status:** Pending

**A. Transcription Config** (`transcription-config.json`)
```json
{
  "whisper": {
    "model": "base",
    "language": "auto",
    "output_format": "json",
    "word_timestamps": true
  },
  "file_handling": {
    "max_size_mb": 500,
    "supported_formats": ["mp3", "wav", "m4a", "ogg", "flac"],
    "storage_retention_days": 365
  }
}
```

**B. Search Config** (`search-config.json`)
```json
{
  "qdrant": {
    "collection_name": "transcription-embeddings",
    "vector_size": 384,
    "distance_metric": "cosine"
  },
  "search": {
    "max_results": 50,
    "similarity_threshold": 0.7,
    "enable_hybrid_search": true
  }
}
```

**C. UI Config** (`ui-config.json`)
```json
{
  "appearance": {
    "theme": "professional",
    "primary_color": "#2563eb",
    "items_per_page": 20
  },
  "features": {
    "auto_save_search": true,
    "keyboard_shortcuts": true,
    "export_formats": ["txt", "pdf", "json", "srt"]
  }
}
```

### 6. Deployment Scripts (deployment/)
**Status:** Pending

**A. Startup Script** (`startup.sh`)
- Initialize MinIO buckets
- Create Qdrant collections
- Apply database schema
- Import n8n workflows
- Deploy Windmill application
- Run health checks

- Validate search functionality
- Confirm AI analysis tools

## Data Flow

### Upload & Transcription Flow
1. User uploads audio file via Windmill UI
2. File stored in MinIO `audio-files` bucket
3. n8n triggers Whisper transcription
4. Transcription text stored in PostgreSQL
5. Ollama generates text embeddings
6. Embeddings stored in Qdrant collection
7. UI updates with transcription status

### Search Flow
1. User enters search query in UI
2. Windmill script calls n8n search workflow
3. Query embedded via Ollama
4. Qdrant returns similar transcriptions
5. PostgreSQL provides transcription details
6. Results displayed in UI with relevance scores

### AI Analysis Flow
1. User selects transcription and clicks AI button
2. Choose analysis type (summary/insights/custom)
3. n8n workflow fetches transcription text
4. Ollama processes with appropriate prompt
5. Analysis result stored in database
6. Result displayed in popup modal

## User Experience Features

### File Management
- Drag-and-drop upload interface
- Upload progress indicators
- File format validation
- Duplicate detection

### Transcription Browser
- Grid/list view toggle
- Sort by date, duration, filename
- Filter by date range, file type
- Bulk operations (delete, export)

### Search Interface
- Real-time search suggestions
- Search history
- Advanced filters (date, duration)
- Search result highlighting

### AI Analysis Tools
- Quick action buttons (Summary, Key Insights)
- Custom prompt input field
- Analysis history per transcription
- Copy/export analysis results

## Performance Considerations

### File Handling
- Chunked uploads for large files
- Background transcription processing
- Progress webhooks for UI updates

### Search Optimization
- Vector index optimization
- Result caching for common queries
- Pagination for large result sets

### Storage Management
- Automatic file cleanup policies
- Compression for stored transcriptions
- CDN-style delivery for audio playback

## Security & Privacy

### File Security
- Secure file upload validation
- Virus scanning integration
- Access control for stored files

### Data Protection
- Encryption at rest (MinIO + PostgreSQL)
- Session-based access control
- Audit logging for file access

## Success Metrics

### Functional Requirements
- ✅ File upload works for all supported formats
- ✅ Transcription accuracy meets user expectations
- ✅ Semantic search returns relevant results
- ✅ AI analysis tools provide valuable insights
- ✅ UI is intuitive and responsive

### Performance Targets
- File upload: < 5 seconds for 100MB files
- Transcription: ~1x realtime (1 hour audio = ~1 hour processing)
- Search: < 2 seconds for query results
- AI analysis: < 30 seconds for summary generation

### Business Value
- Saves 80% of manual transcription time
- Enables content discovery through search
- Provides professional-grade analysis tools
- Suitable for deployment as SaaS product

## Next Steps

1. ✅ Create implementation plan
2. 🔄 Update service.json configuration
3. ⏳ Implement database schema
4. ⏳ Create n8n workflows
5. ⏳ Build Windmill UI application
6. ⏳ Configure resource settings
7. ⏳ Update deployment scripts
8. ⏳ Integration testing

---

*This plan serves as the authoritative guide for implementing the podcast transcription assistant scenario.*