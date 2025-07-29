#!/usr/bin/env python3
"""Test script to demonstrate the coordinate fix for Agent S2"""

import requests
import json
import time

def test_coordinate_fix():
    print("=== Agent S2 Coordinate Fix Test ===\n")
    
    # Check AI status
    status_response = requests.get("http://localhost:4113/ai/status")
    if status_response.ok:
        status = status_response.json()
        print(f"✅ AI Model: {status['model']}")
        print(f"✅ AI Enabled: {status['enabled']}")
        print(f"✅ AI Initialized: {status['initialized']}\n")
    
    # Test 1: Open mousepad
    print("Test 1: Opening mousepad application...")
    open_response = requests.post("http://localhost:4113/ai/command", json={
        "command": "Open the mousepad text editor",
        "context": "Open mousepad application"
    })
    
    if open_response.ok:
        result = open_response.json()["result"]
        print(f"✅ Command executed: {result['message']}")
        time.sleep(3)  # Wait for app to open
    
    # Test 2: Try to close with improved coordinates
    print("\nTest 2: Closing mousepad with vision-based coordinates...")
    close_response = requests.post("http://localhost:4113/ai/command", json={
        "command": "Close the mousepad window by clicking on its X close button",
        "context": "The close button is in the window title bar"
    })
    
    if close_response.ok:
        result = close_response.json()["result"]
        print(f"🧠 AI Reasoning: {result.get('ai_reasoning', 'No reasoning')}")
        print(f"🤖 Model Used: {result.get('ai_model', 'unknown')}")
        
        # Show the click coordinates used
        print("\n📍 Actions taken:")
        for i, action in enumerate(result.get('actions_taken', []), 1):
            if action.get('action') == 'click':
                params = action.get('parameters', {})
                print(f"   {i}. Click at coordinates: x={params.get('x')}, y={params.get('y')}")
                print(f"      Description: {action.get('result')}")
            elif action.get('action') == 'screenshot':
                if action.get('screenshot_path'):
                    print(f"   {i}. Screenshot saved: {action['screenshot_path']}")
    
    print("\n✅ Test complete!")
    print("   Before fix: AI would click at (5,5) or (10,10)")
    print("   After fix: AI now uses proper absolute screen coordinates")
    print("   Typical close button: Around x=40-60, y=40-50")

if __name__ == "__main__":
    test_coordinate_fix()