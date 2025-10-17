#!/usr/bin/env python3
"""Agent S2 API Server - Main entry point"""

import uvicorn
from .app import app
from ..config import Config

if __name__ == "__main__":
    uvicorn.run(
        app,
        host=Config.API_HOST,
        port=Config.API_PORT,
        log_level=Config.LOG_LEVEL.lower()
    )