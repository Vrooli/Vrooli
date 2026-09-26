import { BrowserPool } from '../../../src/session/browser-pool';
import type { Browser } from 'rebrowser-playwright';

const browser = (connected = true): Browser =>
  ({ isConnected: jest.fn(() => connected) }) as unknown as Browser;

interface Deferred<T> {
  promise: Promise<T>;
  resolve: (value: T) => void;
  reject: (reason: Error) => void;
}

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

describe('BrowserPool', () => {
  it('shares one concurrent launch for a pool key', async () => {
    const pool = new BrowserPool();
    const launched = browser();
    const launch = jest.fn().mockResolvedValue(launched);

    const [first, second] = await Promise.all([
      pool.getOrLaunch('synthetic\u0000fixture.wav', launch),
      pool.getOrLaunch('synthetic\u0000fixture.wav', launch),
    ]);

    expect(first).toBe(launched);
    expect(second).toBe(launched);
    expect(launch).toHaveBeenCalledTimes(1);
  });

  it('does not reuse a disconnected browser', async () => {
    const pool = new BrowserPool();
    const stale = browser(false);
    const fresh = browser();
    await pool.getOrLaunch('key', jest.fn().mockResolvedValue(stale));

    await expect(pool.getOrLaunch('key', jest.fn().mockResolvedValue(fresh))).resolves.toBe(fresh);
  });

  it('shares one replacement attempt across failed launch waiters', async () => {
    const pool = new BrowserPool();
    const failedLaunch = deferred<Browser>();
    const recovered = browser();
    const launch = jest
      .fn<Promise<Browser>, []>()
      .mockImplementationOnce(() => failedLaunch.promise)
      .mockResolvedValue(recovered);

    const first = pool.getOrLaunch('key', launch);
    const second = pool.getOrLaunch('key', launch);
    const failure = new Error('synthetic launch failure');
    failedLaunch.reject(failure);

    const [firstResult, secondResult] = await Promise.all([first, second]);
    expect(firstResult).toBe(recovered);
    expect(secondResult).toBe(recovered);
    expect(launch).toHaveBeenCalledTimes(2);

    await expect(pool.getOrLaunch('key', launch)).resolves.toBe(recovered);
    expect(launch).toHaveBeenCalledTimes(2);
  });

  it('waits for a pending launch and closes it before shutdown completes', async () => {
    const pool = new BrowserPool();
    const pendingLaunch = deferred<Browser>();
    const launched = browser();
    const close = jest.fn().mockResolvedValue(undefined);
    const request = pool.getOrLaunch('key', () => pendingLaunch.promise);
    let shutdownCompleted = false;

    const shutdown = pool.closeAll(close).then(() => {
      shutdownCompleted = true;
    });
    await Promise.resolve();
    expect(shutdownCompleted).toBe(false);

    pendingLaunch.resolve(launched);
    await expect(request).rejects.toThrow('shut down during launch');
    await shutdown;

    expect(close).toHaveBeenCalledTimes(1);
    expect(close).toHaveBeenCalledWith('key', launched);
    expect(pool.get('key')).toBeUndefined();

    const afterShutdown = browser();
    await expect(pool.getOrLaunch('key', () => Promise.resolve(afterShutdown))).resolves.toBe(
      afterShutdown
    );
  });
});
