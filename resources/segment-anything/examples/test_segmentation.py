#!/usr/bin/env python3
"""
Test script for Segment Anything API
Demonstrates basic segmentation capabilities
"""

import requests
import json
import base64
from PIL import Image, ImageDraw
import numpy as np
import io
import sys
from pathlib import Path

API_BASE = "http://localhost:11454"

def create_test_image():
    """Create a simple test image with shapes"""
    img = Image.new('RGB', (512, 512), color='white')
    draw = ImageDraw.Draw(img)
    
    # Draw some shapes
    draw.rectangle([50, 50, 200, 200], fill='red', outline='black')
    draw.ellipse([250, 250, 400, 400], fill='blue', outline='black')
    draw.polygon([(300, 50), (400, 150), (250, 150)], fill='green', outline='black')
    
    return img

def test_health():
    """Test health endpoint"""
    print("Testing health endpoint...")
    try:
        response = requests.get(f"{API_BASE}/health", timeout=5)
        response.raise_for_status()
        print(f"✓ Health check passed: {response.json()}")
        return True
    except Exception as e:
        print(f"✗ Health check failed: {e}")
        return False

def test_models():
    """Test models endpoint"""
    print("\nTesting models endpoint...")
    try:
        response = requests.get(f"{API_BASE}/models", timeout=5)
        response.raise_for_status()
        models = response.json()
        print(f"✓ Available models: {models}")
        return True
    except Exception as e:
        print(f"✗ Models endpoint failed: {e}")
        return False

def test_segmentation_point():
    """Test point-based segmentation"""
    print("\nTesting point-based segmentation...")
    
    # Create test image
    img = create_test_image()
    img_bytes = io.BytesIO()
    img.save(img_bytes, format='PNG')
    img_bytes.seek(0)
    
    # Prepare request
    request_data = {
        "points": [
            {"x": 0.25, "y": 0.25, "label": 1},  # Inside red rectangle
        ],
        "multimask_output": True
    }
    
    try:
        files = {'file': ('test.png', img_bytes, 'image/png')}
        data = {'request': json.dumps(request_data)}
        
        response = requests.post(
            f"{API_BASE}/segment",
            files=files,
            data=data,
            timeout=10
        )
        response.raise_for_status()
        
        result = response.json()
        print(f"✓ Segmentation successful:")
        print(f"  - Masks generated: {len(result.get('masks', []))}")
        print(f"  - Processing time: {result.get('processing_time', 0):.3f}s")
        print(f"  - Model used: {result.get('model_used', 'unknown')}")
        return True
        
    except Exception as e:
        print(f"✗ Segmentation failed: {e}")
        return False

def test_segmentation_box():
    """Test box-based segmentation"""
    print("\nTesting box-based segmentation...")
    
    # Create test image
    img = create_test_image()
    img_bytes = io.BytesIO()
    img.save(img_bytes, format='PNG')
    img_bytes.seek(0)
    
    # Prepare request with box prompt
    request_data = {
        "boxes": [
            {"x1": 0.1, "y1": 0.1, "x2": 0.4, "y2": 0.4},  # Around red rectangle
        ],
        "multimask_output": False
    }
    
    try:
        files = {'file': ('test.png', img_bytes, 'image/png')}
        data = {'request': json.dumps(request_data)}
        
        response = requests.post(
            f"{API_BASE}/segment",
            files=files,
            data=data,
            timeout=10
        )
        response.raise_for_status()
        
        result = response.json()
        print(f"✓ Box segmentation successful:")
        print(f"  - Masks generated: {len(result.get('masks', []))}")
        
        # Check mask properties
        if result.get('masks'):
            mask = result['masks'][0]
            print(f"  - Mask area: {mask.get('area', 0)} pixels")
            print(f"  - Predicted IoU: {mask.get('predicted_iou', 0):.3f}")
        
        return True
        
    except Exception as e:
        print(f"✗ Box segmentation failed: {e}")
        return False

def test_auto_segmentation():
    """Test automatic segmentation"""
    print("\nTesting automatic segmentation...")
    
    # Create test image
    img = create_test_image()
    img_bytes = io.BytesIO()
    img.save(img_bytes, format='PNG')
    img_bytes.seek(0)
    
    try:
        files = {'file': ('test.png', img_bytes, 'image/png')}
        data = {
            'points_per_side': 16,
            'pred_iou_thresh': 0.86,
            'stability_score_thresh': 0.92
        }
        
        response = requests.post(
            f"{API_BASE}/segment/auto",
            files=files,
            data=data,
            timeout=15
        )
        response.raise_for_status()
        
        result = response.json()
        print(f"✓ Auto segmentation successful:")
        print(f"  - Objects detected: {result.get('count', 0)}")
        print(f"  - Processing time: {result.get('processing_time', 0):.3f}s")
        
        return True
        
    except Exception as e:
        print(f"✗ Auto segmentation failed: {e}")
        return False

def main():
    """Run all tests"""
    print("=" * 50)
    print("Segment Anything API Test Suite")
    print("=" * 50)
    
    # Check if service is running
    if not test_health():
        print("\n⚠️  Service not running. Start it with:")
        print("    vrooli resource segment-anything develop")
        sys.exit(1)
    
    # Run tests
    tests = [
        test_models,
        test_segmentation_point,
        test_segmentation_box,
        test_auto_segmentation
    ]
    
    passed = 0
    failed = 0
    
    for test in tests:
        if test():
            passed += 1
        else:
            failed += 1
    
    # Summary
    print("\n" + "=" * 50)
    print("Test Summary")
    print("=" * 50)
    print(f"✓ Passed: {passed}")
    print(f"✗ Failed: {failed}")
    print(f"Total: {passed + failed}")
    
    if failed == 0:
        print("\n🎉 All tests passed!")
        return 0
    else:
        print(f"\n⚠️  {failed} test(s) failed")
        return 1

if __name__ == "__main__":
    sys.exit(main())