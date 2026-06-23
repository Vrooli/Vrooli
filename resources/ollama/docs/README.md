# Ollama Documentation

Documentation for the Ollama local LLM inference resource.

Ollama runs **exclusively as a Docker container** managed by the Vrooli
docker-service driver — there is no host install and no `manage.sh` script. Lifecycle
goes through `vrooli resource …`; model commands run inside the container via
`docker exec ollama ollama …`.

## 📚 Documentation Structure

- [Installation Guide](INSTALLATION.md) — Docker-only setup, lifecycle, and configuration
- [Models Guide](MODELS.md) — available models and selection
- [Embedding Models](EMBEDDING_MODELS.md) — embedding model guidance
- [Operations](OPERATIONS.md) — architecture boundary and operator checklist

## 🚀 Quick Start

```bash
# Pull the image and start the container
vrooli resource install ollama
vrooli resource start ollama

# Check status and health
vrooli resource status ollama

# Pull a model and send a prompt
docker exec ollama ollama pull llama3.1:8b
curl http://localhost:11434/api/generate \
  -d '{"model":"llama3.1:8b","prompt":"Explain machine learning","stream":false}'
```

## 📋 Quick Reference

```bash
# Lifecycle (docker-service driver)
vrooli resource start|stop|restart|status|logs ollama

# Model management (no host CLI — run inside the container)
docker exec ollama ollama list
docker exec ollama ollama pull llama3.1:8b
docker exec ollama ollama rm llama3.1:8b
```

Scenarios declare model **roles** in their ollama dependency config; `resource-ollama
ensure` pulls any missing resolved models into the running container.

### Key Endpoints
- **Health**: http://localhost:11434/api/tags
- **Generate**: http://localhost:11434/api/generate
- **Embeddings**: http://localhost:11434/api/embeddings

## 🔗 Quick Links

- **Default URL**: http://localhost:11434
- **Model Recommendations**: [Models Guide](MODELS.md)
- **Official Ollama Docs**: https://ollama.ai/docs
