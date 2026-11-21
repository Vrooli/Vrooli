import { screen } from '@testing-library/react';
import { create } from '@bufbuild/protobuf';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { DiskExpansion } from './DiskExpansion';
import { MemoryExpansion } from './MemoryExpansion';
import { NetworkExpansion } from './NetworkExpansion';
import { GpuExpansion } from './GpuExpansion';
import { CpuExpansion } from './CpuExpansion';
import { buildDiskUsageCard, renderProcessTable } from '../MetricRenderHelpers';
import { FindingsPanel } from '../../../capacity/components/FindingsPanel';
import { ClaimsTable } from '../../../capacity/components/ClaimsTable';
import {
  CapacityClaimSchema,
  CapacityFindingSchema,
} from '@vrooli/proto-types/system-monitor/v1/capacity/capacity_pb';
import {
  CPUMetricsSchema,
  DiskInfoSchema,
  GPUMetricsSchema,
  GPUDeviceMetricsSchema,
  MemoryMetricsSchema,
  NetworkMetricsSchema,
  ProcessInfoSchema,
  StorageIOInfoSchema,
  SwapInfoSchema,
} from '@vrooli/proto-types/system-monitor/v1/metrics/metrics_pb';

vi.mock('../GpuDeviceCard', () => ({ GpuDeviceCard: ({ device }: { device: { index: number } }) => <div>gpu device {device.index}</div> }));

