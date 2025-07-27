#!/usr/bin/env python3
"""
Simple Automation - Basic mouse and keyboard control

This example demonstrates basic automation by:
1. Taking a screenshot
2. Moving the mouse
3. Typing some text
"""

from agent_s2.client import AgentS2Client, ScreenshotClient, AutomationClient
import time

def main():
    print("Agent S2 - Simple Automation Example")
    print("====================================")
    
    # Create clients
    client = AgentS2Client()
    screenshot = ScreenshotClient(client)
    automation = AutomationClient(client)
    
    # Check health first
    if not client.health_check():
        print("❌ Agent S2 is not running. Start it with: ./manage.sh --action start")
        return
    
    print("✅ Agent S2 is running")
    
    # Step 1: Take initial screenshot
    print("\n1. Taking initial screenshot...")
    screenshot.save("before_automation.png", directory="../../testing/test-outputs/screenshots")
    print("   ✅ Saved: before_automation.png")
    
    # Step 2: Get current mouse position
    print("\n2. Getting mouse position...")
    x, y = automation.get_mouse_position()
    print(f"   📍 Current position: ({x}, {y})")
    
    # Step 3: Move mouse to center of screen
    print("\n3. Moving mouse to center of screen...")
    screen_info = screenshot.client.screenshot()
    center_x = screen_info['size']['width'] // 2
    center_y = screen_info['size']['height'] // 2
    automation.move_mouse(center_x, center_y, duration=1.0)
    print(f"   ✅ Moved to: ({center_x}, {center_y})")
    
    # Step 4: Type some text
    print("\n4. Typing text...")
    automation.type_text("Hello from Agent S2!", interval=0.1)
    print("   ✅ Typed: 'Hello from Agent S2!'")
    
    # Step 5: Take final screenshot
    print("\n5. Taking final screenshot...")
    time.sleep(0.5)  # Brief pause to ensure text is visible
    screenshot.save("after_automation.png", directory="../../testing/test-outputs/screenshots")
    print("   ✅ Saved: after_automation.png")
    
    print("\n🎉 Automation complete!")
    print("   Check the screenshots in testing/test-outputs/screenshots/")

if __name__ == "__main__":
    main()