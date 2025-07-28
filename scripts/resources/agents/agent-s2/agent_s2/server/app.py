"""Agent S2 API Server Application

Main FastAPI application setup and configuration.
"""

import os
import logging
from datetime import datetime
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from ..config import Config
from .routes import health, screenshot, mouse, keyboard, ai, tasks, modes
from .middleware import ErrorHandlerMiddleware
from .services.ai_handler import AIHandler

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
    "ai_handler": None
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
        except Exception as e:
            logger.warning(f"AI initialization failed: {e}")
            logger.info("Running in core automation mode only")
    else:
        logger.info("AI disabled - running in core automation mode")
    
    # Startup
    yield
    
    # Shutdown
    logger.info("Shutting down Agent S2 API Server...")
    if app_state["ai_handler"]:
        await app_state["ai_handler"].shutdown()

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