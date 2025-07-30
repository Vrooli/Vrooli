"""AI handler service for Agent S2"""

import os
import logging
import json
import subprocess
import time
from typing import Dict, Any, Optional, List
from datetime import datetime

try:
    import requests
except ImportError:
    requests = None

from ...config import Config
from .capture import ScreenshotService
from .automation import AutomationService
from ...stealth import StealthManager, StealthConfig

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
        
        # Initialize stealth manager
        stealth_config = StealthConfig(
            enabled=Config.STEALTH_MODE_ENABLED,
            session_storage_path=Config.SESSION_STORAGE_PATH
        )
        self.stealth_manager = StealthManager(stealth_config)
        
    async def _discover_ollama_service(self):
        """Auto-detect available Ollama services
        
        Returns:
            Dict with discovered service info or None if not found
        """
        logger.info("🔍 Auto-detecting Ollama services...")
        
        # Common Ollama locations to check
        ollama_locations = [
            ("http://localhost:11434", "Local Ollama"),
            ("http://ollama:11434", "Docker service 'ollama'"),
            ("http://host.docker.internal:11434", "Host machine (Docker Desktop)"),
            ("http://172.17.0.1:11434", "Docker host gateway")
        ]
        
        # Add current configured URL if different
        if self.ollama_base_url not in [loc[0] for loc in ollama_locations]:
            ollama_locations.insert(0, (self.ollama_base_url, "Configured URL"))
        
        for url, description in ollama_locations:
            try:
                logger.debug(f"Checking {description} at {url}")
                response = requests.get(f"{url}/api/tags", timeout=2)
                
                if response.status_code == 200:
                    models_data = response.json()
                    models = models_data.get("models", [])
                    model_names = [m["name"] for m in models]
                    
                    logger.info(f"✅ Found Ollama at {url} ({description}) with {len(models)} models")
                    
                    # Look for vision models first
                    vision_models = [m for m in model_names if "vision" in m.lower() or "llava" in m.lower()]
                    
                    return {
                        "url": url,
                        "description": description,
                        "models": model_names,
                        "vision_models": vision_models,
                        "model_count": len(models)
                    }
                    
            except requests.RequestException as e:
                logger.debug(f"Failed to connect to {url}: {type(e).__name__}")
                continue
            except Exception as e:
                logger.debug(f"Unexpected error checking {url}: {e}")
                continue
        
        logger.warning("❌ No Ollama service found at any common location")
        return None

    async def initialize(self):
        """Initialize AI agent with error handling and auto-detection"""
        if not self.enabled:
            logger.info("AI disabled in configuration")
            return
            
        logger.info(f"Initializing AI handler with provider: {self.provider}")
        logger.info(f"Configured model: {self.model}")
        logger.info(f"Initial Ollama base URL: {self.ollama_base_url}")
        
        try:
            # Check if requests library is available
            if requests is None:
                logger.error("requests library not available. Install with: pip install requests")
                return
            
            # Try auto-detection first
            discovered = await self._discover_ollama_service()
            
            if discovered:
                # Update configuration with discovered service
                self.ollama_base_url = discovered["url"]
                logger.info(f"🎯 Using discovered Ollama at: {discovered['url']} ({discovered['description']})")
                
                # Auto-select best model if current model not available
                if self.model not in discovered["models"]:
                    if discovered["vision_models"]:
                        # Prefer vision models for Agent-S2
                        self.model = discovered["vision_models"][0]
                        logger.info(f"🎨 Auto-selected vision model: {self.model}")
                    elif discovered["models"]:
                        # Fall back to first available model
                        self.model = discovered["models"][0]
                        logger.info(f"🤖 Auto-selected model: {self.model}")
                    else:
                        logger.error("No models available in Ollama. Please pull a model.")
                        logger.error("Recommended: ollama pull llama3.2-vision:11b")
                        return
                else:
                    logger.info(f"✅ Configured model '{self.model}' is available")
            else:
                # No auto-detection successful, try configured URL anyway
                logger.warning("Auto-detection failed, trying configured URL...")
                
            # Check Ollama health using direct HTTP call
            logger.info("Checking Ollama connectivity...")
            try:
                health_response = requests.get(f"{self.ollama_base_url}/api/tags", timeout=5)
                if health_response.status_code != 200:
                    logger.error(f"Ollama health check failed: {health_response.status_code}")
                    logger.error(f"Make sure Ollama is running and accessible at {self.ollama_base_url}")
                    return
                logger.info("✅ Ollama is reachable")
            except requests.RequestException as e:
                logger.error(f"Ollama is not reachable at {self.ollama_base_url}: {e}")
                logger.error("Ensure Ollama is running and listening on all interfaces (OLLAMA_HOST=0.0.0.0)")
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
                
                logger.info(f"Found {len(available_models)} models in Ollama")
                
                if self.model not in available_models:
                    logger.warning(f"Configured model '{self.model}' not found")
                    logger.info(f"Available models: {', '.join(available_models) if available_models else 'None'}")
                    
                    # Look for vision models as preferred fallback
                    vision_models = [m for m in available_models if 'vision' in m.lower() or 'llava' in m.lower()]
                    if vision_models:
                        self.model = vision_models[0]
                        logger.info(f"Using vision model as fallback: {self.model}")
                    elif available_models:
                        self.model = available_models[0]
                        logger.info(f"Using first available model as fallback: {self.model}")
                    else:
                        logger.error("No models available in Ollama. Please pull a model using: ollama pull llama3.2-vision:11b")
                        return
                else:
                    logger.info(f"✅ Model '{self.model}' is available")
                        
            except requests.RequestException as e:
                logger.error(f"Failed to connect to Ollama API: {e}")
                return
                
            # Initialize stealth mode
            try:
                stealth_result = await self.stealth_manager.initialize()
                if stealth_result["success"]:
                    logger.info(f"Stealth mode initialized with features: {stealth_result['features_enabled']}")
                else:
                    logger.warning(f"Stealth mode initialization had errors: {stealth_result['errors']}")
            except Exception as e:
                logger.error(f"Failed to initialize stealth mode: {e}")
                
            self.initialized = True
            logger.info(f"✅ AI handler initialized successfully")
            logger.info(f"   Provider: {self.provider}")
            logger.info(f"   Model: {self.model}")
            logger.info(f"   Base URL: {self.ollama_base_url}")
            
        except Exception as e:
            logger.error(f"Failed to initialize AI handler: {e}")
            self.initialized = False
            
    async def shutdown(self):
        """Shutdown AI handler"""
        self.initialized = False
        
    def _call_ollama(self, prompt: str, model: Optional[str] = None, system: Optional[str] = None, images: Optional[List[str]] = None) -> Dict[str, Any]:
        """Make a call to Ollama API
        
        Args:
            prompt: The prompt to send
            model: Model to use (defaults to self.model)
            system: Optional system prompt
            images: Optional list of base64-encoded images
            
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
            
        if images:
            # Ensure images is a proper list
            if not isinstance(images, list):
                logger.warning(f"images parameter is not a list: {type(images)}")
                images = [images] if images else []
            
            # Validate each image is a string
            validated_images = []
            for img in images:
                if isinstance(img, str):
                    validated_images.append(img)
                else:
                    logger.warning(f"Skipping non-string image: {type(img)}")
            
            payload["images"] = validated_images
            
        try:
            # Log payload structure for debugging
            logger.debug(f"Ollama payload keys: {list(payload.keys())}")
            if images:
                logger.debug(f"Number of images: {len(payload.get('images', []))}")
            
            response = requests.post(
                f"{self.ollama_base_url}/api/generate",
                json=payload,
                timeout=Config.AI_TIMEOUT
            )
            response.raise_for_status()
            return response.json()
            
        except requests.RequestException as e:
            logger.error(f"Ollama API call failed: {e}")
            if hasattr(e, 'response') and e.response:
                logger.error(f"Response status: {e.response.status_code}")
                logger.error(f"Response text: {e.response.text}")
            raise
        except Exception as e:
            logger.error(f"Unexpected error in Ollama call: {type(e).__name__}: {e}")
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
                
            # Execute the plan if we have valid actions
            actions_taken = []
            if ai_plan.get("plan"):
                # Convert plan format to actions format if needed
                ai_actions = []
                for step in ai_plan.get("plan", []):
                    if "action" in step and "parameters" in step:
                        action_dict = {"type": step["action"]}
                        action_dict.update(step.get("parameters", {}))
                        if "description" in step:
                            action_dict["description"] = step["description"]
                        ai_actions.append(action_dict)
                
                # Execute the actions
                if ai_actions:
                    try:
                        actions_taken = await self._execute_ai_actions(ai_actions, command_context=task)
                    except Exception as exec_error:
                        logger.error(f"Failed to execute AI actions: {exec_error}")
                        actions_taken = [{
                            "action": "execution_failed",
                            "result": f"Failed to execute actions: {exec_error}",
                            "status": "failed"
                        }]
            
            return {
                "success": True,
                "task": task,
                "summary": f"AI analyzed and executed task: {task}" if actions_taken else f"AI analyzed task: {task}",
                "actions_taken": actions_taken,
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
            
    async def _execute_ai_actions(self, ai_actions: List[Dict[str, Any]], 
                                 target_app: Optional[str] = None,
                                 command_context: Optional[str] = None) -> List[Dict[str, Any]]:
        """Execute a list of AI-generated actions
        
        Args:
            ai_actions: List of action dictionaries from AI
            target_app: Optional target application
            command_context: Optional context for logging
            
        Returns:
            List of executed action results
        """
        executed_actions = []
        
        # Inject target_app into all actions if specified and not already present
        if target_app:
            for action in ai_actions:
                if action.get("type") in ["click", "type", "key"] and "target_app" not in action:
                    action["target_app"] = target_app
        
        # Check if this is a screenshot save command
        is_screenshot_save_command = command_context and any(
            word in command_context.lower() for word in ["save", "capture", "screenshot"]
        )
        custom_filename = None
        if command_context and "save" in command_context.lower() and "as" in command_context.lower():
            # Try to extract filename from command like "save it as desktop_capture.png"
            import re
            filename_match = re.search(r'save.*?as\s+([^\s\.]+(?:\.[a-z]+)?)', command_context.lower())
            if filename_match:
                custom_filename = filename_match.group(1)
                if not custom_filename.endswith(('.png', '.jpg', '.jpeg')):
                    custom_filename += '.png'
        
        # Execute each action
        for action in ai_actions:
            try:
                if action.get("type") == "click":
                    x, y = action.get("x", 500), action.get("y", 300)
                    button = action.get("button", "left")  # Support right-click
                    action_target_app = action.get("target_app")
                    
                    if action_target_app:
                        # Use targeted click method
                        success, focused_window, focus_time = self.automation_service.click_targeted(
                            x=x, y=y, target_app=action_target_app, button=button
                        )
                        executed_actions.append({
                            "action": "click",
                            "parameters": {"x": x, "y": y, "button": button, "target_app": action_target_app},
                            "result": action.get("description", f"{button.title()}-clicked at position with target focus"),
                            "status": "success",
                            "focused_window": focused_window.title if focused_window else None,
                            "focus_time": focus_time
                        })
                    else:
                        # Use regular click method
                        if button == "right":
                            self.automation_service.right_click(x, y)
                        else:
                            self.automation_service.click(x, y)
                            
                        executed_actions.append({
                            "action": "click",
                            "parameters": {"x": x, "y": y, "button": button},
                            "result": action.get("description", f"{button.title()}-clicked at position"),
                            "status": "success"
                        })
                    
                elif action.get("type") == "type":
                    text = action.get("text", "")
                    action_target_app = action.get("target_app")
                    
                    if text:
                        if action_target_app:
                            # Use targeted typing method
                            success, focused_window, focus_time = self.automation_service.type_text_targeted(
                                text=text, target_app=action_target_app
                            )
                            executed_actions.append({
                                "action": "type",
                                "parameters": {"text": text, "target_app": action_target_app},
                                "result": action.get("description", "Typed text with target focus"),
                                "status": "success",
                                "focused_window": focused_window.title if focused_window else None,
                                "focus_time": focus_time
                            })
                        else:
                            # Use regular typing method
                            self.automation_service.type_text(text)
                            executed_actions.append({
                                "action": "type",
                                "parameters": {"text": text},
                                "result": action.get("description", "Typed text"),
                                "status": "success"
                            })
                        
                elif action.get("type") == "key":
                    key = action.get("key", "")
                    action_target_app = action.get("target_app")
                    
                    if key:
                        if action_target_app:
                            # Use targeted key press method
                            success, focused_window, focus_time = self.automation_service.press_key_targeted(
                                keys=key, target_app=action_target_app
                            )
                            executed_actions.append({
                                "action": "key",
                                "parameters": {"key": key, "target_app": action_target_app},
                                "result": action.get("description", f"Pressed {key} with target focus"),
                                "status": "success",
                                "focused_window": focused_window.title if focused_window else None,
                                "focus_time": focus_time
                            })
                        else:
                            # Use regular key press method
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
                    
                elif action.get("type") == "screenshot" or (action.get("type") in ["capture", "save"] and is_screenshot_save_command):
                    # Take an additional screenshot and save with custom name if specified
                    if custom_filename:
                        save_path = f"/tmp/{custom_filename}"
                    else:
                        save_path = f"/tmp/agent-s2-user-screenshot-{int(time.time())}.png"
                    
                    try:
                        self.screenshot_service.capture_to_file(save_path, format="png")
                        executed_actions.append({
                            "action": "screenshot_save",
                            "parameters": {"filename": save_path},
                            "result": f"Screenshot saved to {save_path}",
                            "status": "success",
                            "screenshot_path": save_path
                        })
                    except Exception as save_error:
                        executed_actions.append({
                            "action": "screenshot_save",
                            "parameters": {"filename": save_path},
                            "result": f"Failed to save screenshot: {save_error}",
                            "status": "failed"
                        })
                    
            except Exception as action_error:
                executed_actions.append({
                    "action": action.get("type", "unknown"),
                    "parameters": action,
                    "result": f"Failed: {action_error}",
                    "status": "failed"
                })
        
        return executed_actions

    async def execute_command(self, command: str, context: Optional[str] = None, 
                            target_app: Optional[str] = None) -> Dict[str, Any]:
        """Execute a natural language command
        
        Args:
            command: Natural language command
            context: Optional context
            target_app: Optional target application for actions
            
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
            
            # Also save screenshot to file for user reference
            screenshot_filename = f"/tmp/agent-s2-screenshot-{int(time.time())}.png"
            try:
                self.screenshot_service.capture_to_file(screenshot_filename, format="png")
                screenshot_saved = True
                screenshot_path = screenshot_filename
            except Exception as e:
                logger.warning(f"Failed to save screenshot to file: {e}")
                screenshot_saved = False
                screenshot_path = None
            
            actions_taken.append({
                "action": "screenshot",
                "result": f"Captured current screen state{f' → Saved to {screenshot_path}' if screenshot_saved else ''}",
                "screenshot_path": screenshot_path if screenshot_saved else None
            })
            
            # Create AI prompt for command execution  
            target_info = f"\n\nTARGET APPLICATION: {target_app}\n- All actions will be focused on this application\n- Include \"target_app\": \"{target_app}\" in all automation actions" if target_app else ""
            
            available_apps = self.automation_service.window_manager.get_running_applications()
            apps_info = f"\nAvailable Applications: {', '.join(available_apps)}" if available_apps else ""
            
            system_prompt = f"""You are an automation assistant that can see and interact with a computer screen. Given a screenshot and a natural language command, analyze what you see and provide specific automation actions.

IMPORTANT SYSTEM INFORMATION:
- Desktop Environment: Fluxbox window manager (NOT Windows or GNOME)
- The screenshot shows the ENTIRE screen (1920x1080 pixels)
- Provide ABSOLUTE screen coordinates, NOT window-relative coordinates
- The top-left corner of the screen is (0, 0)
- The bottom-right corner is (1920, 1080){apps_info}{target_info}

FLUXBOX WINDOW MANAGEMENT:
- Windows DO NOT have visible close/minimize/maximize buttons (X buttons)
- To close windows: RIGHT-click on the window title bar, then click "Close" in the context menu
- To minimize windows: RIGHT-click on title bar, then click "Minimize" 
- To maximize windows: RIGHT-click on title bar, then click "Maximize"
- Window title bars are thin dark bars at the top of application windows

Respond with JSON format:
{{
  "actions": [
    {{"type": "click", "x": 100, "y": 200, "description": "Click on [specific element you see]"{', "target_app": "' + target_app + '"' if target_app else ""}}},
    {{"type": "type", "text": "Hello", "description": "Type text in [specific field]"{', "target_app": "' + target_app + '"' if target_app else ""}}},
    {{"type": "key", "key": "Enter", "description": "Press Enter"{', "target_app": "' + target_app + '"' if target_app else ""}}},
    {{"type": "wait", "duration": 1.0, "description": "Wait 1 second"}}
  ],
  "reasoning": "I can see [describe what you see]. To [command], I will [explain approach]"
}}

{'IMPORTANT: When target_app is specified, always include "target_app": "' + target_app + '" in click, type, and key actions.' if target_app else ''}

CLOSING WINDOWS EXAMPLE:
To close a window:
1. {{"type": "click", "x": 200, "y": 7, "description": "Right-click on window title bar", "button": "right"}}
2. {{"type": "wait", "duration": 0.5, "description": "Wait for context menu"}}
3. {{"type": "click", "x": 134, "y": 147, "description": "Click Close in context menu"}}

Available action types: click, type, key, scroll, wait, drag
For click actions, add "button": "right" for right-clicks

Be specific about what you see and provide EXACT ABSOLUTE screen coordinates!"""

            context_str = f" Context: {context}" if context else ""
            user_prompt = f"""Command: {command}{context_str}

Analyze the screenshot and provide specific automation actions to execute this command."""

            # Convert screenshot to base64
            # Data URL format is "data:image/png;base64,<base64data>"
            try:
                if "base64," in screenshot:
                    parts = screenshot.split("base64,")
                    if len(parts) > 1:
                        screenshot_base64 = parts[1]
                    else:
                        logger.warning("Screenshot split did not produce expected parts")
                        screenshot_base64 = screenshot
                else:
                    screenshot_base64 = screenshot
            except Exception as split_error:
                logger.error(f"Error splitting screenshot data: {type(split_error).__name__}: {split_error}")
                screenshot_base64 = screenshot
            
            # Log for debugging
            logger.info(f"Using AI model: {self.model}")
            try:
                logger.info(f"Screenshot type: {type(screenshot_base64)}, length: {len(screenshot_base64)}")
                if isinstance(screenshot_base64, str) and len(screenshot_base64) > 50:
                    # Use string slicing safely
                    preview = screenshot_base64[0:50]
                    logger.info(f"First 50 chars of base64: {preview}...")
                else:
                    # Use repr safely
                    repr_str = repr(screenshot_base64)
                    if len(repr_str) > 100:
                        repr_str = repr_str[0:100]
                    logger.info(f"Screenshot data: {repr_str}...")
            except Exception as log_error:
                logger.warning(f"Error logging screenshot data: {log_error}")
            
            try:
                # Get AI analysis with vision model
                logger.info(f"Calling Ollama with model: {self.model}")
                
                # Ensure screenshot_base64 is a string and create a proper list
                if not isinstance(screenshot_base64, str):
                    logger.warning(f"screenshot_base64 is not a string: {type(screenshot_base64)}")
                    screenshot_base64 = str(screenshot_base64)
                
                # Create images list explicitly to avoid any slice issues
                images_list = []
                images_list.append(screenshot_base64)
                
                ai_response = self._call_ollama(
                    user_prompt, 
                    model=self.model,  # Use configured AI model
                    system=system_prompt,
                    images=images_list
                )
                ai_text = ai_response.get("response", "")
                logger.info(f"Vision model response received, length: {len(ai_text)}")
            except Exception as e:
                logger.error(f"Vision model error: {type(e).__name__}: {str(e)}")
                logger.error(f"Full error details: {repr(e)}")
                # Fallback to text-only model without image
                logger.info("Falling back to text-only analysis")
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
                    ai_actions, reasoning = self._parse_command_fallback(command, target_app)
            except (json.JSONDecodeError, AttributeError):
                # Fallback: basic command parsing
                ai_actions, reasoning = self._parse_command_fallback(command, target_app)
            
            # Execute the actions using the shared execution method
            executed_actions = await self._execute_ai_actions(
                ai_actions, 
                target_app=target_app,
                command_context=command
            )
            
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
            
    def _parse_command_fallback(self, command: str, target_app: Optional[str] = None) -> tuple:
        """Fallback command parsing when AI fails"""
        command_lower = command.lower()
        actions = []
        
        if "click" in command_lower:
            action = {"type": "click", "x": 500, "y": 300, "description": "Click at center"}
            if target_app:
                action["target_app"] = target_app
            actions.append(action)
        if "type" in command_lower or "write" in command_lower:
            # Try to extract text after "type" or "write"
            import re
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