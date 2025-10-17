"""Task management routes for Agent S2 API"""

import logging
from datetime import datetime
from typing import List, Optional
from fastapi import APIRouter, Request, HTTPException, BackgroundTasks, Query

from ..models.requests import TaskRequest
from ..models.responses import TaskResponse

logger = logging.getLogger(__name__)
router = APIRouter()


@router.post("/execute", response_model=TaskResponse)
async def execute_task(
    task: TaskRequest,
    request: Request,
    background_tasks: BackgroundTasks
):
    """Execute an automation task
    
    Args:
        task: Task execution parameters
        
    Returns:
        Task execution response
    """
    app_state = request.app.state.app_state
    
    # Generate task ID
    app_state["task_counter"] += 1
    task_id = f"task_{app_state['task_counter']}"
    
    # Create task record
    task_record = {
        "id": task_id,
        "type": task.task_type,
        "parameters": task.parameters,
        "status": "pending",
        "created_at": datetime.utcnow().isoformat(),
        "result": None,
        "error": None
    }
    app_state["tasks"][task_id] = task_record
    
    # Get AI executor from AI handler
    ai_handler = app_state.get("ai_handler")
    if not ai_handler or not ai_handler.initialized:
        raise HTTPException(status_code=503, detail="AI handler not initialized")
    
    executor = ai_handler.executor
    
    if task.async_execution:
        # Execute in background
        background_tasks.add_task(
            executor.execute_task_async,
            task_id,
            task,
            task_record
        )
        task_record["status"] = "running"
    else:
        # Execute synchronously
        try:
            result = await executor.execute_task(task_id, task)
            task_record["status"] = "completed"
            task_record["result"] = result
            task_record["completed_at"] = datetime.utcnow().isoformat()
        except Exception as e:
            task_record["status"] = "failed"
            task_record["error"] = str(e)
            task_record["completed_at"] = datetime.utcnow().isoformat()
    
    return TaskResponse(
        task_id=task_id,
        status=task_record["status"],
        result=task_record["result"],
        error=task_record["error"],
        created_at=task_record["created_at"],
        completed_at=task_record.get("completed_at")
    )


@router.get("/{task_id}", response_model=TaskResponse)
async def get_task_status(task_id: str, request: Request):
    """Get status of a specific task
    
    Args:
        task_id: Task identifier
        
    Returns:
        Task status and results
    """
    app_state = request.app.state.app_state
    
    if task_id not in app_state["tasks"]:
        raise HTTPException(status_code=404, detail="Task not found")
    
    task = app_state["tasks"][task_id]
    return TaskResponse(
        task_id=task_id,
        status=task["status"],
        result=task["result"],
        error=task["error"],
        created_at=task["created_at"],
        completed_at=task.get("completed_at")
    )


@router.get("", response_model=List[TaskResponse])
async def list_tasks(
    request: Request,
    status: Optional[str] = Query(default=None, description="Filter by status"),
    limit: int = Query(default=100, ge=1, le=1000, description="Maximum number of tasks"),
    offset: int = Query(default=0, ge=0, description="Offset for pagination")
):
    """List tasks with optional filtering
    
    Args:
        status: Optional status filter
        limit: Maximum number of tasks to return
        offset: Offset for pagination
        
    Returns:
        List of tasks
    """
    app_state = request.app.state.app_state
    tasks = app_state["tasks"]
    
    # Filter by status if provided
    if status:
        filtered_tasks = [
            task for task in tasks.values()
            if task["status"] == status
        ]
    else:
        filtered_tasks = list(tasks.values())
    
    # Sort by created_at (newest first)
    filtered_tasks.sort(key=lambda t: t["created_at"], reverse=True)
    
    # Apply pagination
    paginated_tasks = filtered_tasks[offset:offset + limit]
    
    # Convert to response models
    return [
        TaskResponse(
            task_id=task["id"],
            status=task["status"],
            result=task["result"],
            error=task["error"],
            created_at=task["created_at"],
            completed_at=task.get("completed_at")
        )
        for task in paginated_tasks
    ]


@router.delete("/{task_id}")
async def cancel_task(task_id: str, request: Request):
    """Cancel a running task
    
    Args:
        task_id: Task identifier
        
    Returns:
        Cancellation result
    """
    app_state = request.app.state.app_state
    
    if task_id not in app_state["tasks"]:
        raise HTTPException(status_code=404, detail="Task not found")
    
    task = app_state["tasks"][task_id]
    
    if task["status"] not in ["pending", "running"]:
        raise HTTPException(
            status_code=400,
            detail=f"Cannot cancel task in {task['status']} status"
        )
    
    # Update task status
    task["status"] = "cancelled"
    task["completed_at"] = datetime.utcnow().isoformat()
    task["error"] = "Task cancelled by user"
    
    return {
        "task_id": task_id,
        "status": "cancelled",
        "message": "Task cancelled successfully"
    }


@router.delete("/")
async def clear_tasks(
    request: Request,
    status: Optional[str] = Query(default=None, description="Clear only tasks with this status")
):
    """Clear completed or failed tasks
    
    Args:
        status: Optional status filter
        
    Returns:
        Number of tasks cleared
    """
    app_state = request.app.state.app_state
    tasks = app_state["tasks"]
    
    if status:
        # Clear only tasks with specific status
        tasks_to_clear = [
            task_id for task_id, task in tasks.items()
            if task["status"] == status
        ]
    else:
        # Clear all completed and failed tasks
        tasks_to_clear = [
            task_id for task_id, task in tasks.items()
            if task["status"] in ["completed", "failed", "cancelled"]
        ]
    
    # Remove tasks
    for task_id in tasks_to_clear:
        del tasks[task_id]
    
    return {
        "cleared": len(tasks_to_clear),
        "message": f"Cleared {len(tasks_to_clear)} tasks"
    }