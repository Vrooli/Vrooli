import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { APIKeysSettings } from './APIKeysSettings';
import * as keysHook from '../hooks/useAPIKeysForm';

vi.mock('../hooks/useAPIKeysForm');
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title, actions }: { title: string; actions?: React.ReactNode }) => <><h1>{title}</h1>{actions}</> }));
const addToast = vi.fn();
vi.mock('../../../shared/ui/useToast', () => ({ useToast: () => ({ addToast }) }));

function formState(overrides: Record<string, unknown> = {}) {
  return {
    keys: [], testResults: {}, availableProviders: [{ value: 'openai', label: 'OpenAI', description: 'OpenAI API' }], showAddModal: false, newKeyProvider: '', newKeyValue: '',
    loading: false, testingProvider: null, addingKey: false, fetchKeys: vi.fn(), handleAddKey: vi.fn().mockResolvedValue({ success: true, message: 'Key added' }),
    handleDeleteKey: vi.fn().mockResolvedValue({ success: true, message: 'Key deleted' }), handleTestKey: vi.fn().mockResolvedValue({ success: true, message: 'Connected' }), handleToggleKey: vi.fn().mockResolvedValue({ success: true, message: 'Disabled' }),
    setShowAddModal: vi.fn(), setNewKeyProvider: vi.fn(), setNewKeyValue: vi.fn(), clearAddForm: vi.fn(), ...overrides,
  } as unknown as ReturnType<typeof keysHook.useAPIKeysForm>;
}

describe('APIKeysSettings', () => {
  beforeEach(() => { vi.clearAllMocks(); vi.stubGlobal('confirm', vi.fn(() => true)); });

  it('shows loading and empty-state guidance without rendering secrets', () => {
    vi.mocked(keysHook.useAPIKeysForm).mockReturnValue(formState({ loading: true }));
    const { rerender } = render(<APIKeysSettings />);
    expect(screen.getByText('Loading API keys...')).toBeInTheDocument();
    vi.mocked(keysHook.useAPIKeysForm).mockReturnValue(formState());
    rerender(<APIKeysSettings />);
    expect(screen.getByText('No API keys configured yet')).toBeInTheDocument();
  });

  it('opens the add flow and sends success feedback only through the form hook', async () => {
    const state = formState({ showAddModal: true, newKeyProvider: 'openai', newKeyValue: 'sk-secret' });
    vi.mocked(keysHook.useAPIKeysForm).mockReturnValue(state);
    render(<APIKeysSettings />);
    fireEvent.click(screen.getByRole('button', { name: 'Add Key' }));
    await waitFor(() => { expect(state.handleAddKey).toHaveBeenCalledOnce(); });
    expect(addToast).toHaveBeenCalledWith(expect.objectContaining({ type: 'success', message: 'Key added' }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(state.clearAddForm).toHaveBeenCalledOnce();
  });

  it('tests, disables, and confirm-deletes configured provider keys with operator feedback', async () => {
    const state = formState({ keys: [{ id: 1, provider: 'openai', key_hint: 'sk-...1234', is_active: true, created_at: '2026-01-01T00:00:00Z' }], testResults: { openai: { success: true, message: 'Connected' } } });
    vi.mocked(keysHook.useAPIKeysForm).mockReturnValue(state);
    render(<APIKeysSettings />);
    expect(screen.getByText('sk-...1234')).toBeInTheDocument();
    expect(screen.getByText('Connected')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Test' }));
    fireEvent.click(screen.getByRole('button', { name: 'Disable' }));
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    await waitFor(() => { expect(state.handleTestKey).toHaveBeenCalledWith('openai'); });
    expect(state.handleToggleKey).toHaveBeenCalledWith('openai', true);
    expect(state.handleDeleteKey).toHaveBeenCalledWith('openai');
  });

  it('keeps failed add/test/toggle operations visible and does not delete when confirmation is declined', async () => {
    const state = formState({
      keys: [{ id: 1, provider: 'openai', key_hint: 'sk-...1234', is_active: false, created_at: '2026-01-01T00:00:00Z' }],
      testResults: { openai: { success: false, message: 'Credential rejected' } },
      showAddModal: true,
      newKeyProvider: 'openai',
      newKeyValue: 'sk-invalid',
      handleAddKey: vi.fn().mockResolvedValue({ success: false, message: 'Key rejected' }),
      handleTestKey: vi.fn().mockResolvedValue({ success: false, message: 'Test rejected' }),
      handleToggleKey: vi.fn().mockResolvedValue({ success: false, message: 'Toggle rejected' }),
    });
    vi.mocked(keysHook.useAPIKeysForm).mockReturnValue(state);
    vi.stubGlobal('confirm', vi.fn(() => false));
    render(<APIKeysSettings />);

    expect(screen.getByText('Credential rejected')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Add Key' }));
    fireEvent.click(screen.getByRole('button', { name: 'Test' }));
    fireEvent.click(screen.getByRole('button', { name: 'Enable' }));
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    await waitFor(() => { expect(state.handleAddKey).toHaveBeenCalledOnce(); });

    expect(state.handleTestKey).toHaveBeenCalledWith('openai');
    expect(state.handleToggleKey).toHaveBeenCalledWith('openai', false);
    expect(state.handleDeleteKey).not.toHaveBeenCalled();
    expect(addToast).toHaveBeenCalledWith({ type: 'error', message: 'Key rejected' });
    expect(addToast).toHaveBeenCalledWith({ type: 'error', message: 'Test rejected' });
    expect(addToast).toHaveBeenCalledWith({ type: 'error', message: 'Toggle rejected' });
  });
});
