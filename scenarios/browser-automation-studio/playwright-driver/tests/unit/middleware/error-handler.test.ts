import { sendError, sendJson, send404, send405 } from '../../../src/middleware/error-handler';
import {
  SessionNotFoundError,
  SelectorNotFoundError,
  ResourceLimitError,
  TimeoutError,
} from '../../../src/utils/errors';
import { FailureKind } from '../../../src/proto';
import { createMockHttpResponse } from '../../helpers';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const getErrorPayload = (
  mockRes: ReturnType<typeof createMockHttpResponse>
): Record<string, unknown> => {
  const json = mockRes.getJSON();
  if (!isRecord(json)) {
    throw new Error('Expected JSON response to be an object');
  }
  const error = json.error;
  if (!isRecord(error)) {
    throw new Error('Expected JSON response to include an error object');
  }
  return error;
};

describe('Error Handler', () => {
  describe('sendJson', () => {
    it('should send JSON response', () => {
      const mockRes = createMockHttpResponse();

      sendJson(mockRes, 200, { success: true });

      expect(mockRes.statusCode).toBe(200);
      const setHeaderCalls = mockRes.setHeader.mock.calls;
      expect(setHeaderCalls[0]).toEqual(['Content-Type', 'application/json']);
      expect(mockRes.getJSON()).toEqual({ success: true });
    });
  });

  describe('send404', () => {
    it('should send 404 response', () => {
      const mockRes = createMockHttpResponse();

      send404(mockRes, 'Resource not found');

      expect(mockRes.statusCode).toBe(404);
      const error = getErrorPayload(mockRes);
      expect(error.code).toBe('NOT_FOUND');
      expect(error.message).toBe('Resource not found');
    });
  });

  describe('send405', () => {
    it('should send 405 response with allowed methods', () => {
      const mockRes = createMockHttpResponse();

      send405(mockRes, ['GET', 'POST']);

      expect(mockRes.statusCode).toBe(405);
      const setHeaderCalls = mockRes.setHeader.mock.calls;
      expect(setHeaderCalls[0]).toEqual(['Allow', 'GET, POST']);
      const error = getErrorPayload(mockRes);
      expect(error.code).toBe('METHOD_NOT_ALLOWED');
    });
  });

  describe('sendError', () => {
    it('should handle SessionNotFoundError', () => {
      const mockRes = createMockHttpResponse();
      const error = new SessionNotFoundError('session-123');

      sendError(mockRes, error);

      expect(mockRes.statusCode).toBe(404);
      const errorPayload = getErrorPayload(mockRes);
      expect(errorPayload.code).toBe('SESSION_NOT_FOUND');
      expect(errorPayload.kind).toBe(FailureKind.ENGINE);
      expect(errorPayload.retryable).toBe(false);
    });

    it('should handle SelectorNotFoundError', () => {
      const mockRes = createMockHttpResponse();
      const error = new SelectorNotFoundError('#missing');

      sendError(mockRes, error);

      // SelectorNotFoundError is mapped to 500 (engine error), not 400
      expect(mockRes.statusCode).toBe(500);
      const errorPayload = getErrorPayload(mockRes);
      expect(errorPayload.code).toBe('SELECTOR_NOT_FOUND');
      expect(errorPayload.kind).toBe(FailureKind.ENGINE);
      expect(errorPayload.retryable).toBe(true);
    });

    it('should handle ResourceLimitError', () => {
      const mockRes = createMockHttpResponse();
      // ResourceLimitError takes (message, details) - not (resource, limit)
      const error = new ResourceLimitError('Too many sessions', { limit: 10 });

      sendError(mockRes, error);

      expect(mockRes.statusCode).toBe(429);
      const errorPayload = getErrorPayload(mockRes);
      expect(errorPayload.code).toBe('RESOURCE_LIMIT');
      // ResourceLimitError has retryable=false by default
      expect(errorPayload.retryable).toBe(false);
    });

    it('should handle TimeoutError', () => {
      const mockRes = createMockHttpResponse();
      const error = new TimeoutError('Navigation timed out', 30000);

      sendError(mockRes, error);

      // TimeoutError is mapped to 500 (engine error), not 408
      expect(mockRes.statusCode).toBe(500);
      const errorPayload = getErrorPayload(mockRes);
      expect(errorPayload.kind).toBe(FailureKind.TIMEOUT);
      expect(errorPayload.retryable).toBe(true);
    });

    it('should handle generic Error', () => {
      const mockRes = createMockHttpResponse();
      const error = new Error('Something went wrong');

      sendError(mockRes, error);

      expect(mockRes.statusCode).toBe(500);
      const errorPayload = getErrorPayload(mockRes);
      expect(errorPayload.code).toBe('INTERNAL_ERROR');
      expect(errorPayload.message).toBe('Something went wrong');
    });

    it('should include request path in logging', () => {
      const mockRes = createMockHttpResponse();
      const error = new Error('Test error');

      sendError(mockRes, error, '/session/123/run');

      // Error should be sent successfully regardless of path
      expect(mockRes.statusCode).toBe(500);
    });
  });
});
