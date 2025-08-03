#!/usr/bin/env python3
"""
Validate and update image metadata YAML file.
Helps prevent drift between actual images and metadata.
"""

import os
import yaml
from PIL import Image
from pathlib import Path
import hashlib
import sys

def get_image_info(file_path):
    """Extract actual image properties."""
    try:
        with Image.open(file_path) as img:
            return {
                'format': img.format.lower() if img.format else Path(file_path).suffix[1:],
                'dimensions': list(img.size),
                'fileSize': os.path.getsize(file_path),
                'mode': img.mode,
                'animated': getattr(img, 'is_animated', False)
            }
    except Exception as e:
        print(f"Error reading {file_path}: {e}")
        return None

def validate_metadata(yaml_file='images.yaml'):
    """Validate that metadata matches actual files."""
    with open(yaml_file, 'r') as f:
        data = yaml.safe_load(f)
    
    base_dir = Path(__file__).parent
    errors = []
    warnings = []
    
    # Flatten all image entries
    all_images = []
    def extract_images(obj, prefix=''):
        if isinstance(obj, list):
            all_images.extend(obj)
        elif isinstance(obj, dict):
            for key, value in obj.items():
                if key not in ['testSuites', 'schema', 'version']:
                    extract_images(value, prefix)
    
    extract_images(data.get('images', {}))
    
    # Check each image
    for img_meta in all_images:
        path = img_meta.get('path')
        if not path:
            errors.append("Image entry missing 'path' field")
            continue
            
        full_path = base_dir / path
        
        # Check if file exists
        if not full_path.exists():
            errors.append(f"File not found: {path}")
            continue
        
        # Get actual properties
        actual = get_image_info(full_path)
        if not actual:
            continue
        
        # Compare key properties
        if img_meta.get('format') != actual['format']:
            errors.append(f"{path}: format mismatch - YAML: {img_meta.get('format')}, Actual: {actual['format']}")
        
        if img_meta.get('dimensions') != actual['dimensions']:
            errors.append(f"{path}: dimensions mismatch - YAML: {img_meta.get('dimensions')}, Actual: {actual['dimensions']}")
        
        # File size can vary slightly, warn if > 10% difference
        yaml_size = img_meta.get('fileSize', 0)
        actual_size = actual['fileSize']
        if yaml_size > 0 and abs(yaml_size - actual_size) / actual_size > 0.1:
            warnings.append(f"{path}: file size differs > 10% - YAML: {yaml_size}, Actual: {actual_size}")
    
    # Find orphaned files (images without metadata)
    yaml_paths = {img.get('path') for img in all_images if img.get('path')}
    for img_file in base_dir.rglob('*'):
        if img_file.is_file() and img_file.suffix.lower() in ['.png', '.jpg', '.jpeg', '.gif', '.webp', '.heic']:
            rel_path = img_file.relative_to(base_dir)
            if str(rel_path) not in yaml_paths and not any(part.startswith('.') for part in rel_path.parts):
                if str(rel_path) != 'generate_ocr_test_images.py':
                    warnings.append(f"Image without metadata: {rel_path}")
    
    return errors, warnings

def main():
    """Run validation and report results."""
    print("Validating image metadata...")
    errors, warnings = validate_metadata()
    
    if errors:
        print(f"\n❌ Found {len(errors)} errors:")
        for error in errors:
            print(f"  - {error}")
    
    if warnings:
        print(f"\n⚠️  Found {len(warnings)} warnings:")
        for warning in warnings:
            print(f"  - {warning}")
    
    if not errors and not warnings:
        print("✅ All metadata is valid!")
    
    # Exit with error code if there are errors
    sys.exit(1 if errors else 0)

if __name__ == '__main__':
    main()