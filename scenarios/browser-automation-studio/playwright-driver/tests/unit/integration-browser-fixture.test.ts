import { chromium, type Browser } from 'rebrowser-playwright';
import { ownBrowser, withBrowser } from '../integration/support/browser-fixture';

const captureHooks = () => { const hooks: { before?: () => Promise<void>; after?: () => Promise<void> } = {}; return [hooks, { beforeAll: (cb: () => void | Promise<void>) => { hooks.before = cb as () => Promise<void>; }, afterAll: (cb: () => void | Promise<void>) => { hooks.after = cb as () => Promise<void>; } }] as const; };

describe('integration browser owner', () => {
  it('refuses an unmocked native launch in the ordinary unit boundary', async () => {
    expect(chromium.launch).toHaveProperty('unitBrowserBoundary', true);
    await expect(chromium.launch({ headless: true })).rejects.toThrow('is forbidden in unit tests');
  });
  it('keeps native refusal after an explicit fixture spy is restored', async () => {
    const guarded = chromium.launch;
    const browser = { close: jest.fn() } as unknown as Browser;
    const launch = jest.spyOn(chromium, 'launch').mockResolvedValue(browser);
    expect(await chromium.launch()).toBe(browser);
    launch.mockRestore();
    expect(chromium.launch).toBe(guarded);
    expect(chromium.launch).toHaveProperty('unitBrowserBoundary', true);
    await expect(chromium.launch()).rejects.toThrow('is forbidden in unit tests');
  });
  it('uses launch options and closes the suite browser', async () => {
    const [hooks, registered] = captureHooks();
    const browser = { close: jest.fn() } as unknown as Browser;
    const launch = jest.spyOn(chromium, 'launch').mockResolvedValue(browser);
    const owner = ownBrowser({ headless: true }, false, registered);
    await hooks.before?.();
    expect(launch).toHaveBeenCalledWith({ headless: true });
    expect(owner.browser).toBe(browser);
    await hooks.after?.();
    expect(browser.close).toHaveBeenCalledTimes(1);
    launch.mockRestore();
  });
  it('tolerates optional startup failure and propagates cleanup failure', async () => {
    const [hooks, registered] = captureHooks();
    const owner = ownBrowser({ headless: true }, true, registered, jest.fn().mockRejectedValue(new Error('unavailable')));
    await expect(hooks.before?.()).resolves.toBeUndefined();
    expect(owner.browser).toBeUndefined();
    await hooks.after?.();
    const browser = { close: jest.fn().mockRejectedValue(new Error('close failed')) } as unknown as Browser;
    await expect(withBrowser({ headless: true }, async () => 'done', jest.fn().mockResolvedValue(browser))).rejects.toThrow('close failed');
  });
});


describe('explicit native progress reporting', () => {
  it('names the active suite/case and stops activity after completion', () => {
    const NativeProgress = require('../setup/native-progress.cjs');
    jest.useFakeTimers();
    const write = jest.fn();
    const reporter = new NativeProgress({}, { write });
    try {
      jest.advanceTimersByTime(30000);
      expect(write).not.toHaveBeenCalled();
      reporter.onTestStart({ path: '/fixture/local-native.test.ts' });
      jest.advanceTimersByTime(15000);
      expect(write).toHaveBeenCalledWith(expect.stringContaining('active /fixture/local-native.test.ts elapsed=15s'));
      reporter.onTestCaseResult({}, { status: 'passed', fullName: 'native postcondition' });
      expect(write).toHaveBeenCalledWith(expect.stringContaining('case passed native postcondition'));
      reporter.onTestResult();
      const calls = write.mock.calls.length;
      jest.advanceTimersByTime(60000);
      expect(write).toHaveBeenCalledTimes(calls);
      reporter.onRunComplete();
    } finally {
      reporter.onRunComplete();
      jest.useRealTimers();
    }
  });
});
