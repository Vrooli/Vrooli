import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { TierLimitsSettings } from './TierLimitsSettings';
import * as tierLimitsHook from '../hooks/useTierLimitsForm';

vi.mock('../hooks/useTierLimitsForm');
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title }: { title: string }) => <h1>{title}</h1> }));
const addToast = vi.fn();
vi.mock('../../../shared/ui/useToast', () => ({ useToast: () => ({ addToast }) }));

function formState(overrides: Record<string, unknown> = {}) {
  return {
    limits: {}, loading: false, saving: null, editedValues: {}, toasts: [],
    handleSave: vi.fn(), updateEditedValue: vi.fn(), resetToDefaults: vi.fn(), doubleAllLimits: vi.fn(), clearToasts: vi.fn(),
    getEditKey: (tier: string, key: string) => `${tier}:${key}`,
    getTierLabel: (tier: string) => tier === 'solo' ? 'Solo' : tier,
    getTierColor: () => 'text-white',
    findAICreditsLimit: (limits: unknown) => limits,
    isUnlimitedValue: (value: number) => value === -1,
    TIER_OPTIONS: [{ value: 'solo' }, { value: 'business' }],
    ...overrides,
  } as unknown as ReturnType<typeof tierLimitsHook.useTierLimitsForm>;
}

describe('TierLimitsSettings', () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it('renders loading and empty states before limits are available', () => {
    vi.mocked(tierLimitsHook.useTierLimitsForm).mockReturnValue(formState({ loading: true }));
    const { rerender } = render(<TierLimitsSettings />);
    expect(screen.getByText('Loading tier limits...')).toBeInTheDocument();

    vi.mocked(tierLimitsHook.useTierLimitsForm).mockReturnValue(formState());
    rerender(<TierLimitsSettings />);
    expect(screen.getByText('No tier limits configured')).toBeInTheDocument();
  });

  it('edits finite and unlimited tiers, forwards toast feedback, and exposes bulk operations', () => {
    const soloLimit = { limit_value: 2500000, cost_multiplier: 1000000, display_dollars: 2.5 };
    const businessLimit = { limit_value: -1, cost_multiplier: 1000000, display_dollars: 0 };
    const state = formState({
      limits: { solo: soloLimit, business: businessLimit },
      editedValues: { 'solo:ai_credits': '3.00' },
      toasts: [{ type: 'success', message: 'Tier limits refreshed' }],
    });
    vi.mocked(tierLimitsHook.useTierLimitsForm).mockReturnValue(state);
    render(<TierLimitsSettings />);

    expect(screen.getByText('Unlimited')).toBeInTheDocument();
    expect(screen.getByDisplayValue('3.00')).toBeInTheDocument();
    expect(screen.getByDisplayValue('unlimited')).toBeInTheDocument();
    fireEvent.change(screen.getByDisplayValue('3.00'), { target: { value: '4.25' } });
    const saveButtons = screen.getAllByRole('button', { name: 'Save' });
    fireEvent.click(saveButtons[0]!);
    expect(saveButtons[1]!).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: 'Reset to Defaults' }));
    fireEvent.click(screen.getByRole('button', { name: 'Double All Limits' }));

    expect(state.updateEditedValue).toHaveBeenCalledWith('solo:ai_credits', '4.25');
    expect(state.handleSave).toHaveBeenCalledWith('solo', soloLimit);
    expect(state.resetToDefaults).toHaveBeenCalledOnce();
    expect(state.doubleAllLimits).toHaveBeenCalledOnce();
    expect(addToast).toHaveBeenCalledWith({ type: 'success', message: 'Tier limits refreshed' });
    expect(state.clearToasts).toHaveBeenCalledOnce();
  });
});
