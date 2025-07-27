#!/usr/bin/env python3
"""
AI-Driven: Autonomous Task Planning and Execution
Demonstrates AI's ability to plan and execute complex multi-step goals
"""

import requests
import json
import time
import threading

API_BASE_URL = "http://localhost:4113"

def create_plan(goal, constraints=None):
    """Ask AI to create a plan for achieving a goal"""
    print(f"\n🎯 Goal: {goal}")
    if constraints:
        print("📋 Constraints:")
        for constraint in constraints:
            print(f"   - {constraint}")
    
    request_data = {
        "goal": goal,
        "constraints": constraints or []
    }
    
    response = requests.post(
        f"{API_BASE_URL}/plan",
        json=request_data,
        timeout=30
    )
    
    if response.ok:
        plan = response.json()
        print("\n✅ Plan created successfully")
        
        if plan.get('steps'):
            print(f"\n📝 Plan ({len(plan['steps'])} steps):")
            for i, step in enumerate(plan['steps'], 1):
                print(f"   {i}. {step.get('action', 'Unknown action')}")
                if step.get('description'):
                    print(f"      {step['description']}")
        
        return plan
    else:
        print(f"❌ Planning failed: {response.text}")
        return None

def execute_plan(plan):
    """Execute a plan step by step"""
    if not plan or not plan.get('steps'):
        print("❌ No valid plan to execute")
        return False
    
    print(f"\n🚀 Executing plan: {plan.get('goal', 'Unknown goal')}")
    print(f"   Total steps: {len(plan['steps'])}")
    
    for i, step in enumerate(plan['steps'], 1):
        print(f"\n📍 Step {i}/{len(plan['steps'])}: {step.get('action', 'Unknown')}")
        
        # Execute the step
        response = requests.post(
            f"{API_BASE_URL}/execute/ai",
            json={
                "command": step.get('action'),
                "context": f"Step {i} of plan: {step.get('description', '')}"
            }
        )
        
        if response.ok:
            print(f"   ✅ Step {i} completed")
        else:
            print(f"   ❌ Step {i} failed: {response.text}")
            return False
        
        time.sleep(1)  # Pause between steps
    
    print("\n✅ Plan execution completed!")
    return True

def demonstrate_file_organization():
    """AI autonomously organizes files/desktop"""
    print("\n" + "="*50)
    print("Autonomous File Organization")
    print("="*50)
    
    goal = "Organize the desktop by grouping similar items together"
    constraints = [
        "Don't delete anything",
        "Create a logical arrangement",
        "Keep frequently used items easily accessible"
    ]
    
    plan = create_plan(goal, constraints)
    
    if plan:
        input("\nPress Enter to execute the organization plan")
        execute_plan(plan)

def demonstrate_document_creation():
    """AI autonomously creates a document"""
    print("\n" + "="*50)
    print("Autonomous Document Creation")
    print("="*50)
    
    goal = "Create a professional report about AI automation capabilities"
    constraints = [
        "Include a title, introduction, main points, and conclusion",
        "Make it well-formatted",
        "Save it with a descriptive filename"
    ]
    
    plan = create_plan(goal, constraints)
    
    if plan:
        input("\nPress Enter to execute the document creation plan")
        execute_plan(plan)

def demonstrate_research_task():
    """AI autonomously performs a research task"""
    print("\n" + "="*50)
    print("Autonomous Research Task")
    print("="*50)
    
    goal = "Research and document information about the current system"
    constraints = [
        "Gather information about open applications",
        "Note system time and date",
        "Create a summary of findings",
        "Be thorough but concise"
    ]
    
    plan = create_plan(goal, constraints)
    
    if plan:
        input("\nPress Enter to execute the research plan")
        execute_plan(plan)

