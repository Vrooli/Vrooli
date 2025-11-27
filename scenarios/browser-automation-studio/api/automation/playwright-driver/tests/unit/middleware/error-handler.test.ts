import { sendError, sendJson, send404, send405 } from '../../../src/middleware/error-handler';
import {
  SessionNotFoundError,
  SelectorNotFoundError,
  ResourceLimitError,
  TimeoutError,
  ValidationError,
} from '../../../src/utils/errors';
import { createMockResponse, waitForResponse } from '../../helpers';

describe('Error Handler', () => {
  describe('sendJson', () => {
    it('should send JSON response', async () => {
      const mockRes = createMockResponse();

      sendJson(mockRes, 200, { success: true });
      await waitForResponse(mockRes);

      expect(mockRes.statusCode).toBe(200);
      expect(mockRes.setHeader).toHaveBeenCalledWith('Content-Type', 'application/json');
      expect((mockRes as any).getJSON()).toEqual({ success: true });
    });
  });

  describe('send404', () => {
    it('should send 404 response', async () => {
      const mockRes = createMockResponse();

      send404(mockRes, 'Resource not found');
      await waitForResponse(mockRes);

      expect(mockRes.statusCode).toBe(404);
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('NOT_FOUND');
      expect(json.error.message).toBe('Resource not found');
    });
  });

  describe('send405', () => {
    it('should send 405 response with allowed methods', async () => {
      const mockRes = createMockResponse();

      send405(mockRes, ['GET', 'POST']);
      await waitForResponse(mockRes);

      expect(mockRes.statusCode).toBe(405);
      expect(mockRes.setHeader).toHaveBeenCalledWith('Allow', 'GET, POST');
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('METHOD_NOT_ALLOWED');
    });
  });

  describe('sendError', () => {
    it('should handle SessionNotFoundError', async () => {
      const mockRes = createMockResponse();
      const error = new SessionNotFoundError('session-123');

      sendError(mockRes, error);
      await waitForResponse(mockRes);

      expect(mockRes.statusCode).toBe(404);
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('SESSION_NOT_FOUND');
      expect(json.error.kind).toBe('user');
      expect(json.error.retryable).toBe(false);
    });

    it('should handle SelectorNotFoundError', async () => {
      const mockRes = createMockResponse();
      const error = new SelectorNotFoundError('#missing');

      sendError(mockRes, error);
      await waitForResponse(mockRes);

      expect(mockRes.statusCode).toBe(400);
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('SELECTOR_NOT_FOUND');
    });

    it('should handle ResourceLimitError', async () => {
      const mockRes = createMockResponse();
      const error = new ResourceLimitError('sessions', 10);

      sendError(mockRes, error);
      await waitForResponse(mockRes);

      expect(mockRes.statusCode).toBe(429);
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('RESOURCE_LIMIT_EXCEEDED');
      expect(json.error.retryable).toBe(true);
    });

    it('should handle TimeoutError', async () => {
      const mockRes = createMockResponse();
      const error = new TimeoutError('navigation', 30000);

      sendError(mockRes, error);
      await waitForResponse(mockRes);

      expect(mockRes.statusCode).toBe(408);
      const json = (mockRes as any).getJSON();
      expect(json.error.kind).toBe('timeout');
    });

    it('should handle ValidationError', async () => {
      const mockRes = createMockResponse();
      const error = new ValidationError('Invalid params');

      sendError(mockRes, error);
      await waitForResponse(mockRes);

      expect(mockRes.statusCode).toBe(400);
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('VALIDATION_ERROR');
    });

    it('should handle generic Error', async () => {
      const mockRes = createMockResponse();
      const error = new Error('Something went wrong');

      sendError(mockRes, error);
      await waitForResponse(mockRes);

      expect(mockRes.statusCode).toBe(500);
      const json = (mockRes as any).getJSON();
      expect(json.error.code).toBe('INTERNAL_ERROR');
      expect(json.error.message).toBe('Something went wrong');
    });

    it('should include request path in logging', async () => {
      const mockRes = createMockResponse();
      const error = new Error('Test error');

      sendError(mockRes, error, '/session/123/run');
      await waitForResponse(mockRes);

      // Error should be sent successfully regardless of path
      expect(mockRes.statusCode).toBe(500);
    });
  });
});
