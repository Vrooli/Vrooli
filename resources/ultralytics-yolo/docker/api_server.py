#!/usr/bin/env python3
"""
Ultralytics YOLO API Server
Provides REST endpoints for object detection, segmentation, and classification
"""

import os
import json
import time
import torch
import numpy as np
from pathlib import Path
from typing import Dict, Any, List, Optional
from datetime import datetime

from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.responses import JSONResponse
from pydantic import BaseModel
import uvicorn
from ultralytics import YOLO

# Configuration
PORT = int(os.getenv('YOLO_PORT', 11455))
MODEL_PATH = os.getenv('YOLO_MODEL_PATH', '/models')
DEFAULT_MODEL = os.getenv('YOLO_DEFAULT_MODEL', 'yolov8m.pt')
DEVICE = os.getenv('YOLO_DEVICE', 'auto')
CONFIDENCE = float(os.getenv('YOLO_CONFIDENCE', 0.25))
IOU_THRESHOLD = float(os.getenv('YOLO_IOU_THRESHOLD', 0.45))
IMAGE_SIZE = int(os.getenv('YOLO_IMAGE_SIZE', 640))

# Initialize FastAPI
app = FastAPI(
    title="Ultralytics YOLO API",
    description="Real-time object detection, segmentation, and classification",
    version="1.0.0"
)

# Model cache
models = {}
current_model = None

# Device selection
def get_device():
    if DEVICE == 'auto':
        return 'cuda' if torch.cuda.is_available() else 'cpu'
    return DEVICE

# Load model
def load_model(model_name: str = DEFAULT_MODEL):
    global current_model
    if model_name not in models:
        model_path = Path(MODEL_PATH) / model_name
        if not model_path.exists():
            # Download if not exists
            model = YOLO(model_name)
        else:
            model = YOLO(str(model_path))
        models[model_name] = model
    current_model = models[model_name]
    return current_model

# Health check endpoint
@app.get("/health")
async def health():
    return {
        "status": "healthy",
        "timestamp": datetime.utcnow().isoformat(),
        "device": get_device(),
        "model_loaded": current_model is not None,
        "models_cached": list(models.keys())
    }

# Model status endpoint
@app.get("/models/status")
async def model_status():
    return {
        "current_model": DEFAULT_MODEL if current_model else None,
        "available_models": list(models.keys()),
        "device": get_device(),
        "cuda_available": torch.cuda.is_available()
    }

# List models endpoint
@app.get("/models")
async def list_models():
    model_dir = Path(MODEL_PATH)
    model_files = list(model_dir.glob("*.pt"))
    return {
        "models": [f.name for f in model_files],
        "current": DEFAULT_MODEL if current_model else None
    }

# Detection endpoint
@app.post("/detect")
async def detect(
    file: UploadFile = File(...),
    model: str = DEFAULT_MODEL,
    confidence: float = CONFIDENCE,
    iou: float = IOU_THRESHOLD
):
    try:
        # Load model
        yolo = load_model(model)
        
        # Read image
        contents = await file.read()
        nparr = np.frombuffer(contents, np.uint8)
        import cv2
        img = cv2.imdecode(nparr, cv2.IMREAD_COLOR)
        
        # Run detection
        start_time = time.time()
        results = yolo(
            img, 
            conf=confidence, 
            iou=iou,
            device=get_device()
        )
        inference_time = (time.time() - start_time) * 1000
        
        # Process results
        detections = []
        for r in results:
            if r.boxes is not None:
                for box in r.boxes:
                    detections.append({
                        "class": r.names[int(box.cls)],
                        "confidence": float(box.conf),
                        "bbox": box.xyxy[0].tolist(),
                        "class_id": int(box.cls)
                    })
        
        return {
            "detections": detections,
            "metadata": {
                "model": model,
                "inference_time": inference_time,
                "image_size": [img.shape[1], img.shape[0]],
                "device": get_device()
            }
        }
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

# Initialize on startup
@app.on_event("startup")
async def startup_event():
    load_model(DEFAULT_MODEL)
    print(f"YOLO API Server running on port {PORT}")
    print(f"Device: {get_device()}")

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=PORT)
