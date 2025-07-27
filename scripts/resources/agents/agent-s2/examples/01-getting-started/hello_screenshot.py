#!/usr/bin/env python3
"""
Hello Screenshot - Your first Agent S2 example

This is the simplest possible example - it takes a screenshot
and saves it to a file.
"""

from agent_s2.client import ScreenshotClient

def main():
    print("Agent S2 - Hello Screenshot Example")
    print("===================================")
    
    # Create a screenshot client
    screenshot = ScreenshotClient()
    
    # Take a screenshot and save it
    print("Taking a screenshot...")
    filename = screenshot.save(
        filename="hello_screenshot.png",
        directory="../../testing/test-outputs/screenshots"
    )
    
    print(f"✅ Screenshot saved to: {filename}")
    print("\nThat's it! You've taken your first screenshot with Agent S2.")

if __name__ == "__main__":
    main()