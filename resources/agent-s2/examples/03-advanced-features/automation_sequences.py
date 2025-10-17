#!/usr/bin/env python3
"""
Core Automation: Complex Automation Sequences
Demonstrates multi-step automation workflows without AI
"""

import requests
import time
import json

API_BASE_URL = "http://localhost:4113"

def execute_sequence(steps, name="Automation Sequence"):
    """Execute a sequence of automation steps"""
    print(f"\n🤖 Executing: {name}")
    print(f"   Steps: {len(steps)}")
    
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "automation_sequence",
            "parameters": {"steps": steps},
            "async_execution": False
        }
    )
    
    if response.ok:
        result = response.json()
        print(f"✅ Sequence completed")
        print(f"   Task ID: {result.get('task_id')}")
        print(f"   Status: {result.get('status')}")
        return True
    else:
        print(f"❌ Sequence failed: {response.text}")
        return False

def open_application_sequence():
    """Sequence to open an application (simulated)"""
    steps = [
        # Move to desktop area
        {"type": "mouse_move", "parameters": {"x": 100, "y": 100, "duration": 0.5}},
        {"type": "wait", "parameters": {"seconds": 0.5}},
        
        # Right-click for context menu
        {"type": "click", "parameters": {"button": "right"}},
        {"type": "wait", "parameters": {"seconds": 1}},
        
        # Type to search for terminal
        {"type": "type_text", "parameters": {"text": "terminal", "interval": 0.05}},
        {"type": "wait", "parameters": {"seconds": 0.5}},
        
        # Press Enter to select
        {"type": "key_press", "parameters": {"keys": ["Return"]}},
        {"type": "wait", "parameters": {"seconds": 2}},
        
        # Type a command
        {"type": "type_text", "parameters": {"text": "echo 'Agent S2 Automation!'", "interval": 0.03}},
        {"type": "key_press", "parameters": {"keys": ["Return"]}}
    ]
    
    return execute_sequence(steps, "Open Application")

def create_document_sequence():
    """Sequence to create and save a document"""
    steps = [
        # Open new document (Ctrl+N)
        {"type": "key_press", "parameters": {"keys": ["ctrl", "n"]}},
        {"type": "wait", "parameters": {"seconds": 1}},
        
        # Type document title
        {"type": "type_text", "parameters": {"text": "Agent S2 Automation Report", "interval": 0.05}},
        {"type": "key_press", "parameters": {"keys": ["Return", "Return"]}},
        
        # Type document content
        {"type": "type_text", "parameters": {
            "text": "Date: " + time.strftime("%Y-%m-%d %H:%M:%S"), 
            "interval": 0.03
        }},
        {"type": "key_press", "parameters": {"keys": ["Return", "Return"]}},
        
        {"type": "type_text", "parameters": {
            "text": "This document was created automatically by Agent S2.", 
            "interval": 0.03
        }},
        {"type": "key_press", "parameters": {"keys": ["Return"]}},
        
        {"type": "type_text", "parameters": {
            "text": "Automation sequences can create complex workflows.", 
            "interval": 0.03
        }},
        {"type": "key_press", "parameters": {"keys": ["Return", "Return"]}},
        
        # Save document (Ctrl+S)
        {"type": "key_press", "parameters": {"keys": ["ctrl", "s"]}},
        {"type": "wait", "parameters": {"seconds": 1}},
        
        # Type filename
        {"type": "type_text", "parameters": {"text": "agent_s2_report.txt", "interval": 0.05}},
        {"type": "wait", "parameters": {"seconds": 0.5}},
        
        # Press Enter to save
        {"type": "key_press", "parameters": {"keys": ["Return"]}}
    ]
    
    return execute_sequence(steps, "Create Document")

def window_management_sequence():
    """Sequence for window management operations"""
    steps = [
        # Maximize window (Alt+F10 on many systems)
        {"type": "key_press", "parameters": {"keys": ["alt", "F10"]}},
        {"type": "wait", "parameters": {"seconds": 1}},
        
        # Move to center
        {"type": "mouse_move", "parameters": {"x": 960, "y": 540, "duration": 0.5}},
        {"type": "wait", "parameters": {"seconds": 0.5}},
        
        # Minimize (Alt+F9)
        {"type": "key_press", "parameters": {"keys": ["alt", "F9"]}},
        {"type": "wait", "parameters": {"seconds": 1}},
        
        # Click to restore
        {"type": "click", "parameters": {"x": 960, "y": 540}},
        {"type": "wait", "parameters": {"seconds": 1}},
        
        # Switch windows (Alt+Tab)
        {"type": "key_press", "parameters": {"keys": ["alt", "Tab"]}},
        {"type": "wait", "parameters": {"seconds": 1}}
    ]
    
    return execute_sequence(steps, "Window Management")

