import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { StorageWizard } from './StorageWizard';
import * as wizardHook from '../../hooks/useStorageWizard';

vi.mock('../../hooks/useStorageWizard');
vi.mock('./StepProvider', () => ({ StepProvider: () => <div>Provider step</div> }));
vi.mock('./StepConfiguration', () => ({ StepConfiguration: () => <div>Configuration step</div> }));
vi.mock('./StepCredentials', () => ({ StepCredentials: () => <div>Credentials step</div> }));
vi.mock('./StepVerify', () => ({ StepVerify: () => <div>Verify step</div> }));

function wizardState(overrides: Record<string, unknown> = {}) {
  return {
    state: { step: 0, provider: null, form: {}, credentials: {}, cloudflareAccountId: '', existingSettings: null, loading: false, loadError: null, testStatus: 'idle', testError: null, saveStatus: 'idle', saveError: null },
    currentStepId: 'provider', steps: ['provider', 'configure', 'credentials', 'verify'], canGoBack: false, canGoNext: false, isLastStep: false,
    goToStep: vi.fn(), goBack: vi.fn(), goNext: vi.fn(), setProvider: vi.fn(), setCloudflareAccountId: vi.fn(), setForm: vi.fn(), setCredentials: vi.fn(),
    loadExistingSettings: vi.fn(), testConnection: vi.fn(), saveSettings: vi.fn(), reset: vi.fn(), ...overrides,
  } as unknown as ReturnType<typeof wizardHook.useStorageWizard>;
}

describe('StorageWizard', () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it('shows a loading state while the persisted storage configuration is being read', () => {
    vi.mocked(wizardHook.useStorageWizard).mockReturnValue(wizardState({ state: { ...wizardState().state, loading: true } }));
    render(<StorageWizard initialSettings={null} onComplete={vi.fn()} />);
    expect(screen.getByText('Loading storage settings...')).toBeInTheDocument();
  });

  it('renders provider selection with disabled progression until the hook authorizes it', () => {
    const state = wizardState();
    vi.mocked(wizardHook.useStorageWizard).mockReturnValue(state);
    render(<StorageWizard initialSettings={null} onComplete={vi.fn()} />);
    expect(screen.getByText('Provider step')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Back' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();
    expect(state.loadExistingSettings).toHaveBeenCalledOnce();
  });

  it('renders the correct child step and delegates navigation for configured providers', () => {
    const state = wizardState({
      state: { ...wizardState().state, step: 1, provider: 'minio' }, currentStepId: 'configure', canGoBack: true, canGoNext: true,
    });
    vi.mocked(wizardHook.useStorageWizard).mockReturnValue(state);
    render(<StorageWizard initialSettings={null} onComplete={vi.fn()} />);
    expect(screen.getByText('Configuration step')).toBeInTheDocument();
    vi.mocked(state.goBack).mockClear();
    vi.mocked(state.goNext).mockClear();
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    expect(state.goBack).toHaveBeenCalledOnce();
    expect(state.goNext).toHaveBeenCalledOnce();
  });

  it('keeps verify navigation available and surfaces load errors', () => {
    const state = wizardState({
      state: { ...wizardState().state, step: 3, provider: 'aws-s3', loadError: 'Saved storage settings are malformed' }, currentStepId: 'verify', canGoBack: true, isLastStep: true,
    });
    vi.mocked(wizardHook.useStorageWizard).mockReturnValue(state);
    render(<StorageWizard initialSettings={null} onComplete={vi.fn()} />);
    expect(screen.getByText('Verify step')).toBeInTheDocument();
    expect(screen.getByText('Saved storage settings are malformed')).toBeInTheDocument();
    vi.mocked(state.goBack).mockClear();
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    expect(state.goBack).toHaveBeenCalledOnce();
  });
});
