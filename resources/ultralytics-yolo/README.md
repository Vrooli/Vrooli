# Ultralytics YOLO Vision Intelligence Suite

State-of-the-art YOLOv8 computer vision resource providing real-time object detection, segmentation, classification, and tracking for Vrooli scenarios.

## Features

- **YOLOv8 Models**: Latest YOLO architecture with multiple model sizes (n/s/m/l/x)
- **Multiple Tasks**: Object detection, instance segmentation, image classification, pose estimation
- **GPU Acceleration**: CUDA support with automatic CPU fallback
- **Structured Output**: JSON/CSV formats with bounding boxes, confidence scores, and embeddings
- **Storage Integration**: Automatic storage in Qdrant (vectors), Postgres (metadata), MinIO (artifacts)
- **Real-time Performance**: 30+ FPS on GPU for video streams

## Quick Start

```bash
# Install and start the resource
vrooli resource ultralytics-yolo manage install
vrooli resource ultralytics-yolo manage start --wait

# Run object detection on an image
vrooli resource ultralytics-yolo detect --image /path/to/image.jpg --model yolov8m

# Process a video with segmentation
vrooli resource ultralytics-yolo segment --video /path/to/video.mp4 --output results.json

# List available models
vrooli resource ultralytics-yolo content list

# Check service health
vrooli resource ultralytics-yolo status
```

## Architecture

```
┌─────────────────────────────────────────────┐
│            Ultralytics YOLO                 │
│                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │  YOLOv8  │  │  PyTorch │  │   CUDA   │ │
│  │  Models  │  │  Backend │  │  Support │ │
│  └──────────┘  └──────────┘  └──────────┘ │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │         FastAPI Server               │  │
│  │    /detect  /segment  /classify      │  │
│  └──────────────────────────────────────┘  │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │       Integration Layer              │  │
│  │  Qdrant  PostgreSQL  MinIO  Redis    │  │
│  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

## Configuration

Environment variables:
```bash
YOLO_PORT=11455              # API server port
YOLO_MODEL_PATH=/models      # Model cache directory
YOLO_DEVICE=auto             # auto|cpu|cuda
YOLO_BATCH_SIZE=1            # Batch processing size
YOLO_CONFIDENCE=0.25         # Detection confidence threshold
YOLO_IOU_THRESHOLD=0.45      # NMS IOU threshold
```

## API Reference

### Detection Endpoint
```bash
POST /detect
Content-Type: multipart/form-data

Parameters:
- image: Image file or URL
- model: Model variant (yolov8n/s/m/l/x)
- confidence: Minimum confidence (0.0-1.0)
- save_embeddings: Store in Qdrant (true/false)

Response:
{
  "detections": [
    {
      "class": "person",
      "confidence": 0.92,
      "bbox": [100, 200, 300, 400],
      "embedding": [...]
    }
  ],
  "metadata": {
    "model": "yolov8m",
    "inference_time": 45.2,
    "image_size": [640, 480]
  }
}
```

## Integration Examples

### With N8n Workflow
```javascript
// Trigger pipeline on person detection
{
  "trigger": "webhook",
  "url": "http://localhost:11455/detect",
  "condition": "detections.any(d => d.class === 'person' && d.confidence > 0.8)"
}
```

### With Agent-S2 Browser Automation
```python
# Validate UI elements using YOLO detection
screenshot = await browser.screenshot()
detections = yolo.detect(screenshot, model='yolov8s')
buttons = [d for d in detections if d['class'] == 'button']
```

### With MinIO Storage
```python
# Store detection results
result = yolo.detect(image_path)
minio.put_object('detections', f'{timestamp}.json', json.dumps(result))
```

## Performance Benchmarks

| Model | Size | mAP | FPS (GPU) | FPS (CPU) | Memory |
|-------|------|-----|-----------|-----------|---------|
| YOLOv8n | 6MB | 37.3 | 100+ | 15 | 1GB |
| YOLOv8s | 22MB | 44.9 | 80+ | 10 | 2GB |
| YOLOv8m | 50MB | 50.2 | 60+ | 5 | 3GB |
| YOLOv8l | 84MB | 52.9 | 40+ | 3 | 4GB |
| YOLOv8x | 131MB | 53.9 | 30+ | 2 | 6GB |

## Use Cases

- **Manufacturing QA**: Defect detection on production lines
- **Retail Analytics**: Customer counting and behavior analysis
- **Security**: Intrusion detection and anomaly monitoring
- **Traffic Management**: Vehicle counting and classification
- **Agricultural**: Crop health monitoring and yield estimation

## Troubleshooting

| Issue | Solution |
|-------|----------|
| GPU not detected | Check CUDA installation with `nvidia-smi` |
| Slow inference | Use smaller model (yolov8n) or enable GPU |
| Out of memory | Reduce batch size or use CPU mode |
| Model download fails | Check internet connection and disk space |

## Related Resources

- **segment-anything**: Advanced segmentation for complex scenes
- **deepstack**: Alternative detection models and face recognition
- **vocr**: Extract text from detected objects
- **comfyui**: Generate synthetic training data
- **browserless**: Capture screenshots for UI validation

## Development

```bash
# Run tests
vrooli resource ultralytics-yolo test all

# View logs
vrooli resource ultralytics-yolo logs

# Stop service
vrooli resource ultralytics-yolo manage stop
```

## License

Uses Ultralytics YOLOv8 under AGPL-3.0 license for open-source projects.