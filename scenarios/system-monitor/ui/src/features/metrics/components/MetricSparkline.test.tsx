import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it } from 'vitest';
import { MetricSparkline } from './MetricSparkline';

const point = (timestamp: string, value: number) => ({ timestamp, value });

describe('MetricSparkline', () => {
  it('shows the collection state when no samples exist', () => {
    render(<MetricSparkline ariaLabel="CPU history" />);
    expect(screen.getByText('Collecting data…')).toBeInTheDocument();
  });

  it('renders one and two point histories with labels and thresholds', () => {
    const one = render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 12)]} unit="%" windowLabel="1h" ariaLabel="CPU history" />);
    expect(screen.getByRole('img', { name: 'CPU history' })).toBeInTheDocument();
    expect(screen.getByText('1h')).toBeInTheDocument();
    one.unmount();

    render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 10), point('2026-01-01T00:01:00Z', 20)]} threshold={15} valueDomain={[0, 100]} unit="%" />);
    expect(screen.getByRole('img')).toBeInTheDocument();
    expect(screen.getByRole('img').querySelector('line')).toBeInTheDocument();
  });

  it('smooths multi-point data and exposes the nearest hover sample', () => {
    render(<MetricSparkline data={[
      point('2026-01-01T00:00:00Z', 10),
      point('2026-01-01T00:01:00Z', 30),
      point('2026-01-01T00:02:00Z', 20),
    ]} unit="%" />);
    const svg = screen.getByRole('img');
    Object.defineProperty(svg, 'getBoundingClientRect', { configurable: true, value: () => ({ left: 0, width: 100 }) });
    fireEvent.mouseMove(svg, { clientX: 50 });
    expect(screen.getByText('30.0%')).toBeInTheDocument();
    fireEvent.mouseLeave(svg);
    expect(screen.queryByText('30.0%')).not.toBeInTheDocument();
  });

  it('normalizes non-finite and flat domains and supports custom accessibility', () => {
    const { rerender } = render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', Number.NaN)]} valueDomain={[5, 5]} threshold={99} ariaLabel="custom chart" />);
    expect(screen.getByRole('img', { name: 'custom chart' })).toBeInTheDocument();
    rerender(<MetricSparkline data={[point('2026-01-01T00:00:00Z', Number.POSITIVE_INFINITY), point('2026-01-01T00:01:00Z', 2)]} threshold={-1} />);
    expect(screen.getByRole('img')).toBeInTheDocument();
  });
});
