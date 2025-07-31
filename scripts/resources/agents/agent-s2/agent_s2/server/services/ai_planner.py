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
            
        # TODO: Implement actual AI planning
        # For now, return a template plan
        return {
            "steps": [
                {
                    "step": 1,
                    "action": "analyze_current_state",
                    "description": "Understand the current screen and context"
                },
                {
                    "step": 2,
                    "action": "identify_requirements",
                    "description": f"Determine what's needed for: {goal}"
                },
                {
                    "step": 3,
                    "action": "execute_actions",
                    "description": "Perform necessary actions to achieve goal"
                }
            ],
            "estimated_duration": "2-5 minutes",
            "complexity": "medium"
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