#!/usr/bin/env python3
"""
Core Automation: Screenshot Examples
Demonstrates various screenshot capture capabilities without AI
"""

import requests
import base64
import json
import time
from PIL import Image
import io
import sys
import os

# Add parent directory to path to import constants
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))
from constants import AGENT_S2_BASE_URL, SCREENSHOTS_DIR

API_BASE_URL = AGENT_S2_BASE_URL

def save_screenshot(image_data, filename):
    """Save base64 image data to file"""
    # Create full path using constants
    filepath = os.path.join(SCREENSHOTS_DIR, filename)
    
    if image_data.startswith("data:image"):
        # Remove data URL prefix
        image_data = image_data.split(",")[1]
    
    img_bytes = base64.b64decode(image_data)
    img = Image.open(io.BytesIO(img_bytes))
    img.save(filepath)
    print(f"✅ Saved screenshot to {filepath}")
    return img

def capture_full_screen():
    """Capture full screen screenshot"""
    print("\n📸 Capturing full screen...")
    
    response = requests.post(
        f"{API_BASE_URL}/screenshot?format=png",
        json=None
    )
    
    if response.ok:
        data = response.json()
        img = save_screenshot(data["data"], "fullscreen.png")
        print(f"   Resolution: {data['size']['width']}x{data['size']['height']}")
        print(f"   Format: {data['format']}")
        return True
    else:
        print(f"❌ Failed: {response.text}")
        return False

def capture_region(x, y, width, height):
    """Capture specific region of screen"""
    print(f"\n📸 Capturing region ({x},{y}) {width}x{height}...")
    
    response = requests.post(
        f"{API_BASE_URL}/screenshot?format=png",
        json=[x, y, width, height]  # Region as [x, y, width, height] in body
    )
    
    if response.ok:
        data = response.json()
        save_screenshot(data["data"], f"region_{x}_{y}_{width}x{height}.png")
        return True
    else:
        print(f"❌ Failed: {response.text}")
        return False

def capture_with_quality(quality=50):
    """Capture JPEG screenshot with specific quality"""
    print(f"\n📸 Capturing JPEG with quality {quality}...")
    
    response = requests.post(
        f"{API_BASE_URL}/screenshot?format=jpeg&quality={quality}",
        json=None
    )
    
    if response.ok:
        data = response.json()
        save_screenshot(data["data"], f"screenshot_q{quality}.jpg")
        return True
    else:
        print(f"❌ Failed: {response.text}")
        return False

def continuous_capture(count=5, interval=2):
    """Capture screenshots continuously"""
    print(f"\n📸 Capturing {count} screenshots every {interval} seconds...")
    
    for i in range(count):
        response = requests.post(
            f"{API_BASE_URL}/screenshot?format=png",
            json=None
        )
        
        if response.ok:
            data = response.json()
            save_screenshot(data["data"], f"continuous_{i+1}.png")
            print(f"   Captured {i+1}/{count}")
            
            if i < count - 1:
                time.sleep(interval)
        else:
            print(f"❌ Failed on capture {i+1}: {response.text}")
            return False
    
    return True

def capture_with_cursor():
    """Capture screenshot showing mouse cursor position"""
    print("\n📸 Capturing with cursor position...")
    
    # Get current mouse position
    mouse_response = requests.get(f"{API_BASE_URL}/mouse/position")
    if mouse_response.ok:
        mouse_pos = mouse_response.json()
        print(f"   Mouse at: ({mouse_pos['x']}, {mouse_pos['y']})")
    
    # Take screenshot
    response = requests.post(
        f"{API_BASE_URL}/screenshot?format=png",
        json=None
    )
    
    if response.ok:
        data = response.json()
        save_screenshot(data["data"], "screenshot_with_cursor.png")
        return True
    else:
        print(f"❌ Failed: {response.text}")
        return False

def benchmark_screenshot_performance():
    """Benchmark screenshot capture performance"""
    print("\n⏱️  Benchmarking screenshot performance...")
    
    formats = ["png", "jpeg"]
    qualities = [100, 75, 50, 25]
    
    for format in formats:
        print(f"\n{format.upper()} format:")
        
        for quality in (qualities if format == "jpeg" else [None]):
            start_time = time.time()
            
            params = {"format": format}
            if quality:
                params["quality"] = quality
            
            response = requests.post(f"{API_BASE_URL}/screenshot", params=params, json=None)
            
            if response.ok:
                elapsed = time.time() - start_time
                data = response.json()
                size_kb = len(data["data"]) / 1024
                
                if quality:
                    print(f"   Quality {quality}: {elapsed:.3f}s, {size_kb:.1f}KB")
                else:
                    print(f"   Time: {elapsed:.3f}s, Size: {size_kb:.1f}KB")

def main():
    print("=" * 60)
    print("Agent S2 Core Automation: Screenshot Examples")
    print("=" * 60)
    
    # Check service health
    try:
        health = requests.get(f"{API_BASE_URL}/health", timeout=5)
        if not health.ok:
            print("❌ Agent S2 is not healthy")
            return
    except:
        print("❌ Cannot connect to Agent S2")
        print("   Start it with: ./resources/agent-s2/cli.sh start")
        return
    
    print("✅ Agent S2 is running")
    print("\nRunning screenshot examples...")
    
    # Run examples
    examples = [
        ("Full Screen Capture", capture_full_screen),
        ("Region Capture", lambda: capture_region(100, 100, 400, 300)),
        ("JPEG Quality 90", lambda: capture_with_quality(90)),
        ("JPEG Quality 50", lambda: capture_with_quality(50)),
        ("Screenshot with Cursor", capture_with_cursor),
        ("Continuous Capture", lambda: continuous_capture(3, 1)),
        ("Performance Benchmark", benchmark_screenshot_performance)
    ]
    
    for name, func in examples:
        print(f"\n{'='*40}")
        print(f"Example: {name}")
        print('='*40)
        
        try:
            func()
        except Exception as e:
            print(f"❌ Error: {e}")
    
    print("\n" + "="*60)
    print("✅ Screenshot examples completed!")
    print(f"Check the generated image files in {SCREENSHOTS_DIR}")
    print("="*60)

if __name__ == "__main__":
    main()