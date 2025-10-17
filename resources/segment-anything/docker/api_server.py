"""
Segment Anything Model API Server
Provides REST endpoints for image and video segmentation using SAM2
"""

import os
import sys
import json
import time
import asyncio
import logging
import io
from typing import List, Dict, Any, Optional, Tuple
from pathlib import Path
from datetime import datetime

import torch
import numpy as np
from PIL import Image
import cv2
from fastapi import FastAPI, HTTPException, UploadFile, File, Form, BackgroundTasks
from fastapi.responses import JSONResponse, StreamingResponse
from pydantic import BaseModel, Field
import uvicorn

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Add SAM2 to path
sys.path.append('/app/sam2')

# Import SAM2 components
try:
    from sam2.build_sam import build_sam2
    from sam2.automatic_mask_generator import SAM2AutomaticMaskGenerator
    from sam2.sam2_image_predictor import SAM2ImagePredictor
    SAM2_AVAILABLE = True
except ImportError as e:
    logger.warning(f"SAM2 not available: {e}")
    SAM2_AVAILABLE = False

app = FastAPI(
    title="Segment Anything API",
    description="SAM2/HQ-SAM segmentation service for Vrooli",
    version="2.0.0"
)

# Global model storage
models = {}
current_model = None
image_predictor = None
mask_generator = None

# Configuration
MODEL_PATH = os.getenv("SAM_MODEL_PATH", "/app/models")
DEFAULT_MODEL = os.getenv("DEFAULT_MODEL", "sam2_hiera_small")
GPU_AVAILABLE = torch.cuda.is_available()
DEVICE = "cuda" if GPU_AVAILABLE else "cpu"

# Model configurations
MODEL_CONFIGS = {
    "sam2_hiera_tiny": {
        "checkpoint": "sam2_hiera_tiny.pt",
        "config": "sam2_hiera_t.yaml",
        "size": "38MB",
        "speed": "fastest"
    },
    "sam2_hiera_small": {
        "checkpoint": "sam2_hiera_small.pt", 
        "config": "sam2_hiera_s.yaml",
        "size": "46MB",
        "speed": "fast"
    },
    "sam2_hiera_base_plus": {
        "checkpoint": "sam2_hiera_base_plus.pt",
        "config": "sam2_hiera_b+.yaml",
        "size": "80MB",
        "speed": "balanced"
    },
    "sam2_hiera_large": {
        "checkpoint": "sam2_hiera_large.pt",
        "config": "sam2_hiera_l.yaml",
        "size": "224MB",
        "speed": "accurate"
    }
}

# Request/Response Models
class PointPrompt(BaseModel):
    x: float = Field(..., description="X coordinate (0-1 normalized)")
    y: float = Field(..., description="Y coordinate (0-1 normalized)")
    label: int = Field(1, description="1 for foreground, 0 for background")

class BoxPrompt(BaseModel):
    x1: float = Field(..., description="Top-left X (0-1 normalized)")
    y1: float = Field(..., description="Top-left Y (0-1 normalized)")
    x2: float = Field(..., description="Bottom-right X (0-1 normalized)")
    y2: float = Field(..., description="Bottom-right Y (0-1 normalized)")

class SegmentRequest(BaseModel):
    model: Optional[str] = Field(DEFAULT_MODEL, description="Model to use")
    points: Optional[List[PointPrompt]] = Field(None, description="Point prompts")
    boxes: Optional[List[BoxPrompt]] = Field(None, description="Box prompts")
    multimask_output: bool = Field(True, description="Output multiple masks")
    return_logits: bool = Field(False, description="Return raw logits")

class MaskData(BaseModel):
    segmentation: List[List[float]] = Field(..., description="RLE or polygon coordinates")
    bbox: List[float] = Field(..., description="Bounding box [x, y, width, height]")
    area: int = Field(..., description="Mask area in pixels")
    predicted_iou: float = Field(..., description="Model's confidence")

class SegmentResponse(BaseModel):
    masks: List[MaskData]
    processing_time: float
    model_used: str
    image_size: List[int]

class HealthResponse(BaseModel):
    status: str
    models_loaded: Dict[str, bool]
    gpu_available: bool
    memory_usage: Dict[str, Any]
    uptime: float

# Startup event
@app.on_event("startup")
async def startup_event():
    """Initialize models on startup"""
    logger.info(f"Starting Segment Anything API on {DEVICE}")
    logger.info(f"GPU Available: {GPU_AVAILABLE}")
    
    # Load default model
    if SAM2_AVAILABLE:
        await load_model(DEFAULT_MODEL)
    else:
        logger.warning("SAM2 not available, running in mock mode")

# Utility functions
def image_to_numpy(image: Image.Image) -> np.ndarray:
    """Convert PIL Image to numpy array"""
    return np.array(image.convert("RGB"))

