#!/usr/bin/env python3
"""
Agent S2 Automation Examples
Demonstrates both AI-driven and core automation capabilities
"""

import requests
import time
import sys
import json

# Agent S2 API configuration
API_BASE_URL = "http://localhost:4113"

def check_health():
    """Check if Agent S2 is running and healthy"""
    try:
        response = requests.get(f"{API_BASE_URL}/health", timeout=5)
        if response.ok:
            health = response.json()
            ai_status = health.get('ai_status', {})
            
            print(f"✅ Agent S2 is healthy")
            print(f"   Display: {health.get('display')}")
            print(f"   Screen: {health.get('screen_size', {}).get('width')}x{health.get('screen_size', {}).get('height')}")
            print(f"   AI Available: {'✅' if ai_status.get('available') else '❌'}")
            print(f"   AI Enabled: {'✅' if ai_status.get('enabled') else '❌'}")
            print(f"   AI Initialized: {'✅' if ai_status.get('initialized') else '❌'}")
            
            if ai_status.get('initialized'):
                print(f"   AI Provider: {ai_status.get('provider', 'unknown')}")
                print(f"   AI Model: {ai_status.get('model', 'unknown')}")
            
            return True, ai_status.get('initialized', False)
        else:
            print(f"❌ Agent S2 health check failed: {response.status_code}")
            return False, False
    except requests.exceptions.RequestException as e:
        print(f"❌ Cannot connect to Agent S2: {e}")
        print("   Make sure Agent S2 is running: ./manage.sh --action start")
        return False, False

def take_screenshot(filename="screenshot.png"):
    """Take a screenshot"""
    print(f"\n📸 Taking screenshot...")
    
    response = requests.post(
        f"{API_BASE_URL}/screenshot?format=png&quality=95",
        json=None
    )
    
    if response.ok:
        data = response.json()
        print(f"✅ Screenshot captured: {data['size']['width']}x{data['size']['height']}")
        
        # Save screenshot (base64 data would need to be decoded)
        # This is just a demonstration
        print(f"   (Screenshot data available in response)")
        return True
    else:
        print(f"❌ Screenshot failed: {response.text}")
        return False

def move_mouse(x, y):
    """Move mouse to position"""
    print(f"\n🖱️  Moving mouse to ({x}, {y})...")
    
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "mouse_move",
            "parameters": {"x": x, "y": y, "duration": 0.5}
        }
    )
    
    if response.ok:
        result = response.json()
        print(f"✅ Mouse moved to position")
        return True
    else:
        print(f"❌ Mouse move failed: {response.text}")
        return False

def click_at(x, y):
    """Click at position"""
    print(f"\n👆 Clicking at ({x}, {y})...")
    
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "click",
            "parameters": {"x": x, "y": y, "button": "left", "clicks": 1}
        }
    )
    
    if response.ok:
        print(f"✅ Clicked successfully")
        return True
    else:
        print(f"❌ Click failed: {response.text}")
        return False

def type_text(text):
    """Type text"""
    print(f"\n⌨️  Typing: '{text}'...")
    
    response = requests.post(
        f"{API_BASE_URL}/execute",
        json={
            "task_type": "type_text",
            "parameters": {"text": text, "interval": 0.05}
        }
    )
    
    if response.ok:
        print(f"✅ Text typed successfully")
        return True
    else:
        print(f"❌ Type text failed: {response.text}")
        return False

def automation_sequence():
    """Run a complete automation sequence"""
    print(f"\n🤖 Running automation sequence...")
    
    sequence = {
        "task_type": "automation_sequence",
        "parameters": {
            "steps": [
                {"type": "mouse_move", "parameters": {"x": 960, "y": 540}},
                {"type": "wait", "parameters": {"seconds": 1}},
                {"type": "click", "parameters": {}},
                {"type": "wait", "parameters": {"seconds": 0.5}},
                {"type": "type_text", "parameters": {"text": "Hello from Agent S2!"}},
                {"type": "wait", "parameters": {"seconds": 0.5}},
                {"type": "key_press", "parameters": {"keys": ["Return"]}},
                {"type": "wait", "parameters": {"seconds": 1}},
                {"type": "type_text", "parameters": {"text": f"Automated at {time.strftime('%Y-%m-%d %H:%M:%S')}"}}
            ]
        }
    }
    
    response = requests.post(f"{API_BASE_URL}/execute", json=sequence)
    
    if response.ok:
        result = response.json()
        print(f"✅ Automation sequence completed")
        print(f"   Task ID: {result.get('task_id')}")
        print(f"   Status: {result.get('status')}")
        return True
    else:
        print(f"❌ Automation sequence failed: {response.text}")
        return False

