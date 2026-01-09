import { sendError, sendJson, send404, send405 } from '../../../src/middleware/error-handler';
import {
  SessionNotFoundError,
  SelectorNotFoundError,
  ResourceLimitError,
  TimeoutError,
} from '../../../src/utils/errors';
import { FailureKind } from '../../../src/proto';
import { createMockHttpResponse } from '../../helpers';

describe('Error Handler', () => {
  describe('sendJson', () => {
    it('should send JSON response', () => {
      const mockRes = createMockHttpResponse();

      sendJson(mockRes, 200, { success: true });

      expect(mockRes.statusCode).toBe(200);
      expect(mockRes.setHeader).toHaveBeenCalledWith('Content-Type', 'application/json');
      expect((mockRes as any).getJSON()).toEqual({ success: true });
    });
  });

  describe('send404', () => {
    it('should send 404 response', () => {
      const mockRes = createMockHttpResponse();

      send404(mockRes, 'Resource not found');

      expect(mockRes.statusCode).toBe(404);
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('NOT_FOUND');
      expect(json.error.message).toBe('Resource not found');
    });
  });

  describe('send405', () => {
    it('should send 405 response with allowed methods', () => {
      const mockRes = createMockHttpResponse();

      send405(mockRes, ['GET', 'POST']);

      expect(mockRes.statusCode).toBe(405);
      expect(mockRes.setHeader).toHaveBeenCalledWith('Allow', 'GET, POST');
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('METHOD_NOT_ALLOWED');
    });
  });

  describe('sendError', () => {
    it('should handle SessionNotFoundError', () => {
      const mockRes = createMockHttpResponse();
      const error = new SessionNotFoundError('session-123');

      sendError(mockRes, error);

      expect(mockRes.statusCode).toBe(404);
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('SESSION_NOT_FOUND');
      expect(json.error.kind).toBe(FailureKind.ENGINE);
      expect(json.error.retryable).toBe(false);
    });

    it('should handle SelectorNotFoundError', () => {
      const mockRes = createMockHttpResponse();
      const error = new SelectorNotFoundError('#missing');

      sendError(mockRes, error);

      // SelectorNotFoundError is mapped to 500 (engine error), not 400
      expect(mockRes.statusCode).toBe(500);
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('SELECTOR_NOT_FOUND');
      expect(json.error.kind).toBe(FailureKind.ENGINE);
      expect(json.error.retryable).toBe(true);
    });

    it('should handle ResourceLimitError', () => {
      const mockRes = createMockHttpResponse();
      // ResourceLimitError takes (message, details) - not (resource, limit)
      const error = new ResourceLimitError('Too many sessions', { limit: 10 });

      sendError(mockRes, error);

      expect(mockRes.statusCode).toBe(429);
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('RESOURCE_LIMIT');
      // ResourceLimitError has retryable=false by default
      expect(json.error.retryable).toBe(false);
    });

    it('should handle TimeoutError', () => {
      const mockRes = createMockHttpResponse();
      const error = new TimeoutError('Navigation timed out', 30000);

      sendError(mockRes, error);

      // TimeoutError is mapped to 500 (engine error), not 408
      expect(mockRes.statusCode).toBe(500);
      const json = (mockRes as any).getJSON();
      expect(json.error.kind).toBe(FailureKind.TIMEOUT);
      expect(json.error.retryable).toBe(true);
    });

    it('should handle generic Error', () => {
      const mockRes = createMockHttpResponse();
      const error = new Error('Something went wrong');

      sendError(mockRes, error);

      expect(mockRes.statusCode).toBe(500);
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('INTERNAL_ERROR');
      expect(json.error.message).toBe('Something went wrong');
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
