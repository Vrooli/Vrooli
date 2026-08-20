import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { MemoryDetailView } from './MemoryDetailView';

vi.mock('./MetricDetailViews', () => ({
  MetricDetailLayout: ({ children, onBack, headline }: { children: React.ReactNode; onBack: () => void; headline?: string }) => <div><button onClick={onBack}>back</button><div>{headline}</div>{children}</div>,
  MetricLineChart: () => <div data-testid="chart" />,
}));
vi.mock('./MetricRenderHelpers', () => ({
  renderGrowthPatterns: (patterns: unknown) => <div>{patterns ? 'growth' : 'no growth'}</div>,
  renderProcessTable: (processes: unknown) => <div>{processes ? 'processes' : 'no processes'}</div>,
}));

describe('MemoryDetailView', () => {
  it('renders measured details and optional sections', () => {
    const onBack = vi.fn();
    render(<MemoryDetailView
      metrics={{ memory: { state: { case: 'measured', value: 44 } } } as never}
      detailedMetrics={{ timestamp: { seconds: 0n, nanos: 0 }, memoryDetails: {
        usage: 55, swapUsage: { used: 1024, total: 2048, percent: 50 }, growthPatterns: [{}], topProcesses: [{}],
      } } as never}
      metricHistory={{ memory: [{ timestamp: 'now', value: 55 }] } as never}
      onBack={onBack}
    />);
    expect(screen.getByText('55.0% used')).toBeInTheDocument();
    expect(screen.getByText('Swap Used')).toBeInTheDocument();
    expect(screen.getByText('growth')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'back' }));
    expect(onBack).toHaveBeenCalledOnce();
  });

  it('renders honest unavailable states', () => {
    render(<MemoryDetailView metrics={null} detailedMetrics={null} metricHistory={null} onBack={vi.fn()} />);
    expect(screen.getByText('Utilization not measured')).toBeInTheDocument();
    expect(screen.getByText('Swap metrics unavailable.')).toBeInTheDocument();
    expect(screen.getByText('no growth')).toBeInTheDocument();
    expect(screen.getByText('no processes')).toBeInTheDocument();
  });
});
