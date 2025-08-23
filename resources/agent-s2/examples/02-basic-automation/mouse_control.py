#!/usr/bin/env python3
"""
Core Automation: Mouse Control Examples
Demonstrates precise mouse control capabilities without AI
"""

import requests
import time
import math

API_BASE_URL = "http://localhost:4113"

def get_screen_size():
    """Get screen dimensions"""
    response = requests.get(f"{API_BASE_URL}/health")
    if response.ok:
        data = response.json()
        return data["screen_size"]["width"], data["screen_size"]["height"]
    return 1920, 1080  # Default

def move_to_position(x, y, duration=0.5):
    """Move mouse to specific position"""
    print(f"🖱️  Moving to ({x}, {y}) over {duration}s...")
    
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "mouse_move",
            "parameters": {"x": x, "y": y, "duration": duration}
        }
    )
    
    return response.ok

def click_at_position(x, y, button="left", clicks=1):
    """Click at specific position"""
    print(f"👆 Clicking {button} button {clicks}x at ({x}, {y})...")
    
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "click",
            "parameters": {
                "x": x, 
                "y": y, 
                "button": button, 
                "clicks": clicks
            }
        }
    )
    
    return response.ok

def drag_from_to(start_x, start_y, end_x, end_y, duration=1.0):
    """Drag from one position to another"""
    print(f"🖱️  Dragging from ({start_x}, {start_y}) to ({end_x}, {end_y})...")
    
    # Move to start position
    move_to_position(start_x, start_y, 0.5)
    time.sleep(0.5)
    
    # Perform drag
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "drag",
            "parameters": {
                "start_x": start_x,
                "start_y": start_y,
                "end_x": end_x,
                "end_y": end_y,
                "duration": duration
            }
        }
    )
    
    return response.ok

def draw_square(center_x, center_y, size=200):
    """Draw a square pattern with mouse"""
    print(f"🖱️  Drawing {size}x{size} square at ({center_x}, {center_y})...")
    
    half_size = size // 2
    corners = [
        (center_x - half_size, center_y - half_size),  # Top-left
        (center_x + half_size, center_y - half_size),  # Top-right
        (center_x + half_size, center_y + half_size),  # Bottom-right
        (center_x - half_size, center_y + half_size),  # Bottom-left
        (center_x - half_size, center_y - half_size),  # Back to start
    ]
    
    # Move to starting position
    move_to_position(corners[0][0], corners[0][1], 0.5)
    time.sleep(0.5)
    
    # Draw the square
    for i in range(1, len(corners)):
        move_to_position(corners[i][0], corners[i][1], 0.5)
        time.sleep(0.1)

def draw_circle(center_x, center_y, radius=100, steps=36):
    """Draw a circle pattern with mouse"""
    print(f"🖱️  Drawing circle with radius {radius} at ({center_x}, {center_y})...")
    
    # Calculate circle points
    for i in range(steps + 1):
        angle = (i / steps) * 2 * math.pi
        x = center_x + int(radius * math.cos(angle))
        y = center_y + int(radius * math.sin(angle))
        
        move_to_position(x, y, 0.1)
        time.sleep(0.05)

def demonstrate_all_buttons():
    """Demonstrate all mouse button clicks"""
    width, height = get_screen_size()
    center_x, center_y = width // 2, height // 2
    
    print("\n🖱️  Demonstrating all mouse buttons...")
    
    # Left click
    click_at_position(center_x - 100, center_y, "left", 1)
    time.sleep(1)
    
    # Right click
    click_at_position(center_x, center_y, "right", 1)
    time.sleep(1)
    
    # Middle click
    click_at_position(center_x + 100, center_y, "middle", 1)
    time.sleep(1)
    
    # Double click
    click_at_position(center_x, center_y + 100, "left", 2)

