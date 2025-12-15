/**
 * Screencast Streaming Tests
 *
 * Unit tests for the CDP screencast streaming module.
 * Uses mocks for CDP session and Page since these require a real browser.
 */

import type { CDPSession, Page } from 'playwright';
import {
  startScreencastStreaming,
  type ScreencastConfig,
  type ScreencastState,
} from '../../../src/routes/record-mode/screencast-streaming';

describe('Screencast Streaming', () => {
  let mockCdpSession: jest.Mocked<CDPSession>;
  let mockPage: jest.Mocked<Page>;
  let mockContext: { newCDPSession: jest.Mock };
  let frameEventHandler: ((event: unknown) => Promise<void>) | null = null;
  let visibilityEventHandler: ((event: unknown) => void) | null = null;
  let disconnectedHandler: (() => void) | null = null;
  let pageCloseHandler: (() => void) | null = null;

  const defaultConfig: ScreencastConfig = {
    format: 'jpeg',
    quality: 65,
    maxWidth: 1280,
    maxHeight: 720,
    everyNthFrame: 1,
  };

  const createMockState = (): ScreencastState => ({
    ws: {
      readyState: 1, // OPEN
      send: jest.fn(),
    },
    wsReady: true,
    frameCount: 0,
    isStreaming: true,
  });

  beforeEach(() => {
    // Reset handlers
    frameEventHandler = null;
    visibilityEventHandler = null;
    disconnectedHandler = null;
    pageCloseHandler = null;

    // Create mock CDP session
    mockCdpSession = {
      send: jest.fn().mockResolvedValue(undefined),
      on: jest.fn((event: string, handler: (...args: unknown[]) => void) => {
        if (event === 'Page.screencastFrame') {
          frameEventHandler = handler as (event: unknown) => Promise<void>;
        } else if (event === 'Page.screencastVisibilityChanged') {
          visibilityEventHandler = handler as (event: unknown) => void;
        } else if (event === 'disconnected') {
          disconnectedHandler = handler as () => void;
        }
      }),
      off: jest.fn(),
      detach: jest.fn().mockResolvedValue(undefined),
    } as unknown as jest.Mocked<CDPSession>;

    // Create mock page context
    mockContext = {
      newCDPSession: jest.fn().mockResolvedValue(mockCdpSession),
    };

    // Create mock page
    mockPage = {
      context: jest.fn().mockReturnValue(mockContext),
      on: jest.fn((event: string, handler: () => void) => {
        if (event === 'close') {
          pageCloseHandler = handler;
        }
      }),
      off: jest.fn(),
    } as unknown as jest.Mocked<Page>;
  });

  describe('startScreencastStreaming', () => {
    it('should start screencast with correct config', async () => {
      const state = createMockState();

      await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      expect(mockCdpSession.send).toHaveBeenCalledWith('Page.startScreencast', defaultConfig);
    });

    it('should register event handlers', async () => {
      const state = createMockState();

      await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      // Should register frame and visibility handlers
      expect(mockCdpSession.on).toHaveBeenCalledWith('Page.screencastFrame', expect.any(Function));
      expect(mockCdpSession.on).toHaveBeenCalledWith(
        'Page.screencastVisibilityChanged',
        expect.any(Function)
      );
      expect(mockPage.on).toHaveBeenCalledWith('close', expect.any(Function));
    });

    it('should return a handle with control methods', async () => {
      const state = createMockState();

      const handle = await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      expect(handle).toHaveProperty('cdpSession');
      expect(handle).toHaveProperty('stop');
      expect(handle).toHaveProperty('getFrameCount');
      expect(handle).toHaveProperty('isActive');
      expect(typeof handle.stop).toBe('function');
      expect(typeof handle.getFrameCount).toBe('function');
      expect(typeof handle.isActive).toBe('function');
    });

    it('should start with isActive true and zero frame count', async () => {
      const state = createMockState();

      const handle = await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      expect(handle.isActive()).toBe(true);
      expect(handle.getFrameCount()).toBe(0);
    });
  });

  describe('frame handling', () => {
    it('should decode base64 frame data and send to WebSocket', async () => {
      const state = createMockState();
      const mockWsSend = state.ws!.send as jest.Mock;

      await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      // Simulate receiving a frame
      const frameData = Buffer.from('test-image-data').toString('base64');
      await frameEventHandler!({
        data: frameData,
        metadata: {
          offsetTop: 0,
          pageScaleFactor: 1,
          deviceWidth: 1280,
          deviceHeight: 720,
          scrollOffsetX: 0,
          scrollOffsetY: 0,
        },
        sessionId: 1,
      });

      // Should send decoded buffer to WebSocket
      expect(mockWsSend).toHaveBeenCalledWith(Buffer.from('test-image-data'));
    });

    it('should acknowledge each frame', async () => {
      const state = createMockState();

      await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      // Simulate receiving a frame
      const frameData = Buffer.from('test-image-data').toString('base64');
      await frameEventHandler!({
        data: frameData,
        metadata: {
          offsetTop: 0,
          pageScaleFactor: 1,
          deviceWidth: 1280,
          deviceHeight: 720,
          scrollOffsetX: 0,
          scrollOffsetY: 0,
        },
        sessionId: 42,
      });

      // Should send ACK with the frame's sessionId
      expect(mockCdpSession.send).toHaveBeenCalledWith('Page.screencastFrameAck', {
        sessionId: 42,
      });
    });

    it('should increment frame count for each frame sent', async () => {
      const state = createMockState();

      const handle = await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      // Simulate multiple frames
      for (let i = 0; i < 5; i++) {
        await frameEventHandler!({
          data: Buffer.from('data').toString('base64'),
          metadata: {
            offsetTop: 0,
            pageScaleFactor: 1,
            deviceWidth: 1280,
            deviceHeight: 720,
            scrollOffsetX: 0,
            scrollOffsetY: 0,
          },
          sessionId: i,
        });
      }

      expect(handle.getFrameCount()).toBe(5);
      expect(state.frameCount).toBe(5);
    });

    it('should not send frames when WebSocket is not ready', async () => {
      const state = createMockState();
      state.wsReady = false;
      const mockWsSend = state.ws!.send as jest.Mock;

      await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      await frameEventHandler!({
        data: Buffer.from('test').toString('base64'),
        metadata: {
          offsetTop: 0,
          pageScaleFactor: 1,
          deviceWidth: 1280,
          deviceHeight: 720,
          scrollOffsetX: 0,
          scrollOffsetY: 0,
        },
        sessionId: 1,
      });

      // Should not send to WebSocket
      expect(mockWsSend).not.toHaveBeenCalled();

      // Should still acknowledge the frame
      expect(mockCdpSession.send).toHaveBeenCalledWith('Page.screencastFrameAck', {
        sessionId: 1,
      });
    });

    it('should not send frames when streaming is stopped', async () => {
      const state = createMockState();
      state.isStreaming = false;
      const mockWsSend = state.ws!.send as jest.Mock;

      await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      await frameEventHandler!({
        data: Buffer.from('test').toString('base64'),
        metadata: {
          offsetTop: 0,
          pageScaleFactor: 1,
          deviceWidth: 1280,
          deviceHeight: 720,
          scrollOffsetX: 0,
          scrollOffsetY: 0,
        },
        sessionId: 1,
      });

      expect(mockWsSend).not.toHaveBeenCalled();
    });
  });

  describe('stop', () => {
    it('should stop screencast and detach CDP session', async () => {
      const state = createMockState();

      const handle = await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      await handle.stop();

      expect(mockCdpSession.send).toHaveBeenCalledWith('Page.stopScreencast');
      expect(mockCdpSession.detach).toHaveBeenCalled();
    });

    it('should remove page close listener on stop', async () => {
      const state = createMockState();

      const handle = await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      await handle.stop();

      expect(mockPage.off).toHaveBeenCalledWith('close', expect.any(Function));
    });

    it('should set isActive to false after stop', async () => {
      const state = createMockState();

      const handle = await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      expect(handle.isActive()).toBe(true);

      await handle.stop();

      expect(handle.isActive()).toBe(false);
    });

    it('should be idempotent - calling stop twice should not throw', async () => {
      const state = createMockState();

      const handle = await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      await handle.stop();
      await handle.stop(); // Second call should be a no-op

      // stopScreencast should only be called once
      expect(
        mockCdpSession.send.mock.calls.filter(
          (call) => call[0] === 'Page.stopScreencast'
        ).length
      ).toBe(1);
    });

    it('should handle stopScreencast failure gracefully', async () => {
      const state = createMockState();
      mockCdpSession.send.mockImplementation(async (method: string) => {
        if (method === 'Page.stopScreencast') {
          throw new Error('Page already closed');
        }
        return {} as never;
      });

      const handle = await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      // Should not throw
      await expect(handle.stop()).resolves.toBeUndefined();
    });
  });

  describe('page close handling', () => {
    it('should set isActive to false when page closes', async () => {
      const state = createMockState();

      const handle = await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      expect(handle.isActive()).toBe(true);

      // Simulate page close
      pageCloseHandler!();

      expect(handle.isActive()).toBe(false);
    });

    it('should not process frames after page closes', async () => {
      const state = createMockState();
      const mockWsSend = state.ws!.send as jest.Mock;

      await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      // Simulate page close
      pageCloseHandler!();

      // Try to receive a frame
      await frameEventHandler!({
        data: Buffer.from('test').toString('base64'),
        metadata: {
          offsetTop: 0,
          pageScaleFactor: 1,
          deviceWidth: 1280,
          deviceHeight: 720,
          scrollOffsetX: 0,
          scrollOffsetY: 0,
        },
        sessionId: 1,
      });

      // Frame should not be sent (isActive is false)
      expect(mockWsSend).not.toHaveBeenCalled();
    });
  });

  describe('CDP disconnect handling', () => {
    it('should set isActive to false when CDP session disconnects', async () => {
      const state = createMockState();

      const handle = await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      expect(handle.isActive()).toBe(true);

      // Simulate CDP disconnect
      disconnectedHandler!();

      expect(handle.isActive()).toBe(false);
    });
  });

  describe('visibility changes', () => {
    it('should handle visibility change events without error', async () => {
      const state = createMockState();

      await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      // Should not throw
      visibilityEventHandler!({ visible: false });
      visibilityEventHandler!({ visible: true });
    });
  });

  describe('error handling', () => {
    it('should continue receiving frames after ACK failure', async () => {
      const state = createMockState();
      const mockWsSend = state.ws!.send as jest.Mock;

      // Make ACK fail
      let ackCallCount = 0;
      mockCdpSession.send.mockImplementation(async (method: string) => {
        if (method === 'Page.screencastFrameAck') {
          ackCallCount++;
          if (ackCallCount === 1) {
            throw new Error('ACK failed');
          }
        }
        return {} as never;
      });

      await startScreencastStreaming(mockPage, 'test-session', state, defaultConfig);

      // First frame - ACK will fail
      await frameEventHandler!({
        data: Buffer.from('test1').toString('base64'),
        metadata: {
          offsetTop: 0,
          pageScaleFactor: 1,
          deviceWidth: 1280,
          deviceHeight: 720,
          scrollOffsetX: 0,
          scrollOffsetY: 0,
        },
        sessionId: 1,
      });

      // Second frame - ACK should succeed
      await frameEventHandler!({
        data: Buffer.from('test2').toString('base64'),
        metadata: {
          offsetTop: 0,
          pageScaleFactor: 1,
          deviceWidth: 1280,
          deviceHeight: 720,
          scrollOffsetX: 0,
          scrollOffsetY: 0,
        },
        sessionId: 2,
      });

      // Both frames should have been sent to WebSocket
      expect(mockWsSend).toHaveBeenCalledTimes(2);
    });
  });
});
