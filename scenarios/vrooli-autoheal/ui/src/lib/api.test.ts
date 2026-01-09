// API client error handling tests
import { describe, it, expect } from 'vitest';
import { APIError } from './api';

describe('[REQ:FAIL-SAFE-001] APIError', () => {
  describe('constructor', () => {
    it('creates error with correct properties', () => {
      const error = new APIError('Test message', 'DATABASE_ERROR', 500, 'req-123');

      expect(error.message).toBe('Test message');
      expect(error.code).toBe('DATABASE_ERROR');
      expect(error.statusCode).toBe(500);
      expect(error.requestId).toBe('req-123');
      expect(error.name).toBe('APIError');
    });
  });

  describe('isRetryable', () => {
    it('marks 5xx errors as retryable', () => {
      expect(new APIError('', 'ERROR', 500).isRetryable).toBe(true);
      expect(new APIError('', 'ERROR', 502).isRetryable).toBe(true);
      expect(new APIError('', 'ERROR', 503).isRetryable).toBe(true);
    });

    it('marks network errors (status 0) as retryable', () => {
      expect(new APIError('', 'NETWORK_ERROR', 0).isRetryable).toBe(true);
    });

    it('marks timeout (408) as retryable', () => {
      expect(new APIError('', 'TIMEOUT', 408).isRetryable).toBe(true);
    });

    it('marks 4xx errors as non-retryable', () => {
      expect(new APIError('', 'ERROR', 400).isRetryable).toBe(false);
      expect(new APIError('', 'ERROR', 404).isRetryable).toBe(false);
      expect(new APIError('', 'ERROR', 422).isRetryable).toBe(false);
    });
  });

  describe('getUserMessage', () => {
    it('returns friendly message for DATABASE_ERROR', () => {
      const error = new APIError('Raw error', 'DATABASE_ERROR', 500);
      expect(error.getUserMessage()).toContain('Database');
    });

    it('returns original message for NOT_FOUND', () => {
      const error = new APIError("Check 'test-123' not found", 'NOT_FOUND', 404);
      expect(error.getUserMessage()).toBe("Check 'test-123' not found");
    });

    it('returns friendly message for TIMEOUT', () => {
      const error = new APIError('Raw error', 'TIMEOUT', 504);
      expect(error.getUserMessage()).toContain('took too long');
    });

    it('returns friendly message for SERVICE_UNAVAILABLE', () => {
      const error = new APIError('Raw error', 'SERVICE_UNAVAILABLE', 503);
      expect(error.getUserMessage()).toContain('unavailable');
    });

    it('returns generic message for unknown codes', () => {
      const error = new APIError('Raw error', 'UNKNOWN_CODE', 500);
      expect(error.getUserMessage()).toBe('Something went wrong. Please try again.');
    });
  });

  describe('getSuggestedAction', () => {
    it('suggests retry for retryable errors', () => {
      const error = new APIError('', 'DATABASE_ERROR', 500);
      expect(error.getSuggestedAction()).toContain('Try again');
    });

    it('suggests checking removal for NOT_FOUND', () => {
      const error = new APIError('', 'NOT_FOUND', 404);
      expect(error.getSuggestedAction()).toContain('may have been removed');
    });

    it('suggests checking scenario for non-retryable errors', () => {
      const error = new APIError('', 'UNKNOWN', 400);
      expect(error.getSuggestedAction()).toContain('scenario');
    });
  });
});
