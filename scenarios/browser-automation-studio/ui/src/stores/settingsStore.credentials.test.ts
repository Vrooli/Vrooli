import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useSettingsStore } from './settingsStore';

describe('settingsStore credential handling', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 200 })));
    useSettingsStore.getState().clearApiKeys();
  });

  it('provisions provider keys without writing them to browser local storage', () => {
    const secret = 'test-openrouter-secret';

    useSettingsStore.getState().setApiKey('openrouterApiKey', secret);

    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/credentials/provision',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ identity: 'vrooli/openrouter', field: 'api-key', value: secret }),
      })
    );
    expect(localStorage.getItem('browserAutomation.settings.apiKeys')).toBeNull();
  });
});
