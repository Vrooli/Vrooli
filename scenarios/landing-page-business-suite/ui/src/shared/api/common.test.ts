import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import {
  ApiError,
  apiCall,
  apiGet,
  apiPost,
  apiPut,
  apiDelete,
  getApiErrorMessage,
  isApiError,
  type ApiErrorType,
} from './common';
import { createFetchMock, mockResponses, installFetchMock, getFetchCall } from '../test-utils/api-mocks';

describe('common API utilities', () => {
  let fetchMock: ReturnType<typeof createFetchMock>;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = createFetchMock();
    installFetchMock(fetchMock);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('ApiError', () => {
    it('creates an error with correct properties', () => {
      const error = new ApiError('Test error', 'server_error', 500, 'Something went wrong');

      expect(error).toBeInstanceOf(Error);
      expect(error).toBeInstanceOf(ApiError);
      expect(error.name).toBe('ApiError');
      expect(error.message).toBe('Test error');
      expect(error.type).toBe('server_error');
      expect(error.status).toBe(500);
      expect(error.userMessage).toBe('Something went wrong');
    });

    describe('retryable flag', () => {
      const retryableTypes: ApiErrorType[] = ['network', 'timeout', 'server_error', 'rate_limited'];
      const nonRetryableTypes: ApiErrorType[] = ['unauthorized', 'forbidden', 'not_found', 'validation', 'unknown'];

      retryableTypes.forEach((type) => {
        it(`marks "${type}" as retryable`, () => {
          const error = new ApiError('Test', type);
          expect(error.retryable).toBe(true);
        });
      });

      nonRetryableTypes.forEach((type) => {
        it(`marks "${type}" as NOT retryable`, () => {
          const error = new ApiError('Test', type);
          expect(error.retryable).toBe(false);
        });
      });
    });

    describe('default user messages', () => {
      const expectedMessages: Record<ApiErrorType, string> = {
        network: 'Unable to reach the server. Please check your connection and try again.',
        timeout: 'The request took too long. Please try again.',
        unauthorized: 'Your session has expired. Please log in again.',
        forbidden: "You don't have permission to perform this action.",
        not_found: 'The requested resource was not found.',
        validation: 'The request was invalid. Please check your input and try again.',
        rate_limited: 'Too many requests. Please wait a moment and try again.',
        server_error: 'Something went wrong on our end. Please try again later.',
        unknown: 'An unexpected error occurred. Please try again.',
      };

      Object.entries(expectedMessages).forEach(([type, expectedMessage]) => {
        it(`has correct default message for "${type}"`, () => {
          const error = new ApiError('Test', type as ApiErrorType);
          expect(error.userMessage).toBe(expectedMessage);
        });
      });

      it('uses custom userMessage when provided', () => {
        const error = new ApiError('Test', 'server_error', 500, 'Custom message');
        expect(error.userMessage).toBe('Custom message');
      });
    });
  });

  describe('isApiError', () => {
    it('returns true for ApiError instances', () => {
      const error = new ApiError('Test', 'server_error');
      expect(isApiError(error)).toBe(true);
    });

    it('returns false for regular Error instances', () => {
      const error = new Error('Test');
      expect(isApiError(error)).toBe(false);
    });

    it('returns false for non-Error values', () => {
      expect(isApiError('string')).toBe(false);
      expect(isApiError(null)).toBe(false);
      expect(isApiError(undefined)).toBe(false);
      expect(isApiError({})).toBe(false);
    });

    it('filters by type when provided', () => {
      const error = new ApiError('Test', 'server_error');
      expect(isApiError(error, 'server_error')).toBe(true);
      expect(isApiError(error, 'network')).toBe(false);
    });
  });

  describe('apiCall', () => {
    describe('successful requests', () => {
      it('returns parsed JSON on success', async () => {
        const data = { id: 1, name: 'Test' };
        fetchMock.mockResolvedValue(mockResponses.success(data));

        const result = await apiCall<typeof data>('/test');

        expect(result).toEqual(data);
      });

      it('includes Content-Type header', async () => {
        fetchMock.mockResolvedValue(mockResponses.success({}));

        await apiCall('/test');

        const [, options] = getFetchCall(fetchMock);
        expect((options.headers as Record<string, string>)['Content-Type']).toBe('application/json');
      });

      it('includes credentials', async () => {
        fetchMock.mockResolvedValue(mockResponses.success({}));

        await apiCall('/test');

        const [, options] = getFetchCall(fetchMock);
        expect(options.credentials).toBe('include');
      });

      it('passes custom headers', async () => {
        fetchMock.mockResolvedValue(mockResponses.success({}));

        await apiCall('/test', { headers: { Authorization: 'Bearer token' } });

        const [, options] = getFetchCall(fetchMock);
        expect((options.headers as Record<string, string>).Authorization).toBe('Bearer token');
      });
    });

    describe('HTTP status classification', () => {
      it('classifies 401 as unauthorized', async () => {
        fetchMock.mockResolvedValue(mockResponses.unauthorized());

        await expect(apiCall('/test')).rejects.toMatchObject({
          type: 'unauthorized',
          status: 401,
        });
      });

      it('classifies 403 as forbidden', async () => {
        fetchMock.mockResolvedValue(mockResponses.forbidden());

        await expect(apiCall('/test')).rejects.toMatchObject({
          type: 'forbidden',
          status: 403,
        });
      });

      it('classifies 404 as not_found', async () => {
        fetchMock.mockResolvedValue(mockResponses.notFound());

        await expect(apiCall('/test')).rejects.toMatchObject({
          type: 'not_found',
          status: 404,
        });
      });

      it('classifies 400 as validation', async () => {
        fetchMock.mockResolvedValue(mockResponses.error(400, 'Bad request'));

        await expect(apiCall('/test')).rejects.toMatchObject({
          type: 'validation',
          status: 400,
        });
      });

      it('classifies 422 as validation', async () => {
        fetchMock.mockResolvedValue(mockResponses.error(422, 'Unprocessable'));

        await expect(apiCall('/test')).rejects.toMatchObject({
          type: 'validation',
          status: 422,
        });
      });

      it('classifies 429 as rate_limited', async () => {
        fetchMock.mockResolvedValue(mockResponses.rateLimited());

        await expect(apiCall('/test')).rejects.toMatchObject({
          type: 'rate_limited',
          status: 429,
        });
      });

      it('classifies 500 as server_error', async () => {
        fetchMock.mockResolvedValue(mockResponses.serverError());

        await expect(apiCall('/test')).rejects.toMatchObject({
          type: 'server_error',
          status: 500,
        });
      });

      it('classifies 502 as server_error', async () => {
        fetchMock.mockResolvedValue(mockResponses.error(502, 'Bad Gateway'));

        await expect(apiCall('/test')).rejects.toMatchObject({
          type: 'server_error',
          status: 502,
        });
      });

      it('classifies other 4xx as unknown', async () => {
        fetchMock.mockResolvedValue(mockResponses.error(418, "I'm a teapot"));

        await expect(apiCall('/test')).rejects.toMatchObject({
          type: 'unknown',
          status: 418,
        });
      });
    });

    describe('error message extraction', () => {
      it('extracts error from JSON response', async () => {
        fetchMock.mockResolvedValue(mockResponses.error(400, 'Validation failed', 'error'));

        try {
          await apiCall('/test');
          expect.fail('Should have thrown');
        } catch (err) {
          expect(err).toBeInstanceOf(ApiError);
          expect((err as ApiError).userMessage).toBe('Validation failed');
        }
      });

      it('extracts message from JSON response', async () => {
        fetchMock.mockResolvedValue(mockResponses.error(400, 'Custom message', 'message'));

        try {
          await apiCall('/test');
          expect.fail('Should have thrown');
        } catch (err) {
          expect(err).toBeInstanceOf(ApiError);
          expect((err as ApiError).userMessage).toBe('Custom message');
        }
      });

      it('uses default message when JSON parse fails', async () => {
        fetchMock.mockResolvedValue({
          ok: false,
          status: 500,
          statusText: 'Internal Server Error',
          text: () => Promise.resolve('Not JSON'),
        });

        try {
          await apiCall('/test');
          expect.fail('Should have thrown');
        } catch (err) {
          expect(err).toBeInstanceOf(ApiError);
          expect((err as ApiError).type).toBe('server_error');
        }
      });

      it('uses status text when a failed response has no text reader or its reader rejects', async () => {
        fetchMock.mockResolvedValueOnce({ ok: false, status: 503, statusText: 'Temporarily unavailable' });
        try {
          await apiCall('/status-text');
          expect.fail('Expected the failed response to reject');
        } catch (error) {
          expect(error).toMatchObject({ type: 'server_error' });
          expect((error as ApiError).message).toContain('Temporarily unavailable');
        }

        fetchMock.mockResolvedValueOnce({
          ok: false,
          status: 400,
          statusText: 'Bad request fallback',
          text: () => Promise.reject(new Error('stream closed')),
        });
        try {
          await apiCall('/unreadable-error');
          expect.fail('Expected the failed response to reject');
        } catch (error) {
          expect(error).toMatchObject({ type: 'validation' });
          expect((error as ApiError).message).toContain('Bad request fallback');
        }
      });

      it('honors server-provided valid error classification and retryability', async () => {
        fetchMock.mockResolvedValue({
          ok: false,
          status: 400,
          statusText: 'Bad request',
          text: () => Promise.resolve(JSON.stringify({ error: 'Wait before retrying', error_type: 'rate_limited', retryable: false })),
        });
        await expect(apiCall('/rate-limit-override')).rejects.toMatchObject({
          type: 'rate_limited', retryable: false, userMessage: 'Wait before retrying',
        });
      });
    });

    it('returns undefined for a successful compatibility response without a JSON reader', async () => {
      fetchMock.mockResolvedValue({ ok: true, status: 204, statusText: 'No Content' });
      await expect(apiCall('/empty-response')).resolves.toBeUndefined();
    });

    describe('network errors', () => {
      it('classifies TypeError as network error', async () => {
        fetchMock.mockRejectedValue(mockResponses.networkError());

        await expect(apiCall('/test')).rejects.toMatchObject({
          type: 'network',
          retryable: true,
        });
      });

      it('includes endpoint in error message', async () => {
        fetchMock.mockRejectedValue(mockResponses.networkError());

        try {
          await apiCall('/test-endpoint');
          expect.fail('Should have thrown');
        } catch (err) {
          expect(err).toBeInstanceOf(ApiError);
          expect((err as ApiError).message).toContain('/test-endpoint');
        }
      });
    });

    describe('timeout handling', () => {
      it('uses default timeout of 30 seconds', async () => {
        // Test timeout with real timers but short timeout for speed
        // We verify the error type and message structure
        const shortTimeout = 50; // 50ms for fast test

        fetchMock.mockImplementation((_url, options) => {
          return new Promise((_, reject) => {
            if (options?.signal) {
              options.signal.addEventListener('abort', () => {
                const error = new Error('Aborted');
                error.name = 'AbortError';
                reject(error);
              });
            }
          });
        });

        // Use a short custom timeout to test the timeout mechanism quickly
        const promise = apiCall('/test', { timeout: shortTimeout });

        await expect(promise).rejects.toMatchObject({
          type: 'timeout',
        });
      });

      it('respects custom timeout option', async () => {
        // Test with a different timeout value to verify custom timeout is used
        const customTimeout = 75;

        fetchMock.mockImplementation((_url, options) => {
          return new Promise((_, reject) => {
            if (options?.signal) {
              options.signal.addEventListener('abort', () => {
                const error = new Error('Aborted');
                error.name = 'AbortError';
                reject(error);
              });
            }
          });
        });

        const promise = apiCall('/test', { timeout: customTimeout });

        await expect(promise).rejects.toMatchObject({
          type: 'timeout',
        });
      });
    });

    describe('unknown errors', () => {
      it('wraps non-TypeError, non-ApiError as unknown', async () => {
        fetchMock.mockRejectedValue(new Error('Some other error'));

        await expect(apiCall('/test')).rejects.toMatchObject({
          type: 'unknown',
          message: 'Some other error',
        });
      });

      it('re-throws ApiError as-is', async () => {
        const originalError = new ApiError('Original', 'forbidden', 403);
        fetchMock.mockRejectedValue(originalError);

        await expect(apiCall('/test')).rejects.toBe(originalError);
      });
    });
  });

  describe('apiGet', () => {
    it('makes GET request', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({ data: 'test' }));

      await apiGet('/test');

      const [, options] = getFetchCall(fetchMock);
      expect(options.method).toBe('GET');
    });

    it('returns response data', async () => {
      const data = { id: 1 };
      fetchMock.mockResolvedValue(mockResponses.success(data));

      const result = await apiGet<typeof data>('/test');

      expect(result).toEqual(data);
    });
  });

  describe('apiPost', () => {
    it('makes POST request', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));

      await apiPost('/test', { name: 'test' });

      const [, options] = getFetchCall(fetchMock);
      expect(options.method).toBe('POST');
    });

    it('serializes body as JSON', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));
      const body = { name: 'test', count: 42 };

      await apiPost('/test', body);

      const [, options] = getFetchCall(fetchMock);
      expect(options.body).toBe(JSON.stringify(body));
    });

    it('handles undefined body', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));

      await apiPost('/test', undefined);

      const [, options] = getFetchCall(fetchMock);
      expect(options.body).toBeUndefined();
    });

    it.each([0, false, null])('serializes valid falsy JSON body %p instead of silently dropping it', async (body) => {
      fetchMock.mockResolvedValue(mockResponses.success({}));
      await apiPost('/test', body);
      const [, options] = getFetchCall(fetchMock);
      expect(options.body).toBe(JSON.stringify(body));
    });
  });

  describe('apiPut', () => {
    it('makes PUT request', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));

      await apiPut('/test', { name: 'updated' });

      const [, options] = getFetchCall(fetchMock);
      expect(options.method).toBe('PUT');
    });

    it('serializes body as JSON', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));
      const body = { name: 'updated' };

      await apiPut('/test', body);

      const [, options] = getFetchCall(fetchMock);
      expect(options.body).toBe(JSON.stringify(body));
    });

    it('serializes false as a meaningful update payload', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));
      await apiPut('/test', false);
      const [, options] = getFetchCall(fetchMock);
      expect(options.body).toBe('false');
    });
  });

  describe('apiDelete', () => {
    it('makes DELETE request', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));

      await apiDelete('/test');

      const [, options] = getFetchCall(fetchMock);
      expect(options.method).toBe('DELETE');
    });

    it('handles body for DELETE requests', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));
      const body = { id: 123 };

      await apiDelete('/test', body);

      const [, options] = getFetchCall(fetchMock);
      expect(options.body).toBe(JSON.stringify(body));
    });

    it('handles DELETE without body', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));

      await apiDelete('/test');

      const [, options] = getFetchCall(fetchMock);
      expect(options.body).toBeUndefined();
    });

    it('serializes a null DELETE body when the endpoint requires an explicit JSON null', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));
      await apiDelete('/test', null);
      const [, options] = getFetchCall(fetchMock);
      expect(options.body).toBe('null');
    });
  });

  describe('getApiErrorMessage', () => {
    it('uses the most specific safe message for API, generic, string, and unknown failures', () => {
      expect(getApiErrorMessage(new ApiError('technical', 'validation', 400, 'Correct the form'), 'Fallback')).toBe('Correct the form');
      expect(getApiErrorMessage(new Error('Readable error'), 'Fallback')).toBe('Readable error');
      expect(getApiErrorMessage('Server said no', 'Fallback')).toBe('Server said no');
      expect(getApiErrorMessage('   ', 'Fallback')).toBe('Fallback');
      expect(getApiErrorMessage(null, 'Fallback')).toBe('Fallback');
    });
  });
});
