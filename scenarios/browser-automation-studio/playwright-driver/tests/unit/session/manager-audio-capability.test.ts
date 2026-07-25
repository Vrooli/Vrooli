import { SessionManager } from '../../../src/session/manager';

describe('SessionManager.measureBareRealtimeAudio', () => {
  it('uses a fresh context from the managed browser and closes it after probing', async () => {
    const page = {
      goto: jest.fn().mockResolvedValue(undefined),
      evaluate: jest.fn().mockResolvedValue({
        supported: true,
        currentTimeDelta: 2,
        callbackCount: 10,
        outputLatency: 0.03,
        state: 'running',
      }),
      close: jest.fn().mockResolvedValue(undefined),
    };
    const context = {
      newPage: jest.fn().mockResolvedValue(page),
      close: jest.fn().mockResolvedValue(undefined),
    };
    const browserManager = {
      getBrowser: jest.fn().mockResolvedValue({ newContext: jest.fn().mockResolvedValue(context) }),
    };
    const manager = new SessionManager({} as never, browserManager as never);

    await expect(manager.measureBareRealtimeAudio()).resolves.toMatchObject({ available: true });

    expect(page.goto).toHaveBeenCalledWith('about:blank');
    expect(page.close).toHaveBeenCalledTimes(1);
    expect(context.close).toHaveBeenCalledTimes(1);
  });
});
