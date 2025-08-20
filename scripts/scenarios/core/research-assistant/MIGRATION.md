# Workflow Migration Plan

## Purpose
Track the migration of direct resource API calls to shared workflows for better reliability and consistency.

## Workflows Requiring Migration

### 1. research-orchestrator.json
- **Line 236**: Direct Ollama generation call → Migrate to shared `ollama.json` workflow
- **Line 304**: Direct Ollama embeddings call → Migrate to shared `embedding-generator.json` workflow

### 2. chat-rag-workflow.json  
- **Line 78**: Direct Ollama embeddings call → Migrate to shared `embedding-generator.json` workflow
- **Line 149**: Direct Ollama generation call → Migrate to shared `ollama.json` workflow

### 3. main-workflow.json
- Contains template syntax - will be addressed during scenario conversion

## Migration Benefits
- Centralized error handling
- Consistent timeout management
- Better resource utilization through CLI integration
- Easier maintenance and updates

## Implementation Notes
- Use n8n's Execute Workflow node to call shared workflows
- Pass parameters via JSON payload matching the shared workflow's webhook structure
- Ensure workflow IDs are properly referenced