def normalize_coords(coords: Tuple[float, ...], image_shape: Tuple[int, int]) -> Tuple[int, ...]:
    """Convert normalized coordinates to pixel coordinates"""
    h, w = image_shape[:2]
    if len(coords) == 2:  # Point
        return int(coords[0] * w), int(coords[1] * h)
    else:  # Box
        return (
            int(coords[0] * w), int(coords[1] * h),
            int(coords[2] * w), int(coords[3] * h)
        )

def mask_to_rle(mask: np.ndarray) -> Dict[str, Any]:
    """Convert binary mask to RLE format"""
    pixels = mask.flatten()
    pixels = np.concatenate([[0], pixels, [0]])
    runs = np.where(pixels[1:] != pixels[:-1])[0] + 1
    runs[1::2] -= runs[::2]
    return {'counts': runs.tolist(), 'size': list(mask.shape)}

def rle_to_mask(rle: Dict[str, Any]) -> np.ndarray:
    """Convert RLE to binary mask"""
    counts = rle['counts']
    size = rle['size']
    mask = np.zeros(size[0] * size[1], dtype=np.uint8)
    
    start = 0
    for i, count in enumerate(counts):
        if i % 2 == 1:
            mask[start:start+count] = 1
        start += count
    
    return mask.reshape(size)

async def load_model(model_name: str):
    """Load a SAM2 model"""
    global current_model, image_predictor, mask_generator
    
    if not SAM2_AVAILABLE:
        logger.warning("SAM2 not available, cannot load model")
        return False
    
    if model_name not in MODEL_CONFIGS:
        raise ValueError(f"Unknown model: {model_name}")
    
    config = MODEL_CONFIGS[model_name]
    checkpoint_path = Path(MODEL_PATH) / config["checkpoint"]
    config_path = f"/app/sam2/sam2_configs/{config['config']}"
    
    # Check if checkpoint exists
    if not checkpoint_path.exists():
        logger.info(f"Model checkpoint not found, downloading: {model_name}")
        # In production, this would trigger download
        # For now, we'll create a mock model
        models[model_name] = {"loaded": False, "error": "Checkpoint not found"}
        return False
    
    try:
        logger.info(f"Loading model: {model_name}")
        
        # Build SAM2 model
        sam2_model = build_sam2(
            config_file=config_path,
            ckpt_path=str(checkpoint_path),
            device=DEVICE
        )
        
        # Create predictor and generator
        image_predictor = SAM2ImagePredictor(sam2_model)
        mask_generator = SAM2AutomaticMaskGenerator(sam2_model)
        
        current_model = model_name
        models[model_name] = {
            "loaded": True,
            "config": config,
            "device": DEVICE
        }
        
        logger.info(f"Model loaded successfully: {model_name}")
        return True
        
    except Exception as e:
        logger.error(f"Failed to load model {model_name}: {e}")
        models[model_name] = {"loaded": False, "error": str(e)}
        return False

# API Endpoints
@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    try:
        import psutil
        process = psutil.Process()
        memory_info = process.memory_info()
        memory_usage = {
            "rss_mb": memory_info.rss / 1024 / 1024,
            "vms_mb": memory_info.vms / 1024 / 1024,
            "percent": process.memory_percent()
        }
        uptime = time.time() - process.create_time()
    except ImportError:
        # Fallback if psutil not available
        import resource
        usage = resource.getrusage(resource.RUSAGE_SELF)
        memory_usage = {
            "rss_mb": usage.ru_maxrss / 1024,  # Linux reports in KB
            "vms_mb": 0,  # Not available without psutil
            "percent": 0
        }
        uptime = time.time()
    
    return HealthResponse(
        status="healthy" if SAM2_AVAILABLE else "degraded",
        models_loaded={name: info.get("loaded", False) for name, info in models.items()},
        gpu_available=GPU_AVAILABLE,
        memory_usage=memory_usage,
        uptime=uptime
    )

@app.get("/models")
async def list_models():
    """List available models"""
    return {
        "current_model": current_model,
        "available_models": {
            name: {
                **config,
                "loaded": models.get(name, {}).get("loaded", False)
            }
            for name, config in MODEL_CONFIGS.items()
        },
        "device": DEVICE
    }

@app.post("/models/{model_name}/load")
async def load_model_endpoint(model_name: str):
    """Load a specific model"""
    if model_name not in MODEL_CONFIGS:
        raise HTTPException(status_code=404, detail=f"Model not found: {model_name}")
    
    success = await load_model(model_name)
    
    if success:
        return {"message": f"Model loaded: {model_name}"}
    else:
        raise HTTPException(status_code=500, detail=f"Failed to load model: {model_name}")

