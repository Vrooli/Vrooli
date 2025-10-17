"""AI Planner for Agent S2

Handles task planning, screen analysis, and command parsing.
"""

import logging
import json
import re
from typing import Dict, Any, Optional, List, Tuple
from datetime import datetime

from .ai_service_manager import AIServiceManager
from .search_engine_service import search_service
from .task_classifier import task_classifier
from .semantic_validator import semantic_validator

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
    
    def _parse_markdown_response(self, ai_text: str) -> Dict[str, Any]:
        """Parse markdown-formatted AI response and extract action plan
        
        Args:
            ai_text: Raw AI response text in markdown format
            
        Returns:
            Parsed plan dictionary with actions
        """
        plan_actions = []
        reasoning = ""
        estimated_duration = "Unknown"
        
        try:
            # Extract reasoning/explanation from the response
            reasoning_match = re.search(r'\*\*(?:Task|Plan|Approach):\s*([^*]+)\*\*', ai_text, re.IGNORECASE)
            if reasoning_match:
                reasoning = reasoning_match.group(1).strip()
            else:
                reasoning = ai_text[:200] + "..." if len(ai_text) > 200 else ai_text
            
            # Look for numbered action items in various markdown formats
            action_patterns = [
                r'(\d+)\.\s*\*\*([^*]+)\*\*[:\s]*\n?\s*-\s*(?:Action|Type):\s*([^\n]+)\n?\s*-\s*(?:Parameters?|Args?):\s*([^\n]+)',
                r'(\d+)\.\s*\*\*([^*]+)\*\*[:\s]*\n?\s*-\s*Action:\s*([^\n]+)',
                r'(\d+)\.\s*([^:\n]+):\s*([^\n]+)',
                r'\*\*(\d+)\.\s*([^*]+)\*\*[:\s]*([^\n]+)',
                r'(\d+)\.\s*([^\n]+)'
            ]
            
            for pattern in action_patterns:
                matches = re.findall(pattern, ai_text, re.IGNORECASE | re.MULTILINE)
                for match in matches:
                    try:
                        if len(match) >= 4:  # Full format with parameters
                            step_num, description, action_type, parameters = match
                            action_dict = self._parse_action_parameters(action_type, parameters)
                        elif len(match) >= 3:  # Action specified
                            step_num, description, action_or_details = match
                            if any(keyword in action_or_details.lower() for keyword in ['launch_app', 'click', 'type', 'key']):
                                action_dict = self._parse_action_string(action_or_details)
                            else:
                                action_dict = self._infer_action_from_description(description + " " + action_or_details)
                        else:  # Just step description
                            step_num, description = match
                            action_dict = self._infer_action_from_description(description)
                        
                        action_dict["description"] = description.strip()
                        plan_actions.append(action_dict)
                        
                    except Exception as e:
                        logger.warning(f"Failed to parse action from match {match}: {e}")
                        continue
                
                if plan_actions:  # Found actions with this pattern, stop trying others
                    break
            
            # If no actions found, try to extract key action verbs
            if not plan_actions:
                action_verbs = ['launch', 'open', 'click', 'type', 'press', 'scroll', 'navigate', 'go to']
                for verb in action_verbs:
                    verb_pattern = rf'{verb}\s+([^\n.]+)'
                    matches = re.findall(verb_pattern, ai_text, re.IGNORECASE)
                    for match in matches:
                        action_dict = self._infer_action_from_description(f"{verb} {match}")
                        plan_actions.append(action_dict)
            
            # Extract estimated duration if mentioned
            duration_match = re.search(r'(?:duration|time|takes?)[:\s]*([^\n]+)', ai_text, re.IGNORECASE)
            if duration_match:
                estimated_duration = duration_match.group(1).strip()
            
        except Exception as e:
            logger.error(f"Error parsing markdown response: {e}")
            reasoning = f"Markdown parsing failed: {e}"
        
        # Ensure we have at least one action
        if not plan_actions:
            plan_actions = [{
                "action": "manual_review",
                "parameters": {},
                "description": "Could not parse AI response - manual review needed"
            }]
        
        return {
            "plan": plan_actions,
            "reasoning": reasoning,
            "estimated_duration": estimated_duration
        }
    
    def _parse_action_parameters(self, action_type: str, parameters_str: str) -> Dict[str, Any]:
        """Parse action parameters from string"""
        action_type = action_type.strip().lower()
        
        # Try to parse JSON-like parameters
        try:
            # Clean up the parameters string
            params_cleaned = parameters_str.strip()
            if params_cleaned.startswith('{') and params_cleaned.endswith('}'):
                params = json.loads(params_cleaned)
            else:
                # Parse key-value pairs
                params = {}
                for item in params_cleaned.split(','):
                    if ':' in item:
                        key, value = item.split(':', 1)
                        params[key.strip().strip('"')] = value.strip().strip('"')
        except:
            params = {}
        
        return {"action": action_type, "parameters": params}
    
    def _parse_action_string(self, action_str: str) -> Dict[str, Any]:
        """Parse action from a descriptive string"""
        action_str = action_str.lower().strip()
        
        if 'launch_app' in action_str or 'launch' in action_str:
            app_match = re.search(r'(?:launch|open)\s+([^\s,]+)', action_str)
            app_name = app_match.group(1) if app_match else "Firefox ESR"
            return {"action": "launch_app", "parameters": {"app_name": app_name}}
            
        elif 'click' in action_str:
            # Try to extract coordinates
            coord_match = re.search(r'(\d+)[,\s]+(\d+)', action_str)
            if coord_match:
                x, y = int(coord_match.group(1)), int(coord_match.group(2))
            else:
                x, y = 500, 300  # Default center
            return {"action": "click", "parameters": {"x": x, "y": y}}
            
        elif 'type' in action_str:
            text_match = re.search(r'["\']([^"\']+)["\']', action_str)
            text = text_match.group(1) if text_match else action_str.replace('type', '').strip()
            return {"action": "type", "parameters": {"text": text}}
            
        elif 'key' in action_str or 'press' in action_str:
            key_match = re.search(r'(?:key|press)\s+([^\s,]+)', action_str)
            key = key_match.group(1) if key_match else "Enter"
            return {"action": "key", "parameters": {"key": key}}
        
        return {"action": "manual_review", "parameters": {}}
    
    def _infer_action_from_description(self, description: str) -> Dict[str, Any]:
        """Infer action type from natural language description using intelligent classification"""
        desc_lower = description.lower()
        
        # Use task classifier to understand intent
        classification = task_classifier.classify_task(description)
        
        if any(word in desc_lower for word in ['launch', 'open', 'start']):
            if 'firefox' in desc_lower or 'browser' in desc_lower:
                return {"action": "launch_app", "parameters": {"app_name": "Firefox ESR"}}
            else:
                # Default to Firefox for web-related tasks
                return {"action": "launch_app", "parameters": {"app_name": "Firefox ESR"}}
                
        elif any(word in desc_lower for word in ['click', 'press', 'tap']):
            if 'address' in desc_lower or 'url' in desc_lower:
                return {"action": "click", "parameters": {"x": 960, "y": 100}}  # Address bar area
            else:
                return {"action": "click", "parameters": {"x": 500, "y": 300}}
                
        elif any(word in desc_lower for word in ['type', 'enter', 'input']):
            # Use intelligent search/navigation logic instead of hardcoded URLs
            if classification["task_type"].value in ["search", "image_search"]:
                # Generate appropriate search URL
                search_url, _ = search_service.get_appropriate_search_url(description)
                return {"action": "type", "parameters": {"text": search_url}}
            elif classification["task_type"].value == "navigation":
                # Extract target and validate
                target = classification["metadata"].get("target")
                if target and semantic_validator.validate_url_for_task(f"https://{target}", description)["is_valid"]:
                    return {"action": "type", "parameters": {"text": f"https://{target}"}}
                else:
                    # Fallback to search if navigation target is unclear/unsafe
                    search_url, _ = search_service.get_appropriate_search_url(description)
                    return {"action": "type", "parameters": {"text": search_url}}
            elif 'url' in desc_lower or 'address' in desc_lower:
                # Generate appropriate search URL as safe default
                search_url, _ = search_service.get_appropriate_search_url(description)
                return {"action": "type", "parameters": {"text": search_url}}
            else:
                return {"action": "type", "parameters": {"text": "text to type"}}
                
        elif any(word in desc_lower for word in ['wait', 'pause']):
            return {"action": "wait", "parameters": {"seconds": 2}}
            
        elif any(word in desc_lower for word in ['navigate', 'go to', 'visit']):
            # Use intelligent navigation with validation
            if classification["task_type"].value == "navigation":
                target = classification["metadata"].get("target")
                if target and semantic_validator.validate_url_for_task(f"https://{target}", description)["is_valid"]:
                    return {"action": "type", "parameters": {"text": f"https://{target}"}}
            
            # Fallback to search for ambiguous navigation requests
            search_url, _ = search_service.get_appropriate_search_url(description)
            return {"action": "type", "parameters": {"text": search_url}}
        
        return {"action": "manual_review", "parameters": {}}
    
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
    
    def _validate_plan_completeness(self, task: str, plan: List[Dict[str, Any]]) -> bool:
        """Validate if a plan is complete for the given task
        
        Args:
            task: The original task description
            plan: The generated plan
            
        Returns:
            True if plan appears complete, False otherwise
        """
        task_lower = task.lower()
        
        # Check for navigation tasks
        if any(phrase in task_lower for phrase in ['go to', 'navigate to', 'visit', 'open website']):
            # For navigation tasks, we need more than just launching a browser
            has_launch = any(step.get('action') == 'launch_app' for step in plan)
            has_url_entry = any(step.get('action') == 'type' and 
                              ('http' in str(step.get('parameters', {}).get('text', '')) or
                               '.com' in str(step.get('parameters', {}).get('text', '')))
                              for step in plan)
            has_navigation = any(step.get('action') == 'key' and 
                               step.get('parameters', {}).get('key') == 'Enter' 
                               for step in plan)
            
            if has_launch and not (has_url_entry or has_navigation):
                logger.warning(f"Navigation task '{task}' has incomplete plan - missing URL entry/navigation steps")
                return False
                
        # Check for search tasks
        elif any(phrase in task_lower for phrase in ['search for', 'look up', 'find']):
            has_search_entry = any(step.get('action') == 'type' for step in plan)
            has_submit = any(step.get('action') == 'key' and 
                           step.get('parameters', {}).get('key') == 'Enter' 
                           for step in plan)
            
            if not (has_search_entry and has_submit):
                logger.warning(f"Search task '{task}' has incomplete plan - missing search entry/submit steps")
                return False
        
        # Check minimum steps - most real tasks need at least 2-3 steps
        if len(plan) < 2 and not any(word in task_lower for word in ['screenshot', 'capture', 'analyze']):
            logger.warning(f"Task '{task}' has suspiciously few steps ({len(plan)})")
            return False
            
        return True
    
    def _validate_and_fix_plan_urls(self, task: str, plan: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Validate and fix URLs in the plan using semantic validation
        
        Args:
            task: Original task description
            plan: List of plan steps
            
        Returns:
            Updated plan with validated/fixed URLs
        """
        fixed_plan = []
        url_fixes_applied = 0
        
        for step in plan:
            # Create a copy of the step
            fixed_step = step.copy()
            
            # Check if this step contains a URL to validate
            if step.get("action") == "type":
                text_to_type = step.get("parameters", {}).get("text", "")
                
                # Check if the text looks like a URL
                if (text_to_type.startswith("http") or 
                    ("." in text_to_type and 
                     any(tld in text_to_type for tld in [".com", ".org", ".net", ".edu", ".gov", ".io"]))):
                    
                    # Validate the URL semantically
                    validation = semantic_validator.validate_url_for_task(text_to_type, task)
                    
                    if not validation["is_valid"] or validation["confidence"] < 0.7:
                        logger.warning(f"Invalid/inappropriate URL detected: {text_to_type} for task '{task}'")
                        logger.warning(f"Issues: {', '.join(validation['issues'])}")
                        
                        # Use the suggested alternative if available
                        if validation["alternative_url"]:
                            logger.info(f"Replacing with safer alternative: {validation['alternative_url']}")
                            fixed_step["parameters"]["text"] = validation["alternative_url"]
                            fixed_step["description"] = f"Type appropriate URL (replaced suspicious/inappropriate URL)"
                            url_fixes_applied += 1
                        else:
                            # Generate a search URL as fallback
                            search_url, _ = search_service.get_appropriate_search_url(task)
                            logger.info(f"Generating search fallback: {search_url}")
                            fixed_step["parameters"]["text"] = search_url
                            fixed_step["description"] = f"Type search URL (replaced inappropriate URL)"
                            url_fixes_applied += 1
                    else:
                        logger.info(f"URL validation passed: {text_to_type} (confidence: {validation['confidence']:.2f})")
            
            fixed_plan.append(fixed_step)
        
        if url_fixes_applied > 0:
            logger.info(f"Applied {url_fixes_applied} URL fixes to plan for task '{task}'")
        
        return fixed_plan
    
    async def plan_task_execution(self, task: str, context: Optional[Dict[str, Any]] = None, 
                                 screenshot: Optional[str] = None, window_context: Optional[str] = None) -> Dict[str, Any]:
        """Plan the execution of a task
        
        Args:
            task: Task description
            context: Optional context information
            screenshot: Optional screenshot data (base64 encoded)
            window_context: Optional window context information
            
        Returns:
            Task execution plan with debug information
        """
        if not self.service_manager.is_ready():
            raise RuntimeError("AI service manager not initialized")
        
        # Build context string
        context_str = ""
        if context:
            context_str = f"\nContext: {json.dumps(context, indent=2)}"
            
        # Add window context if available
        if window_context:
            context_str += f"\n\nWindow Information:\n{window_context}"
        
        # Use intelligent task classification before creating prompt
        task_classification = task_classifier.classify_task(task)
        search_url = None
        
        # Pre-generate appropriate URL if this is a search/navigation task
        if task_classification["task_type"].value in ["search", "image_search"]:
            search_url, search_intent = search_service.get_appropriate_search_url(task)
            logger.info(f"Pre-generated search URL for '{task}': {search_url}")
        
        # Create intelligent, bias-free AI prompt for task planning
        system_prompt = f"""You are an AI assistant that helps with computer automation tasks. 
You can analyze screen content and provide step-by-step instructions for automating tasks.
Always provide specific, actionable steps with clear parameters for mouse clicks, keyboard input, etc.
Be precise about coordinates and text to type.

CRITICAL: YOUR PLAN MUST BE COMPLETE AND ACHIEVE THE FULL TASK OBJECTIVE!
- DO NOT stop after just launching an application
- Include ALL steps needed to complete the task from start to finish
- For search tasks, use appropriate search engines or websites
- For navigation tasks, validate that URLs make sense for the task

APPLICATION LAUNCHING:
- To open/launch applications, ALWAYS use the "launch_app" action type
- DO NOT type launcher commands like "Alt+F1 → firefox" - use launch_app instead
- Example: To open Firefox, use action "launch_app" with parameters {{"app_name": "Firefox ESR"}}

INTELLIGENT URL SELECTION:
- For SEARCH queries (like "show me puppies", "find tutorials"), use search engines
- For IMAGE searches, prefer image search engines or photo sites
- For specific WEBSITES, use the actual domain (e.g., "go to YouTube" → "https://youtube.com")
- NEVER generate suspicious or nonsensical domain names
- When unsure, default to search engines for safety

SECURITY GUIDELINES FOR WEB NAVIGATION:
- ALWAYS use full URLs with https:// protocol when navigating to websites
- Only navigate to legitimate, well-known domains
- For ambiguous requests, use search engines instead of guessing domains
- Verify the URL makes semantic sense for the task
- NEVER type a domain name if it's already in the address bar
- When navigating to a new site, first click on the address bar, then clear it (Ctrl+A), then type the full URL

IMPORTANT WINDOW FOCUS RULES:
1. The ACTIVE WINDOW receives keyboard input
2. To interact with a different window, you MUST click on its Focus Point coordinates (shown above)
3. NEVER guess window positions - always use the exact Focus Point coordinates provided
4. Each window shows "Focus Point: (x, y)" - these are the exact coordinates to click for focusing that window

EXAMPLE COMPLETE PLANS:
For "go to YouTube":
1. Launch Firefox ESR (if not already open)
2. Click on the address bar (specific coordinates)
3. Clear address bar (Ctrl+A)
4. Type "https://youtube.com"
5. Press Enter
6. Wait for page to load

For "show me puppies" (SEARCH task):
1. Launch Firefox ESR (if not already open)
2. Click on the address bar
3. Clear address bar (Ctrl+A)
4. Type appropriate search URL (e.g., DuckDuckGo or image search)
5. Press Enter
6. Wait for results

For "search for tutorials":
1. Launch Firefox ESR (if not already open)
2. Click on the address bar
3. Clear address bar (Ctrl+A)
4. Type search engine URL with query
5. Press Enter
6. Review results

REMEMBER: A plan that only launches an application is INCOMPLETE for navigation/search tasks!
TASK CLASSIFICATION: {task_classification["task_type"].value} - {task_classification["reasoning"]}"""

        # Add intelligent URL suggestion to user prompt if available
        url_guidance = ""
        if search_url:
            url_guidance = f"\nSUGGESTED URL FOR THIS TASK: {search_url}\n- This URL was intelligently selected based on task classification\n- Use this URL instead of guessing domain names\n- This ensures safe and appropriate navigation\n"

        user_prompt = f"""Task: {task}
        
Current screen context available (screenshot was taken).{context_str}
{url_guidance}
CRITICAL REQUIREMENTS:
1. You MUST respond with ONLY valid JSON. No markdown, no explanations outside the JSON, no code blocks. Start your response with {{ and end with }}.
2. Your plan MUST be COMPLETE and achieve the FULL task objective
3. For search tasks, use the suggested URL above if provided
4. For navigation tasks, only use legitimate, well-known domains
5. NEVER generate suspicious or nonsensical domain names

Required JSON format:
{{
  "plan": [
    {{
      "action": "action_type",
      "parameters": {{}},
      "description": "What this step does"
    }}
  ],
  "reasoning": "Brief explanation of the approach",
  "estimated_duration": "Time estimate like '30 seconds'"
}}

Valid action types: "click", "type", "key", "scroll", "wait", "launch_app"
Example parameters: 
- click: {{"x": 100, "y": 200}}
- type: {{"text": "hello"}}
- key: {{"key": "Enter"}}
- launch_app: {{"app_name": "Firefox ESR"}}
- wait: {{"seconds": 2}}

EXAMPLE COMPLETE PLAN for "go to YouTube":
{{
  "plan": [
    {{"action": "launch_app", "parameters": {{"app_name": "Firefox ESR"}}, "description": "Launch Firefox browser"}},
    {{"action": "wait", "parameters": {{"seconds": 2}}, "description": "Wait for Firefox to fully load"}},
    {{"action": "click", "parameters": {{"x": 700, "y": 100}}, "description": "Click on address bar"}},
    {{"action": "key", "parameters": {{"key": "Ctrl+A"}}, "description": "Select all text in address bar"}},
    {{"action": "type", "parameters": {{"text": "https://youtube.com"}}, "description": "Type YouTube URL"}},
    {{"action": "key", "parameters": {{"key": "Enter"}}, "description": "Press Enter to navigate"}},
    {{"action": "wait", "parameters": {{"seconds": 3}}, "description": "Wait for page to load"}}
  ],
  "reasoning": "Navigate to YouTube by launching Firefox, clearing the address bar, typing the URL, and pressing Enter",
  "estimated_duration": "10 seconds"
}}

Respond with the JSON object only - no other text before or after."""

        try:
            # Prepare images list for AI call if screenshot provided
            images = []
            if screenshot and screenshot.startswith('data:image'):
                # Extract base64 part from data URL
                base64_data = screenshot.split(',')[1]
                images = [base64_data]
            
            # Call AI for task planning with screenshot if available
            ai_response = self.service_manager.call_ollama(
                user_prompt, 
                system=system_prompt,
                images=images if images else None
            )
            ai_text = ai_response.get("response", "")
            
            # Try to parse AI response as JSON, fallback to markdown parsing
            try:
                # Look for JSON in the response
                json_match = re.search(r'\{.*\}', ai_text, re.DOTALL)
                if json_match:
                    ai_plan = json.loads(json_match.group())
                else:
                    # Enhanced fallback: parse markdown-formatted response
                    logger.info("JSON not found, attempting markdown parsing")
                    ai_plan = self._parse_markdown_response(ai_text)
            except (json.JSONDecodeError, AttributeError) as e:
                # Enhanced fallback for unparseable JSON - try markdown parsing
                logger.warning(f"JSON parsing failed: {e}, attempting markdown parsing")
                ai_plan = self._parse_markdown_response(ai_text)
            
            # Validate plan completeness and URLs
            plan = ai_plan.get("plan", [])
            
            # First, validate URLs in the plan for semantic appropriateness
            plan = self._validate_and_fix_plan_urls(task, plan)
            
            if not self._validate_plan_completeness(task, plan):
                logger.warning(f"Plan appears incomplete for task '{task}', attempting to enhance...")
                
                # If the plan is incomplete, add common missing steps with intelligent URL selection
                if any(phrase in task.lower() for phrase in ['go to', 'navigate to', 'visit']):
                    # Add missing navigation steps with intelligent URL
                    enhanced_plan = []
                    for step in plan:
                        enhanced_plan.append(step)
                        # After launching browser, add navigation steps if missing
                        if step.get('action') == 'launch_app' and len(plan) <= 2:
                            # Use intelligent URL generation instead of hardcoded fallback
                            if search_url:
                                intelligent_url = search_url
                            else:
                                intelligent_url, _ = search_service.get_appropriate_search_url(task)
                            
                            enhanced_plan.extend([
                                {"action": "wait", "parameters": {"seconds": 2}, "description": "Wait for browser to load"},
                                {"action": "click", "parameters": {"x": 700, "y": 100}, "description": "Click on address bar"},
                                {"action": "key", "parameters": {"key": "Ctrl+A"}, "description": "Select all text in address bar"},
                                {"action": "type", "parameters": {"text": intelligent_url}, "description": "Type appropriate URL"},
                                {"action": "key", "parameters": {"key": "Enter"}, "description": "Press Enter to navigate"},
                                {"action": "wait", "parameters": {"seconds": 3}, "description": "Wait for page to load"}
                            ])
                    plan = enhanced_plan
                    ai_plan["plan"] = plan
                    logger.info(f"Enhanced plan from {len(ai_plan.get('plan', []))} to {len(plan)} steps")
            
            return {
                "task": task,
                "plan": plan,
                "reasoning": ai_plan.get("reasoning", "AI provided task analysis"),
                "estimated_duration": ai_plan.get("estimated_duration", "Unknown"),
                "ai_model": self.service_manager.model,
                "raw_ai_response": ai_text,
                "debug_info": {
                    "system_prompt": system_prompt,
                    "user_prompt": user_prompt,
                    "screenshot_included": bool(images),
                    "context": context_str if context_str else None,
                    "window_context_included": bool(window_context),
                    "plan_validated": self._validate_plan_completeness(task, plan),
                    "plan_enhanced": len(plan) > len(ai_plan.get("plan", []))
                }
            }
            
        except Exception as e:
            logger.error(f"Task planning failed: {e}")
            return {
                "task": task,
                "plan": [{"action": "error", "parameters": {}, "description": f"Planning failed: {e}"}],
                "reasoning": "Task planning encountered an error",
                "estimated_duration": "Unknown",
                "error": str(e),
                "debug_info": {
                    "system_prompt": system_prompt if 'system_prompt' in locals() else None,
                    "user_prompt": user_prompt if 'user_prompt' in locals() else None,
                    "screenshot_included": bool(screenshot),
                    "context": context_str if 'context_str' in locals() else None,
                    "window_context_included": bool(window_context),
                    "error": str(e)
                }
            }