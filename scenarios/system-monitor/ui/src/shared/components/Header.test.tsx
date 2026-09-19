import { fireEvent, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import type { InvestigationAgentState } from '../../types';
import type { SystemHealthStatus } from '../../features/monitoring/hooks/useSystemMonitor';
import { Header } from './Header';

const renderHeader = () => {
  const onToggleMonitoring = vi.fn().mockResolvedValue(undefined);
  const onRefreshHealth = vi.fn().mockResolvedValue(undefined);
  const onToggleTerminal = vi.fn();
  const onOpenSettings = vi.fn();
  const props = {
    unreadErrorCount: 0,
    agents: [] as InvestigationAgentState[],
    onStopAgent: vi.fn().mockResolvedValue(undefined),
    stoppingAgentIds: new Set<string>(),
    agentErrors: {},
    onToggleTerminal,
    onOpenSettings,
    healthStatus: { status: 'healthy', processor_active: false } as SystemHealthStatus,
    healthError: null,
    onToggleMonitoring,
    onRefreshHealth,
    isLoadingHealth: false,
  };
  return { ...render(<Header {...props} />), onToggleMonitoring, onRefreshHealth };
};

describe('Header mobile navigation', () => {
  it('opens a labeled navigation panel, traps focus, and restores focus on Escape', () => {
    renderHeader();
    const trigger = screen.getByRole('button', { name: 'Open navigation' });

    fireEvent.click(trigger);

    expect(screen.getByRole('complementary', { name: 'Primary navigation' })).toBeInTheDocument();
    const panel = screen.getByRole('complementary', { name: 'Primary navigation' });
    expect(within(panel).getByRole('link', { name: /Dashboard/ })).toBeInTheDocument();
    expect(document.activeElement).toBe(within(panel).getByRole('button', { name: 'Close navigation' }));

    fireEvent.keyDown(document, { key: 'Escape' });

    expect(screen.queryByRole('complementary', { name: 'Primary navigation' })).not.toBeInTheDocument();
    expect(document.activeElement).toBe(trigger);
  });

  it('wraps Tab focus within the open panel', () => {
    renderHeader();
    fireEvent.click(screen.getByRole('button', { name: 'Open navigation' }));
    const panel = screen.getByRole('complementary', { name: 'Primary navigation' });
    const closeButton = within(panel).getByRole('button', { name: 'Close navigation' });
    const links = within(panel).getAllByRole('link');
    const lastLink = links[links.length - 1];

    lastLink.focus();
    fireEvent.keyDown(document, { key: 'Tab' });
    expect(document.activeElement).toBe(closeButton);

    fireEvent.keyDown(document, { key: 'Tab', shiftKey: true });
    expect(document.activeElement).toBe(lastLink);
  });

  it('names the machine in view, reaches linking from the picker, and disables local output remotely', () => {
    const onAddMachine = vi.fn();
    render(<Header
      unreadErrorCount={0}
      agents={[]}
      onStopAgent={vi.fn().mockResolvedValue(undefined)}
      stoppingAgentIds={new Set()}
      agentErrors={{}}
      onToggleTerminal={vi.fn()}
      onOpenSettings={vi.fn()}
      healthStatus={null}
      healthError={null}
      onToggleMonitoring={vi.fn().mockResolvedValue(undefined)}
      onRefreshHealth={vi.fn().mockResolvedValue(undefined)}
      isLoadingHealth={false}
      machines={[
        { id: '', name: 'This machine', os: 'linux', arch: 'x86_64', online: true, heartbeat_fresh: true, dispatchable: true, status: 'local' },
        { id: 'mac-node', name: 'Mac mini', os: 'darwin', arch: 'amd64', online: true, heartbeat_fresh: true, heartbeat_age_seconds: 8, dispatchable: true, status: 'online', grant: 'Read only; changes are not permitted', scopes: ['system-monitor:read'] },
        { id: 'gone', name: 'swarminator', os: 'linux', arch: 'amd64', online: false, heartbeat_fresh: false, heartbeat_age_seconds: 639479, dispatchable: false, status: 'offline', readiness: [{ identity: 'heartbeat_fresh', passed: false }] }
      ]}
      selectedMachineID="mac-node"
      onSelectMachine={vi.fn()}
      onAddMachine={onAddMachine}
      terminalDisabledReason="System output is local to this computer"
    />);

    // The trigger names the subject without being opened; a reader must never
    // have to open a menu to learn which machine they are looking at.
    expect(screen.getByTestId('machine-picker')).toHaveTextContent('Mac mini');

    fireEvent.click(screen.getByTestId('machine-picker'));

    // Reachability is stated in the picker, not as an error after choosing.
    expect(screen.getByRole('option', { name: /Mac mini/ })).toHaveTextContent('darwin · amd64 · 8s ago');
    expect(screen.getByRole('option', { name: /swarminator/ })).toHaveTextContent('not responding · 7d ago');

    // The grant is legible before any action is offered.
    expect(screen.getByRole('option', { name: /This machine/ })).toHaveTextContent('linux · x86_64');

    fireEvent.click(screen.getByTestId('add-machine'));
    expect(onAddMachine).toHaveBeenCalledOnce();
    expect(screen.getByRole('button', { name: 'System output is local to this computer' })).toBeDisabled();
  });

  it('offers linking even when the only machine is this computer', () => {
    const onAddMachine = vi.fn();
    render(<Header
      unreadErrorCount={0}
      agents={[]}
      onStopAgent={vi.fn().mockResolvedValue(undefined)}
      stoppingAgentIds={new Set()}
      agentErrors={{}}
      onToggleTerminal={vi.fn()}
      onOpenSettings={vi.fn()}
      healthStatus={null}
      healthError={null}
      onToggleMonitoring={vi.fn().mockResolvedValue(undefined)}
      onRefreshHealth={vi.fn().mockResolvedValue(undefined)}
      isLoadingHealth={false}
      machines={[{ id: '', name: 'This machine', os: 'linux', arch: 'x86_64', online: true, heartbeat_fresh: true, dispatchable: true, status: 'local' }]}
      selectedMachineID=""
      onSelectMachine={vi.fn()}
      onAddMachine={onAddMachine}
    />);

    fireEvent.click(screen.getByTestId('machine-picker'));
    // A fleet of one must still reach linking from here: routing that through
    // vrooli-bridge is the detour this control exists to remove.
    expect(screen.getByTestId('add-machine')).toBeInTheDocument();
  });
});
