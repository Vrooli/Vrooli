import { beforeEach, describe, expect, it, vi } from 'vitest';

const client = vi.hoisted(() => ({ getBranding: vi.fn(), updateBranding: vi.fn(), clearBrandingField: vi.fn(), getPublicBranding: vi.fn() }));
vi.mock('@connectrpc/connect', () => ({ createClient: vi.fn(() => client) }));
vi.mock('./common', () => ({ CONNECT_API_BASE: 'http://api.example.test' }));

import { clearBrandingField, getBranding, getPublicBranding, updateBranding } from './branding';

describe('branding Connect API', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('normalizes full branding into the existing UI model', async () => {
    client.getBranding.mockResolvedValue({ branding: { toJson: () => ({ id: '1', siteName: 'Acme', smtpHost: 'smtp.example.test', comingSoonEnabled: true }) } });
    await expect(getBranding()).resolves.toEqual({ id: 1, site_name: 'Acme', smtp_host: 'smtp.example.test', coming_soon_enabled: true });
    expect(client.getBranding).toHaveBeenCalledWith({});
  });

  it('preserves update presence and clear-field requests', async () => {
    const response = { branding: { toJson: () => ({ id: '1', siteName: 'Acme' }) } };
    client.updateBranding.mockResolvedValue(response);
    client.clearBrandingField.mockResolvedValue(response);
    await updateBranding({ theme_primary_color: '#123456', coming_soon_enabled: false });
    await clearBrandingField('smtp_password');
    expect(client.updateBranding).toHaveBeenCalledWith({ themePrimaryColor: '#123456', comingSoonEnabled: false });
    expect(client.clearBrandingField).toHaveBeenCalledWith({ field: 'smtp_password' });
  });

  it('normalizes the redacted public response', async () => {
    client.getPublicBranding.mockResolvedValue({ branding: { toJson: () => ({ siteName: 'Acme', supportChatUrl: 'https://chat.example.test' }) } });
    await expect(getPublicBranding()).resolves.toEqual({ site_name: 'Acme', support_chat_url: 'https://chat.example.test' });
  });
});
