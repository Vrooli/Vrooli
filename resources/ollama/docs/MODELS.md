# Ollama Models Guide

Guide to available models and selection strategies.

Ollama runs as a Docker container, so there is no host `ollama` CLI. Pull models
inside the container with `docker exec ollama ollama pull <model>`. Scenarios should
prefer declaring model **roles** in their ollama dependency config (resolved through
[`model-policy.json`](../model-policy.json)); `resource-ollama ensure` then pulls any
missing resolved models automatically. The manual `docker exec` commands below are for
ad-hoc/local use.

## Recommended Default Models

A good starting set covering most use cases:

| Model | Size | Best For | Download Size |
|-------|------|----------|---------------|
| **llama3.1:8b** | 8B params | General chat, Q&A | 4.7GB |
| **deepseek-r1:8b** | 8B params | Reasoning, math, analysis | 4.9GB |
| **qwen2.5-coder:7b** | 7B params | Code generation, debugging | 4.2GB |

**Total**: ~14GB storage required (stored in the `/root/.ollama` bind-mount volume).

## Model Categories

### General Purpose Models

Best for: Chat, Q&A, general assistance, content creation

```bash
docker exec ollama ollama pull llama3.1:8b      # Balanced performance
docker exec ollama ollama pull llama3.1:70b     # Highest quality (40GB)
docker exec ollama ollama pull llama3.2:3b      # Lightweight (2GB)
docker exec ollama ollama pull gemma2:9b        # Google's model
docker exec ollama ollama pull mistral:7b       # Fast inference
```

### Code Generation Models

Best for: Programming, code review, debugging, technical documentation

```bash
docker exec ollama ollama pull qwen2.5-coder:7b    # Multi-language coding
docker exec ollama ollama pull codellama:13b       # Meta's code model
docker exec ollama ollama pull deepseek-coder:6.7b # Strong code reasoning
docker exec ollama ollama pull codegemma:7b        # Google code model
```

### Reasoning & Math Models

Best for: Logic problems, mathematical reasoning, step-by-step analysis

```bash
docker exec ollama ollama pull deepseek-r1:8b      # Advanced reasoning
docker exec ollama ollama pull qwen2.5:14b         # Strong math skills
docker exec ollama ollama pull llama3.1:70b        # Best overall reasoning
```

### Vision/Multimodal Models

Best for: Image analysis, visual Q&A, document understanding

```bash
docker exec ollama ollama pull llava:13b           # Image + text
docker exec ollama ollama pull llava-phi3:3.8b     # Lightweight vision
docker exec ollama ollama pull bakllava:7b         # Alternative vision model
```

### Lightweight Models

Best for: Resource-constrained environments, quick responses

```bash
docker exec ollama ollama pull llama3.2:3b         # 2GB, good performance
docker exec ollama ollama pull qwen2.5:3b          # Multilingual, compact
docker exec ollama ollama pull gemma2:2b           # Ultra-lightweight
```

## Model Selection Strategy

### By Use Case

```bash
# Development & programming
docker exec ollama ollama pull qwen2.5-coder:7b
docker exec ollama ollama pull deepseek-coder:6.7b

# Research & analysis
docker exec ollama ollama pull deepseek-r1:8b
docker exec ollama ollama pull qwen2.5:14b
```

### By Hardware Constraints

```bash
# 8GB RAM — conservative
docker exec ollama ollama pull llama3.2:3b
docker exec ollama ollama pull qwen2.5:3b

# 16GB RAM — balanced
docker exec ollama ollama pull llama3.1:8b
docker exec ollama ollama pull qwen2.5-coder:7b

# 32GB+ RAM — full capability
docker exec ollama ollama pull llama3.1:70b
docker exec ollama ollama pull deepseek-r1:8b
```

## Model Management

```bash
# List installed models
docker exec ollama ollama list

# Show model details
docker exec ollama ollama show llama3.1:8b

# Remove a model
docker exec ollama ollama rm old-model:7b

# Re-pull (refresh) a model
docker exec ollama ollama pull llama3.1:8b
```

Model files live in the `${RESOURCE_DATA_DIR}` → `/root/.ollama` bind-mount and
persist across container restarts.

## Model Performance Guide

### Inference Speed (tokens/second)

| Model Size | CPU (16 cores) | RTX 4090 | RTX 3080 |
|------------|----------------|----------|----------|
| 3B models  | 15-25 tok/s    | 80-120 tok/s | 60-90 tok/s |
| 7-8B models| 5-12 tok/s     | 40-70 tok/s  | 25-45 tok/s |
| 13-14B models| 2-6 tok/s    | 20-35 tok/s  | 12-25 tok/s |
| 70B models | 0.5-2 tok/s    | 8-15 tok/s   | 4-8 tok/s |

### Quality vs Speed

- **Fastest**: llama3.2:3b, qwen2.5:3b, gemma2:2b
- **Balanced**: llama3.1:8b, qwen2.5-coder:7b, deepseek-r1:8b
- **Highest Quality**: llama3.1:70b, qwen2.5:72b

## Advanced Configuration

### Memory / concurrency

Concurrency and loaded-model limits are declared in
[`resource.json`](../resource.json) under `runtime.env` (`OLLAMA_NUM_PARALLEL`,
`OLLAMA_MAX_LOADED_MODELS`). The container's hard memory ceiling is
`runtime.memory_limit: 12g`. Edit those and `vrooli resource restart ollama`.

### Custom Models

```bash
# Create a custom model from a Modelfile placed in the data volume
docker exec ollama ollama create my-assistant -f /root/.ollama/Modelfile
```

## Next Steps

- [Installation Guide](INSTALLATION.md) — setup and configuration
- [Embedding Models](EMBEDDING_MODELS.md) — embedding model guidance
- [Resource docs](README.md) — current usage guidance
