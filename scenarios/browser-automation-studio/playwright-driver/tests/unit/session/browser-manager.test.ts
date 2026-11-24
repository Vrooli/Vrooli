/**
 * Browser Manager Tests
 *
 * Tests the browser lifecycle management including:
 * - Browser launch and verification
 * - Connection monitoring
 * - Concurrent launch protection
 * - Graceful shutdown
 */

// Mock rebrowser-playwright before importing BrowserManager
jest.mock('rebrowser-playwright', () => ({
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
    browserPoolRecycles: {
      inc: jest.fn(),
    },
    browserPoolBrowsers: { set: jest.fn() },
    browserPoolContexts: { set: jest.fn() },
    browserPoolIdleBrowsers: { set: jest.fn() },
    browserPoolMaxIdleSeconds: { set: jest.fn() },
  },
}));

import { chromium } from 'rebrowser-playwright';
import { chmodSync, mkdirSync, mkdtempSync, realpathSync, rmSync, symlinkSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import type { Browser, BrowserContext, Page } from 'rebrowser-playwright';
import { metrics } from '../../../src/utils';
import {
  BrowserManager,
  createBrowserManager,
  getBrowserPoolKey,
  resolveInstalledExecutablePath,
  validateInstalledBrowserVersion,
} from '../../../src/session/browser-manager';
import { getAudioLaunchArgs } from '../../../src/session/audio';
import type { Config } from '../../../src/config';

// Helper to create mock browser
function createMockBrowser(
  options: { connected?: boolean; version?: string } = {}
): jest.Mocked<Browser> {
  const { connected = true, version = '120.0.0.0' } = options;
  return {
    isConnected: jest.fn().mockReturnValue(connected),
    version: jest.fn().mockReturnValue(version),
    contexts: jest.fn().mockReturnValue([]),
    newContext: jest.fn(),
    close: jest.fn().mockResolvedValue(undefined),
  } as unknown as jest.Mocked<Browser>;
}

// Helper to create mock context and page
function createMockContextAndPage(): {
  mockContext: jest.Mocked<BrowserContext>;
  mockPage: jest.Mocked<Page>;
} {
  const mockPage = {
    goto: jest.fn().mockResolvedValue(undefined),
    close: jest.fn().mockResolvedValue(undefined),
  } as unknown as jest.Mocked<Page>;
  const mockContext = {
    newPage: jest.fn().mockResolvedValue(mockPage),
    close: jest.fn().mockResolvedValue(undefined),
  } as unknown as jest.Mocked<BrowserContext>;
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
    (chromium.launch as jest.MockedFunction<typeof chromium.launch>).mockResolvedValue(mockBrowser);
    manager = new BrowserManager(testConfig);
  });

  describe('initial state', () => {
    it('is not connected', () => {
      expect(manager.isConnected()).toBe(false);
    });

    it('returns not verified status', () => {
      const status = manager.getBrowserStatus();
      expect(status.healthy).toBe(false);
      expect(status.runtime_identity_status).toBe('unavailable');
      expect(status.error).toBe('Browser not yet verified');
    });
  });

  describe('verifyBrowserLaunch', () => {
    it('returns null on successful verification', async () => {
      const { mockContext, mockPage } = createMockContextAndPage();
      mockBrowser.newContext.mockResolvedValue(mockContext);

      const error = await manager.verifyBrowserLaunch();

      expect(error).toBeNull();
      const launchCalls = (chromium.launch as jest.MockedFunction<typeof chromium.launch>).mock
        .calls;
      expect(launchCalls[0]?.[0]).toEqual({
        headless: true,
        executablePath: undefined,
        args: [],
      });
      expect(mockBrowser.newContext.mock.calls.length).toBeGreaterThan(0);
      expect(mockContext.newPage.mock.calls.length).toBeGreaterThan(0);
      const gotoCalls = mockPage.goto.mock.calls as Array<[string]>;
      expect(gotoCalls[0]?.[0]).toBe('about:blank');
      expect(mockPage.close.mock.calls.length).toBeGreaterThan(0);
      expect(mockContext.close.mock.calls.length).toBeGreaterThan(0);
    });

    it('returns error message on launch failure', async () => {
      (chromium.launch as jest.MockedFunction<typeof chromium.launch>).mockRejectedValue(
        new Error('Chromium not found')
      );

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
      expect(mockBrowser.newContext.mock.calls.length).toBe(1);
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
        runtime_identity_status: 'unavailable',
        error: 'Browser not yet verified',
      });
    });

    it('returns error after failed verification', async () => {
      (chromium.launch as jest.Mock).mockRejectedValue(new Error('Launch failed'));
      await manager.verifyBrowserLaunch();

      const status = manager.getBrowserStatus();
      expect(status).toEqual({
        healthy: false,
        runtime_identity_status: 'unavailable',
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
        runtime_identity_status: 'reported',
        family: 'chromium',
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
      expect(status).toEqual({ healthy: true, runtime_identity_status: 'unavailable' });
    });
  });

  describe('getBrowser', () => {
    it('launches browser on first call', async () => {
      const browser = await manager.getBrowser();

      expect(browser).toBe(mockBrowser);
      expect((chromium.launch as jest.Mock).mock.calls.length).toBe(1);
    });

    it('publishes pool occupancy and counts only successful idle recycles', async () => {
      await manager.getBrowser();
      expect(metrics.browserPoolBrowsers.set).toHaveBeenCalledWith(1);
      expect(metrics.browserPoolContexts.set).toHaveBeenCalledWith(0);
      expect(metrics.browserPoolIdleBrowsers.set).toHaveBeenCalledWith(1);

      await expect(manager.recycleIdleBrowsers(0)).resolves.toBe(1);
      expect(metrics.browserPoolRecycles.inc).toHaveBeenCalledTimes(1);
      expect(metrics.browserPoolBrowsers.set).toHaveBeenLastCalledWith(0);
    });

    it('returns cached browser on subsequent calls', async () => {
      const browser1 = await manager.getBrowser();
      const browser2 = await manager.getBrowser();

      expect(browser1).toBe(browser2);
      expect((chromium.launch as jest.Mock).mock.calls.length).toBe(1);
    });

    it('relaunches if browser disconnected', async () => {
      await manager.getBrowser();

      // Simulate browser disconnect
      mockBrowser.isConnected.mockReturnValue(false);

      const newMockBrowser = createMockBrowser();
      (chromium.launch as jest.Mock).mockResolvedValue(newMockBrowser);

      const browser = await manager.getBrowser();

      expect(browser).toBe(newMockBrowser);
      expect((chromium.launch as jest.Mock).mock.calls.length).toBe(2);
    });

    it('handles concurrent launch requests', async () => {
      // Slow down launch to simulate race condition
      let resolveFirst!: (value: Browser | PromiseLike<Browser>) => void;
      const slowLaunch: Promise<Browser> = new Promise((resolve) => {
        resolveFirst = resolve;
      });
      (chromium.launch as jest.MockedFunction<typeof chromium.launch>).mockReturnValue(slowLaunch);

      // Start two concurrent getBrowser calls
      const promise1 = manager.getBrowser();
      const promise2 = manager.getBrowser();

      // Resolve the slow launch
      resolveFirst(mockBrowser);

      const [browser1, browser2] = await Promise.all([promise1, promise2]);

      // Both should get the same browser
      expect(browser1).toBe(browser2);
      // Launch should only be called once
      expect(
        (chromium.launch as jest.MockedFunction<typeof chromium.launch>).mock.calls.length
      ).toBe(1);
    });

    it('coalesces one retry across callers when a launch fails', async () => {
      // First launch fails
      let rejectFirst!: (reason: Error) => void;
      const failingLaunch: Promise<Browser> = new Promise((_, reject) => {
        rejectFirst = reject;
      });
      (chromium.launch as jest.MockedFunction<typeof chromium.launch>).mockReturnValueOnce(
        failingLaunch
      );

      // Second launch succeeds
      const newMockBrowser = createMockBrowser();
      (chromium.launch as jest.MockedFunction<typeof chromium.launch>).mockResolvedValueOnce(
        newMockBrowser
      );

      // Start a concurrent launch wave that shares the initial failure and one retry.
      const promise1 = manager.getBrowser();
      const promise2 = manager.getBrowser();
      rejectFirst(new Error('First launch failed'));

      const [browser1, browser2] = await Promise.all([promise1, promise2]);
      expect(browser1).toBe(newMockBrowser);
      expect(browser2).toBe(newMockBrowser);
      expect(
        (chromium.launch as jest.MockedFunction<typeof chromium.launch>).mock.calls.length
      ).toBe(2);
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

      const launchCalls = (chromium.launch as jest.MockedFunction<typeof chromium.launch>).mock
        .calls;
      expect(launchCalls[0]?.[0]).toEqual({
        headless: false,
        executablePath: '/custom/chromium',
        args: ['--no-sandbox', '--disable-gpu'],
      });
    });

    it('enables Chromium real fake-media capture only for an explicit fixture', async () => {
      const fixtureConfig = {
        browser: {
          headless: false,
          executablePath: undefined,
          args: ['--headless=new'],
          ignoreHTTPSErrors: false,
          fakeMicrophoneFile: '/fixtures/speech.wav',
        },
      } as unknown as Config;
      const fixtureManager = new BrowserManager(fixtureConfig);

      await fixtureManager.getBrowser();

      expect(
        (chromium.launch as jest.MockedFunction<typeof chromium.launch>).mock.calls[0]?.[0]
      ).toEqual({
        headless: false,
        executablePath: undefined,
        args: [
          '--headless=new',
          '--use-fake-device-for-media-stream',
          '--use-fake-ui-for-media-stream',
          '--use-file-for-fake-audio-capture=/fixtures/speech.wav',
        ],
      });
    });
  });

  describe('installed executable selection', () => {
    let root: string;
    let executable: string;

    beforeEach(() => {
      root = mkdtempSync(join(tmpdir(), 'bas-installed-browser-'));
      executable = join(root, 'chrome');
      writeFileSync(executable, 'test fixture; never executed');
      chmodSync(executable, 0o755);
    });

    afterEach(() => rmSync(root, { recursive: true, force: true }));

    it('requires an explicitly configured absolute executable and rejects broken paths', () => {
      expect(() => resolveInstalledExecutablePath(undefined)).toThrow('absolute BROWSER_EXECUTABLE_PATH');
      expect(() => resolveInstalledExecutablePath('relative/chrome')).toThrow('absolute BROWSER_EXECUTABLE_PATH');
      expect(() => resolveInstalledExecutablePath(join(root, 'missing'))).toThrow('missing or not executable');
      const directory = join(root, 'directory');
      mkdirSync(directory);
      expect(() => resolveInstalledExecutablePath(directory)).toThrow('missing or not executable');
      const noExecute = join(root, 'no-execute');
      writeFileSync(noExecute, 'fixture');
      chmodSync(noExecute, 0o644);
      expect(() => resolveInstalledExecutablePath(noExecute)).toThrow('missing or not executable');
      const brokenLink = join(root, 'broken-link');
      symlinkSync(join(root, 'absent-target'), brokenLink);
      expect(() => resolveInstalledExecutablePath(brokenLink)).toThrow('missing or not executable');
      expect(chromium.launch).not.toHaveBeenCalled();
    });

    it('uses only the configured executable and keeps installed and managed processes isolated', async () => {
      const configured = { browser: { ...testConfig.browser, executablePath: executable } } as Config;
      const configuredManager = new BrowserManager(configured);
      await configuredManager.getBrowser(undefined, 'host_device', false, '', 'installed');
      await configuredManager.getBrowser(undefined, 'host_device', false, '', 'managed');
      const launchCalls = (chromium.launch as jest.MockedFunction<typeof chromium.launch>).mock.calls;
      expect(launchCalls).toHaveLength(2);
      expect(launchCalls[0]?.[0]?.executablePath).toBe(realpathSync(executable));
      expect(launchCalls[1]?.[0]?.executablePath).toBe(executable);
      expect(getBrowserPoolKey('', 'host_device', false, '', 'installed', executable)).not.toBe(
        getBrowserPoolKey('', 'host_device', false, '', 'managed', executable)
      );
    });

    it('validates the installed browser reported version before returning it', async () => {
      const invalidBrowser = createMockBrowser({ version: 'unknown' });
      (chromium.launch as jest.MockedFunction<typeof chromium.launch>).mockResolvedValueOnce(invalidBrowser);
      const configuredManager = new BrowserManager({ browser: { ...testConfig.browser, executablePath: executable } } as Config);
      await expect(configuredManager.getBrowser(undefined, 'host_device', false, '', 'installed')).rejects.toThrow('valid Chromium version');
      expect(chromium.launch).toHaveBeenCalledTimes(1);
      expect(invalidBrowser.close).toHaveBeenCalledTimes(1);
      expect(() => validateInstalledBrowserVersion('140.0.7339.80')).not.toThrow();
      expect(() => validateInstalledBrowserVersion('not-a-version')).toThrow('valid Chromium version');
    });
  });

  describe('shutdown', () => {
    it('closes the browser', async () => {
      await manager.getBrowser();
      await manager.shutdown();

      expect(mockBrowser.close.mock.calls.length).toBeGreaterThan(0);
    });

    it('handles close errors gracefully', async () => {
      mockBrowser.close.mockRejectedValue(new Error('Close failed'));
      await manager.getBrowser();

      // Should not throw
      await expect(manager.shutdown()).resolves.toBeUndefined();
    });

    it('does nothing if browser not launched', async () => {
      await manager.shutdown();

      expect(mockBrowser.close.mock.calls.length).toBe(0);
    });

    it('allows new browser launch after shutdown', async () => {
      await manager.getBrowser();
      await manager.shutdown();

      const newMockBrowser = createMockBrowser();
      (chromium.launch as jest.Mock).mockResolvedValue(newMockBrowser);

      const browser = await manager.getBrowser();

      expect(browser).toBe(newMockBrowser);
      expect((chromium.launch as jest.Mock).mock.calls.length).toBe(2);
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

describe('getBrowserPoolKey', () => {
  it('separates audio strategies for one fake microphone WAV', () => {
    expect(getBrowserPoolKey('/tmp/input.wav', 'host_device')).not.toBe(
      getBrowserPoolKey('/tmp/input.wav', 'synthetic_sink')
    );
  });

  it('separates host and synthetic strategies without fake media', () => {
    expect(getBrowserPoolKey('', 'host_device')).not.toBe(getBrowserPoolKey('', 'synthetic_sink'));
  });

  it('isolates extension-debugging Chromium by admitted BAS session', () => {
    const ordinary = getBrowserPoolKey('', 'host_device');
    const extensionA = getBrowserPoolKey('', 'host_device', true, 'session-a');
    const extensionB = getBrowserPoolKey('', 'host_device', true, 'session-b');
    expect(extensionA).not.toBe(ordinary);
    expect(extensionB).not.toBe(ordinary);
    expect(extensionA).not.toBe(extensionB);
  });
});

describe('getAudioLaunchArgs', () => {
  it('enables the muted AudioContext API only for the synthetic pool', () => {
    expect(getAudioLaunchArgs('host_device')).toEqual([]);
    expect(getAudioLaunchArgs('synthetic_sink')).toEqual(
      expect.arrayContaining([
        '--enable-blink-features=AudioContextSetSinkId',
        '--enable-features=AudioContextSetSinkId',
      ])
    );
  });
});

describe('createBrowserManager', () => {
  it('returns a BrowserManager instance', () => {
    const manager = createBrowserManager(testConfig);
    expect(manager).toBeInstanceOf(BrowserManager);
  });
});
