# Problems and Solutions Log

## 2025-09-28: Health Endpoint Timeout Issue

### Problem
The `/health` endpoint was timing out (>120s) when trying to check Qdrant collections.

### Root Cause
The `execResourceQdrant` function had no timeout, causing it to hang indefinitely when the resource-qdrant CLI encountered issues.

### Solution
Added a 5-second timeout context to the `execResourceQdrant` function:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
cmd := exec.CommandContext(ctx, s.config.ResourceCLI, args...)
```

### Impact
- Health endpoint response time reduced from timeout to ~1.2s
- API became responsive and usable

---

## 2025-09-28: Missing Ollama Models

### Problem
Health check reported missing required models: `llama3.2` and `nomic-embed-text`

### Solution
Installed the models using:
```bash
ollama pull llama3.2
ollama pull nomic-embed-text
```

### Impact
- Ollama dependency status changed from unhealthy to degraded
- Enhanced semantic analysis capabilities available

---

## 2025-09-28: Empty Qdrant Collections

### Problem
All 44 Qdrant collections exist but have 0 points (no data).

### Root Cause
Collections were created but never populated with actual vector embeddings.

### Current State
- Collections are accessible via API
- Search returns empty results (expected with no data)
- Knowledge graph returns empty structure

### Recommended Fix
Future agents should populate collections using:
- `resource-qdrant embeddings add` commands
- Bulk data import from existing knowledge sources
- Incremental population as new scenarios are created

---

## 2025-09-28: CLI Test Failures

### Problem
Two CLI tests fail in the bats test suite:
- "CLI status command works" (test 4)
- "CLI status with JSON flag" (test 5)

### Root Cause
The CLI defaults to port 20270 but the API runs on dynamically allocated ports (e.g., 17822).

### Workaround
Set the API_PORT environment variable before running CLI:
```bash
export API_PORT=17822
knowledge-observatory status
```

### Recommended Fix
Update CLI to discover the API port dynamically or use a consistent port allocation strategy.

---

## 2025-09-28: Resource-Qdrant CLI Exit Status 5

### Problem
Many `resource-qdrant collections info` commands return exit status 5.

### Impact
- Individual collection info retrieval fails
- Health check marks some collections as degraded

### Current Workaround
Use direct Qdrant REST API for collection information:
```bash
curl http://localhost:6333/collections/COLLECTION_NAME
```

### Recommended Investigation
Check resource-qdrant CLI implementation for why info command fails while list command works.