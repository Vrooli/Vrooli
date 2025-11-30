import {
  PlaywrightDriverError,
  SessionNotFoundError,
  SelectorNotFoundError,
  FrameNotFoundError,
  InvalidInstructionError,
  UnsupportedInstructionError,
  ResourceLimitError,
  TimeoutError,
  NavigationError,
  ConfigurationError,
  normalizeError,
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

    it('should accept details', () => {
      const error = new PlaywrightDriverError('Test error', 'TEST_ERROR', 'engine', false, { foo: 'bar' });

      expect(error.details).toEqual({ foo: 'bar' });
    });
  });

  describe('SessionNotFoundError', () => {
    it('should create session not found error', () => {
      const error = new SessionNotFoundError('session-123');

      expect(error.message).toBe('Session not found: session-123');
      expect(error.code).toBe('SESSION_NOT_FOUND');
      expect(error.kind).toBe('engine');
      expect(error.retryable).toBe(false);
      expect(error.name).toBe('SessionNotFoundError');
    });
  });

  describe('SelectorNotFoundError', () => {
    it('should create selector not found error', () => {
      const error = new SelectorNotFoundError('#test-selector');

      expect(error.message).toBe('Selector not found: #test-selector');
      expect(error.code).toBe('SELECTOR_NOT_FOUND');
      expect(error.kind).toBe('engine');
      expect(error.retryable).toBe(true);
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

      expect(error.message).toContain('Frame not found');
      expect(error.message).toContain('selector=iframe#test');
      expect(error.code).toBe('FRAME_NOT_FOUND');
      expect(error.kind).toBe('engine');
      expect(error.retryable).toBe(true);
    });

    it('should create frame not found error with frameId', () => {
      const error = new FrameNotFoundError(undefined, 'frame-123');

      expect(error.message).toContain('frameId=frame-123');
    });

    it('should create frame not found error with frameUrl', () => {
      const error = new FrameNotFoundError(undefined, undefined, 'https://example.com/frame');

      expect(error.message).toContain('url=https://example.com/frame');
    });
  });

  describe('InvalidInstructionError', () => {
    it('should create invalid instruction error', () => {
      const error = new InvalidInstructionError('Missing selector parameter');

      expect(error.message).toBe('Missing selector parameter');
      expect(error.code).toBe('INVALID_INSTRUCTION');
      expect(error.kind).toBe('orchestration');
      expect(error.retryable).toBe(false);
    });
  });

  describe('UnsupportedInstructionError', () => {
    it('should create unsupported instruction error', () => {
      const error = new UnsupportedInstructionError('custom-action');

      expect(error.message).toBe('Unsupported instruction type: custom-action');
      expect(error.code).toBe('UNSUPPORTED_INSTRUCTION');
      expect(error.kind).toBe('orchestration');
      expect(error.retryable).toBe(false);
    });
  });

  describe('ResourceLimitError', () => {
    it('should create resource limit error', () => {
      const error = new ResourceLimitError('Too many sessions', { limit: 10, current: 10 });

      expect(error.message).toBe('Too many sessions');
      expect(error.code).toBe('RESOURCE_LIMIT');
      expect(error.kind).toBe('infra');
      expect(error.retryable).toBe(false);
      expect(error.details).toEqual({ limit: 10, current: 10 });
    });
  });

  describe('TimeoutError', () => {
    it('should create timeout error with message and timeout', () => {
      const error = new TimeoutError('Navigation timed out', 30000);

      expect(error.message).toBe('Navigation timed out');
      expect(error.code).toBe('TIMEOUT');
      expect(error.kind).toBe('timeout');
      expect(error.retryable).toBe(true);
      expect(error.details).toEqual({ timeout: 30000 });
    });
  });

  describe('NavigationError', () => {
    it('should create navigation error', () => {
      const error = new NavigationError('Page load failed', 'https://example.com');

      expect(error.message).toBe('Page load failed');
      expect(error.code).toBe('NAVIGATION_ERROR');
      expect(error.kind).toBe('engine');
      expect(error.retryable).toBe(true);
      expect(error.details).toEqual({ url: 'https://example.com' });
    });
  });

  describe('ConfigurationError', () => {
    it('should create configuration error', () => {
      const error = new ConfigurationError('Invalid browser config', { option: 'headless' });

      expect(error.message).toBe('Invalid browser config');
      expect(error.code).toBe('CONFIGURATION_ERROR');
      expect(error.kind).toBe('orchestration');
      expect(error.retryable).toBe(false);
    });
  });

  describe('normalizeError', () => {
    it('should return PlaywrightDriverError unchanged', () => {
      const original = new SelectorNotFoundError('#test');
      const result = normalizeError(original);

      expect(result).toBe(original);
    });

    it('should convert timeout errors', () => {
      const original = new Error('Timeout 30000ms exceeded');
      const result = normalizeError(original);

      expect(result).toBeInstanceOf(TimeoutError);
      expect(result.code).toBe('TIMEOUT');
    });

    it('should convert selector errors', () => {
      const original = new Error('Element not found for selector');
      const result = normalizeError(original);

      expect(result).toBeInstanceOf(SelectorNotFoundError);
      expect(result.code).toBe('SELECTOR_NOT_FOUND');
    });

    it('should convert navigation errors', () => {
      const original = new Error('Navigation failed');
      const result = normalizeError(original);

      expect(result).toBeInstanceOf(NavigationError);
      expect(result.code).toBe('NAVIGATION_ERROR');
    });

    it('should convert frame errors', () => {
      const original = new Error('Frame not found');
      const result = normalizeError(original);

      expect(result).toBeInstanceOf(FrameNotFoundError);
      expect(result.code).toBe('FRAME_NOT_FOUND');
    });

    it('should create generic error for unknown errors', () => {
      const original = new Error('Something unexpected happened');
      const result = normalizeError(original);

      expect(result).toBeInstanceOf(PlaywrightDriverError);
      expect(result.code).toBe('PLAYWRIGHT_ERROR');
    });

    it('should handle non-Error objects', () => {
      const result = normalizeError('string error');

      expect(result).toBeInstanceOf(PlaywrightDriverError);
      expect(result.code).toBe('UNKNOWN_ERROR');
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