# AI-Driven Examples
def ai_command_example():
    """Demonstrate natural language command execution"""
    print(f"\n🧠 AI Command Example...")
    print("   Using AI to understand and execute: 'take a screenshot and move mouse to center'")
    
    response = requests.post(
        f"{API_BASE_URL}/ai/command",
        json={
            "command": "take a screenshot and move mouse to center",
            "context": "demonstration of AI capabilities",
            "async_execution": False
        }
    )
    
    if response.ok:
        result = response.json()
        print(f"✅ AI command executed")
        print(f"   Task ID: {result.get('task_id')}")
        print(f"   Status: {result.get('status')}")
        if result.get('result'):
            print(f"   Message: {result.get('result', {}).get('message', '')}")
        return True
    else:
        print(f"❌ AI command failed: {response.text}")
        return False

def ai_planning_example():
    """Demonstrate AI planning capabilities"""
    print(f"\n🎯 AI Planning Example...")
    print("   Asking AI to plan: 'organize desktop workspace'")
    
    response = requests.post(
        f"{API_BASE_URL}/plan",
        json={
            "goal": "organize desktop workspace",
            "constraints": ["do not delete any files", "keep important apps visible"]
        }
    )
    
    if response.ok:
        plan = response.json()
        print(f"✅ AI plan generated")
        print(f"   Goal: {plan.get('goal')}")
        print(f"   Steps: {len(plan.get('steps', []))}")
        for step in plan.get('steps', [])[:3]:  # Show first 3 steps
            print(f"     {step.get('step')}. {step.get('action')}")
        return True
    else:
        print(f"❌ AI planning failed: {response.text}")
        return False

def ai_screen_analysis_example():
    """Demonstrate AI screen analysis"""
    print(f"\n👁️  AI Screen Analysis Example...")
    print("   Asking AI: 'What applications are currently visible?'")
    
    response = requests.post(
        f"{API_BASE_URL}/analyze-screen",
        json={
            "question": "What applications are currently visible on the screen?"
        }
    )
    
    if response.ok:
        analysis = response.json()
        print(f"✅ AI screen analysis completed")
        print(f"   Question: {analysis.get('question')}")
        print(f"   Analysis: {analysis.get('analysis')}")
        print(f"   Screen Size: {analysis.get('screen_size', {}).get('width')}x{analysis.get('screen_size', {}).get('height')}")
        return True
    else:
        print(f"❌ AI screen analysis failed: {response.text}")
        return False

def get_capabilities():
    """Get and display Agent S2 capabilities"""
    print(f"\n🔍 Checking Agent S2 capabilities...")
    
    response = requests.get(f"{API_BASE_URL}/capabilities")
    
    if response.ok:
        caps = response.json()
        capabilities = caps.get('capabilities', {})
        supported_tasks = caps.get('supported_tasks', [])
        
        print(f"✅ Agent S2 capabilities:")
        print(f"   AI Reasoning: {'✅' if capabilities.get('ai_reasoning') else '❌'}")
        print(f"   Natural Language: {'✅' if capabilities.get('natural_language') else '❌'}")
        print(f"   Screen Understanding: {'✅' if capabilities.get('screen_understanding') else '❌'}")
        print(f"   Planning: {'✅' if capabilities.get('planning') else '❌'}")
        print(f"   GUI Automation: {'✅' if capabilities.get('gui_automation') else '❌'}")
        
        ai_tasks = [task for task in supported_tasks if task.startswith('ai_')]
        core_tasks = [task for task in supported_tasks if not task.startswith('ai_')]
        
        print(f"   AI Tasks Available: {len(ai_tasks)} ({', '.join(ai_tasks)})")
        print(f"   Core Automation Tasks: {len(core_tasks)} ({', '.join(core_tasks[:5])}{'...' if len(core_tasks) > 5 else ''})")
        
        return True, len(ai_tasks) > 0
    else:
        print(f"❌ Failed to get capabilities: {response.text}")
        return False, False

