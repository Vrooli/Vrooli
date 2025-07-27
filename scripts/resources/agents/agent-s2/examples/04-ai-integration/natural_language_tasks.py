#!/usr/bin/env python3
"""
AI-Driven: Natural Language Task Execution
Uses local Ollama models to interpret and execute natural language commands
"""

import requests
import json
import time
import os

API_BASE_URL = "http://localhost:4113"

def check_ai_status():
    """Check if AI is available and properly configured"""
    try:
        response = requests.get(f"{API_BASE_URL}/health", timeout=5)
        if response.ok:
            health = response.json()
            ai_status = health.get('ai_status', {})
            
            print("🧠 AI Status:")
            print(f"   Available: {'✅' if ai_status.get('available') else '❌'}")
            print(f"   Enabled: {'✅' if ai_status.get('enabled') else '❌'}")
            print(f"   Initialized: {'✅' if ai_status.get('initialized') else '❌'}")
            
            if ai_status.get('initialized'):
                print(f"   Provider: {ai_status.get('provider', 'unknown')}")
                print(f"   Model: {ai_status.get('model', 'unknown')}")
                return True
            else:
                print("\n⚠️  AI not initialized. Checking Ollama setup...")
                check_ollama_setup()
                return False
        return False
    except Exception as e:
        print(f"❌ Error checking AI status: {e}")
        return False

def check_ollama_setup():
    """Provide guidance for Ollama setup"""
    print("\n📋 Ollama Setup Instructions:")
    print("1. Ensure Ollama is installed:")
    print("   curl -fsSL https://ollama.ai/install.sh | sh")
    print("\n2. Pull a suitable model:")
    print("   ollama pull llama2:7b")
    print("   ollama pull mistral:7b")
    print("   ollama pull codellama:7b")
    print("\n3. Start Agent S2 with Ollama provider:")
    print("   ./scripts/resources/agents/agent-s2/manage.sh --action install \\")
    print("     --llm-provider ollama \\")
    print("     --llm-model llama2:7b \\")
    print("     --enable-ai yes")

def execute_natural_language_command(command, context=None):
    """Execute a natural language command using AI"""
    print(f"\n🗣️  Command: '{command}'")
    if context:
        print(f"   Context: {context}")
    
    request_data = {
        "command": command,
        "async_execution": False
    }
    if context:
        request_data["context"] = context
    
    response = requests.post(
        f"{API_BASE_URL}/ai/command",
        json=request_data,
        timeout=30  # AI commands may take longer
    )
    
    if response.ok:
        result = response.json()
        print("✅ AI command executed successfully")
        
        # Show basic info
        if result.get('task_id'):
            print(f"   Task ID: {result['task_id']}")
        
        # Show execution time
        if result.get('created_at') and result.get('completed_at'):
            print(f"   Duration: {result['completed_at'][:19]} - {result['created_at'][:19]}")
        
        # Show AI reasoning (most important!)
        task_result = result.get('result', {})
        if task_result.get('ai_reasoning'):
            print(f"🧠 AI Reasoning:")
            print(f"   {task_result['ai_reasoning']}")
        
        # Show AI model used
        if task_result.get('ai_model'):
            print(f"🤖 Model: {task_result['ai_model']}")
        
        # Show detailed actions
        if task_result.get('actions_taken'):
            print(f"⚡ Actions Executed ({len(task_result['actions_taken'])}):")
            for i, action in enumerate(task_result['actions_taken'], 1):
                action_type = action.get('action', 'unknown')
                action_result = action.get('result', 'No description')
                status = action.get('status', 'unknown')
                
                status_icon = "✅" if status == "success" else "❌" if status == "failed" else "🔄"
                print(f"   {i}. {status_icon} {action_type.title()}: {action_result}")
                
                # Show parameters if available
                if action.get('parameters'):
                    params = action['parameters']
                    param_str = ", ".join([f"{k}={v}" for k, v in params.items()])
                    print(f"      Parameters: {param_str}")
                
                # Show screenshot path for any action that saves screenshots
                if action.get('screenshot_path'):
                    filename = action['screenshot_path'].split('/')[-1]  # Get just filename
                    print(f"      📸 Screenshot saved: {action['screenshot_path']}")
                    print(f"      📋 To copy from container: docker cp agent-s2:{action['screenshot_path']} ./{filename}")
                    print(f"      🖼️  To view via VNC: vnc://localhost:5900 (password: agents2vnc)")
        
        # Show overall message
        if task_result.get('message'):
            print(f"📝 Summary: {task_result['message']}")
        
        return True
    else:
        print(f"❌ AI command failed: {response.text}")
        return False

