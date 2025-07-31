"""Refactored AI Handler for Agent S2

Clean, modular AI handler that orchestrates specialized components.
"""

import logging
from typing import Dict, Any, Optional, List

from ...config import Config
from ...stealth import StealthManager, StealthConfig
from .ai_service_manager import AIServiceManager
from .ai_planner import AIPlanner
from .ai_security_validator import AISecurityValidator
from .ai_executor import AIExecutor

logger = logging.getLogger(__name__)


class AIHandler:
    """Refactored AI handler that orchestrates specialized AI components"""
    
    def __init__(self):
        """Initialize AI handler with modular components"""
        self.initialized = False
        
        # Initialize core components
        self.service_manager = AIServiceManager()
        self.security_validator = AISecurityValidator()
        self.planner = AIPlanner(self.service_manager)
        self.executor = AIExecutor(self.service_manager, self.planner, self.security_validator)
        
        # Initialize stealth manager
        stealth_config = StealthConfig(
            enabled=Config.STEALTH_MODE_ENABLED,
            session_storage_path=Config.SESSION_STORAGE_PATH
        )
        self.stealth_manager = StealthManager(stealth_config)
    
    async def initialize(self):
        """Initialize AI handler with error handling and auto-detection"""
        logger.info("Initializing modular AI handler...")
        
        try:
            # Initialize AI service manager
            service_init_success = await self.service_manager.initialize()
            if not service_init_success:
                logger.error("AI service manager initialization failed")
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
            logger.info("✅ Modular AI handler initialized successfully")
            logger.info(f"   Service Manager: {'Ready' if self.service_manager.is_ready() else 'Not Ready'}")
            logger.info(f"   Planner: Ready")
            logger.info(f"   Security Validator: Ready") 
            logger.info(f"   Executor: Ready")
            
        except Exception as e:
            logger.error(f"Failed to initialize AI handler: {e}")
            self.initialized = False
            
    async def shutdown(self):
        """Shutdown AI handler"""
        logger.info("Shutting down modular AI handler...")
        
        # Shutdown service manager
        await self.service_manager.shutdown()
        
        self.initialized = False
        logger.info("Modular AI handler shut down")
    
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
            
        return await self.executor.execute_action(task, screenshot, context)
    
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
            
        return await self.executor.execute_command(command, context, target_app)
    
    async def execute_command_async(self,
                                  task_id: str,
                                  command: str,
                                  context: Optional[str],
                                  task_record: Dict[str, Any]):
        """Execute command asynchronously"""
        if not self.initialized:
            raise RuntimeError("AI handler not initialized")
            
        return await self.executor.execute_command_async(task_id, command, context, task_record)
    
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
            
        return await self.planner.generate_plan(goal, constraints)
    
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
            
        return await self.planner.analyze_screen(question, screenshot)
    
    # Properties for backward compatibility
    @property
    def enabled(self) -> bool:
        """Check if AI is enabled"""
        return self.service_manager.enabled
    
    @property
    def provider(self) -> str:
        """Get AI provider"""
        return self.service_manager.provider
    
    @property
    def model(self) -> str:
        """Get AI model"""
        return self.service_manager.model
    
    @property
    def api_url(self) -> str:
        """Get API URL"""
        return self.service_manager.api_url
    
    @property
    def ollama_base_url(self) -> str:
        """Get Ollama base URL"""
        return self.service_manager.ollama_base_url
    
    def get_service_info(self) -> Dict[str, Any]:
        """Get comprehensive service information"""
        return {
            "initialized": self.initialized,
            "service_manager": self.service_manager.get_service_info(),
            "components": {
                "planner": "Ready",
                "security_validator": "Ready",
                "executor": "Ready"
            },
            "capabilities": self.get_capabilities()
        }