def mouse_hover_test():
    """Test mouse hovering over different areas"""
    width, height = get_screen_size()
    
    print("\n🖱️  Testing mouse hover positions...")
    
    positions = [
        ("Top-left corner", 50, 50),
        ("Top-right corner", width - 50, 50),
        ("Center", width // 2, height // 2),
        ("Bottom-left corner", 50, height - 50),
        ("Bottom-right corner", width - 50, height - 50),
    ]
    
    for name, x, y in positions:
        print(f"   Hovering at {name}")
        move_to_position(x, y, 0.8)
        time.sleep(1)

def smooth_movement_demo():
    """Demonstrate smooth mouse movements"""
    width, height = get_screen_size()
    center_x, center_y = width // 2, height // 2
    
    print("\n🖱️  Demonstrating smooth movements...")
    
    # Fast movement
    print("   Fast movement (0.2s)")
    move_to_position(center_x - 200, center_y, 0.2)
    time.sleep(0.5)
    
    # Normal movement
    print("   Normal movement (0.5s)")
    move_to_position(center_x, center_y, 0.5)
    time.sleep(0.5)
    
    # Slow movement
    print("   Slow movement (2.0s)")
    move_to_position(center_x + 200, center_y, 2.0)

def get_current_position():
    """Get and display current mouse position"""
    response = requests.get(f"{API_BASE_URL}/mouse/position")
    
    if response.ok:
        pos = response.json()
        print(f"📍 Current mouse position: ({pos['x']}, {pos['y']})")
        return pos['x'], pos['y']
    return None, None

def precision_test():
    """Test mouse positioning precision"""
    print("\n🎯 Testing mouse precision...")
    
    test_positions = [
        (100, 100),
        (500, 500),
        (1000, 700),
        (1500, 900)
    ]
    
    for target_x, target_y in test_positions:
        # Move to position
        move_to_position(target_x, target_y, 0.3)
        time.sleep(0.5)
        
        # Check actual position
        actual_x, actual_y = get_current_position()
        if actual_x and actual_y:
            error = math.sqrt((actual_x - target_x)**2 + (actual_y - target_y)**2)
            print(f"   Target: ({target_x}, {target_y}), Actual: ({actual_x}, {actual_y}), Error: {error:.2f}px")

def main():
    print("=" * 60)
    print("Agent S2 Core Automation: Mouse Control Examples")
    print("=" * 60)
    
    # Check service health
    try:
        health = requests.get(f"{API_BASE_URL}/health", timeout=5)
        if not health.ok:
            print("❌ Agent S2 is not healthy")
            return
    except:
        print("❌ Cannot connect to Agent S2")
        print("   Start it with: ./scripts/resources/agents/agent-s2/manage.sh --action start")
        return
    
    print("✅ Agent S2 is running")
    print("\n⚠️  Watch the VNC display to see mouse movements!")
    print("   VNC URL: vnc://localhost:5900")
    print("   Password: agents2vnc")
    
    # Get screen info
    width, height = get_screen_size()
    print(f"\n📺 Screen size: {width}x{height}")
    
    # Run examples
    examples = [
        ("Get Current Position", get_current_position),
        ("Mouse Hover Test", mouse_hover_test),
        ("All Button Types", demonstrate_all_buttons),
        ("Smooth Movement Demo", smooth_movement_demo),
        ("Draw Square Pattern", lambda: draw_square(width//2, height//2, 300)),
        ("Draw Circle Pattern", lambda: draw_circle(width//2, height//2, 150)),
        ("Drag Operation", lambda: drag_from_to(500, 300, 800, 600)),
        ("Precision Test", precision_test)
    ]
    
    for name, func in examples:
        print(f"\n{'='*40}")
        print(f"Example: {name}")
        print('='*40)
        
        try:
            func()
            time.sleep(1)  # Pause between examples
        except Exception as e:
            print(f"❌ Error: {e}")
    
    print("\n" + "="*60)
    print("✅ Mouse control examples completed!")
    print("="*60)

if __name__ == "__main__":
    main()