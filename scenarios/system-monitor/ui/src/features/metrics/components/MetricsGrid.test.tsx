import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { describe, expect, it, vi } from 'vitest';
import type { DetailedMetrics, MetricsResponse } from '../../../types';
import { combineDiskRates, MetricsGrid } from './MetricsGrid';

vi.mock('./MetricCard', () => ({
  MetricCard: ({ label, type, value, unit, secondaryText, emptyHistoryMessage, currentUnavailableReason, onToggle, onOpenDetails }: { label: string; type: string; value?: number; unit: string; secondaryText?: string; emptyHistoryMessage?: string; currentUnavailableReason?: string; onToggle: () => void; onOpenDetails?: () => void }) => (
    <article>
      <span>{label}</span>
      {type === 'disk' && <span data-testid="disk-summary">{value} {unit} {secondaryText} {emptyHistoryMessage} {currentUnavailableReason}</span>}
      <button type="button" onClick={onToggle}>toggle-{type}</button>
      {onOpenDetails && <button type="button" onClick={onOpenDetails}>open-{type}</button>}
    </article>
  ),
}));

const timestamp = timestampFromDate(new Date('2026-01-01T00:00:00Z'));

describe('MetricsGrid', () => {
  it('does not present the previous throughput as current after refresh failure', () => {
    const storageIO = { readRate: { state: { case: 'measured', value: 1 } }, writeRate: { state: { case: 'measured', value: 2 } } } as never;
    render(<MetricsGrid metrics={null} detailedMetrics={null} expandedCards={new Set()} onToggleCard={vi.fn()} metricHistory={null} onOpenDetail={vi.fn()} storageIO={storageIO} infrastructureUnavailable />);
    expect(screen.getByTestId('disk-summary')).toHaveTextContent('Disk throughput refresh failed');
    expect(screen.getByTestId('disk-summary')).not.toHaveTextContent('3 MB/s');
  });

  it('explains why a disk throughput sample is unavailable', () => {
    const storageIO = { readRate: { state: { case: 'notYetSampledReason', value: 'A second disk counter sample is required' } }, writeRate: { state: { case: 'notYetSampledReason', value: 'A second disk counter sample is required' } } } as never;
    render(<MetricsGrid metrics={null} detailedMetrics={null} expandedCards={new Set()} onToggleCard={vi.fn()} metricHistory={null} onOpenDetail={vi.fn()} storageIO={storageIO} />);
    expect(screen.getByTestId('disk-summary')).toHaveTextContent('A second disk counter sample is required');
  });
  it('does not add read and write rates from different observations', () => {
    const later = timestampFromDate(new Date('2026-01-01T00:00:01Z'));
    const storageIO = { readRate: { observedAt: timestamp, state: { case: 'measured', value: 1 } }, writeRate: { observedAt: later, state: { case: 'measured', value: 2 } } } as never;
    render(<MetricsGrid metrics={null} detailedMetrics={null} expandedCards={new Set()} onToggleCard={vi.fn()} metricHistory={null} onOpenDetail={vi.fn()} storageIO={storageIO} />);
    expect(screen.getByTestId('disk-summary')).not.toHaveTextContent('3 MB/s');
  });
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
    render(<MetricsGrid metrics={metrics} detailedMetrics={detailedMetrics} expandedCards={new Set()} onToggleCard={onToggleCard} metricHistory={metricHistory} storageIO={{ readRate: { observedAt: timestamp, state: { case: 'measured', value: 1 } }, writeRate: { observedAt: timestamp, state: { case: 'measured', value: 3 } } } as never} onOpenDetail={onOpenDetail} />);
    expect(screen.getByText('CPU USAGE')).toBeInTheDocument();
    expect(screen.getByText('NETWORK & CONNECTIONS')).toBeInTheDocument();
    expect(screen.getByTestId('disk-summary')).toHaveTextContent('4 MB/s 80.0% space used');
    fireEvent.click(screen.getByRole('button', { name: 'toggle-cpu' }));
    fireEvent.click(screen.getByRole('button', { name: 'open-disk' }));
    expect(onToggleCard).toHaveBeenCalledWith('cpu');
    expect(onOpenDetail).toHaveBeenCalledWith('disk');
  });

  it('combines disk rates only at matching timestamps with both measurements', () => {
    expect(combineDiskRates(
      [{ timestamp: 'a', value: 6 }, { timestamp: 'b', value: 8 }, { timestamp: 'c', value: 0 }],
      [{ timestamp: 'c', value: 2 }, { timestamp: 'a', value: 12 }, { timestamp: 'd', value: 4 }],
    )).toEqual([{ timestamp: 'a', value: 18 }, { timestamp: 'c', value: 2 }]);
  });

  it('preserves a missing-observation break in the combined disk series', () => {
    expect(combineDiskRates([{ timestamp: 'a', value: 1, gapBefore: true }], [{ timestamp: 'a', value: 2 }])).toEqual([{ timestamp: 'a', value: 3, gapBefore: true }]);
  });

  it('handles absent telemetry and empty history without inventing values', () => {
    render(<MetricsGrid metrics={null} detailedMetrics={null} expandedCards={new Set()} onToggleCard={vi.fn()} metricHistory={null} onOpenDetail={vi.fn()} />);
    expect(screen.getAllByRole('article')).toHaveLength(5);
    expect(screen.getByTestId('disk-summary')).toHaveTextContent('Disk throughput is unavailable from the current source');
  });
});

describe('card alert badges', () => {
  it('count only active server alerts whose rule belongs to the card', async () => {
    const { cardAlertCount } = await import('./MetricsGrid');
    const alerts = [
      { id: '1', timestamp: '', severity: 'warning' as const, category: 'tcp_connections', message: 'TCP: climbing', resolved: false },
      { id: '2', timestamp: '', severity: 'critical' as const, category: 'memory_pressure', message: 'pressure', resolved: false },
      { id: '3', timestamp: '', severity: 'warning' as const, category: 'memory_usage', message: 'mem', resolved: true },
    ];
    expect(cardAlertCount('network', alerts)).toBe(1);
    expect(cardAlertCount('memory', alerts)).toBe(1);
    expect(cardAlertCount('cpu', alerts)).toBe(0);
    // A busy host with many connections but no server alert shows no badge.
    expect(cardAlertCount('network', [])).toBe(0);
  });
});
