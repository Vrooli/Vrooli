"""AI Planner for Agent S2

Handles task planning, screen analysis, and command parsing.
"""

import logging
import json
import re
from typing import Dict, Any, Optional, List, Tuple
from datetime import datetime

from .ai_service_manager import AIServiceManager

logger = logging.getLogger(__name__)


class AIPlanner:
    """Handles AI-driven planning, analysis, and command parsing"""
    
    def __init__(self, service_manager: AIServiceManager):
        """Initialize AI planner
        
        Args:
            service_manager: AI service manager for API calls
        """
        self.service_manager = service_manager
    
    async def generate_plan(self, goal: str, constraints: Optional[List[str]] = None) -> Dict[str, Any]:
        """Generate a plan for achieving a goal
        
        Args:
            goal: Goal to achieve
            constraints: Optional constraints
            
        Returns:
            Generated plan
        """
        if not self.service_manager.is_ready():
            raise RuntimeError("AI service manager not initialized")
            
        # Build constraints string
        constraints_str = ""
        if constraints:
            constraints_str = f"\nConstraints to consider:\n" + "\n".join(f"- {c}" for c in constraints)
        
        # Create AI prompt for high-level planning
        system_prompt = """You are an expert planning assistant for computer automation tasks.
You help break down goals into logical, sequential steps that can be automated.
Focus on high-level planning rather than specific technical implementation details.
Always consider prerequisite steps and dependencies between actions."""

        user_prompt = f"""Goal: {goal}{constraints_str}

Please create a high-level plan to achieve this goal. Format your response as JSON:
{{
  "steps": [
    {{
      "step": 1,
      "action": "action_name",
      "description": "Clear description of what to do",
      "prerequisites": ["any prerequisites"],
      "expected_outcome": "what should happen"
    }}
  ],
  "estimated_duration": "time estimate",
  "complexity": "low/medium/high",
  "success_criteria": ["how to know the goal is achieved"]
}}

Keep steps at a high level - focus on what needs to be done rather than exact technical implementation.
Example actions: navigate_to_location, gather_information, create_content, configure_settings, verify_results"""

        try:
            # Get AI planning response
            ai_response = self.service_manager.call_ollama(user_prompt, system=system_prompt)
            ai_text = ai_response.get("response", "")
            
            # Parse AI response
            try:
                json_match = re.search(r'\{.*\}', ai_text, re.DOTALL)
                if json_match:
                    ai_plan = json.loads(json_match.group())
                    
                    # Validate and enhance the plan structure
                    steps = ai_plan.get("steps", [])
                    if not steps:
                        raise ValueError("No steps provided in AI response")
                    
                    # Ensure each step has required fields
                    for i, step in enumerate(steps):
                        if "step" not in step:
                            step["step"] = i + 1
                        if "action" not in step:
                            step["action"] = f"action_{i + 1}"
                        if "description" not in step:
                            step["description"] = f"Step {i + 1} for: {goal}"
                        if "prerequisites" not in step:
                            step["prerequisites"] = []
                        if "expected_outcome" not in step:
                            step["expected_outcome"] = "Progress towards goal"
                    
                    return {
                        "goal": goal,
                        "constraints": constraints or [],
                        "steps": steps,
                        "estimated_duration": ai_plan.get("estimated_duration", "Unknown"),
                        "complexity": ai_plan.get("complexity", "medium"),
                        "success_criteria": ai_plan.get("success_criteria", ["Goal achieved"]),
                        "ai_model": self.service_manager.model,
                        "raw_ai_response": ai_text
                    }
                else:
                    raise ValueError("No JSON found in AI response")
                    
            except (json.JSONDecodeError, ValueError) as e:
                logger.warning(f"AI response parsing failed: {e}, using fallback plan")
                # Create a fallback plan based on the goal
                return self._create_fallback_plan(goal, constraints, ai_text)
                
        except Exception as e:
            logger.error(f"AI plan generation failed: {e}")
            return self._create_fallback_plan(goal, constraints, f"Error: {e}")
    
    def _create_fallback_plan(self, goal: str, constraints: Optional[List[str]], ai_response: str) -> Dict[str, Any]:
        """Create a fallback plan when AI planning fails
        
        Args:
            goal: The original goal
            constraints: Original constraints
            ai_response: The AI response (for reference)
            
        Returns:
            Fallback plan structure
        """
        return {
            "goal": goal,
            "constraints": constraints or [],
            "steps": [
                {
                    "step": 1,
                    "action": "analyze_current_state",
                    "description": "Take screenshot and analyze current desktop state",
                    "prerequisites": [],
                    "expected_outcome": "Understanding of current environment"
                },
                {
                    "step": 2,
                    "action": "identify_requirements",
                    "description": f"Determine what applications, websites, or tools are needed for: {goal}",
                    "prerequisites": ["analyze_current_state"],
                    "expected_outcome": "Clear list of required resources"
                },
                {
                    "step": 3,
                    "action": "execute_actions",
                    "description": "Perform the necessary actions to achieve the goal",
                    "prerequisites": ["identify_requirements"],
                    "expected_outcome": "Progress towards goal completion"
                },
                {
                    "step": 4,
                    "action": "verify_results",
                    "description": "Check if the goal has been successfully achieved",
                    "prerequisites": ["execute_actions"],
                    "expected_outcome": "Confirmation of goal completion"
                }
            ],
            "estimated_duration": "5-10 minutes",
            "complexity": "medium",
            "success_criteria": [f"Goal '{goal}' is achieved", "All required actions completed"],
            "ai_model": self.service_manager.model if hasattr(self, 'service_manager') else "fallback",
            "raw_ai_response": ai_response,
            "note": "Fallback plan used due to AI response parsing issues"
        }
        
    async def analyze_screen(self,
                           question: Optional[str] = None,
                           screenshot: Optional[str] = None) -> Dict[str, Any]:
        """Analyze the screen content
        
        Args:
            question: Optional specific question
            screenshot: Optional screenshot to analyze
            
        Returns:
            Analysis results
        """
        if not self.service_manager.is_ready():
            raise RuntimeError("AI service manager not initialized")
            
        # Default screen size if not provided
        screen_size = {"width": 1920, "height": 1080}
            
        try:
            # Create prompt for screen analysis
            system_prompt = """You are an expert at analyzing computer interfaces and suggesting automation actions.
Without seeing the actual image, provide intelligent suggestions based on the question asked.
Focus on common UI patterns and automation possibilities."""

            user_prompt = f"""Question: {question or 'Provide a general screen analysis'}

Please analyze what might be on screen and suggest possible automation actions. Format as JSON:
{{
  "analysis": "Description of likely screen content",
  "elements_detected": [
    {{"type": "element_type", "description": "Element description"}}
  ],
  "suggested_actions": [
    {{"action": "action_type", "target": "target_description", "reasoning": "why this action"}}
  ]
}}

Common UI elements: buttons, text fields, menus, toolbars, windows, dialogs
Common actions: click, type, scroll, drag, key presses"""

            # Get AI analysis
            ai_response = self.service_manager.call_ollama(user_prompt, system=system_prompt)
            ai_text = ai_response.get("response", "")
            
            # Parse AI response
            try:
                json_match = re.search(r'\{.*\}', ai_text, re.DOTALL)
                if json_match:
                    ai_analysis = json.loads(json_match.group())
                else:
                    # Fallback structure
                    ai_analysis = {
                        "analysis": ai_text[0:200] if ai_text and len(ai_text) > 200 else (ai_text or "AI analysis not available"),
                        "elements_detected": [
                            {"type": "unknown", "description": "AI response parsing failed"}
                        ],
                        "suggested_actions": [
                            {"action": "manual_review", "target": "screen", "reasoning": "AI analysis incomplete"}
                        ]
                    }
            except (json.JSONDecodeError, AttributeError):
                # Fallback for unparseable responses
                ai_analysis = {
                    "analysis": f"Raw AI response: {ai_text[0:200] if len(ai_text) > 200 else ai_text}...",
                    "elements_detected": [
                        {"type": "text", "description": "AI provided text analysis"}
                    ],
                    "suggested_actions": [
                        {"action": "manual_review", "target": "screen", "reasoning": "Response parsing failed"}
                    ]
                }
                
            return {
                "screen_size": screen_size,
                "question": question,
                "analysis": ai_analysis.get("analysis", "No analysis available"),
                "elements_detected": ai_analysis.get("elements_detected", []),
                "suggested_actions": ai_analysis.get("suggested_actions", []),
                "ai_model": self.service_manager.model,
                "raw_ai_response": ai_text,
                "note": "Analysis based on AI reasoning without direct image analysis (vision models not configured)"
            }
            
        except Exception as e:
            logger.error(f"Screen analysis failed: {e}")
            return {
                "screen_size": screen_size,
                "question": question,
                "analysis": f"Analysis failed: {e}",
                "elements_detected": [],
                "suggested_actions": [],
                "error": str(e)
            }
    
    def parse_command_fallback(self, command: str, target_app: Optional[str] = None) -> Tuple[List[Dict[str, Any]], str]:
        """Fallback command parsing when AI fails
        
        Args:
            command: Command to parse
            target_app: Optional target application
            
        Returns:
            Tuple of (actions list, reasoning string)
        """
        command_lower = command.lower()
        actions = []
        
        if "click" in command_lower:
            action = {"type": "click", "x": 500, "y": 300, "description": "Click at center"}
            if target_app:
                action["target_app"] = target_app
            actions.append(action)
            
        if "type" in command_lower or "write" in command_lower:
            # Try to extract text after "type" or "write"
            text_match = re.search(r'(?:type|write)\s+["\']([^"\']*)["\']', command, re.IGNORECASE)
            if text_match:
                text = text_match.group(1)
            else:
                text = "Sample text"
            action = {"type": "type", "text": text, "description": f"Type: {text}"}
            if target_app:
                action["target_app"] = target_app
            actions.append(action)
            
        if "enter" in command_lower:
            action = {"type": "key", "key": "Return", "description": "Press Enter"}
            if target_app:
                action["target_app"] = target_app
            actions.append(action)
            
        reasoning = f"Used fallback command parsing (AI parsing failed){f' with target app: {target_app}' if target_app else ''}"
        return actions, reasoning
    
    async def plan_task_execution(self, task: str, context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Plan the execution of a task
        
        Args:
            task: Task description
            context: Optional context information
            
        Returns:
            Task execution plan
        """
        if not self.service_manager.is_ready():
            raise RuntimeError("AI service manager not initialized")
        
        # Build context string
        context_str = ""
        if context:
            context_str = f"\nContext: {json.dumps(context, indent=2)}"
        
        # Create AI prompt for task planning
        system_prompt = f"""You are an AI assistant that helps with computer automation tasks. 
You can analyze screen content and provide step-by-step instructions for automating tasks.
Always provide specific, actionable steps with clear parameters for mouse clicks, keyboard input, etc.
Be precise about coordinates and text to type.

APPLICATION LAUNCHING:
- To open/launch applications, ALWAYS use the "launch_app" action type
- DO NOT type launcher commands like "Alt+F1 → firefox" - use launch_app instead
- Example: To open Firefox, use action "launch_app" with parameters {{"app_name": "Firefox ESR"}}

SECURITY GUIDELINES FOR WEB NAVIGATION:
- ALWAYS use full URLs with https:// protocol when navigating to websites
- For example: use "https://reddit.com" not "reddit.com"
- Double-check typed URLs for typos before pressing Enter
- Verify the URL in address bar matches the intended destination
- NEVER type a domain name if it's already in the address bar (to avoid duplicates like reddit.comreddit.com)
- When navigating to a new site, first click on the address bar, then clear it (Ctrl+A), then type the full URL

IMPORTANT WINDOW FOCUS RULES:
1. The ACTIVE WINDOW receives keyboard input
2. To interact with a different window, you MUST click on its Focus Point coordinates (shown above)
3. NEVER guess window positions - always use the exact Focus Point coordinates provided
4. Each window shows "Focus Point: (x, y)" - these are the exact coordinates to click for focusing that window"""

        user_prompt = f"""Task: {task}
        
Current screen context available (screenshot was taken).{context_str}

Please provide a step-by-step plan to accomplish this task. Format your response as a JSON object with:
- "plan": Array of steps, each with "action", "parameters", and "description"
- "reasoning": Brief explanation of the approach
- "estimated_duration": Time estimate

Example action types: "click", "type", "key", "scroll", "wait", "launch_app"
Example parameters: {{"x": 100, "y": 200}}, {{"text": "hello"}}, {{"key": "Enter"}}, {{"app_name": "Firefox ESR"}}

For launching applications: {{"action": "launch_app", "parameters": {{"app_name": "ApplicationName"}}, "description": "Launch application"}}"""

        try:
            # Call AI for task planning
            ai_response = self.service_manager.call_ollama(user_prompt, system=system_prompt)
            ai_text = ai_response.get("response", "")
            
            # Try to parse AI response as JSON, fallback to text analysis
            try:
                # Look for JSON in the response
                json_match = re.search(r'\{.*\}', ai_text, re.DOTALL)
                if json_match:
                    ai_plan = json.loads(json_match.group())
                else:
                    # Fallback: create plan from text response
                    ai_plan = {
                        "plan": [{"action": "manual_review", "parameters": {}, "description": ai_text[0:200] if len(ai_text) > 200 else ai_text}],
                        "reasoning": "AI provided text response instead of structured plan",
                        "estimated_duration": "Manual review needed"
                    }
            except (json.JSONDecodeError, AttributeError):
                # Fallback for unparseable responses
                ai_plan = {
                    "plan": [{"action": "manual_review", "parameters": {}, "description": "AI response parsing failed"}],
                    "reasoning": f"Raw AI response: {ai_text[0:100] if len(ai_text) > 100 else ai_text}...",
                    "estimated_duration": "Manual review needed"
                }
            
            return {
                "task": task,
                "plan": ai_plan.get("plan", []),
                "reasoning": ai_plan.get("reasoning", "AI provided task analysis"),
                "estimated_duration": ai_plan.get("estimated_duration", "Unknown"),
                "ai_model": self.service_manager.model,
                "raw_ai_response": ai_text
            }
            
        except Exception as e:
            logger.error(f"Task planning failed: {e}")
            return {
                "task": task,
                "plan": [{"action": "error", "parameters": {}, "description": f"Planning failed: {e}"}],
                "reasoning": "Task planning encountered an error",
                "estimated_duration": "Unknown",
                "error": str(e)
            }