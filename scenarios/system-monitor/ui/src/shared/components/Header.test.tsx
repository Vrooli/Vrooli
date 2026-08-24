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
});
