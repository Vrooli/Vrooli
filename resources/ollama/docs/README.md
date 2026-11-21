# Ollama Documentation

Documentation for the Ollama local LLM inference resource.

Ollama runs as a Vrooli-managed service from a checksum-verified native artifact;
there is no host install, Docker fallback, or `manage.sh` script. Lifecycle
goes through `vrooli resource …`; model commands use the Ollama API.

## 📚 Documentation Structure

- [Installation Guide](INSTALLATION.md) — managed-service setup, lifecycle, and configuration
- [Models Guide](MODELS.md) — available models and selection
- [Embedding Models](EMBEDDING_MODELS.md) — embedding model guidance
- [Operations](OPERATIONS.md) — architecture boundary and operator checklist
- [Maturity assessment](maturity-assessment.md) — evidence, scores, and target limits

## 🚀 Quick Start

```bash
# Verify the pinned artifact and start the managed service
vrooli resource install ollama
vrooli resource start ollama

# Check status and health
vrooli resource status ollama

# Pull a model and send a prompt through the service API
curl http://localhost:11434/api/pull \
  -d '{"name":"llama3.1:8b","stream":false}'
curl http://localhost:11434/api/generate \
  -d '{"model":"llama3.1:8b","prompt":"Explain machine learning","stream":false}'
```

## 📋 Quick Reference

```bash
# Lifecycle (managed-service driver)
vrooli resource start|stop|restart|status|logs ollama

# Model management (use the service API or resource gateway)
curl http://localhost:11434/api/tags
resource-ollama ensure --config-base64 <scenario-model-config>
```

Scenarios declare model **roles** in their ollama dependency config. `resource-ollama
ensure` leaves installed models untouched. For missing models it requires a
shared disk-job reservation and a frozen default-registry manifest before the
Ollama pull request. Private registries and remote Ollama endpoints are refused.
Uncertain pull failures leave the owner lease active until stopped work and
cleanup can be verified.

### Key Endpoints
- **Health**: http://localhost:11434/api/tags
- **Generate**: http://localhost:11434/api/generate
- **Embeddings**: http://localhost:11434/api/embeddings

## 🔗 Quick Links

- **Default URL**: http://localhost:11434
- **Model Recommendations**: [Models Guide](MODELS.md)
- **Official Ollama Docs**: https://ollama.ai/docs
