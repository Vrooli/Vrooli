#!/usr/bin/env python3
"""
Adaptive Automation - AI that adapts to different scenarios

This example shows how AI can handle variations and unexpected
situations in automation tasks.
"""

from agent_s2.client import AIClient, ScreenshotClient
import time
import json

def main():
    print("Agent S2 - Adaptive Automation Example")
    print("=====================================")
    
    # Create clients
    ai = AIClient()
    screenshot = ScreenshotClient()
    
    print("\nThis example demonstrates AI adapting to different scenarios.")
    print("The AI will handle variations without explicit programming.\n")
    
    # Scenario 1: Find and click a button regardless of position
    print("Scenario 1: Adaptive Button Clicking")
    print("-" * 40)
    
    # Take a screenshot for context
    screenshot.save("adaptive_start.png", directory="../../data/test-outputs/screenshots")
    
    # Ask AI to find and click a button with various descriptions
    button_variations = [
        "Click the primary action button",
        "Click the most prominent button on screen",
        "Find and click the submit or continue button",
        "Click whatever button seems most important"
    ]
    
    for i, instruction in enumerate(button_variations, 1):
        print(f"\nAttempt {i}: {instruction}")
        
        try:
            result = ai.perform_task(instruction)
            print(f"✅ AI Response: {result.get('summary', 'Action completed')}")
            
            # Give different context each time
            context = {
                "attempt": i,
                "previous_instruction": button_variations[i-2] if i > 1 else None,
                "user_preference": "looking for positive action"
            }
            
        except Exception as e:
            print(f"❌ Failed: {e}")
            
        time.sleep(1)
    
    # Scenario 2: Intelligent form filling
    print("\n\nScenario 2: Adaptive Form Filling")
    print("-" * 40)
    
    # AI can adapt to different form layouts
    form_scenarios = [
        {
            "task": "Fill in a contact form with realistic test data",
            "constraints": ["Use example.com email", "Use safe test data"]
        },
        {
            "task": "Find any input fields and fill them appropriately",
            "constraints": ["Detect field types", "Use contextual data"]
        },
        {
            "task": "Complete whatever form is visible on screen",
            "constraints": ["Skip optional fields", "Focus on required fields"]
        }
    ]
    
    for scenario in form_scenarios:
        print(f"\nTask: {scenario['task']}")
        print(f"Constraints: {', '.join(scenario['constraints'])}")
        
        try:
            # Analyze current screen
            analysis = ai.analyze_screen()
            print(f"Screen analysis: Found {len(analysis.get('elements_detected', []))} elements")
            
            # Perform adaptive task
            result = ai.perform_task(
                scenario['task'],
                context={"constraints": scenario['constraints']}
            )
            
            print(f"✅ Completed: {result.get('summary', 'Task done')}")
            
        except Exception as e:
            print(f"❌ Failed: {e}")
            
        time.sleep(2)
    
    # Scenario 3: Recovery from unexpected states
    print("\n\nScenario 3: Error Recovery and Adaptation")
    print("-" * 40)
    
    recovery_scenarios = [
        "If there's an error dialog, close it and retry the last action",
        "Navigate back to a known good state if you're lost",
        "Handle any popups or notifications that appear",
        "Recover from whatever state the application is currently in"
    ]
    
    for scenario in recovery_scenarios:
        print(f"\n🔄 {scenario}")
        
        try:
            # First, check current state
            state_check = ai.verify_state("in a normal, usable state")
            
            if not state_check:
                print("❗ Detected unusual state")
                
                # Ask AI to recover
                recovery_result = ai.perform_task(scenario)
                print(f"✅ Recovery: {recovery_result.get('summary', 'Recovered')}")
            else:
                print("✅ System in good state, no recovery needed")
                
        except Exception as e:
            print(f"❌ Recovery failed: {e}")
            
        time.sleep(1)
    
    # Scenario 4: Learning and improving
    print("\n\nScenario 4: Contextual Learning")
    print("-" * 40)
    
    # Simulate a sequence where AI learns from previous actions
    learning_sequence = [
        ("Find the menu button", None),
        ("Now find a similar button", "Use what you learned from the previous task"),
        ("Find all buttons like the first one", "Apply the pattern you've identified"),
        ("Click the next logical button in the sequence", "Predict based on the pattern")
    ]
    
    learned_context = {}
    
    for task, hint in learning_sequence:
        print(f"\nTask: {task}")
        if hint:
            print(f"Hint: {hint}")
            
        try:
            # Include learning context
            result = ai.perform_task(
                task,
                context={
                    "learned": learned_context,
                    "hint": hint
                }
            )
            
            # Simulate AI "learning" by storing information
            if 'actions_taken' in result:
                learned_context[f"step_{len(learned_context)}"] = {
                    "task": task,
                    "result": result.get('summary', ''),
                    "pattern": "button recognition improved"
                }
                
            print(f"✅ Result: {result.get('summary', 'Completed')}")
            print(f"📚 Learned: {len(learned_context)} patterns")
            
        except Exception as e:
            print(f"❌ Failed: {e}")
            
        time.sleep(1)
    
    # Final screenshot
    screenshot.save("adaptive_end.png", directory="../../data/test-outputs/screenshots")
    
    print("\n\n✅ Adaptive Automation Complete!")
    print("\nKey Concepts Demonstrated:")
    print("1. **Flexibility** - AI handles variations in instructions")
    print("2. **Context Awareness** - Uses screen content to make decisions")
    print("3. **Error Recovery** - Adapts to unexpected situations")
    print("4. **Learning** - Improves performance over time")
    print("\nThe AI adapted to:")
    print(f"- {len(button_variations)} different button descriptions")
    print(f"- {len(form_scenarios)} form filling scenarios")
    print(f"- {len(recovery_scenarios)} recovery situations")
    print(f"- {len(learned_context)} learned patterns")

if __name__ == "__main__":
    main()