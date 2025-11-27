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

  logger.error('Request failed', {
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
