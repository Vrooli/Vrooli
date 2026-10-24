import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { AlertPanel } from './AlertPanel';

describe('AlertPanel feed states', () => {
  it('never renders an unreachable alert feed as a healthy empty list', () => {
    render(<AlertPanel alerts={[]} feed={{ status: 'error', error: 'Unable to reach the server.', ruleIssues: [] }} />);
    expect(screen.getByText('ALERT STATE UNAVAILABLE')).toBeInTheDocument();
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
    expect(screen.getByText('[CRITICAL] memory_pressure')).toBeInTheDocument();
    expect(screen.getByText('Escalated to investigation (anomaly_1)')).toBeInTheDocument();
    expect(screen.getByText(/orphaned_agents: unsupported/)).toBeInTheDocument();
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
    expect(screen.getByText('[WARNING] orphaned_agents')).toBeInTheDocument();
    expect(screen.getByText('5 orphaned sessions')).toBeInTheDocument();
  });
});

