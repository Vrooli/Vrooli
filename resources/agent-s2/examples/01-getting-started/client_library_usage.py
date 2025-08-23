#!/usr/bin/env python3
"""Example usage of Agent S2 client library"""

from agent_s2.client import AgentS2Client, ScreenshotClient, AutomationClient, AIClient

def basic_usage():
    """Basic client usage example"""
    # Create base client
    client = AgentS2Client()
    
    # Check health
    if client.health_check():
        print("✅ Agent S2 is healthy")
    
    # Take a screenshot
    screenshot = client.screenshot(format="png")
    print(f"📸 Screenshot captured: {screenshot['size']}")
    
    # Get mouse position
    pos = client.mouse_position()
    print(f"🖱️  Mouse at: ({pos['x']}, {pos['y']})")

def screenshot_client_usage():
    """Screenshot client usage example"""
    screenshot = ScreenshotClient()
    
    # Capture full screen
    data = screenshot.capture_full_screen()
    print("📸 Full screen captured")
    
    # Capture region
    data = screenshot.capture_region(100, 100, 400, 300)
    print("📸 Region captured")
    
    # Save with auto-generated filename
    path = screenshot.save(directory="./screenshots")
    print(f"💾 Saved to: {path}")
    
    # Capture series
    series = screenshot.capture_series(count=3, interval=1.0)
    print(f"📸 Captured {len(series)} screenshots")

def automation_client_usage():
    """Automation client usage example"""
    automation = AutomationClient()
    
    # Get mouse position
    x, y = automation.get_mouse_position()
    print(f"🖱️  Mouse at: ({x}, {y})")
    
    # Move mouse
    automation.move_mouse(500, 300, duration=1.0)
    print("🖱️  Moved mouse to (500, 300)")
    
    # Click at position
    automation.click(600, 400)
    print("🖱️  Clicked at (600, 400)")
    
    # Type text
    automation.type_text("Hello, Agent S2!", interval=0.1)
    print("⌨️  Typed text")
    
    # Use keyboard shortcuts
    automation.select_all()
    automation.copy()
    print("⌨️  Selected all and copied")

def ai_client_usage():
    """AI client usage example"""
    ai = AIClient()
    
    # Analyze current screen
    analysis = ai.analyze_screen()
    print("🤖 Screen analyzed")
    
    # Perform a task
    result = ai.perform_task("Find the search box and type 'Agent S2 demo'")
    print(f"🤖 Task result: {result.get('summary', 'completed')}")
    
    # Smart click
    if ai.smart_click("the blue button"):
        print("🤖 Successfully clicked the blue button")
    
    # Suggest actions
    suggestions = ai.suggest_actions()
    print(f"🤖 Suggested actions: {suggestions[:3]}")

def context_manager_usage():
    """Context manager usage example"""
    with AgentS2Client() as client:
        # Client is automatically cleaned up after use
        screenshot = client.screenshot()
        print("📸 Screenshot captured with context manager")

if __name__ == "__main__":
    print("Agent S2 Client Library Examples")
    print("=" * 40)
    
    # Run examples
    examples = [
        ("Basic Usage", basic_usage),
        ("Screenshot Client", screenshot_client_usage),
        ("Automation Client", automation_client_usage),
        ("AI Client", ai_client_usage),
        ("Context Manager", context_manager_usage)
    ]
    
    for name, func in examples:
        print(f"\n{name}:")
        print("-" * len(name))
        try:
            func()
        except Exception as e:
            print(f"❌ Error: {e}")