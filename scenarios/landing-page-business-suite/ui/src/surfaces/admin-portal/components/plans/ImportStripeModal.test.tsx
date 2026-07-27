import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { ImportStripeModal } from './ImportStripeModal';

const stripeImport = () => ({
  isModalOpen: true, closeModal: vi.fn(), loading: false, error: null, importing: false, importResult: { imported: 1, overwritten: 2, skipped: 0, errors: ['warning'] },
  preview: { bundle_product_id: 'prod_current', bundle_plan_count: 2, bundle_product_found: true, products: [
    { product_id: 'prod_current', product_name: 'Current Product', is_current_bundle: true, prices: [
      { price_id: 'price_month', amount_cents: 7900, currency: 'usd', interval: 'month', active: true, exists_locally: true, lookup_key: 'pro_month' },
      { price_id: 'price_year', amount_cents: 79000, currency: 'usd', interval: 'year', active: false, exists_locally: false },
    ] },
    { product_id: 'prod_other', product_name: 'Other Product', prices: [] },
  ] },
  selectedProductId: 'prod_current', selectedProduct: { product_id: 'prod_current', product_name: 'Current Product', prices: [
    { price_id: 'price_month', amount_cents: 7900, currency: 'usd', interval: 'month', active: true, exists_locally: true, lookup_key: 'pro_month' },
    { price_id: 'price_year', amount_cents: 79000, currency: 'usd', interval: 'year', active: false, exists_locally: false },
  ] },
  selectProduct: vi.fn(), selections: { price_month: true }, setPriceSelected: vi.fn(), setSelectionsForPrices: vi.fn(), selectActive: vi.fn(), selectExisting: vi.fn(), clearSelections: vi.fn(), handleImport: vi.fn(),
} as never);

describe('ImportStripeModal', () => {
  it('filters Stripe data and invokes selection/import controls with destructive confirmation', () => {
    const data = stripeImport();
    render(<ImportStripeModal stripeImport={data} />);

    expect(screen.getByText('Imported: 1 | Overwritten: 2 | Skipped: 0 | Errors: 1')).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText('Search products'), { target: { value: 'other' } });
    expect(screen.getByRole('button', { name: /Other Product/ })).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText('Search products'), { target: { value: 'missing' } });
    expect(screen.getByText('No products match your search.')).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText('Search products'), { target: { value: '' } });
    fireEvent.click(screen.getByRole('button', { name: /Other Product/ }));
    fireEvent.click(screen.getByRole('button', { name: 'Select Active' }));
    fireEvent.click(screen.getByRole('button', { name: 'Select Existing' }));
    fireEvent.click(screen.getByRole('button', { name: 'Clear Selection' }));
    fireEvent.click(screen.getByRole('button', { name: 'Inactive' }));
    expect(screen.getByText('price_year')).toBeInTheDocument();
    fireEvent.click(screen.getAllByRole('checkbox')[1]!);
    fireEvent.click(screen.getAllByRole('checkbox')[2]!);
    fireEvent.click(screen.getByRole('checkbox', { name: /I understand/ }));
    expect(screen.getByRole('button', { name: 'Import Selected' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect((data as any).selectProduct).toHaveBeenCalledWith('prod_other');
    expect((data as any).selectActive).toHaveBeenCalledOnce();
    expect((data as any).setPriceSelected).toHaveBeenCalled();
    expect((data as any).closeModal).toHaveBeenCalledOnce();
  });

  it('reports loading, import errors, and an empty Stripe catalog without enabling destructive import', () => {
    const data = stripeImport();
    Object.assign(data as object, {
      loading: true,
      error: 'Stripe credentials need attention',
      importResult: null,
      preview: { bundle_product_id: '', bundle_plan_count: 0, bundle_product_found: false, products: [] },
      selectedProductId: null,
      selectedProduct: null,
      selections: {},
    });
    render(<ImportStripeModal stripeImport={data} />);

    expect(screen.getByText('Loading Stripe products...')).toBeInTheDocument();
    expect(screen.getByText('Stripe credentials need attention')).toBeInTheDocument();
    expect(screen.queryByText('No products found in your Stripe account.')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Import Selected' })).toBeDisabled();
  });

  it('requires acknowledgement before relinking an existing bundle and imports only after confirmation', () => {
    const data = stripeImport();
    Object.assign(data as object, {
      selectedProductId: 'prod_other',
      selectedProduct: {
        product_id: 'prod_other', product_name: 'Other Product', prices: [
          { price_id: 'price_once', amount_cents: 500, currency: 'usd', active: true, exists_locally: false },
        ],
      },
      selections: { price_once: true },
      preview: {
        bundle_product_id: 'prod_current', bundle_plan_count: 1, bundle_product_found: false,
        products: [{ product_id: 'prod_other', product_name: 'Other Product', prices: [] }],
      },
    });
    render(<ImportStripeModal stripeImport={data} />);

    expect(screen.getByText(/Current bundle product prod_current was not found/)).toBeInTheDocument();
    expect(screen.getByText(/Switching products will relink/)).toBeInTheDocument();
    expect(screen.getByText('Import will replace 1 existing plan in the catalog.')).toBeInTheDocument();
    expect(screen.getByText('one-time')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Import Selected' })).toBeDisabled();
    fireEvent.click(screen.getByRole('checkbox', { name: /I understand this will relink/ }));
    fireEvent.click(screen.getByRole('button', { name: 'Import Selected' }));

    expect((data as any).handleImport).toHaveBeenCalledOnce();
  });
});
