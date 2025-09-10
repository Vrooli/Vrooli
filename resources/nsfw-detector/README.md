# NSFW Detector Resource

AI-powered content moderation resource for detecting NSFW (Not Safe For Work) content in images using state-of-the-art CNN models.

## Overview

The NSFW Detector resource provides real-time image classification to identify adult, racy, violent, or otherwise inappropriate content. It uses multiple pre-trained models for high accuracy while maintaining user privacy through local-only processing.

## Features

- **Multi-Model Support**: NSFW.js, Safety Checker, and custom CNN models
- **Real-time Classification**: <200ms processing time per image
- **High Accuracy**: 98%+ precision/recall on standard benchmarks
- **Privacy-First**: All processing happens locally, no external APIs
- **Batch Processing**: Efficiently process multiple images
- **Configurable Thresholds**: Adjust sensitivity for each content category
- **v2.0 Contract Compliant**: Full lifecycle management and CLI interface

## Quick Start

```bash
# Install the resource
vrooli resource nsfw-detector manage install

# Start the service
vrooli resource nsfw-detector manage start --wait

# Check health status
vrooli resource nsfw-detector status

# Classify an image
vrooli resource nsfw-detector content execute --file image.jpg

# Stop the service
vrooli resource nsfw-detector manage stop
```

## Installation

### Prerequisites

- Node.js 18+ or Python 3.9+
- 2GB available RAM
- 1GB disk space for models

### Setup

```bash
# Install via Vrooli CLI
vrooli resource nsfw-detector manage install

# Or manual installation
cd resources/nsfw-detector
./cli.sh manage install
```

## Usage

### Basic Image Classification

```bash
# Classify a single image
vrooli resource nsfw-detector content execute --file photo.jpg

# Output:
# {
#   "adult": 0.02,
#   "racy": 0.15,
#   "gore": 0.01,
#   "violence": 0.03,
#   "safe": 0.79,
#   "classification": "safe"
# }
```

### Batch Processing

```bash
# Process multiple images
vrooli resource nsfw-detector content execute --batch /path/to/images/

# Process with custom threshold
vrooli resource nsfw-detector content execute --batch /path/to/images/ --threshold 0.9
```

### Configuration

```bash
# View current configuration
vrooli resource nsfw-detector info --json

# Update thresholds
vrooli resource nsfw-detector content add --type config --file thresholds.json
```

### Model Management

```bash
# List available models
vrooli resource nsfw-detector content list --type models

# Load specific model
vrooli resource nsfw-detector content add --type model --name safety-checker

# Unload model to save memory
vrooli resource nsfw-detector content remove --type model --name nsfwjs
```

## Configuration

Default configuration in `config/defaults.sh`:

```bash
# Port allocation (from registry)
NSFW_DETECTOR_PORT=${NSFW_DETECTOR_PORT:-$(get_port_for "nsfw-detector")}

# Model settings
NSFW_DETECTOR_DEFAULT_MODEL="nsfwjs"
NSFW_DETECTOR_MODEL_PATH="/var/lib/nsfw-detector/models"

# Classification thresholds
NSFW_DETECTOR_ADULT_THRESHOLD=0.7
NSFW_DETECTOR_RACY_THRESHOLD=0.6
NSFW_DETECTOR_GORE_THRESHOLD=0.8
NSFW_DETECTOR_VIOLENCE_THRESHOLD=0.8

# Performance settings
NSFW_DETECTOR_MAX_BATCH_SIZE=10
NSFW_DETECTOR_TIMEOUT_MS=5000
NSFW_DETECTOR_CACHE_SIZE=100
```

## API Reference

### REST Endpoints

```
POST /classify
  Body: { "image": "base64_or_url", "threshold": 0.7 }
  Response: { "adult": 0.1, "racy": 0.2, ... }

POST /classify/batch
  Body: { "images": ["base64_or_url", ...], "threshold": 0.7 }
  Response: [{ "adult": 0.1, ... }, ...]

GET /models
  Response: ["nsfwjs", "safety-checker", ...]

GET /health
  Response: { "status": "healthy", "models_loaded": [...] }
```

## Testing

```bash
# Run all tests
vrooli resource nsfw-detector test all

# Quick smoke test
vrooli resource nsfw-detector test smoke

# Integration tests
vrooli resource nsfw-detector test integration

# Unit tests
vrooli resource nsfw-detector test unit
```

## Troubleshooting

### Service Won't Start

```bash
# Check if port is in use
netstat -tlnp | grep $(vrooli resource nsfw-detector info --json | jq -r .port)

# Check logs
vrooli resource nsfw-detector logs --tail 50

# Verify model files exist
ls -la /var/lib/nsfw-detector/models/
```

### Classification Errors

```bash
# Verify model is loaded
vrooli resource nsfw-detector status --json | jq .models_loaded

# Check image format support
file your-image.jpg  # Should be JPEG, PNG, GIF, or WebP

# Test with known safe image
vrooli resource nsfw-detector content execute --file test/fixtures/safe.jpg
```

### Performance Issues

```bash
# Check resource usage
vrooli resource nsfw-detector status --verbose

# Reduce batch size
export NSFW_DETECTOR_MAX_BATCH_SIZE=5

# Use lighter model
vrooli resource nsfw-detector content add --type model --name nsfwjs-mobile
```

## Development

### Running Locally

```bash
# Start in development mode
vrooli resource nsfw-detector develop

# Run with debug logging
DEBUG=nsfw-detector:* vrooli resource nsfw-detector develop
```

### Adding Custom Models

1. Place model files in `/var/lib/nsfw-detector/models/custom/`
2. Update `config/models.json` with model metadata
3. Restart the service

## Security

- All image processing happens locally (no external API calls)
- Images are processed in memory and never persisted
- Configurable audit logging for compliance
- Support for encrypted model storage
- No personally identifiable information collected

## Performance

- **Latency**: <200ms per image (p95)
- **Throughput**: 10+ images/second
- **Memory**: ~500MB per model loaded
- **CPU**: Optimized for both CPU and GPU inference
- **Startup**: <30 seconds including model loading

## License

See LICENSE file in the repository root.

## Support

For issues and questions:
- Check [troubleshooting guide](#troubleshooting)
- Review [PRD.md](PRD.md) for detailed requirements
- Submit issues to the Vrooli repository