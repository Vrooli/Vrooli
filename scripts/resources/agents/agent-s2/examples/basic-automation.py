#!/usr/bin/env python3
"""
Basic Agent S2 Automation Example
Demonstrates simple GUI automation tasks
"""

import requests
import time
import sys

# Agent S2 API configuration
API_BASE_URL = "http://localhost:4113"

def check_health():
    """Check if Agent S2 is running and healthy"""
    try:
        response = requests.get(f"{API_BASE_URL}/health", timeout=5)
        if response.ok:
            health = response.json()
            print(f"✅ Agent S2 is healthy")
            print(f"   Display: {health.get('display')}")
            print(f"   Screen: {health.get('screen_size', {}).get('width')}x{health.get('screen_size', {}).get('height')}")
            return True
        else:
            print(f"❌ Agent S2 health check failed: {response.status_code}")
            return False
    except requests.exceptions.RequestException as e:
        print(f"❌ Cannot connect to Agent S2: {e}")
        print("   Make sure Agent S2 is running: ./manage.sh --action start")
        return False

def take_screenshot(filename="screenshot.png"):
    """Take a screenshot"""
    print(f"\n📸 Taking screenshot...")
    
    response = requests.post(
        f"{API_BASE_URL}/screenshot",
        json={"format": "png", "quality": 95}
    )
    
    if response.ok:
        data = response.json()
        print(f"✅ Screenshot captured: {data['size']['width']}x{data['size']['height']}")
        
        # Save screenshot (base64 data would need to be decoded)
        # This is just a demonstration
        print(f"   (Screenshot data available in response)")
        return True
    else:
        print(f"❌ Screenshot failed: {response.text}")
        return False

def move_mouse(x, y):
    """Move mouse to position"""
    print(f"\n🖱️  Moving mouse to ({x}, {y})...")
    
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "mouse_move",
            "parameters": {"x": x, "y": y, "duration": 0.5}
        }
    )
    
    if response.ok:
        result = response.json()
        print(f"✅ Mouse moved to position")
        return True
    else:
        print(f"❌ Mouse move failed: {response.text}")
        return False

def click_at(x, y):
    """Click at position"""
    print(f"\n👆 Clicking at ({x}, {y})...")
    
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "click",
            "parameters": {"x": x, "y": y, "button": "left", "clicks": 1}
        }
    )
    
    if response.ok:
        print(f"✅ Clicked successfully")
        return True
    else:
        print(f"❌ Click failed: {response.text}")
        return False

def type_text(text):
    """Type text"""
    print(f"\n⌨️  Typing: '{text}'...")
    
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "type_text",
            "parameters": {"text": text, "interval": 0.05}
        }
    )
    
    if response.ok:
        print(f"✅ Text typed successfully")
        return True
    else:
        print(f"❌ Type text failed: {response.text}")
        return False

def automation_sequence():
    """Run a complete automation sequence"""
    print(f"\n🤖 Running automation sequence...")
    
    sequence = {
        "task_type": "automation_sequence",
        "parameters": {
            "steps": [
                {"type": "mouse_move", "parameters": {"x": 960, "y": 540}},
                {"type": "wait", "parameters": {"seconds": 1}},
                {"type": "click", "parameters": {}},
                {"type": "wait", "parameters": {"seconds": 0.5}},
                {"type": "type_text", "parameters": {"text": "Hello from Agent S2!"}},
                {"type": "wait", "parameters": {"seconds": 0.5}},
                {"type": "key_press", "parameters": {"keys": ["Return"]}},
                {"type": "wait", "parameters": {"seconds": 1}},
                {"type": "type_text", "parameters": {"text": f"Automated at {time.strftime('%Y-%m-%d %H:%M:%S')}"}}
            ]
        }
    }
    
    response = requests.post(f"{API_BASE_URL}/execute", json=sequence)
    
    if response.ok:
        result = response.json()
        print(f"✅ Automation sequence completed")
        print(f"   Task ID: {result.get('task_id')}")
        print(f"   Status: {result.get('status')}")
        return True
    else:
        print(f"❌ Automation sequence failed: {response.text}")
        return False

def main():
    """Main demonstration function"""
    print("=" * 60)
    print("Agent S2 Basic Automation Example")
    print("=" * 60)
    
    # Check health
    if not check_health():
        sys.exit(1)
    
    print("\nThis demo will perform the following actions:")
    print("1. Take a screenshot")
    print("2. Move the mouse")
    print("3. Click at a position")
    print("4. Type some text")
    print("5. Run an automation sequence")
    print("\n⚠️  Watch the VNC display to see the automation in action!")
    print("   VNC URL: vnc://localhost:5900")
    print("   Password: agents2vnc")
    
    input("\nPress Enter to start the demonstration...")
    
    # Run demonstrations
    demos = [
        lambda: take_screenshot(),
        lambda: move_mouse(500, 300),
        lambda: click_at(500, 300),
        lambda: type_text("Agent S2 Demo"),
        lambda: automation_sequence()
    ]
    
    for i, demo in enumerate(demos, 1):
        print(f"\n--- Demo {i}/{len(demos)} ---")
        demo()
        time.sleep(2)
    
    print("\n" + "=" * 60)
    print("✅ Demonstration completed!")
    print("\nTo learn more about Agent S2 capabilities, try:")
    print("  ./manage.sh --action usage --usage-type all")

if __name__ == "__main__":
    main()