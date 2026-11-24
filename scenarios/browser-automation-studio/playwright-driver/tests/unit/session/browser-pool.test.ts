import { BrowserPool } from '../../../src/session/browser-pool';
import type { Browser } from 'rebrowser-playwright';

const browser = (connected = true, contextCount = 0): Browser =>
  ({ isConnected: jest.fn(() => connected), contexts: jest.fn(() => Array.from({ length: contextCount }, () => ({}))) }) as unknown as Browser;

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
  it('permits process attribution only for one owned browser and one context', async () => {
    const pool = new BrowserPool();
    const owned = browser(true, 1);
    await pool.getOrLaunch('capture', jest.fn().mockResolvedValue(owned));
    expect(pool.ownsExclusiveContext(owned)).toBe(true);

    const sharedContext = browser(true, 2);
    const otherPool = new BrowserPool();
    await otherPool.getOrLaunch('shared', jest.fn().mockResolvedValue(sharedContext));
    expect(otherPool.ownsExclusiveContext(sharedContext)).toBe(false);

    const second = browser(true, 0);
    await pool.getOrLaunch('other-browser', jest.fn().mockResolvedValue(second));
    expect(pool.ownsExclusiveContext(owned)).toBe(false);
  });

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

  it('recycles only a sufficiently idle browser with no open contexts', async () => {
    const pool = new BrowserPool();
    const launched = browser();
    await pool.getOrLaunch('key', () => Promise.resolve(launched));
    const close = jest.fn().mockResolvedValue(undefined);

    await expect(pool.closeIdle(1_000, close, Date.now() + 999)).resolves.toBe(0);
    await expect(pool.closeIdle(1_000, close, Date.now() + 1_000)).resolves.toBe(1);
    expect(close).toHaveBeenCalledWith('key', launched);
    expect(pool.get('key')).toBeUndefined();
  });

  it('starts the idle grace period when session activity ends', async () => {
    const pool = new BrowserPool();
    const launched = browser();
    await pool.getOrLaunch('key', () => Promise.resolve(launched));
    const idleSince = Date.now() + 10_000;
    pool.markUsed(idleSince);
    const close = jest.fn().mockResolvedValue(undefined);

    await expect(pool.closeIdle(1_000, close, idleSince + 999)).resolves.toBe(0);
    await expect(pool.closeIdle(1_000, close, idleSince + 1_000)).resolves.toBe(1);
    expect(close).toHaveBeenCalledTimes(1);
  });

  it('keeps a long-idle browser while any context is open', async () => {
    const pool = new BrowserPool();
    const launched = browser();
    (launched.contexts as jest.Mock).mockReturnValue([{}]);
    await pool.getOrLaunch('key', () => Promise.resolve(launched));
    const close = jest.fn().mockResolvedValue(undefined);

    await expect(pool.closeIdle(1, close, Date.now() + 10_000)).resolves.toBe(0);
    expect(close).not.toHaveBeenCalled();
    expect(pool.get('key')).toBe(launched);
  });

  it('reports pooled browsers, open contexts, and the oldest empty-pool idle age', async () => {
    const pool = new BrowserPool();
    const withContext = browser();
    const idle = browser();
    (withContext.contexts as jest.Mock).mockReturnValue([{}, {}]);
    await pool.getOrLaunch('active', () => Promise.resolve(withContext));
    await pool.getOrLaunch('idle', () => Promise.resolve(idle));

    expect(pool.usageSnapshot(Date.now() + 3_000)).toEqual({
      browserCount: 2,
      contextCount: 2,
      idleBrowserCount: 1,
      maxIdleSeconds: 3,
    });
  });

  it('makes a concurrent get wait for idle eviction before launching a replacement', async () => {
    const pool = new BrowserPool();
    const oldBrowser = browser();
    const replacement = browser();
    await pool.getOrLaunch('key', () => Promise.resolve(oldBrowser));
    const closeDeferred = deferred<void>();
    const close = jest.fn(() => closeDeferred.promise);
    const eviction = pool.closeIdle(1, close, Date.now() + 10_000);
    const launch = jest.fn().mockResolvedValue(replacement);
    const request = pool.getOrLaunch('key', launch);

    await Promise.resolve();
    expect(launch).not.toHaveBeenCalled();
    closeDeferred.resolve(undefined);
    await expect(eviction).resolves.toBe(1);
    await expect(request).resolves.toBe(replacement);
    expect(launch).toHaveBeenCalledTimes(1);
  });

  it('retains a connected browser when idle close fails', async () => {
    const pool = new BrowserPool();
    const launched = browser();
    await pool.getOrLaunch('key', () => Promise.resolve(launched));
    const closeDeferred = deferred<void>();
    const eviction = pool.closeIdle(1, () => closeDeferred.promise, Date.now() + 10_000);
    const launch = jest.fn().mockResolvedValue(browser());
    const request = pool.getOrLaunch('key', launch);
    closeDeferred.reject(new Error('close failed'));
    await expect(eviction).rejects.toThrow('close failed');
    await expect(request).resolves.toBe(launched);
    expect(launch).not.toHaveBeenCalled();
    expect(pool.get('key')).toBe(launched);
  });
});
