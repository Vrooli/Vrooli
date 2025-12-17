/**
 * Browser Manager Tests
 *
 * Tests the browser lifecycle management including:
 * - Browser launch and verification
 * - Connection monitoring
 * - Concurrent launch protection
 * - Graceful shutdown
 */

// Mock playwright before importing BrowserManager
jest.mock('playwright', () => ({
  chromium: {
    launch: jest.fn(),
  },
}));

// Mock utils to avoid proto dependency chain
jest.mock('../../../src/utils', () => ({
  logger: {
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn(),
    debug: jest.fn(),
  },
  metrics: {
    cleanupFailures: {
      inc: jest.fn(),
    },
  },
}));

import { chromium } from 'playwright';
import { BrowserManager, createBrowserManager } from '../../../src/session/browser-manager';
import type { Config } from '../../../src/config';

// Helper to create mock browser
function createMockBrowser(options: { connected?: boolean; version?: string } = {}) {
  const { connected = true, version = '120.0.0.0' } = options;
  return {
    isConnected: jest.fn().mockReturnValue(connected),
    version: jest.fn().mockReturnValue(version),
    newContext: jest.fn(),
    close: jest.fn().mockResolvedValue(undefined),
  };
}

// Helper to create mock context and page
function createMockContextAndPage() {
  const mockPage = {
    goto: jest.fn().mockResolvedValue(undefined),
    close: jest.fn().mockResolvedValue(undefined),
  };
  const mockContext = {
    newPage: jest.fn().mockResolvedValue(mockPage),
    close: jest.fn().mockResolvedValue(undefined),
  };
  return { mockContext, mockPage };
}

// Create a partial config that matches what BrowserManager actually uses
// BrowserManager only accesses config.browser properties
const testConfig = {
  browser: {
    headless: true,
    executablePath: undefined,
    args: [],
    ignoreHTTPSErrors: false,
  },
} as unknown as Config;

