import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { CouponsManagement } from './CouponsManagement';
import * as couponHook from '../hooks/useCouponsManagement';
import * as importHook from '../hooks/useCouponImport';
import * as billing from '../../../shared/api/billing';

vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title }: { title: string }) => <h1>{title}</h1> }));
vi.mock('../components/coupons', () => ({
  CouponCard: ({ coupon, onEdit }: { coupon: { id: string }; onEdit: (coupon: { id: string }) => void }) => <button type="button" onClick={() => { onEdit(coupon); }}>{coupon.id}</button>,
  CreateCouponModal: ({ isOpen }: { isOpen: boolean }) => isOpen ? <div>Create modal</div> : null,
  ImportCouponModal: () => <div>Import modal</div>,
  EditCouponModal: ({ isOpen, onSave }: { isOpen: boolean; onSave: (id: string, name: string) => Promise<unknown> }) => isOpen ? <button type="button" onClick={() => { void onSave('launch-20', 'Updated'); }}>Save edit</button> : null,
}));
vi.mock('../hooks/useCouponsManagement');
vi.mock('../hooks/useCouponImport');
vi.mock('../../../shared/api/billing', async () => ({ ...(await vi.importActual('../../../shared/api/billing')), updateCoupon: vi.fn() }));

const coupon = { id: 'launch-20', duration: 'once' as const, times_redeemed: 0, valid: true, created: 1_704_067_200, is_intro_coupon: false, percent_off: 20 };
const actions = { loadCoupons: vi.fn().mockResolvedValue(undefined), openCreateModal: vi.fn(), closeCreateModal: vi.fn(), handleCreate: vi.fn(), handleDelete: vi.fn() };

function managementState(overrides: Partial<ReturnType<typeof couponHook.useCouponsManagement>> = {}) {
  return {
    coupons: [coupon], filteredCoupons: [coupon], introCouponMap: { pro: 'launch-20' }, usageStats: [{ coupon_id: 'launch-20', total_uses: 1 }],
    filter: 'all' as const, setFilter: vi.fn(), totalCount: 1, activeCount: 1, introConfiguredCount: 1,
    loading: false, error: null, createModalOpen: false, creating: false, createError: null, deletingId: null,
    ...actions, clearError: vi.fn(), clearCreateError: vi.fn(), ...overrides,
  };
}

describe('CouponsManagement', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(importHook.useCouponImport).mockReturnValue({ isModalOpen: false, openModal: vi.fn(), closeModal: vi.fn(), preview: null, loading: false, error: null, refreshPreview: vi.fn() });
    vi.mocked(billing.updateCoupon).mockResolvedValue(coupon);
  });

  it('shows monetization summary, intro mapping, and routes operator actions to their hooks', () => {
    const state = managementState();
    vi.mocked(couponHook.useCouponsManagement).mockReturnValue(state);
    const couponImport = importHook.useCouponImport();
    render(<CouponsManagement />);

    expect(screen.getByText('Coupon Management')).toBeInTheDocument();
    expect(screen.getByTestId('coupons-stats')).toHaveTextContent('Total Coupons');
    expect(screen.getByText(/Environment variables are set for 1 tier/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Create Coupon' }));
    fireEvent.click(screen.getByRole('button', { name: 'View Stripe Coupons' }));
    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
    fireEvent.click(screen.getByRole('button', { name: 'Active' }));
    expect(actions.openCreateModal).toHaveBeenCalledOnce();
    expect(couponImport.openModal).toHaveBeenCalledOnce();
    expect(actions.loadCoupons).toHaveBeenCalledOnce();
    expect(state.setFilter).toHaveBeenCalledWith('active');
  });

  it('renders loading and authentication error states without exposing coupon controls', () => {
    vi.mocked(couponHook.useCouponsManagement).mockReturnValue(managementState({ loading: true }));
    const { rerender } = render(<CouponsManagement />);
    expect(screen.getByText('Loading coupons...')).toBeInTheDocument();

    vi.mocked(couponHook.useCouponsManagement).mockReturnValue(managementState({ error: 'Authentication failed' }));
    rerender(<CouponsManagement />);
    expect(screen.getByText('Error: Authentication failed')).toBeInTheDocument();
    expect(screen.getByText('Using a Restricted Stripe Key?')).toBeInTheDocument();
  });

  it('guides the operator through empty state and missing intro coupon remediation', () => {
    vi.mocked(couponHook.useCouponsManagement).mockReturnValue(managementState({ filteredCoupons: [], introCouponMap: { starter: 'missing-coupon' } }));
    const { rerender } = render(<CouponsManagement />);
    expect(screen.getByText('No coupons found')).toBeInTheDocument();
    expect(screen.getByText('Create a coupon to get started')).toBeInTheDocument();

    vi.mocked(couponHook.useCouponsManagement).mockReturnValue(managementState({ introCouponMap: { starter: 'missing-coupon' } }));
    rerender(<CouponsManagement />);
    expect(screen.getByText('Missing Intro Coupon')).toBeInTheDocument();
    expect(screen.getByText(/Coupon "missing-coupon" is configured for the starter tier/)).toBeInTheDocument();
  });

  it('updates a selected coupon and refreshes the source of truth after edit persistence', async () => {
    vi.mocked(couponHook.useCouponsManagement).mockReturnValue(managementState());
    render(<CouponsManagement />);
    fireEvent.click(screen.getByRole('button', { name: 'launch-20' }));
    fireEvent.click(screen.getByRole('button', { name: 'Save edit' }));
    await waitFor(() => { expect(billing.updateCoupon).toHaveBeenCalledWith('launch-20', { name: 'Updated' }); });
    expect(actions.loadCoupons).toHaveBeenCalledOnce();
  });
});
