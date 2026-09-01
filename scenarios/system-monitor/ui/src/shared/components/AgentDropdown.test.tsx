import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { AgentDropdown } from './AgentDropdown';
import type { InvestigationAgentState } from '../../types';

const active: InvestigationAgentState = {
  id: 'active-1', status: 'investigating', startTime: new Date().toISOString(), autoFix: true,
  operationMode: 'report-only', model: 'model-x', note: 'check disk', label: 'Disk agent',
};
const complete: InvestigationAgentState = {
  id: 'complete-1', status: 'completed', startTime: new Date(Date.now() - 60000).toISOString(), autoFix: false,
};

const renderDropdown = (agents: InvestigationAgentState[] = [active], stopping = new Set<string>(), errors: Record<string, string> = {}) => {
  const onStop = vi.fn().mockResolvedValue(undefined);
  const onRefresh = vi.fn();
  render(<AgentDropdown agents={agents} stoppingAgentIds={stopping} agentErrors={errors} onStopAgent={onStop} onRefreshAgents={onRefresh} />);
  return { onStop, onRefresh };
};

describe('AgentDropdown', () => {
  it('shows empty state and closes when agents disappear', async () => {
    const { rerender } = render(<AgentDropdown agents={[active]} stoppingAgentIds={new Set()} agentErrors={{}} onStopAgent={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Investigation agents: 1 active' }));
    expect(screen.getByText('Disk agent')).toBeInTheDocument();
    rerender(<AgentDropdown agents={[]} stoppingAgentIds={new Set()} agentErrors={{}} onStopAgent={vi.fn()} />);
    await waitFor(() => expect(screen.queryByText('No investigation agents are running.')).not.toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: 'Investigation agents: none running' }));
    expect(screen.getByText('No investigation agents are running.')).toBeInTheDocument();
  });

  it('sorts agents, exposes status metadata, refreshes, stops, and handles errors', async () => {
    const { onRefresh } = renderDropdown([complete, active], new Set(['active-1']), { 'active-1': 'Cannot stop now' });
    expect(screen.getByRole('button', { name: 'Investigation agents: 2 active' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Investigation agents: 2 active' }));
    expect(screen.getByText('Disk agent')).toBeInTheDocument();
    expect(screen.getAllByText('Mode: report-only')).toHaveLength(2);
    expect(screen.getByText('Auto-fix: enabled')).toBeInTheDocument();
    expect(screen.getByText('Model: model-x')).toBeInTheDocument();
    expect(screen.getByText('"check disk"')).toBeInTheDocument();
    expect(screen.getByText('Cannot stop now')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'STOPPING' })).toBeDisabled();
    fireEvent.click(screen.getByTitle('Refresh agent status'));
    expect(onRefresh).toHaveBeenCalledOnce();
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(screen.queryByText('Disk agent')).not.toBeInTheDocument();

    const second = renderDropdown([active]);
    fireEvent.click(screen.getByRole('button', { name: 'Investigation agents: 1 active' }));
    fireEvent.click(screen.getByRole('button', { name: 'STOP' }));
    await waitFor(() => { expect(second.onStop).toHaveBeenCalledWith('active-1'); });
  });

  it('labels a terminal-only set as complete and disables clearing', () => {
    renderDropdown([complete]);
    expect(screen.getByRole('button', { name: 'Investigation agents: 1 complete' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Investigation agents: 1 complete' }));
    expect(screen.getByRole('button', { name: 'CLEARED' })).toBeDisabled();
  });
});
