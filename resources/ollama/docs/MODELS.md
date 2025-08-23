# Ollama Models Guide

Comprehensive guide to available models and selection strategies.

## Default Models (2025 Recommendations)

These models are installed by default and cover most use cases:

| Model | Size | Best For | Download Size |
|-------|------|----------|---------------|
| **llama3.1:8b** | 8B params | General chat, Q&A | 4.7GB |
| **deepseek-r1:8b** | 8B params | Reasoning, math, analysis | 4.9GB |
| **qwen2.5-coder:7b** | 7B params | Code generation, debugging | 4.2GB |

**Total**: ~14GB storage required

## Model Categories

### General Purpose Models

Best for: Chat, Q&A, general assistance, content creation

```bash
# Recommended general models
./manage.sh --action pull --models "llama3.1:8b"      # Balanced performance
./manage.sh --action pull --models "llama3.1:70b"     # Highest quality (40GB)
./manage.sh --action pull --models "llama3.2:3b"      # Lightweight (2GB)

# Alternative options
./manage.sh --action pull --models "gemma2:9b"        # Google's model
./manage.sh --action pull --models "mistral:7b"       # Fast inference
```

### Code Generation Models

Best for: Programming, code review, debugging, technical documentation

```bash
# Code-specialized models
./manage.sh --action pull --models "qwen2.5-coder:7b"    # Multi-language coding
./manage.sh --action pull --models "codellama:13b"       # Meta's code model
./manage.sh --action pull --models "deepseek-coder:6.7b" # Strong code reasoning

# Instruction-tuned for code
./manage.sh --action pull --models "codegemma:7b"        # Google code model
```

### Reasoning & Math Models

Best for: Logic problems, mathematical reasoning, step-by-step analysis

```bash
# Reasoning specialists
./manage.sh --action pull --models "deepseek-r1:8b"      # Advanced reasoning
./manage.sh --action pull --models "qwen2.5:14b"         # Strong math skills
./manage.sh --action pull --models "llama3.1:70b"        # Best overall reasoning
```

### Vision/Multimodal Models

Best for: Image analysis, visual Q&A, document understanding

```bash
# Vision-capable models
./manage.sh --action pull --models "llava:13b"           # Image + text
./manage.sh --action pull --models "llava-phi3:3.8b"     # Lightweight vision
./manage.sh --action pull --models "bakllava:7b"         # Alternative vision model
```

### Lightweight Models

Best for: Resource-constrained environments, quick responses

```bash
# Small but capable
./manage.sh --action pull --models "llama3.2:3b"         # 2GB, good performance
./manage.sh --action pull --models "qwen2.5:3b"          # Multilingual, compact
./manage.sh --action pull --models "gemma2:2b"           # Ultra-lightweight
```

## Model Selection Strategy

### By Use Case

#### Development & Programming
```bash
# Optimal setup for developers
./manage.sh --action pull --models "qwen2.5-coder:7b,deepseek-coder:6.7b,llama3.1:8b"
```

#### Research & Analysis  
```bash
# Best for analytical work
./manage.sh --action pull --models "deepseek-r1:8b,llama3.1:70b,qwen2.5:14b"
```

#### General Productivity
```bash
# Balanced for everyday use
./manage.sh --action pull --models "llama3.1:8b,qwen2.5-coder:7b,gemma2:9b"
```

### By Hardware Constraints

#### 8GB RAM
```bash
# Conservative selection
./manage.sh --action pull --models "llama3.2:3b,qwen2.5:3b"
```

#### 16GB RAM
```bash
# Balanced selection
./manage.sh --action pull --models "llama3.1:8b,qwen2.5-coder:7b"
```

#### 32GB+ RAM
```bash
# Full capability
./manage.sh --action pull --models "llama3.1:70b,deepseek-r1:8b,qwen2.5-coder:7b"
```

## Model Management

### Installation Commands

```bash
# Install single model
./manage.sh --action pull --models "llama3.1:8b"

# Install multiple models
./manage.sh --action pull --models "llama3.1:8b,deepseek-r1:8b"

# Force re-download
./manage.sh --action pull --models "llama3.1:8b" --force

# Install all default models
./manage.sh --action install-defaults
```

### Model Information

```bash
# List installed models
./manage.sh --action list-installed

# Show model details
./manage.sh --action model-info --model "llama3.1:8b"

# Check disk usage
./manage.sh --action disk-usage

# Available models catalog
./manage.sh --action models
```

### Cleanup and Optimization

```bash
# Remove unused models
./manage.sh --action cleanup-models

# Remove specific model
./manage.sh --action remove --model "old-model:7b"

# Free up space (remove all but defaults)
./manage.sh --action reset-models
```

## Automatic Model Selection

The manage.sh script automatically selects models based on task type:

### Type-Based Selection

```bash
# Automatically chooses best available model for task
./manage.sh --action prompt --text "Fix this Python code" --type code
./manage.sh --action prompt --text "Solve this equation" --type reasoning
./manage.sh --action prompt --text "Explain quantum physics" --type general
```

### Selection Priority

1. **Code tasks**: qwen2.5-coder → deepseek-coder → codellama → general models
2. **Reasoning tasks**: deepseek-r1 → llama3.1:70b → qwen2.5:14b → general models  
3. **General tasks**: llama3.1:8b → gemma2 → mistral → any available
4. **Vision tasks**: llava → bakllava → text-only models

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

### Memory Management

```bash
# Limit models in memory
export OLLAMA_MAX_LOADED_MODELS=2

# Set memory threshold
export OLLAMA_LOW_MEMORY_THRESHOLD=0.8

# Preload specific models
./manage.sh --action preload --models "llama3.1:8b,qwen2.5-coder:7b"
```

### Custom Models

```bash
# Create custom model from Modelfile
./manage.sh --action create-model --name "my-assistant" --modelfile /path/to/Modelfile

# Import GGUF files
./manage.sh --action import-gguf --file /path/to/model.gguf --name "custom-model"
```

## Next Steps

- [Configure model parameters](CONFIGURATION.md#model-configuration)
- [Learn the API](API.md) for programmatic access
- [Optimize performance](PERFORMANCE.md) for your hardware
- [Explore examples](../examples/README.md) with different models