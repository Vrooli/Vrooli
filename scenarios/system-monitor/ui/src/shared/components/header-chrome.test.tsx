import { describe, expect, it, vi } from 'vitest';
import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders } from '../../test-utils/renderWithProviders';
import { AgentDropdown } from './AgentDropdown';
import { StatusIndicator } from './StatusIndicator';
import type { InvestigationAgentState } from '../../types';

const agent = (overrides: Partial<InvestigationAgentState>): InvestigationAgentState => ({
  agentId: 'a1',
  status: 'investigating',
  startTime: '2026-08-27T00:00:00Z',
  ...overrides
} as InvestigationAgentState);

const dropdown = (agents: InvestigationAgentState[]) => renderWithProviders(
  <AgentDropdown
    agents={agents}
    stoppingAgentIds={new Set()}
    agentErrors={{}}
    onStopAgent={vi.fn().mockResolvedValue(undefined)}
    onRefreshAgents={vi.fn()}
  />
);

const indicator = (props: Partial<React.ComponentProps<typeof StatusIndicator>> = {}) => renderWithProviders(
  <StatusIndicator
    healthStatus={null}
    healthError={null}
    onToggleMonitoring={vi.fn().mockResolvedValue(undefined)}
    onRefreshHealth={vi.fn().mockResolvedValue(undefined)}
    isLoading={false}
    {...props}
  />
);

describe('agent trigger tone', () => {
  it('is loud only while something is running', () => {
    dropdown([agent({ status: 'investigating' })]);
    // A finished investigation used to light the same accent as a running one,
    // which made it the loudest object in the bar for no reason.
    expect(screen.getByRole('button', { name: /1 active/ })).toHaveClass('agent-dropdown-active');
  });

  it('is quiet once every agent has finished', () => {
    dropdown([agent({ status: 'completed' })]);
    expect(screen.getByRole('button', { name: /1 complete/ })).toHaveClass('agent-dropdown-success');
  });

  it('is quiet when there are no agents at all', () => {
    dropdown([]);
    expect(screen.getByRole('button', { name: /none running/ })).toHaveClass('agent-dropdown-idle');
  });
});

describe('health segment', () => {
  it('names the state it is reporting, not just the control', () => {
    indicator({ healthStatus: { status: 'healthy', service: 'system-monitor', maintenance_state: 'active' } });
    expect(screen.getByRole('button', { name: /System status: healthy/ })).toBeInTheDocument();
  });

  it('reports an unreachable API as offline rather than healthy', () => {
    indicator({ healthStatus: { status: 'unhealthy', service: 'system-monitor' } });
    expect(screen.getByRole('button', { name: /System status: offline/ })).toBeInTheDocument();
  });

  it('reports a health error as an error, not as a status', () => {
    indicator({ healthError: 'connection refused' });
    expect(screen.getByRole('button', { name: /System status: error/ })).toBeInTheDocument();
  });

  it('says it is still loading rather than guessing a state', () => {
    indicator({ isLoading: true });
    expect(screen.getByRole('button', { name: /System status: loading/ })).toBeDisabled();
  });

  it('names both the state and what pressing the toggle does', () => {
    indicator({ healthStatus: { status: 'healthy', maintenance_state: 'active' } });
    const toggle = screen.getByRole('button', { name: 'Monitoring active. Pause monitoring' });
    expect(toggle).toHaveAttribute('aria-pressed', 'true');
    expect(toggle).toHaveTextContent('Active');
  });

  it('falls back to the maintenance state when the processor flag is absent', () => {
    indicator({ healthStatus: { status: 'healthy', maintenance_state: 'inactive' } });
    expect(screen.getByRole('button', { name: 'Monitoring inactive. Activate monitoring' })).toHaveTextContent('Inactive');
  });

  it('opens the detail popover from the lamp half of the segment', () => {
    indicator({ healthStatus: { status: 'healthy', service: 'system-monitor', maintenance_state: 'active', api_connectivity: { connected: true, latency_ms: 4 } } });
    fireEvent.click(screen.getByRole('button', { name: /System status: healthy/ }));
    const popover = screen.getByRole('dialog', { name: 'System status details' });
    expect(popover).toHaveTextContent('system-monitor');
    expect(popover).toHaveTextContent('Connected');
  });

  it('shows the health error inside the popover', () => {
    indicator({ healthError: 'connection refused' });
    fireEvent.click(screen.getByRole('button', { name: /System status: error/ }));
    expect(screen.getByRole('dialog')).toHaveTextContent('connection refused');
  });
});
