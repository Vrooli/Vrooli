# Ollama Documentation

Comprehensive documentation for the Ollama local LLM inference resource.

## 📚 Documentation Structure

### Getting Started
- [Installation Guide](INSTALLATION.md) - Setup and model installation
- [Configuration](CONFIGURATION.md) - Environment variables and settings
- [Models Guide](MODELS.md) - Available models and selection

### Usage & API
- [API Reference](API.md) - REST endpoints and examples
- [Command Line Usage](CLI.md) - Using the manage.sh script
- [Integration Examples](../examples/README.md) - Code examples and workflows

### Advanced Topics
- [Performance Tuning](PERFORMANCE.md) - Optimization and hardware requirements
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions
- [Development](DEVELOPMENT.md) - Contributing and extending

## 🚀 Quick Start

```bash
# Install Ollama with default models
./manage.sh --action install

# Check status and models
./manage.sh --action status

# Send a prompt
./manage.sh --action prompt --text "Explain machine learning"
```

## 📋 Quick Reference

### Essential Commands
```bash
# Model management
./manage.sh --action models                    # List available models
./manage.sh --action pull --models "llama3.1:8b"  # Download specific model

# Text generation
./manage.sh --action prompt --text "Your question" --type code
./manage.sh --action prompt --model "deepseek-r1:8b" --type reasoning

# System management
./manage.sh --action start|stop|restart|status
```

### Key Endpoints
- **Health**: http://localhost:11434/api/tags
- **Generate**: http://localhost:11434/api/generate
- **Models**: http://localhost:11434/api/tags

## 🔗 Quick Links

- **Default URL**: http://localhost:11434
- **Model Recommendations**: [Models Guide](MODELS.md#default-models)
- **API Examples**: [API Reference](API.md#examples)
- **Official Ollama Docs**: https://ollama.ai/docs