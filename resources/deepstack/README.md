# DeepStack Computer Vision Resource

Cross-platform AI engine providing pre-built computer vision models via REST API.

## Overview

DeepStack is a comprehensive computer vision platform that provides:
- **Object Detection**: Detect 80+ object classes with bounding boxes (COCO dataset)
- **Face Detection & Recognition**: Detect faces, extract landmarks, and perform recognition
- **Scene Classification**: Classify environments and locations (Places365 dataset)
- **Custom Models**: Load and use your own trained models
- **GPU Acceleration**: Automatic GPU detection with CPU fallback
- **Redis Caching**: Optional result caching for improved performance

## Quick Start

```bash
# Install the resource
resource-deepstack manage install

# Start the service
resource-deepstack manage start --wait

# Check status
resource-deepstack status

# Detect objects in an image
resource-deepstack content execute --file image.jpg --type object

# View help
resource-deepstack help
```

## Installation

### Prerequisites
- Docker installed and running
- 10GB+ free disk space
- (Optional) NVIDIA GPU with CUDA 11.0+ for acceleration
- (Optional) Redis for caching

### Install Command
```bash
resource-deepstack manage install
```

This will:
1. Pull the DeepStack Docker image
2. Create required directories
3. Set up Docker network
4. Detect GPU availability
5. Configure the service

## Configuration

### Environment Variables
```bash
# Service configuration
DEEPSTACK_PORT=11453              # API port
DEEPSTACK_HOST=127.0.0.1         # Host to bind to
DEEPSTACK_API_KEY=                # Optional API key

# Performance
DEEPSTACK_MODE=High               # High, Medium, Low
DEEPSTACK_ENABLE_GPU=auto         # auto, true, false
DEEPSTACK_THREADS=4               # Processing threads
DEEPSTACK_CONFIDENCE_THRESHOLD=0.45  # Min confidence

# Redis caching (optional)
DEEPSTACK_REDIS_ENABLED=false    # Enable caching
DEEPSTACK_REDIS_HOST=127.0.0.1   # Redis host
DEEPSTACK_REDIS_PORT=6380        # Redis port
```

### Configuration Files
- `config/defaults.sh` - Default environment variables
- `config/runtime.json` - Runtime behavior and dependencies
- `config/schema.json` - Configuration schema

## Usage Examples

### Object Detection
```bash
# Detect objects in an image
resource-deepstack content execute --file car.jpg --type object

# Response includes:
# - Detected objects with labels
# - Bounding box coordinates
# - Confidence scores
```

### Face Detection
```bash
# Detect faces in an image
resource-deepstack content execute --file group.jpg --type face

# Register a face for recognition
resource-deepstack content add --type face --name "John" --file john.jpg

# Recognize faces
resource-deepstack content execute --file unknown.jpg --type face-recognize
```

### Scene Classification
```bash
# Classify scene/environment
resource-deepstack content execute --file landscape.jpg --type scene

# Returns scene type (indoor/outdoor, specific locations)
```

## API Endpoints

### Core Detection
- `POST /v1/vision/detection` - Object detection
- `POST /v1/vision/face` - Face detection
- `POST /v1/vision/face/register` - Register face
- `POST /v1/vision/face/recognize` - Face recognition
- `POST /v1/vision/scene` - Scene classification

### System
- `GET /health` - Service health check
- `GET /v1/vision/status` - Detailed status

### Example API Call
```bash
curl -X POST "http://localhost:11453/v1/vision/detection" \
  -F "image=@image.jpg" \
  -F "min_confidence=0.45"
```

## Testing

```bash
# Run all tests
resource-deepstack test all

# Quick health check (<30s)
resource-deepstack test smoke

# Full integration tests (<120s)
resource-deepstack test integration

# Unit tests (<60s)
resource-deepstack test unit
```

## Performance

### CPU Mode
- Object Detection: ~500ms per image
- Face Recognition: ~200ms per face
- Scene Classification: ~300ms per image

### GPU Mode (NVIDIA)
- Object Detection: ~100ms per image
- Face Recognition: ~50ms per face
- Scene Classification: ~80ms per image

### Optimization Tips
1. Enable Redis caching for repeated detections
2. Use GPU acceleration when available
3. Adjust confidence thresholds based on use case
4. Use batch processing for multiple images
5. Configure appropriate thread count

## Integration with Vrooli

### Direct Integrations
- **Image Tools**: Smart cropping, object-aware editing
- **Smart File Manager**: Auto-categorization, face grouping
- **Security Monitoring**: Intrusion detection, tracking
- **Content Moderation**: Combined with NSFW detector

### Workflow Integration
```javascript
// N8n workflow node example
{
  "name": "DeepStack Detection",
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "http://localhost:11453/v1/vision/detection",
    "method": "POST",
    "bodyParameters": {
      "image": "={{$binary.image}}",
      "min_confidence": 0.45
    }
  }
}
```

## Troubleshooting

### Service Won't Start
```bash
# Check Docker status
docker ps -a | grep deepstack

# View logs
resource-deepstack logs --tail 100

# Check port availability
netstat -tlnp | grep 11453
```

### Slow Performance
1. Check if GPU is detected: `resource-deepstack status`
2. Verify Redis connection if caching enabled
3. Adjust performance mode: `DEEPSTACK_MODE=High`
4. Increase thread count: `DEEPSTACK_THREADS=8`

### Detection Issues
1. Lower confidence threshold for more detections
2. Check image format (JPEG/PNG supported)
3. Verify image size (<10MB default)
4. Ensure adequate lighting in images

## Models

### Pre-trained Models
- **YOLOv5**: Fast object detection (80 classes)
- **RetinaFace**: Accurate face detection with landmarks
- **ArcFace**: Face recognition with 99.5% accuracy
- **Places365**: Scene classification (365 categories)

### Custom Models
Support for custom PyTorch models coming soon.

## Hardware Requirements

### Minimum (CPU Only)
- CPU: 4 cores
- RAM: 8GB
- Storage: 10GB

### Recommended (GPU)
- CPU: 8 cores
- RAM: 16GB
- GPU: NVIDIA GTX 1060 6GB+
- Storage: 20GB

### Production
- CPU: 16 cores
- RAM: 32GB
- GPU: NVIDIA RTX 3080 10GB+
- Storage: 50GB SSD

## Security

- Input validation on all uploads
- Configurable size limits
- Optional API key authentication
- No image storage by default
- Runs in isolated Docker container

## Support

For issues or questions:
1. Check the troubleshooting section
2. View logs: `resource-deepstack logs`
3. Run diagnostics: `resource-deepstack test smoke`
4. See PRD.md for detailed requirements

## License

DeepStack is provided by DeepQuest AI. This resource wrapper follows Vrooli's v2.0 contract specification.