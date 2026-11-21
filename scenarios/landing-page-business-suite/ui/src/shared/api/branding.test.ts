import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, renderHook, waitFor } from '@testing-library/react';
import { create, fromJsonString, toJson } from '@bufbuild/protobuf';
import { BrandingResponseSchema, PublicBrandingResponseSchema, UpdateBrandingRequestSchema, type BrandingService } from '@vrooli/proto-types/landing-page-business-suite/v1/branding_pb';
import { Code, ConnectError, type Client } from '@connectrpc/connect';

type BrandingClient = Client<typeof BrandingService>;
const client = vi.hoisted(() => ({
  getBranding: vi.fn<BrandingClient['getBranding']>(), updateBranding: vi.fn<BrandingClient['updateBranding']>(),
  clearBrandingField: vi.fn<BrandingClient['clearBrandingField']>(), getPublicBranding: vi.fn<BrandingClient['getPublicBranding']>(),
}));
vi.mock('@connectrpc/connect', async original => ({
  ...await original<typeof import('@connectrpc/connect')>(), createClient: vi.fn(() => client),
}));
vi.mock('./common', () => ({ CONNECT_API_BASE: 'http://api.example.test' }));

import { clearBrandingField, getBranding, getPublicBranding, updateBranding } from './branding';
import { useComingSoonToggle } from '../../surfaces/admin-portal/hooks/useComingSoonToggle';

// No service/hook/codec mocks: the admin hook uses the real waitlist service,
// branding API and installed generated descriptors. Only RPC delivery is synthetic.
const privateFields = {
  id: '7', site_name: 'Orion Workspace', tagline: 'Owner tagline', logo_url: '/logo.svg',
  logo_icon_url: '/mark.svg', favicon_url: '/favicon.svg', apple_touch_icon_url: '/apple.png',
  default_title: 'Private default title', default_description: 'Owner description', default_og_image_url: '/og.png',
  theme_primary_color: '#123456', theme_background_color: '#fafafa', canonical_base_url: 'https://owner.example',
  google_site_verification: 'private-verification', robots_txt: 'User-agent: *',
  support_chat_url: 'https://support.example', support_email: 'support@example.test',
  smtp_host: 'smtp.example.test', smtp_port: 587, smtp_username: 'private-user', smtp_password: 'private-test-secret', smtp_from: 'owner@example.test',
  coming_soon_enabled: true, coming_soon_message: 'Configured availability',
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-09-15T00:00:00Z',
};
const privateResponse = (comingSoon = true) => fromJsonString(BrandingResponseSchema, JSON.stringify({ branding: { ...privateFields, coming_soon_enabled: comingSoon } }));
const publicFields = {
  site_name: 'Public owner', tagline: 'Public tagline', logo_url: '/public-logo.svg', logo_icon_url: '/public-mark.svg',
  favicon_url: '/public-favicon.svg', theme_primary_color: '#654321', theme_background_color: '#eeeeee',
  canonical_base_url: 'https://canonical.example/base', support_chat_url: 'https://public-support.example',
  coming_soon_enabled: false, coming_soon_message: '',
  // Business identity is deliberately public: it is shown in the footer, contact and legal pages.
  support_email: 'hello@public.example', legal_name: 'Public Owner LLC', contact_address: '1 Main Street\nSpringfield',
  privacy_policy_markdown: '# Privacy', terms_markdown: '# Terms', privacy_effective_date: '2026-09-16', terms_effective_date: '2026-09-01',
};
const publicResponse = () => fromJsonString(PublicBrandingResponseSchema, JSON.stringify({ branding: publicFields }));

beforeEach(() => {
  vi.clearAllMocks();
  client.getBranding.mockResolvedValue(privateResponse());
  client.updateBranding.mockResolvedValue(privateResponse(false));
  client.clearBrandingField.mockResolvedValue(privateResponse(false));
  client.getPublicBranding.mockResolvedValue(publicResponse());
});
afterEach(() => { cleanup(); vi.restoreAllMocks(); });

