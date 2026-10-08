import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it } from 'vitest';
import type { GPUDeviceMetrics } from '../../../types';
import { GpuDeviceCard } from './GpuDeviceCard';

const device = {
  index: 0, uuid: 'gpu-0', name: 'Test GPU', utilizationPercent: 42, memoryUsedMb: 512, memoryTotalMb: 1024,
  memoryUtilizationPercent: 50, temperatureC: 55, fanSpeedPercent: 40, powerDrawW: 120, smClockMhz: 1500, memoryClockMhz: 7000,
  processes: [{ pid: 10, processName: 'api', memoryUsedMb: 100 }],
} as unknown as GPUDeviceMetrics;

describe('GpuDeviceCard', () => {
  it('renders detail and compact expansion variants with optional stats', () => {
    const { rerender } = render(<GpuDeviceCard device={device} />);
    expect(screen.getByText('Test GPU (GPU 0)')).toBeInTheDocument();
    expect(screen.getByText('Processes')).toBeInTheDocument();
    expect(screen.getByText('api (10)')).toBeInTheDocument();
    rerender(<GpuDeviceCard device={{ ...device, temperatureC: undefined, fanSpeedPercent: undefined, powerDrawW: undefined, smClockMhz: undefined, memoryClockMhz: undefined, utilizationPercent: undefined, processes: [] }} variant="expansion" />);
    expect(screen.queryByText('Active Processes')).not.toBeInTheDocument();
    expect(screen.getByText('—%')).toBeInTheDocument();
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
  });
});
