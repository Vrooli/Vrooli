# Ollama Embedding Models Compatibility Matrix

This document provides guidance on which Ollama models to use for different embedding tasks, with emphasis on dimension compatibility with Qdrant collections.

## 🎯 Quick Reference

**For most use cases, use the `embedding.default` role** — resolve it with
`resource-ollama policy resolve --role embedding.default` (currently a 768-dim
text-embedding model). The role is the source of truth; the concrete tag below
is shown only to illustrate the dimensions contract.

## 📊 Dimensions contract

Embedding dimensions must match the target Qdrant collection. The
`embedding.default` role is sized for the project's 768-dim collections.

| Model class | Dimensions | Status | Notes |
|-------------|------------|--------|-------|
| `embedding.default` role (768-dim text embedder) | **768** | ✅ **USE THIS** | Text embeddings, property search, code similarity |
| General chat/code models (8B+) | 4096 | ❌ **AVOID** | Wrong dimensionality for the 768-dim collections — use them for chat/code, never embeddings |
| Vision models | N/A | ❌ **NO EMBEDDINGS** | No embedding output at all |

## 🗄️ Qdrant Collection Compatibility

### Current Collections

| Collection | Dimensions | Compatible Model | Purpose |
|------------|------------|------------------|---------|
| `code_embeddings` | 768 | `nomic-embed-text:latest` | Code similarity search |
| `real_estate_properties_v2` | 768 | `nomic-embed-text:latest` | Property matching |
| `agent_memory` | 1536 | ⚠️ **NEEDS MODEL** | AI agent memory |
| `conversation_history` | 1536 | ⚠️ **NEEDS MODEL** | Chat context |
| `document_chunks` | 1536 | ⚠️ **NEEDS MODEL** | Document RAG |

`mxbai-embed-large:latest` is available as an optional 1024-dimensional model for
shadow migrations and experiments. It is deliberately not the default role. Existing
768-dimensional collections must be reindexed into a suffixed shadow collection and
evaluated before any provider pointer is changed.

### Missing Models for 1536-Dimensional Collections

Currently, there are no local Ollama models that produce 1536 dimensions. Options:

1. **Cloud API Integration**: Use OpenAI's `text-embedding-ada-002` (1536 dims)
2. **Collection Migration**: Recreate 1536-dim collections as 768-dim
3. **Hybrid Approach**: Use cloud for some collections, local for others

## 🚀 Usage Examples

### ✅ Correct Usage - Property Search

```bash
# Generate embedding for property search
curl -X POST "http://localhost:11434/api/embed" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "nomic-embed-text:latest",
    "input": "Looking for a modern downtown condo with parking"
  }' > client_preferences.json

# Extract vector and search
VECTOR=$(jq '.embeddings[0]' client_preferences.json)
curl -X POST "http://localhost:6333/collections/real_estate_properties_v2/points/search" \
  -H "Content-Type: application/json" \
  -d "{\"vector\": $VECTOR, \"limit\": 5, \"with_payload\": true}"
```

### ❌ Incorrect Usage - Dimension Mismatch

```bash
# DON'T DO THIS - Wrong model
curl -X POST "http://localhost:11434/api/embed" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.1:8b",
    "input": "property search"
  }'
# This produces 4096 dimensions but collection expects 768!
```

## 🔧 Installation Commands

### Install Recommended Embedding Model

```bash
# Install the primary embedding model
ollama pull nomic-embed-text:latest

# Verify installation
curl -X POST "http://localhost:11434/api/embed" \
  -H "Content-Type: application/json" \
  -d '{"model": "nomic-embed-text:latest", "input": "test"}' | \
  jq '.embeddings[0] | length'
# Should return: 768
```

### Verify Model Compatibility

```bash
# Check model dimensions
check_model_dimensions() {
    local model="$1"
    echo "Testing model: $model"
    dims=$(curl -s -X POST "http://localhost:11434/api/embed" \
           -H "Content-Type: application/json" \
           -d "{\"model\": \"$model\", \"input\": \"test\"}" | \
           jq '.embeddings[0] | length')
    echo "Dimensions: $dims"
}

check_model_dimensions "nomic-embed-text:latest"
```

## 🐛 Troubleshooting

### Problem: "Vector dimension error: expected dim: 768, got 4096"

**Cause**: Using wrong model (likely `llama3.1:8b` instead of `nomic-embed-text:latest`)

**Solution**:
```bash
# Use the correct model
curl -X POST "http://localhost:11434/api/embed" \
  -d '{"model": "nomic-embed-text:latest", "input": "your text"}'
```

### Problem: "Model not found"

**Cause**: Model not installed

**Solution**:
```bash
ollama pull nomic-embed-text:latest
```

### Problem: Silent insertion failures

**Cause**: Dimension mismatch with Qdrant collection

**Solution**: Always validate dimensions before insertion:
```bash
# Check collection dimensions
curl -X GET "http://localhost:6333/collections/your_collection" | \
  jq '.result.config.params.vectors.size'

# Check vector dimensions  
jq 'length' your_vector.json
```

## 📋 Best Practices

1. **Always use `nomic-embed-text:latest`** for new embedding tasks
2. **Validate dimensions** before inserting vectors into Qdrant
3. **Test with small samples** before processing large datasets
4. **Document your model choice** in project README files
5. **Use consistent models** across related collections

## 🔮 Future Considerations

- Monitor for new embedding-optimized models in Ollama
- Consider upgrading collections to higher-dimension models when available
- Evaluate trade-offs between local and cloud embedding services
- Plan for model version updates and migration strategies

---

The authoritative embedding model is the `embedding.default` role in
[`model-policy.json`](../model-policy.json); see git history for revision dates.
