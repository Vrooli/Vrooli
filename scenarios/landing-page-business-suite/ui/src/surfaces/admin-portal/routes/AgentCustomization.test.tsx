import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AgentCustomization } from './AgentCustomization';
import * as agentForm from '../hooks/useAgentForm';

const navigate = vi.fn();
vi.mock('react-router-dom', () => ({ useNavigate: () => navigate }));
vi.mock('../hooks/useAgentForm');
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title, actions }: { title: string; actions?: React.ReactNode }) => <><h1>{title}</h1>{actions}</> }));

function formState(overrides: Record<string, unknown> = {}) {
  return { form: { brief: '', assets: '', preview: true }, result: null, submitting: false, error: null, validationError: null, setBrief: vi.fn(), setAssets: vi.fn(), setPreview: vi.fn(), handleSubmit: vi.fn(), clearResult: vi.fn(), clearValidationError: vi.fn(), ...overrides } as unknown as ReturnType<typeof agentForm.useAgentForm>;
}

describe('AgentCustomization', () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it('collects a structured brief, assets, and preview preference before triggering the agent', () => {
    const state = formState();
    vi.mocked(agentForm.useAgentForm).mockReturnValue(state);
    render(<AgentCustomization />);
    fireEvent.change(screen.getByTestId('agent-brief-input'), { target: { value: 'Improve enterprise conversions' } });
    fireEvent.change(screen.getByTestId('agent-assets-input'), { target: { value: 'https://example.com/logo.svg' } });
    fireEvent.click(screen.getByTestId('agent-preview-input'));
    fireEvent.click(screen.getByTestId('agent-submit'));
    expect(state.setBrief).toHaveBeenCalledWith('Improve enterprise conversions');
    expect(state.setAssets).toHaveBeenCalledWith('https://example.com/logo.svg');
    expect(state.setPreview).toHaveBeenCalledWith(false);
    expect(state.handleSubmit).toHaveBeenCalledOnce();
  });

  it('renders validation feedback and result details, allowing the operator to start another request', () => {
    const state = formState({ result: { job_id: 'job-42', status: 'queued', agent_id: 'agent-7' }, validationError: { title: 'Brief required', message: 'Provide a goal first' } });
    vi.mocked(agentForm.useAgentForm).mockReturnValue(state);
    render(<AgentCustomization />);
    expect(screen.getByText('Agent Customization Triggered')).toBeInTheDocument();
    expect(screen.getByText('job-42')).toBeInTheDocument();
    expect(screen.getByText('queued')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Create Another Request' }));
    fireEvent.click(screen.getByRole('button', { name: 'Back to Customization' }));
    expect(state.clearResult).toHaveBeenCalledOnce();
    expect(navigate).toHaveBeenCalledWith('/admin/customization');
  });
});
