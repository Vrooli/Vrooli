"""AI handler service for Agent S2"""

import os
import logging
import json
import subprocess
from typing import Dict, Any, Optional, List
from datetime import datetime

try:
    import requests
except ImportError:
    requests = None

from ...config import Config
from .capture import ScreenshotService
from .automation import AutomationService

logger = logging.getLogger(__name__)


class AIHandler:
    """Handler for AI-driven operations"""
    
    def __init__(self):
        """Initialize AI handler"""
        self.initialized = False
        self.enabled = Config.AI_ENABLED
        self.provider = "ollama"
        self.model = Config.AI_MODEL
        self.api_url = Config.AI_API_URL
        # Use configured API URL base or fallback to localhost
        if self.api_url:
            # Extract base URL from full API URL
            if "/api/" in self.api_url:
                self.ollama_base_url = self.api_url.split("/api/")[0]
            else:
                self.ollama_base_url = self.api_url
        else:
            self.ollama_base_url = "http://localhost:11434"
        
        # Core services for AI to use
        self.screenshot_service = ScreenshotService()
        self.automation_service = AutomationService()
        
    async def initialize(self):
        """Initialize AI agent with error handling"""
        if not self.enabled:
            logger.info("AI disabled in configuration")
            return
            
        try:
            # Check if requests library is available
            if requests is None:
                logger.error("requests library not available. Install with: pip install requests")
                return
                
            # Check Ollama health using direct HTTP call
            try:
                health_response = requests.get(f"{self.ollama_base_url}/api/tags", timeout=5)
                if health_response.status_code != 200:
                    logger.error(f"Ollama health check failed: {health_response.status_code}")
                    return
            except requests.RequestException as e:
                logger.error(f"Ollama is not reachable: {e}")
                return
                
            # Test API connectivity
            try:
                response = requests.get(f"{self.ollama_base_url}/api/tags", timeout=5)
                if response.status_code != 200:
                    logger.error(f"Ollama API not responding: {response.status_code}")
                    return
                    
                # Check if our model is available
                models = response.json().get("models", [])
                available_models = [model["name"] for model in models]
                
                if self.model not in available_models:
                    logger.warning(f"Configured model '{self.model}' not found. Available models: {available_models}")
                    # Use the first available model as fallback
                    if available_models:
                        self.model = available_models[0]
                        logger.info(f"Using fallback model: {self.model}")
                    else:
                        logger.error("No models available in Ollama")
                        return
                        
            except requests.RequestException as e:
                logger.error(f"Failed to connect to Ollama API: {e}")
                return
                
            self.initialized = True
            logger.info(f"AI handler initialized with Ollama (model: {self.model})")
            
        except Exception as e:
            logger.error(f"Failed to initialize AI handler: {e}")
            self.initialized = False
            
    async def shutdown(self):
        """Shutdown AI handler"""
        self.initialized = False
        
    def _call_ollama(self, prompt: str, model: Optional[str] = None, system: Optional[str] = None) -> Dict[str, Any]:
        """Make a call to Ollama API
        
        Args:
            prompt: The prompt to send
            model: Model to use (defaults to self.model)
            system: Optional system prompt
            
        Returns:
            Response from Ollama
        """
        if not self.initialized:
            raise RuntimeError("AI handler not initialized")
            
        model = model or self.model
        
        payload = {
            "model": model,
            "prompt": prompt,
            "stream": False
        }
        
        if system:
            payload["system"] = system
            
        try:
            response = requests.post(
                f"{self.ollama_base_url}/api/generate",
                json=payload,
                timeout=Config.AI_TIMEOUT
            )
            response.raise_for_status()
            return response.json()
            
        except requests.RequestException as e:
            logger.error(f"Ollama API call failed: {e}")
            raise
        
    def get_capabilities(self) -> List[str]:
        """Get AI capabilities"""
        if not self.initialized:
            return []
            
        return [
            "natural_language_commands",
            "screen_understanding",
            "task_planning",
            "multi_step_automation",
            "visual_element_detection",
            "context_aware_actions"
        ]
        
    async def execute_action(self,
                           task: str,
                           screenshot: Optional[str] = None,
                           context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Execute an AI-driven action
        
        Args:
            task: Task description
            screenshot: Optional screenshot data
            context: Optional context
            
        Returns:
            Execution result
        """
        if not self.initialized:
            raise RuntimeError("AI handler not initialized")
            
        try:
            # Take screenshot if not provided
            if not screenshot:
                screenshot_data = self.screenshot_service.capture()
                screenshot = screenshot_data["data"]
                
            # Build context for AI
            context_str = ""
            if context:
                context_str = f"\nContext: {json.dumps(context, indent=2)}"
                
            # Create AI prompt for task execution
            system_prompt = """You are an AI assistant that helps with computer automation tasks. 
You can analyze screen content and provide step-by-step instructions for automating tasks.
Always provide specific, actionable steps with clear parameters for mouse clicks, keyboard input, etc.
Be precise about coordinates and text to type."""

            user_prompt = f"""Task: {task}
            
Current screen context available (screenshot was taken).{context_str}

Please provide a step-by-step plan to accomplish this task. Format your response as a JSON object with:
- "plan": Array of steps, each with "action", "parameters", and "description"
- "reasoning": Brief explanation of the approach
- "estimated_duration": Time estimate

Example action types: "click", "type", "key_press", "scroll", "wait"
Example parameters: {{"x": 100, "y": 200}}, {{"text": "hello"}}, {{"key": "Enter"}}"""

            # Call Ollama for task planning
            ai_response = self._call_ollama(user_prompt, system=system_prompt)
            ai_text = ai_response.get("response", "")
            
            # Try to parse AI response as JSON, fallback to text analysis
            try:
                # Look for JSON in the response
                import re
                json_match = re.search(r'\{.*\}', ai_text, re.DOTALL)
                if json_match:
                    ai_plan = json.loads(json_match.group())
                else:
                    # Fallback: create plan from text response
                    ai_plan = {
                        "plan": [{"action": "manual_review", "parameters": {}, "description": ai_text[:200]}],
                        "reasoning": "AI provided text response instead of structured plan",
                        "estimated_duration": "Manual review needed"
                    }
            except (json.JSONDecodeError, AttributeError):
                # Fallback for unparseable responses
                ai_plan = {
                    "plan": [{"action": "manual_review", "parameters": {}, "description": "AI response parsing failed"}],
                    "reasoning": f"Raw AI response: {ai_text[:100]}...",
                    "estimated_duration": "Manual review needed"
                }
                
            return {
                "success": True,
                "task": task,
                "summary": f"AI analyzed task: {task}",
                "plan": ai_plan.get("plan", []),
                "reasoning": ai_plan.get("reasoning", "AI provided task analysis"),
                "estimated_duration": ai_plan.get("estimated_duration", "Unknown"),
                "ai_model": self.model,
                "raw_ai_response": ai_text
            }
            
        except Exception as e:
            logger.error(f"AI action execution failed: {e}")
            return {
                "success": False,
                "task": task,
                "summary": "Execution failed",
                "error": str(e)
            }
            
    async def execute_command(self, command: str, context: Optional[str] = None) -> Dict[str, Any]:
        """Execute a natural language command
        
        Args:
            command: Natural language command
            context: Optional context
            
        Returns:
            Command execution result
        """
        if not self.initialized:
            raise RuntimeError("AI handler not initialized")
            
        logger.info(f"Executing AI command: {command}")
        
        actions_taken = []
        
        try:
            # Take initial screenshot for AI analysis
            screenshot = self.screenshot_service.capture()
            actions_taken.append({
                "action": "screenshot",
                "result": "Captured current screen state"
            })
            
            # Create AI prompt for command execution
            system_prompt = """You are an automation assistant. Given a natural language command, analyze it and provide specific automation actions.

Respond with JSON format:
{
  "actions": [
    {"type": "click", "x": 100, "y": 200, "description": "Click on button"},
    {"type": "type", "text": "Hello", "description": "Type text"},
    {"type": "key", "key": "Enter", "description": "Press Enter"},
    {"type": "wait", "duration": 1.0, "description": "Wait 1 second"}
  ],
  "reasoning": "Explanation of the approach"
}

Available action types: click, type, key, scroll, wait, drag"""

            context_str = f" Context: {context}" if context else ""
            user_prompt = f"""Command: {command}{context_str}

Current screen state captured. Provide specific automation actions to execute this command."""

            # Get AI analysis
            ai_response = self._call_ollama(user_prompt, system=system_prompt)
            ai_text = ai_response.get("response", "")
            
            # Parse AI response for actions
            try:
                import re
                json_match = re.search(r'\{.*\}', ai_text, re.DOTALL)
                if json_match:
                    ai_result = json.loads(json_match.group())
                    ai_actions = ai_result.get("actions", [])
                    reasoning = ai_result.get("reasoning", "AI provided action plan")
                else:
                    # Fallback: basic command parsing
                    ai_actions, reasoning = self._parse_command_fallback(command)
            except (json.JSONDecodeError, AttributeError):
                # Fallback: basic command parsing
                ai_actions, reasoning = self._parse_command_fallback(command)
            
            # Execute the actions
            executed_actions = []
            for action in ai_actions:
                try:
                    if action.get("type") == "click":
                        x, y = action.get("x", 500), action.get("y", 300)
                        self.automation_service.click(x, y)
                        executed_actions.append({
                            "action": "click",
                            "parameters": {"x": x, "y": y},
                            "result": action.get("description", "Clicked at position"),
                            "status": "success"
                        })
                        
                    elif action.get("type") == "type":
                        text = action.get("text", "")
                        if text:
                            self.automation_service.type_text(text)
                            executed_actions.append({
                                "action": "type",
                                "parameters": {"text": text},
                                "result": action.get("description", "Typed text"),
                                "status": "success"
                            })
                            
                    elif action.get("type") == "key":
                        key = action.get("key", "")
                        if key:
                            self.automation_service.press_key([key])
                            executed_actions.append({
                                "action": "key",
                                "parameters": {"key": key},
                                "result": action.get("description", f"Pressed {key}"),
                                "status": "success"
                            })
                            
                    elif action.get("type") == "wait":
                        duration = action.get("duration", 1.0)
                        import asyncio
                        await asyncio.sleep(duration)
                        executed_actions.append({
                            "action": "wait",
                            "parameters": {"duration": duration},
                            "result": action.get("description", f"Waited {duration}s"),
                            "status": "success"
                        })
                        
                except Exception as action_error:
                    executed_actions.append({
                        "action": action.get("type", "unknown"),
                        "parameters": action,
                        "result": f"Failed: {action_error}",
                        "status": "failed"
                    })
            
            return {
                "command": command,
                "context": context,
                "executed": True,
                "status": "completed",
                "actions_taken": actions_taken + executed_actions,
                "ai_reasoning": reasoning,
                "ai_model": self.model,
                "message": f"Command executed using AI: {len(executed_actions)} actions performed"
            }
            
        except Exception as e:
            logger.error(f"Command execution failed: {e}")
            return {
                "command": command,
                "context": context,
                "executed": False,
                "status": "failed",
                "actions_taken": actions_taken,
                "error": str(e),
                "message": "Command execution failed"
            }
            
    def _parse_command_fallback(self, command: str) -> tuple:
        """Fallback command parsing when AI fails"""
        command_lower = command.lower()
        actions = []
        
        if "click" in command_lower:
            actions.append({"type": "click", "x": 500, "y": 300, "description": "Click at center"})
        if "type" in command_lower or "write" in command_lower:
            # Try to extract text after "type" or "write"
            import re
            text_match = re.search(r'(?:type|write)\s+["\']([^"\']*)["\']', command, re.IGNORECASE)
            if text_match:
                text = text_match.group(1)
            else:
                text = "Sample text"
            actions.append({"type": "type", "text": text, "description": f"Type: {text}"})
        if "enter" in command_lower:
            actions.append({"type": "key", "key": "Return", "description": "Press Enter"})
            
        reasoning = "Used fallback command parsing (AI parsing failed)"
        return actions, reasoning
            
    async def execute_command_async(self,
                                  task_id: str,
                                  command: str,
                                  context: Optional[str],
                                  task_record: Dict[str, Any]):
        """Execute command asynchronously"""
        try:
            result = await self.execute_command(command, context)
            task_record["status"] = "completed"
            task_record["result"] = result
            task_record["completed_at"] = datetime.utcnow().isoformat()
        except Exception as e:
            task_record["status"] = "failed"
            task_record["error"] = str(e)
            task_record["completed_at"] = datetime.utcnow().isoformat()
            
    async def generate_plan(self, goal: str, constraints: Optional[List[str]] = None) -> Dict[str, Any]:
        """Generate a plan for achieving a goal
        
        Args:
            goal: Goal to achieve
            constraints: Optional constraints
            
        Returns:
            Generated plan
        """
        if not self.initialized:
            raise RuntimeError("AI handler not initialized")
            
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
        if not self.initialized:
            raise RuntimeError("AI handler not initialized")
            
        # Take screenshot if not provided
        if not screenshot:
            screenshot_data = self.screenshot_service.capture()
            screen_size = screenshot_data["size"]
        else:
            screen_size = {"width": 1920, "height": 1080}  # Default
            
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
            ai_response = self._call_ollama(user_prompt, system=system_prompt)
            ai_text = ai_response.get("response", "")
            
            # Parse AI response
            try:
                import re
                json_match = re.search(r'\{.*\}', ai_text, re.DOTALL)
                if json_match:
                    ai_analysis = json.loads(json_match.group())
                else:
                    # Fallback structure
                    ai_analysis = {
                        "analysis": ai_text[:200] if ai_text else "AI analysis not available",
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
                    "analysis": f"Raw AI response: {ai_text[:200]}...",
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
                "ai_model": self.model,
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