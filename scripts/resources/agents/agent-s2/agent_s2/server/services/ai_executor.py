"""AI Executor for Agent S2

Handles execution of AI-driven actions, commands, and automation orchestration.
"""

import os
import logging
import json
import subprocess
import time
import re
import asyncio
import sys
from typing import Dict, Any, Optional, List
from datetime import datetime
from pathlib import Path

from ...config import Config
from .capture import ScreenshotService
from .automation import AutomationService
from .ai_service_manager import AIServiceManager
from .ai_planner import AIPlanner
from .ai_security_validator import AISecurityValidator
from ...environment.context import ModeContext
from ...environment.discovery import EnvironmentDiscovery
from .browser_health import browser_monitor

# Import shortcut detector if available
try:
    # Use configurable path or default
    shortcuts_path = Path(Config.SHORTCUTS_PATH)
    if shortcuts_path.exists():
        sys.path.insert(0, str(shortcuts_path))
        from detector import ShortcutDetector
    else:
        ShortcutDetector = None
except ImportError:
    ShortcutDetector = None

logger = logging.getLogger(__name__)


class AIExecutor:
    """Handles execution of AI-driven actions and commands"""
    
    def __init__(self, 
                 service_manager: AIServiceManager, 
                 planner: AIPlanner, 
                 security_validator: AISecurityValidator):
        """Initialize AI executor
        
        Args:
            service_manager: AI service manager for API calls
            planner: AI planner for task planning
            security_validator: Security validator for action filtering
        """
        self.service_manager = service_manager
        self.planner = planner
        self.security_validator = security_validator
        
        # Core services for execution
        self.screenshot_service = ScreenshotService()
        self.automation_service = AutomationService()
        
        # Initialize shortcut detector if available
        self.shortcut_detector = None
        if ShortcutDetector:
            try:
                shortcuts_path = Config.SHORTCUTS_PATH
                self.shortcut_detector = ShortcutDetector(shortcuts_path)
                logger.info("Keyboard shortcut detector initialized in executor")
            except Exception as e:
                logger.warning(f"Failed to initialize shortcut detector in executor: {e}")
        
        # Initialize environment context for application discovery
        try:
            self.environment_discovery = EnvironmentDiscovery()
            self.mode_context = ModeContext(discovery=self.environment_discovery)
            logger.info("Environment discovery and mode context initialized in executor")
        except Exception as e:
            logger.warning(f"Failed to initialize environment context in executor: {e}")
            self.environment_discovery = None
            self.mode_context = None
    
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
        if not self.service_manager.is_ready():
            raise RuntimeError("AI service manager not initialized")
            
        logger.info(f"execute_action called with task: {task}")
        
        # Ensure clean browser state before starting task
        logger.info("Checking browser state before task execution...")
        try:
            clean_state = browser_monitor.ensure_clean_state()
            if not clean_state:
                logger.warning("Failed to ensure clean browser state, proceeding anyway")
        except Exception as e:
            logger.error(f"Error during browser state cleanup: {e}")
            # Continue anyway - don't block task execution
        
        debug_info = {}
        screenshot_included = False
        
        try:
            # Take screenshot if not provided
            screenshot_data = None
            if not screenshot:
                screenshot_data = self.screenshot_service.capture()
                screenshot = screenshot_data["data"]
                screenshot_included = True
                
                # Validate screenshot before proceeding
                screenshot_valid = self._validate_screenshot(screenshot)
                
                debug_info["screenshot_capture"] = {
                    "captured": True,
                    "format": screenshot_data.get("format", "unknown"),
                    "size": screenshot_data.get("size", {}),
                    "data_length": len(screenshot) if screenshot else 0,
                    "valid": screenshot_valid
                }
                
                if not screenshot_valid:
                    logger.warning("Screenshot validation failed - proceeding without visual context")
                    screenshot = None
                    screenshot_included = False
            else:
                screenshot_included = True
                debug_info["screenshot_capture"] = {
                    "captured": False,
                    "provided": True,
                    "data_length": len(screenshot) if screenshot else 0
                }
                
            # Get window context
            window_context = self._get_window_context()
            logger.info(f"Window context in execute_action: {window_context[:200]}...")
            debug_info["window_context"] = window_context
            
            # Use planner to create task execution plan
            logger.info(f"Calling planner with task='{task}', screenshot_included={screenshot_included}")
            plan_result = await self.planner.plan_task_execution(task, context, screenshot, window_context)
            logger.info(f"Planner returned: keys={list(plan_result.keys())}")
            
            # Collect debug information from planner
            if "debug_info" in plan_result:
                logger.info("Debug info found in plan_result")
                debug_info["from_planner"] = plan_result["debug_info"]
            else:
                # Fallback: create basic debug info if planner didn't provide it
                logger.warning("Planner did not return debug_info, creating fallback debug info")
                debug_info["from_planner"] = {
                    "system_prompt": "Planner debug info not available",
                    "user_prompt": f"Task: {task}",
                    "screenshot_included": screenshot_included,
                    "context": str(context) if context else None,
                    "window_context_included": bool(window_context),
                    "planner_error": "Debug info not returned by planner"
                }
            
            # Always log what debug_info contains at this point
            logger.info(f"Debug info after planner: keys={list(debug_info.keys())}")
            
            # Execute the plan if we have valid actions
            actions_taken = []
            if plan_result.get("plan"):
                # Validate and convert plan format to actions format
                ai_actions = []
                validation_errors = []
                
                for i, step in enumerate(plan_result.get("plan", [])):
                    try:
                        validated_action = self._validate_and_convert_action(step, i)
                        if validated_action:
                            ai_actions.append(validated_action)
                        else:
                            validation_errors.append(f"Step {i+1}: Invalid action structure")
                    except Exception as e:
                        validation_errors.append(f"Step {i+1}: {str(e)}")
                        logger.warning(f"Action validation failed for step {i+1}: {e}")
                
                # Log validation results
                if validation_errors:
                    logger.warning(f"Action validation errors: {validation_errors}")
                    debug_info["validation_errors"] = validation_errors
                
                logger.info(f"Validated {len(ai_actions)} actions out of {len(plan_result.get('plan', []))} planned steps")
                
                # Execute the actions
                if ai_actions:
                    try:
                        logger.info(f"AI actions to execute: {ai_actions}")
                        # Extract security config from context
                        security_config = context.get("security_config") if context else None
                        actions_taken = await self._execute_ai_actions(
                            ai_actions, 
                            command_context=task,
                            security_config=security_config
                        )
                    except Exception as exec_error:
                        logger.error(f"Failed to execute AI actions: {exec_error}")
                        actions_taken = [{
                            "action": "execution_failed",
                            "result": f"Failed to execute actions: {exec_error}",
                            "status": "failed"
                        }]
            
            # Log final debug_info before returning
            logger.info(f"Final debug_info being returned: keys={list(debug_info.keys())}")
            if "from_planner" in debug_info:
                logger.info(f"from_planner debug_info keys: {list(debug_info['from_planner'].keys())}")
            
            return {
                "success": True,
                "task": task,
                "summary": f"AI analyzed and executed task: {task}" if actions_taken else f"AI analyzed task: {task}",
                "actions_taken": actions_taken,
                "plan": plan_result.get("plan", []),
                "reasoning": plan_result.get("reasoning", "AI provided task analysis"),
                "estimated_duration": plan_result.get("estimated_duration", "Unknown"),
                "ai_model": self.service_manager.model,
                "raw_ai_response": plan_result.get("raw_ai_response", ""),
                "debug_info": debug_info
            }
            
        except Exception as e:
            logger.error(f"AI action execution failed: {e}")
            return {
                "success": False,
                "task": task,
                "summary": "Execution failed",
                "error": str(e),
                "debug_info": debug_info
            }
    
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
        if not self.service_manager.is_ready():
            raise RuntimeError("AI service manager not initialized")
            
        logger.info(f"Executing AI command: {command}")
        
        actions_taken = []
        
        try:
            # Take initial screenshot for AI analysis
            screenshot = self.screenshot_service.capture()
            
            # Also save screenshot to file for user reference
            screenshot_filename = f"{Config.TEMP_DIR}/agent-s2-screenshot-{int(time.time())}.png"
            try:
                self.screenshot_service.capture_to_file(screenshot_filename, format="png")
                screenshot_saved = True
                screenshot_path = screenshot_filename
            except (OSError, PermissionError, ValueError) as e:
                logger.warning(f"Failed to save screenshot to file: {e}")
                screenshot_saved = False
                screenshot_path = None
            except Exception as e:
                logger.error(f"Unexpected error saving screenshot: {e}")
                screenshot_saved = False
                screenshot_path = None
            
            actions_taken.append({
                "action": "screenshot",
                "result": f"Captured current screen state{f' → Saved to {screenshot_path}' if screenshot_saved else ''}",
                "screenshot_path": screenshot_path if screenshot_saved else None
            })
            
            # Get comprehensive window context
            window_context = self._get_window_context()
            logger.info(f"Window context retrieved: {window_context[:200]}...")
            
            # Create AI prompt for command execution  
            target_info = f"\n\nTARGET APPLICATION: {target_app}\n- All actions will be focused on this application\n- Include \"target_app\": \"{target_app}\" in all automation actions" if target_app else ""
            
            available_apps = self.automation_service.window_manager.get_running_applications()
            apps_info = f"\nAvailable Applications: {', '.join(available_apps)}" if available_apps else ""
            
            # Get keyboard shortcuts context if available
            shortcuts_context = ""
            if self.shortcut_detector:
                try:
                    shortcuts_context = self.shortcut_detector.format_shortcuts_context(command)
                except Exception as e:
                    logger.warning(f"Failed to get shortcuts context: {e}")
            
            system_prompt = self._build_command_system_prompt(apps_info, target_info, window_context, shortcuts_context, target_app)
            
            context_str = f" Context: {context}" if context else ""
            user_prompt = f"""Command: {command}{context_str}

Analyze the screenshot and provide specific automation actions to execute this command."""

            # Convert screenshot to base64
            screenshot_base64 = self._prepare_screenshot_for_ai(screenshot)
            
            # Log AI input for debugging
            self._log_ai_input_debug(system_prompt, user_prompt, screenshot_base64)
            
            try:
                # Get AI analysis with vision model
                logger.info(f"Calling Ollama with model: {self.service_manager.model}")
                
                # Create images list explicitly
                images_list = [screenshot_base64] if screenshot_base64 else []
                
                ai_response = self.service_manager.call_ollama(
                    user_prompt, 
                    model=self.service_manager.model,
                    system=system_prompt,
                    images=images_list
                )
                ai_text = ai_response.get("response", "")
                logger.info(f"Vision model response received, length: {len(ai_text)}")
                
                # Log AI output for debugging
                self._log_ai_output_debug(ai_text)
                
            except Exception as e:
                logger.error(f"Vision model error: {type(e).__name__}: {str(e)}")
                # Fallback to text-only model without image
                logger.info("Falling back to text-only analysis")
                ai_response = self.service_manager.call_ollama(user_prompt, system=system_prompt)
                ai_text = ai_response.get("response", "")
            
            # Parse AI response for actions
            try:
                json_match = re.search(r'\{.*\}', ai_text, re.DOTALL)
                if json_match:
                    ai_result = json.loads(json_match.group())
                    ai_actions = ai_result.get("actions", [])
                    reasoning = ai_result.get("reasoning", "AI provided action plan")
                else:
                    # Fallback: basic command parsing
                    ai_actions, reasoning = self.planner.parse_command_fallback(command, target_app)
            except (json.JSONDecodeError, AttributeError):
                # Fallback: basic command parsing
                ai_actions, reasoning = self.planner.parse_command_fallback(command, target_app)
            
            # Execute the actions
            logger.info(f"AI actions to execute in command: {ai_actions}")
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
                "ai_model": self.service_manager.model,
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
    
    def _validate_and_convert_action(self, step: Dict[str, Any], step_index: int) -> Optional[Dict[str, Any]]:
        """Validate and convert a plan step to an executable action
        
        Args:
            step: Plan step from AI response
            step_index: Index of the step for error reporting
            
        Returns:
            Validated action dictionary or None if invalid
            
        Raises:
            ValueError: If step is malformed
        """
        if not isinstance(step, dict):
            raise ValueError(f"Step must be a dictionary, got {type(step)}")
        
        # Check for required fields
        if "action" not in step:
            raise ValueError("Missing required 'action' field")
        
        action_type = step["action"]
        parameters = step.get("parameters", {})
        description = step.get("description", f"Step {step_index + 1}")
        
        # Validate action type
        valid_actions = ["click", "type", "key", "scroll", "wait", "launch_app", "screenshot", "manual_review"]
        if action_type not in valid_actions:
            logger.warning(f"Unknown action type '{action_type}' in step {step_index + 1}, using manual_review")
            action_type = "manual_review"
            parameters = {}
        
        # Validate parameters based on action type
        if action_type == "click":
            if not isinstance(parameters, dict):
                parameters = {"x": 500, "y": 300}  # Default center click
            else:
                # Ensure x, y are integers
                try:
                    parameters["x"] = int(parameters.get("x", 500))
                    parameters["y"] = int(parameters.get("y", 300))
                except (ValueError, TypeError):
                    parameters = {"x": 500, "y": 300}
                    
        elif action_type == "type":
            if not isinstance(parameters, dict) or "text" not in parameters:
                raise ValueError("Type action requires 'text' parameter")
            if not isinstance(parameters["text"], str):
                parameters["text"] = str(parameters["text"])
                
        elif action_type == "key":
            if not isinstance(parameters, dict) or "key" not in parameters:
                parameters = {"key": "Enter"}  # Default key
            
        elif action_type == "launch_app":
            if not isinstance(parameters, dict) or "app_name" not in parameters:
                parameters = {"app_name": "Firefox ESR"}  # Default app
                
        elif action_type == "wait":
            if not isinstance(parameters, dict):
                parameters = {"seconds": 2}
            else:
                try:
                    parameters["seconds"] = float(parameters.get("seconds", 2))
                except (ValueError, TypeError):
                    parameters["seconds"] = 2
                    
        elif action_type == "scroll":
            if not isinstance(parameters, dict):
                parameters = {"direction": "down", "amount": 3}
            else:
                parameters["direction"] = parameters.get("direction", "down")
                try:
                    parameters["amount"] = int(parameters.get("amount", 3))
                except (ValueError, TypeError):
                    parameters["amount"] = 3
        
        # Create validated action
        validated_action = {
            "type": action_type,
            "description": description
        }
        validated_action.update(parameters)
        
        logger.debug(f"Validated step {step_index + 1}: {action_type} with parameters {parameters}")
        return validated_action
    
    async def _execute_ai_actions(self, ai_actions: List[Dict[str, Any]], 
                                 target_app: Optional[str] = None,
                                 command_context: Optional[str] = None,
                                 security_config: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
        """Execute a list of AI-generated actions
        
        This is the core action execution engine that handles all types of automation actions.
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
        custom_filename = self._extract_screenshot_filename(command_context)
        
        # Execute each action
        for action in ai_actions:
            try:
                if action.get("type") == "click":
                    executed_actions.append(await self._execute_click_action(action))
                    
                elif action.get("type") == "type":
                    executed_actions.append(await self._execute_type_action(action, security_config))
                    
                elif action.get("type") in ["key", "key_press"]:
                    executed_actions.append(await self._execute_key_action(action))
                    
                elif action.get("type") == "wait":
                    executed_actions.append(await self._execute_wait_action(action))
                    
                elif action.get("type") == "screenshot" or (action.get("type") in ["capture", "save"] and is_screenshot_save_command):
                    executed_actions.append(await self._execute_screenshot_action(action, custom_filename))
                    
                elif action.get("type") == "launch_app":
                    executed_actions.append(await self._execute_launch_app_action(action))
                    
            except Exception as action_error:
                executed_actions.append({
                    "action": action.get("type", "unknown"),
                    "parameters": action,
                    "result": f"Failed: {action_error}",
                    "status": "failed"
                })
        
        return executed_actions
    
    async def _execute_click_action(self, action: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a click action"""
        x, y = action.get("x", 500), action.get("y", 300)
        button = action.get("button", "left")
        action_target_app = action.get("target_app")
        
        if action_target_app:
            # Use targeted click method
            success, focused_window, focus_time = self.automation_service.click_targeted(
                x=x, y=y, target_app=action_target_app, button=button
            )
            return {
                "action": "click",
                "parameters": {"x": x, "y": y, "button": button, "target_app": action_target_app},
                "result": action.get("description", f"{button.title()}-clicked at position with target focus"),
                "status": "success",
                "focused_window": focused_window.title if focused_window else None,
                "focus_time": focus_time
            }
        else:
            # Use regular click method
            if button == "right":
                self.automation_service.right_click(x, y)
            else:
                self.automation_service.click(x, y)
                
            return {
                "action": "click",
                "parameters": {"x": x, "y": y, "button": button},
                "result": action.get("description", f"{button.title()}-clicked at position"),
                "status": "success"
            }
    
    async def _execute_type_action(self, action: Dict[str, Any], security_config: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Execute a type action with security validation"""
        text = action.get("text", "")
        action_target_app = action.get("target_app")
        
        if not text:
            return {
                "action": "type",
                "parameters": {"text": text},
                "result": "No text to type",
                "status": "skipped"
            }
        
        # Validate text for security
        validation_result = self.security_validator.validate_text_action(text, security_config)
        
        if not validation_result.valid:
            logger.warning(f"Text validation failed: {validation_result.reason}")
            result = {
                "action": "type",
                "parameters": {"text": text},
                "result": validation_result.blocked_reason or f"BLOCKED: {validation_result.reason}",
                "status": "blocked",
                "security_reason": validation_result.reason
            }
            
            # If we have a suggested action, use that instead
            if validation_result.suggested_action:
                logger.info(f"Using suggested text: {validation_result.suggested_action['text']}")
                text = validation_result.suggested_action["text"]
                result["suggested_url"] = text
            else:
                # Skip this action entirely
                return result
        
        # Proceed with typing
        if action_target_app:
            # Use targeted typing method
            success, focused_window, focus_time = self.automation_service.type_text_targeted(
                text=text, target_app=action_target_app
            )
            return {
                "action": "type",
                "parameters": {"text": text, "target_app": action_target_app},
                "result": action.get("description", "Typed text with target focus"),
                "status": "success",
                "focused_window": focused_window.title if focused_window else None,
                "focus_time": focus_time
            }
        else:
            # Use regular typing method
            self.automation_service.type_text(text)
            return {
                "action": "type",
                "parameters": {"text": text},
                "result": action.get("description", "Typed text"),
                "status": "success"
            }
    
    async def _execute_key_action(self, action: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a key press action"""
        key = action.get("key", "")
        action_target_app = action.get("target_app")
        
        if not key:
            return {
                "action": "key",
                "parameters": {"key": key},
                "result": "No key specified",
                "status": "skipped"
            }
        
        logger.info(f"Executing key press: '{key}'")
        if action_target_app:
            # Use targeted key press method
            success, focused_window, focus_time = self.automation_service.press_key_targeted(
                keys=key, target_app=action_target_app
            )
            return {
                "action": "key",
                "parameters": {"key": key, "target_app": action_target_app},
                "result": action.get("description", f"Pressed {key} with target focus"),
                "status": "success",
                "focused_window": focused_window.title if focused_window else None,
                "focus_time": focus_time
            }
        else:
            # Use regular key press method
            self.automation_service.press_key([key])
            return {
                "action": "key",
                "parameters": {"key": key},
                "result": action.get("description", f"Pressed {key}"),
                "status": "success"
            }
    
    async def _execute_wait_action(self, action: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a wait action"""
        duration = action.get("duration", 1.0)
        await asyncio.sleep(duration)
        return {
            "action": "wait",
            "parameters": {"duration": duration},
            "result": action.get("description", f"Waited {duration}s"),
            "status": "success"
        }
    
    async def _execute_screenshot_action(self, action: Dict[str, Any], custom_filename: Optional[str] = None) -> Dict[str, Any]:
        """Execute a screenshot save action"""
        if custom_filename:
            save_path = f"{Config.SCREENSHOT_SAVE_DIR}/{custom_filename}"
        else:
            save_path = f"{Config.SCREENSHOT_SAVE_DIR}/agent-s2-user-screenshot-{int(time.time())}.png"
        
        try:
            self.screenshot_service.capture_to_file(save_path, format="png")
            return {
                "action": "screenshot_save",
                "parameters": {"filename": save_path},
                "result": f"Screenshot saved to {save_path}",
                "status": "success",
                "screenshot_path": save_path
            }
        except Exception as save_error:
            return {
                "action": "screenshot_save",
                "parameters": {"filename": save_path},
                "result": f"Failed to save screenshot: {save_error}",
                "status": "failed"
            }
    
    async def _execute_launch_app_action(self, action: Dict[str, Any]) -> Dict[str, Any]:
        """Execute an application launch action"""
        app_name = action.get("app_name", "")
        
        if not app_name:
            return {
                "action": "launch_app",
                "parameters": {"app_name": app_name},
                "result": "Failed: No app_name specified",
                "status": "failed"
            }
        
        if not self.mode_context:
            return {
                "action": "launch_app",
                "parameters": {"app_name": app_name},
                "result": "Failed: Application discovery not initialized",
                "status": "failed"
            }
            
        try:
            # Get application info
            app_info = self.mode_context.get_application_info(app_name)
            
            if not app_info:
                return {
                    "action": "launch_app",
                    "parameters": {"app_name": app_name},
                    "result": f"Failed: Application '{app_name}' not found",
                    "status": "failed"
                }
            
            # Launch the application using subprocess
            command = app_info.get("command", "")
            if not command:
                return {
                    "action": "launch_app",
                    "parameters": {"app_name": app_name},
                    "result": f"Failed: No command found for '{app_name}'",
                    "status": "failed"
                }
            
            # Launch in background with proper display
            env = os.environ.copy()
            env["DISPLAY"] = Config.DISPLAY
            
            process = subprocess.Popen(
                [command],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                env=env,
                start_new_session=True
            )
            
            # Give the app a moment to start
            await asyncio.sleep(2.0)
            
            return {
                "action": "launch_app",
                "parameters": {"app_name": app_name, "command": command},
                "result": f"Successfully launched {app_info.get('name', app_name)}",
                "status": "success",
                "process_id": process.pid,
                "app_info": app_info
            }
            
        except Exception as launch_error:
            return {
                "action": "launch_app",
                "parameters": {"app_name": app_name},
                "result": f"Failed to launch '{app_name}': {launch_error}",
                "status": "failed"
            }
    
    def _get_window_context(self) -> str:
        """Get comprehensive window context for AI"""
        try:
            # Get all windows
            all_windows = self.automation_service.window_manager.list_all_windows()
            
            # Get focused window
            focused_window = self.automation_service.window_manager.get_focused_window()
            
            # Format window information
            context_parts = []
            
            # Add keyboard shortcuts for active window if available
            if self.shortcut_detector and focused_window:
                try:
                    shortcuts_data = self.shortcut_detector.get_shortcuts_for_window(
                        window_title=focused_window.title,
                        window_class=focused_window.app_name
                    )
                    if shortcuts_data.get('priority_actions'):
                        context_parts.append("\nKEYBOARD SHORTCUTS AVAILABLE:")
                        for task, actions in shortcuts_data['priority_actions'].items():
                            task_formatted = task.replace('_', ' ').title()
                            context_parts.append(f"- {task_formatted}: Use keyboard shortcut instead of clicking")
                except Exception as e:
                    logger.debug(f"Could not get shortcuts for window: {e}")
            
            # Current active window
            if focused_window:
                # Calculate focus point (center of title bar area)
                focus_x = focused_window.geometry['x'] + (focused_window.geometry['width'] // 2)
                focus_y = focused_window.geometry['y'] + 20  # 20 pixels down from top for title bar
                
                context_parts.append(f"ACTIVE WINDOW: {focused_window.app_name} - \"{focused_window.title}\"")
                context_parts.append(f"  Position: ({focused_window.geometry['x']}, {focused_window.geometry['y']})")
                context_parts.append(f"  Size: {focused_window.geometry['width']}x{focused_window.geometry['height']}")
                context_parts.append(f"  Focus Point: ({focus_x}, {focus_y}) - Click here to ensure this window is focused")
                context_parts.append(f"  Window ID: {focused_window.window_id}")
            else:
                context_parts.append("ACTIVE WINDOW: None detected")
            
            # All windows list
            context_parts.append("\nALL WINDOWS:")
            for window in all_windows:
                # Calculate focus point for each window
                focus_x = window.geometry['x'] + (window.geometry['width'] // 2)
                focus_y = window.geometry['y'] + 20  # 20 pixels down from top for title bar
                
                is_active = " [ACTIVE]" if window.is_focused else ""
                context_parts.append(f"- {window.app_name}: \"{window.title}\"{is_active}")
                context_parts.append(f"  Position: ({window.geometry['x']}, {window.geometry['y']}) Size: {window.geometry['width']}x{window.geometry['height']}")
                context_parts.append(f"  Focus Point: ({focus_x}, {focus_y}) - Click here to focus this window")
            
            # Important note about window focus
            context_parts.append("\nIMPORTANT: The ACTIVE WINDOW receives keyboard input. To interact with a different window:")
            context_parts.append("1. Click on the window's Focus Point coordinates shown above to focus it")
            context_parts.append("2. Or use target_app parameter in actions to auto-focus before interaction")
            context_parts.append("3. ALWAYS use the Focus Point coordinates when you need to click on a window to focus it")
            
            # Add available applications information
            if self.mode_context:
                try:
                    applications = self.mode_context.context_data.get("applications", {})
                    if applications:
                        context_parts.append("\nAVAILABLE APPLICATIONS (can be launched):")
                        for app_name, app_info in applications.items():
                            description = app_info.get("description", "No description")
                            command = app_info.get("command", "Unknown command")
                            launcher = app_info.get("launcher", "Use launch_app action")
                            context_parts.append(f"- {app_name}: {description}")
                            context_parts.append(f"  Command: {command}")
                            context_parts.append(f"  How to launch: {launcher}")
                        
                        context_parts.append("\nLAUNCHING APPLICATIONS:")
                        context_parts.append("To launch any application above, use: {\"type\": \"launch_app\", \"app_name\": \"ApplicationName\"}")
                        context_parts.append("Example: {\"type\": \"launch_app\", \"app_name\": \"Firefox ESR\"}")
                    else:
                        context_parts.append("\nAVAILABLE APPLICATIONS: None detected")
                except Exception as e:
                    logger.debug(f"Failed to get applications context: {e}")
                    context_parts.append("\nAVAILABLE APPLICATIONS: Error retrieving application list")
            else:
                context_parts.append("\nAVAILABLE APPLICATIONS: Application discovery not initialized")
            
            return "\n".join(context_parts)
            
        except Exception as e:
            logger.warning(f"Failed to get window context: {e}")
            return "Window context unavailable"
    
    def _extract_screenshot_filename(self, command_context: Optional[str]) -> Optional[str]:
        """Extract custom filename from screenshot save commands"""
        if not command_context:
            return None
            
        if "save" in command_context.lower() and "as" in command_context.lower():
            # Try to extract filename from command like "save it as desktop_capture.png"
            filename_match = re.search(r'save.*?as\s+([^\s\.]+(?:\.[a-z]+)?)', command_context.lower())
            if filename_match:
                custom_filename = filename_match.group(1)
                if not custom_filename.endswith(('.png', '.jpg', '.jpeg')):
                    custom_filename += '.png'
                return custom_filename
        return None
    
    def _prepare_screenshot_for_ai(self, screenshot: str) -> str:
        """Prepare screenshot data for AI processing"""
        try:
            if "base64," in screenshot:
                parts = screenshot.split("base64,")
                if len(parts) > 1:
                    screenshot_base64 = parts[1].replace('\n', '').replace('\r', '').strip()
                else:
                    logger.warning("Screenshot split did not produce expected parts")
                    screenshot_base64 = screenshot
            else:
                screenshot_base64 = screenshot
                
            # Ensure it's a string
            if not isinstance(screenshot_base64, str):
                logger.warning(f"screenshot_base64 is not a string: {type(screenshot_base64)}")
                screenshot_base64 = str(screenshot_base64)
                
            return screenshot_base64
        except Exception as e:
            logger.error(f"Error preparing screenshot for AI: {e}")
            return screenshot
    
    def _build_command_system_prompt(self, apps_info: str, target_info: str, window_context: str, shortcuts_context: str, target_app: Optional[str] = None) -> str:
        """Build the system prompt for command execution"""
        return f"""You are an automation assistant that can see and interact with a computer screen. Given a screenshot and a natural language command, analyze what you see and provide specific automation actions.

IMPORTANT SYSTEM INFORMATION:
- Desktop Environment: Fluxbox window manager (NOT Windows or GNOME)
- The screenshot shows the ENTIRE screen (1920x1080 pixels)
- Provide ABSOLUTE screen coordinates, NOT window-relative coordinates
- The top-left corner of the screen is (0, 0)
- The bottom-right corner is (1920, 1080){apps_info}{target_info}

APPLICATION LAUNCHING:
- To open/launch applications, ALWAYS use the "launch_app" action type
- DO NOT type launcher commands like "Alt+F1 → firefox" - use launch_app instead
- Example: To open Firefox, use {{"type": "launch_app", "app_name": "Firefox ESR"}}
- Available applications are listed in the WINDOW CONTEXT section below

SECURITY GUIDELINES FOR WEB NAVIGATION:
- ALWAYS use full URLs with https:// protocol when navigating to websites
- For example: use "https://reddit.com" not "reddit.com"
- Double-check typed URLs for typos before pressing Enter
- Verify the URL in address bar matches the intended destination
- NEVER type a domain name if it's already in the address bar (to avoid duplicates like reddit.comreddit.com)
- When navigating to a new site, first click on the address bar, then clear it (Ctrl+A), then type the full URL

WINDOW CONTEXT:
{window_context}

IMPORTANT WINDOW FOCUS RULES:
1. The ACTIVE WINDOW receives keyboard input
2. To interact with a different window, you MUST click on its Focus Point coordinates (shown above)
3. NEVER guess window positions - always use the exact Focus Point coordinates provided
4. Each window shows "Focus Point: (x, y)" - these are the exact coordinates to click for focusing that window

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
    {{"type": "launch_app", "app_name": "Firefox ESR", "description": "Launch Firefox browser"}},
    {{"type": "wait", "duration": 1.0, "description": "Wait 1 second"}}
  ],
  "reasoning": "I can see [describe what you see]. To [command], I will [explain approach]"
}}

