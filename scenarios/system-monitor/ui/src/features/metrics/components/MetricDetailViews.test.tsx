import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { cloneElement, type ReactElement } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { DetailSection, MetricDetailLayout, MetricLineChart } from './MetricDetailViews';

vi.mock('recharts', () => {
  const Box = ({ children, ...props }: { children?: React.ReactNode } & Record<string, unknown>) => <div {...props}>{children}</div>;
  return {
    ResponsiveContainer: Box,
    ComposedChart: Box,
    CartesianGrid: () => <div data-testid="grid" />,
    XAxis: () => <div data-testid="x-axis" />,
    Area: ({ dataKey, hide }: { dataKey: string; hide?: boolean }) => <div data-testid={`area-${dataKey}`} data-hidden={String(hide)} />,
    Line: ({ dataKey, hide, yAxisId }: { dataKey: string; hide?: boolean; yAxisId?: string }) => <div data-testid={`line-${dataKey}`} data-hidden={String(hide)} data-y-axis={yAxisId} />,
    YAxis: ({ yAxisId, scale, domain }: { yAxisId?: string; scale?: string; domain?: unknown }) => <div data-testid={`y-axis-${yAxisId ?? 'left'}`} data-scale={scale} data-domain={JSON.stringify(domain)} />,
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
  beforeEach(() => localStorage.clear());

  it('renders the shared detail layout and back action', () => {
    const onBack = vi.fn();
    render(<MetricDetailLayout layoutId="test" title="CPU" icon={<span>icon</span>} headline="42%" subhead="Last sample" onBack={onBack}><div>content</div></MetricDetailLayout>);
    expect(screen.getAllByText('CPU').length).toBeGreaterThan(0);
    expect(screen.getByText('42%')).toBeInTheDocument();
    expect(screen.getByText('42%').parentElement).toHaveClass('metric-detail-heading');
    expect(screen.getByText('Last sample')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Back To Dashboard/ }));
    expect(onBack).toHaveBeenCalledOnce();
  });

  it('recovers invalid preferences and supports keyboard-safe hide, reorder, density, columns, and reset', () => {
    localStorage.setItem('system-monitor-detail-layout:prefs', JSON.stringify({
      version: 1,
      order: ['missing'],
      hidden: ['missing'],
      density: 'invalid',
      columns: 9,
    }));
    render(
      <MetricDetailLayout layoutId="prefs" title="CPU" icon={<span>icon</span>} headline="42%" onBack={vi.fn()}>
        <DetailSection id="overview" title="Overview"><div>overview content</div></DetailSection>
        <DetailSection id="history" title="History"><div>history content</div></DetailSection>
      </MetricDetailLayout>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Customize layout' }));
    expect(screen.getByRole('combobox', { name: 'Detail density' })).toHaveValue('comfortable');
    expect(screen.getByRole('combobox', { name: 'Detail columns' })).toHaveValue('1');
    expect(screen.getByRole('checkbox', { name: 'Overview' })).toBeChecked();
    fireEvent.click(screen.getByRole('checkbox', { name: 'Overview' }));
    expect(screen.queryByText('overview content')).not.toBeInTheDocument();
    fireEvent.change(screen.getByRole('combobox', { name: 'Detail density' }), { target: { value: 'compact' } });
    fireEvent.change(screen.getByRole('combobox', { name: 'Detail columns' }), { target: { value: '2' } });
    fireEvent.click(screen.getByRole('button', { name: 'Move History up' }));
    fireEvent.click(screen.getByRole('button', { name: 'Reset to default' }));
    expect(screen.getByText('overview content')).toBeInTheDocument();
    expect(screen.getByText('history content')).toBeInTheDocument();
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

  it('binds a secondary series to the right axis when configured', () => {
    render(<MetricLineChart
      data={[{ timestamp: '2026-01-01T00:00:00Z', level: 45, traffic: 128 }]}
      lines={[
        { dataKey: 'level', name: 'Swap level', color: 'blue' },
        { dataKey: 'traffic', name: 'Swap traffic', color: 'orange', yAxisId: 'right' }
      ]}
      unit="%"
      rightYAxisUnit="/sec"
    />);
    expect(screen.getByTestId('line-level')).toHaveAttribute('data-y-axis', 'left');
    expect(screen.getByTestId('line-traffic')).toHaveAttribute('data-y-axis', 'right');
    expect(screen.getByTestId('y-axis-left')).toBeInTheDocument();
    expect(screen.getByTestId('y-axis-right')).toBeInTheDocument();
  });

  it('supports a logarithmic primary axis for bursty rate series', () => {
    render(<MetricLineChart
      data={[{ timestamp: '2026-01-01T00:00:00Z', value: 2 }]}
      lines={[{ dataKey: 'value', name: 'Major faults', color: 'red' }]}
      unit="/sec"
      yDomain={[1, 'auto']}
      yAxisScale="log"
    />);
    expect(screen.getByTestId('y-axis-left')).toHaveAttribute('data-scale', 'log');
    expect(screen.getByTestId('y-axis-left')).toHaveAttribute('data-domain', '[1,"auto"]');
  });

  it('provides accessible zoom, pan, reset, and summary controls without changing the source data', () => {
    const data = Array.from({ length: 8 }, (_, index) => ({
      timestamp: `2026-01-01T00:0${index}:00Z`,
      cpu: index + 1,
    }));
    render(<MetricLineChart data={data} lines={[{ dataKey: 'cpu', name: 'CPU', color: 'red' }]} unit="%" seriesLabel="CPU" />);
    expect(screen.getByTestId('metric-chart-summary')).toHaveTextContent('8 samples');
    expect(screen.getByText('8 of 8 samples')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Zoom in' }));
    expect(screen.getByText('6 of 8 samples')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Pan chart right' })).toBeEnabled();
    fireEvent.click(screen.getByRole('button', { name: 'Pan chart right' }));
    expect(screen.getByText('6 of 8 samples')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Reset chart view' }));
    expect(screen.getByText('8 of 8 samples')).toBeInTheDocument();
  });
});
