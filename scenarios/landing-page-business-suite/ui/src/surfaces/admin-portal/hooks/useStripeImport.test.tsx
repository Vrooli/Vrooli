import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import type { StripeImportPreview, StripeImportResult } from '../../../shared/api/billing';
import { useStripeImport } from './useStripeImport';
import type { getStripeImportPreview, importStripePlans } from '../../../shared/api/billing';

type GetStripeImportPreviewFn = typeof getStripeImportPreview;
type ImportStripePlansFn = typeof importStripePlans;

const getStripeImportPreviewMock = vi.fn<Parameters<GetStripeImportPreviewFn>, ReturnType<GetStripeImportPreviewFn>>();
const importStripePlansMock = vi.fn<Parameters<ImportStripePlansFn>, ReturnType<ImportStripePlansFn>>();

vi.mock('../../../shared/api/billing', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api/billing')>('../../../shared/api/billing');
  return {
    ...actual,
    getStripeImportPreview: (...args: Parameters<GetStripeImportPreviewFn>) => getStripeImportPreviewMock(...args),
    importStripePlans: (...args: Parameters<ImportStripePlansFn>) => importStripePlansMock(...args),
  };
});

const preview: StripeImportPreview = {
  bundle_key: 'business_suite',
  bundle_product_id: 'prod_123',
  bundle_product_found: true,
  bundle_plan_count: 1,
  products: [
    {
      product_id: 'prod_123',
      product_name: 'Business Suite',
      is_current_bundle: true,
      prices: [
        {
          price_id: 'price_123',
          lookup_key: 'pro_month',
          currency: 'usd',
          amount_cents: 9900,
          interval: 'month',
          product_id: 'prod_123',
          product_name: 'Business Suite',
          active: true,
          exists_locally: false,
        },
      ],
    },
  ],
  total_prices: 1,
  conflict_count: 0,
  new_count: 1,
};

describe('useStripeImport', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getStripeImportPreviewMock.mockResolvedValue(preview);
  });

  it('closes the modal and refreshes on successful import', async () => {
    const resultPayload: StripeImportResult = {
      imported: 1,
      overwritten: 0,
      skipped: 0,
    };
    importStripePlansMock.mockResolvedValue(resultPayload);

    const onImportComplete = vi.fn();
    const { result } = renderHook(() => useStripeImport(onImportComplete));

    await act(async () => {
      await result.current.openModal();
    });

    expect(result.current.isModalOpen).toBe(true);
    expect(result.current.selections['price_123']).toBe(true);

    await act(async () => {
      await result.current.handleImport();
    });

    expect(onImportComplete).toHaveBeenCalledTimes(1);
    expect(result.current.isModalOpen).toBe(false);
    expect(result.current.preview).toBeNull();
  });

  it('reports preview failures while leaving the operator with a closable modal', async () => {
    getStripeImportPreviewMock.mockRejectedValueOnce(new Error('Stripe unavailable'));
    const { result } = renderHook(() => useStripeImport());
    await act(async () => { await result.current.openModal(); });
    expect(result.current).toMatchObject({ isModalOpen: true, loading: false, error: 'Stripe unavailable', preview: null });
    act(() => { result.current.closeModal(); });
    expect(result.current.isModalOpen).toBe(false);
  });

  it('supports product-specific active, existing, manual, and cleared selections', async () => {
    const multiProduct = {
      ...preview,
      bundle_product_id: '',
      products: [{
        ...preview.products[0]!, product_id: 'prod_other', prices: [
          { ...preview.products[0]!.prices[0]!, price_id: 'active-new', active: true, exists_locally: false },
          { ...preview.products[0]!.prices[0]!, price_id: 'inactive-existing', active: false, exists_locally: true },
        ],
      }, { ...preview.products[0]!, product_id: 'prod_second', prices: [] }],
    };
    getStripeImportPreviewMock.mockResolvedValueOnce(multiProduct);
    const { result } = renderHook(() => useStripeImport());
    await act(async () => { await result.current.openModal(); });
    expect(result.current.selectedProductId).toBe('');
    act(() => { result.current.selectProduct('prod_other'); });
    expect(result.current.selections).toEqual({ 'active-new': true, 'inactive-existing': false });
    act(() => { result.current.selectExisting(); });
    expect(result.current.selections).toEqual({ 'active-new': false, 'inactive-existing': true });
    act(() => { result.current.setPriceSelected('active-new', true); });
    act(() => { result.current.clearSelections(); });
    expect(result.current.selections).toEqual({ 'active-new': false, 'inactive-existing': false });
    act(() => { result.current.selectActive(); });
    expect(result.current.selections).toEqual({ 'active-new': true, 'inactive-existing': false });
  });

  it('rejects imports with no selected product or no selected price before making network requests', async () => {
    const { result } = renderHook(() => useStripeImport());
    await act(async () => { await result.current.handleImport(); });
    expect(result.current.error).toBe('Select a Stripe product to import');
    await act(async () => { await result.current.openModal(); });
    act(() => { result.current.clearSelections(); });
    await act(async () => { await result.current.handleImport(); });
    expect(result.current.error).toBe('No prices selected for import');
    expect(importStripePlansMock).not.toHaveBeenCalled();
  });

  it('keeps partial imports open, refreshes catalog state, and reports import failures', async () => {
    const refreshed = { ...preview, products: [] };
    importStripePlansMock.mockResolvedValueOnce({ imported: 0, overwritten: 0, skipped: 0, errors: ['price unavailable'] });
    getStripeImportPreviewMock.mockResolvedValueOnce(preview).mockResolvedValueOnce(refreshed);
    const { result } = renderHook(() => useStripeImport());
    await act(async () => { await result.current.openModal(); });
    await act(async () => { await result.current.handleImport(); });
    expect(result.current).toMatchObject({ isModalOpen: true, selectedProductId: '', error: 'Import completed with 1 error(s)' });
    expect(result.current.importResult?.errors).toEqual(['price unavailable']);
    act(() => { result.current.resetImportResult(); });
    expect(result.current.importResult).toBeNull();

    importStripePlansMock.mockRejectedValueOnce(new Error('Import denied'));
    getStripeImportPreviewMock.mockResolvedValueOnce(preview);
    await act(async () => { await result.current.openModal(); });
    await act(async () => { await result.current.handleImport(); });
    expect(result.current.error).toBe('Import denied');
  });
});