def demonstrate_basic_commands():
    """Demonstrate basic natural language commands"""
    print("\n" + "="*50)
    print("Basic Natural Language Commands")
    print("="*50)
    
    commands = [
        # Screenshot and visual tasks
        ("Take a screenshot", None),
        ("Take a screenshot and save it as desktop_capture.png", None),
        ("Capture the current screen and describe what you see", None),
        
        # Mouse control
        ("Move the mouse to the center of the screen", None),
        ("Click in the top-left corner of the screen", None),
        ("Move the mouse in a circle pattern", None),
        
        # Keyboard input
        ("Type 'Hello from AI-driven Agent S2'", None),
        ("Press the Enter key", None),
        ("Type today's date and time", None),
        
        # Complex actions
        ("Open a text editor and write a short poem about automation", 
         "We want to demonstrate creative text generation"),
        
        ("Take a screenshot, then move the mouse to any text you can see", 
         "Testing visual recognition capabilities")
    ]
    
    for command, context in commands:
        input(f"\nPress Enter to execute: '{command}'")
        execute_natural_language_command(command, context)
        time.sleep(2)

def demonstrate_multi_step_tasks():
    """Demonstrate complex multi-step tasks"""
    print("\n" + "="*50)
    print("Multi-Step Natural Language Tasks")
    print("="*50)
    
    tasks = [
        {
            "command": "Create a new text document with a todo list for today",
            "context": "Include at least 5 items that would be typical for a workday"
        },
        {
            "command": "Draw a simple smiley face using the mouse",
            "context": "Use smooth circular motions"
        },
        {
            "command": "Open a calculator and calculate 42 * 17 + 108",
            "context": "Show the step-by-step process"
        },
        {
            "command": "Organize the desktop by moving icons to the edges",
            "context": "Group similar items together if possible"
        }
    ]
    
    for task in tasks:
        input(f"\nPress Enter to execute multi-step task: '{task['command']}'")
        execute_natural_language_command(task['command'], task['context'])
        time.sleep(3)

def demonstrate_contextual_understanding():
    """Demonstrate AI's contextual understanding"""
    print("\n" + "="*50)
    print("Contextual Understanding Examples")
    print("="*50)
    
    # Sequential commands that build on each other
    sequence = [
        ("Take a screenshot and remember what's on the screen", 
         "We'll refer back to this"),
        
        ("Click on the largest visible area of empty space", 
         "Based on what you just saw"),
        
        ("Type a description of what was in the screenshot", 
         "Describe the main elements you observed"),
        
        ("Now click on something interesting you saw earlier", 
         "Choose based on the screenshot analysis")
    ]
    
    for command, context in sequence:
        input(f"\nPress Enter for contextual command: '{command}'")
        execute_natural_language_command(command, context)
        time.sleep(2)

