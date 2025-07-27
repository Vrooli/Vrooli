#!/usr/bin/env python3
"""
Core Automation: Keyboard Input Examples
Demonstrates keyboard control capabilities without AI
"""

import requests
import time

API_BASE_URL = "http://localhost:4113"

def type_text(text, interval=0.05):
    """Type text with specified interval between characters"""
    print(f"⌨️  Typing: '{text}' (interval: {interval}s)...")
    
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "type_text",
            "parameters": {"text": text, "interval": interval}
        }
    )
    
    return response.ok

def press_key(key):
    """Press a single key"""
    print(f"⌨️  Pressing key: {key}")
    
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "key_press",
            "parameters": {"keys": [key]}
        }
    )
    
    return response.ok

def key_combination(keys):
    """Press multiple keys simultaneously (e.g., Ctrl+C)"""
    print(f"⌨️  Key combination: {'+'.join(keys)}")
    
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "key_press",
            "parameters": {"keys": keys}
        }
    )
    
    return response.ok

def demonstrate_typing_speeds():
    """Show different typing speeds"""
    print("\n⌨️  Demonstrating typing speeds...")
    
    speeds = [
        ("Very fast", 0.01),
        ("Fast", 0.05),
        ("Normal", 0.1),
        ("Slow", 0.2),
        ("Very slow", 0.5)
    ]
    
    for name, interval in speeds:
        print(f"\n   {name} typing (interval: {interval}s):")
        type_text(f"This is {name.lower()} typing speed.", interval)
        time.sleep(1)
        press_key("Return")

def demonstrate_special_keys():
    """Demonstrate special key presses"""
    print("\n⌨️  Demonstrating special keys...")
    
    special_keys = [
        ("Tab", "Tab"),
        ("Enter", "Return"),
        ("Space", "space"),
        ("Escape", "Escape"),
        ("Backspace", "BackSpace"),
        ("Delete", "Delete"),
        ("Home", "Home"),
        ("End", "End"),
        ("Page Up", "Page_Up"),
        ("Page Down", "Page_Down"),
        ("Arrow Up", "Up"),
        ("Arrow Down", "Down"),
        ("Arrow Left", "Left"),
        ("Arrow Right", "Right")
    ]
    
    for name, key in special_keys:
        print(f"\n   Pressing {name}")
        press_key(key)
        time.sleep(0.5)

def demonstrate_modifiers():
    """Demonstrate modifier key combinations"""
    print("\n⌨️  Demonstrating modifier combinations...")
    
    combinations = [
        ("Select all", ["ctrl", "a"]),
        ("Copy", ["ctrl", "c"]),
        ("Paste", ["ctrl", "v"]),
        ("Cut", ["ctrl", "x"]),
        ("Undo", ["ctrl", "z"]),
        ("Redo", ["ctrl", "y"]),
        ("Save", ["ctrl", "s"]),
        ("Find", ["ctrl", "f"]),
        ("New", ["ctrl", "n"]),
        ("Open", ["ctrl", "o"]),
        ("Close", ["alt", "F4"])
    ]
    
    for name, keys in combinations:
        print(f"\n   {name}: {'+'.join(keys)}")
        key_combination(keys)
        time.sleep(0.5)

def type_multiline_text():
    """Type multi-line text"""
    print("\n⌨️  Typing multi-line text...")
    
    lines = [
        "This is the first line of text.",
        "This is the second line.",
        "And here's the third line!",
        "",
        "After a blank line, here's more text."
    ]
    
    for line in lines:
        if line:
            type_text(line, 0.05)
        press_key("Return")
        time.sleep(0.2)

def type_code_snippet():
    """Type a code snippet with proper formatting"""
    print("\n⌨️  Typing code snippet...")
    
    code = '''def hello_world():
    """Print hello world"""
    print("Hello from Agent S2!")
    return True

# Call the function
result = hello_world()
print(f"Success: {result}")'''
    
    lines = code.split('\n')
    for line in lines:
        type_text(line, 0.03)
        press_key("Return")
        time.sleep(0.1)

