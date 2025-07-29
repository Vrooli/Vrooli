# Ollama - Local LLM Inference

Run large language models locally with Ollama. Perfect for privacy-sensitive tasks, offline environments, and cost-effective AI inference.

## 🚀 Quick Start

```bash
# Install with default models (llama3.1:8b, deepseek-r1:8b, qwen2.5-coder:7b)
./manage.sh --action install

# Send a prompt
./manage.sh --action prompt --text "Explain machine learning in simple terms"

# Check status and available models
./manage.sh --action status
```

**Default URL**: http://localhost:11434

## 📚 Documentation

For comprehensive guides and advanced configuration:

- **[📖 Full Documentation](docs/README.md)** - Complete guide with all topics
- **[⚡ Installation Guide](docs/INSTALLATION.md)** - Setup and model installation
- **[🤖 Models Guide](docs/MODELS.md)** - Available models and selection
- **[⚙️ Configuration](docs/CONFIGURATION.md)** - Advanced settings and tuning
- **[💻 API Reference](docs/API.md)** - REST endpoints and examples

## 🎯 Key Features

- **Privacy-first**: All inference runs locally
- **Multiple models**: 50+ available models for different tasks
- **GPU acceleration**: NVIDIA GPU support for faster inference
- **Type-aware**: Automatic model selection based on task type
- **Resource efficient**: Smart memory management and model loading

## 🔧 Common Commands

```bash
# Model management
./manage.sh --action models                         # List available models
./manage.sh --action pull --models "llama3.1:70b"  # Download specific model

# Text generation with automatic model selection
./manage.sh --action prompt --text "Write a Python function" --type code
./manage.sh --action prompt --text "Solve: x² + 5x = 0" --type reasoning
./manage.sh --action prompt --text "Hello, how are you?" --type general

# Advanced generation with custom parameters
./manage.sh --action prompt \
  --text "Explain quantum computing" \
  --model "llama3.1:8b" \
  --temperature 0.3 \
  --max-tokens 500
```

## 📋 System Requirements

- **Minimum**: 8GB RAM, 4GB disk space
- **Recommended**: 16GB+ RAM, 50GB+ disk space  
- **GPU**: NVIDIA with 8GB+ VRAM (optional but recommended)
- **Dependencies**: Docker, NVIDIA Container Toolkit (for GPU)

## 🤖 Default Models

| Model | Size | Best For | Download |
|-------|------|----------|----------|
| llama3.1:8b | 4.7GB | General chat, Q&A | Auto |
| deepseek-r1:8b | 4.9GB | Reasoning, math | Auto |
| qwen2.5-coder:7b | 4.2GB | Code generation | Auto |

See the [Models Guide](docs/MODELS.md) for 50+ additional models.

## 🔗 Quick Links

- **Web UI**: http://localhost:11434
- **Health Check**: http://localhost:11434/api/tags
- **Model Examples**: [examples/](examples/README.md)
- **Official Ollama**: https://ollama.ai

---

**Need help?** Check the [troubleshooting guide](docs/TROUBLESHOOTING.md) or explore [performance tuning](docs/PERFORMANCE.md).