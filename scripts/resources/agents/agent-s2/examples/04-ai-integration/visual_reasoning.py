#!/usr/bin/env python3
"""
AI-Driven: Visual Reasoning and Screen Understanding
Demonstrates AI's ability to understand and interact with visual elements
"""

import requests
import json
import time
import base64
from PIL import Image
import io

API_BASE_URL = "http://localhost:4113"

def analyze_screen(question=None):
    """Use AI to analyze current screen content"""
    print(f"\n👁️  Analyzing screen...")
    if question:
        print(f"   Question: {question}")
    
    request_data = {}
    if question:
        request_data["question"] = question
    
    response = requests.post(
        f"{API_BASE_URL}/analyze-screen",
        json=request_data,
        timeout=30
    )
    
    if response.ok:
        result = response.json()
        print("✅ Screen analysis completed")
        
        if result.get('analysis'):
            print(f"\n📋 Analysis:\n{result['analysis']}")
        
        if result.get('elements_found'):
            print("\n🔍 Elements found:")
            for element in result['elements_found']:
                print(f"   - {element}")
        
        return result
    else:
        print(f"❌ Screen analysis failed: {response.text}")
        return None

def visual_element_interaction():
    """AI identifies and interacts with visual elements"""
    print("\n" + "="*50)
    print("Visual Element Interaction")
    print("="*50)
    
    tasks = [
        # Basic element identification
        {
            "analyze": "What applications or windows are currently visible?",
            "command": "Click on the largest window or application",
            "context": "Focus on the most prominent element"
        },
        
        # Text recognition
        {
            "analyze": "Can you see any text on the screen?",
            "command": "Click on the first piece of text you can identify",
            "context": "Look for readable text elements"
        },
        
        # Button/UI element detection
        {
            "analyze": "Are there any buttons or clickable elements visible?",
            "command": "Click on a button that looks important",
            "context": "Identify UI controls like buttons, links, or icons"
        },
        
        # Color-based interaction
        {
            "analyze": "What are the dominant colors on the screen?",
            "command": "Click on something that is blue or has a blue background",
            "context": "Use color as a visual cue"
        }
    ]
    
    for task in tasks:
        # First analyze
        print(f"\n🔍 Analysis phase...")
        analyze_screen(task["analyze"])
        time.sleep(2)
        
        # Then interact based on analysis
        input(f"\nPress Enter to execute: '{task['command']}'")
        
        response = requests.post(
            f"{API_BASE_URL}/ai/command",
            json={
                "command": task["command"],
                "context": task["context"]
            }
        )
        
        if response.ok:
            print("✅ Visual interaction completed")
        
        time.sleep(2)

def icon_and_symbol_recognition():
    """AI recognizes and interacts with icons and symbols"""
    print("\n" + "="*50)
    print("Icon and Symbol Recognition")
    print("="*50)
    
    icon_tasks = [
        {
            "command": "Find and click on a folder icon if one is visible",
            "context": "Look for typical folder symbols"
        },
        {
            "command": "Locate and click on a close button (X) if present",
            "context": "Look for window control buttons"
        },
        {
            "command": "Find and click on a settings or gear icon",
            "context": "Look for configuration symbols"
        },
        {
            "command": "Click on any icon that looks like it's for files or documents",
            "context": "Identify file-related symbols"
        }
    ]
    
    for task in icon_tasks:
        input(f"\nPress Enter to find icon: '{task['command']}'")
        
        response = requests.post(
            f"{API_BASE_URL}/ai/command",
            json={
                "command": task["command"],
                "context": task["context"]
            }
        )
        
        if response.ok:
            print("✅ Icon interaction completed")
        time.sleep(2)

