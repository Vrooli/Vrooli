import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it } from 'vitest';
import { CpuExpansion } from './CpuExpansion';
import { MemoryExpansion } from './MemoryExpansion';
import { DiskExpansion } from './DiskExpansion';
import { NetworkExpansion } from './NetworkExpansion';
import { GpuExpansion } from './GpuExpansion';
import type { CPUMetrics, MemoryMetrics, DiskCardDetails, NetworkMetrics, GPUCardDetails } from '../../../../types';

describe('metric expansion views', () => {
  it('renders CPU and memory process details and safe empty values', () => {
    render(<CpuExpansion details={{ topProcesses: [{ name: 'api', pid: 10, cpuPercent: 3.2 }], loadAverage: [1, 2, 3], contextSwitches: 1234, totalGoroutines: 9 } as unknown as CPUMetrics} />);
    expect(screen.getByText('api (10)')).toBeInTheDocument();
    expect(screen.getByText('1.00, 2.00, 3.00')).toBeInTheDocument();
    expect(screen.getByText('1,234')).toBeInTheDocument();

    render(<MemoryExpansion details={{ topProcesses: [{ name: 'api', pid: 10, memoryMb: 44 }], swapUsage: { percent: 2 }, diskUsage: { percent: 70 }, growthPatterns: [{ process: 'api', riskLevel: 'high', growthMbPerHour: 3 }] } as unknown as MemoryMetrics} />);
    expect(screen.getByText('Top Processes by Memory:')).toBeInTheDocument();
    expect(screen.getByText('44.0 MB')).toBeInTheDocument();
    expect(screen.getByText(/3.0 MB\/hr/)).toBeInTheDocument();
    render(<CpuExpansion details={{} as CPUMetrics} />);
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
  });

  it('renders disk capacity, IO, network states, and optional sections', () => {
    render(<DiskExpansion details={{ diskUsage: { used: 5 * 1024, total: 10 * 1024, percent: 50 }, storageIO: { diskQueueDepth: 1.2, ioWaitPercent: 2, readMbPerSec: 3, writeMbPerSec: 4 }, lastUpdated: '2026-01-01T00:00:00Z' } as unknown as DiskCardDetails} />);
    expect(screen.getByText('Total Capacity:')).toBeInTheDocument();
    expect(screen.getAllByText('5.00 KB')).toHaveLength(2);
    expect(screen.getByText('1.20')).toBeInTheDocument();
    expect(screen.getByText(/Updated/)).toBeInTheDocument();

    render(<NetworkExpansion details={{ tcpStates: { established: 2, timeWait: 3, listen: 4, total: 9 }, portUsage: { used: 2, total: 10 }, networkStats: { dnsSuccessRate: 98 } } as unknown as NetworkMetrics} />);
    expect(screen.getByText('Established:')).toBeInTheDocument();
    expect(screen.getByText('2 / 10')).toBeInTheDocument();
    expect(screen.getByText('98.0%')).toBeInTheDocument();
  });

  it('reports unavailable GPUs and renders GPU summaries and errors', () => {
    render(<GpuExpansion details={{} as GPUCardDetails} />);
    expect(screen.getByText(/GPU metrics unavailable/)).toBeInTheDocument();
    render(<GpuExpansion details={{ metrics: { summary: { deviceCount: 0, averageUtilizationPercent: 0, usedMemoryMb: 0, totalMemoryMb: 0, averageTemperatureC: 0 }, devices: [], errors: ['driver error'], driverVersion: '550', primaryModel: 'Test GPU' }, lastUpdated: '2026-01-01T00:00:00Z' } as unknown as GPUCardDetails} />);
    expect(screen.getByText('Driver Version:')).toBeInTheDocument();
    expect(screen.getByText('driver error')).toBeInTheDocument();
    expect(screen.getByText('No GPU devices detected.')).toBeInTheDocument();
  });
});
