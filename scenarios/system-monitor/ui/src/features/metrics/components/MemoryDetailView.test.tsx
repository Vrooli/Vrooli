import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { MemoryDetailView } from './MemoryDetailView';

vi.mock('./MetricDetailViews', () => ({
  MetricDetailLayout: ({ children, onBack, headline }: { children: React.ReactNode; onBack: () => void; headline?: string }) => <div><button onClick={onBack}>back</button><div>{headline}</div>{children}</div>,
  MetricLineChart: ({ data, seriesLabel }: { data?: unknown[]; seriesLabel?: string }) => <div data-testid="chart" data-series-label={seriesLabel}>{JSON.stringify(data ?? [])}</div>,
}));
vi.mock('./MetricRenderHelpers', () => ({
  renderProcessTable: (processes: unknown) => <div>{processes ? 'processes' : 'no processes'}</div>,
}));

describe('MemoryDetailView', () => {
  it('renders measured details and optional sections', () => {
    const onBack = vi.fn();
    render(<MemoryDetailView
      metrics={{ memory: { state: { case: 'measured', value: 44 } } } as never}
      detailedMetrics={{ timestamp: { seconds: 0n, nanos: 0 }, memoryDetails: {
        usage: 55, swapUsage: { used: 1024, total: 2048, percent: 50 }, topProcesses: [{}],
      } } as never}
      metricHistory={{ memory: [{ timestamp: 'now', value: 55 }] } as never}
      onBack={onBack}
    />);
    expect(screen.getByText('55.0% used')).toBeInTheDocument();
    expect(screen.getByText('Swap Used')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'back' }));
    expect(onBack).toHaveBeenCalledOnce();
  });

  it('renders honest unavailable states', () => {
    render(<MemoryDetailView metrics={null} detailedMetrics={null} metricHistory={null} onBack={vi.fn()} />);
    expect(screen.getByText('Utilization not measured')).toBeInTheDocument();
    expect(screen.getByText('Swap metrics unavailable.')).toBeInTheDocument();
    expect(screen.getAllByText('no processes')).toHaveLength(2);
  });

  it('renders the fragmentation trend only for measured state and explains unsupported state', () => {
    const base: any = { timestamp: { seconds: 0n, nanos: 0 }, memoryDetails: {
      usage: 55, paging: {}, fragmentation: { maxFreeOrder: { state: { case: 'unsupportedReason', value: 'Linux buddyinfo unavailable' } } }
    } };
    const { rerender } = render(<MemoryDetailView metrics={null} detailedMetrics={base} metricHistory={{ fragmentation: [] } as never} onBack={vi.fn()} />);
    expect(screen.getByText('Linux buddyinfo unavailable')).toBeInTheDocument();
    expect(screen.queryByText('Memory Fragmentation')).not.toBeInTheDocument();
    rerender(<MemoryDetailView metrics={null} detailedMetrics={{ ...base, memoryDetails: { ...base.memoryDetails, fragmentation: { maxFreeOrder: { state: { case: 'measured', value: 6 } }, buddyinfo: { Normal: '[0 0 0 0 0 0 1]' } } } }} metricHistory={{ fragmentation: [{ timestamp: 'now', value: 6 }] } as never} onBack={vi.fn()} />);
    expect(screen.getByText('Highest free order')).toBeInTheDocument();
  });

  it('passes measured major-fault points to the dedicated faults chart', () => {
    render(<MemoryDetailView
      metrics={null}
      detailedMetrics={{ memoryDetails: { usage: 55, fragmentation: {} } } as never}
      metricHistory={{ majorFaults: [{ timestamp: '2026-08-22T15:53:11Z', value: 180.8 }] } as never}
      onBack={vi.fn()}
    />);
    const chart = screen.getAllByTestId('chart').find(node => node.getAttribute('data-series-label') === 'major faults');
    expect(chart).toHaveTextContent('180.8');
  });
});
