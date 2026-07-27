import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as variants from './variants';
import { apiCall } from './common';

vi.mock('./common', () => ({ apiCall: vi.fn() }));
const mockApiCall = vi.mocked(apiCall);

describe('variant API transport', () => {
  beforeEach(() => { vi.resetAllMocks(); mockApiCall.mockResolvedValue({} as never); });

  it('uses the correct public and admin endpoints and fails closed for malformed variant payloads', async () => {
    await expect(variants.getPublicVariant('control')).rejects.toThrow('Invalid variant response');
    await expect(variants.getVariant('control')).rejects.toThrow('Invalid variant response');
    await expect(variants.createVariant({ name: 'Control', slug: 'control', axes: {} })).rejects.toThrow('Invalid variant response');
    await expect(variants.updateVariant('control', { weight: 50 })).rejects.toThrow('Invalid variant response');
    expect(mockApiCall).toHaveBeenCalledWith('/public/variants/control');
    expect(mockApiCall).toHaveBeenCalledWith('/variants/control');
    expect(mockApiCall).toHaveBeenCalledWith('/variants', expect.objectContaining({ method: 'POST', body: JSON.stringify({ name: 'Control', slug: 'control', axes: {} }) }));
    expect(mockApiCall).toHaveBeenCalledWith('/variants/control', expect.objectContaining({ method: 'PATCH', body: JSON.stringify({ weight: 50 }) }));
  });

  it('uses snapshot, lifecycle, selection, space, and SEO endpoints with validation', async () => {
    await expect(variants.exportVariantSnapshot('control')).rejects.toThrow('Invalid variant snapshot response');
    await expect(variants.importVariantSnapshot('control', {} as never)).rejects.toThrow('Invalid variant snapshot response');
    await expect(variants.archiveVariant('control')).resolves.toEqual({});
    await expect(variants.deleteVariant('control')).resolves.toEqual({});
    await expect(variants.selectVariant()).rejects.toThrow('Invalid variant selection response');
    await expect(variants.selectVariant('control')).rejects.toThrow('Invalid variant selection response');
    await expect(variants.getVariantSpace()).rejects.toThrow('Invalid variant space response');
    await expect(variants.getVariantSEO('control')).rejects.toThrow('Invalid variant SEO response');
    await expect(variants.updateVariantSEO('control', {} as never)).resolves.toEqual({});
    expect(mockApiCall).toHaveBeenCalledWith('/admin/variants/control/export');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/variants/control/import', expect.objectContaining({ method: 'PUT' }));
    expect(mockApiCall).toHaveBeenCalledWith('/variants/control/archive', { method: 'POST' });
    expect(mockApiCall).toHaveBeenCalledWith('/variants/control', { method: 'DELETE' });
    expect(mockApiCall).toHaveBeenCalledWith('/variants/select');
    expect(mockApiCall).toHaveBeenCalledWith('/variants/select?variant_slug=control');
    expect(mockApiCall).toHaveBeenCalledWith('/variant-space');
    expect(mockApiCall).toHaveBeenCalledWith('/seo/control');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/variants/control/seo', expect.objectContaining({ method: 'PUT', credentials: 'include' }));
  });

  it('treats an invalid variant-list envelope as an empty list instead of leaking malformed data', async () => {
    await expect(variants.listVariants()).resolves.toEqual({ variants: [] });
    expect(mockApiCall).toHaveBeenCalledWith('/variants');
  });

  it('returns valid variants and lists without bypassing runtime schema validation', async () => {
    const variant = {
      id: 1, slug: 'control', name: 'Control', status: 'active',
      created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-02T00:00:00Z', archived_at: '', axes: {},
    };
    mockApiCall
      .mockResolvedValueOnce(variant as never)
      .mockResolvedValueOnce(variant as never)
      .mockResolvedValueOnce(variant as never)
      .mockResolvedValueOnce(variant as never)
      .mockResolvedValueOnce({ variants: [variant] } as never);

    await expect(variants.getPublicVariant('control')).resolves.toMatchObject({ slug: 'control' });
    await expect(variants.getVariant('control')).resolves.toMatchObject({ name: 'Control' });
    await expect(variants.createVariant({ name: 'Control', slug: 'control', axes: {} })).resolves.toMatchObject({ status: 'active' });
    await expect(variants.updateVariant('control', { description: 'Updated' })).resolves.toMatchObject({ id: 1 });
    await expect(variants.listVariants()).resolves.toEqual({ variants: [variant] });
  });
});
