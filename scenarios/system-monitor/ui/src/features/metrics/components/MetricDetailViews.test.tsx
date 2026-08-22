import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { cloneElement, type ReactElement } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';

vi.mock('recharts', () => {
  const Box = ({ children, ...props }: { children?: React.ReactNode } & Record<string, unknown>) => <div {...props}>{children}</div>;
  return {
    ResponsiveContainer: Box,
    ComposedChart: Box,
    CartesianGrid: () => <div data-testid="grid" />,
    XAxis: () => <div data-testid="x-axis" />,
    YAxis: () => <div data-testid="y-axis" />,
    Area: ({ dataKey, hide }: { dataKey: string; hide?: boolean }) => <div data-testid={`area-${dataKey}`} data-hidden={String(hide)} />,
    Line: ({ dataKey, hide }: { dataKey: string; hide?: boolean }) => <div data-testid={`line-${dataKey}`} data-hidden={String(hide)} />,
    Tooltip: ({ content, cursor }: { content: React.ReactNode; cursor: React.ReactNode }) => <div data-testid="tooltip">
      {cloneElement(content as ReactElement, { active: true, payload: [{ value: 42, name: 'CPU', color: 'red', dataKey: 'cpu' }, { value: Number.NaN, name: 'Memory', color: 'blue', dataKey: 'memory' }], label: 'not-a-date' })}
      {cloneElement(cursor as ReactElement, { points: [{ x: 10 }], height: 20 })}
      {cloneElement(cursor as ReactElement, { points: [], height: 20 })}
    </div>,
    // The chart renders its own legend via `content`, so the mock renders that
    // rather than reimplementing recharts' inferred-payload legend.
    Legend: ({ content }: { content: () => React.ReactNode }) => <>{typeof content === 'function' ? content() : content}</>,
  };
});

describe('MetricDetailViews', () => {
  it('renders the shared detail layout and back action', () => {
    const onBack = vi.fn();
    render(<MetricDetailLayout title="CPU" icon={<span>icon</span>} headline="42%" subhead="Last sample" onBack={onBack}><div>content</div></MetricDetailLayout>);
    expect(screen.getAllByText('CPU').length).toBeGreaterThan(0);
    expect(screen.getByText('42%')).toBeInTheDocument();
    expect(screen.getByText('Last sample')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Back To Dashboard/ }));
    expect(onBack).toHaveBeenCalledOnce();
  });

  // Loading, empty, and error must stay visually and semantically distinct. A
  // single shared message is what allowed 16 hours of metrics-persistence
  // failure to read as a slow load.
  it('renders an honest empty chart state when the window has no samples', () => {
    render(<MetricLineChart data={[]} lines={[{ dataKey: 'cpu', name: 'CPU', color: 'red' }]} unit="%" seriesLabel="CPU" />);
    expect(screen.getByText('No data in this window')).toBeInTheDocument();
    expect(screen.getByText(/No CPU samples were recorded/)).toBeInTheDocument();
    expect(screen.queryByLabelText('Loading chart data')).not.toBeInTheDocument();
  });

  it('renders a loading skeleton, not an empty state, while history is still being fetched', () => {
    render(<MetricLineChart data={[]} lines={[{ dataKey: 'cpu', name: 'CPU', color: 'red' }]} unit="%" status="loading" />);
    expect(screen.getByLabelText('Loading chart data')).toBeInTheDocument();
    expect(screen.getByText('Loading timeseries…')).toBeInTheDocument();
    expect(screen.queryByText('No data in this window')).not.toBeInTheDocument();
  });

  it('renders a distinct error state that cannot be mistaken for loading or empty', () => {
    render(
      <MetricLineChart
        data={[]}
        lines={[{ dataKey: 'cpu', name: 'CPU', color: 'red' }]}
        unit="%"
        status="error"
        errorMessage="metrics history could not be read"
      />
    );
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText('Timeseries unavailable')).toBeInTheDocument();
    expect(screen.getByText('metrics history could not be read')).toBeInTheDocument();
    expect(screen.queryByLabelText('Loading chart data')).not.toBeInTheDocument();
    expect(screen.queryByText('No data in this window')).not.toBeInTheDocument();
  });

  it('renders chart series, tooltip content, and toggles one series while retaining another', () => {
    render(<MetricLineChart
      data={[{ timestamp: '2026-01-01T00:00:00Z', cpu: 42, memory: 20 }]}
      lines={[{ dataKey: 'cpu', name: 'CPU', color: 'red' }, { dataKey: 'memory', name: 'Memory', color: 'blue' }]}
      unit="%"
      valueFormatter={value => `${value.toFixed(0)}%`}
    />);
    expect(screen.getByTestId('line-cpu')).toHaveAttribute('data-hidden', 'false');
    expect(screen.getByTestId('line-memory')).toHaveAttribute('data-hidden', 'false');
    expect(screen.getAllByText('CPU').length).toBeGreaterThan(0);
    // Each series contributes a Line and a decorative Area. The legend is
    // built from the series list so it must name each series exactly once;
    // an inferred legend listed every series twice.
    expect(screen.getAllByRole('button', { name: 'CPU' })).toHaveLength(1);
    expect(screen.getAllByRole('button', { name: 'Memory' })).toHaveLength(1);
    fireEvent.click(screen.getByRole('button', { name: 'CPU' }));
    expect(screen.getByTestId('line-cpu')).toHaveAttribute('data-hidden', 'true');
    fireEvent.click(screen.getByRole('button', { name: 'CPU' }));
    expect(screen.getByTestId('line-cpu')).toHaveAttribute('data-hidden', 'false');
  });
});
