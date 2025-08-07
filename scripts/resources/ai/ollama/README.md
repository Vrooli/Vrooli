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
- **[🎨 Model Customization](#-model-customization-with-modelfiles)** - Create specialized models with Modelfiles
- **[⚙️ Configuration](docs/CONFIGURATION.md)** - Advanced settings and tuning
- **[💻 API Reference](docs/API.md)** - REST endpoints and examples

## 🎯 Key Features

- **Privacy-first**: All inference runs locally
- **Multiple models**: 50+ available models for different tasks
- **Model customization**: Create specialized models via Modelfiles
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

# Model customization (create specialized models)
ollama create my-specialist -f my-modelfile     # Create custom model
ollama list                                     # List all models (including custom)
ollama show my-specialist                       # View model configuration
ollama rm my-specialist                         # Delete custom model
```

## 🎨 Model Customization with Modelfiles

Create specialized AI models without training! Ollama's Modelfile feature lets you customize any base model with domain-specific behavior, custom system prompts, and specialized parameters.

### 🚀 Quick Example: Real Estate Chatbot

```bash
# 1. Create a Modelfile
cat > /tmp/real-estate-specialist << 'EOF'
FROM llama3.1:8b

SYSTEM """You are an expert real estate chatbot assistant. Your role is to help clients with property inquiries, schedule viewings, and provide comprehensive real estate information.

Key responsibilities:
- Answer property-related questions with accurate, helpful information
- Assist with scheduling property viewings and appointments
- Provide market insights and neighborhood information
- Help qualify leads by understanding client needs and budget
- Maintain a professional, friendly, and knowledgeable tone

When a client inquires about properties, always gather:
- Budget range and preferred location
- Property type and bedroom/bathroom requirements
- Special requirements (pet-friendly, parking, etc.)
- Timeframe for purchasing/renting
"""

PARAMETER temperature 0.7
PARAMETER top_p 0.9
EOF

# 2. Create the specialized model (use CLI - more reliable than API)
ollama create real-estate-specialist -f /tmp/real-estate-specialist

# 3. Use your specialized model
curl -X POST http://localhost:11434/api/generate -d '{
  "model": "real-estate-specialist",
  "prompt": "I need a 3-bedroom house under $500k",
  "stream": false
}'
```

### 🔧 Modelfile Components

```dockerfile
FROM llama3.1:8b                    # Base model to customize

SYSTEM """Your specialized behavior and instructions"""

PARAMETER temperature 0.7           # Creativity (0.0-1.0)
PARAMETER top_p 0.9                 # Response diversity
PARAMETER top_k 40                  # Vocabulary restriction
PARAMETER repeat_penalty 1.1       # Reduce repetition

TEMPLATE """{{ .System }}{{ .Prompt }}"""  # Custom formatting (optional)
```

### 💡 Common Use Cases

| Specialization | Modelfile Focus | Example System Prompt |
|---------------|-----------------|----------------------|
| **Customer Support** | Helpful, policy-aware | "You are a customer service expert for [Company]. Always be helpful and follow company policies..." |
| **Code Review** | Technical, detailed | "You are a senior software engineer. Review code for bugs, performance, and best practices..." |
| **Legal Assistant** | Precise, cautious | "You are a legal research assistant. Provide accurate information and always recommend consulting a lawyer..." |
| **Medical Info** | Careful, factual | "You are a medical information assistant. Provide general health information but always recommend consulting healthcare professionals..." |

### ⚠️ CLI vs API: Important Differences

**✅ Use Ollama CLI for model creation:**
```bash
ollama create my-model -f modelfile    # ✅ Reliable, clear errors
ollama list                            # ✅ View all models
ollama rm my-model                     # ✅ Delete models
```

**⚠️ API has limitations for model creation:**
```bash
curl -X POST /api/create ...           # ❌ Complex, undocumented requirements
```

**✅ Use REST API for inference:**
```bash
curl -X POST /api/generate ...         # ✅ Perfect for applications
curl -X POST /api/chat ...             # ✅ Conversation interface
```

### 🎯 Pro Tips

1. **Start Simple**: Begin with just a SYSTEM prompt, add parameters later
2. **Test Iteratively**: Create, test, modify, recreate until perfect
3. **Use Examples**: Include conversation examples in your SYSTEM prompt
4. **Parameter Tuning**: Lower temperature (≤0.3) for factual tasks, higher (0.7-0.9) for creative tasks
5. **Version Control**: Use descriptive model names like `customer-support-v2`

### 🔄 Model Management

```bash
# List all your custom models
ollama list

# Copy a model (for versioning)
ollama cp real-estate-specialist real-estate-v2

# Show model configuration
ollama show real-estate-specialist

# Delete old versions
ollama rm real-estate-v1
```

**Key Insight**: This approach often outperforms traditional fine-tuning for business applications - it's faster, more controllable, and requires no training data!

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

## 🧪 Testing & Examples

### Individual Resource Tests
- **Test Location**: `scripts/__test/resources/single/ai/ollama.test.sh`
- **Test Coverage**: Health checks, model listing, text generation, API functionality
- **Run Test**: `cd scripts/__test/resources && ./quick-test.sh ollama`

### Working Examples
- **Examples Folder**: [examples/](examples/)
- **Basic Usage**: Simple text generation and API calls
- **Integration Examples**: Multi-resource workflows combining Ollama with other services
- **Model Management**: Creating and using custom specialized models

### Integration with Scenarios
Ollama is used in these business scenarios:
- **[Multi-Modal AI Assistant](../../scenarios/core/multi-modal-ai-assistant/)** - Voice + AI + visual workflows ($10k-25k projects)
- **[Research Assistant](../../scenarios/core/research-assistant/)** - Knowledge management and analysis ($10k-25k projects)  
- **[Campaign Content Studio](../../scenarios/core/campaign-content-studio/)** - Content creation workflows ($8k-20k projects)
- **[Secure Document Processing](../../scenarios/core/secure-document-processing/)** - Compliant document processing ($20k-40k projects)

### Test Fixtures
- **Shared Test Data**: `scripts/__test/resources/fixtures/documents/` (prompts, text samples)
- **Audio Fixtures**: `scripts/__test/resources/fixtures/audio/` (for multi-modal scenarios)

### Quick Test Commands
```bash
# Test individual Ollama functionality
./scripts/__test/resources/quick-test.sh ollama

# Test in real business scenarios
cd ./scripts/scenarios/core/research-assistant && ./test.sh

# Run all tests using Ollama
./scripts/scenarios/tools/test-by-resource.sh --resource ollama
```

## 🔗 Quick Links

- **Web UI**: http://localhost:11434
- **Health Check**: http://localhost:11434/api/tags
- **Model Examples**: [examples/](examples/README.md)
- **Official Ollama**: https://ollama.ai

---

**Need help?** Check the [troubleshooting guide](docs/TROUBLESHOOTING.md) or explore [performance tuning](docs/PERFORMANCE.md).