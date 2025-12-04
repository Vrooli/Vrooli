# Resource Health Checks

Resource checks monitor Vrooli-managed infrastructure services via the `vrooli resource` CLI.

---

## resource-postgres: PostgreSQL Database

**Interval:** 60 seconds
**Platforms:** All

Monitors the PostgreSQL database resource.

### Why It Matters
PostgreSQL is the primary data store for:
- Scenario state and configuration
- User data and sessions
- Health check history
- Action logs

### Status Meanings
- **OK**: Database running and healthy
- **Warning**: Degraded but partially functional
- **Critical**: Database down or unreachable

### Recovery Actions
- **Start**: `vrooli resource start postgres`
- **Stop**: `vrooli resource stop postgres`
- **Restart**: `vrooli resource restart postgres`
- **View Logs**: Recent container logs

### Troubleshooting
1. Check status: `vrooli resource status postgres`
2. View logs: `vrooli resource logs postgres`
3. Check disk space (database can fill disk)
4. Verify port 5432 is not in use by another process

---

## resource-redis: Redis Cache

**Interval:** 60 seconds
**Platforms:** All

Monitors the Redis cache resource.

### Why It Matters
Redis provides:
- Session storage
- Real-time pub/sub messaging
- Rate limiting counters
- Temporary data caching

### Status Meanings
- **OK**: Redis running and healthy
- **Critical**: Redis down

### Recovery Actions
- **Start/Stop/Restart**: Lifecycle management
- **View Logs**: Recent container logs

### Troubleshooting
1. Check memory: Redis can hit maxmemory limits
2. Check persistence: RDB/AOF file issues
3. Verify port 6379 availability

---

## resource-ollama: Ollama AI

**Interval:** 60 seconds
**Platforms:** All

Monitors the Ollama local AI inference service.

### Why It Matters
Ollama provides:
- Local LLM inference
- Embeddings generation
- AI-powered features without API costs

### Status Meanings
- **OK**: Ollama running and ready
- **Warning**: Running but degraded
- **Critical**: Ollama down

### Recovery Actions
- **Start/Stop/Restart**: Lifecycle management
- **View Logs**: Recent service logs

### Troubleshooting
1. Check GPU availability: `nvidia-smi`
2. Verify model downloaded: `ollama list`
3. Check memory: Large models need significant RAM
4. Port 11434 must be available

---

## resource-qdrant: Qdrant Vector Database

**Interval:** 60 seconds
**Platforms:** All

Monitors the Qdrant vector database.

### Why It Matters
Qdrant provides:
- Vector similarity search
- Semantic search capabilities
- Embedding storage

### Status Meanings
- **OK**: Qdrant running and healthy
- **Critical**: Qdrant down

### Recovery Actions
- **Start/Stop/Restart**: Lifecycle management
- **View Logs**: Recent container logs

### Troubleshooting
1. Check disk space for index storage
2. Verify port 6333 availability
3. Check memory usage

---

## resource-searxng: SearXNG Search

**Interval:** 60 seconds
**Platforms:** All

Monitors the SearXNG metasearch engine.

### Why It Matters
SearXNG provides:
- Web search capabilities
- Privacy-focused search aggregation
- Research and information gathering

### Status Meanings
- **OK**: SearXNG running and healthy
- **Critical**: SearXNG down

### Recovery Actions
- **Start/Stop/Restart**: Lifecycle management
- **View Logs**: Recent container logs

### Troubleshooting
1. Check network connectivity (requires internet)
2. Verify upstream search engines are accessible
3. Check rate limiting issues

---

## resource-browserless: Browserless Chrome

**Interval:** 60 seconds
**Platforms:** All

Monitors the Browserless headless Chrome service.

### Why It Matters
Browserless provides:
- Web scraping capabilities
- Screenshot generation
- PDF rendering
- Browser automation

### Status Meanings
- **OK**: Browserless running and healthy
- **Critical**: Browserless down

### Recovery Actions
- **Start/Stop/Restart**: Lifecycle management
- **View Logs**: Recent container logs

### Troubleshooting
1. Check memory: Chrome is memory-intensive
2. Verify port 3000 availability
3. Check for zombie Chrome processes
4. Review concurrent connection limits
