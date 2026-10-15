import { chromium, type Browser } from 'rebrowser-playwright';
import { ownBrowser, withBrowser } from '../integration/support/browser-fixture';

const captureHooks = () => { const hooks: { before?: () => Promise<void>; after?: () => Promise<void> } = {}; return [hooks, { beforeAll: (cb: () => void | Promise<void>) => { hooks.before = cb as () => Promise<void>; }, afterAll: (cb: () => void | Promise<void>) => { hooks.after = cb as () => Promise<void>; } }] as const; };

describe('integration browser owner', () => {
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
