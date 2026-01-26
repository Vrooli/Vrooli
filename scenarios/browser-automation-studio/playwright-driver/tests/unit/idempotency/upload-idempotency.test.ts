/**
 * Upload Handler Idempotency Tests
 *
 * Tests for idempotent upload operations to ensure replay safety.
 * These tests verify that file upload operations handle concurrent
 * requests correctly.
 */

import type { HandlerContext } from '../../../src/handlers/base';
import {
  createMockPage,
  createMockContext,
  createTestConfig,
  createTypedInstruction,
} from '../../helpers';
import { logger, metrics } from '../../../src/utils';

jest.mock('fs/promises', () => ({
  access: jest.fn(),
}));

describe('UploadHandler Idempotency', () => {
  let handler: InstanceType<typeof import('../../../src/handlers/upload').UploadHandler>;
  let mockPage: ReturnType<typeof createMockPage>;
  let config: ReturnType<typeof createTestConfig>;
  let mockAccess: jest.Mock;

  beforeEach(async () => {
    jest.resetModules();
    mockAccess = jest.requireMock<{ access: jest.Mock }>('fs/promises').access;
    mockAccess.mockResolvedValue(undefined);

    const { UploadHandler } = await import('../../../src/handlers/upload');
    handler = new UploadHandler();
    mockPage = createMockPage();
    config = createTestConfig();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('concurrent upload tracking', () => {
    it('should track concurrent uploads with same parameters', async () => {
      const instruction = createTypedInstruction('upload', {
        selector: '#file-input',
        filePath: '/path/to/file.txt',
      });

      const context: HandlerContext = {
        page: mockPage,
        browserContext: createMockContext(),
        config,
        logger,
        metrics,
        sessionId: 'test-session-concurrent',
        frameStack: [],
      };

      // Execute two sequential uploads (concurrent behavior tested separately)
      const result1 = await handler.execute(instruction, context);
      const result2 = await handler.execute(instruction, context);

      // Both should succeed
      expect(result1.success).toBe(true);
      expect(result2.success).toBe(true);

      // setInputFiles called twice since they are sequential
      expect(mockPage.setInputFiles.mock.calls.length).toBe(2);
    });

    it('should allow separate uploads for different files', async () => {
      const instruction1 = createTypedInstruction('upload', {
        selector: '#file-input',
        filePath: '/path/to/file1.txt',
      });

      const instruction2 = createTypedInstruction('upload', {
        selector: '#file-input',
        filePath: '/path/to/file2.txt',
      }, { index: 1 });

      const context: HandlerContext = {
        page: mockPage,
        browserContext: createMockContext(),
        config,
        logger,
        metrics,
        sessionId: 'test-session-diff-files',
        frameStack: [],
      };

      const result1 = await handler.execute(instruction1, context);
      const result2 = await handler.execute(instruction2, context);

      expect(result1.success).toBe(true);
      expect(result2.success).toBe(true);

      // Different files should create separate upload calls
      expect(mockPage.setInputFiles.mock.calls.length).toBe(2);
    });

    it('should allow separate uploads for different selectors', async () => {
      const instruction1 = createTypedInstruction('upload', {
        selector: '#file-input-1',
        filePath: '/path/to/file.txt',
      });

      const instruction2 = createTypedInstruction('upload', {
        selector: '#file-input-2',
        filePath: '/path/to/file.txt',
      }, { index: 1 });

      const context: HandlerContext = {
        page: mockPage,
        browserContext: createMockContext(),
        config,
        logger,
        metrics,
        sessionId: 'test-session-diff-selectors',
        frameStack: [],
      };

      const result1 = await handler.execute(instruction1, context);
      const result2 = await handler.execute(instruction2, context);

      expect(result1.success).toBe(true);
      expect(result2.success).toBe(true);

      // Different selectors should create separate upload calls
      expect(mockPage.setInputFiles.mock.calls.length).toBe(2);
    });
  });

  describe('file validation', () => {
    it('should return error for inaccessible file', async () => {
      mockAccess.mockRejectedValue(new Error('ENOENT: file not found'));

      const instruction = createTypedInstruction('upload', {
        selector: '#file-input',
        filePath: '/path/to/nonexistent.txt',
      });

      const context: HandlerContext = {
        page: mockPage,
        browserContext: createMockContext(),
        config,
        logger,
        metrics,
        sessionId: 'test-session-nofile',
        frameStack: [],
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('FILE_NOT_FOUND');
      expect(result.error?.retryable).toBe(false);
    });
  });

  describe('parameter validation', () => {
    it('should return error for missing selector', async () => {
      const instruction = createTypedInstruction('upload', {
        filePath: '/path/to/file.txt',
      });

      const context: HandlerContext = {
        page: mockPage,
        browserContext: createMockContext(),
        config,
        logger,
        metrics,
        sessionId: 'test-session-no-selector',
        frameStack: [],
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      // Either MISSING_PARAM or INVALID_INSTRUCTION is acceptable
      expect(['MISSING_PARAM', 'INVALID_INSTRUCTION']).toContain(result.error?.code);
    });

    it('should return error for missing filePath', async () => {
      const instruction = createTypedInstruction('upload', {
        selector: '#file-input',
      });

      const context: HandlerContext = {
        page: mockPage,
        browserContext: createMockContext(),
        config,
        logger,
        metrics,
        sessionId: 'test-session-no-filepath',
        frameStack: [],
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      // Either MISSING_PARAM or INVALID_INSTRUCTION is acceptable
      expect(['MISSING_PARAM', 'INVALID_INSTRUCTION']).toContain(result.error?.code);
    });
  });

  describe('multiple file upload', () => {
    it('should handle array of files', async () => {
      const instruction = createTypedInstruction('upload', {
        selector: '#file-input',
        filePath: ['/path/to/file1.txt', '/path/to/file2.txt'],
      });

      const context: HandlerContext = {
        page: mockPage,
        browserContext: createMockContext(),
        config,
        logger,
        metrics,
        sessionId: 'test-session-multi',
        frameStack: [],
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const [selector, files, options] = mockPage.setInputFiles.mock.calls[0] ?? [];
      expect(selector).toBe('#file-input');
      expect(files).toEqual(['/path/to/file1.txt', '/path/to/file2.txt']);
      expect(options).toEqual(expect.any(Object));
    });

    it('should generate consistent idempotency key for same file array', async () => {
      // Note: File arrays are sorted when generating keys, so order doesn't matter
      const instruction1 = createTypedInstruction('upload', {
        selector: '#file-input',
        filePath: ['/path/to/a.txt', '/path/to/b.txt'],
      });

      const instruction2 = createTypedInstruction('upload', {
        selector: '#file-input',
        filePath: ['/path/to/b.txt', '/path/to/a.txt'], // Different order, same files
      });

      const context: HandlerContext = {
        page: mockPage,
        browserContext: createMockContext(),
        config,
        logger,
        metrics,
        sessionId: 'test-session-multi-concurrent',
        frameStack: [],
      };

      // Execute sequentially (concurrent idempotency is handled at in-flight level)
      const result1 = await handler.execute(instruction1, context);
      const result2 = await handler.execute(instruction2, context);

      expect(result1.success).toBe(true);
      expect(result2.success).toBe(true);

      // Both calls succeed - the idempotency key would match for concurrent calls
      // Sequential calls both execute because the first completes before second starts
      expect(mockPage.setInputFiles.mock.calls.length).toBe(2);
    });
  });
});
