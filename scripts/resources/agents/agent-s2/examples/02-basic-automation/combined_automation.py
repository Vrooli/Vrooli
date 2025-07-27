#!/usr/bin/env python3
"""
Combined Automation - Real-world automation example

This example shows how to combine mouse and keyboard actions
to perform a realistic task.
"""

from agent_s2.client import AutomationClient, ScreenshotClient
import time

def main():
    print("Agent S2 - Combined Automation Example")
    print("======================================")
    
    # Create clients
    automation = AutomationClient()
    screenshot = ScreenshotClient()
    
    print("\nThis example will demonstrate a typical automation workflow:")
    print("1. Open a text area (simulated)")
    print("2. Type a message")
    print("3. Use keyboard shortcuts")
    print("4. Save the work")
    
    input("\nPress Enter to start the automation...")
    
    # Take initial screenshot
    print("\n📸 Taking initial screenshot...")
    screenshot.save("combined_start.png", directory="../../testing/test-outputs/screenshots")
    
    # Step 1: Click in the center of screen (where a text area might be)
    screen_info = screenshot.client.screenshot()
    center_x = screen_info['size']['width'] // 2
    center_y = screen_info['size']['height'] // 2
    
    print(f"\n1️⃣ Clicking at center ({center_x}, {center_y})...")
    automation.click(center_x, center_y)
    time.sleep(0.5)
    
    # Step 2: Clear any existing text
    print("\n2️⃣ Clearing existing text (Ctrl+A)...")
    automation.select_all()
    time.sleep(0.2)
    
    # Step 3: Type a message
    print("\n3️⃣ Typing a message...")
    message = """Agent S2 Automation Demo
    
This text was typed automatically using Agent S2.
It can handle:
- Regular text input
- Multiple lines
- Special characters (!@#$%^&*)
- And precise timing control

Time: {}
""".format(time.strftime("%Y-%m-%d %H:%M:%S"))
    
    automation.type_text(message, interval=0.05)
    time.sleep(0.5)
    
    # Step 4: Select all text
    print("\n4️⃣ Selecting all text (Ctrl+A)...")
    automation.select_all()
    time.sleep(0.5)
    
    # Step 5: Copy the text
    print("\n5️⃣ Copying text (Ctrl+C)...")
    automation.copy()
    time.sleep(0.2)
    
    # Step 6: Move to a new location
    new_x = center_x + 200
    new_y = center_y + 100
    print(f"\n6️⃣ Moving to new location ({new_x}, {new_y})...")
    automation.move_mouse(new_x, new_y, duration=1.0)
    automation.click()
    time.sleep(0.5)
    
    # Step 7: Paste the text
    print("\n7️⃣ Pasting text (Ctrl+V)...")
    automation.paste()
    time.sleep(0.5)
    
    # Step 8: Save (simulate)
    print("\n8️⃣ Saving work (Ctrl+S)...")
    automation.save()
    time.sleep(0.5)
    
    # Take final screenshot
    print("\n📸 Taking final screenshot...")
    screenshot.save("combined_end.png", directory="../../testing/test-outputs/screenshots")
    
    print("\n✅ Automation complete!")
    print("\nWhat we demonstrated:")
    print("- Mouse positioning and clicking")
    print("- Text selection and manipulation")
    print("- Keyboard shortcuts (Ctrl+A, Ctrl+C, Ctrl+V, Ctrl+S)")
    print("- Timed sequences of actions")
    print("- Screenshot capture for verification")

if __name__ == "__main__":
    main()