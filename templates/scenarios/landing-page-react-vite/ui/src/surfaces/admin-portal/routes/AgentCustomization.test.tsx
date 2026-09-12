import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '../../../test-utils';
import { AgentCustomization } from './AgentCustomization';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async (importActual) => {
  const actual = await importActual<typeof import('react-router-dom')>();
  return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock('../components/AdminLayout', () => ({
  AdminLayout: ({ children }: { children: React.ReactNode }) => <div data-testid="admin-layout">{children}</div>,
}));

beforeEach(() => {
  vi.clearAllMocks();
});

describe('AgentCustomization [REQ:AGENT-TRIGGER,AGENT-INPUT]', () => {
  it('renders the brief form with asset and preview inputs', () => {
    renderWithProviders(<AgentCustomization />);
    expect(screen.getByTestId('agent-brief-input')).toBeInTheDocument();
    expect(screen.getByTestId('agent-assets-input')).toBeInTheDocument();
    expect(screen.getByTestId('agent-preview-input')).toBeChecked();
  });

  it('blocks submission and surfaces an error when the brief is empty', async () => {
    const user = userEvent.setup();
    renderWithProviders(<AgentCustomization />);
    await user.click(screen.getByTestId('agent-submit'));
    expect(screen.getByText('Please provide a brief for the agent')).toBeInTheDocument();
    // Still on the form, not the captured-result view.
    expect(screen.queryByText('Customization Request Captured')).not.toBeInTheDocument();
  });

  it('captures a structured request with parsed asset URLs on submit', async () => {
    const user = userEvent.setup();
    renderWithProviders(<AgentCustomization />);

    await user.type(screen.getByTestId('agent-brief-input'), 'Make the hero punchier');
    await user.type(
      screen.getByTestId('agent-assets-input'),
      'https://cdn/logo.png\n\n  https://cdn/hero.jpg  ',
    );
    await user.click(screen.getByTestId('agent-submit'));

    expect(screen.getByText('Customization Request Captured')).toBeInTheDocument();
    expect(screen.getByText('Make the hero punchier')).toBeInTheDocument();
    // Two non-empty, trimmed asset lines.
    expect(screen.getByText('2')).toBeInTheDocument();
    expect(screen.getByText('on')).toBeInTheDocument();
  });

  it('resets back to an empty form when creating another request', async () => {
    const user = userEvent.setup();
    renderWithProviders(<AgentCustomization />);

    await user.type(screen.getByTestId('agent-brief-input'), 'Improve CTA copy');
    await user.click(screen.getByTestId('agent-submit'));
    expect(screen.getByText('Customization Request Captured')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Create Another Request/i }));
    expect(screen.getByTestId('agent-brief-input')).toHaveValue('');
  });

  it('navigates back to customization from the header and cancel controls', async () => {
    const user = userEvent.setup();
    renderWithProviders(<AgentCustomization />);
    await user.click(screen.getByRole('button', { name: /^Back$/i }));
    await user.click(screen.getByRole('button', { name: /^Cancel$/i }));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/customization');
  });

  it('reflects preview mode off in the captured request', async () => {
    const user = userEvent.setup();
    renderWithProviders(<AgentCustomization />);
    await user.click(screen.getByTestId('agent-preview-input'));
    await user.type(screen.getByTestId('agent-brief-input'), 'No preview run');
    await user.click(screen.getByTestId('agent-submit'));
    expect(screen.getByText('off')).toBeInTheDocument();
  });
});
