/**
 * Middleware Module
 *
 * HTTP request/response processing utilities:
 * - Body Parser: JSON parsing with size limits
 * - Error Handler: Normalize errors to structured responses
 *
 * These are transport-layer concerns separated from business logic.
 */

export * from './body-parser';
export * from './error-handler';
