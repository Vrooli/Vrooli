# Segment Anything Resource

Foundation segmentation service providing Meta's Segment Anything Model (SAM2/HQ-SAM) with zero-shot object segmentation capabilities for images and videos.

## Overview

The Segment Anything resource bundles SAM2 and HQ-SAM variants with ONNX/PyTorch runtimes, providing universal segmentation primitives for computer vision workflows. It enables prompt-based segmentation with minimal user input, supporting point, box, and mask prompts.

## Features

- **Multiple Model Variants**: SAM2 (Tiny/Small/Base/Large) and HQ-SAM
- **Flexible Prompting**: Point, box, mask, and automatic segmentation
- **Multiple Export Formats**: PNG, COCO RLE, GeoJSON, binary arrays
- **GPU/CPU Support**: CUDA acceleration with CPU fallback
- **Caching & Storage**: Redis caching, MinIO storage, Postgres metadata
- **REST API**: FastAPI-based service on port 11454

## Quick Start

```bash
# Install the resource (builds Docker image)
vrooli resource segment-anything manage install

# Start the service
vrooli resource segment-anything manage start --wait

# Check health
vrooli resource segment-anything test smoke

# Test the API (after service starts)
python resources/segment-anything/examples/test_segmentation.py

# Run segmentation
vrooli resource segment-anything content execute \
  --image /path/to/image.jpg \
  --prompt "point:320,240" \
  --format png
```

**Note**: First startup downloads model weights (~46MB for default). GPU support requires NVIDIA Docker runtime.

## Installation

### Prerequisites

- Docker installed and running
- 10GB disk space for models
- 4GB RAM minimum (8GB recommended)
- NVIDIA GPU with CUDA 11.8+ (optional)

### Install Command

```bash
vrooli resource segment-anything manage install
```

This will:
1. Build the Docker container
2. Download the default model (base)
3. Create required directories
4. Validate the installation

## Configuration

### Environment Variables

```bash
# Model selection
export SEGMENT_ANYTHING_MODEL_SIZE=base  # tiny, small, base, large
export SEGMENT_ANYTHING_MODEL_TYPE=sam2   # sam2, hq-sam
export SEGMENT_ANYTHING_DEVICE=auto       # auto, cpu, cuda

# Performance
export SEGMENT_ANYTHING_MAX_WORKERS=4
export SEGMENT_ANYTHING_BATCH_SIZE=1
export SEGMENT_ANYTHING_GPU_MEMORY_FRACTION=0.8

# Integrations (optional)
export SEGMENT_ANYTHING_REDIS_HOST=localhost
export SEGMENT_ANYTHING_REDIS_PORT=6380
export SEGMENT_ANYTHING_MINIO_ENDPOINT=localhost:9000
```

### Model Variants

| Model | Parameters | Speed | Quality | Memory |
|-------|------------|-------|---------|---------|
| tiny | 39M | Fastest | Good | 2GB |
| small | 46M | Fast | Better | 3GB |
| base | 91M | Balanced | Best | 4GB |
| large | 308M | Slower | Excellent | 8GB |

## API Endpoints

### Health Check
```bash
GET http://localhost:11454/health
```

### List Models
```bash
GET http://localhost:11454/api/v1/models
```

### Segment Image
```bash
POST http://localhost:11454/api/v1/segment
Content-Type: multipart/form-data

image: <binary>
prompt: "point:100,200" | "box:50,50,200,200" | "auto"
format: "png" | "coco" | "geojson"
```

## Usage Examples

### Point-based Segmentation
```bash
# Click point at coordinates (320, 240)
vrooli resource segment-anything content execute \
  --image photo.jpg \
  --prompt "point:320,240"
```

### Box-based Segmentation
```bash
# Select region with bounding box
vrooli resource segment-anything content execute \
  --image photo.jpg \
  --prompt "box:100,100,400,300"
```

### Automatic Segmentation
```bash
# Segment all objects automatically
vrooli resource segment-anything content execute \
  --image photo.jpg \
  --prompt "auto"
```

### Export to Different Formats
```bash
# Export as GeoJSON for GIS applications
vrooli resource segment-anything content execute \
  --image aerial.jpg \
  --prompt "auto" \
  --format geojson > segments.json
```

