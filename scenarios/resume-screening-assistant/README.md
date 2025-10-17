# Resume Screening Assistant

## Purpose
AI-powered resume screening and candidate evaluation platform with semantic matching and automated assessment workflows. Helps HR teams efficiently process large volumes of resumes and find the best candidates for open positions.

## Key Features
- **Smart Resume Processing**: Extracts key information from resumes using Ollama LLM
- **Semantic Matching**: Uses vector embeddings for intelligent candidate-job matching  
- **Automated Workflows**: N8n workflows handle processing, searching, and job management
- **Retro-Futuristic UI**: JavaScript-based UI with a unique "HyperRecruit 3000" theme

## Resources Used
- **Ollama**: For AI text analysis and extraction (via shared workflow)
- **N8n**: Workflow automation backbone
- **PostgreSQL**: Structured data storage for candidates and jobs
- **Qdrant**: Vector database for semantic search

## Workflows
1. **resume-processing-pipeline.json**: Processes uploaded resumes, extracts info, generates embeddings
2. **job-management-workflow.json**: Manages job postings and requirements
3. **semantic-search-workflow.json**: Finds best candidate matches using vector similarity

## API Endpoints
- `GET /health` - Health check
- `GET /api/jobs` - List all job postings
- `GET /api/candidates` - List all candidates
- `POST /api/search` - Semantic search for candidates

## CLI Usage
```bash
# After installation
resume-screening-assistant --help
resume-screening-assistant process-resume <file>
resume-screening-assistant search-candidates <job-id>
resume-screening-assistant list-jobs
```

## Integration with Other Scenarios
Can be used by:
- **product-manager-agent**: For hiring product team members
- **personal-relationship-manager**: Track professional contacts from resumes
- **mind-maps**: Build knowledge graphs from candidate skills

## UI Style
Retro-futuristic "HyperRecruit 3000" theme with:
- Neon green accents on dark background
- Terminal-style fonts
- Animated scan lines and glitch effects
- Dashboard showing real-time processing stats