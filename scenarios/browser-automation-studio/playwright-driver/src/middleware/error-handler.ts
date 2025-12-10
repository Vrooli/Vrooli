import type { ServerResponse } from 'http';
import {
  PlaywrightDriverError,
  SessionNotFoundError,
  InvalidInstructionError,
  UnsupportedInstructionError,
  SelectorNotFoundError,
  TimeoutError,
  NavigationError,
  FrameNotFoundError,
  ResourceLimitError,
  ConfigurationError,
  logger,
} from '../utils';

/**
 * Send JSON response
 */
export function sendJson(
  res: ServerResponse,
  statusCode: number,
  data: unknown
): void {
  res.statusCode = statusCode;
  res.setHeader('Content-Type', 'application/json');
  res.end(JSON.stringify(data));
}

/**
 * Get actionable hint for common error scenarios
 */
function getErrorHint(error: Error | PlaywrightDriverError): string | undefined {
  if (error instanceof SessionNotFoundError) {
    return 'Session may have timed out or been closed. Create a new session with POST /session/start';
  }
  if (error instanceof SelectorNotFoundError) {
    return 'Element not found. Verify the selector is correct and the element exists on the page. Consider adding a wait instruction before this step.';
  }
  if (error instanceof TimeoutError) {
    return 'Operation timed out. The page may be slow to load or the element may not exist. Try increasing the timeout or verifying the page state.';
  }
  if (error instanceof NavigationError) {
    return 'Navigation failed. Verify the URL is correct and accessible. Check for network issues or authentication requirements.';
  }
  if (error instanceof FrameNotFoundError) {
    return 'Frame not found. Verify the frame selector or URL pattern. The frame may have been removed or not yet loaded.';
  }
  if (error instanceof ResourceLimitError) {
    return 'Resource limit reached. Close unused sessions or wait for idle sessions to be cleaned up.';
  }
  if (error instanceof InvalidInstructionError) {
    return 'Check instruction parameters match the expected schema for this instruction type.';
  }
  if (error instanceof UnsupportedInstructionError) {
    return 'This instruction type is not supported. Check available instruction types in the documentation.';
  }
  return undefined;
}

/**
 * Send error response
 */
export function sendError(
  res: ServerResponse,
  error: Error | PlaywrightDriverError,
  requestPath?: string
): void {
  let statusCode = 500;
  let errorCode = 'INTERNAL_ERROR';
  let message = error.message;
  let kind = 'engine';
  let retryable = false;

  // Map driver errors to HTTP status codes
  if (error instanceof SessionNotFoundError) {
    statusCode = 404;
    errorCode = error.code;
    kind = error.kind;
    retryable = error.retryable;
  } else if (error instanceof InvalidInstructionError || error instanceof UnsupportedInstructionError) {
    statusCode = 400;
    errorCode = error.code;
    kind = error.kind;
    retryable = error.retryable;
  } else if (error instanceof ResourceLimitError) {
    statusCode = 429;
    errorCode = error.code;
    kind = error.kind;
    retryable = error.retryable;
  } else if (error instanceof ConfigurationError) {
    statusCode = 500;
    errorCode = error.code;
    kind = error.kind;
    retryable = error.retryable;
  } else if (error instanceof SelectorNotFoundError || error instanceof TimeoutError || error instanceof NavigationError || error instanceof FrameNotFoundError) {
    statusCode = 500;
    errorCode = error.code;
    kind = error.kind;
    retryable = error.retryable;
  } else if (error instanceof PlaywrightDriverError) {
    statusCode = 500;
    errorCode = error.code;
    kind = error.kind;
    retryable = error.retryable;
  }

  const hint = getErrorHint(error);

  logger.error('request: failed', {
    path: requestPath,
    statusCode,
    errorCode,
    message,
    kind,
    retryable,
  });

  sendJson(res, statusCode, {
    error: {
      code: errorCode,
      message,
      kind,
      retryable,
      ...(hint && { hint }),
    },
  });
}

/**
 * Send 404 not found response
 */
export function send404(res: ServerResponse, message: string = 'Not found'): void {
  sendJson(res, 404, {
    error: {
      code: 'NOT_FOUND',
      message,
    },
  });
}

/**
 * Send 405 method not allowed response
 */
export function send405(res: ServerResponse, allowedMethods: string[]): void {
  res.setHeader('Allow', allowedMethods.join(', '));
  sendJson(res, 405, {
    error: {
      code: 'METHOD_NOT_ALLOWED',
      message: `Method not allowed. Allowed methods: ${allowedMethods.join(', ')}`,
    },
  });
}