def demonstrate_adaptive_behavior():
    """Demonstrate AI adapting to different scenarios"""
    print("\n" + "="*50)
    print("Adaptive Behavior Examples")
    print("="*50)
    
    scenarios = [
        {
            "command": "Find and click on any button or clickable element",
            "context": "Identify UI elements that look interactive"
        },
        {
            "command": "If there's a text field visible, type your name in it",
            "context": "Look for input fields or text areas"
        },
        {
            "command": "Navigate to the file menu if one exists, otherwise right-click",
            "context": "Adapt based on what's available"
        },
        {
            "command": "Close any open windows or dialogs",
            "context": "Look for close buttons or use keyboard shortcuts"
        }
    ]
    
    for scenario in scenarios:
        input(f"\nPress Enter for adaptive scenario: '{scenario['command']}'")
        execute_natural_language_command(scenario['command'], scenario['context'])
        time.sleep(2)

def demonstrate_creative_tasks():
    """Demonstrate creative and interpretive tasks"""
    print("\n" + "="*50)
    print("Creative Task Examples")
    print("="*50)
    
    creative_tasks = [
        {
            "command": "Draw a simple house using the mouse",
            "context": "Include a roof, door, and window"
        },
        {
            "command": "Write a haiku about computer automation",
            "context": "Follow the 5-7-5 syllable pattern"
        },
        {
            "command": "Create an ASCII art pattern",
            "context": "Use keyboard characters to make a simple design"
        },
        {
            "command": "Arrange desktop icons in an aesthetically pleasing way",
            "context": "Consider symmetry and visual balance"
        }
    ]
    
    for task in creative_tasks:
        input(f"\nPress Enter for creative task: '{task['command']}'")
        execute_natural_language_command(task['command'], task['context'])
        time.sleep(3)

def interactive_conversation_mode():
    """Interactive mode where user can type custom commands"""
    print("\n" + "="*50)
    print("Interactive AI Command Mode")
    print("="*50)
    print("Type natural language commands for the AI to execute.")
    print("Type 'exit' to return to main menu.\n")
    
    while True:
        command = input("🗣️  Your command: ").strip()
        
        if command.lower() == 'exit':
            break
        
        if command:
            context = input("   Context (optional): ").strip()
            execute_natural_language_command(
                command, 
                context if context else None
            )
            time.sleep(2)

def main():
    print("=" * 70)
    print("Agent S2 AI-Driven: Natural Language Task Execution")
    print("=" * 70)
    print("\nThis example demonstrates AI-driven automation using natural language.")
    print("The AI interprets your commands and uses core automation functions.")
    
    # Check if AI is available
    if not check_ai_status():
        print("\n❌ AI is not available. Please set up Ollama or another LLM provider.")
        return
    
    print("\n✅ AI is ready for natural language commands!")
    print("\n⚠️  Watch the VNC display to see AI-driven actions!")
    print("   VNC URL: vnc://localhost:5900")
    print("   Password: agents2vnc")
    
    while True:
        print("\n" + "="*50)
        print("Choose an example:")
        print("1. Basic Natural Language Commands")
        print("2. Multi-Step Tasks")
        print("3. Contextual Understanding")
        print("4. Adaptive Behavior")
        print("5. Creative Tasks")
        print("6. Interactive Mode (type your own commands)")
        print("0. Exit")
        
        choice = input("\nYour choice (0-6): ").strip()
        
        if choice == '0':
            break
        elif choice == '1':
            demonstrate_basic_commands()
        elif choice == '2':
            demonstrate_multi_step_tasks()
        elif choice == '3':
            demonstrate_contextual_understanding()
        elif choice == '4':
            demonstrate_adaptive_behavior()
        elif choice == '5':
            demonstrate_creative_tasks()
        elif choice == '6':
            interactive_conversation_mode()
        else:
            print("Invalid choice. Please try again.")
    
    print("\n" + "="*70)
    print("✅ Natural language task examples completed!")
    print("\nKey Insights:")
    print("- AI interprets natural language into executable actions")
    print("- Commands can be vague - AI figures out the details")
    print("- AI adapts to what it sees on screen")
    print("- Complex tasks are broken down automatically")
    print("="*70)

if __name__ == "__main__":
    main()