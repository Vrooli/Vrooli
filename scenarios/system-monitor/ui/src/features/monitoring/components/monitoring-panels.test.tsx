import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { ProcessMonitor } from './ProcessMonitor';
import { InfrastructureMonitor } from './InfrastructureMonitor';
import type { ProcessMonitorData, InfrastructureMonitorData, SystemHealth } from '../../../types';

const mocks = vi.hoisted(() => ({ apiFetch: vi.fn() }));
vi.mock('../../../shared/api/apiFetch', () => ({ apiFetch: mocks.apiFetch }));

describe('monitoring panels', () => {
  it('expands process data, confirms termination, and handles cancellation', async () => {
    const toggle = vi.fn();
    mocks.apiFetch.mockResolvedValueOnce({ message: 'killed' });
    const data = { processHealth: {
      totalProcesses: 4,
      zombieProcesses: [{ pid: 11, name: 'zombie' }],
      highThreadCount: [{ pid: 12, name: 'worker', threads: 100 }],
      leakCandidates: [{ pid: 13, name: 'leaky', memoryMb: 55 }],
    } } as unknown as ProcessMonitorData;
    render(<ProcessMonitor data={data} isExpanded onToggle={toggle} />);
    expect(screen.getByText('Total Processes:')).toBeInTheDocument();
    expect(screen.getByText(/worker \(PID: 12\)/)).toBeInTheDocument();
    fireEvent.click(screen.getByTitle('Kill zombie process'));
    expect(screen.getByText('CONFIRM PROCESS TERMINATION')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'CANCEL' }));
    expect(screen.queryByText('CONFIRM PROCESS TERMINATION')).not.toBeInTheDocument();
    fireEvent.click(screen.getByTitle('Kill zombie process'));
    fireEvent.click(screen.getByRole('button', { name: 'TERMINATE PROCESS' }));
    await waitFor(() => { expect(mocks.apiFetch).toHaveBeenCalledWith('/processes/11/kill', expect.objectContaining({ method: 'POST' })); });
    fireEvent.click(screen.getByText('PROCESS MONITOR'));
    expect(toggle).toHaveBeenCalledOnce();
    cleanup();
    render(<ProcessMonitor data={null} collapsible={false} />);
    expect(screen.getByText('SCANNING SYSTEM...')).toBeInTheDocument();
  });

  it('renders infrastructure pools, queues, kernel limits, dependencies, and certificate states', () => {
    const toggle = vi.fn();
    const data = {
      databasePools: [{ name: 'db', active: 1, idle: 2, maxSize: 4, healthy: true }],
      httpClientPools: [{ name: 'http', active: 2, waiting: 1, maxSize: 4, healthy: false, leakRisk: 'high' }],
      messageQueues: { redisPubsub: { subscribers: 2, channels: 3 }, backgroundJobs: { pending: 1, active: 2, failed: 0 } },
    } as unknown as InfrastructureMonitorData;
    const health = {
      fileDescriptors: { used: 10, max: 100, percent: 10 },
      inotifyWatchers: { supported: true, watchesUsed: 2, watchesMax: 10, watchesPercent: 20, instancesUsed: 1, instancesMax: 5, instancesPercent: 20 },
      serviceDependencies: [{ name: 'api', status: 'healthy', latencyMs: 3 }],
      certificates: [{ domain: 'localhost', status: 'valid', daysToExpiry: 90 }],
    } as unknown as SystemHealth;
    render(<InfrastructureMonitor data={data} isExpanded onToggle={toggle} systemHealth={health} />);
    expect(screen.getByText('Database Pools:')).toBeInTheDocument();
    expect(screen.getByText('Active: 1 | Idle: 2 | Max: 4')).toBeInTheDocument();
    expect(screen.getByText('high')).toBeInTheDocument();
    expect(screen.getByText('Subscribers:')).toBeInTheDocument();
    expect(screen.getByText('File Descriptors')).toBeInTheDocument();
    expect(screen.getByText('Operational')).toBeInTheDocument();
    expect(screen.getByText('90 days')).toBeInTheDocument();
    fireEvent.click(screen.getByText('INFRASTRUCTURE MONITOR'));
    expect(toggle).toHaveBeenCalledOnce();
    cleanup();
    render(<InfrastructureMonitor data={null} isExpanded={false} onToggle={toggle} />);
    expect(screen.queryByText('Database Pools:')).not.toBeInTheDocument();
  });

  it('renders empty and unsupported infrastructure branches honestly', () => {
    const data = {
      databasePools: [{ name: 'db-down', active: 0, idle: 0, maxSize: 4, healthy: false }],
      httpClientPools: [{ name: 'http-ok', active: 0, waiting: 0, maxSize: 4, healthy: true, leakRisk: 'low' }],
      messageQueues: {},
    } as unknown as InfrastructureMonitorData;
    const health = {
      fileDescriptors: { used: 1, max: 10, percent: 10 },
      inotifyWatchers: { supported: false, watchesUsed: 0, watchesMax: 0, watchesPercent: 0, instancesUsed: 0, instancesMax: 0, instancesPercent: 0 },
      serviceDependencies: [],
      certificates: [
        { domain: 'urgent', status: 'expiring', daysToExpiry: 10 },
        { domain: 'soon', status: 'valid', daysToExpiry: 30 },
        { domain: 'safe', status: 'valid', daysToExpiry: 60 },
      ],
    } as unknown as SystemHealth;
    render(<InfrastructureMonitor data={data} isExpanded onToggle={vi.fn()} systemHealth={health} />);
    expect(screen.getByText('db-down')).toBeInTheDocument();
    expect(screen.getByText('Dependency telemetry unavailable.')).toBeInTheDocument();
    expect(screen.getByText('Inotify metrics unavailable on this host.')).toBeInTheDocument();
    expect(screen.getByText('10 days')).toBeInTheDocument();
    expect(screen.getByText('30 days')).toBeInTheDocument();
    expect(screen.getByText('60 days')).toBeInTheDocument();
    cleanup();
    render(<InfrastructureMonitor data={{} as InfrastructureMonitorData} isExpanded onToggle={vi.fn()} />);
    expect(screen.getByText('No certificate data reported.')).toBeInTheDocument();
  });
});