## Integration Examples

### With N8n Workflow
```javascript
// N8n HTTP Request node configuration
{
  "method": "POST",
  "url": "http://localhost:11454/api/v1/segment",
  "sendBody": true,
  "bodyContentType": "multipart-form-data",
  "body": {
    "image": "={{$binary.image}}",
    "prompt": "auto"
  }
}
```

### With Python
```python
import requests

# Segment an image
with open('image.jpg', 'rb') as f:
    response = requests.post(
        'http://localhost:11454/api/v1/segment',
        files={'image': f},
        data={'prompt': 'point:100,200', 'format': 'coco'}
    )
    masks = response.json()['masks']
```

### With MinIO Storage
```bash
# Configure MinIO integration
export SEGMENT_ANYTHING_MINIO_ENDPOINT=localhost:9000

# Segmentation results automatically stored in MinIO
vrooli resource segment-anything content execute \
  --image large_image.jpg \
  --prompt "auto" \
  --store-minio
```

## Testing

```bash
# Quick health check (<30s)
vrooli resource segment-anything test smoke

# Full functionality test (<120s)
vrooli resource segment-anything test integration

# All tests
vrooli resource segment-anything test all
```

## Performance Optimization

### GPU Acceleration
```bash
# Ensure CUDA is available
nvidia-smi

# Force GPU usage
export SEGMENT_ANYTHING_DEVICE=cuda
```

### Batch Processing
```bash
# Process multiple images
export SEGMENT_ANYTHING_BATCH_SIZE=4
```

### Enable Torch Compilation
```bash
# Faster inference after warmup
export SEGMENT_ANYTHING_ENABLE_COMPILE=true
```

## Troubleshooting

### Service Won't Start
```bash
# Check Docker
docker ps
docker logs vrooli-segment-anything

# Verify port availability
lsof -i :11454
```

### Out of Memory
```bash
# Use smaller model
export SEGMENT_ANYTHING_MODEL_SIZE=tiny

# Reduce GPU memory
export SEGMENT_ANYTHING_GPU_MEMORY_FRACTION=0.5
```

### Slow Performance
```bash
# Check if using CPU
vrooli resource segment-anything status

# Enable GPU if available
export SEGMENT_ANYTHING_DEVICE=cuda
```

## Advanced Usage

### Custom Model Loading
```bash
# Download specific model
vrooli resource segment-anything content add sam2-large

# Use custom checkpoint
docker cp custom_model.pth vrooli-segment-anything:/app/models/
```

### Integration with Blender
```python
# Export masks for 3D reconstruction
masks = segment_image('render.png')
import_to_blender(masks, format='mesh')
```

### Geospatial Analysis
```bash
# Segment satellite imagery
vrooli resource segment-anything content execute \
  --image satellite.tif \
  --prompt "auto" \
  --format geojson | \
  ogr2ogr output.shp /vsistdin/
```

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Client    │────▶│  FastAPI     │────▶│    SAM2     │
│  (CLI/API)  │     │   Server     │     │   Models    │
└─────────────┘     └──────────────┘     └─────────────┘
                           │                      │
                           ▼                      ▼
                    ┌──────────────┐     ┌─────────────┐
                    │    Redis     │     │   MinIO     │
                    │    Cache     │     │   Storage   │
                    └──────────────┘     └─────────────┘
```

## Resource Contract

This resource follows the Vrooli v2.0 universal contract:
- **Port**: 11454 (AI services range)
- **Health**: Responds in <5s
- **Startup**: ~30-60s for model loading
- **Category**: AI/Computer Vision
- **Dependencies**: None required, Redis/MinIO optional

## Future Enhancements

- Real-time video stream processing
- 3D point cloud segmentation
- Text-guided segmentation (GroundingDINO)
- Mobile/edge deployment
- Fine-tuning interface

## Support

For issues or questions:
- Check logs: `vrooli resource segment-anything logs`
- Run diagnostics: `vrooli resource segment-anything test all`
- See PRD: `resources/segment-anything/PRD.md`

## License

This resource wraps Meta's Segment Anything Model, which is licensed under Apache 2.0.