def demonstrate_text_editing():
    """Demonstrate text editing operations"""
    print("\n⌨️  Demonstrating text editing...")
    
    # Type initial text
    type_text("This is some example text for editing", 0.05)
    time.sleep(1)
    
    # Select all
    print("   Selecting all text")
    key_combination(["ctrl", "a"])
    time.sleep(1)
    
    # Delete and type new text
    print("   Deleting and typing new text")
    press_key("Delete")
    type_text("This is the new text!", 0.05)
    time.sleep(1)
    
    # Move cursor to beginning
    print("   Moving to beginning")
    press_key("Home")
    time.sleep(0.5)
    
    # Type at beginning
    type_text("START: ", 0.05)
    time.sleep(0.5)
    
    # Move to end
    print("   Moving to end")
    press_key("End")
    time.sleep(0.5)
    
    # Type at end
    type_text(" :END", 0.05)

def type_special_characters():
    """Type special characters and symbols"""
    print("\n⌨️  Typing special characters...")
    
    special_chars = [
        "!@#$%^&*()",
        "[]{}()<>",
        "+-*/=",
        ".,;:'\"`~",
        "\\|/?",
        "αβγδε",  # Greek letters
        "½¼¾",    # Fractions
        "©®™",    # Symbols
        "♠♣♥♦",   # Card suits
        "✓✗✨✭"    # Check marks and stars
    ]
    
    for chars in special_chars:
        print(f"   Typing: {chars}")
        type_text(chars, 0.1)
        press_key("Return")
        time.sleep(0.5)

def demonstrate_function_keys():
    """Demonstrate function key presses"""
    print("\n⌨️  Demonstrating function keys...")
    
    for i in range(1, 13):
        print(f"   Pressing F{i}")
        press_key(f"F{i}")
        time.sleep(0.3)

def create_formatted_document():
    """Create a formatted document using keyboard shortcuts"""
    print("\n⌨️  Creating formatted document...")
    
    # Title
    type_text("Agent S2 Keyboard Demo", 0.05)
    press_key("Return")
    press_key("Return")
    
    # Bold text (assuming text editor supports Ctrl+B)
    key_combination(["ctrl", "b"])
    type_text("Bold Text Section", 0.05)
    key_combination(["ctrl", "b"])  # Toggle off
    press_key("Return")
    press_key("Return")
    
    # Regular paragraph
    type_text("This demonstrates the keyboard automation capabilities of Agent S2. ", 0.03)
    type_text("We can type at different speeds, use special keys, and create formatted documents.", 0.03)
    press_key("Return")
    press_key("Return")
    
    # Bullet points
    type_text("Key features:", 0.05)
    press_key("Return")
    type_text("- Fast and accurate typing", 0.05)
    press_key("Return")
    type_text("- Support for special keys", 0.05)
    press_key("Return")
    type_text("- Keyboard shortcuts", 0.05)
    press_key("Return")
    type_text("- Multi-language support", 0.05)

def main():
    print("=" * 60)
    print("Agent S2 Core Automation: Keyboard Input Examples")
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
    print("\n⚠️  Make sure a text editor is open in the VNC display!")
    print("   VNC URL: vnc://localhost:5900")
    print("   Password: agents2vnc")
    
    input("\nPress Enter when ready to start keyboard demonstrations...")
    
    # Run examples
    examples = [
        ("Basic Text Typing", lambda: type_text("Hello from Agent S2!", 0.1)),
        ("Typing Speeds Demo", demonstrate_typing_speeds),
        ("Special Keys", demonstrate_special_keys),
        ("Modifier Combinations", demonstrate_modifiers),
        ("Multi-line Text", type_multiline_text),
        ("Code Snippet", type_code_snippet),
        ("Text Editing", demonstrate_text_editing),
        ("Special Characters", type_special_characters),
        ("Function Keys", demonstrate_function_keys),
        ("Formatted Document", create_formatted_document)
    ]
    
    for name, func in examples:
        print(f"\n{'='*40}")
        print(f"Example: {name}")
        print('='*40)
        
        try:
            func()
            time.sleep(2)  # Pause between examples
        except Exception as e:
            print(f"❌ Error: {e}")
    
    print("\n" + "="*60)
    print("✅ Keyboard input examples completed!")
    print("="*60)

if __name__ == "__main__":
    main()