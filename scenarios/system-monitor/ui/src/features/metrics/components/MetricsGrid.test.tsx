import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { describe, expect, it, vi } from 'vitest';
import type { DetailedMetrics, MetricsResponse } from '../../../types';
import { MetricsGrid } from './MetricsGrid';

vi.mock('./MetricCard', () => ({
  MetricCard: ({ label, type, onToggle, onOpenDetails }: { label: string; type: string; onToggle: () => void; onOpenDetails?: () => void }) => (
    <article>
      <span>{label}</span>
      <button type="button" onClick={onToggle}>toggle-{type}</button>
      {onOpenDetails && <button type="button" onClick={onOpenDetails}>open-{type}</button>}
    </article>
  ),
}));

const timestamp = timestampFromDate(new Date('2026-01-01T00:00:00Z'));

describe('MetricsGrid', () => {
  it('passes measured metrics, details, and merged disk history to every card', () => {
    const onToggleCard = vi.fn();
    const onOpenDetail = vi.fn();
    const metrics = {
      cpu: { state: { case: 'measured', value: 10 } },
      memory: { state: { case: 'measured', value: 20 } },
      gpu: { state: { case: 'measured', value: 30 } },
      disk: { state: { case: 'measured', value: 80 } },
      connections: { state: { case: 'measured', value: 5 } },
    } as unknown as MetricsResponse;
    const detailedMetrics = {
      timestamp,
      memoryDetails: { diskUsage: { used: BigInt(8), total: BigInt(10), percent: 80 } },
      gpuDetails: { devices: [], errors: [] },
    } as unknown as DetailedMetrics;
    const metricHistory = {
      windowSeconds: 3600,
      diskRead: [{ timestamp: '2026-01-01T00:00:00Z', value: 1 }, { timestamp: '', value: 2 }],
      diskWrite: [{ timestamp: '2026-01-01T00:00:00Z', value: 3 }],
      cpu: [], memory: [], network: [], gpu: [],
    };
    render(<MetricsGrid metrics={metrics} detailedMetrics={detailedMetrics} expandedCards={new Set()} onToggleCard={onToggleCard} metricHistory={metricHistory} storageIO={null} onOpenDetail={onOpenDetail} />);
    expect(screen.getByText('CPU USAGE')).toBeInTheDocument();
    expect(screen.getByText('NETWORK & CONNECTIONS')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'toggle-cpu' }));
    fireEvent.click(screen.getByRole('button', { name: 'open-disk' }));
    expect(onToggleCard).toHaveBeenCalledWith('cpu');
    expect(onOpenDetail).toHaveBeenCalledWith('disk');
  });

  it('handles absent telemetry and empty history without inventing values', () => {
    render(<MetricsGrid metrics={null} detailedMetrics={null} expandedCards={new Set()} onToggleCard={vi.fn()} metricHistory={null} onOpenDetail={vi.fn()} />);
    expect(screen.getAllByRole('article')).toHaveLength(5);
  });
});
