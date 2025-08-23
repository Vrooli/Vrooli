"""Error handling middleware for Agent S2 API"""

import logging
import traceback
from typing import Callable

from fastapi import Request, Response
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware

logger = logging.getLogger(__name__)


class ErrorHandlerMiddleware(BaseHTTPMiddleware):
    """Middleware for handling errors and exceptions"""
    
    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        """Process requests and handle errors"""
        try:
            response = await call_next(request)
            return response
            
        except ValueError as e:
            # Handle validation errors
            logger.warning(f"Validation error: {e}")
            return JSONResponse(
                status_code=400,
                content={
                    "detail": str(e),
                    "type": "validation_error"
                }
            )
            
        except PermissionError as e:
            # Handle permission errors
            logger.error(f"Permission error: {e}")
            return JSONResponse(
                status_code=403,
                content={
                    "detail": "Permission denied",
                    "type": "permission_error"
                }
            )
            
        except FileNotFoundError as e:
            # Handle file not found errors
            logger.error(f"File not found: {e}")
            return JSONResponse(
                status_code=404,
                content={
                    "detail": "Resource not found",
                    "type": "not_found_error"
                }
            )
            
        except TimeoutError as e:
            # Handle timeout errors
            logger.error(f"Operation timed out: {e}")
            return JSONResponse(
                status_code=504,
                content={
                    "detail": "Operation timed out",
                    "type": "timeout_error"
                }
            )
            
        except Exception as e:
            # Handle all other exceptions
            logger.error(f"Unhandled exception: {e}")
            logger.error(traceback.format_exc())
            
            # Don't expose internal errors in production
            return JSONResponse(
                status_code=500,
                content={
                    "detail": "Internal server error",
                    "type": "internal_error"
                }
            )