def main():
    """Main demonstration function"""
    print("=" * 70)
    print("Agent S2 Automation Examples - AI & Core Automation")
    print("=" * 70)
    
    # Check health and AI status
    healthy, ai_available = check_health()
    if not healthy:
        sys.exit(1)
    
    # Get capabilities
    caps_ok, ai_tasks_available = get_capabilities()
    
    print("\n" + "=" * 70)
    print("DEMONSTRATION OPTIONS")
    print("=" * 70)
    
    print("\n🤖 AI-Driven Examples (requires AI initialization):")
    if ai_available and ai_tasks_available:
        print("   1. AI Natural Language Command")
        print("   2. AI Planning & Goal Achievement")
        print("   3. AI Screen Analysis & Understanding")
        ai_demos_available = True
    else:
        print("   ❌ AI examples not available")
        if not ai_available:
            print("      Reason: AI not initialized")
        if not ai_tasks_available:
            print("      Reason: No AI tasks supported")
        ai_demos_available = False
    
    print("\n⚙️  Core Automation Examples (always available):")
    print("   4. Take Screenshot")
    print("   5. Mouse Movement & Clicking")
    print("   6. Text Input")
    print("   7. Automation Sequence")
    
    print("\n⚠️  Watch the VNC display to see automation in action!")
    print("   VNC URL: vnc://localhost:5900")
    print("   Password: agents2vnc")
    
    # User choice
    if ai_demos_available:
        choice = input("\nRun [A]I examples, [C]ore automation examples, or [B]oth? (A/C/B): ").upper()
    else:
        choice = input("\nRun [C]ore automation examples? (C): ").upper()
        if choice not in ['C', '']:
            choice = 'C'
    
    # Prepare demonstrations
    ai_demos = []
    core_demos = []
    
    if ai_demos_available and choice in ['A', 'B']:
        ai_demos = [
            ("AI Natural Language Command", ai_command_example),
            ("AI Planning", ai_planning_example),
            ("AI Screen Analysis", ai_screen_analysis_example)
        ]
    
    if choice in ['C', 'B', '']:
        core_demos = [
            ("Screenshot Capture", take_screenshot),
            ("Mouse Movement", lambda: move_mouse(500, 300)),
            ("Mouse Click", lambda: click_at(500, 300)),
            ("Text Input", lambda: type_text("Agent S2 Demo")),
            ("Automation Sequence", automation_sequence)
        ]
    
    all_demos = ai_demos + core_demos
    
    if not all_demos:
        print("❌ No demonstrations available")
        return
    
    print(f"\n🚀 Starting demonstrations ({len(all_demos)} total)...")
    print("=" * 70)
    
    # Run demonstrations
    for i, (name, demo_func) in enumerate(all_demos, 1):
        print(f"\n--- Demo {i}/{len(all_demos)}: {name} ---")
        try:
            success = demo_func()
            if success:
                print(f"✅ {name} completed successfully")
            else:
                print(f"❌ {name} failed")
        except Exception as e:
            print(f"❌ {name} error: {e}")
        
        if i < len(all_demos):
            time.sleep(2)
    
    print("\n" + "=" * 70)
    print("✅ All demonstrations completed!")
    print("\n📚 Next Steps:")
    print("   • Explore more examples: ./manage.sh --action usage --usage-type all")
    print("   • Check API docs: http://localhost:4113/docs")
    print("   • View logs: ./manage.sh --action logs")
    
    if ai_available:
        print("\n🧠 Try your own AI commands:")
        print("   curl -X POST http://localhost:4113/ai/command \\")
        print("        -H 'Content-Type: application/json' \\")
        print("        -d '{\"command\": \"your natural language instruction here\"}'")
    
    print("=" * 70)

if __name__ == "__main__":
    main()