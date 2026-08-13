import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AppLimitsSettings } from './AppLimitsSettings';
import * as limitsHook from '../hooks/useAppLimitsForm';

vi.mock('../hooks/useAppLimitsForm');
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title }: { title: string }) => <h1>{title}</h1> }));
const addToast = vi.fn();
vi.mock('../../../shared/ui/useToast', () => ({ useToast: () => ({ addToast }) }));

function formState(overrides: Record<string, unknown> = {}) {
  return {
    selectedApp: 'browser-automation-studio', limits: {}, editedValues: {}, newLimit: { tier_id: 'solo', limit_key: '', display_dollars: '' }, limitKeys: new Set(),
    loading: false, saving: null, showAddLimit: false, setSelectedApp: vi.fn(), setEditedValue: vi.fn(), setNewLimit: vi.fn(), setShowAddLimit: vi.fn(),
    handleSave: vi.fn().mockResolvedValue({ success: true, message: 'Limit saved' }), handleAddLimit: vi.fn().mockResolvedValue({ success: true, message: 'Limit created' }),
    handleDeleteLimit: vi.fn().mockResolvedValue({ success: true, message: 'Limit deleted' }), refreshLimits: vi.fn(), resetNewLimitForm: vi.fn(), clearEditedValue: vi.fn(),
    getEditedOrDisplayValue: vi.fn().mockReturnValue('10.00'), isEdited: vi.fn().mockReturnValue(true), ...overrides,
  } as unknown as ReturnType<typeof limitsHook.useAppLimitsForm>;
}

describe('AppLimitsSettings', () => {
  beforeEach(() => { vi.clearAllMocks(); vi.stubGlobal('confirm', vi.fn(() => true)); });

  it('shows loading and directs an operator to create the first app-specific limit', () => {
    vi.mocked(limitsHook.useAppLimitsForm).mockReturnValue(formState({ loading: true }));
    const { rerender } = render(<AppLimitsSettings />);
    expect(screen.getByText('Loading app limits...')).toBeInTheDocument();
    const state = formState();
    vi.mocked(limitsHook.useAppLimitsForm).mockReturnValue(state);
    rerender(<AppLimitsSettings />);
    expect(screen.getByText(/No app-specific limits configured/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Add First Limit' }));
    expect(state.setShowAddLimit).toHaveBeenCalledWith(true);
  });

  it('edits, saves, and confirms deletion of a tier limit with result feedback', async () => {
    const limit = { id: 'limit-1', tier_id: 'solo', limit_type: 'app_specific', limit_key: 'api_calls', limit_value: 10000000, cost_multiplier: 1000000, reset_period: 'monthly', created_at: '', updated_at: '' } as const;
    const state = formState({ limits: { solo: [limit] }, limitKeys: new Set(['api_calls']) });
    vi.mocked(limitsHook.useAppLimitsForm).mockReturnValue(state);
    render(<AppLimitsSettings />);
    fireEvent.change(screen.getByLabelText('Limit'), { target: { value: '12.00' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    fireEvent.click(screen.getByRole('button', { name: 'Delete api_calls limit for Solo' }));
    await waitFor(() => { expect(state.handleSave).toHaveBeenCalledWith('solo', limit); });
    expect(state.setEditedValue).toHaveBeenCalledWith('solo:api_calls', '12.00');
    expect(state.handleDeleteLimit).toHaveBeenCalledWith('solo', 'api_calls');
    expect(addToast).toHaveBeenCalledWith(expect.objectContaining({ type: 'success', message: 'Limit saved' }));
    expect(addToast).toHaveBeenCalledWith(expect.objectContaining({ type: 'success', message: 'Limit deleted' }));
  });

  it('collects the add-limit fields and clears the form when the modal is cancelled', async () => {
    const state = formState({ showAddLimit: true });
    vi.mocked(limitsHook.useAppLimitsForm).mockReturnValue(state);
    render(<AppLimitsSettings />);
    fireEvent.change(screen.getByLabelText('Limit Key'), { target: { value: 'workflow_exports' } });
    fireEvent.change(screen.getByLabelText('Dollar Value'), { target: { value: '5.50' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create Limit' }));
    await waitFor(() => { expect(state.handleAddLimit).toHaveBeenCalledOnce(); });
    expect(state.setNewLimit).toHaveBeenCalledWith({ limit_key: 'workflow_exports' });
    expect(state.setNewLimit).toHaveBeenCalledWith({ display_dollars: '5.50' });
    expect(addToast).toHaveBeenCalledWith(expect.objectContaining({ type: 'success', message: 'Limit created' }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(state.setShowAddLimit).toHaveBeenCalledWith(false);
    expect(state.resetNewLimitForm).toHaveBeenCalledOnce();
  });
});