def spatial_reasoning_demo():
    """Demonstrate AI's spatial understanding"""
    print("\n" + "="*50)
    print("Spatial Reasoning Examples")
    print("="*50)
    
    spatial_tasks = [
        {
            "command": "Click on the item in the top-right corner of the screen",
            "context": "Use spatial positioning"
        },
        {
            "command": "Find the center of the largest empty area and click there",
            "context": "Identify negative space"
        },
        {
            "command": "Click on something that's between two other elements",
            "context": "Understand relative positioning"
        },
        {
            "command": "Draw a line from the top-left to bottom-right of the screen",
            "context": "Demonstrate diagonal movement across screen space"
        },
        {
            "command": "Click on elements in clockwise order starting from the top",
            "context": "Follow a circular pattern"
        }
    ]
    
    for task in spatial_tasks:
        input(f"\nPress Enter for spatial task: '{task['command']}'")
        
        response = requests.post(
            f"{API_BASE_URL}/ai/command",
            json={
                "command": task["command"],
                "context": task["context"]
            }
        )
        
        if response.ok:
            print("✅ Spatial reasoning task completed")
        time.sleep(2)

def dynamic_scene_adaptation():
    """AI adapts to changing screen content"""
    print("\n" + "="*50)
    print("Dynamic Scene Adaptation")
    print("="*50)
    
    print("This demo shows how AI adapts to screen changes.")
    
    # Take initial screenshot and analyze
    print("\n📸 Taking initial screenshot...")
    analyze_screen("Describe the current state of the screen in detail")
    
    input("\nNow change something on the screen (open/close window, move items)")
    
    # Analyze changes
    print("\n🔄 Analyzing changes...")
    analyze_screen("What has changed on the screen compared to before?")
    
    # Adapt to changes
    adaptation_commands = [
        {
            "command": "Click on something that wasn't there before",
            "context": "Identify new elements"
        },
        {
            "command": "If a window was closed, try to reopen it",
            "context": "Respond to removed elements"
        },
        {
            "command": "Interact with whatever changed most significantly",
            "context": "Focus on the biggest difference"
        }
    ]
    
    for cmd in adaptation_commands:
        input(f"\nPress Enter to adapt: '{cmd['command']}'")
        
        response = requests.post(
            f"{API_BASE_URL}/ai/command",
            json={
                "command": cmd["command"],
                "context": cmd["context"]
            }
        )
        
        if response.ok:
            print("✅ Adaptation completed")
        time.sleep(2)

def visual_search_and_find():
    """AI searches for specific visual elements"""
    print("\n" + "="*50)
    print("Visual Search and Find")
    print("="*50)
    
    search_targets = [
        "a search box or input field",
        "the word 'File' or 'Edit'",
        "anything that looks clickable",
        "an image or icon",
        "the smallest visible element",
        "something with rounded corners",
        "a menu or dropdown",
        "a checkbox or radio button"
    ]
    
    for target in search_targets:
        input(f"\nPress Enter to search for: {target}")
        
        # First analyze to find it
        analysis = analyze_screen(f"Can you find {target} on the screen?")
        time.sleep(1)
        
        # Then interact with it
        response = requests.post(
            f"{API_BASE_URL}/ai/command",
            json={
                "command": f"Click on {target} if you found it",
                "context": "Based on your visual analysis"
            }
        )
        
        if response.ok:
            print("✅ Visual search completed")
        time.sleep(2)

def screenshot_comparison_demo():
    """Compare screenshots to detect differences"""
    print("\n" + "="*50)
    print("Screenshot Comparison Demo")
    print("="*50)
    
    # Take first screenshot
    print("📸 Taking reference screenshot...")
    response1 = requests.post(f"{API_BASE_URL}/screenshot", json={"format": "png"})
    if response1.ok:
        print("✅ Reference screenshot captured")
    
    # Analyze initial state
    analyze_screen("Remember this screen layout for comparison")
    
    # Make some changes
    print("\n🎯 Making automated changes...")
    requests.post(
        f"{API_BASE_URL}/ai/command",
        json={
            "command": "Move the mouse around and click somewhere randomly",
            "context": "Create some visual changes"
        }
    )
    time.sleep(2)
    
    # Take second screenshot
    print("\n📸 Taking comparison screenshot...")
    response2 = requests.post(f"{API_BASE_URL}/screenshot", json={"format": "png"})
    if response2.ok:
        print("✅ Comparison screenshot captured")
    
    # Analyze differences
    print("\n🔍 Analyzing differences...")
    analyze_screen("What changed between the two screenshots?")
    
    # Act on differences
    response = requests.post(
        f"{API_BASE_URL}/ai/command",
        json={
            "command": "Click on the area that changed the most",
            "context": "Based on the screenshot comparison"
        }
    )
    
    if response.ok:
        print("✅ Acted on visual differences")

