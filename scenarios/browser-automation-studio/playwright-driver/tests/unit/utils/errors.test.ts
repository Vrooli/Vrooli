import {
  PlaywrightDriverError,
  SessionNotFoundError,
  SelectorNotFoundError,
  FrameNotFoundError,
  TabNotFoundError,
  InvalidParameterError,
  UnsupportedInstructionError,
  AssertionFailedError,
  ResourceLimitError,
  TimeoutError,
  NetworkError,
  ValidationError,
} from '../../../src/utils/errors';

describe('Errors', () => {
  describe('PlaywrightDriverError', () => {
    it('should create base error with message and code', () => {
      const error = new PlaywrightDriverError('Test error', 'TEST_ERROR');

      expect(error.message).toBe('Test error');
      expect(error.code).toBe('TEST_ERROR');
      expect(error.kind).toBe('engine');
      expect(error.retryable).toBe(false);
      expect(error.name).toBe('PlaywrightDriverError');
    });

    it('should allow custom kind and retryable', () => {
      const error = new PlaywrightDriverError('Test error', 'TEST_ERROR', 'timeout', true);

      expect(error.kind).toBe('timeout');
      expect(error.retryable).toBe(true);
    });
  });

  describe('SessionNotFoundError', () => {
    it('should create session not found error', () => {
      const error = new SessionNotFoundError('session-123');

      expect(error.message).toBe('Session not found: session-123');
      expect(error.code).toBe('SESSION_NOT_FOUND');
      expect(error.kind).toBe('user');
      expect(error.retryable).toBe(false);
      expect(error.name).toBe('SessionNotFoundError');
    });
  });

  describe('SelectorNotFoundError', () => {
    it('should create selector not found error', () => {
      const error = new SelectorNotFoundError('#test-selector');

      expect(error.message).toBe('Selector not found: #test-selector');
      expect(error.code).toBe('SELECTOR_NOT_FOUND');
      expect(error.kind).toBe('user');
      expect(error.retryable).toBe(false);
      expect(error.name).toBe('SelectorNotFoundError');
    });

    it('should create error with timeout', () => {
      const error = new SelectorNotFoundError('#test-selector', 5000);

      expect(error.message).toBe('Selector not found: #test-selector (timeout: 5000ms)');
    });
  });

  describe('FrameNotFoundError', () => {
    it('should create frame not found error with selector', () => {
      const error = new FrameNotFoundError('iframe#test');

      expect(error.message).toBe('Frame not found: iframe#test');
      expect(error.code).toBe('FRAME_NOT_FOUND');
      expect(error.kind).toBe('user');
      expect(error.retryable).toBe(false);
    });
  });

  describe('TabNotFoundError', () => {
    it('should create tab not found error', () => {
      const error = new TabNotFoundError('tab-123');

      expect(error.message).toBe('Tab not found: tab-123');
      expect(error.code).toBe('TAB_NOT_FOUND');
      expect(error.kind).toBe('user');
      expect(error.retryable).toBe(false);
    });
  });

  describe('InvalidParameterError', () => {
    it('should create invalid parameter error', () => {
      const error = new InvalidParameterError('timeout', 'must be positive');

      expect(error.message).toBe('Invalid parameter "timeout": must be positive');
      expect(error.code).toBe('INVALID_PARAMETER');
      expect(error.kind).toBe('user');
      expect(error.retryable).toBe(false);
    });
  });

  describe('UnsupportedInstructionError', () => {
    it('should create unsupported instruction error', () => {
      const error = new UnsupportedInstructionError('custom-action');

      expect(error.message).toBe('Unsupported instruction type: custom-action');
      expect(error.code).toBe('UNSUPPORTED_INSTRUCTION');
      expect(error.kind).toBe('user');
      expect(error.retryable).toBe(false);
    });
  });

  describe('AssertionFailedError', () => {
    it('should create assertion failed error with expected and actual', () => {
      const error = new AssertionFailedError('Element should be visible', 'visible', 'hidden');

      expect(error.message).toBe('Element should be visible');
      expect(error.code).toBe('ASSERTION_FAILED');
      expect(error.kind).toBe('user');
      expect(error.retryable).toBe(false);
      expect(error.expected).toBe('visible');
      expect(error.actual).toBe('hidden');
    });
  });

  describe('ResourceLimitError', () => {
    it('should create resource limit error', () => {
      const error = new ResourceLimitError('sessions', 10);

      expect(error.message).toBe('Resource limit exceeded for sessions (limit: 10)');
      expect(error.code).toBe('RESOURCE_LIMIT_EXCEEDED');
      expect(error.kind).toBe('infra');
      expect(error.retryable).toBe(true);
    });
  });

  describe('TimeoutError', () => {
    it('should create timeout error with operation and duration', () => {
      const error = new TimeoutError('navigation', 30000);

      expect(error.message).toBe('Operation timed out: navigation (30000ms)');
      expect(error.code).toBe('TIMEOUT');
      expect(error.kind).toBe('timeout');
      expect(error.retryable).toBe(true);
    });
  });

  describe('NetworkError', () => {
    it('should create network error', () => {
      const error = new NetworkError('Failed to connect to server');

      expect(error.message).toBe('Failed to connect to server');
      expect(error.code).toBe('NETWORK_ERROR');
      expect(error.kind).toBe('infra');
      expect(error.retryable).toBe(true);
    });
  });

  describe('ValidationError', () => {
    it('should create validation error', () => {
      const error = new ValidationError('Invalid instruction format');

      expect(error.message).toBe('Invalid instruction format');
      expect(error.code).toBe('VALIDATION_ERROR');
      expect(error.kind).toBe('user');
      expect(error.retryable).toBe(false);
    });
  });

  describe('Error inheritance', () => {
    it('should inherit from Error', () => {
      const error = new PlaywrightDriverError('Test', 'TEST');

      expect(error instanceof Error).toBe(true);
      expect(error instanceof PlaywrightDriverError).toBe(true);
    });

    it('should have stack trace', () => {
      const error = new PlaywrightDriverError('Test', 'TEST');

      expect(error.stack).toBeDefined();
      expect(error.stack).toContain('PlaywrightDriverError');
    });
  });
});