describe('generated branding codec', () => {
  it('preserves every private owner field, canonical timestamps and a safe numeric identity', async () => {
    await expect(getBranding()).resolves.toEqual({ ...privateFields, id: 7 });
    expect(client.getBranding).toHaveBeenCalledWith({});
    expect(client.getPublicBranding).not.toHaveBeenCalled();
  });

  it('preserves explicit false/empty updates and leaves omitted fields absent', async () => {
    await expect(updateBranding({ coming_soon_enabled: false, tagline: '', smtp_port: 0 })).resolves.toMatchObject({ coming_soon_enabled: false });
    expect(client.updateBranding).toHaveBeenCalledWith(fromJsonString(UpdateBrandingRequestSchema, '{"coming_soon_enabled":false,"tagline":"","smtp_port":0}'));
    const request = client.updateBranding.mock.calls[0]?.[0];
    expect(request).toBeDefined();
    expect(toJson(UpdateBrandingRequestSchema, create(UpdateBrandingRequestSchema, request), { useProtoFieldName: true })).toEqual({ coming_soon_enabled: false, tagline: '', smtp_port: 0 });
    expect(client.getPublicBranding).not.toHaveBeenCalled();
  });

  it('uses the existing clear-field RPC and accepts the owner-cleared optional field', async () => {
    const { smtp_password: _password, ...cleared } = privateFields;
    client.clearBrandingField.mockResolvedValue(fromJsonString(BrandingResponseSchema, JSON.stringify({ branding: cleared })));
    const result = await clearBrandingField('smtp_password');
    expect(client.clearBrandingField).toHaveBeenCalledWith({ field: 'smtp_password' });
    expect(result.smtp_password).toBeUndefined();
    expect(result.site_name).toBe('Orion Workspace');
  });

  it('preserves the exact public subset and canonical authority, never private metadata', async () => {
    const response = publicResponse();
    Object.assign(response.branding ?? {}, { smtpPassword: 'must-not-leak', smtpHost: 'smtp.private.test', defaultTitle: 'PRIVATE', id: 12n });
    client.getPublicBranding.mockResolvedValue(response);
    await expect(getPublicBranding()).resolves.toEqual(publicFields);
    expect(client.getBranding).not.toHaveBeenCalled();
  });

  it('retains implicit public false/empty values while keeping private optional values absent', async () => {
    client.getPublicBranding.mockResolvedValue(fromJsonString(PublicBrandingResponseSchema, '{"branding":{"site_name":"Owner"}}'));
    expect(await getPublicBranding()).toMatchObject({ site_name: 'Owner', coming_soon_enabled: false, canonical_base_url: '' });
    client.getBranding.mockResolvedValue(fromJsonString(BrandingResponseSchema, '{"branding":{"id":"1","site_name":"Owner"}}'));
    const admin = await getBranding();
    expect(admin).toEqual({ id: 1, site_name: 'Owner' });
  });

  it.each(['get', 'update', 'clear', 'public'] as const)('fails closed for missing %s branding instead of using public/private defaults', async method => {
    const empty = fromJsonString(BrandingResponseSchema, '{}');
    client.getBranding.mockResolvedValue(empty); client.updateBranding.mockResolvedValue(empty); client.clearBrandingField.mockResolvedValue(empty);
    client.getPublicBranding.mockResolvedValue(fromJsonString(PublicBrandingResponseSchema, '{}'));
    const result = method === 'get' ? getBranding() : method === 'update' ? updateBranding({ coming_soon_enabled: false }) : method === 'clear' ? clearBrandingField('tagline') : getPublicBranding();
    await expect(result).rejects.toThrow('Missing branding response');
  });

  it('rejects unsafe int64 identity rather than silently rounding it', async () => {
    client.getBranding.mockResolvedValue(fromJsonString(BrandingResponseSchema, '{"branding":{"id":"9007199254740993","site_name":"Owner"}}'));
    await expect(getBranding()).rejects.toThrow('Invalid SiteBranding response');
  });

  it('validates update names and scalar values through the generated descriptor before sending', () => {
    const unknownField = { site_name: 'Owner', private_extra: true };
    expect(() => updateBranding(unknownField)).toThrow();
    expect(() => updateBranding({ smtp_port: 1.5 })).toThrow();
    expect(client.updateBranding).not.toHaveBeenCalled();
  });
});

describe('coming-soon admin path through generated branding responses', () => {
  it.each([true, false])('reads the private owner and explicitly toggles %s without public fallback or automatic writes', async initial => {
    client.getBranding.mockResolvedValue(privateResponse(initial));
    client.updateBranding.mockResolvedValue(privateResponse(!initial));
    const { result } = renderHook(() => useComingSoonToggle());
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    expect(result.current.error).toBeUndefined();
    expect(result.current.comingSoonEnabled).toBe(initial);
    expect(client.updateBranding).not.toHaveBeenCalled();
    await act(async () => { expect(await result.current.handleToggle()).toEqual({ success: true }); });
    expect(client.updateBranding).toHaveBeenCalledWith(fromJsonString(UpdateBrandingRequestSchema, JSON.stringify({ coming_soon_enabled: !initial })));
    expect(result.current.comingSoonEnabled).toBe(!initial);
    expect(client.getPublicBranding).not.toHaveBeenCalled();
    expect(JSON.stringify(result.current)).not.toContain('private-test-secret');
  });

  it.each(['missing', 'unauthorized'] as const)('keeps the toggle unavailable after a %s private response without consulting public branding', async kind => {
    if (kind === 'missing') client.getBranding.mockResolvedValue(fromJsonString(BrandingResponseSchema, '{}'));
    else client.getBranding.mockRejectedValue(new ConnectError('Administrator session expired', Code.Unauthenticated));
    const { result } = renderHook(() => useComingSoonToggle());
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    expect(result.current.error).toContain('unavailable');
    await act(async () => { expect((await result.current.handleToggle()).success).toBe(false); });
    expect(client.updateBranding).not.toHaveBeenCalled();
    expect(client.getPublicBranding).not.toHaveBeenCalled();
  });
});