{'IMPORTANT: When target_app is specified, always include "target_app": "' + target_app + '" in click, type, and key actions.' if target_app else ''}

LAUNCHING APPLICATION EXAMPLE:
To open Firefox:
{{"type": "launch_app", "app_name": "Firefox ESR", "description": "Launch Firefox web browser"}}

CLOSING WINDOWS EXAMPLE:
To close a window:
1. {{"type": "click", "x": 200, "y": 7, "description": "Right-click on window title bar", "button": "right"}}
2. {{"type": "wait", "duration": 0.5, "description": "Wait for context menu"}}
3. {{"type": "click", "x": 134, "y": 147, "description": "Click Close in context menu"}}

Available action types: click, type, key, scroll, wait, drag, launch_app
For click actions, add "button": "right" for right-clicks
For launch_app actions, use exact app names from AVAILABLE APPLICATIONS list

Be specific about what you see and provide EXACT ABSOLUTE screen coordinates!
{shortcuts_context}"""
    
    def _log_ai_input_debug(self, system_prompt: str, user_prompt: str, screenshot_base64: str):
        """Log AI input for debugging"""
        logger.debug(f"AI Input - System prompt length: {len(system_prompt)}, User prompt length: {len(user_prompt)}")
        logger.debug(f"Screenshot included: {'Yes' if screenshot_base64 else 'No'}")
        logger.debug(f"Using model: {self.service_manager.model}")
    
    def _log_ai_output_debug(self, ai_text: str):
        """Log AI output for debugging"""
        logger.debug(f"AI Response length: {len(ai_text)} characters")
        if logger.isEnabledFor(logging.DEBUG):
            logger.debug(f"AI Response preview: {ai_text[:200]}..." if len(ai_text) > 200 else ai_text)
    
    # Task Executor compatibility methods
    async def execute_task(self, task_id: str, task) -> Dict[str, Any]:
        """Execute a task (compatibility method for TaskExecutor replacement)
        
        Args:
            task_id: Task identifier
            task: Task request with task_type and parameters
            
        Returns:
            Task execution result
        """
        logger.info(f"Executing task {task_id}: {task.task_type}")
        
        try:
            # Route to appropriate handler
            if task.task_type == "screenshot":
                return await self._execute_task_screenshot(task.parameters)
            elif task.task_type == "click":
                return await self._execute_task_click(task.parameters)
            elif task.task_type == "double_click":
                return await self._execute_task_double_click(task.parameters)
            elif task.task_type == "right_click":
                return await self._execute_task_right_click(task.parameters)
            elif task.task_type == "type_text":
                return await self._execute_task_type_text(task.parameters)
            elif task.task_type == "key_press":
                return await self._execute_task_key_press(task.parameters)
            elif task.task_type == "mouse_move":
                return await self._execute_task_mouse_move(task.parameters)
            elif task.task_type == "drag_drop":
                return await self._execute_task_drag_drop(task.parameters)
            elif task.task_type == "scroll":
                return await self._execute_task_scroll(task.parameters)
            elif task.task_type == "automation_sequence":
                return await self._execute_task_automation_sequence(task.parameters)
            else:
                # Use AI to interpret unknown task types
                return await self.execute_action(
                    task=f"Execute {task.task_type} with parameters: {task.parameters}",
                    context={"task_type": task.task_type, "parameters": task.parameters}
                )
                    
        except Exception as e:
            logger.error(f"Task execution failed: {e}")
            raise
            
    async def execute_task_async(self, task_id: str, task, task_record: Dict[str, Any]):
        """Execute task asynchronously and update record"""
        try:
            result = await self.execute_task(task_id, task)
            task_record["status"] = "completed"
            task_record["result"] = result
            task_record["completed_at"] = datetime.utcnow().isoformat()
        except Exception as e:
            task_record["status"] = "failed"
            task_record["error"] = str(e)
            task_record["completed_at"] = datetime.utcnow().isoformat()
    
    async def _execute_task_screenshot(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute screenshot task"""
        logger.info("Taking screenshot")
        
        screenshot = self.screenshot_service.capture()
        
        return {
            "task_type": "screenshot",
            "status": "completed",
            "screenshot": screenshot,
            "timestamp": datetime.utcnow().isoformat()
        }
    
    async def _execute_task_click(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute click task"""
        x = params.get("x", 0)
        y = params.get("y", 0)
        button = params.get("button", "left")
        
        logger.info(f"Clicking at ({x}, {y}) with {button} button")
        self.automation_service.click(x, y, button)
        
        return {
            "task_type": "click",
            "status": "completed",
            "x": x,
            "y": y,
            "button": button,
            "timestamp": datetime.utcnow().isoformat()
        }
    
    async def _execute_task_double_click(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute double click task"""
        x = params.get("x", 0)
        y = params.get("y", 0)
        
        logger.info(f"Double clicking at ({x}, {y})")
        self.automation_service.double_click(x, y)
        
        return {
            "task_type": "double_click",
            "status": "completed",
            "x": x,
            "y": y,
            "timestamp": datetime.utcnow().isoformat()
        }
    
    async def _execute_task_right_click(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute right click task"""
        x = params.get("x", 0)
        y = params.get("y", 0)
        
        logger.info(f"Right clicking at ({x}, {y})")
        self.automation_service.click(x, y, "right")
        
        return {
            "task_type": "right_click",
            "status": "completed",
            "x": x,
            "y": y,
            "timestamp": datetime.utcnow().isoformat()
        }
    
    async def _execute_task_type_text(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute type text task"""
        text = params.get("text", "")
        
        logger.info(f"Typing text: {text[:50]}...")
        self.automation_service.type_text(text)
        
        return {
            "task_type": "type_text",
            "status": "completed",
            "text": text,
            "timestamp": datetime.utcnow().isoformat()
        }
    
    async def _execute_task_key_press(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute key press task"""
        key = params.get("key", "")
        modifiers = params.get("modifiers", [])
        
        logger.info(f"Pressing key: {key} with modifiers: {modifiers}")
        
        if modifiers:
            self.automation_service.key_combination(modifiers + [key])
        else:
            self.automation_service.key_press(key)
        
        return {
            "task_type": "key_press",
            "status": "completed",
            "key": key,
            "modifiers": modifiers,
            "timestamp": datetime.utcnow().isoformat()
        }
    
    async def _execute_task_mouse_move(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute mouse move task"""
        x = params.get("x", 0)
        y = params.get("y", 0)
        duration = params.get("duration", 0)
        
        logger.info(f"Moving mouse to ({x}, {y})")
        
        if duration > 0:
            self.automation_service.mouse_move_smooth(x, y, duration)
        else:
            self.automation_service.mouse_move(x, y)
        
        return {
            "task_type": "mouse_move",
            "status": "completed",
            "x": x,
            "y": y,
            "duration": duration,
            "timestamp": datetime.utcnow().isoformat()
        }
    
    async def _execute_task_drag_drop(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute drag and drop task"""
        start_x = params.get("start_x", 0)
        start_y = params.get("start_y", 0)
        end_x = params.get("end_x", 0)
        end_y = params.get("end_y", 0)
        duration = params.get("duration", 1.0)
        
        logger.info(f"Dragging from ({start_x}, {start_y}) to ({end_x}, {end_y})")
        self.automation_service.drag_drop(start_x, start_y, end_x, end_y, duration)
        
        return {
            "task_type": "drag_drop",
            "status": "completed",
            "start_x": start_x,
            "start_y": start_y,
            "end_x": end_x,
            "end_y": end_y,
            "duration": duration,
            "timestamp": datetime.utcnow().isoformat()
        }
    
    async def _execute_task_scroll(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute scroll task"""
        direction = params.get("direction", "down")
        amount = params.get("amount", 3)
        
        logger.info(f"Scrolling {direction} by {amount}")
        self.automation_service.scroll(direction, amount)
        
        return {
            "task_type": "scroll",
            "status": "completed",
            "direction": direction,
            "amount": amount,
            "timestamp": datetime.utcnow().isoformat()
        }
    
    async def _execute_task_automation_sequence(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute automation sequence task"""
        actions = params.get("actions", [])
        
        logger.info(f"Executing automation sequence with {len(actions)} actions")
        
        results = []
        for i, action in enumerate(actions):
            action_type = action.get("type")
            action_params = action.get("parameters", {})
            
            logger.info(f"Executing action {i+1}/{len(actions)}: {action_type}")
            
            # Route to appropriate action handler
            if action_type == "click":
                await self._execute_task_click(action_params)
            elif action_type == "type_text":
                await self._execute_task_type_text(action_params)
            elif action_type == "key_press":
                await self._execute_task_key_press(action_params)
            elif action_type == "mouse_move":
                await self._execute_task_mouse_move(action_params)
            elif action_type == "wait":
                wait_time = action_params.get("duration", 1)
                await asyncio.sleep(wait_time)
            
            results.append({
                "action": action_type,
                "status": "completed"
            })
        
        return {
            "task_type": "automation_sequence",
            "status": "completed",
            "actions_executed": len(actions),
            "results": results,
            "timestamp": datetime.utcnow().isoformat()
        }
    
    def _validate_screenshot(self, screenshot: Optional[str]) -> bool:
        """Validate that screenshot data is valid
        
        Args:
            screenshot: Screenshot data (data URL or base64)
            
        Returns:
            True if screenshot is valid, False otherwise
        """
        if not screenshot:
            logger.warning("Screenshot validation failed: no screenshot data")
            return False
        
        try:
            # Check if it's a data URL
            if screenshot.startswith('data:image'):
                # Extract base64 part
                if ',' not in screenshot:
                    logger.warning("Screenshot validation failed: malformed data URL")
                    return False
                
                mime_type, base64_data = screenshot.split(',', 1)
                
                # Validate MIME type
                if 'image' not in mime_type.lower():
                    logger.warning(f"Screenshot validation failed: invalid MIME type {mime_type}")
                    return False
                
                # Validate base64 data
                if len(base64_data) < 100:  # Minimum reasonable size
                    logger.warning("Screenshot validation failed: base64 data too small")
                    return False
                
                # Try to decode base64 to verify it's valid
                import base64
                try:
                    decoded = base64.b64decode(base64_data)
                    if len(decoded) < 1000:  # Minimum reasonable image size
                        logger.warning("Screenshot validation failed: decoded image too small")
                        return False
                except Exception as e:
                    logger.warning(f"Screenshot validation failed: invalid base64 data - {e}")
                    return False
                
                # Check for PNG/JPEG magic bytes
                if decoded.startswith(b'\x89PNG'):
                    logger.debug("Screenshot validation: valid PNG detected")
                    return True
                elif decoded.startswith(b'\xff\xd8\xff'):
                    logger.debug("Screenshot validation: valid JPEG detected")
                    return True
                else:
                    logger.warning("Screenshot validation failed: no valid image magic bytes")
                    return False
            
            else:
                # Assume it's raw base64
                import base64
                try:
                    decoded = base64.b64decode(screenshot)
                    if len(decoded) < 1000:
                        logger.warning("Screenshot validation failed: raw base64 image too small")
                        return False
                    return True
                except Exception as e:
                    logger.warning(f"Screenshot validation failed: invalid raw base64 - {e}")
                    return False
                    
        except Exception as e:
            logger.error(f"Screenshot validation error: {e}")
            return False