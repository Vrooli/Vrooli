"""Agent S2 AI Client

Specialized client for AI-driven automation.
"""

from typing import Optional, List, Dict, Any, Union
import json
import logging

from .base import AgentS2Client
from .screenshot import ScreenshotClient
from .automation import AutomationClient

logger = logging.getLogger(__name__)


class AIClient:
    """Client specialized for AI-driven operations"""
    
    def __init__(self, client: Optional[AgentS2Client] = None):
        """Initialize AI client
        
        Args:
            client: Base Agent S2 client (creates new if None)
        """
        self.client = client or AgentS2Client()
        self.screenshot = ScreenshotClient(self.client)
        self.automation = AutomationClient(self.client)
        
    def perform_task(self, task: str, context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Perform an AI-driven task
        
        Args:
            task: Natural language task description
            context: Optional context information
            
        Returns:
            Task execution result with AI reasoning
        """
        # Capture current screen state
        screenshot = self.screenshot.capture()
        
        # Execute AI action
        result = self.client.ai_action(
            task=task,
            screenshot=screenshot['data'],
            context=context
        )
        
        return result
    
    def navigate_to(self, target: str) -> Dict[str, Any]:
        """Navigate to a specific UI element or location
        
        Args:
            target: Description of where to navigate
            
        Returns:
            Navigation result
        """
        return self.perform_task(f"Navigate to {target}")
    
    def fill_form(self, form_data: Dict[str, str]) -> Dict[str, Any]:
        """Fill a form with provided data
        
        Args:
            form_data: Dictionary of field names/labels to values
            
        Returns:
            Form filling result
        """
        task = "Fill the form with the following data:\n"
        for field, value in form_data.items():
            task += f"- {field}: {value}\n"
            
        return self.perform_task(task)
    
    def extract_text(self, description: Optional[str] = None) -> str:
        """Extract text from the current screen
        
        Args:
            description: Optional description of specific text to extract
            
        Returns:
            Extracted text
        """
        task = "Extract text from the screen"
        if description:
            task += f", specifically: {description}"
            
        result = self.perform_task(task)
        return result.get('extracted_text', '')
    
    def find_and_interact(self, element: str, action: str = "click") -> Dict[str, Any]:
        """Find an element and interact with it
        
        Args:
            element: Description of the element to find
            action: Action to perform (click, double-click, right-click, etc.)
            
        Returns:
            Interaction result
        """
        return self.perform_task(f"Find '{element}' and {action} on it")
    
    def wait_for_element(self, element: str, timeout: int = 30) -> bool:
        """Wait for an element to appear
        
        Args:
            element: Description of element to wait for
            timeout: Maximum wait time in seconds
            
        Returns:
            True if element appeared, False if timeout
        """
        import time
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            result = self.perform_task(f"Check if '{element}' is visible on screen")
            if result.get('found', False):
                return True
            time.sleep(2)
            
        return False
    
    def verify_state(self, expected_state: str) -> bool:
        """Verify the current UI state matches expectation
        
        Args:
            expected_state: Description of expected state
            
        Returns:
            True if state matches, False otherwise
        """
        result = self.perform_task(f"Verify that {expected_state}")
        return result.get('verified', False)
    
    def perform_sequence(self, steps: List[str]) -> List[Dict[str, Any]]:
        """Perform a sequence of tasks
        
        Args:
            steps: List of task descriptions
            
        Returns:
            List of results for each step
        """
        results = []
        context = {"previous_steps": []}
        
        for i, step in enumerate(steps):
            logger.info(f"Executing step {i+1}/{len(steps)}: {step}")
            result = self.perform_task(step, context)
            results.append(result)
            
            # Update context for next step
            context["previous_steps"].append({
                "step": step,
                "result": result.get('summary', 'completed')
            })
            
        return results
    
    def automate_workflow(self, workflow_name: str, 
                         parameters: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Execute a predefined workflow
        
        Args:
            workflow_name: Name of the workflow
            parameters: Workflow parameters
            
        Returns:
            Workflow execution result
        """
        workflows = {
            "login": [
                "Find the username field and click on it",
                f"Type '{parameters.get('username', '')}'"
                "Find the password field and click on it",
                f"Type '{parameters.get('password', '')}'"
                "Find and click the login button"
            ],
            "search": [
                "Find the search box and click on it",
                f"Type '{parameters.get('query', '')}'"
                "Press Enter or click the search button"
            ],
            "download_file": [
                f"Find and click on '{parameters.get('filename', 'the file')}'"
                "Click the download button",
                "Wait for download to complete"
            ]
        }
        
        if workflow_name not in workflows:
            raise ValueError(f"Unknown workflow: {workflow_name}")
            
        steps = workflows[workflow_name]
        return {
            "workflow": workflow_name,
            "parameters": parameters,
            "results": self.perform_sequence(steps)
        }
    
    def smart_click(self, description: str) -> bool:
        """Intelligently find and click an element
        
        Args:
            description: Natural language description of what to click
            
        Returns:
            True if successful, False otherwise
        """
        result = self.find_and_interact(description, "click")
        return result.get('success', False)
    
    def smart_type(self, text: str, target: Optional[str] = None) -> bool:
        """Intelligently type text, optionally in a specific field
        
        Args:
            text: Text to type
            target: Optional description of where to type
            
        Returns:
            True if successful, False otherwise
        """
        if target:
            task = f"Click on '{target}' and type '{text}'"
        else:
            task = f"Type '{text}' in the current field"
            
        result = self.perform_task(task)
        return result.get('success', False)
    
    def analyze_screen(self) -> Dict[str, Any]:
        """Analyze the current screen and describe what's visible
        
        Returns:
            Screen analysis with identified elements
        """
        return self.perform_task("Analyze the current screen and describe what you see, including all clickable elements, text fields, and important information")
    
    def suggest_actions(self) -> List[str]:
        """Suggest possible actions based on current screen
        
        Returns:
            List of suggested actions
        """
        result = self.perform_task("Based on the current screen, suggest 5 possible actions a user might want to take")
        return result.get('suggestions', [])