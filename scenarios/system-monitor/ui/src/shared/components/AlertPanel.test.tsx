import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { AlertPanel } from './AlertPanel';

describe('AlertPanel feed states', () => {
  it('keeps condition order stable while focused and reranks after focus leaves', () => {
    const older = { id: 'old', timestamp: '2026-10-07T11:40:00Z', severity: 'warning', category: 'cpu_usage', message: 'Older warning', resolved: false };
    const newer = { id: 'new', timestamp: '2026-10-07T11:50:00Z', severity: 'warning', category: 'memory_usage', message: 'Newer warning', resolved: false };
    const { container, rerender } = render(<AlertPanel alerts={[older, newer]} feed={{ status: 'ready', ruleIssues: [] }} onInspect={vi.fn()} />);
    const order = () => [...container.querySelectorAll('.alert-item__content strong')].map(element => element.textContent);
    expect(order()).toEqual(['memory_usage', 'cpu_usage']);
    const inspect = screen.getByRole('button', { name: /inspect memory/i });
    fireEvent.focus(inspect);
    rerender(<AlertPanel alerts={[{ ...older, severity: 'critical' }, newer]} feed={{ status: 'ready', ruleIssues: [] }} onInspect={vi.fn()} />);
    expect(order()).toEqual(['memory_usage', 'cpu_usage']);
    fireEvent.blur(inspect, { relatedTarget: document.body });
    expect(order()).toEqual(['cpu_usage', 'memory_usage']);
  });

  it('ranks conditions, retains known conditions when evaluation is unknown, and opens their resource', () => {
    const inspect = vi.fn();
    render(<AlertPanel onInspect={inspect} alerts={[
      { id: 'm', timestamp: '2026-10-07T11:50:00Z', severity: 'warning', category: 'memory_usage', message: 'Memory high', resolved: false },
      { id: 'c', timestamp: '2026-10-07T11:45:00Z', severity: 'critical', category: 'cpu_usage', message: 'CPU high', resolved: false },
    ]} feed={{ status: 'error', error: 'offline', ruleIssues: [] }} />);
    expect(screen.getByText(/not revalidated/)).toBeInTheDocument();
    const actions = screen.getAllByRole('button', { name: /Inspect/ });
    expect(actions.map(action => action.textContent)).toEqual(['Inspect cpu', 'Inspect memory']);
    const firstAction = actions.at(0);
    if (!firstAction) throw new Error('Expected at least one alert inspection action');
    fireEvent.click(firstAction);
    expect(inspect).toHaveBeenCalledWith('cpu', expect.objectContaining({ id: 'c', category: 'cpu_usage' }), false);
  });

  it('offers the existing investigation workspace for an unmapped server condition', () => {
    const open = vi.fn();
    render(<AlertPanel alerts={[{ id: 'a', timestamp: '', severity: 'warning', category: 'orphaned_agents', message: 'Needs attention', resolved: false }]} feed={{ status: 'ready', ruleIssues: [] }} onInspectOther={open} />);
    fireEvent.click(screen.getByRole('button', { name: 'View investigations' }));
    expect(open).toHaveBeenCalledOnce();
  });

  it('offers a read-only refresh when evaluation status is unavailable', () => {
    const refresh = vi.fn();
    render(<AlertPanel alerts={[]} feed={{ status: 'error', error: 'offline', ruleIssues: [] }} onRefresh={refresh} />);
    fireEvent.click(screen.getByRole('button', { name: 'Refresh evaluation status' }));
    expect(refresh).toHaveBeenCalledOnce();
  });

  it('distinguishes a paused view from collection and evaluation', () => {
    render(<AlertPanel alerts={[]} feed={{ status: 'ready', ruleIssues: [] }} viewPaused />);
    expect(screen.getByText(/Collection and alert evaluation may continue/)).toBeInTheDocument();
  });

  it('never renders an unreachable alert feed as a healthy empty list', () => {
    render(<AlertPanel alerts={[]} feed={{ status: 'error', error: 'Unable to reach the server.', ruleIssues: [] }} />);
    expect(screen.getByText('ALERT STATE UNAVAILABLE')).toBeInTheDocument();
    expect(screen.getByLabelText('Current alert count unavailable')).toHaveTextContent('—');
    expect(screen.queryByText('NO ACTIVE ALERTS')).not.toBeInTheDocument();
  });

  it('says monitoring is paused instead of claiming no alerts', () => {
    render(<AlertPanel alerts={[]} feed={{ status: 'paused', ruleIssues: [] }} />);
    expect(screen.getByText('MONITORING PAUSED — ALERTS NOT EVALUATED')).toBeInTheDocument();
  });

  it('lists server alerts with escalation detail and rules that cannot evaluate', () => {
    render(
      <AlertPanel
        alerts={[{ id: 'a', timestamp: '2026-10-07T11:50:00Z', severity: 'critical', category: 'memory_pressure', message: 'Memory pressure is critical', resolved: false, details: 'Escalated to investigation (anomaly_1)' }]}
        feed={{ status: 'ready', ruleIssues: [{ ruleId: 'orphaned_agents', status: 'unsupported', reason: 'control plane does not classify orphans yet' }] }}
      />,
    );
    expect(screen.getByText('CRITICAL')).toBeInTheDocument();
    expect(screen.getByText('memory_pressure')).toBeInTheDocument();
    expect(screen.getByText('Escalated to investigation (anomaly_1)')).toBeInTheDocument();
    expect(screen.getByText(/orphaned_agents: unsupported/)).toBeInTheDocument();
  });

  it('does not claim a fully evaluated empty scope when rules cannot evaluate', () => {
    render(<AlertPanel alerts={[]} feed={{ status: 'ready', ruleIssues: [{ ruleId: 'cpu_usage', status: 'failed', reason: 'no sample' }] }} />);
    expect(screen.getByText('NO ACTIVE ALERTS IN EVALUATED SCOPE')).toBeInTheDocument();
    expect(screen.getByText(/cpu_usage: failed/)).toBeInTheDocument();
  });

  it('shows the healthy empty state only when the evaluator has run', () => {
    render(<AlertPanel alerts={[]} feed={{ status: 'ready', ruleIssues: [] }} />);
    expect(screen.getByText('NO ACTIVE ALERTS')).toBeInTheDocument();
  });

  it('says the rules have not run yet before the first evaluation', () => {
    render(<AlertPanel alerts={[]} feed={{ status: 'unevaluated', ruleIssues: [] }} />);
    expect(screen.getByText('ALERT RULES HAVE NOT RUN YET')).toBeInTheDocument();
  });

  it('keeps the last known alerts visible, and says so, when a refresh fails', () => {
    render(
      <AlertPanel
        alerts={[{ id: 'b', timestamp: '', severity: 'warning', category: 'orphaned_agents', message: '5 orphaned sessions', resolved: false }]}
        feed={{ status: 'error', error: 'Unable to reach the server.', ruleIssues: [] }}
      />,
    );
    expect(screen.getByText(/could not be refreshed \(Unable to reach the server\.\); showing the last known alerts/)).toBeInTheDocument();
    expect(screen.getByText('WARNING')).toBeInTheDocument();
    expect(screen.getByText('orphaned_agents')).toBeInTheDocument();
    expect(screen.getByText('5 orphaned sessions')).toBeInTheDocument();
  });
});