describe('metric expansion variants', () => {
  it('renders disk, memory, and network optional details', () => {
    const { rerender } = render(<DiskExpansion details={{
      diskUsage: create(DiskInfoSchema, { total: 1000n, used: 400n, percent: 40 }),
      storageIO: create(StorageIOInfoSchema, { diskQueueDepth: 1, ioWaitPercent: 2, readMbPerSec: 3, writeMbPerSec: 4 }),
      lastUpdated: '2026-01-01T00:00:00Z',
    }} />);
    expect(screen.getByText('Free:')).toBeInTheDocument();
    expect(screen.getByText(/Disk Queue Depth/)).toBeInTheDocument();
    rerender(<DiskExpansion details={{ diskUsage: create(DiskInfoSchema) }} />);
    expect(screen.getAllByText(/—/).length).toBeGreaterThan(0);

    rerender(<MemoryExpansion details={create(MemoryMetricsSchema, {
      topProcesses: [create(ProcessInfoSchema, { pid: 1, name: 'one', memoryMb: 3 })],
      swapUsage: create(SwapInfoSchema, { percent: 4 }),
      diskUsage: create(DiskInfoSchema, { percent: 5 }),
    })} />);
    expect(screen.getByText('one (1)')).toBeInTheDocument();
    rerender(<MemoryExpansion details={create(MemoryMetricsSchema)} />);
    expect(screen.getAllByText(/—/).length).toBeGreaterThan(0);

    rerender(<NetworkExpansion details={create(NetworkMetricsSchema, {
      tcpStates: { established: 1, timeWait: 2, listen: 3, total: 4 },
      portUsage: { used: 1, total: 2 },
      networkStats: { dnsSuccessRate: 99 },
    })} />);
    expect(screen.getByText('Connection States:')).toBeInTheDocument();
    expect(screen.getByText('1 / 2')).toBeInTheDocument();
    rerender(<NetworkExpansion details={create(NetworkMetricsSchema)} />);
    expect(screen.queryByText('Connection States:')).not.toBeInTheDocument();
    rerender(<NetworkExpansion details={create(NetworkMetricsSchema, { tcpStates: {}, networkStats: {}, portUsage: {} })} />);
    expect(screen.getByText('0 / 0')).toBeInTheDocument();
    expect(screen.getByText('0.0%')).toBeInTheDocument();
  });

  it('renders CPU process details and all metric states', () => {
    render(<CpuExpansion details={create(CPUMetricsSchema, {
      topProcesses: [{ pid: 7, name: 'worker', cpuPercent: 12.5 }],
      usageState: { state: { case: 'measured', value: 42 } },
      modeBreakdown: { steal: { state: { case: 'notYetSampledReason', value: 'booting' } } },
      runQueueDepth: { state: { case: 'unsupportedReason', value: 'kernel unavailable' }, provenance: 'kernel' },
    })} />);
    expect(screen.getByText('worker (7)')).toBeInTheDocument();
    expect(screen.getByText('42.0')).toBeInTheDocument();
    expect(screen.getByText('waiting for next sample: booting')).toBeInTheDocument();
    expect(screen.getByText('unsupportedReason: kernel unavailable (kernel)')).toBeInTheDocument();
    expect(screen.getAllByText('not yet sampled')).toHaveLength(2);
  });

  it('renders capacity findings and claims in empty and populated states', () => {
    const { rerender } = render(<FindingsPanel available={false} findings={[]} />);
    expect(screen.getByText(/Capacity sensing is unavailable/)).toBeInTheDocument();
    rerender(<FindingsPanel available findings={[]} />);
    expect(screen.getByText(/No GPU consumers/)).toBeInTheDocument();
    rerender(<FindingsPanel available findings={[
      create(CapacityFindingSchema, { ownerId: 'owner', pid: 1, class: 'mystery', severity: 'info', observedBytes: 100n, message: '' }),
      create(CapacityFindingSchema, { ownerId: 'warn', pid: 2, class: 'unclaimed', severity: 'warn', observedBytes: 100n, message: 'explicit' }),
    ]} />);
    expect(screen.getByText('mystery')).toBeInTheDocument();
    expect(screen.getByText('explicit')).toBeInTheDocument();

    rerender(<ClaimsTable claims={[]} />);
    expect(screen.getByText('No active capacity claims.')).toBeInTheDocument();
    rerender(<ClaimsTable claims={[create(CapacityClaimSchema, {
      claimId: 'c', ownerKind: 'agent', ownerId: 'a', resourceKind: 'gpu', gpuIndex: 0,
      status: 'active', priorityTier: 'custom', activityState: '', amountBytes: 100n, protected: true,
    })]} />);
    expect(screen.getByLabelText('protected')).toBeInTheDocument();
    expect(screen.getByText('custom')).toBeInTheDocument();
  });

  it('renders GPU expansion and shared render helper fallbacks', () => {
    const { rerender } = render(<GpuExpansion details={{}} />);
    expect(screen.getByText(/GPU metrics unavailable/)).toBeInTheDocument();
    rerender(<GpuExpansion details={{ metrics: create(GPUMetricsSchema, {
      summary: { deviceCount: 1, averageUtilizationPercent: 12, usedMemoryMb: 1, totalMemoryMb: 2, averageTemperatureC: 0 },
      driverVersion: 'x', primaryModel: 'model', errors: ['warning'],
      devices: [create(GPUDeviceMetricsSchema, { index: 0, uuid: 'u' })],
    }), lastUpdated: '2026-01-01T00:00:00Z' }} />);
    expect(screen.getByText('Driver Version:')).toBeInTheDocument();
    expect(screen.getByText('warning')).toBeInTheDocument();
    expect(screen.getByText('gpu device 0')).toBeInTheDocument();
    rerender(<GpuExpansion details={{ metrics: create(GPUMetricsSchema, { devices: [] }) }} />);
    expect(screen.getByText('No GPU devices detected.')).toBeInTheDocument();

    rerender(buildDiskUsageCard(undefined, { title: 'Custom disk' }));
    expect(screen.getByText('Custom disk')).toBeInTheDocument();
    rerender(buildDiskUsageCard(create(DiskInfoSchema, { total: 100n, used: 25n, percent: 25 }), { subtitle: 'custom subtitle' }));
    expect(screen.getByText('custom subtitle')).toBeInTheDocument();
    rerender(renderProcessTable([{ name: 'p', pid: 1 }], 'CPU', process => process.cpuPercent));
    expect(screen.getByText('—')).toBeInTheDocument();
  });
});
