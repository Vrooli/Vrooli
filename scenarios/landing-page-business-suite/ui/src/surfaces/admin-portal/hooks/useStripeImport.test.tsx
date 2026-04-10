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
});
