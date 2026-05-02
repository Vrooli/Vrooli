import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { fetchEmptyResponse, fetchJsonResponse, installFetchMock, type FetchMock } from '@/test-utils';
import { fetchReplayStylePayload, persistReplayStyleConfig } from '../adapters/api';
import { REPLAY_STYLE_DEFAULTS } from '../model';

vi.mock('@/config', () => ({
  getConfig: async () => ({ API_URL: 'http://localhost/api' }),
}));

describe('replay style api adapter', () => {
  let fetchMock: FetchMock;

  beforeEach(() => {
    fetchMock = installFetchMock();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('fetches payload with style and extra config', async () => {
    fetchMock.mockResolvedValueOnce(
      fetchJsonResponse({
        config: {
          style: {
            chromeTheme: 'midnight',
          },
          extra: {
            cursorSpeedProfile: 'linear',
            watermark: { enabled: true },
          },
        },
      })
    );

    const payload = await fetchReplayStylePayload();

    expect(payload?.style.chromeTheme).toBe('midnight');
    expect(payload?.extra).toMatchObject({
      cursorSpeedProfile: 'linear',
      watermark: { enabled: true },
    });
  });

  it('persists replay style with extra config', async () => {
    fetchMock.mockResolvedValueOnce(fetchEmptyResponse());

    const style = { ...REPLAY_STYLE_DEFAULTS, chromeTheme: 'chromium' };
    await persistReplayStyleConfig(style, { cursorSpeedProfile: 'easeInOut' });

    const call = fetchMock.mock.calls[0];
    expect(call[0]).toBe('http://localhost/api/replay-config');
    const body = JSON.parse((call[1] as RequestInit).body as string) as {
      config: Record<string, unknown>;
    };
    expect((body.config.style as Record<string, unknown>).chromeTheme).toBe('chromium');
    expect((body.config.extra as Record<string, unknown>).cursorSpeedProfile).toBe('easeInOut');
  });
});