def data_entry_sequence():
    """Sequence for form data entry"""
    form_data = [
        ("John Doe", "Tab"),
        ("john.doe@example.com", "Tab"),
        ("555-1234", "Tab"),
        ("123 Main St", "Tab"),
        ("Anytown", "Tab"),
        ("CA", "Tab"),
        ("90210", "Tab")
    ]
    
    steps = []
    for data, key in form_data:
        steps.extend([
            {"type": "type_text", "parameters": {"text": data, "interval": 0.03}},
            {"type": "key_press", "parameters": {"keys": [key]}},
            {"type": "wait", "parameters": {"seconds": 0.2}}
        ])
    
    # Submit form
    steps.extend([
        {"type": "wait", "parameters": {"seconds": 0.5}},
        {"type": "key_press", "parameters": {"keys": ["Return"]}}
    ])
    
    return execute_sequence(steps, "Data Entry Form")

def copy_paste_sequence():
    """Sequence demonstrating copy and paste operations"""
    steps = [
        # Type source text
        {"type": "type_text", "parameters": {
            "text": "This text will be copied and pasted multiple times.", 
            "interval": 0.03
        }},
        {"type": "wait", "parameters": {"seconds": 0.5}},
        
        # Select all (Ctrl+A)
        {"type": "key_press", "parameters": {"keys": ["ctrl", "a"]}},
        {"type": "wait", "parameters": {"seconds": 0.5}},
        
        # Copy (Ctrl+C)
        {"type": "key_press", "parameters": {"keys": ["ctrl", "c"]}},
        {"type": "wait", "parameters": {"seconds": 0.5}},
        
        # Move to end and new line
        {"type": "key_press", "parameters": {"keys": ["End"]}},
        {"type": "key_press", "parameters": {"keys": ["Return", "Return"]}},
        
        # Paste multiple times
        {"type": "key_press", "parameters": {"keys": ["ctrl", "v"]}},
        {"type": "key_press", "parameters": {"keys": ["Return"]}},
        {"type": "wait", "parameters": {"seconds": 0.3}},
        
        {"type": "key_press", "parameters": {"keys": ["ctrl", "v"]}},
        {"type": "key_press", "parameters": {"keys": ["Return"]}},
        {"type": "wait", "parameters": {"seconds": 0.3}},
        
        {"type": "key_press", "parameters": {"keys": ["ctrl", "v"]}},
        {"type": "key_press", "parameters": {"keys": ["Return"]}}
    ]
    
    return execute_sequence(steps, "Copy and Paste")

def navigation_sequence():
    """Sequence for text/document navigation"""
    steps = [
        # Type some sample text
        {"type": "type_text", "parameters": {
            "text": "Line 1: Beginning of document", 
            "interval": 0.03
        }},
        {"type": "key_press", "parameters": {"keys": ["Return"]}},
        
        {"type": "type_text", "parameters": {
            "text": "Line 2: Some content here", 
            "interval": 0.03
        }},
        {"type": "key_press", "parameters": {"keys": ["Return"]}},
        
        {"type": "type_text", "parameters": {
            "text": "Line 3: More content", 
            "interval": 0.03
        }},
        {"type": "key_press", "parameters": {"keys": ["Return"]}},
        
        {"type": "type_text", "parameters": {
            "text": "Line 4: End of document", 
            "interval": 0.03
        }},
        {"type": "wait", "parameters": {"seconds": 1}},
        
        # Navigate to beginning (Ctrl+Home)
        {"type": "key_press", "parameters": {"keys": ["ctrl", "Home"]}},
        {"type": "wait", "parameters": {"seconds": 0.5}},
        
        # Navigate word by word (Ctrl+Right)
        {"type": "key_press", "parameters": {"keys": ["ctrl", "Right"]}},
        {"type": "wait", "parameters": {"seconds": 0.3}},
        {"type": "key_press", "parameters": {"keys": ["ctrl", "Right"]}},
        {"type": "wait", "parameters": {"seconds": 0.3}},
        
        # Navigate to end (Ctrl+End)
        {"type": "key_press", "parameters": {"keys": ["ctrl", "End"]}},
        {"type": "wait", "parameters": {"seconds": 0.5}},
        
        # Page up
        {"type": "key_press", "parameters": {"keys": ["Page_Up"]}},
        {"type": "wait", "parameters": {"seconds": 0.5}}
    ]
    
    return execute_sequence(steps, "Document Navigation")