def demonstrate_workflow_automation():
    """AI creates and executes a complete workflow"""
    print("\n" + "="*50)
    print("Workflow Automation")
    print("="*50)
    
    workflows = [
        {
            "name": "Email Workflow",
            "goal": "Simulate checking and responding to emails",
            "constraints": [
                "Open an email application or simulate one",
                "Check for new messages",
                "Compose a reply",
                "Include proper email etiquette"
            ]
        },
        {
            "name": "Data Entry Workflow",
            "goal": "Complete a data entry task with multiple fields",
            "constraints": [
                "Fill in all required fields",
                "Use realistic sample data",
                "Tab between fields properly",
                "Submit or save when complete"
            ]
        },
        {
            "name": "Testing Workflow",
            "goal": "Test UI elements on the current screen",
            "constraints": [
                "Click on different types of elements",
                "Test keyboard shortcuts",
                "Verify responses",
                "Document what works and what doesn't"
            ]
        }
    ]
    
    print("Available workflows:")
    for i, wf in enumerate(workflows, 1):
        print(f"{i}. {wf['name']}: {wf['goal']}")
    
    choice = input("\nSelect workflow (1-3): ").strip()
    
    if choice in ['1', '2', '3']:
        workflow = workflows[int(choice) - 1]
        plan = create_plan(workflow['goal'], workflow['constraints'])
        
        if plan:
            input(f"\nPress Enter to execute the {workflow['name']}")
            execute_plan(plan)

def demonstrate_adaptive_planning():
    """AI adapts its plan based on current conditions"""
    print("\n" + "="*50)
    print("Adaptive Planning")
    print("="*50)
    
    # First, analyze current state
    print("🔍 Analyzing current screen state...")
    response = requests.post(
        f"{API_BASE_URL}/analyze-screen",
        json={"question": "What applications and elements are currently visible?"}
    )
    
    if response.ok:
        analysis = response.json()
        print(f"Current state: {analysis.get('analysis', 'Unknown')[:200]}...")
    
    # Create adaptive goal based on what's available
    goal = "Interact with the available applications in a meaningful way"
    constraints = [
        "Work with what's currently on screen",
        "If no apps are open, open something useful",
        "Demonstrate different types of interactions",
        "Be creative but safe"
    ]
    
    plan = create_plan(goal, constraints)
    
    if plan:
        input("\nPress Enter to execute the adaptive plan")
        execute_plan(plan)

def demonstrate_error_recovery():
    """AI handles errors and recovers gracefully"""
    print("\n" + "="*50)
    print("Error Recovery Planning")
    print("="*50)
    
    goal = "Try to open a non-existent application and recover gracefully"
    constraints = [
        "Attempt to open 'SuperApp2000' (which doesn't exist)",
        "When it fails, find an alternative",
        "Document what went wrong",
        "Complete a useful task instead"
    ]
    
    plan = create_plan(goal, constraints)
    
    if plan:
        input("\nPress Enter to see error recovery in action")
        execute_plan(plan)

def demonstrate_parallel_tasks():
    """AI plans tasks that could be done in parallel"""
    print("\n" + "="*50)
    print("Parallel Task Planning")
    print("="*50)
    
    goal = "Set up multiple windows for different tasks simultaneously"
    constraints = [
        "Open at least 2-3 different applications",
        "Arrange them on screen efficiently",
        "Start a task in each one",
        "Switch between them to show multitasking"
    ]
    
    plan = create_plan(goal, constraints)
    
    if plan:
        input("\nPress Enter to execute parallel task setup")
        execute_plan(plan)

def interactive_goal_mode():
    """Let user define custom goals for AI to plan and execute"""
    print("\n" + "="*50)
    print("Interactive Goal Planning")
    print("="*50)
    print("Define your own goals for the AI to plan and execute.")
    print("Type 'exit' to return to main menu.\n")
    
    while True:
        goal = input("🎯 Your goal: ").strip()
        
        if goal.lower() == 'exit':
            break
        
        if goal:
            # Get constraints
            constraints = []
            print("📋 Add constraints (press Enter with empty line to finish):")
            while True:
                constraint = input("   - ").strip()
                if not constraint:
                    break
                constraints.append(constraint)
            
            # Create and optionally execute plan
            plan = create_plan(goal, constraints if constraints else None)
            
            if plan:
                execute = input("\nExecute this plan? (y/n): ").strip().lower()
                if execute == 'y':
                    execute_plan(plan)
            
            time.sleep(2)

