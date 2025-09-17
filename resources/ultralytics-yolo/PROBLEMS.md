# Ultralytics YOLO - Known Issues and Solutions

## 1. Docker Container Name Matching Issue
**Problem**: Test suite sometimes fails to detect running container even when it's present.
**Cause**: Variable substitution or grep pattern matching issue with `${YOLO_CONTAINER_NAME}` in tests.
**Solution**: Hardcoded container name in test.sh temporarily. Long-term fix needed for variable expansion.
**Status**: Partially resolved

## 2. Service Startup Time
**Problem**: Service needs 3-5 seconds after container starts before health endpoint responds.
**Cause**: Model loading and initialization takes time, especially for larger models.
**Solution**: Added wait logic in start command, but tests may need additional delay.
**Status**: Resolved

## 3. Detection API Response Time
**Problem**: First detection request can take 100+ ms even with GPU.
**Cause**: Model warm-up and CUDA initialization on first inference.
**Solution**: Consider pre-warming the model on startup.
**Status**: Known issue

## 4. GPU Memory Management
**Problem**: Multiple YOLO instances can exhaust GPU memory.
**Cause**: Each instance loads full model into VRAM.
**Solution**: Implemented GPU_MEMORY_FRACTION config but needs testing with multiple instances.
**Status**: Needs testing

## 5. Model Download Speed
**Problem**: Initial model download can be slow (49.7MB for yolov8m).
**Cause**: Downloads from GitHub releases on first use.
**Solution**: Pre-cache models in Docker image or use local model repository.
**Status**: Enhancement needed