describe('BrowserManager', () => {
  let manager: BrowserManager;
  let mockBrowser: ReturnType<typeof createMockBrowser>;

  beforeEach(() => {
    jest.clearAllMocks();
    mockBrowser = createMockBrowser();
    (chromium.launch as jest.Mock).mockResolvedValue(mockBrowser);
    manager = new BrowserManager(testConfig);
  });

  describe('initial state', () => {
    it('is not connected', () => {
      expect(manager.isConnected()).toBe(false);
    });

    it('returns not verified status', () => {
      const status = manager.getBrowserStatus();
      expect(status.healthy).toBe(false);
      expect(status.error).toBe('Browser not yet verified');
    });
  });

  describe('verifyBrowserLaunch', () => {
    it('returns null on successful verification', async () => {
      const { mockContext, mockPage } = createMockContextAndPage();
      mockBrowser.newContext.mockResolvedValue(mockContext);

      const error = await manager.verifyBrowserLaunch();

      expect(error).toBeNull();
      expect(chromium.launch).toHaveBeenCalledWith({
        headless: true,
        executablePath: undefined,
        args: [],
      });
      expect(mockBrowser.newContext).toHaveBeenCalled();
      expect(mockContext.newPage).toHaveBeenCalled();
      expect(mockPage.goto).toHaveBeenCalledWith('about:blank');
      expect(mockPage.close).toHaveBeenCalled();
      expect(mockContext.close).toHaveBeenCalled();
    });

    it('returns error message on launch failure', async () => {
      (chromium.launch as jest.Mock).mockRejectedValue(new Error('Chromium not found'));

      const error = await manager.verifyBrowserLaunch();

      expect(error).toBe('Chromium not found');
    });

    it('returns error message on context creation failure', async () => {
      mockBrowser.newContext.mockRejectedValue(new Error('Context creation failed'));

      const error = await manager.verifyBrowserLaunch();

      expect(error).toBe('Context creation failed');
    });

    it('skips verification on subsequent calls', async () => {
      const { mockContext } = createMockContextAndPage();
      mockBrowser.newContext.mockResolvedValue(mockContext);

      await manager.verifyBrowserLaunch();
      await manager.verifyBrowserLaunch();

      // chromium.launch should only be called once due to caching,
      // and verification should only happen once
      expect(mockBrowser.newContext).toHaveBeenCalledTimes(1);
    });

    it('returns cached error on subsequent calls after failure', async () => {
      (chromium.launch as jest.Mock).mockRejectedValue(new Error('Initial failure'));

      const error1 = await manager.verifyBrowserLaunch();
      const error2 = await manager.verifyBrowserLaunch();

      expect(error1).toBe('Initial failure');
      expect(error2).toBe('Initial failure');
    });
  });

  describe('getBrowserStatus', () => {
    it('returns not verified before verification', () => {
      const status = manager.getBrowserStatus();
      expect(status).toEqual({
        healthy: false,
        error: 'Browser not yet verified',
      });
    });

    it('returns error after failed verification', async () => {
      (chromium.launch as jest.Mock).mockRejectedValue(new Error('Launch failed'));
      await manager.verifyBrowserLaunch();

      const status = manager.getBrowserStatus();
      expect(status).toEqual({
        healthy: false,
        error: 'Launch failed',
      });
    });

    it('returns healthy with version when browser connected', async () => {
      const { mockContext } = createMockContextAndPage();
      mockBrowser.newContext.mockResolvedValue(mockContext);
      await manager.verifyBrowserLaunch();

      const status = manager.getBrowserStatus();
      expect(status).toEqual({
        healthy: true,
        version: '120.0.0.0',
      });
    });

    it('returns healthy without version when verified but not connected', async () => {
      const { mockContext } = createMockContextAndPage();
      mockBrowser.newContext.mockResolvedValue(mockContext);
      await manager.verifyBrowserLaunch();

      // Disconnect browser
      mockBrowser.isConnected.mockReturnValue(false);

      const status = manager.getBrowserStatus();
      expect(status).toEqual({ healthy: true });
    });
  });

  describe('getBrowser', () => {
    it('launches browser on first call', async () => {
      const browser = await manager.getBrowser();

      expect(browser).toBe(mockBrowser);
      expect(chromium.launch).toHaveBeenCalledTimes(1);
    });

    it('returns cached browser on subsequent calls', async () => {
      const browser1 = await manager.getBrowser();
      const browser2 = await manager.getBrowser();

      expect(browser1).toBe(browser2);
      expect(chromium.launch).toHaveBeenCalledTimes(1);
    });

    it('relaunches if browser disconnected', async () => {
      await manager.getBrowser();

      // Simulate browser disconnect
      mockBrowser.isConnected.mockReturnValue(false);

      const newMockBrowser = createMockBrowser();
      (chromium.launch as jest.Mock).mockResolvedValue(newMockBrowser);

      const browser = await manager.getBrowser();

      expect(browser).toBe(newMockBrowser);
      expect(chromium.launch).toHaveBeenCalledTimes(2);
    });

    it('handles concurrent launch requests', async () => {
      // Slow down launch to simulate race condition
      let resolveFirst!: (value: unknown) => void;
      const slowLaunch = new Promise((resolve) => {
        resolveFirst = resolve;
      });
      (chromium.launch as jest.Mock).mockReturnValue(slowLaunch);

      // Start two concurrent getBrowser calls
      const promise1 = manager.getBrowser();
      const promise2 = manager.getBrowser();

      // Resolve the slow launch
      resolveFirst(mockBrowser);

      const [browser1, browser2] = await Promise.all([promise1, promise2]);

      // Both should get the same browser
      expect(browser1).toBe(browser2);
      // Launch should only be called once
      expect(chromium.launch).toHaveBeenCalledTimes(1);
    });

    it('retries if concurrent launch fails', async () => {
      // First launch fails
      let rejectFirst!: (reason: Error) => void;
      const failingLaunch = new Promise((_, reject) => {
        rejectFirst = reject;
      });
      (chromium.launch as jest.Mock).mockReturnValueOnce(failingLaunch);

      // Second launch succeeds
      const newMockBrowser = createMockBrowser();
      (chromium.launch as jest.Mock).mockResolvedValueOnce(newMockBrowser);

      // Start first call that will fail
      const promise1 = manager.getBrowser();

      // Fail the first launch
      rejectFirst(new Error('First launch failed'));

      await expect(promise1).rejects.toThrow('First launch failed');

      // Second call should retry and succeed
      const browser = await manager.getBrowser();
      expect(browser).toBe(newMockBrowser);
    });

    it('passes config options to chromium.launch', async () => {
      const customConfig = {
        browser: {
          headless: false,
          executablePath: '/custom/chromium',
          args: ['--no-sandbox', '--disable-gpu'],
          ignoreHTTPSErrors: false,
        },
      } as unknown as Config;
      const customManager = new BrowserManager(customConfig);

      await customManager.getBrowser();

      expect(chromium.launch).toHaveBeenCalledWith({
        headless: false,
        executablePath: '/custom/chromium',
        args: ['--no-sandbox', '--disable-gpu'],
      });
    });
  });

  describe('shutdown', () => {
    it('closes the browser', async () => {
      await manager.getBrowser();
      await manager.shutdown();

      expect(mockBrowser.close).toHaveBeenCalled();
    });

    it('handles close errors gracefully', async () => {
      mockBrowser.close.mockRejectedValue(new Error('Close failed'));
      await manager.getBrowser();

      // Should not throw
      await expect(manager.shutdown()).resolves.toBeUndefined();
    });

    it('does nothing if browser not launched', async () => {
      await manager.shutdown();

      expect(mockBrowser.close).not.toHaveBeenCalled();
    });

    it('allows new browser launch after shutdown', async () => {
      await manager.getBrowser();
      await manager.shutdown();

      const newMockBrowser = createMockBrowser();
      (chromium.launch as jest.Mock).mockResolvedValue(newMockBrowser);

      const browser = await manager.getBrowser();

      expect(browser).toBe(newMockBrowser);
      expect(chromium.launch).toHaveBeenCalledTimes(2);
    });
  });

  describe('isConnected', () => {
    it('returns false when browser not launched', () => {
      expect(manager.isConnected()).toBe(false);
    });

    it('returns true when browser connected', async () => {
      await manager.getBrowser();
      expect(manager.isConnected()).toBe(true);
    });

    it('returns false when browser disconnected', async () => {
      await manager.getBrowser();
      mockBrowser.isConnected.mockReturnValue(false);
      expect(manager.isConnected()).toBe(false);
    });
  });
});

describe('createBrowserManager', () => {
  it('returns a BrowserManager instance', () => {
    const manager = createBrowserManager(testConfig);
    expect(manager).toBeInstanceOf(BrowserManager);
  });
});
