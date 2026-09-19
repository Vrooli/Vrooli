import { BrowserPool } from '../../../src/session/browser-pool';

const browser = (connected = true) => ({ isConnected: jest.fn(() => connected) }) as never;

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
});