def screenshot_annotation_sequence():
    """Sequence to take screenshot and annotate it"""
    steps = [
        # Take a screenshot first
        {"type": "screenshot", "parameters": {"format": "png"}},
        {"type": "wait", "parameters": {"seconds": 1}},
        
        # Move to annotation position
        {"type": "mouse_move", "parameters": {"x": 500, "y": 300, "duration": 0.5}},
        {"type": "wait", "parameters": {"seconds": 0.5}},
        
        # Click to start annotation
        {"type": "click", "parameters": {}},
        {"type": "wait", "parameters": {"seconds": 0.5}},
        
        # Type annotation
        {"type": "type_text", "parameters": {
            "text": "Important: This area shows the automation target", 
            "interval": 0.03
        }},
        {"type": "wait", "parameters": {"seconds": 1}},
        
        # Draw arrow (drag)
        {"type": "drag", "parameters": {
            "start_x": 600, "start_y": 350,
            "end_x": 700, "end_y": 400,
            "duration": 0.5
        }},
        {"type": "wait", "parameters": {"seconds": 0.5}},
        
        # Take final screenshot
        {"type": "screenshot", "parameters": {"format": "png"}}
    ]
    
    return execute_sequence(steps, "Screenshot Annotation")

def loop_demonstration():
    """Demonstrate repetitive tasks"""
    steps = []
    
    # Create a pattern by repeating actions
    for i in range(5):
        steps.extend([
            {"type": "type_text", "parameters": {
                "text": f"Iteration {i + 1}: ", 
                "interval": 0.03
            }},
            {"type": "type_text", "parameters": {
                "text": "Automated task completed", 
                "interval": 0.03
            }},
            {"type": "key_press", "parameters": {"keys": ["Return"]}},
            {"type": "wait", "parameters": {"seconds": 0.3}}
        ])
    
    return execute_sequence(steps, "Loop Demonstration")

def conditional_sequence():
    """Simulate conditional logic in sequences"""
    # This demonstrates how you might structure conditional workflows
    # In real use, you'd have application logic determining which branch to take
    
    # Branch A: Morning workflow
    morning_steps = [
        {"type": "type_text", "parameters": {"text": "Good morning! Starting daily tasks...", "interval": 0.03}},
        {"type": "key_press", "parameters": {"keys": ["Return"]}},
        {"type": "type_text", "parameters": {"text": "1. Checking emails", "interval": 0.03}},
        {"type": "key_press", "parameters": {"keys": ["Return"]}},
        {"type": "type_text", "parameters": {"text": "2. Reviewing calendar", "interval": 0.03}},
        {"type": "key_press", "parameters": {"keys": ["Return"]}}
    ]
    
    # Branch B: Evening workflow
    evening_steps = [
        {"type": "type_text", "parameters": {"text": "Good evening! Wrapping up tasks...", "interval": 0.03}},
        {"type": "key_press", "parameters": {"keys": ["Return"]}},
        {"type": "type_text", "parameters": {"text": "1. Saving work", "interval": 0.03}},
        {"type": "key_press", "parameters": {"keys": ["Return"]}},
        {"type": "type_text", "parameters": {"text": "2. Backing up files", "interval": 0.03}},
        {"type": "key_press", "parameters": {"keys": ["Return"]}}
    ]
    
    # Choose based on time
    import datetime
    current_hour = datetime.datetime.now().hour
    
    if current_hour < 12:
        return execute_sequence(morning_steps, "Morning Workflow")
    else:
        return execute_sequence(evening_steps, "Evening Workflow")

def main():
    print("=" * 60)
    print("Agent S2 Core Automation: Complex Sequences")
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
    print("\n⚠️  Watch the VNC display to see automation in action!")
    print("   VNC URL: vnc://localhost:5900")
    print("   Password: agents2vnc")
    
    print("\nThese examples demonstrate complex multi-step workflows.")
    print("Each sequence combines mouse, keyboard, and timing operations.")
    
    input("\nPress Enter to start sequence demonstrations...")
    
    # Run examples
    examples = [
        ("Copy and Paste Operations", copy_paste_sequence),
        ("Document Navigation", navigation_sequence),
        ("Data Entry Form", data_entry_sequence),
        ("Loop Demonstration", loop_demonstration),
        ("Conditional Workflow", conditional_sequence),
        ("Create Document", create_document_sequence),
        ("Window Management", window_management_sequence),
        ("Screenshot Annotation", screenshot_annotation_sequence)
    ]
    
    for name, func in examples:
        print(f"\n{'='*40}")
        print(f"Example: {name}")
        print('='*40)
        
        try:
            result = func()
            if result:
                print("✅ Sequence completed successfully")
            time.sleep(3)  # Pause between sequences
        except Exception as e:
            print(f"❌ Error: {e}")
    
    print("\n" + "="*60)
    print("✅ Automation sequence examples completed!")
    print("\nKey Takeaways:")
    print("- Sequences can combine any automation primitives")
    print("- Timing control with wait steps is crucial")
    print("- Complex workflows can be built from simple steps")
    print("- Conditional logic can be implemented in application code")
    print("="*60)

if __name__ == "__main__":
    main()