/**
 * Frame Handler Idempotency Tests
 *
 * Tests for idempotent frame operations to ensure replay safety.
 * These tests verify that frame operations handle stale references,
 * concurrent requests, and repeated operations correctly.
 */

import { FrameHandler } from '../../../src/handlers/frame';
import type { CompiledInstruction } from '../../../src/types';
import type { HandlerContext } from '../../../src/handlers/base';
import { createMockPage, createMockFrame, createTestConfig } from '../../helpers';
import { logger, metrics } from '../../../src/utils';

describe('FrameHandler Idempotency', () => {
  let handler: FrameHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let config: ReturnType<typeof createTestConfig>;

  beforeEach(() => {
    handler = new FrameHandler();
    mockPage = createMockPage();
    config = createTestConfig();
  });

  describe('frame-switch enter idempotency', () => {
    it('should return success with idempotent flag when already in target frame', async () => {
      // Create a mock frame
      const targetFrame = createMockFrame();
      targetFrame.url.mockReturnValue('https://example.com/iframe');
      (targetFrame.name as jest.Mock).mockReturnValue('test-iframe');

      // Setup page.frames() to return our target frame
      mockPage.frames.mockReturnValue([targetFrame]);

      // Setup the frame stack with the target frame already entered
      const frameStack = [targetFrame];

      const instruction: CompiledInstruction = {
        type: 'frame-switch',
        node_id: 'test-node',
        index: 0,
        params: {
          action: 'enter',
          frameUrl: 'https://example.com/iframe',
        },
      };

      const context: HandlerContext = {
        page: mockPage,
        context: {} as any,
        config,
        logger,
        metrics,
        sessionId: 'test-session-enter-idempotent',
        frameStack,
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const data = result.extracted_data as Record<string, unknown>;
      expect(data?.idempotent).toBe(true);
      expect(data?.frameUrl).toBe('https://example.com/iframe');
    });

    it('should enter frame normally when not already in it', async () => {
      const mainFrame = createMockFrame();
      mainFrame.url.mockReturnValue('https://example.com');
      (mainFrame.name as jest.Mock).mockReturnValue('');

      const targetFrame = createMockFrame();
      targetFrame.url.mockReturnValue('https://example.com/iframe');
      (targetFrame.name as jest.Mock).mockReturnValue('test-iframe');

      mockPage.frames.mockReturnValue([mainFrame, targetFrame]);
      mockPage.mainFrame.mockReturnValue(mainFrame);

      // Empty frame stack - not currently in any iframe
      const frameStack: any[] = [];

      const instruction: CompiledInstruction = {
        type: 'frame-switch',
        node_id: 'test-node',
        index: 0,
        params: {
          action: 'enter',
          frameUrl: 'https://example.com/iframe',
        },
      };

      const context: HandlerContext = {
        page: mockPage,
        context: {} as any,
        config,
        logger,
        metrics,
        sessionId: 'test-session-enter-normal',
        frameStack,
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const data = result.extracted_data as Record<string, unknown>;
      // Should NOT have idempotent flag since we actually entered
      expect(data?.idempotent).toBeUndefined();
      expect(data?.stackDepth).toBe(1);
    });

    it('should return error when frame not found', async () => {
      mockPage.frames.mockReturnValue([]);

      const instruction: CompiledInstruction = {
        type: 'frame-switch',
        node_id: 'test-node',
        index: 0,
        params: {
          action: 'enter',
          frameUrl: 'https://nonexistent.com/iframe',
        },
      };

      const context: HandlerContext = {
        page: mockPage,
        context: {} as any,
        config,
        logger,
        metrics,
        sessionId: 'test-session-enter-notfound',
        frameStack: [],
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('FRAME_NOT_FOUND');
    });
  });

  describe('frame-switch exit idempotency', () => {
    it('should return error when already at main frame', async () => {
      const instruction: CompiledInstruction = {
        type: 'frame-switch',
        node_id: 'test-node',
        index: 0,
        params: {
          action: 'exit',
        },
      };

      const context: HandlerContext = {
        page: mockPage,
        context: {} as any,
        config,
        logger,
        metrics,
        sessionId: 'test-session-exit-main',
        frameStack: [], // Empty = at main frame
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('NOT_IN_FRAME');
      expect(result.error?.retryable).toBe(false);
    });

    it('should successfully exit frame when in an iframe', async () => {
      const frame = createMockFrame();
      frame.url.mockReturnValue('https://example.com/iframe');

      const instruction: CompiledInstruction = {
        type: 'frame-switch',
        node_id: 'test-node',
        index: 0,
        params: {
          action: 'exit',
        },
      };

      const context: HandlerContext = {
        page: mockPage,
        context: {} as any,
        config,
        logger,
        metrics,
        sessionId: 'test-session-exit',
        frameStack: [frame],
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const data = result.extracted_data as Record<string, unknown>;
      expect(data?.stackDepth).toBe(0);
    });

    it('should handle multiple consecutive exit calls gracefully', async () => {
      const frame = createMockFrame();
      frame.url.mockReturnValue('https://example.com/iframe');

      const instruction: CompiledInstruction = {
        type: 'frame-switch',
        node_id: 'test-node',
        index: 0,
        params: {
          action: 'exit',
        },
      };

      const frameStack = [frame];
      const context: HandlerContext = {
        page: mockPage,
        context: {} as any,
        config,
        logger,
        metrics,
        sessionId: 'test-session-exit-multiple',
        frameStack,
      };

      // First exit should succeed
      const result1 = await handler.execute(instruction, context);
      expect(result1.success).toBe(true);

      // Second exit should fail (already at main frame)
      const result2 = await handler.execute(instruction, context);
      expect(result2.success).toBe(false);
      expect(result2.error?.code).toBe('NOT_IN_FRAME');
    });
  });

  describe('stale frame reference handling', () => {
    it('should detect and remove stale frame references', async () => {
      // Create a mock frame that will throw when url() is called (stale)
      const staleFrame = {
        url: jest.fn().mockImplementation(() => {
          throw new Error('Frame detached');
        }),
        name: jest.fn().mockResolvedValue('stale-frame'),
      } as any;

      const validFrame = createMockFrame();
      validFrame.url.mockReturnValue('https://example.com/iframe');

      // Setup page with a valid frame to enter
      const targetFrame = createMockFrame();
      targetFrame.url.mockReturnValue('https://example.com/new-iframe');
      (targetFrame.name as jest.Mock).mockReturnValue('new-iframe');
      mockPage.frames.mockReturnValue([targetFrame]);
      mockPage.mainFrame.mockReturnValue(createMockFrame());

      const instruction: CompiledInstruction = {
        type: 'frame-switch',
        node_id: 'test-node',
        index: 0,
        params: {
          action: 'enter',
          frameUrl: 'https://example.com/new-iframe',
        },
      };

      // Start with a stale frame in the stack
      const frameStack = [staleFrame, validFrame];
      const context: HandlerContext = {
        page: mockPage,
        context: {} as any,
        config,
        logger,
        metrics,
        sessionId: 'test-session-stale',
        frameStack,
      };

      const result = await handler.execute(instruction, context);

      // Operation should succeed
      expect(result.success).toBe(true);

      // Stale frame should have been removed from stack
      // Stack should now contain: validFrame + mainFrame (from entering new iframe)
      expect(frameStack.length).toBe(2);
    });
  });

  describe('frame-switch parent idempotency', () => {
    it('should return error when already at main frame (same as exit)', async () => {
      const instruction: CompiledInstruction = {
        type: 'frame-switch',
        node_id: 'test-node',
        index: 0,
        params: {
          action: 'parent',
        },
      };

      const context: HandlerContext = {
        page: mockPage,
        context: {} as any,
        config,
        logger,
        metrics,
        sessionId: 'test-session-parent-main',
        frameStack: [],
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('NOT_IN_FRAME');
    });

    it('should successfully go to parent frame', async () => {
      const parentFrame = createMockFrame();
      parentFrame.url.mockReturnValue('https://example.com/parent');

      const childFrame = createMockFrame();
      childFrame.url.mockReturnValue('https://example.com/child');

      const instruction: CompiledInstruction = {
        type: 'frame-switch',
        node_id: 'test-node',
        index: 0,
        params: {
          action: 'parent',
        },
      };

      const context: HandlerContext = {
        page: mockPage,
        context: {} as any,
        config,
        logger,
        metrics,
        sessionId: 'test-session-parent',
        frameStack: [parentFrame, childFrame],
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const data = result.extracted_data as Record<string, unknown>;
      expect(data?.stackDepth).toBe(1);
    });
  });

  describe('invalid action handling', () => {
    it('should return error for unknown action', async () => {
      const instruction: CompiledInstruction = {
        type: 'frame-switch',
        node_id: 'test-node',
        index: 0,
        params: {
          action: 'invalid-action',
        },
      };

      const context: HandlerContext = {
        page: mockPage,
        context: {} as any,
        config,
        logger,
        metrics,
        sessionId: 'test-session-invalid',
        frameStack: [],
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      // Either INVALID_ACTION or INVALID_INSTRUCTION depending on validation order
      expect(['INVALID_ACTION', 'INVALID_INSTRUCTION']).toContain(result.error?.code);
      expect(result.error?.retryable).toBe(false);
    });

    it('should return error when enter is called without target', async () => {
      const instruction: CompiledInstruction = {
        type: 'frame-switch',
        node_id: 'test-node',
        index: 0,
        params: {
          action: 'enter',
          // No selector, frameId, or frameUrl
        },
      };

      const context: HandlerContext = {
        page: mockPage,
        context: {} as any,
        config,
        logger,
        metrics,
        sessionId: 'test-session-no-target',
        frameStack: [],
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('MISSING_PARAM');
    });
  });
});
