import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { TiersManagement } from './TiersManagement';
import * as billingForm from '../hooks/useBillingForm';
import * as stripeImportHook from '../hooks/useStripeImport';
import * as couponMappingsHook from '../hooks/useCouponMappings';

vi.mock('../hooks/useBillingForm');
vi.mock('../hooks/useStripeImport');
vi.mock('../hooks/useCouponMappings');
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title }: { title: string }) => <h1>{title}</h1> }));
vi.mock('../components/plans', () => ({
  PlanDisplayManager: ({ bundles, onAddPlan }: { bundles: unknown[]; onAddPlan: (key: string) => void }) => <><p>Plan display: {bundles.length}</p><button onClick={() => { onAddPlan('bundle-2'); }}>Add bundle plan</button></>,
  ImportStripeModal: () => <div>Stripe import modal</div>,
  AddPlanModal: ({ bundleKey, isOpen, onClose, onSuccess }: { bundleKey: string; isOpen: boolean; onClose: () => void; onSuccess: () => void }) => isOpen ? <div>Adding to {bundleKey}<button onClick={onClose}>Close plan modal</button><button onClick={onSuccess}>Plan added</button></div> : null,
}));

function formState(overrides: Record<string, unknown> = {}) {
  return {
    bundles: [], priceForms: {}, bundleError: null, loadingBundles: false, loadBundles: vi.fn(), includeDemoPlaceholders: false, toggleDemoPlaceholders: vi.fn(),
    handlePriceChange: vi.fn(), handleSavePrice: vi.fn(), handleVerifyPrice: vi.fn(), priceChecks: {}, removeDemoPlan: vi.fn(), handleDeletePlan: vi.fn(), handleReorderPlans: vi.fn(), pricingTab: 'plans', setPricingTab: vi.fn(),
    ...overrides,
  } as unknown as ReturnType<typeof billingForm.useBillingForm>;
}

describe('TiersManagement', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(couponMappingsHook.useCouponMappings).mockReturnValue({ availableCoupons: [], mappings: [], saving: false, assignCoupon: vi.fn(), unassignCoupon: vi.fn() } as unknown as ReturnType<typeof couponMappingsHook.useCouponMappings>);
    vi.mocked(stripeImportHook.useStripeImport).mockReturnValue({ openModal: vi.fn() } as unknown as ReturnType<typeof stripeImportHook.useStripeImport>);
  });

  it('counts visible monthly and yearly plans while excluding demo placeholders', () => {
    vi.mocked(billingForm.useBillingForm).mockReturnValue(formState({ bundles: [{ bundle: { bundle_key: 'business_suite' }, prices: [
      { display_enabled: true, billing_interval: 'month' }, { display_enabled: false, billing_interval: 'year' }, { display_enabled: true, metadata: { __demo_placeholder: true } },
    ] }] }));
    render(<TiersManagement />);
    expect(screen.getByTestId('tiers-stats')).toHaveTextContent('Total Plans2');
    expect(screen.getByTestId('tiers-stats')).toHaveTextContent('Enabled1');
    expect(screen.getByTestId('tiers-stats')).toHaveTextContent('Monthly1');
    expect(screen.getByTestId('tiers-stats')).toHaveTextContent('Yearly1');
  });

  it('opens add-plan flows with the default or selected bundle and reloads after success', () => {
    const state = formState({ bundles: [{ bundle: { bundle_key: 'business_suite' }, prices: [] }] });
    vi.mocked(billingForm.useBillingForm).mockReturnValue(state);
    render(<TiersManagement />);
    fireEvent.click(screen.getByRole('button', { name: 'Add plan' }));
    expect(screen.getByText('Adding to business_suite')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Close plan modal' }));
    fireEvent.click(screen.getByRole('button', { name: 'Add bundle plan' }));
    expect(screen.getByText('Adding to bundle-2')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Plan added' }));
    expect(state.loadBundles).toHaveBeenCalledOnce();
  });

  it('opens the Stripe import workflow for importing configured products', () => {
    const stripeImport = { openModal: vi.fn().mockResolvedValue(undefined) } as unknown as ReturnType<typeof stripeImportHook.useStripeImport>;
    vi.mocked(stripeImportHook.useStripeImport).mockReturnValue(stripeImport);
    vi.mocked(billingForm.useBillingForm).mockReturnValue(formState());
    render(<TiersManagement />);
    fireEvent.click(screen.getByRole('button', { name: 'Import from Stripe' }));
    expect(stripeImport.openModal).toHaveBeenCalledOnce();
  });
});