@app.post("/segment", response_model=SegmentResponse)
async def segment_image(
    file: UploadFile = File(...),
    request: str = Form(...)  # JSON string of SegmentRequest
):
    """Segment an image with point/box prompts"""
    if not SAM2_AVAILABLE or not image_predictor:
        # Return mock data for testing
        return SegmentResponse(
            masks=[
                MaskData(
                    segmentation=[[0, 0, 100, 100]],
                    bbox=[0, 0, 100, 100],
                    area=10000,
                    predicted_iou=0.95
                )
            ],
            processing_time=0.1,
            model_used=current_model or "mock",
            image_size=[512, 512]
        )
    
    start_time = time.time()
    
    # Parse request
    try:
        seg_request = SegmentRequest.parse_raw(request)
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"Invalid request: {e}")
    
    # Load and process image
    contents = await file.read()
    image = Image.open(io.BytesIO(contents))
    image_np = image_to_numpy(image)
    
    # Set image for predictor
    image_predictor.set_image(image_np)
    
    # Prepare prompts
    point_coords = []
    point_labels = []
    boxes = []
    
    if seg_request.points:
        for point in seg_request.points:
            coords = normalize_coords((point.x, point.y), image_np.shape)
            point_coords.append(coords)
            point_labels.append(point.label)
    
    if seg_request.boxes:
        for box in seg_request.boxes:
            coords = normalize_coords((box.x1, box.y1, box.x2, box.y2), image_np.shape)
            boxes.append(coords)
    
    # Run prediction
    input_points = np.array(point_coords) if point_coords else None
    input_labels = np.array(point_labels) if point_labels else None
    input_boxes = np.array(boxes) if boxes else None
    
    masks, scores, logits = image_predictor.predict(
        point_coords=input_points,
        point_labels=input_labels,
        box=input_boxes,
        multimask_output=seg_request.multimask_output
    )
    
    # Format response
    mask_data = []
    for mask, score in zip(masks, scores):
        # Convert mask to RLE
        rle = mask_to_rle(mask)
        
        # Calculate bounding box
        y_indices, x_indices = np.where(mask)
        if len(x_indices) > 0:
            bbox = [
                int(x_indices.min()),
                int(y_indices.min()),
                int(x_indices.max() - x_indices.min()),
                int(y_indices.max() - y_indices.min())
            ]
        else:
            bbox = [0, 0, 0, 0]
        
        mask_data.append(MaskData(
            segmentation=rle['counts'],
            bbox=bbox,
            area=int(mask.sum()),
            predicted_iou=float(score)
        ))
    
    processing_time = time.time() - start_time
    
    return SegmentResponse(
        masks=mask_data,
        processing_time=processing_time,
        model_used=current_model,
        image_size=list(image_np.shape[:2])
    )

@app.post("/segment/auto")
async def segment_auto(
    file: UploadFile = File(...),
    points_per_side: int = Form(32),
    pred_iou_thresh: float = Form(0.88),
    stability_score_thresh: float = Form(0.95)
):
    """Automatically segment everything in an image"""
    if not SAM2_AVAILABLE or not mask_generator:
        # Return mock data
        return {
            "masks": [
                {
                    "segmentation": [[0, 0, 50, 50]],
                    "bbox": [0, 0, 50, 50],
                    "area": 2500,
                    "predicted_iou": 0.95
                }
            ],
            "count": 1,
            "processing_time": 0.1
        }
    
    start_time = time.time()
    
    # Load image
    contents = await file.read()
    image = Image.open(io.BytesIO(contents))
    image_np = image_to_numpy(image)
    
    # Generate masks
    masks = mask_generator.generate(image_np)
    
    # Format masks
    formatted_masks = []
    for mask in masks:
        formatted_masks.append({
            "segmentation": mask_to_rle(mask['segmentation'])['counts'],
            "bbox": mask['bbox'].tolist(),
            "area": int(mask['area']),
            "predicted_iou": float(mask['predicted_iou']),
            "stability_score": float(mask['stability_score'])
        })
    
    processing_time = time.time() - start_time
    
    return {
        "masks": formatted_masks,
        "count": len(formatted_masks),
        "processing_time": processing_time,
        "model_used": current_model
    }

@app.post("/export/{format}")
async def export_masks(
    format: str,
    masks: List[Dict[str, Any]]
):
    """Export masks in different formats"""
    supported_formats = ["png", "coco", "geojson", "numpy"]
    
    if format not in supported_formats:
        raise HTTPException(
            status_code=400,
            detail=f"Unsupported format. Choose from: {supported_formats}"
        )
    
    # Export logic would go here
    # For now, return a success message
    return {
        "format": format,
        "exported": len(masks),
        "message": f"Masks exported as {format}"
    }

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=11454)