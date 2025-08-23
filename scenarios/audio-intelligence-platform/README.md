# Audio Intelligence Platform

## Purpose
A comprehensive audio processing and intelligence system that handles transcription, AI analysis, and semantic search across audio content. This scenario enables Vrooli to process spoken content, extract insights, and make audio content searchable and actionable.

## Core Features
- **Audio Transcription**: Convert audio files to text using Whisper
- **AI Analysis**: Generate summaries, extract key insights, or perform custom analysis on transcriptions
- **Semantic Search**: Find relevant content across all transcriptions using vector embeddings
- **Multi-format Support**: Handles various audio formats (MP3, WAV, M4A, etc.)
- **Batch Processing**: Process multiple audio files efficiently

## Dependencies
### Required Resources
- **whisper**: Audio transcription engine
- **ollama**: AI model for analysis and embeddings
- **postgres**: Stores transcriptions and metadata
- **qdrant**: Vector database for semantic search
- **n8n**: Workflow automation engine

### Shared Workflows Used
- `initialization/n8n/ollama.json`: For AI generation and embeddings
- `initialization/n8n/embedding-generator.json`: For generating embeddings

## Integration Points
### Cross-Scenario Usage
- **stream-of-consciousness-analyzer**: Can process voice notes and recordings
- **morning-vision-walk**: Transcribes and analyzes walk recordings
- **study-buddy**: Processes lecture recordings for study material
- **notes**: Integrates transcribed audio into note-taking system

### API Endpoints
- `POST /api/upload`: Upload audio files for transcription
- `GET /api/transcriptions`: List all transcriptions
- `POST /api/analyze`: Perform AI analysis on transcriptions
- `POST /api/search`: Semantic search across transcriptions

### CLI Commands
```bash
audio-intelligence-platform upload <file>     # Upload and transcribe audio
audio-intelligence-platform list              # List all transcriptions
audio-intelligence-platform analyze <id>      # Analyze a transcription
audio-intelligence-platform search <query>    # Search transcriptions
```

## UI/UX Style
**Theme**: Professional audio studio aesthetic
- Dark background with waveform visualizations
- Audio player with real-time transcription display
- Split-panel interface: audio controls on left, transcription/analysis on right
- Subtle animations for processing states
- Color scheme: Dark blues and teals with bright accent colors for interactions

## Technical Architecture
- **API**: Go-based REST API handling file uploads and processing
- **Storage**: PostgreSQL for structured data, filesystem for audio files
- **Processing**: Asynchronous pipeline using n8n workflows
- **Search**: Vector embeddings stored in Qdrant for semantic similarity

## Value Proposition
This scenario transforms Vrooli into an audio intelligence hub, enabling:
- Automatic meeting transcription and insight extraction
- Podcast/lecture content searchability
- Voice note organization and retrieval
- Audio content analysis for other scenarios

## Configuration
Default settings in `initialization/configuration/`:
- `transcription-config.json`: Whisper model settings
- `search-config.json`: Embedding and search parameters
- `ui-config.json`: Interface customization options

## Testing
- `test.sh`: Basic functionality tests
- `custom-tests.sh`: Audio processing validation
- `test/test-analysis-endpoint.sh`: API endpoint testing

## Performance Considerations
- Audio files are processed asynchronously to prevent blocking
- Transcription uses chunked processing for long audio files
- Vector embeddings are generated in batches for efficiency
- Search results are cached for frequently accessed queries

## Future Enhancements
- Real-time transcription for live audio streams
- Multi-language support with automatic detection
- Speaker diarization for multi-person recordings
- Integration with video processing scenarios