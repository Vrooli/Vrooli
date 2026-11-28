import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const assetsClient = vi.hoisted(() => ({ listAssets: vi.fn(), deleteAsset: vi.fn() }));
vi.mock('@connectrpc/connect', () => ({ createClient: vi.fn(() => assetsClient) }));

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
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('resolves external, embedded, API-relative, and storage paths safely', () => {
    expect(getAssetUrl('https://cdn.example/logo.png')).toBe('https://cdn.example/logo.png');
    expect(getAssetUrl('http://cdn.example/logo.png')).toBe('http://cdn.example/logo.png');
    expect(getAssetUrl('data:image/png;base64,abc')).toBe('data:image/png;base64,abc');
    expect(getAssetUrl('')).toBe('');
    expect(getAssetUrl('logos/logo.png')).toMatch(/\/uploads\/logos\/logo\.png$/);
    expect(getAssetUrl('/logos/logo.png')).toMatch(/\/uploads\/logos\/logo\.png$/);
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

  it('reports failed uploads and lists empty or filtered assets through the generated contract', async () => {
    fetchMock.mockResolvedValueOnce(new Response('too large', { status: 413 }));
    await expect(uploadAsset(new File(['x'], 'x.png'))).rejects.toThrow('too large');

    fetchMock.mockResolvedValueOnce(new Response('', { status: 500 }));
    await expect(uploadAsset(new File(['x'], 'x.png'))).rejects.toThrow('Upload failed with status 500');

    assetsClient.listAssets.mockResolvedValueOnce({ assets: [] });
    await expect(listAssets()).resolves.toEqual([]);

    assetsClient.listAssets.mockResolvedValueOnce({ assets: [{ id: 7n, filename: 'logo.png', originalFilename: 'logo.png', mimeType: 'image/png', sizeBytes: 42n, storagePath: 'logos/logo.png', category: 'logo', createdAt: { seconds: 1767225600n, nanos: 0 }, url: '/uploads/logos/logo.png', derivatives: {} }] });
    await expect(listAssets('logo')).resolves.toEqual([asset]);
    expect(assetsClient.listAssets).toHaveBeenLastCalledWith({ category: 'logo' });
  });

  it('preserves populated optional asset metadata from the generated contract', async () => {
    assetsClient.listAssets.mockResolvedValueOnce({
      assets: [{
        id: 8n,
        filename: 'social.png',
        originalFilename: 'social-source.png',
        mimeType: 'image/png',
        sizeBytes: 84n,
        storagePath: 'og-images/social.png',
        thumbnailPath: 'og-images/social-thumb.png',
        altText: 'Launch graphic',
        category: 'og_image',
        uploadedBy: 'operator@example.test',
        createdAt: { seconds: 1767225600n, nanos: 500_000_000 },
        url: '/uploads/og-images/social.png',
        derivatives: { og_image_1200x630: 'og-images/social-1200.png' },
      }],
    });

    await expect(listAssets()).resolves.toEqual([{
      id: 8,
      filename: 'social.png',
      original_filename: 'social-source.png',
      mime_type: 'image/png',
      size_bytes: 84,
      storage_path: 'og-images/social.png',
      thumbnail_path: 'og-images/social-thumb.png',
      alt_text: 'Launch graphic',
      category: 'og_image',
      uploaded_by: 'operator@example.test',
      created_at: '2026-01-01T00:00:00.500Z',
      url: '/uploads/og-images/social.png',
      derivatives: { og_image_1200x630: 'og-images/social-1200.png' },
    }]);
  });

  it('deletes assets through the generated contract and requires acknowledgement', async () => {
    assetsClient.deleteAsset.mockResolvedValueOnce({ deleted: true });
    await expect(deleteAsset(7)).resolves.toBeUndefined();
    expect(assetsClient.deleteAsset).toHaveBeenCalledWith({ id: 7n });
    assetsClient.deleteAsset.mockResolvedValueOnce({ deleted: false });
    await expect(deleteAsset(7)).rejects.toThrow('acknowledged');
  });
});