def demonstrate_learning_behavior():
    """AI demonstrates learning from previous actions"""
    print("\n" + "="*50)
    print("Learning and Improvement")
    print("="*50)
    
    # First attempt
    print("📚 First attempt at a task...")
    goal1 = "Click on three different UI elements"
    plan1 = create_plan(goal1, ["Be quick but accurate"])
    
    if plan1:
        input("\nPress Enter for first attempt")
        execute_plan(plan1)
    
    # Second attempt with learning
    print("\n📚 Second attempt with improvements...")
    goal2 = "Click on three different UI elements more efficiently than before"
    constraints2 = [
        "Learn from the previous attempt",
        "Optimize the path between elements",
        "Minimize unnecessary movements"
    ]
    
    plan2 = create_plan(goal2, constraints2)
    
    if plan2:
        input("\nPress Enter for improved attempt")
        execute_plan(plan2)

def main():
    print("=" * 70)
    print("Agent S2 AI-Driven: Autonomous Planning and Execution")
    print("=" * 70)
    print("\nThis example demonstrates AI's autonomous planning capabilities.")
    print("The AI creates multi-step plans and executes them independently.")
    
    # Check AI status
    try:
        health = requests.get(f"{API_BASE_URL}/health", timeout=5)
        if not health.ok or not health.json().get('ai_status', {}).get('initialized'):
            print("\n❌ AI is not initialized. Please set up Ollama or another LLM provider.")
            print("\nTo use Ollama:")
            print("1. Install: curl -fsSL https://ollama.ai/install.sh | sh")
            print("2. Pull model: ollama pull llama2:7b")
            print("3. Restart Agent S2 with: --llm-provider ollama --llm-model llama2:7b")
            return
    except:
        print("\n❌ Cannot connect to Agent S2")
        return
    
    print("\n✅ AI planning system ready!")
    print("\n⚠️  Watch the VNC display to see autonomous execution!")
    print("   VNC URL: vnc://localhost:5900")
    
    while True:
        print("\n" + "="*50)
        print("Choose a planning demonstration:")
        print("1. File/Desktop Organization")
        print("2. Document Creation")
        print("3. Research Task")
        print("4. Workflow Automation")
        print("5. Adaptive Planning")
        print("6. Error Recovery")
        print("7. Parallel Tasks")
        print("8. Learning Behavior")
        print("9. Interactive Mode (custom goals)")
        print("0. Exit")
        
        choice = input("\nYour choice (0-9): ").strip()
        
        if choice == '0':
            break
        elif choice == '1':
            demonstrate_file_organization()
        elif choice == '2':
            demonstrate_document_creation()
        elif choice == '3':
            demonstrate_research_task()
        elif choice == '4':
            demonstrate_workflow_automation()
        elif choice == '5':
            demonstrate_adaptive_planning()
        elif choice == '6':
            demonstrate_error_recovery()
        elif choice == '7':
            demonstrate_parallel_tasks()
        elif choice == '8':
            demonstrate_learning_behavior()
        elif choice == '9':
            interactive_goal_mode()
        else:
            print("Invalid choice. Please try again.")
    
    print("\n" + "="*70)
    print("✅ Autonomous planning examples completed!")
    print("\nKey Capabilities Demonstrated:")
    print("- Goal decomposition into executable steps")
    print("- Constraint-aware planning")
    print("- Adaptive plan creation based on context")
    print("- Error recovery and alternative strategies")
    print("- Complex multi-step workflow automation")
    print("="*70)

if __name__ == "__main__":
    main()