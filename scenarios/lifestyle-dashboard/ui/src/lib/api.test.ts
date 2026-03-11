/**
 * Unit tests for API error handling.
 * Tests the APIError class and error categorization logic.
 *
 * [REQ:LD-FUNC-001] Tests error handling for API operations
 */
import { describe, it, expect } from 'vitest';
import { APIError, type APIErrorResponse, type ErrorCategory } from './api';

describe('APIError', () => {
  const createErrorResponse = (
    category: ErrorCategory,
    code: string,
    message: string,
    recovery?: string
  ): APIErrorResponse => ({
    error: true,
    category,
    code,
    message,
    recovery,
  });

  describe('constructor', () => {
    it('creates error with all properties', () => {
      const response = createErrorResponse(
        'validation',
        'MISSING_FIELD',
        'Field domain is required',
        'Check the request body'
      );
      const error = new APIError(response, 400);

      expect(error.name).toBe('APIError');
      expect(error.message).toBe('Field domain is required');
      expect(error.category).toBe('validation');
      expect(error.code).toBe('MISSING_FIELD');
      expect(error.recovery).toBe('Check the request body');
      expect(error.status).toBe(400);
    });

    it('handles response without recovery hint', () => {
      const response = createErrorResponse('internal', 'DATABASE_ERROR', 'Connection failed');
      const error = new APIError(response, 500);

      expect(error.recovery).toBeUndefined();
    });

    it('handles response with details', () => {
      const response: APIErrorResponse = {
        error: true,
        category: 'validation',
        code: 'INVALID_FIELD',
        message: 'Invalid value',
        details: { field: 'timestamp', value: 'not-a-date' },
      };
      const error = new APIError(response, 400);

      expect(error.details).toEqual({ field: 'timestamp', value: 'not-a-date' });
    });
  });

  describe('isRetryable', () => {
    it('returns true for internal errors', () => {
      const response = createErrorResponse('internal', 'DATABASE_ERROR', 'Connection failed');
      const error = new APIError(response, 500);

      expect(error.isRetryable).toBe(true);
    });

    it('returns true for unavailable errors', () => {
      const response = createErrorResponse('unavailable', 'SERVICE_DOWN', 'Service unavailable');
      const error = new APIError(response, 503);

      expect(error.isRetryable).toBe(true);
    });

    it('returns false for validation errors', () => {
      const response = createErrorResponse('validation', 'MISSING_FIELD', 'Field required');
      const error = new APIError(response, 400);

      expect(error.isRetryable).toBe(false);
    });

    it('returns false for not_found errors', () => {
      const response = createErrorResponse('not_found', 'EVENT_NOT_FOUND', 'Event not found');
      const error = new APIError(response, 404);

      expect(error.isRetryable).toBe(false);
    });

    it('returns false for conflict errors', () => {
      const response = createErrorResponse('conflict', 'STATE_CONFLICT', 'State conflict');
      const error = new APIError(response, 409);

      expect(error.isRetryable).toBe(false);
    });
  });

  describe('isValidation', () => {
    it('returns true for validation errors', () => {
      const response = createErrorResponse('validation', 'MISSING_FIELD', 'Field required');
      const error = new APIError(response, 400);

      expect(error.isValidation).toBe(true);
    });

    it('returns false for other error categories', () => {
      const categories: ErrorCategory[] = ['not_found', 'conflict', 'internal', 'unavailable'];

      categories.forEach(category => {
        const response = createErrorResponse(category, 'CODE', 'Message');
        const error = new APIError(response, 400);
        expect(error.isValidation).toBe(false);
      });
    });
  });

  describe('isNotFound', () => {
    it('returns true for not_found errors', () => {
      const response = createErrorResponse('not_found', 'EVENT_NOT_FOUND', 'Event not found');
      const error = new APIError(response, 404);

      expect(error.isNotFound).toBe(true);
    });

    it('returns false for other error categories', () => {
      const categories: ErrorCategory[] = ['validation', 'conflict', 'internal', 'unavailable'];

      categories.forEach(category => {
        const response = createErrorResponse(category, 'CODE', 'Message');
        const error = new APIError(response, 400);
        expect(error.isNotFound).toBe(false);
      });
    });
  });

  describe('error inheritance', () => {
    it('is instance of Error', () => {
      const response = createErrorResponse('validation', 'CODE', 'Message');
      const error = new APIError(response, 400);

      expect(error).toBeInstanceOf(Error);
    });

    it('can be caught as Error', () => {
      const response = createErrorResponse('validation', 'CODE', 'Message');
      const error = new APIError(response, 400);

      expect(() => { throw error; }).toThrow(Error);
    });

    it('preserves stack trace', () => {
      const response = createErrorResponse('validation', 'CODE', 'Message');
      const error = new APIError(response, 400);

      expect(error.stack).toBeDefined();
    });
  });

  describe('error categories coverage', () => {
    const testCases: Array<{
      category: ErrorCategory;
      status: number;
      isRetryable: boolean;
      isValidation: boolean;
      isNotFound: boolean;
    }> = [
      { category: 'validation', status: 400, isRetryable: false, isValidation: true, isNotFound: false },
      { category: 'not_found', status: 404, isRetryable: false, isValidation: false, isNotFound: true },
      { category: 'conflict', status: 409, isRetryable: false, isValidation: false, isNotFound: false },
      { category: 'internal', status: 500, isRetryable: true, isValidation: false, isNotFound: false },
      { category: 'unavailable', status: 503, isRetryable: true, isValidation: false, isNotFound: false },
    ];

    testCases.forEach(({ category, status, isRetryable, isValidation, isNotFound }) => {
      it(`categorizes ${category} error correctly`, () => {
        const response = createErrorResponse(category, 'CODE', `${category} error`);
        const error = new APIError(response, status);

        expect(error.category).toBe(category);
        expect(error.status).toBe(status);
        expect(error.isRetryable).toBe(isRetryable);
        expect(error.isValidation).toBe(isValidation);
        expect(error.isNotFound).toBe(isNotFound);
      });
    });
  });
});
