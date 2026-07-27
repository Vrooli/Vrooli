import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { deleteAsset, getAssetUrl, listAssets, uploadAsset } from './assets';

const asset = {
  id: 7,
  filename: 'logo.png',
  original_filename: 'logo.png',
  mime_type: 'image/png',
  size_bytes: 42,
  storage_path: 'logos/logo.png',
  category: 'logo',
  created_at: '2026-01-01T00:00:00Z',
  url: '/uploads/logos/logo.png',
};

describe('assets API', () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('resolves external, embedded, API-relative, and storage paths safely', () => {
    expect(getAssetUrl('https://cdn.example/logo.png')).toBe('https://cdn.example/logo.png');
    expect(getAssetUrl('data:image/png;base64,abc')).toBe('data:image/png;base64,abc');
    expect(getAssetUrl('')).toBe('');
    expect(getAssetUrl('logos/logo.png')).toMatch(/\/uploads\/logos\/logo\.png$/);
    expect(getAssetUrl('/api/v1/uploads/logos/logo.png')).toMatch(/\/api\/v1\/uploads\/logos\/logo\.png$/);
  });

  it('uploads a file with optional metadata and validates the response', async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify(asset), { status: 201 }));
    const result = await uploadAsset(new File(['image'], 'logo.png', { type: 'image/png' }), {
      category: 'logo', altText: 'Company logo', uploadedBy: 'admin',
    });
    expect(result).toEqual(asset);
    const [, options] = fetchMock.mock.calls[0] as [string, RequestInit];
    const formData = options.body as FormData;
    expect(options.method).toBe('POST');
    expect(formData.get('category')).toBe('logo');
    expect(formData.get('alt_text')).toBe('Company logo');
  });

  it('reports failed uploads and lists empty or filtered assets safely', async () => {
    fetchMock.mockResolvedValueOnce(new Response('too large', { status: 413 }));
    await expect(uploadAsset(new File(['x'], 'x.png'))).rejects.toThrow('too large');

    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({}), { status: 200 }));
    await expect(listAssets()).resolves.toEqual([]);

    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ assets: [asset] }), { status: 200 }));
    await expect(listAssets('logo')).resolves.toEqual([asset]);
    const [filteredListURL] = fetchMock.mock.calls[2] as unknown as [string, RequestInit];
    expect(filteredListURL).toContain('category=logo');
  });

  it('deletes assets and surfaces deletion failures', async () => {
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }));
    await expect(deleteAsset(7)).resolves.toBeUndefined();
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 500 }));
    await expect(deleteAsset(7)).rejects.toThrow('Failed to delete asset: 500');
  });
});
