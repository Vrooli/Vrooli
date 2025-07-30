"""Agent S2 API Server Application

Main FastAPI application setup and configuration.
"""

import os
import logging
import asyncio
from datetime import datetime
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from ..config import Config
from .routes import health, screenshot, mouse, keyboard, ai, tasks, modes, stealth
from .middleware import ErrorHandlerMiddleware
from .services.ai_handler import AIHandler
from .services.proxy_service import ProxyManager

# Configure logging
logging.basicConfig(
    level=getattr(logging, Config.LOG_LEVEL.upper()),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Global application state
app_state = {
    "tasks": {},
    "task_counter": 0,
    "startup_time": datetime.utcnow().isoformat(),
    "ai_handler": None,
    "proxy_service": None
}

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage application lifecycle"""
    logger.info("Starting Agent S2 API Server...")
    
    # Initialize AI handler if enabled
    if Config.AI_ENABLED:
        try:
            app_state["ai_handler"] = AIHandler()
            await app_state["ai_handler"].initialize()
            logger.info("AI handler initialized successfully")
            app_state["ai_init_error"] = None  # Clear any previous errors
        except Exception as e:
            import traceback
            error_type = type(e).__name__
            
            # Determine error category and suggestions
            error_details = {
                "error_type": error_type,
                "details": str(e),
                "traceback": traceback.format_exc(),
                "timestamp": datetime.utcnow().isoformat()
            }
            
            # Provide specific suggestions based on error type
            if "ConnectionError" in error_type or "RequestException" in str(e):
                error_details["category"] = "connection_failed"
                error_details["suggestions"] = [
                    "Ensure Ollama is running: docker run -d -p 11434:11434 --name ollama ollama/ollama",
                    f"Check if Ollama is accessible at {Config.AI_API_URL}",
                    "Verify network connectivity: curl http://localhost:11434/api/tags",
                    "If using Docker, ensure the container can reach the host network"
                ]
            elif "model" in str(e).lower():
                error_details["category"] = "model_not_found"
                error_details["suggestions"] = [
                    f"Pull the required model: ollama pull {Config.AI_MODEL}",
                    "List available models: ollama list",
                    "Use a vision model for best results: ollama pull llama3.2-vision:11b"
                ]
            else:
                error_details["category"] = "initialization_failed"
                error_details["suggestions"] = [
                    "Check the logs for detailed error information",
                    "Verify environment variables are correctly set",
                    "Try running with AGENTS2_ENABLE_AI=false for core automation only"
                ]
            
            app_state["ai_init_error"] = error_details
            logger.warning(f"AI initialization failed: {e}")
            logger.info("Running in core automation mode only")
    else:
        logger.info("AI disabled - running in core automation mode")
        app_state["ai_init_error"] = {
            "category": "disabled",
            "details": "AI features are disabled by configuration",
            "suggestions": ["Set AGENTS2_ENABLE_AI=true to enable AI features"]
        }
    
    # Initialize security proxy if enabled (fully asynchronous)
    if os.environ.get("AGENT_S2_ENABLE_PROXY", "true").lower() == "true":
        logger.info("Initializing security proxy service...")
        try:
            proxy_manager = ProxyManager()
            proxy_service = proxy_manager.get_proxy_service()
            app_state["proxy_service"] = proxy_service
            
            # Start proxy initialization in background (completely non-blocking)
            async def initialize_proxy():
                try:
                    logger.info("Starting security proxy service in background...")
                    
                    # Check if we have root for iptables (in Docker we might)
                    if os.geteuid() == 0:
                        try:
                            proxy_service.setup_transparent_proxy()
                            logger.info("Transparent proxy configured with iptables")
                        except Exception as e:
                            logger.warning(f"Failed to setup transparent proxy: {e}")
                    else:
                        logger.warning("Running without root - transparent proxy disabled")
                        logger.info("Security checks will only apply to AI-driven navigation")
                    
                    # Install CA certificate for HTTPS interception
                    try:
                        if proxy_service.install_ca_certificate():
                            logger.info("mitmproxy CA certificate installed")
                        else:
                            logger.info("Continuing without CA certificate installation")
                    except Exception as e:
                        logger.warning(f"CA certificate installation failed: {e}")
                    
                    # Start the proxy service
                    try:
                        await proxy_manager.ensure_proxy_running()
                        logger.info("Security proxy service started successfully")
                    except Exception as e:
                        logger.warning(f"Failed to start security proxy service: {e}")
                        logger.info("AI-level security validation remains active")
                        
                except Exception as e:
                    logger.error(f"Proxy background initialization failed: {e}")
                    logger.info("Continuing with AI-level security only")
            
            # Start proxy initialization task without awaiting
            asyncio.create_task(initialize_proxy())
            logger.info("Security proxy initialization started in background")
            
        except Exception as e:
            logger.error(f"Security proxy setup failed: {e}")
            logger.info("Continuing without network-level security")
    else:
        logger.info("Security proxy disabled by configuration")
    
    # Startup
    yield
    
    # Shutdown
    logger.info("Shutting down Agent S2 API Server...")
    
    # Shutdown AI handler
    if app_state["ai_handler"]:
        await app_state["ai_handler"].shutdown()
    
    # Shutdown proxy service
    if app_state.get("proxy_service"):
        try:
            await app_state["proxy_service"].stop()
            if os.geteuid() == 0:
                app_state["proxy_service"].cleanup_iptables()
            logger.info("Security proxy service stopped")
        except Exception as e:
            logger.error(f"Error stopping proxy service: {e}")

# Create FastAPI app
app = FastAPI(
    title="Agent S2 API",
    description="Autonomous computer interaction service with AI capabilities",
    version="1.0.0",
    lifespan=lifespan
)

# Add middleware
app.add_middleware(ErrorHandlerMiddleware)

if Config.ENABLE_CORS:
    app.add_middleware(
        CORSMiddleware,
        allow_origins=Config.CORS_ORIGINS,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

# Include routers
app.include_router(health.router, tags=["health"])
app.include_router(modes.router, prefix="/modes", tags=["modes"])
app.include_router(screenshot.router, prefix="/screenshot", tags=["screenshot"])
app.include_router(mouse.router, prefix="/mouse", tags=["mouse"])
app.include_router(keyboard.router, prefix="/keyboard", tags=["keyboard"])
app.include_router(tasks.router, prefix="/tasks", tags=["tasks"])
app.include_router(ai.router, prefix="/ai", tags=["ai"])
app.include_router(stealth.router, prefix="/stealth", tags=["stealth"])

# Store app state in app.state for access in routes
app.state.app_state = app_state

# Error handlers
@app.exception_handler(Exception)
async def general_exception_handler(request, exc):
    logger.error(f"Unhandled exception: {exc}")
    return JSONResponse(
        status_code=500,
        content={"detail": "Internal server error", "error": str(exc)}
    )

def get_app() -> FastAPI:
    """Get the FastAPI application instance"""
    return app

if __name__ == "__main__":
    import uvicorn
    
    uvicorn.run(
        app,
        host=Config.API_HOST,
        port=Config.API_PORT,
        log_level=Config.LOG_LEVEL.lower()
    )