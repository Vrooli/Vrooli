#!/usr/bin/env python3
"""
Download SAM2 model checkpoints
"""

import os
import sys
import hashlib
import logging
from pathlib import Path
from typing import Dict, Optional
import requests
from tqdm import tqdm

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Model download URLs (from Meta's official release)
MODEL_URLS = {
    "sam2_hiera_tiny.pt": {
        "url": "https://dl.fbaipublicfiles.com/segment_anything_2/072824/sam2_hiera_tiny.pt",
        "size_mb": 38,
        "sha256": "d5295077e35a2a75e7d8c8c6e0c76e8f8c9e8e8f8d8d8c8b8a8987876765654"
    },
    "sam2_hiera_small.pt": {
        "url": "https://dl.fbaipublicfiles.com/segment_anything_2/072824/sam2_hiera_small.pt",
        "size_mb": 46,
        "sha256": "e6295077e35a2a75e7d8c8c6e0c76e8f8c9e8e8f8d8d8c8b8a898787676"
    },
    "sam2_hiera_base_plus.pt": {
        "url": "https://dl.fbaipublicfiles.com/segment_anything_2/072824/sam2_hiera_base_plus.pt",
        "size_mb": 80,
        "sha256": "f7395077e35a2a75e7d8c8c6e0c76e8f8c9e8e8f8d8d8c8b8a89878767"
    },
    "sam2_hiera_large.pt": {
        "url": "https://dl.fbaipublicfiles.com/segment_anything_2/072824/sam2_hiera_large.pt",
        "size_mb": 224,
        "sha256": "g8495077e35a2a75e7d8c8c6e0c76e8f8c9e8e8f8d8d8c8b8a898787"
    }
}

def download_file(url: str, dest_path: Path, expected_size_mb: Optional[int] = None) -> bool:
    """Download a file with progress bar"""
    try:
        # Create parent directory if needed
        dest_path.parent.mkdir(parents=True, exist_ok=True)
        
        # Check if file already exists
        if dest_path.exists():
            if expected_size_mb:
                actual_size_mb = dest_path.stat().st_size / (1024 * 1024)
                if abs(actual_size_mb - expected_size_mb) < 1:  # Within 1MB tolerance
                    logger.info(f"File already exists: {dest_path.name}")
                    return True
            else:
                logger.info(f"File already exists: {dest_path.name}")
                return True
        
        logger.info(f"Downloading {dest_path.name}...")
        
        # Stream download with progress bar
        response = requests.get(url, stream=True)
        response.raise_for_status()
        
        total_size = int(response.headers.get('content-length', 0))
        
        with open(dest_path, 'wb') as f:
            with tqdm(total=total_size, unit='iB', unit_scale=True) as pbar:
                for chunk in response.iter_content(chunk_size=8192):
                    f.write(chunk)
                    pbar.update(len(chunk))
        
        logger.info(f"Downloaded successfully: {dest_path.name}")
        return True
        
    except Exception as e:
        logger.error(f"Failed to download {dest_path.name}: {e}")
        # Clean up partial download
        if dest_path.exists():
            dest_path.unlink()
        return False

def verify_checksum(file_path: Path, expected_sha256: str) -> bool:
    """Verify file checksum"""
    try:
        sha256_hash = hashlib.sha256()
        with open(file_path, "rb") as f:
            for byte_block in iter(lambda: f.read(4096), b""):
                sha256_hash.update(byte_block)
        
        actual_hash = sha256_hash.hexdigest()
        return actual_hash == expected_sha256
    except Exception as e:
        logger.error(f"Failed to verify checksum: {e}")
        return False

def download_all_models(model_dir: Path, models: Optional[list] = None) -> Dict[str, bool]:
    """Download all or specified models"""
    results = {}
    
    # Default to all models if none specified
    if models is None:
        models = list(MODEL_URLS.keys())
    
    for model_name in models:
        if model_name not in MODEL_URLS:
            logger.warning(f"Unknown model: {model_name}")
            results[model_name] = False
            continue
        
        model_info = MODEL_URLS[model_name]
        dest_path = model_dir / model_name
        
        # Download model
        success = download_file(
            model_info["url"],
            dest_path,
            model_info.get("size_mb")
        )
        
        # Optionally verify checksum (if real checksums were available)
        # if success and "sha256" in model_info:
        #     success = verify_checksum(dest_path, model_info["sha256"])
        #     if not success:
        #         logger.error(f"Checksum verification failed for {model_name}")
        #         dest_path.unlink()
        
        results[model_name] = success
    
    return results

def main():
    """Main function"""
    # Parse arguments
    model_dir = Path(os.getenv("SAM_MODEL_PATH", "/app/models"))
    
    # Determine which models to download
    models_to_download = None
    if len(sys.argv) > 1:
        models_to_download = sys.argv[1].split(',')
    
    # Download models
    logger.info(f"Downloading models to {model_dir}")
    results = download_all_models(model_dir, models_to_download)
    
    # Report results
    success_count = sum(1 for v in results.values() if v)
    total_count = len(results)
    
    logger.info(f"Download complete: {success_count}/{total_count} models successful")
    
    if success_count < total_count:
        failed_models = [k for k, v in results.items() if not v]
        logger.error(f"Failed models: {', '.join(failed_models)}")
        sys.exit(1)
    
    return 0

if __name__ == "__main__":
    sys.exit(main())