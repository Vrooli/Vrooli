import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { fetchReplayStylePayload, persistReplayStyleConfig } from '../adapters/api';
import { REPLAY_STYLE_DEFAULTS } from '../model';

vi.mock('@/config', () => ({
  getConfig: async () => ({ API_URL: 'http://localhost/api' }),
}));

describe('replay style api adapter', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('fetches payload with style and extra config', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        config: {
          chromeTheme: 'midnight',
          cursorSpeedProfile: 'linear',
          watermark: { enabled: true },
        },
      }),
    } as Response);

    const payload = await fetchReplayStylePayload();

    expect(payload?.style.chromeTheme).toBe('midnight');
    expect(payload?.extraConfig).toMatchObject({
      cursorSpeedProfile: 'linear',
      watermark: { enabled: true },
    });
    expect(payload?.extraConfig).not.toHaveProperty('chromeTheme');
  });

  it('persists replay style with extra config', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({ ok: true } as Response);

    const style = { ...REPLAY_STYLE_DEFAULTS, chromeTheme: 'chromium' };
    await persistReplayStyleConfig(style, { cursorSpeedProfile: 'easeInOut' });

    const call = vi.mocked(global.fetch).mock.calls[0];
    expect(call[0]).toBe('http://localhost/api/replay-config');
    const body = JSON.parse((call[1] as RequestInit).body as string) as {
      config: Record<string, unknown>;
    };
    expect(body.config.chromeTheme).toBe('chromium');
    expect(body.config.cursorSpeedProfile).toBe('easeInOut');
  });
});
