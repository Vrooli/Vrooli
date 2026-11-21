import { screen } from '@testing-library/react';
import { create } from '@bufbuild/protobuf';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it } from 'vitest';
import { CpuExpansion } from './CpuExpansion';
import { MemoryExpansion } from './MemoryExpansion';
import { DiskExpansion } from './DiskExpansion';
import { NetworkExpansion } from './NetworkExpansion';
import { GpuExpansion } from './GpuExpansion';
import {
  CPUMetricsSchema,
  DiskInfoSchema,
  GPUMetricsSchema,
  MemoryMetricsSchema,
  NetworkMetricsSchema,
  ProcessInfoSchema,
  StorageIOInfoSchema,
  SwapInfoSchema,
} from '@vrooli/proto-types/system-monitor/v1/metrics/metrics_pb';

describe('metric expansion views', () => {
  it('renders CPU and memory process details and safe empty values', () => {
    render(<CpuExpansion details={create(CPUMetricsSchema, {
      topProcesses: [create(ProcessInfoSchema, { name: 'api', pid: 10, cpuPercent: 3.2 })],
      loadAverage: [1, 2, 3],
    })} />);
    expect(screen.getByText('api (10)')).toBeInTheDocument();
    expect(screen.getByText('Utilization:')).toBeInTheDocument();
    expect(screen.getAllByText('not yet sampled').length).toBeGreaterThan(0);

    render(<MemoryExpansion details={create(MemoryMetricsSchema, {
      topProcesses: [create(ProcessInfoSchema, { name: 'api', pid: 10, memoryMb: 44 })],
      swapUsage: create(SwapInfoSchema, { percent: 2 }),
      diskUsage: create(DiskInfoSchema, { percent: 70 }),
    })} />);
    expect(screen.getByText('Top Processes by Memory:')).toBeInTheDocument();
    expect(screen.getByText('44.0 MB')).toBeInTheDocument();
    render(<CpuExpansion details={create(CPUMetricsSchema)} />);
    expect(screen.getAllByText('not yet sampled').length).toBeGreaterThan(0);
  });

  it('renders typed CPU expansion states', () => {
    render(<CpuExpansion details={create(CPUMetricsSchema, {
      runQueueDepth: { state: { case: 'measured', value: 2 } },
      cpuPsiSomeAvg10: { state: { case: 'measured', value: 4 } },
      topProcesses: [],
    })} />);
    expect(screen.getByText('2.0 processes')).toBeInTheDocument();
    expect(screen.getByText('4.0% some')).toBeInTheDocument();
    expect(screen.getAllByText('not yet sampled').length).toBeGreaterThan(0);
  });

  it('renders disk capacity, IO, network states, and optional sections', () => {
    render(<DiskExpansion details={{
      diskUsage: create(DiskInfoSchema, { used: 5n * 1024n, total: 10n * 1024n, percent: 50 }),
      storageIO: create(StorageIOInfoSchema, { diskQueueDepth: 1.2, ioWaitPercent: 2, readMbPerSec: 3, writeMbPerSec: 4 }),
      lastUpdated: '2026-01-01T00:00:00Z',
    }} />);
    expect(screen.getByText('Total Capacity:')).toBeInTheDocument();
    expect(screen.getAllByText('5.00 KB')).toHaveLength(2);
    expect(screen.getByText('1.20')).toBeInTheDocument();
    expect(screen.getByText(/Updated/)).toBeInTheDocument();

    render(<NetworkExpansion details={create(NetworkMetricsSchema, {
      tcpStates: { established: 2, timeWait: 3, listen: 4, total: 9 },
      portUsage: { used: 2, total: 10 },
      networkStats: { dnsSuccessRate: 98 },
    })} />);
    expect(screen.getByText('Established:')).toBeInTheDocument();
    expect(screen.getByText('2 / 10')).toBeInTheDocument();
    expect(screen.getByText('98.0%')).toBeInTheDocument();
  });

  it('reports unavailable GPUs and renders GPU summaries and errors', () => {
    render(<GpuExpansion details={{}} />);
    expect(screen.getByText(/GPU metrics unavailable/)).toBeInTheDocument();
    render(<GpuExpansion details={{ metrics: create(GPUMetricsSchema, {
      summary: { deviceCount: 0, averageUtilizationPercent: 0, usedMemoryMb: 0, totalMemoryMb: 0, averageTemperatureC: 0 },
      devices: [],
      errors: ['driver error'],
      driverVersion: '550',
      primaryModel: 'Test GPU',
    }), lastUpdated: '2026-01-01T00:00:00Z' }} />);
    expect(screen.getByText('Driver Version:')).toBeInTheDocument();
    expect(screen.getByText('driver error')).toBeInTheDocument();
    expect(screen.getByText('No GPU devices detected.')).toBeInTheDocument();
  });
});
