import type { Browser } from 'rebrowser-playwright';
import { SessionManager } from '../../../src/session/manager';
import { ResourceLimitError } from '../../../src/utils/errors';
import { createTestConfig } from '../../helpers';

describe('public capture session capacity', () => {
  it('reserves one normal BAS slot until the fresh capture callback finishes', async () => {
    const browser = {} as Browser;
    const browserManager = { getBrowser: jest.fn(async () => browser) };
    const manager = new SessionManager(
      createTestConfig({
        session: {
          maxConcurrent: 1,
          idleTimeoutMs: 300_000,
          poolSize: 2,
          cleanupIntervalMs: 60_000,
        },
      }),
      browserManager as never
    );
    let enteredCapture!: () => void;
    let finishCapture!: () => void;
    const entered = new Promise<void>((resolve) => {
      enteredCapture = resolve;
    });
    const captureGate = new Promise<void>((resolve) => {
      finishCapture = resolve;
    });

    const capture = manager.withPublicCaptureBrowser(async (selected) => {
      enteredCapture();
      await captureGate;
      return selected;
    });
    await entered;
    await expect(manager.withPublicCaptureBrowser(async () => 'second')).rejects.toBeInstanceOf(
      ResourceLimitError
    );
    finishCapture();
    await expect(capture).resolves.toBe(browser);
    await expect(manager.withPublicCaptureBrowser(async () => 'after-release')).resolves.toBe(
      'after-release'
    );
    expect(browserManager.getBrowser).toHaveBeenCalledTimes(2);
  });
});
