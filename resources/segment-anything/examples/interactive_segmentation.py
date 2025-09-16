#!/usr/bin/env python3
"""
Interactive Segmentation Example
Demonstrates various segmentation modes using the Segment Anything API
"""

import requests
import json
import base64
from pathlib import Path
from PIL import Image, ImageDraw
import io

# Configuration
API_BASE = "http://localhost:11454/api/v1"
HEALTH_URL = f"{API_BASE[:-7]}/health"

def check_health():
    """Check if the service is healthy"""
    try:
        response = requests.get(HEALTH_URL, timeout=5)
        if response.status_code == 200:
            data = response.json()
            print(f"✓ Service healthy: {data['status']}")
            print(f"  Device: {data['device']}")
            return True
    except Exception as e:
        print(f"✗ Service not available: {e}")
    return False

def segment_with_point(image_path, x, y):
    """Segment object at point coordinates"""
    print(f"\nSegmenting with point: ({x}, {y})")
    
    with open(image_path, 'rb') as f:
        response = requests.post(
            f"{API_BASE}/segment",
            files={'image': f},
            data={'prompt': f'point:{x},{y}'}
        )
    
    if response.status_code == 200:
        result = response.json()
        print(f"  Found {len(result.get('masks', []))} masks")
        return result
    else:
        print(f"  Error: {response.status_code}")
        return None

def segment_with_box(image_path, x1, y1, x2, y2):
    """Segment object within bounding box"""
    print(f"\nSegmenting with box: ({x1},{y1}) to ({x2},{y2})")
    
    with open(image_path, 'rb') as f:
        response = requests.post(
            f"{API_BASE}/segment",
            files={'image': f},
            data={'prompt': f'box:{x1},{y1},{x2},{y2}'}
        )
    
    if response.status_code == 200:
        result = response.json()
        print(f"  Found {len(result.get('masks', []))} masks")
        return result
    else:
        print(f"  Error: {response.status_code}")
        return None

def segment_automatic(image_path):
    """Automatically segment all objects"""
    print("\nPerforming automatic segmentation...")
    
    with open(image_path, 'rb') as f:
        response = requests.post(
            f"{API_BASE}/segment",
            files={'image': f},
            data={'prompt': 'auto'}
        )
    
    if response.status_code == 200:
        result = response.json()
        masks = result.get('masks', [])
        print(f"  Found {len(masks)} objects")
        return result
    else:
        print(f"  Error: {response.status_code}")
        return None

def create_sample_image():
    """Create a sample test image with simple shapes"""
    print("Creating sample image...")
    
    # Create image with shapes
    img = Image.new('RGB', (800, 600), 'white')
    draw = ImageDraw.Draw(img)
    
    # Draw shapes
    draw.rectangle([100, 100, 300, 250], fill='blue', outline='darkblue', width=3)
    draw.ellipse([400, 150, 600, 350], fill='red', outline='darkred', width=3)
    draw.polygon([(150, 400), (250, 350), (350, 400), (300, 500), (200, 500)], 
                 fill='green', outline='darkgreen', width=3)
    draw.ellipse([500, 400, 700, 500], fill='yellow', outline='orange', width=3)
    
    # Save image
    img_path = '/tmp/sample_segmentation.png'
    img.save(img_path)
    print(f"  Saved to: {img_path}")
    return img_path

def visualize_masks(image_path, masks):
    """Visualize segmentation masks on the image"""
    print("\nVisualizing masks...")
    
    # Load original image
    img = Image.open(image_path).convert('RGBA')
    
    # Create overlay for masks
    overlay = Image.new('RGBA', img.size, (0, 0, 0, 0))
    draw = ImageDraw.Draw(overlay)
    
    # Colors for different masks
    colors = ['red', 'blue', 'green', 'yellow', 'purple', 'orange']
    
    # Draw each mask
    for i, mask in enumerate(masks[:6]):  # Limit to 6 masks for visualization
        color = colors[i % len(colors)]
        # In real implementation, decode mask data
        # For demo, just draw a placeholder
        if 'bbox' in mask:
            bbox = mask['bbox']
            draw.rectangle(bbox, outline=color, width=3)
    
    # Composite
    result = Image.alpha_composite(img, overlay)
    result_path = '/tmp/segmentation_result.png'
    result.save(result_path)
    print(f"  Visualization saved to: {result_path}")
    return result_path

def export_to_coco(masks):
    """Export masks in COCO format"""
    print("\nExporting to COCO format...")
    
    coco_data = {
        "images": [{"id": 1, "file_name": "image.jpg", "width": 800, "height": 600}],
        "annotations": [],
        "categories": [{"id": 1, "name": "object"}]
    }
    
    for i, mask in enumerate(masks):
        annotation = {
            "id": i + 1,
            "image_id": 1,
            "category_id": 1,
            "segmentation": mask.get('segmentation', []),
            "area": mask.get('area', 0),
            "bbox": mask.get('bbox', [0, 0, 0, 0]),
            "iscrowd": 0
        }
        coco_data["annotations"].append(annotation)
    
    # Save COCO JSON
    coco_path = '/tmp/segmentation_coco.json'
    with open(coco_path, 'w') as f:
        json.dump(coco_data, f, indent=2)
    print(f"  COCO data saved to: {coco_path}")
    return coco_path

def main():
    """Run interactive segmentation examples"""
    print("=" * 50)
    print("Segment Anything Interactive Example")
    print("=" * 50)
    
    # Check service health
    if not check_health():
        print("\n⚠ Please start the service first:")
        print("  vrooli resource segment-anything manage start --wait")
        return
    
    # Create sample image
    image_path = create_sample_image()
    
    # Example 1: Point-based segmentation
    result_point = segment_with_point(image_path, 200, 175)  # Center of blue rectangle
    
    # Example 2: Box-based segmentation
    result_box = segment_with_box(image_path, 380, 130, 620, 370)  # Around red circle
    
    # Example 3: Automatic segmentation
    result_auto = segment_automatic(image_path)
    
    # Visualize results
    if result_auto and 'masks' in result_auto:
        visualize_masks(image_path, result_auto['masks'])
        export_to_coco(result_auto['masks'])
    
    print("\n" + "=" * 50)
    print("Examples completed!")
    print("\nGenerated files:")
    print("  - Sample image: /tmp/sample_segmentation.png")
    print("  - Visualization: /tmp/segmentation_result.png")
    print("  - COCO export: /tmp/segmentation_coco.json")

if __name__ == "__main__":
    main()