def pattern_recognition_demo():
    """AI recognizes and continues patterns"""
    print("\n" + "="*50)
    print("Pattern Recognition Demo")
    print("="*50)
    
    patterns = [
        {
            "setup": "I'll click in three corners of the screen. Continue the pattern.",
            "command": "Complete the pattern by clicking in the fourth corner",
            "context": "Recognize the rectangular pattern"
        },
        {
            "setup": "I'll type: A, B, C",
            "command": "Continue the alphabetical pattern with the next three letters",
            "context": "Recognize the sequence pattern"
        },
        {
            "setup": "I'll click: top, middle, bottom",
            "command": "Repeat the vertical pattern from left to right",
            "context": "Recognize and transpose the pattern"
        }
    ]
    
    for pattern in patterns:
        print(f"\n🎯 Pattern: {pattern['setup']}")
        
        # Set up the pattern (simplified - in real use this would be more complex)
        input("Press Enter after the pattern is demonstrated")
        
        # AI continues the pattern
        response = requests.post(
            f"{API_BASE_URL}/ai/command",
            json={
                "command": pattern["command"],
                "context": pattern["context"]
            }
        )
        
        if response.ok:
            print("✅ Pattern completed")
        time.sleep(2)

def main():
    print("=" * 70)
    print("Agent S2 AI-Driven: Visual Reasoning and Screen Understanding")
    print("=" * 70)
    print("\nThis example demonstrates AI's visual reasoning capabilities.")
    print("The AI can understand screen content and make intelligent decisions.")
    
    # Check AI status
    try:
        health = requests.get(f"{API_BASE_URL}/health", timeout=5)
        if not health.ok or not health.json().get('ai_status', {}).get('initialized'):
            print("\n❌ AI is not initialized. Please set up Ollama or another LLM provider.")
            return
    except:
        print("\n❌ Cannot connect to Agent S2")
        return
    
    print("\n✅ AI visual reasoning ready!")
    print("\n⚠️  Watch the VNC display to see AI visual understanding!")
    print("   VNC URL: vnc://localhost:5900")
    
    while True:
        print("\n" + "="*50)
        print("Choose a visual reasoning demo:")
        print("1. Visual Element Interaction")
        print("2. Icon and Symbol Recognition")
        print("3. Spatial Reasoning")
        print("4. Dynamic Scene Adaptation")
        print("5. Visual Search and Find")
        print("6. Screenshot Comparison")
        print("7. Pattern Recognition")
        print("0. Exit")
        
        choice = input("\nYour choice (0-7): ").strip()
        
        if choice == '0':
            break
        elif choice == '1':
            visual_element_interaction()
        elif choice == '2':
            icon_and_symbol_recognition()
        elif choice == '3':
            spatial_reasoning_demo()
        elif choice == '4':
            dynamic_scene_adaptation()
        elif choice == '5':
            visual_search_and_find()
        elif choice == '6':
            screenshot_comparison_demo()
        elif choice == '7':
            pattern_recognition_demo()
        else:
            print("Invalid choice. Please try again.")
    
    print("\n" + "="*70)
    print("✅ Visual reasoning examples completed!")
    print("\nKey Capabilities Demonstrated:")
    print("- Screen content analysis and understanding")
    print("- Visual element identification and interaction")
    print("- Spatial reasoning and positioning")
    print("- Pattern recognition and continuation")
    print("- Dynamic adaptation to screen changes")
    print("="*70)

if __name__ == "__main__":
    main()