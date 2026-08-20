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
    Legend: ({ onClick, formatter }: { onClick: (entry: { dataKey?: string }) => void; formatter: (value: string, entry?: { dataKey?: string }) => React.ReactNode }) => (
      <><button type="button" onClick={() => { onClick({ dataKey: 'cpu' }); }}>{formatter('CPU', { dataKey: 'cpu' })}</button><button type="button">{formatter('Other')}</button></>
    ),
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

  it('renders an honest empty chart state', () => {
    render(<MetricLineChart data={[]} lines={[{ dataKey: 'cpu', name: 'CPU', color: 'red' }]} unit="%" />);
    expect(screen.getByText('Waiting for timeseries data...')).toBeInTheDocument();
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
    fireEvent.click(screen.getByRole('button', { name: 'CPU' }));
    expect(screen.getByTestId('line-cpu')).toHaveAttribute('data-hidden', 'true');
    fireEvent.click(screen.getByRole('button', { name: 'CPU' }));
    expect(screen.getByTestId('line-cpu')).toHaveAttribute('data-hidden', 'false');
  });
});
