import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { MemoryDetailView } from './MemoryDetailView';

vi.mock('./MetricDetailViews', () => ({
  DetailSection: ({ children }: { children: React.ReactNode }) => <section>{children}</section>,
  MetricDetailLayout: ({ children, onBack, headline }: { children: React.ReactNode; onBack: () => void; headline?: string }) => <div><button onClick={onBack}>back</button><div>{headline}</div>{children}</div>,
  MetricLineChart: ({ data, seriesLabel }: { data?: unknown[]; seriesLabel?: string }) => <div data-testid="chart" data-series-label={seriesLabel}>{JSON.stringify(data ?? [])}</div>,
}));
vi.mock('./MetricRenderHelpers', () => ({
  renderProcessTable: (processes: unknown) => <div>{processes ? 'processes' : 'no processes'}</div>,
}));

describe('MemoryDetailView', () => {
  it('shows kernel slab accounting and its retained trend without declaring a leak', () => {
    render(<MemoryDetailView metrics={null} onBack={vi.fn()}
      detailedMetrics={{ memoryDetails: { usage: 48, kernelMemory: {
        slabBytes: { state: { case: 'measured', value: 18 * 1024 ** 3 } },
        reclaimableBytes: { state: { case: 'measured', value: 5 * 1024 ** 3 } },
        unreclaimableBytes: { state: { case: 'measured', value: 13 * 1024 ** 3 } },
        hostTotalBytes: { state: { case: 'measured', value: 64 * 1024 ** 3 } }
      } } } as never}
      metricHistory={{ kernelSlab: [{ timestamp: 'now', value: 18 * 1024 ** 3 }], kernelUnreclaimable: [{ timestamp: 'now', value: 13 * 1024 ** 3 }] } as never} />);
    expect(screen.getByText('Slab total')).toBeInTheDocument();
    expect(screen.getByText('20.3% of physical RAM')).toBeInTheDocument();
    const chart = screen.getAllByTestId('chart').find(node => node.getAttribute('data-series-label') === 'kernel slab bytes');
    expect(chart).toHaveTextContent('13958643712');
    expect(screen.getByText(/not a leak diagnosis/i)).toBeInTheDocument();
  });

  it('preserves unsupported kernel slab state instead of displaying zero', () => {
    render(<MemoryDetailView metrics={null} onBack={vi.fn()} metricHistory={null}
      detailedMetrics={{ memoryDetails: { usage: 48, kernelMemory: {
        slabBytes: { state: { case: 'unsupportedReason', value: 'Linux slab accounting unavailable on darwin' } }
      } } } as never} />);
    expect(screen.getByText('Linux slab accounting unavailable on darwin')).toBeInTheDocument();
  });

  it('uses the server pressure verdict instead of declaring thrashing from swap traffic', () => {
    render(<MemoryDetailView metrics={null} metricHistory={null} onBack={vi.fn()}
      detailedMetrics={{ memoryDetails: {
        usage: 48, swapUsage: { used: 38n, total: 69n, percent: 55 },
        paging: { swapTrafficPagesPerSecond: { state: { case: 'measured', value: 5000 } } },
        memoryPressure: { level: 'normal', levelReason: 'PSI and swap-in are below elevated boundaries', levelSource: 'linux psi + swap-in rate',
          signals: { swap_in_bytes_per_second: { state: { case: 'measured', value: 0 } }, swap_out_bytes_per_second: { state: { case: 'measured', value: 20480000 } } } }
      } } as never} />);
    expect(screen.queryByText(/indicates thrashing/)).not.toBeInTheDocument();
    expect(screen.getByText('normal')).toBeInTheDocument();
    expect(screen.getByText('PSI and swap-in are below elevated boundaries')).toBeInTheDocument();
    expect(screen.getByText('19.5 MiB/s')).toBeInTheDocument();
  });

  it('shows critical pressure even when swap traffic is quiet', () => {
    render(<MemoryDetailView metrics={null} metricHistory={null} onBack={vi.fn()}
      detailedMetrics={{ memoryDetails: { usage: 48,
        paging: { swapTrafficPagesPerSecond: { state: { case: 'measured', value: 0 } } },
        memoryPressure: { level: 'critical', levelReason: 'memory PSI full avg60 8.00% >= 5.00%', signals: {} }
      } } as never} />);
    expect(screen.getByText('critical')).toBeInTheDocument();
    expect(screen.getByText('memory PSI full avg60 8.00% >= 5.00%')).toBeInTheDocument();
    expect(screen.queryByText('Swap residency is quiet.')).not.toBeInTheDocument();
  });

  it('preserves unavailable paging reasons without reporting a healthy pressure level', () => {
    render(<MemoryDetailView metrics={null} metricHistory={null} onBack={vi.fn()}
      detailedMetrics={{ memoryDetails: { usage: 48,
        paging: { swapTrafficPagesPerSecond: { state: { case: 'failedError', value: 'vmstat read failed' } } },
        memoryPressure: { level: 'unknown', levelReason: 'pressure signals unavailable', signals: {} }
      } } as never} />);
    expect(screen.getByText(/vmstat read failed/)).toBeInTheDocument();
    expect(screen.getByText('unknown')).toBeInTheDocument();
    expect(screen.getByText('pressure signals unavailable')).toBeInTheDocument();
    expect(screen.queryByText('normal')).not.toBeInTheDocument();
  });

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
    const base = { timestamp: { seconds: 0n, nanos: 0 }, memoryDetails: {
      usage: 55, paging: {}, fragmentation: { maxFreeOrder: { state: { case: 'unsupportedReason', value: 'Linux buddyinfo unavailable' } } }
    } } as never;
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
