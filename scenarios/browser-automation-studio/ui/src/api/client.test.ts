import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { API_BASE, authenticatedFetch } from './client';
import { useAuthStore } from '../stores/authStore';

describe('authenticatedFetch', () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    fetchMock.mockReset().mockResolvedValue(new Response('{}', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);
    sessionStorage.clear();
    window.desktop = {
      auth: {
        getAccessToken: vi.fn().mockResolvedValue('native-desktop-token'),
      },
    } as unknown as Window['desktop'];
  });

  afterEach(() => {
    delete window.desktop;
    vi.unstubAllGlobals();
  });

  it('uses the native Electron token for same-origin API requests and preserves consent headers', async () => {
    sessionStorage.setItem('bas.ai.remote-consent-receipt', 'opaque-receipt');
    sessionStorage.setItem('bas.ai.remote-consent-source', 'own-key');

    await authenticatedFetch(`${API_BASE}/browser_automation_studio.v1.ai.AIService/AIAnalyzeElements`);

    const init = fetchMock.mock.calls[0][1];
    const headers = new Headers(init?.headers);
    expect(headers.get('Authorization')).toBe('Bearer native-desktop-token');
    expect(headers.get('X-Vrooli-AI-Consent-Grant')).toBe('opaque-receipt');
    expect(headers.get('X-Vrooli-AI-Consent-Source')).toBe('own-key');
    expect(window.desktop?.auth.getAccessToken).toHaveBeenCalledOnce();
  });

  it('does not forward the native token or consent receipt to another origin', async () => {
    sessionStorage.setItem('bas.ai.remote-consent-receipt', 'opaque-receipt');
    sessionStorage.setItem('bas.ai.remote-consent-source', 'own-key');

    await authenticatedFetch('https://provider.invalid/v1/run');

    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers);
    expect(headers.has('Authorization')).toBe(false);
    expect(headers.has('X-Vrooli-AI-Consent-Grant')).toBe(false);
    expect(headers.has('X-Vrooli-AI-Consent-Source')).toBe(false);
  });

  it('keeps the browser token fallback when the desktop auth bridge is absent', async () => {
    delete window.desktop;
    sessionStorage.setItem('vrooli.web.access-token', JSON.stringify({
      accessToken: 'browser-session-token',
      expiresAt: new Date(Date.now() + 60_000).toISOString(),
    }));

    await authenticatedFetch(`${API_BASE}/api/v1/ai/privacy-consent`);

    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers);
    expect(headers.get('Authorization')).toBe('Bearer browser-session-token');
    expect(useAuthStore.getState().getAccessToken()).resolves.toBe('browser-session-token');
  });
});
