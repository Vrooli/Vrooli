"""Task executor service for Agent S2"""

import logging
from typing import Dict, Any
from datetime import datetime

from ..models.requests import TaskRequest
from .capture import ScreenshotService
from .automation import AutomationService

logger = logging.getLogger(__name__)


class TaskExecutor:
    """Service for executing automation tasks"""
    
    def __init__(self, app_state: Dict[str, Any]):
        """Initialize task executor
        
        Args:
            app_state: Application state containing AI handler
        """
        self.app_state = app_state
        self.screenshot_service = ScreenshotService()
        self.automation_service = AutomationService()
        
    async def execute_task(self, task_id: str, task: TaskRequest) -> Dict[str, Any]:
        """Execute a task
        
        Args:
            task_id: Task identifier
            task: Task request
            
        Returns:
            Task execution result
        """
        logger.info(f"Executing task {task_id}: {task.task_type}")
        
        try:
            # Route to appropriate handler
            if task.task_type == "screenshot":
                return await self._execute_screenshot(task.parameters)
            elif task.task_type == "click":
                return await self._execute_click(task.parameters)
            elif task.task_type == "double_click":
                return await self._execute_double_click(task.parameters)
            elif task.task_type == "right_click":
                return await self._execute_right_click(task.parameters)
            elif task.task_type == "type_text":
                return await self._execute_type_text(task.parameters)
            elif task.task_type == "key_press":
                return await self._execute_key_press(task.parameters)
            elif task.task_type == "mouse_move":
                return await self._execute_mouse_move(task.parameters)
            elif task.task_type == "drag_drop":
                return await self._execute_drag_drop(task.parameters)
            elif task.task_type == "scroll":
                return await self._execute_scroll(task.parameters)
            elif task.task_type == "automation_sequence":
                return await self._execute_automation_sequence(task.parameters)
            else:
                # Try AI fallback if available
                ai_handler = self.app_state.get("ai_handler")
                if ai_handler and ai_handler.initialized:
                    return await self._execute_ai_fallback(task.task_type, task.parameters)
                else:
                    raise ValueError(f"Unknown task type: {task.task_type}")
                    
        except Exception as e:
            logger.error(f"Task execution failed: {e}")
            raise
            
    async def execute_task_async(self, task_id: str, task: TaskRequest, task_record: Dict[str, Any]):
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
            
    async def _execute_screenshot(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute screenshot task"""
        result = self.screenshot_service.capture(
            format=params.get("format", "png"),
            quality=params.get("quality", 95),
            region=params.get("region")
        )
        
        return {
            "task_type": "screenshot",
            "format": result["format"],
            "size": result["size"],
            "data": result["data"]
        }
        
    async def _execute_click(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute click task"""
        x = params.get("x")
        y = params.get("y")
        button = params.get("button", "left")
        clicks = params.get("clicks", 1)
        
        self.automation_service.click(x, y, button, clicks)
        
        return {
            "task_type": "click",
            "action": "click",
            "position": {"x": x, "y": y} if x and y else "current",
            "button": button,
            "clicks": clicks
        }
        
    async def _execute_double_click(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute double click task"""
        x = params.get("x")
        y = params.get("y")
        
        self.automation_service.double_click(x, y)
        
        return {
            "task_type": "double_click",
            "action": "double_click",
            "position": {"x": x, "y": y} if x and y else "current"
        }
        
    async def _execute_right_click(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute right click task"""
        x = params.get("x")
        y = params.get("y")
        
        self.automation_service.right_click(x, y)
        
        return {
            "task_type": "right_click",
            "action": "right_click",
            "position": {"x": x, "y": y} if x and y else "current"
        }
        
    async def _execute_type_text(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute type text task"""
        text = params.get("text", "")
        interval = params.get("interval", 0.0)
        
        self.automation_service.type_text(text, interval)
        
        return {
            "task_type": "type_text",
            "text": text,
            "characters": len(text),
            "interval": interval
        }
        
    async def _execute_key_press(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute key press task"""
        key = params.get("key", "")
        modifiers = params.get("modifiers", [])
        
        if modifiers:
            keys = modifiers + [key]
            self.automation_service.hotkey(*keys)
            key_desc = "+".join(keys)
        else:
            self.automation_service.press_key(key)
            key_desc = key
            
        return {
            "task_type": "key_press",
            "key": key_desc,
            "modifiers": modifiers
        }
        
    async def _execute_mouse_move(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute mouse move task"""
        x = params.get("x", 0)
        y = params.get("y", 0)
        duration = params.get("duration", 0.0)
        relative = params.get("relative", False)
        
        if relative:
            self.automation_service.move_mouse_relative(x, y, duration)
        else:
            self.automation_service.move_mouse(x, y, duration)
            
        return {
            "task_type": "mouse_move",
            "position": {"x": x, "y": y},
            "duration": duration,
            "relative": relative
        }
        
    async def _execute_drag_drop(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute drag and drop task"""
        start = params.get("start", [0, 0])
        end = params.get("end", [0, 0])
        duration = params.get("duration", 1.0)
        button = params.get("button", "left")
        
        self.automation_service.drag(
            start_x=start[0],
            start_y=start[1],
            end_x=end[0],
            end_y=end[1],
            duration=duration,
            button=button
        )
        
        return {
            "task_type": "drag_drop",
            "start": {"x": start[0], "y": start[1]},
            "end": {"x": end[0], "y": end[1]},
            "duration": duration,
            "button": button
        }
        
    async def _execute_scroll(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute scroll task"""
        clicks = params.get("clicks", 0)
        x = params.get("x")
        y = params.get("y")
        
        self.automation_service.scroll(clicks, x, y)
        
        return {
            "task_type": "scroll",
            "clicks": clicks,
            "direction": "up" if clicks > 0 else "down",
            "position": {"x": x, "y": y} if x and y else "current"
        }
        
    async def _execute_automation_sequence(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a sequence of automation tasks"""
        steps = params.get("steps", [])
        results = []
        
        for i, step in enumerate(steps):
            step_type = step.get("type")
            step_params = step.get("parameters", {})
            
            try:
                # Create a mini task for each step
                mini_task = TaskRequest(
                    task_type=step_type,
                    parameters=step_params,
                    async_execution=False
                )
                
                # Execute the step
                step_result = await self.execute_task(f"seq_{i}", mini_task)
                results.append({
                    "step": i + 1,
                    "type": step_type,
                    "status": "completed",
                    "result": step_result
                })
                
                # Optional delay between steps
                delay = step.get("delay", 0)
                if delay > 0:
                    import asyncio
                    await asyncio.sleep(delay)
                    
            except Exception as e:
                results.append({
                    "step": i + 1,
                    "type": step_type,
                    "status": "failed",
                    "error": str(e)
                })
                # Stop on error unless continue_on_error is set
                if not params.get("continue_on_error", False):
                    break
                    
        return {
            "task_type": "automation_sequence",
            "total_steps": len(steps),
            "completed_steps": len([r for r in results if r["status"] == "completed"]),
            "results": results
        }
        
    async def _execute_ai_fallback(self, task_type: str, params: Dict[str, Any]) -> Dict[str, Any]:
        """Attempt to execute unknown task using AI"""
        ai_handler = self.app_state.get("ai_handler")
        
        return await ai_handler.execute_action(
            task=f"Execute {task_type} with parameters: {params}",
            context={"task_type": task_type, "parameters": params}
        )