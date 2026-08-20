import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { DiskExpansion } from './DiskExpansion';
import { MemoryExpansion } from './MemoryExpansion';
import { NetworkExpansion } from './NetworkExpansion';
import { GpuExpansion } from './GpuExpansion';
import { buildDiskUsageCard, renderGrowthPatterns, renderProcessTable } from '../MetricRenderHelpers';
import { FindingsPanel } from '../../../capacity/components/FindingsPanel';
import { ClaimsTable } from '../../../capacity/components/ClaimsTable';

vi.mock('../GpuDeviceCard', () => ({ GpuDeviceCard: ({ device }: { device: { index: number } }) => <div>gpu device {device.index}</div> }));

describe('metric expansion variants', () => {
  it('renders disk, memory, and network optional details', () => {
    const { rerender } = render(<DiskExpansion details={{ diskUsage: { total: 1000, used: 400, percent: 40 }, storageIO: { diskQueueDepth: 1, ioWaitPercent: 2, readMbPerSec: 3, writeMbPerSec: 4 }, lastUpdated: '2026-01-01T00:00:00Z' } as never} />);
    expect(screen.getByText('Free:')).toBeInTheDocument();
    expect(screen.getByText(/Disk Queue Depth/)).toBeInTheDocument();
    rerender(<DiskExpansion details={{ diskUsage: {} } as never} />);
    expect(screen.getAllByText(/—/).length).toBeGreaterThan(0);

    rerender(<MemoryExpansion details={{ topProcesses: [{ pid: 1, name: 'one', memoryMb: 3 }], swapUsage: { percent: 4 }, diskUsage: { percent: 5 }, growthPatterns: [{ process: 'one', riskLevel: 'high', growthMbPerHour: 6 }] } as never} />);
    expect(screen.getByText('one (1)')).toBeInTheDocument();
    expect(screen.getByText(/Memory Growth Patterns/)).toBeInTheDocument();
    rerender(<MemoryExpansion details={{} as never} />);
    expect(screen.getAllByText(/—/).length).toBeGreaterThan(0);

    rerender(<NetworkExpansion details={{ tcpStates: { established: 1, timeWait: 2, listen: 3, total: 4 }, portUsage: { used: 1, total: 2 }, networkStats: { dnsSuccessRate: 99 } } as never} />);
    expect(screen.getByText('Connection States:')).toBeInTheDocument();
    expect(screen.getByText('1 / 2')).toBeInTheDocument();
    rerender(<NetworkExpansion details={{} as never} />);
    expect(screen.queryByText('Connection States:')).not.toBeInTheDocument();
    rerender(<NetworkExpansion details={{ tcpStates: {}, networkStats: {}, portUsage: {} } as never} />);
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
  });

  it('renders capacity findings and claims in empty and populated states', () => {
    const { rerender } = render(<FindingsPanel available={false} findings={[]} />);
    expect(screen.getByText(/Capacity sensing is unavailable/)).toBeInTheDocument();
    rerender(<FindingsPanel available findings={[]} />);
    expect(screen.getByText(/No GPU consumers/)).toBeInTheDocument();
    rerender(<FindingsPanel available findings={[{ ownerId: 'owner', pid: 1, class: 'mystery', severity: 'info', observedBytes: 100, message: '' } as never, { ownerId: 'warn', pid: 2, class: 'unclaimed', severity: 'warn', observedBytes: 100, message: 'explicit' } as never]} />);
    expect(screen.getByText('mystery')).toBeInTheDocument();
    expect(screen.getByText('explicit')).toBeInTheDocument();

    rerender(<ClaimsTable claims={[]} />);
    expect(screen.getByText('No active capacity claims.')).toBeInTheDocument();
    rerender(<ClaimsTable claims={[{ claimId: 'c', ownerKind: 'agent', ownerId: 'a', resourceKind: 'gpu', gpuIndex: 0, status: 'active', priorityTier: 'custom', activityState: '', amountBytes: 100, protected: true } as never]} />);
    expect(screen.getByLabelText('protected')).toBeInTheDocument();
    expect(screen.getByText('custom')).toBeInTheDocument();
  });

  it('renders GPU expansion and shared render helper fallbacks', () => {
    const { rerender } = render(<GpuExpansion details={{} as never} />);
    expect(screen.getByText(/GPU metrics unavailable/)).toBeInTheDocument();
    rerender(<GpuExpansion details={{ metrics: { summary: { deviceCount: 1, averageUtilizationPercent: 12, usedMemoryMb: 1, totalMemoryMb: 2, averageTemperatureC: 0 }, driverVersion: 'x', primaryModel: 'model', errors: ['warning'], devices: [{ index: 0, uuid: 'u' }] }, lastUpdated: '2026-01-01T00:00:00Z' } as never} />);
    expect(screen.getByText('Driver Version:')).toBeInTheDocument();
    expect(screen.getByText('warning')).toBeInTheDocument();
    expect(screen.getByText('gpu device 0')).toBeInTheDocument();
    rerender(<GpuExpansion details={{ metrics: { devices: [] } } as never} />);
    expect(screen.getByText('No GPU devices detected.')).toBeInTheDocument();

    rerender(buildDiskUsageCard(undefined, { title: 'Custom disk' }));
    expect(screen.getByText('Custom disk')).toBeInTheDocument();
    rerender(buildDiskUsageCard({ total: 100, used: 25, percent: 25 } as never, { subtitle: 'custom subtitle' }));
    expect(screen.getByText('custom subtitle')).toBeInTheDocument();
    rerender(renderProcessTable([{ name: 'p', pid: 1 }], 'CPU', process => process.cpuPercent));
    expect(screen.getByText('—')).toBeInTheDocument();
    rerender(renderGrowthPatterns([{ process: 'p', growthMbPerHour: 1, riskLevel: 'high' }]));
    expect(screen.getByText(/1.0 MB\/hr/)).toBeInTheDocument();
  });
});
