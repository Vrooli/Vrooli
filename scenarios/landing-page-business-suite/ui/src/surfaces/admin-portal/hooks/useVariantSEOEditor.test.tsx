import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useVariantSEOEditor } from './useVariantSEOEditor';
import * as seoController from '../controllers/seoController';

vi.mock('../controllers/seoController', async () => ({
  ...(await vi.importActual('../controllers/seoController')),
  loadVariantSEOConfig: vi.fn(),
  saveVariantSEOConfig: vi.fn(),
}));

describe('useVariantSEOEditor', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(seoController.loadVariantSEOConfig).mockResolvedValue({ title: 'Original title', twitter_card: 'summary_large_image' });
    vi.mocked(seoController.saveVariantSEOConfig).mockResolvedValue(undefined);
  });

  afterEach(() => { vi.useRealTimers(); });

  it('loads the editable variant config and reloads when the target variant changes', async () => {
    const { result, rerender } = renderHook(({ slug }) => useVariantSEOEditor({ variantSlug: slug }), { initialProps: { slug: 'control' } });
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    expect(result.current.seoConfig.title).toBe('Original title');

    rerender({ slug: 'campaign-b' });
    await waitFor(() => { expect(seoController.loadVariantSEOConfig).toHaveBeenLastCalledWith('campaign-b', undefined); });
  });

  it('saves changed SEO fields, invokes the parent refresh, and clears the success state', async () => {
    const onSave = vi.fn();
    const { result } = renderHook(() => useVariantSEOEditor({ variantSlug: 'control', onSave }));
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    vi.useFakeTimers();

    act(() => { result.current.updateField('title', 'Campaign title'); });
    await act(async () => { await result.current.handleSave(); });
    expect(seoController.saveVariantSEOConfig).toHaveBeenCalledWith('control', expect.objectContaining({ title: 'Campaign title' }));
    expect(onSave).toHaveBeenCalledOnce();
    expect(result.current.success).toBe(true);

    act(() => { vi.advanceTimersByTime(3000); });
    expect(result.current.success).toBe(false);
  });

  it('preserves editability and returns a useful error when a save fails', async () => {
    vi.mocked(seoController.saveVariantSEOConfig).mockRejectedValue(new Error('SEO settings were rejected'));
    const { result } = renderHook(() => useVariantSEOEditor({ variantSlug: 'control' }));
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    act(() => { result.current.updateField('noindex', true); });

    await act(async () => { await result.current.handleSave(); });
    expect(result.current).toMatchObject({ saving: false, success: false, error: 'SEO settings were rejected' });
    expect(result.current.seoConfig.noindex).toBe(true);
  });

  it('returns a safe load error rather than showing stale SEO data', async () => {
    vi.mocked(seoController.loadVariantSEOConfig).mockRejectedValue(new Error('Network unavailable'));
    const { result } = renderHook(() => useVariantSEOEditor({ variantSlug: 'control' }));
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    expect(result.current).toMatchObject({ seoConfig: {}, error: 'Failed to load SEO settings' });
  });
});
