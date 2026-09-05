import { describe, expect, it, vi } from 'vitest';
import { fetchReplayStylePayload, persistReplayStyleConfig } from '../adapters/api';
import { REPLAY_STYLE_DEFAULTS } from '../model';

const getMock = vi.fn();
const putMock = vi.fn();

vi.mock('@/api/replayConfig', () => ({
  replayConfigClient: {
    get: (...args: unknown[]) => getMock(...args),
    put: (...args: unknown[]) => putMock(...args),
    reset: vi.fn(),
  },
}));

describe('replay style api adapter', () => {
  it('fetches payload with style and extra config', async () => {
    getMock.mockResolvedValueOnce({
      config: {
        style: { chromeTheme: 'midnight' },
        extra: { cursorSpeedProfile: 'linear', watermark: { enabled: true } },
      },
    });

    const payload = await fetchReplayStylePayload();

    expect(payload?.style.chromeTheme).toBe('midnight');
    expect(payload?.extra).toMatchObject({
      cursorSpeedProfile: 'linear',
      watermark: { enabled: true },
    });
  });

  it('persists replay style with extra config', async () => {
    putMock.mockResolvedValueOnce({ config: {} });

    const style = { ...REPLAY_STYLE_DEFAULTS, chromeTheme: 'chromium' };
    await persistReplayStyleConfig(style, { cursorSpeedProfile: 'easeInOut' });

    const arg = putMock.mock.calls[0][0] as { config: Record<string, unknown> };
    expect((arg.config.style as Record<string, unknown>).chromeTheme).toBe('chromium');
    expect((arg.config.extra as Record<string, unknown>).cursorSpeedProfile).toBe('easeInOut');
  });
});
