# QwenCoder Resource

Alibaba's code-specialized LLM for advanced code generation, completion, review, and bug fixing.

## Overview

QwenCoder provides state-of-the-art code generation capabilities with models ranging from 0.5B to 32B parameters. It supports 92+ programming languages with an exceptional 256K token context window (expandable to 1M), achieving 85%+ accuracy on HumanEval benchmarks.

## Features

- **Multi-model support**: 0.5B to 32B parameter models
- **Large context**: 256K tokens (expandable to 1M)
- **92+ languages**: Comprehensive programming language support
- **Fill-in-the-middle**: Advanced code completion at cursor
- **Function calling**: Native tool integration support
- **Apache 2.0 license**: Commercial deployment ready

## Quick Start

```bash
# Install the resource
vrooli resource qwencoder manage install

# Start QwenCoder service
vrooli resource qwencoder manage start

# Check health status
vrooli resource qwencoder status

# Run a code completion
vrooli resource qwencoder content execute --prompt "def fibonacci(n):" --language python
```

## Configuration

### Environment Variables
```bash
QWENCODER_PORT=11452              # API port (default: 11452)
QWENCODER_MODEL=qwencoder-1.5b    # Default model
QWENCODER_DEVICE=auto             # Device: auto, cpu, cuda
QWENCODER_MAX_MEMORY=16GB         # Memory limit
```

### Model Selection
- **qwencoder-0.5b**: Lightweight completions (2GB RAM)
- **qwencoder-1.5b**: Standard development (4GB RAM)
- **qwencoder-7b**: Complex generation (16GB RAM)
- **qwencoder-14b**: Enterprise features (32GB RAM)
- **qwencoder-32b**: Large codebase analysis (64GB RAM)

## API Usage

### Code Completion
```bash
curl -X POST http://localhost:11452/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwencoder-1.5b",
    "prompt": "def sort_array(arr):",
    "max_tokens": 150,
    "language": "python"
  }'
```

### Chat with Function Calling
```bash
curl -X POST http://localhost:11452/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwencoder-1.5b",
    "messages": [{"role": "user", "content": "Write a function to calculate factorial"}],
    "functions": [{
      "name": "calculate",
      "parameters": {"type": "object", "properties": {"n": {"type": "integer"}}}
    }]
  }'
```

## Testing

```bash
# Run smoke tests (quick validation)
vrooli resource qwencoder test smoke

# Run integration tests
vrooli resource qwencoder test integration

# Run all tests
vrooli resource qwencoder test all
```

## Integration Examples

### With Code Review Scenarios
```python
from qwencoder import QwenCoderClient

client = QwenCoderClient(base_url="http://localhost:11452")

# Review code for issues
review = client.review(
    code=pull_request_diff,
    focus=["security", "performance", "style"]
)
```

### With Test Generation
```python
# Generate unit tests
tests = client.generate_tests(
    function_code=my_function,
    framework="pytest",
    coverage_target=0.8
)
```

## Troubleshooting

### Model Download Issues
```bash
# Check download status
vrooli resource qwencoder content list

# Retry download with specific model
vrooli resource qwencoder content add --model qwencoder-1.5b
```

### Memory Issues
```bash
# Use smaller model
export QWENCODER_MODEL=qwencoder-0.5b

# Enable quantization
export QWENCODER_QUANTIZE=8bit
```

### Performance Optimization
```bash
# Enable GPU if available
export QWENCODER_DEVICE=cuda

# Adjust batch size
export QWENCODER_BATCH_SIZE=4
```

## Dependencies

- Python 3.10+
- transformers >= 4.38.0
- torch >= 2.1.0
- FastAPI >= 0.109.0
- Optional: CUDA 11.8+ for GPU acceleration

## License

Apache 2.0 - Commercial use permitted

## Support

For issues or questions:
- Check logs: `vrooli resource qwencoder logs`
- Run diagnostics: `vrooli resource qwencoder test smoke`
- Review PRD: `/resources/qwencoder/PRD.md`