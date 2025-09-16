#!/bin/bash
# Default configuration for Ultralytics YOLO

# Service configuration
export YOLO_PORT="${YOLO_PORT:-11455}"
export YOLO_HOST="${YOLO_HOST:-0.0.0.0}"
export YOLO_WORKERS="${YOLO_WORKERS:-1}"
export YOLO_LOG_LEVEL="${YOLO_LOG_LEVEL:-info}"

# Model configuration
export YOLO_MODEL_PATH="${YOLO_MODEL_PATH:-/models}"
export YOLO_DEFAULT_MODEL="${YOLO_DEFAULT_MODEL:-yolov8m.pt}"
export YOLO_MODEL_VARIANT="${YOLO_MODEL_VARIANT:-yolov8m}"  # n/s/m/l/x variants
export YOLO_TASK="${YOLO_TASK:-detect}"  # detect/segment/classify/pose

# Device configuration
export YOLO_DEVICE="${YOLO_DEVICE:-auto}"  # auto/cpu/cuda/mps
export YOLO_GPU_MEMORY_FRACTION="${YOLO_GPU_MEMORY_FRACTION:-0.8}"
export YOLO_CUDA_DEVICE="${YOLO_CUDA_DEVICE:-0}"  # GPU device index

# Inference configuration
export YOLO_CONFIDENCE="${YOLO_CONFIDENCE:-0.25}"  # Detection confidence threshold
export YOLO_IOU_THRESHOLD="${YOLO_IOU_THRESHOLD:-0.45}"  # NMS IOU threshold
export YOLO_MAX_DETECTIONS="${YOLO_MAX_DETECTIONS:-300}"  # Max detections per image
export YOLO_IMAGE_SIZE="${YOLO_IMAGE_SIZE:-640}"  # Input image size
export YOLO_BATCH_SIZE="${YOLO_BATCH_SIZE:-1}"  # Batch processing size
export YOLO_AUGMENT="${YOLO_AUGMENT:-false}"  # Test time augmentation

# Output configuration
export YOLO_SAVE_IMAGES="${YOLO_SAVE_IMAGES:-false}"  # Save annotated images
export YOLO_SAVE_EMBEDDINGS="${YOLO_SAVE_EMBEDDINGS:-true}"  # Store in Qdrant
export YOLO_SAVE_METADATA="${YOLO_SAVE_METADATA:-true}"  # Store in Postgres
export YOLO_OUTPUT_FORMAT="${YOLO_OUTPUT_FORMAT:-json}"  # json/csv/txt

# Integration configuration
export YOLO_ENABLE_QDRANT="${YOLO_ENABLE_QDRANT:-false}"
export QDRANT_HOST="${QDRANT_HOST:-localhost}"
export QDRANT_PORT="${QDRANT_PORT:-6333}"
export QDRANT_COLLECTION="${QDRANT_COLLECTION:-yolo_detections}"

export YOLO_ENABLE_POSTGRES="${YOLO_ENABLE_POSTGRES:-false}"
export POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
export POSTGRES_PORT="${POSTGRES_PORT:-5433}"
export POSTGRES_DB="${POSTGRES_DB:-yolo}"
export POSTGRES_USER="${POSTGRES_USER:-yolo}"
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-yolo123}"

export YOLO_ENABLE_MINIO="${YOLO_ENABLE_MINIO:-false}"
export MINIO_ENDPOINT="${MINIO_ENDPOINT:-localhost:9000}"
export MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
export MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin}"
export MINIO_BUCKET="${MINIO_BUCKET:-yolo-artifacts}"

export YOLO_ENABLE_REDIS="${YOLO_ENABLE_REDIS:-false}"
export REDIS_HOST="${REDIS_HOST:-localhost}"
export REDIS_PORT="${REDIS_PORT:-6380}"
export REDIS_TTL="${REDIS_TTL:-3600}"  # Cache TTL in seconds

# Docker configuration
export YOLO_DOCKER_IMAGE="${YOLO_DOCKER_IMAGE:-ultralytics/ultralytics:latest}"
export YOLO_CONTAINER_NAME="${YOLO_CONTAINER_NAME:-vrooli-ultralytics-yolo}"
export YOLO_DOCKER_GPU="${YOLO_DOCKER_GPU:-all}"  # GPU device allocation

# Performance configuration
export YOLO_MAX_WORKERS="${YOLO_MAX_WORKERS:-4}"
export YOLO_REQUEST_TIMEOUT="${YOLO_REQUEST_TIMEOUT:-30}"
export YOLO_STARTUP_TIMEOUT="${YOLO_STARTUP_TIMEOUT:-60}"
export YOLO_HEALTH_CHECK_INTERVAL="${YOLO_HEALTH_CHECK_INTERVAL:-30}"